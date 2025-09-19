package animal

import (
	"context"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// AnimalRoom 动物房间（基于Erlang的zoo_room）
type AnimalRoom struct {
	mu     sync.RWMutex
	logger *zap.Logger

	// 房间基本信息
	id       uint32
	roomType pb.EZooType
	status   string

	// 动物管理
	animals   map[uint32]*AnimalRoute // 当前活跃动物
	generator *AnimalGenerator        // 动物生成器

	// 玩家管理
	players map[uint32]*PlayerSession // 房间内玩家

	// 游戏状态
	iceTime     time.Time             // 冰冻结束时间
	timers      map[uint32]*time.Timer // 动物定时器
	ticker      *time.Ticker           // 房间更新定时器
	ctx         context.Context
	cancel      context.CancelFunc

	// 动物生成控制
	lastGenerateTime time.Time // 上次生成动物的时间
	generateCooldown time.Duration // 生成冷却时间

	// 消息推送回调
	pushCallback func(*PushMessage)
}

// NewAnimalRoom 创建动物房间
func NewAnimalRoom(id uint32, roomType pb.EZooType, logger *zap.Logger, pushCallback func(*PushMessage)) *AnimalRoom {
	ctx, cancel := context.WithCancel(context.Background())

	room := &AnimalRoom{
		logger:           logger,
		id:               id,
		roomType:         roomType,
		status:           "normal",
		animals:          make(map[uint32]*AnimalRoute),
		players:          make(map[uint32]*PlayerSession),
		timers:           make(map[uint32]*time.Timer),
		ctx:              ctx,
		cancel:           cancel,
		pushCallback:     pushCallback,
		generator:        NewAnimalGenerator(id, logger),
		generateCooldown: 2 * time.Second, // 每2秒最多生成一只动物
	}

	return room
}

// Start 启动房间（基于Erlang的zoo_room:init）
func (r *AnimalRoom) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 启动房间更新循环
	r.ticker = time.NewTicker(1 * time.Second) // 1秒更新频率，避免过度生成动物
	go r.run()

	// 初始生成一些动物
	r.generateInitialAnimals()

	r.logger.Info("[AnimalRoom] 房间启动完成",
		zap.Uint32("room_id", r.id),
		zap.String("room_type", r.roomType.String()))
}

// Stop 停止房间
func (r *AnimalRoom) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cancel()
	if r.ticker != nil {
		r.ticker.Stop()
	}

	// 清理所有定时器
	for _, timer := range r.timers {
		timer.Stop()
	}

	r.logger.Info("[AnimalRoom] 房间已停止",
		zap.Uint32("room_id", r.id))
}

// GetPlayerCount 获取房间当前玩家数
func (r *AnimalRoom) GetPlayerCount() uint32 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return uint32(len(r.players))
}

// GetRoomID 获取房间ID
func (r *AnimalRoom) GetRoomID() uint32 {
	return r.id
}

// run 房间主循环（基于Erlang的zoo_room主循环）
func (r *AnimalRoom) run() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			r.logger.Error("[AnimalRoom] 房间运行异常", zap.Any("error", panicErr))
		}
	}()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.ticker.C:
			r.update()
		}
	}
}

// update 更新房间状态
func (r *AnimalRoom) update() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 更新动物位置
	r.updateAnimalPositions()

	// 检查动物是否到达终点
	r.checkAnimalsReachEnd()

	// 生成新动物维持数量
	r.maintainAnimalCount()
}

// updateAnimalPositions 更新动物位置
func (r *AnimalRoom) updateAnimalPositions() {
	now := time.Now()
	for _, animal := range r.animals {
		if animal.State == pb.EAnimalState_state_normal {
			// 根据时间计算新位置
			elapsed := now.Sub(animal.SpawnAt).Seconds()
			pathLine := r.generator.GetLineByID(animal.LineID)
			if pathLine != nil {
				// 简单的线性移动：每秒移动1个点
				newPoint := animal.Point + uint32(elapsed)
				if newPoint <= uint32(pathLine.Point) {
					animal.Point = newPoint
				}
			}
		}
	}
}

// checkAnimalsReachEnd 检查动物是否到达终点
func (r *AnimalRoom) checkAnimalsReachEnd() {
	var expiredAnimals []uint32

	for animalID, animal := range r.animals {
		pathLine := r.generator.GetLineByID(animal.LineID)
		if pathLine != nil && animal.Point >= uint32(pathLine.Point) {
			expiredAnimals = append(expiredAnimals, animalID)
		}
	}

	// 移除到达终点的动物
	for _, animalID := range expiredAnimals {
		r.removeAnimal(animalID)
	}
}

// maintainAnimalCount 维持动物数量（基于Erlang原版：18-22只动物）
func (r *AnimalRoom) maintainAnimalCount() {
	currentCount := len(r.animals)
	minCount := 18    // 最小动物数量
	targetCount := 20 // 理想数量
	maxCount := 22    // 最大数量

	now := time.Now()

	// 如果低于最小数量，立即补充（无冷却限制）
	if currentCount < minCount {
		for i := currentCount; i < minCount; i++ {
			r.generateNewAnimal()
		}
		r.lastGenerateTime = now
	} else if currentCount < targetCount {
		// 如果低于理想数量，检查冷却时间后有概率补充
		if now.Sub(r.lastGenerateTime) >= r.generateCooldown {
			if r.generator.rand.Float32() < 0.1 { // 降低到10%概率
				r.generateNewAnimal()
				r.lastGenerateTime = now
			}
		}
	} else if currentCount > maxCount {
		// 如果超过最大数量，移除最老的动物
		r.removeOldestAnimal()
	}
	// 通过正常的超时机制进行自然淘汰
}

// generateInitialAnimals 生成初始动物（基于Erlang原版：20只）
func (r *AnimalRoom) generateInitialAnimals() {
	// 基于Erlang原版：初始化20只动物
	initialCount := 20 // 固定20只
	for i := 0; i < initialCount; i++ {
		r.generateNewAnimal()
	}

	r.logger.Info("[AnimalRoom] 初始动物生成完成",
		zap.Uint32("room_id", r.id),
		zap.Int("initial_count", initialCount))
}

// generateNewAnimal 生成新动物（基于Erlang的add_animal）
func (r *AnimalRoom) generateNewAnimal() {
	// 获取当前房间已存在的动物类型，避免重复
	var excludeTypes []pb.EAnimal
	for _, animal := range r.animals {
		excludeTypes = append(excludeTypes, animal.Animal)
	}

	// 生成新动物
	newAnimal := r.generator.GenerateAnimal(excludeTypes)
	if newAnimal == nil {
		return
	}

	// 立即添加到房间（所有动物都立即加入，这样GetAnimals能获取到）
	r.animals[newAnimal.ID] = newAnimal

	// 设置动物离开定时器
	r.scheduleAnimalRemoval(newAnimal)

	// 检查是否是大象，需要延迟推送进场消息
	if newAnimal.Animal == pb.EAnimal_elephant {
		// 推送1883消息 - 大象将在5秒后进场
		r.pushAnimalComing(newAnimal.Animal, 5)

		// 5秒后推送进场消息（不需要再次获取锁）
		time.AfterFunc(5*time.Second, func() {
			// 推送动物进入消息
			r.pushAnimalEnter(newAnimal)
		})
	} else {
		// 非大象直接推送进场消息
		r.pushAnimalEnter(newAnimal)
	}
}

// scheduleAnimalRemoval 安排动物移除（基于Erlang的erlang:send_after逻辑）
func (r *AnimalRoom) scheduleAnimalRemoval(animal *AnimalRoute) {
	pathLine := r.generator.GetLineByID(animal.LineID)
	if pathLine == nil {
		return
	}

	// 计算动物在场时间
	duration := time.Duration(pathLine.Point) * time.Second

	// 设置定时器
	timer := time.AfterFunc(duration, func() {
		r.onAnimalTimeout(animal.ID)
	})

	r.timers[animal.ID] = timer
}

// onAnimalTimeout 动物超时处理
func (r *AnimalRoom) onAnimalTimeout(animalID uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.removeAnimal(animalID)
}

// removeAnimal 移除动物（基于Erlang的out_animal）
func (r *AnimalRoom) removeAnimal(animalID uint32) {
	animal, exists := r.animals[animalID]
	if !exists {
		return
	}

	// 移除动物
	delete(r.animals, animalID)

	// 清理定时器
	if timer, exists := r.timers[animalID]; exists {
		timer.Stop()
		delete(r.timers, animalID)
	}

	// 推送动物离开消息
	r.pushAnimalLeave(animal)

	r.logger.Debug("[AnimalRoom] 动物离开",
		zap.Uint32("room_id", r.id),
		zap.Uint32("animal_id", animalID),
		zap.String("animal_type", animal.Animal.String()))
}

// removeOldestAnimal 移除最老的动物（基于Erlang的动物数量控制）
func (r *AnimalRoom) removeOldestAnimal() {
	r.mu.Lock()
	defer r.mu.Unlock()

	var oldestAnimal *AnimalRoute
	var oldestID uint32

	// 找到最老的动物
	for id, animal := range r.animals {
		if oldestAnimal == nil || animal.SpawnAt.Before(oldestAnimal.SpawnAt) {
			oldestAnimal = animal
			oldestID = id
		}
	}

	if oldestAnimal != nil {
		// 移除最老的动物
		r.removeAnimal(oldestID)
		r.logger.Info("[AnimalRoom] 移除最老的动物",
			zap.Uint32("room_id", r.id),
			zap.Uint32("animal_id", oldestID),
			zap.String("animal_type", oldestAnimal.Animal.String()),
			zap.Time("spawn_time", oldestAnimal.SpawnAt))
	}
}

// GetAnimals 获取当前所有动物（基于Erlang的get_animals）
func (r *AnimalRoom) GetAnimals() []*pb.PRoute {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var routes []*pb.PRoute
	for _, animal := range r.animals {
		route := &pb.PRoute{
			Id:       proto.Uint32(animal.ID),
			Bet:      animal.Animal.Enum(),
			LineId:   proto.Uint32(animal.LineID),
			Point:    proto.Uint32(animal.Point),
			RedState: proto.Bool(animal.Red),
			Status:   animal.State.Enum(),
		}
		routes = append(routes, route)
	}

	return routes
}

// EnterRoom 玩家进入房间（基于Erlang的enter_zoo）
func (r *AnimalRoom) EnterRoom(playerID uint32, name, icon string) (*pb.M_1801Toc, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 创建玩家会话
	session := &PlayerSession{
		Player: &Player{
			ID:   playerID,
			Name: name,
			Icon: icon,
		},
		ZooType:   r.roomType,
		EnteredAt: time.Now(),
		Skills:    make(map[pb.EAnimalSkillType]*PlayerSkill),
	}

	r.players[playerID] = session

	// 准备响应数据
	response := &pb.M_1801Toc{
		BetVal:  r.getBetValues(),
		Odds:    r.getAnimalOdds(),
		Animals: r.GetAnimals(),
		Players: r.getPlayerList(),
		Time:    proto.Uint32(uint32(time.Until(r.iceTime).Seconds())),
	}

	// 推送玩家进入消息给其他玩家
	r.pushPlayerEnter(session)

	r.logger.Info("[AnimalRoom] 玩家进入房间",
		zap.Uint32("room_id", r.id),
		zap.Uint32("player_id", playerID),
		zap.String("name", name))

	return response, nil
}

// LeaveRoom 玩家离开房间
func (r *AnimalRoom) LeaveRoom(playerID uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.players[playerID]
	if !exists {
		return
	}

	delete(r.players, playerID)

	// 推送玩家离开消息
	r.pushPlayerLeave(session)

	r.logger.Info("[AnimalRoom] 玩家离开房间",
		zap.Uint32("room_id", r.id),
		zap.Uint32("player_id", playerID))
}

// getBetValues 获取房间下注档位
func (r *AnimalRoom) getBetValues() []uint32 {
	// 基于房间类型返回不同的下注档位
	switch r.roomType {
	case pb.EZooType_free:
		return []uint32{100, 200, 300, 400, 500, 1000, 2000, 5000, 10000, 20000}
	case pb.EZooType_single:
		return []uint32{100, 200, 300, 400, 500, 1000, 2000, 5000, 10000, 20000}
	case pb.EZooType_diamond:
		return []uint32{100, 200, 300, 400, 500, 1000, 2000, 5000, 10000, 20000}
	default:
		return []uint32{100, 500, 1000, 2000, 5000}
	}
}

// getAnimalOdds 获取动物赔率
func (r *AnimalRoom) getAnimalOdds() []*pb.PAnimalOdds {
	// 基础赔率配置
	animalOdds := map[pb.EAnimal][]uint32{
		pb.EAnimal_turtle:   {12, 20},
		pb.EAnimal_cock:     {15, 25},
		pb.EAnimal_dog:      {18, 30},
		pb.EAnimal_monkey:   {20, 35},
		pb.EAnimal_horse:    {22, 40},
		pb.EAnimal_ox:       {25, 45},
		pb.EAnimal_panda:    {50, 100},
		pb.EAnimal_hippo:    {60, 120},
		pb.EAnimal_lion:     {80, 150},
		pb.EAnimal_elephant: {100, 200},
		pb.EAnimal_pikachu:  {80, 160},
		pb.EAnimal_bomber:   {200, 400},
	}

	var odds []*pb.PAnimalOdds
	for animal, oddsRange := range animalOdds {
		animalOdd := &pb.PAnimalOdds{
			Bet:  animal.Enum(),
			Odds: oddsRange,
		}
		odds = append(odds, animalOdd)
	}

	return odds
}

// getPlayerList 获取房间玩家列表
func (r *AnimalRoom) getPlayerList() []*pb.PAnimalPlayer {
	var players []*pb.PAnimalPlayer
	for _, session := range r.players {
		player := &pb.PAnimalPlayer{
			RoleId: proto.Uint32(session.Player.ID),
			Icon:   proto.String(session.Player.Icon),
			Name:   proto.String(session.Player.Name),
		}
		players = append(players, player)
	}
	return players
}

// 推送消息方法
func (r *AnimalRoom) pushAnimalEnter(animal *AnimalRoute) {
	if r.pushCallback != nil {
		route := &pb.PRoute{
			Id:       proto.Uint32(animal.ID),
			Bet:      animal.Animal.Enum(),
			LineId:   proto.Uint32(animal.LineID),
			Point:    proto.Uint32(animal.Point),
			RedState: proto.Bool(animal.Red),
			Status:   animal.State.Enum(),
		}

		msg := &pb.M_1887Toc{
			Animal: []*pb.PRoute{route},
		}

		r.pushCallback(&PushMessage{
			MsgID:   1887,
			ZooType: r.roomType,
			Message: msg,
		})
	}
}

// pushAnimalComing 推送动物即将进场消息（1883）
func (r *AnimalRoom) pushAnimalComing(animalType pb.EAnimal, timeSeconds uint32) {
	if r.pushCallback != nil {
		msg := &pb.M_1883Toc{
			Animal: &animalType,
			Time:   proto.Uint32(timeSeconds),
		}

		r.pushCallback(&PushMessage{
			MsgID:   1883,
			ZooType: r.roomType,
			Message: msg,
		})

		r.logger.Info("[AnimalRoom] 推送大象延迟进场",
			zap.String("animal_type", animalType.String()),
			zap.Uint32("delay_seconds", timeSeconds))
	}
}

func (r *AnimalRoom) pushAnimalLeave(animal *AnimalRoute) {
	if r.pushCallback != nil {
		msg := &pb.M_1888Toc{
			Id: proto.Uint32(animal.ID),
		}

		r.pushCallback(&PushMessage{
			MsgID:   1888,
			ZooType: r.roomType,
			Message: msg,
		})
	}
}

func (r *AnimalRoom) pushPlayerEnter(session *PlayerSession) {
	if r.pushCallback != nil {
		player := &pb.PAnimalPlayer{
			RoleId: proto.Uint32(session.Player.ID),
			Icon:   proto.String(session.Player.Icon),
			Name:   proto.String(session.Player.Name),
		}

		msg := &pb.M_1886Toc{
			Player: player,
		}

		r.pushCallback(&PushMessage{
			MsgID:   1886,
			ZooType: r.roomType,
			Message: msg,
		})
	}
}

func (r *AnimalRoom) pushPlayerLeave(session *PlayerSession) {
	if r.pushCallback != nil {
		msg := &pb.M_1885Toc{
			RoleId: proto.Uint32(session.Player.ID),
		}

		r.pushCallback(&PushMessage{
			MsgID:   1885,
			ZooType: r.roomType,
			Message: msg,
		})
	}
}

// killAnimal 击杀动物（基于Erlang原版动物被打死逻辑）
func (r *AnimalRoom) killAnimal(animalID uint32, damage uint32) bool {
	animal, exists := r.animals[animalID]
	if !exists {
		return false
	}

	// 基于Erlang原版：直接击杀逻辑，无HP系统
	animal.State = pb.EAnimalState_state_ice // 使用冰冻状态表示死亡

	// 推送动物死亡消息
	r.pushAnimalDeath(animal)

	// 移除动物
	r.removeAnimal(animalID)

	r.logger.Info("[AnimalRoom] 动物被击杀",
		zap.Uint32("room_id", r.id),
		zap.Uint32("animal_id", animalID),
		zap.String("animal_type", animal.Animal.String()),
		zap.Uint32("damage", damage))

	return true
}

// DamageAnimal 对动物造成伤害（公共接口）
func (r *AnimalRoom) DamageAnimal(animalID uint32, damage uint32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.killAnimal(animalID, damage)
}

// pushAnimalDeath 推送动物死亡消息（复用动物离开消息）
func (r *AnimalRoom) pushAnimalDeath(animal *AnimalRoute) {
	if r.pushCallback != nil {
		msg := &pb.M_1888Toc{
			Id: proto.Uint32(animal.ID),
		}

		r.pushCallback(&PushMessage{
			MsgID:   1888, // 使用动物离开消息ID
			ZooType: r.roomType,
			Message: msg,
		})
	}
}