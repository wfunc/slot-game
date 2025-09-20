package animal

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"google.golang.org/protobuf/proto"
)

var (
	ErrRoomNotFound      = errors.New("animal: room not found")
	ErrVIPRequirement    = errors.New("animal: vip requirement not met")
	ErrSkillUnavailable  = errors.New("animal: skill unavailable")
	ErrInsufficientFunds = errors.New("animal: insufficient balance")
	ErrPlayerNotInRoom   = errors.New("animal: player not in room")
)

var defaultBetValues = map[pb.EZooType][]uint32{
	pb.EZooType_civilian: {10, 20, 50, 100},
	pb.EZooType_petty:    {50, 100, 200, 500},
	pb.EZooType_rich:     {200, 500, 1000, 2000},
	pb.EZooType_gold:     {500, 1000, 2000, 5000},
	pb.EZooType_diamond:  {1000, 2000, 5000, 10000},
	pb.EZooType_single:   {100, 200, 500, 1000},
	pb.EZooType_free:     {0, 10, 20, 50},
}

var vipRequirement = map[pb.EZooType]uint32{
	pb.EZooType_civilian: 0,
	pb.EZooType_petty:    1,
	pb.EZooType_rich:     3,
	pb.EZooType_gold:     5,
	pb.EZooType_diamond:  8,
	pb.EZooType_single:   0,
	pb.EZooType_free:     0,
}

var defaultAnimalOrder = []pb.EAnimal{
	pb.EAnimal_turtle,
	pb.EAnimal_cock,
	pb.EAnimal_dog,
	pb.EAnimal_monkey,
	pb.EAnimal_horse,
	pb.EAnimal_ox,
	pb.EAnimal_panda,
	pb.EAnimal_hippo,
	pb.EAnimal_lion,
	pb.EAnimal_elephant,
	pb.EAnimal_pikachu,
	pb.EAnimal_bomber,
	pb.EAnimal_tiger,
	pb.EAnimal_sheep,
	pb.EAnimal_bear,
	pb.EAnimal_tuzi,
	pb.EAnimal_lv,
	pb.EAnimal_baozi,
	pb.EAnimal_zhu,
	pb.EAnimal_hema,
}

// NewManager 创建动物游戏管理器
func NewManager() *Manager {
	m := &Manager{
		rooms:       make(map[uint32]*Room),
		roomsByType: make(map[pb.EZooType][]uint32),
		nextRoomID:  1,
		players:     make(map[uint32]*Player),
		rewards:     make([]*pb.PAnimalReward, 0, 32),
		rand:        randSource(),
	}

	// 初始化时只为每个房间类型创建1个房间
	for zooType, bets := range defaultBetValues {
		roomID := m.nextRoomID
		m.nextRoomID++

		room := &Room{
			ID:             roomID,
			Type:           zooType,
			BetValues:      append([]uint32(nil), bets...),
			MaxPlayer:      MAX_PLAYERS_PER_ROOM,
			MinVIP:         vipRequirement[zooType],
			CurrentPlayers: 0,
			animals:        make(map[uint32]*AnimalRoute),
			nextAnimalID:   1,
			players:        make(map[uint32]*PlayerSession),
			jackpot:        nil,
			redBag:         true,
		}

		// 初始生成动物
		for i := 0; i < 10; i++ {
			animal := defaultAnimalOrder[i%len(defaultAnimalOrder)]
			room.spawnAnimal(animal, m.rand)
		}

		m.rooms[roomID] = room
		m.roomsByType[zooType] = []uint32{roomID}
	}

	return m
}

// FindOrCreateRoom 查找或创建房间
// 如果指定类型有未满员的房间，返回该房间
// 如果所有房间都满员，创建新房间
func (m *Manager) FindOrCreateRoom(zooType pb.EZooType) (*Room, error) {
	// 注意：此方法应该在调用者已经持有锁的情况下调用

	// 查找该类型的所有房间
	roomIDs, exists := m.roomsByType[zooType]
	if !exists || len(roomIDs) == 0 {
		// 该类型还没有房间，创建第一个
		return m.createRoom(zooType)
	}

	// 查找第一个未满员的房间
	for _, roomID := range roomIDs {
		room, ok := m.rooms[roomID]
		if !ok {
			continue
		}
		if room.CurrentPlayers < room.MaxPlayer {
			// 找到未满员的房间
			return room, nil
		}
	}

	// 所有房间都满员，创建新房间
	return m.createRoom(zooType)
}

// createRoom 创建新房间
func (m *Manager) createRoom(zooType pb.EZooType) (*Room, error) {
	// 从defaultBetValues获取配置
	bets, ok := defaultBetValues[zooType]
	if !ok {
		return nil, fmt.Errorf("unknown zoo type: %v", zooType)
	}

	roomID := m.nextRoomID
	m.nextRoomID++

	room := &Room{
		ID:             roomID,
		Type:           zooType,
		BetValues:      append([]uint32(nil), bets...),
		MaxPlayer:      MAX_PLAYERS_PER_ROOM,
		MinVIP:         vipRequirement[zooType],
		CurrentPlayers: 0,
		animals:        make(map[uint32]*AnimalRoute),
		nextAnimalID:   1,
		players:        make(map[uint32]*PlayerSession),
		jackpot:        NewJackpotPool(),     // 初始化彩金池
		taskManager:    NewTaskManager(),     // 初始化任务管理器
		oneBlowManager: NewOneBlowManager(),  // 初始化一击必杀管理器
		profitControl:  &RoomProfitControl{TotalBet: 0, TotalWin: 0}, // 初始化盈亏控制
		redBag:         true,
	}

	// 初始生成动物
	for _, animal := range defaultAnimalOrder {
		room.spawnAnimal(animal, m.rand)
	}

	// 添加到管理器
	m.rooms[roomID] = room
	m.roomsByType[zooType] = append(m.roomsByType[zooType], roomID)

	// 记录日志
	log.Printf("[Manager] 创建新房间: ID=%d, Type=%v, MaxPlayers=%d",
		roomID, zooType, room.MaxPlayer)

	return room, nil
}

func randSource() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// EnterRoom 玩家进入房间
func (m *Manager) EnterRoom(playerID uint32, name, icon string, vip uint32, req *pb.M_1801Tos) (*pb.M_1801Toc, []PushMessage, error) {
	zooType := req.GetType()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 查找或创建合适的房间
	room, err := m.FindOrCreateRoom(zooType)
	if err != nil {
		return nil, nil, err
	}

	if vip < room.MinVIP {
		return nil, nil, ErrVIPRequirement
	}

	player := m.ensurePlayer(playerID, name, icon, vip)
	session, exists := room.players[playerID]
	if !exists {
		session = &PlayerSession{
			Player:    player,
			ZooType:   zooType,
			EnteredAt: time.Now(),
			Skills:    make(map[pb.EAnimalSkillType]*PlayerSkill),
			SkillEnds: make(map[pb.EAnimalSkillType]time.Time),
		}

		// 将玩家技能库存复制到会话
		for t, skill := range player.Skills {
			session.Skills[t] = &PlayerSkill{
				Type:  t,
				Value: skill.Value,
				Count: skill.Count,
				Time:  skill.Time,
			}
		}

		room.players[playerID] = session
		room.CurrentPlayers++
		log.Printf("[Manager] 玩家 %d 进入房间 %d，当前人数 %d/%d",
			playerID, room.ID, room.CurrentPlayers, room.MaxPlayer)
	} else {
		session.Player = player
	}

	// 同步技能库存
	for t, skill := range player.Skills {
		if existing, ok := session.Skills[t]; ok {
			existing.Count = skill.Count
			existing.Value = skill.Value
			existing.Time = skill.Time
		} else {
			session.Skills[t] = &PlayerSkill{
				Type:  t,
				Value: skill.Value,
				Count: skill.Count,
				Time:  skill.Time,
			}
		}
	}

	if session.CurrentBet == 0 && len(room.BetValues) > 0 {
		session.CurrentBet = room.BetValues[0]
	}

	resp := &pb.M_1801Toc{
		BetVal:   append([]uint32(nil), room.BetValues...),
		Odds:     defaultOddsProto(),
		Animals:  room.routesProto(),
		Players:  room.playersProto(),
		RedState: proto.Bool(room.redBag),
		Skill:    buildSkillList(session.Skills, session.SkillEnds),
		Time:     proto.Uint32(room.nextSkillExpiry(session.SkillEnds)),
	}

	if player.FreeGold > 0 {
		resp.FreeGold = proto.Uint64(player.FreeGold)
	}

	resp.Cj = proto.String(fmt.Sprintf("%d", room.jackpot.GetCurrentAmount()))

	pushes := []PushMessage{}
	if !exists {
		pushes = append(pushes, PushMessage{
			MsgID:   1886,
			ZooType: zooType,
			Targets: room.otherPlayerIDs(playerID),
			Message: &pb.M_1886Toc{
				Player: &pb.PAnimalPlayer{
					RoleId: proto.Uint32(player.ID),
					Icon:   proto.String(player.Icon),
					Name:   proto.String(player.Name),
				},
			},
		})
	}

	return resp, pushes, nil
}

// LeaveRoom 玩家离开房间
func (m *Manager) LeaveRoom(playerID uint32, req *pb.M_1802Tos) (*pb.M_1802Toc, []PushMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, room := range m.rooms {
		if session, ok := room.players[playerID]; ok {
			total := clampUint32(session.TotalWin)
			delete(room.players, playerID)
			room.CurrentPlayers--
			log.Printf("[Manager] 玩家 %d 离开房间 %d，当前人数 %d/%d",
				playerID, room.ID, room.CurrentPlayers, room.MaxPlayer)

			push := PushMessage{
				MsgID:   1885,
				ZooType: room.Type,
				Targets: room.otherPlayerIDs(playerID),
				Message: &pb.M_1885Toc{RoleId: proto.Uint32(playerID)},
			}

			resp := &pb.M_1802Toc{TotalWin: proto.Uint32(total)}

			return resp, []PushMessage{push}, nil
		}
	}

	return &pb.M_1802Toc{TotalWin: proto.Uint32(0)}, nil, nil
}

// Bet 玩家下注（兼容旧版本，无子弹系统）
func (m *Manager) Bet(playerID uint32, req *pb.M_1803Tos) (*pb.M_1803Toc, []PushMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := m.findRoomByPlayer(playerID)
	if room == nil {
		return nil, nil, ErrRoomNotFound
	}

	session := room.players[playerID]
	player := session.Player

	betVal := session.CurrentBet
	if betVal == 0 {
		if len(room.BetValues) > 0 {
			betVal = room.BetValues[0]
			session.CurrentBet = betVal
		} else {
			betVal = 100
		}
	}

	if player.Balance < uint64(betVal) {
		return nil, nil, ErrInsufficientFunds
	}

	player.Balance -= uint64(betVal)

	// 模拟中奖
	target := room.animals[req.GetId()]
	outcome := m.simulateBetOutcome(room, target, betVal)
	player.Balance += uint64(outcome.WinAmount)
	player.FreeGold += outcome.FreeGold
	session.TotalWin += uint64(outcome.WinAmount)

	animalType := pb.EAnimal_balance
	if target != nil {
		animalType = target.Animal
	}

	record := &pb.PPlayerAnimal{
		Id:     proto.Uint32(m.nextRecordID()),
		Time:   proto.Uint32(uint32(time.Now().Unix())),
		BetVal: proto.Uint32(betVal),
		Win:    proto.Uint32(outcome.WinAmount),
		Animal: animalType.Enum(),
	}

	player.History = append([]*pb.PPlayerAnimal{record}, player.History...)
	if len(player.History) > 50 {
		player.History = player.History[:50]
	}

	if outcome.WinAmount >= betVal*5 {
		m.appendReward(player, animalType, betVal, outcome.WinAmount)
	}

	resp := &pb.M_1803Toc{
		Balance:  proto.Uint64(player.Balance),
		Win:      proto.Uint32(outcome.WinAmount),
		RedBag:   proto.Uint32(outcome.RedBag),
		Skill:    buildSkillList(session.Skills, session.SkillEnds),
		TotalWin: proto.Uint64(session.TotalWin),
	}

	if player.FreeGold > 0 {
		resp.FreeGold = proto.Uint64(player.FreeGold)
	}

	pushes := m.buildBetPushes(room, session, req, outcome)

	return resp, pushes, nil
}

// GetRecord 玩家历史记录
func (m *Manager) GetRecord(playerID uint32, req *pb.M_1804Tos) (*pb.M_1804Toc, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	player := m.players[playerID]
	if player == nil {
		return &pb.M_1804Toc{}, nil
	}

	limit := int(req.GetAmount())
	if limit <= 0 || limit > len(player.History) {
		limit = len(player.History)
	}

	return &pb.M_1804Toc{Info: append([]*pb.PPlayerAnimal(nil), player.History[:limit]...)}, nil
}

// GetRewards 获取最近大奖
func (m *Manager) GetRewards() *pb.M_1805Toc {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &pb.M_1805Toc{Info: append([]*pb.PAnimalReward(nil), m.rewards...)}
}

// UseSkill 使用技能
func (m *Manager) UseSkill(playerID uint32, req *pb.M_1806Tos) (*pb.M_1806Toc, []PushMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := m.findRoomByPlayer(playerID)
	if room == nil {
		return nil, nil, ErrRoomNotFound
	}

	session := room.players[playerID]
	skill := session.Skills[req.GetType()]
	if skill == nil || skill.Count == 0 {
		return nil, nil, ErrSkillUnavailable
	}

	skill.Count--
	duration := time.Duration(skill.Time) * time.Second
	session.SkillEnds[req.GetType()] = time.Now().Add(duration)

	skillMsg := &pb.PAnimalSkill{
		Type:  req.GetType().Enum(),
		Val:   proto.Uint32(skill.Value),
		Time:  proto.Uint32(skill.Time),
		Count: proto.Uint32(skill.Count),
	}

	resp := &pb.M_1806Toc{Skill: skillMsg}

	push := PushMessage{
		MsgID:   1882,
		ZooType: room.Type,
		Targets: room.otherPlayerIDs(playerID),
		Message: &pb.M_1882Toc{
			RoleId: proto.Uint32(playerID),
			Type:   req.GetType().Enum(),
			Time:   proto.Uint32(skill.Time),
		},
	}

	return resp, []PushMessage{push}, nil
}

// GetZooTypes 获取所有场信息
func (m *Manager) GetZooTypes() *pb.M_1807Toc {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]*pb.PZooTypeInfo, 0, len(m.rooms))
	for _, room := range m.rooms {
		infos = append(infos, &pb.PZooTypeInfo{
			Type:   room.Type.Enum(),
			BetVal: append([]uint32(nil), room.BetValues...),
			MaxNum: proto.Uint32(uint32(len(room.players))),
			Vip:    proto.Uint32(room.MinVIP),
		})
	}

	sort.Slice(infos, func(i, j int) bool { return infos[i].GetType() < infos[j].GetType() })

	return &pb.M_1807Toc{Info: infos}
}

// BuyTool 购买技能
func (m *Manager) BuyTool(playerID uint32, req *pb.M_1808Tos) (*pb.M_1808Toc, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	player := m.players[playerID]
	if player == nil {
		return nil, ErrRoomNotFound
	}

	price := toolPrice(req.GetType())
	if player.Balance < uint64(price) {
		return nil, ErrInsufficientFunds
	}

	player.Balance -= uint64(price)
	skill := player.Skills[req.GetType()]
	if skill == nil {
		skill = &PlayerSkill{Type: req.GetType(), Value: 1, Time: 10}
		player.Skills[req.GetType()] = skill
	}
	skill.Count++

	// 更新活跃会话中的技能信息
	if room := m.findRoomByPlayer(playerID); room != nil {
		if session, ok := room.players[playerID]; ok {
			if s, ok := session.Skills[req.GetType()]; ok {
				s.Count = skill.Count
				s.Value = skill.Value
				s.Time = skill.Time
			} else {
				session.Skills[req.GetType()] = &PlayerSkill{
					Type:  req.GetType(),
					Value: skill.Value,
					Count: skill.Count,
					Time:  skill.Time,
				}
			}
		}
	}

	return &pb.M_1808Toc{}, nil
}

// GetToolPrice 获取道具价格
func (m *Manager) GetToolPrice() *pb.M_1809Toc {
	return &pb.M_1809Toc{Val: proto.Uint32(500)}
}

// helper functions below

func (m *Manager) ensurePlayer(id uint32, name, icon string, vip uint32) *Player {
	player, ok := m.players[id]
	if !ok {
		player = &Player{
			ID:       id,
			Name:     defaultName(name, id),
			Icon:     defaultIcon(icon),
			VIP:      vip,
			Balance:  50000,
			FreeGold: 1000,
			History:  make([]*pb.PPlayerAnimal, 0, 16),
			Skills:   defaultSkills(),
		}
		m.players[id] = player
	} else {
		if name != "" {
			player.Name = name
		}
		if icon != "" {
			player.Icon = icon
		}
		if vip > player.VIP {
			player.VIP = vip
		}
	}
	return player
}

func (m *Manager) findRoomByPlayer(playerID uint32) *Room {
	for _, room := range m.rooms {
		if _, ok := room.players[playerID]; ok {
			return room
		}
	}
	return nil
}

var recordSeq uint32 = 1

func (m *Manager) nextRecordID() uint32 {
	id := recordSeq
	recordSeq++
	return id
}

func clampUint32(v uint64) uint32 {
	if v > uint64(^uint32(0)) {
		return ^uint32(0)
	}
	return uint32(v)
}

func (m *Manager) appendReward(player *Player, animal pb.EAnimal, bet, win uint32) {
	m.rewardCursor++
	reward := &pb.PAnimalReward{
		Id:     proto.Uint32(m.rewardCursor),
		Animal: animal.Enum(),
		BetVal: proto.Uint32(bet),
		Win:    proto.Uint32(win),
		Time:   proto.Uint32(uint32(time.Now().Unix())),
		Name:   proto.String(player.Name),
		Icon:   proto.String(player.Icon),
		Vip:    proto.Uint32(player.VIP),
	}

	m.rewards = append([]*pb.PAnimalReward{reward}, m.rewards...)
	if len(m.rewards) > 50 {
		m.rewards = m.rewards[:50]
	}
}

func defaultSkills() map[pb.EAnimalSkillType]*PlayerSkill {
	return map[pb.EAnimalSkillType]*PlayerSkill{
		pb.EAnimalSkillType_skill_ice: {
			Type:  pb.EAnimalSkillType_skill_ice,
			Value: 5,
			Count: 3,
			Time:  10,
		},
		pb.EAnimalSkillType_locking: {
			Type:  pb.EAnimalSkillType_locking,
			Value: 1,
			Count: 2,
			Time:  8,
		},
		pb.EAnimalSkillType_improve_odds: {
			Type:  pb.EAnimalSkillType_improve_odds,
			Value: 2,
			Count: 2,
			Time:  12,
		},
	}
}

func toolPrice(skill pb.EAnimalSkillType) uint32 {
	switch skill {
	case pb.EAnimalSkillType_skill_ice:
		return 300
	case pb.EAnimalSkillType_locking:
		return 400
	case pb.EAnimalSkillType_improve_odds:
		return 500
	default:
		return 500
	}
}

func defaultIcon(icon string) string {
	if icon != "" {
		return icon
	}
	return "https://cdn.example.com/avatar/default.png"
}

func defaultName(name string, id uint32) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("玩家%d", id)
}

func buildSkillList(skills map[pb.EAnimalSkillType]*PlayerSkill, ends map[pb.EAnimalSkillType]time.Time) []*pb.PAnimalSkill {
	now := time.Now()
	result := make([]*pb.PAnimalSkill, 0, len(skills))
	for t, skill := range skills {
		remaining := uint32(0)
		if end, ok := ends[t]; ok && end.After(now) {
			remaining = uint32(end.Sub(now).Seconds())
		}
		result = append(result, &pb.PAnimalSkill{
			Type:  t.Enum(),
			Val:   proto.Uint32(skill.Value),
			Time:  proto.Uint32(remaining),
			Count: proto.Uint32(skill.Count),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].GetType() < result[j].GetType()
	})

	return result
}

func defaultOddsProto() []*pb.PAnimalOdds {
	odds := make([]*pb.PAnimalOdds, 0, len(defaultAnimalOrder))
	for idx, animal := range defaultAnimalOrder {
		odds = append(odds, &pb.PAnimalOdds{
			Bet:  animal.Enum(),
			Odds: []uint32{uint32(1 + idx), uint32(2 + idx)},
		})
	}
	return odds
}

func (r *Room) spawnAnimal(animal pb.EAnimal, rnd *rand.Rand) *AnimalRoute {
	id := r.nextAnimalID
	r.nextAnimalID++

	route := &AnimalRoute{
		ID:      id,
		Animal:  animal,
		LineID:  uint32(rnd.Intn(6) + 1),
		Point:   uint32(rnd.Intn(8) + 1),
		Red:     rnd.Intn(100) < 20,
		State:   pb.EAnimalState_state_normal,
		SpawnAt: time.Now(),
	}

	r.animals[id] = route
	return route
}

func (r *Room) routesProto() []*pb.PRoute {
	routes := make([]*pb.PRoute, 0, len(r.animals))
	for _, route := range r.animals {
		routes = append(routes, &pb.PRoute{
			Id:       proto.Uint32(route.ID),
			Bet:      route.Animal.Enum(),
			LineId:   proto.Uint32(route.LineID),
			Point:    proto.Uint32(route.Point),
			RedState: proto.Bool(route.Red),
			Status:   route.State.Enum(),
		})
	}

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].GetId() < routes[j].GetId()
	})

	return routes
}

func (r *Room) playersProto() []*pb.PAnimalPlayer {
	players := make([]*pb.PAnimalPlayer, 0, len(r.players))
	for _, session := range r.players {
		players = append(players, &pb.PAnimalPlayer{
			RoleId: proto.Uint32(session.Player.ID),
			Icon:   proto.String(session.Player.Icon),
			Name:   proto.String(session.Player.Name),
		})
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].GetRoleId() < players[j].GetRoleId()
	})

	return players
}

func (r *Room) otherPlayerIDs(exclude uint32) []uint32 {
	ids := make([]uint32, 0, len(r.players))
	for id := range r.players {
		if id == exclude {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func (r *Room) nextSkillExpiry(ends map[pb.EAnimalSkillType]time.Time) uint32 {
	now := time.Now()
	var min time.Duration = -1
	for _, end := range ends {
		if end.After(now) {
			remaining := end.Sub(now)
			if min < 0 || remaining < min {
				min = remaining
			}
		}
	}

	if min <= 0 {
		return 0
	}
	return uint32(min.Seconds())
}

// BetWithBullet 使用子弹进行下注
func (m *Manager) BetWithBullet(playerID uint32, targetID uint32, betVal uint32, multiple uint32) (*pb.M_1803Toc, []PushMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room := m.findRoomByPlayer(playerID)
	if room == nil {
		return nil, nil, ErrRoomNotFound
	}

	session := room.players[playerID]
	if session == nil {
		return nil, nil, ErrPlayerNotInRoom
	}

	// 获取目标动物
	target := room.animals[targetID]
	if target == nil {
		return nil, nil, fmt.Errorf("目标动物不存在: %d", targetID)
	}

	// 使用房间的ProcessBet方法进行处理
	outcome := room.ProcessBet(session, targetID, betVal, multiple)

	// 处理结果
	if outcome.WinAmount > 0 {
		session.Player.Balance += uint64(outcome.WinAmount)
		session.TotalWin += uint64(outcome.WinAmount)
	}

	// 构建响应
	resp := &pb.M_1803Toc{
		Win:      proto.Uint32(outcome.WinAmount),
		RedBag:   proto.Uint32(outcome.RedBag),
		Balance:  proto.Uint64(session.Player.Balance),
		TotalWin: proto.Uint64(session.TotalWin),
	}

	// 如果有技能获得
	if len(outcome.SkillGain) > 0 {
		resp.Skill = outcome.SkillGain
	}

	// 如果有彩金赢取
	if outcome.JackpotWin > 0 {
		resp.Balance = proto.Uint64(session.Player.Balance + outcome.JackpotWin)
	}

	// 如果是体验场，设置体验币
	if room.Type == pb.EZooType_free && outcome.FreeGold > 0 {
		resp.FreeGold = proto.Uint64(outcome.FreeGold)
	}

	// 构建推送消息
	pushes := m.buildBetPushesWithOutcome(room, session, targetID, outcome)

	return resp, pushes, nil
}

// buildBetPushesWithOutcome 根据实际结果构建推送消息
func (m *Manager) buildBetPushesWithOutcome(room *Room, session *PlayerSession, targetID uint32, outcome *BetOutcome) []PushMessage {
	pushes := []PushMessage{}

	// 推送玩家打动物
	pushes = append(pushes, PushMessage{
		MsgID:   1899,
		ZooType: room.Type,
		Targets: room.otherPlayerIDs(session.Player.ID),
		Message: &pb.M_1899Toc{
			RoleId: proto.Uint32(session.Player.ID),
			Id:     proto.Uint32(targetID),
		},
	})

	// 推送动物死亡
	if len(outcome.KilledRoutes) > 0 {
		killedAnimals := make([]*pb.PAnimalOne, 0, len(outcome.KilledRoutes))
		for _, killed := range outcome.KilledRoutes {
			killedAnimals = append(killedAnimals, &pb.PAnimalOne{
				Id:     proto.Uint32(killed.ID),
				Win:    proto.Uint32(0), // 可以根据实际赢得金额填充
				RedBag: proto.Uint32(0), // 可以根据实际红包金额填充
			})
		}

		pushes = append(pushes, PushMessage{
			MsgID:   1884,
			ZooType: room.Type,
			Targets: nil, // 广播给房间所有人
			Message: &pb.M_1884Toc{
				RoleId: proto.Uint32(session.Player.ID),
				Type:   outcome.EffectType.Enum(),
				Ids:    killedAnimals,
			},
		})

		// 推送动物离开
		for _, killed := range outcome.KilledRoutes {
			pushes = append(pushes, PushMessage{
				MsgID:   1888,
				ZooType: room.Type,
				Targets: nil,
				Message: &pb.M_1888Toc{
					Id: proto.Uint32(killed.ID),
				},
			})
		}
	}

	// 推送彩金中奖
	if outcome.JackpotWin > 0 {
		pushes = append(pushes, PushMessage{
			MsgID:   1811,
			ZooType: room.Type,
			Targets: nil,
			Message: &pb.M_1811Toc{
				Bonus: proto.String(fmt.Sprintf("%d", outcome.JackpotWin)),
			},
		})
	}

	return pushes
}

func (m *Manager) simulateBetOutcome(room *Room, target *AnimalRoute, betVal uint32) BetOutcome {
	win := uint32(0)
	red := uint32(0)

	multiplier := uint32(1 + m.rand.Intn(5))
	if m.rand.Intn(100) < 40 {
		win = betVal * multiplier
	}

	if m.rand.Intn(100) < 15 {
		red = uint32(1 + m.rand.Intn(3))
	}

	freeGold := uint64(0)
	if room.Type == pb.EZooType_free {
		freeGold = uint64(betVal / 2)
	}

	killed := make([]*AnimalRoute, 0, 1)
	if win > 0 && target != nil {
		if m.rand.Intn(100) < 45 {
			killed = append(killed, target)
			delete(room.animals, target.ID)
			room.spawnAnimal(target.Animal, m.rand)
		} else {
			target.State = pb.EAnimalState_state_ice
		}
	}

	return BetOutcome{
		WinAmount:    win,
		RedBag:       red,
		FreeGold:     freeGold,
		KilledRoutes: killed,
	}
}

func (m *Manager) buildBetPushes(room *Room, session *PlayerSession, req *pb.M_1803Tos, outcome BetOutcome) []PushMessage {
	pushes := []PushMessage{}

	// 推送玩家打动物
	pushes = append(pushes, PushMessage{
		MsgID:   1899,
		ZooType: room.Type,
		Targets: room.otherPlayerIDs(session.Player.ID),
		Message: &pb.M_1899Toc{
			RoleId: proto.Uint32(session.Player.ID),
			Id:     proto.Uint32(req.GetId()),
		},
	})

	if outcome.WinAmount > 0 && len(outcome.KilledRoutes) > 0 {
		// 推送动物被打死
		pushes = append(pushes, PushMessage{
			MsgID:   1884,
			ZooType: room.Type,
			Targets: room.otherPlayerIDs(session.Player.ID),
			Message: &pb.M_1884Toc{
				RoleId: proto.Uint32(session.Player.ID),
				Type:   pb.EAnimalType_type_normal.Enum(),
				Ids:    buildKilledList(outcome),
			},
		})
	}

	if outcome.RedBag > 0 {
		room.redBag = true
	}

	return pushes
}

func buildKilledList(outcome BetOutcome) []*pb.PAnimalOne {
	if len(outcome.KilledRoutes) == 0 {
		return nil
	}

	list := make([]*pb.PAnimalOne, 0, len(outcome.KilledRoutes))
	perWin := outcome.WinAmount
	if len(outcome.KilledRoutes) > 0 && outcome.WinAmount > 0 {
		perWin = outcome.WinAmount / uint32(len(outcome.KilledRoutes))
		if perWin == 0 {
			perWin = outcome.WinAmount
		}
	}

	for _, route := range outcome.KilledRoutes {
		list = append(list, &pb.PAnimalOne{
			Id:     proto.Uint32(route.ID),
			Win:    proto.Uint32(perWin),
			RedBag: proto.Uint32(outcome.RedBag),
		})
	}

	return list
}
