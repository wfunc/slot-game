package animal

import (
	"math/rand"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
)

// RoomMessage 房间消息
type RoomMessage struct {
	Type    string
	Payload interface{}
}






// UpdateAnimals 更新动物位置和状态
func (r *Room) UpdateAnimals(deltaTime float32) (removedAnimals []uint32) {
	removedAnimals = []uint32{}
	now := time.Now()

	for id, animal := range r.animals {
		// 更新位置（简化实现）
		if animal.State == pb.EAnimalState_state_normal {
			// 每秒移动10个点
			animal.Point += uint32(10 * deltaTime)

			// 动物离开场景
			if animal.Point >= 100 {
				removedAnimals = append(removedAnimals, id)
			}
		}

		// 解冻检查
		if animal.State == pb.EAnimalState_state_ice {
			// 冰冻5秒后解冻
			if now.Sub(animal.SpawnAt).Seconds() > 5 {
				animal.State = pb.EAnimalState_state_normal
			}
		}
	}

	// 移除离开的动物
	for _, id := range removedAnimals {
		delete(r.animals, id)
	}

	return removedAnimals
}

// ProcessBet 处理下注
func (r *Room) ProcessBet(session *PlayerSession, targetID uint32, betAmount uint32) *BetOutcome {
	outcome := &BetOutcome{
		WinAmount:    0,
		RedBag:       0,
		SkillGain:    []*pb.PAnimalSkill{},
		FreeGold:     session.Player.FreeGold,
		KilledRoutes: []*AnimalRoute{},
	}

	// 查找目标动物
	animal, exists := r.animals[targetID]
	if !exists {
		return outcome
	}

	// 简化的命中判定（50%概率）
	if rand.Float32() < 0.5 {
		// 计算赔率（简化实现）
		odds := getAnimalOdds(animal.Animal)
		outcome.WinAmount = uint32(float32(betAmount) * odds)

		// 红包奖励
		if animal.Red {
			outcome.RedBag = uint32(rand.Intn(100) + 50)
		}

		// 记录被击杀的动物
		outcome.KilledRoutes = append(outcome.KilledRoutes, animal)

		// 处理特殊动物效果
		switch animal.Animal {
		case pb.EAnimal_pikachu:
			// 闪电链效果，击杀附近动物
			for id, nearbyAnimal := range r.animals {
				if id != targetID && rand.Float32() < 0.3 { // 30%概率连锁
					outcome.KilledRoutes = append(outcome.KilledRoutes, nearbyAnimal)
					delete(r.animals, id)
				}
			}

		case pb.EAnimal_bomber:
			// 全屏爆炸，击杀所有动物
			for id, a := range r.animals {
				if id != targetID {
					outcome.KilledRoutes = append(outcome.KilledRoutes, a)
					delete(r.animals, id)
				}
			}
		}

		// 移除被击杀的主目标
		delete(r.animals, targetID)
	}

	return outcome
}

// GetPlayerCount 获取房间玩家数量
func (r *Room) GetPlayerCount() int {
	return len(r.players)
}

// 辅助函数

// getAnimalOdds 获取动物基础赔率（简化实现）
func getAnimalOdds(animal pb.EAnimal) float32 {
	odds := map[pb.EAnimal]float32{
		pb.EAnimal_turtle:   1.5,
		pb.EAnimal_cock:     2.0,
		pb.EAnimal_dog:      2.5,
		pb.EAnimal_monkey:   3.0,
		pb.EAnimal_horse:    3.5,
		pb.EAnimal_ox:       4.0,
		pb.EAnimal_panda:    6.0,
		pb.EAnimal_hippo:    5.0,
		pb.EAnimal_lion:     5.5,
		pb.EAnimal_elephant: 8.0,
		pb.EAnimal_pikachu:  10.0,
		pb.EAnimal_bomber:   15.0,
		pb.EAnimal_tiger:    6.0,
		pb.EAnimal_sheep:    2.5,
		pb.EAnimal_bear:     5.0,
		pb.EAnimal_tuzi:     2.0,
		pb.EAnimal_lv:       2.5,
		pb.EAnimal_baozi:    7.0,
		pb.EAnimal_zhu:      3.5,
		pb.EAnimal_hema:     5.0,
	}

	if odd, exists := odds[animal]; exists {
		// 添加一些随机性 ±20%
		return odd * (0.8 + rand.Float32()*0.4)
	}
	return 1.0
}

// getAnimalSpeed 获取动物速度
func getAnimalSpeed(animal pb.EAnimal) float32 {
	// 基础速度 + 随机变化
	baseSpeed := float32(0.05)
	switch animal {
	case pb.EAnimal_pikachu:
		baseSpeed = 0.08
	case pb.EAnimal_bomber:
		baseSpeed = 0.06
	case pb.EAnimal_panda:
		baseSpeed = 0.03
	case pb.EAnimal_elephant:
		baseSpeed = 0.04
	}
	return baseSpeed * (0.8 + rand.Float32()*0.4)
}