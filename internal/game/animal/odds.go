package animal

import (
	"math/rand"
	"sync"

	"github.com/wfunc/slot-game/internal/pb"
)

// OddsSystem 赔率系统
type OddsSystem struct {
	mu sync.RWMutex
}

// AnimalOddsNormal 正式场赔率（需要除以10）
var AnimalOddsNormal = map[pb.EAnimal][2]float32{
	pb.EAnimal_turtle:   {7, 14},       // 0.7-1.4
	pb.EAnimal_cock:     {12, 12},      // 1.2
	pb.EAnimal_dog:      {20, 20},      // 2.0
	pb.EAnimal_monkey:   {40, 40},      // 4.0
	pb.EAnimal_horse:    {60, 60},      // 6.0
	pb.EAnimal_tuzi:     {80, 80},      // 8.0
	pb.EAnimal_ox:       {100, 100},    // 10.0
	pb.EAnimal_panda:    {200, 200},    // 20.0
	pb.EAnimal_lv:       {300, 300},    // 30.0
	pb.EAnimal_baozi:    {400, 400},    // 40.0
	pb.EAnimal_zhu:      {600, 600},    // 60.0
	pb.EAnimal_hema:     {800, 800},    // 80.0
	pb.EAnimal_hippo:    {1000, 1000},  // 100.0
	pb.EAnimal_pikachu:  {500, 500},    // 50.0
	pb.EAnimal_lion:     {2000, 2000},  // 200.0
	pb.EAnimal_elephant: {10000, 10000}, // 1000.0
	pb.EAnimal_bomber:   {0, 0},         // 特殊（炸弹人）
	pb.EAnimal_tiger:    {600, 600},    // 60.0
	pb.EAnimal_sheep:    {250, 250},    // 25.0
	pb.EAnimal_bear:     {500, 500},    // 50.0
}

// AnimalOddsFree 体验场赔率（固定值，除以10）
var AnimalOddsFree = map[pb.EAnimal]float32{
	pb.EAnimal_turtle:   2.0,
	pb.EAnimal_cock:     4.0,
	pb.EAnimal_dog:      6.0,
	pb.EAnimal_monkey:   10.0,
	pb.EAnimal_horse:    20.0,
	pb.EAnimal_ox:       30.0,
	pb.EAnimal_panda:    50.0,
	pb.EAnimal_hippo:    100.0,
	pb.EAnimal_lion:     200.0,
	pb.EAnimal_elephant: 300.0,
	pb.EAnimal_pikachu:  50.0,
	pb.EAnimal_bomber:   0.0, // 特殊处理
}

// CalculateDynamicOdds 计算动态赔率
func (o *OddsSystem) CalculateDynamicOdds(animal pb.EAnimal, roomType pb.EZooType, vipLevel uint32, profitControl *RoomProfitControl) float32 {
	// 体验场使用固定赔率
	if roomType == pb.EZooType_free {
		if odds, exists := AnimalOddsFree[animal]; exists {
			return odds
		}
		return 1.0
	}

	// 正式场使用动态赔率
	baseRange, exists := AnimalOddsNormal[animal]
	if !exists {
		return 1.0
	}

	// 获取赔率范围
	minOdds := baseRange[0] / 10.0
	maxOdds := baseRange[1] / 10.0

	// 在范围内随机
	var odds float32
	if minOdds == maxOdds {
		odds = minOdds
	} else {
		odds = minOdds + rand.Float32()*(maxOdds-minOdds)
	}

	// VIP加成（每级VIP增加2%赔率）
	vipBonus := 1.0 + float32(vipLevel)*0.02
	odds *= vipBonus

	// 根据房间盈亏调整赔率
	if profitControl != nil {
		odds = o.adjustOddsByProfit(odds, profitControl)
	}

	return odds
}

// adjustOddsByProfit 根据房间盈亏调整赔率
func (o *OddsSystem) adjustOddsByProfit(baseOdds float32, profitControl *RoomProfitControl) float32 {
	profitControl.mu.RLock()
	totalBet := profitControl.TotalBet
	totalWin := profitControl.TotalWin
	profitControl.mu.RUnlock()

	// 计算盈亏差值
	diff := int64(totalWin) - int64(totalBet)

	// 房间亏损过大（玩家赢太多），降低赔率
	if diff > 10000000 {
		return baseOdds * 0.8 // 降低20%赔率
	}

	// 房间盈利过大（玩家输太多），提高赔率
	if diff < -10000000 {
		return baseOdds * 1.2 // 提高20%赔率
	}

	return baseOdds
}

// ShouldHit 计算是否命中（考虑房间盈亏）
func (o *OddsSystem) ShouldHit(profitControl *RoomProfitControl, baseHitRate float32) bool {
	hitRate := baseHitRate

	if profitControl != nil {
		profitControl.mu.RLock()
		totalBet := profitControl.TotalBet
		totalWin := profitControl.TotalWin
		profitControl.mu.RUnlock()

		diff := int64(totalWin) - int64(totalBet)

		// 房间亏损过大，降低命中率
		if diff > 10000000 {
			// 30分之1的概率强制不中
			if rand.Intn(30) == 0 {
				return false
			}
			hitRate *= 0.9 // 降低10%命中率
		}

		// 房间盈利过大，提高命中率
		if diff < -10000000 {
			hitRate *= 1.1 // 提高10%命中率
		}
	}

	return rand.Float32() < hitRate
}

// GetAnimalOddsRange 获取动物赔率范围（用于客户端显示）
func (o *OddsSystem) GetAnimalOddsRange(roomType pb.EZooType) []*pb.PAnimalOdds {
	var result []*pb.PAnimalOdds

	if roomType == pb.EZooType_free {
		// 体验场返回固定赔率
		for animal, odds := range AnimalOddsFree {
			a := animal // 创建局部变量
			result = append(result, &pb.PAnimalOdds{
				Bet:  &a,
				Odds: []uint32{uint32(odds * 10), uint32(odds * 10)},
			})
		}
	} else {
		// 正式场返回赔率范围
		for animal, oddsRange := range AnimalOddsNormal {
			a := animal // 创建局部变量
			result = append(result, &pb.PAnimalOdds{
				Bet:  &a,
				Odds: []uint32{uint32(oddsRange[0]), uint32(oddsRange[1])},
			})
		}
	}

	return result
}

// CalculateRedPacket 计算红包奖励
func (o *OddsSystem) CalculateRedPacket(animal pb.EAnimal, winAmount uint32, hasRedBag bool) (uint32, uint32) {
	if !hasRedBag {
		return 0, winAmount
	}

	// 基于动物类型的红包概率和倍率
	redBagConfig := map[pb.EAnimal]struct {
		chance float32
		min    float32
		max    float32
	}{
		pb.EAnimal_turtle:   {0.1, 0.05, 0.1},
		pb.EAnimal_cock:     {0.15, 0.05, 0.1},
		pb.EAnimal_dog:      {0.2, 0.05, 0.15},
		pb.EAnimal_monkey:   {0.25, 0.1, 0.2},
		pb.EAnimal_panda:    {0.3, 0.15, 0.3},
		pb.EAnimal_elephant: {0.5, 0.2, 0.5},
		pb.EAnimal_pikachu:  {0.35, 0.15, 0.25},
	}

	config, exists := redBagConfig[animal]
	if !exists {
		config = struct {
			chance float32
			min    float32
			max    float32
		}{0.2, 0.05, 0.1} // 默认配置
	}

	// 检查是否触发红包
	if rand.Float32() > config.chance {
		return 0, winAmount
	}

	// 计算红包金额（基于赢取金额的百分比）
	redBagRatio := config.min + rand.Float32()*(config.max-config.min)
	redBag := uint32(float32(winAmount) * redBagRatio)

	// 红包转金豆（1元红包 = 1200金豆）
	goldFromRedBag := redBag * 1200
	totalGold := winAmount + goldFromRedBag

	return redBag, totalGold
}