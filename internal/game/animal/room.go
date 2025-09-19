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

// ProcessBet 处理下注（完整实现）
func (r *Room) ProcessBet(session *PlayerSession, targetID uint32, betAmount uint32, multiple uint32) *BetOutcome {
	// 初始化结果
	outcome := &BetOutcome{
		WinAmount:    0,
		RedBag:       0,
		GoldAmount:   0,
		SkillGain:    []*pb.PAnimalSkill{},
		FreeGold:     session.Player.FreeGold,
		KilledRoutes: []*AnimalRoute{},
		EffectType:   pb.EAnimalType_type_normal,
		ChainKills:   []uint32{},
	}

	// 查找目标动物
	animal, exists := r.animals[targetID]
	if !exists {
		return outcome
	}

	// 检查一击必杀权限
	isOneBlow := false
	if r.oneBlowManager != nil && r.oneBlowManager.CheckOneBlow(session.Player.ID) {
		isOneBlow = true
	}

	// 使用新的赔率系统计算命中
	oddsSystem := &OddsSystem{}
	baseHitRate := float32(0.5) // 基础命中率

	// 一击必杀直接命中
	if isOneBlow {
		baseHitRate = 1.0
	}

	// 根据房间盈亏调整命中率
	if oddsSystem.ShouldHit(r.profitControl, baseHitRate) {
		// 处理特殊动物效果
		specialProcessor := NewSpecialEffectProcessor(r)

		switch animal.Animal {
		case pb.EAnimal_pikachu:
			// 皮卡丘闪电链效果
			outcome = specialProcessor.ProcessPikachuLightning(targetID, animal, betAmount, multiple, session.Player.ID)

		case pb.EAnimal_bomber:
			// 炸弹人全屏爆炸
			outcome = specialProcessor.ProcessBomberExplosion(targetID, animal, betAmount, multiple, session.Player.ID)

		default:
			// 普通击杀
			outcome = specialProcessor.ProcessNormalKill(targetID, animal, betAmount, multiple, session.Player.ID)
		}

		// 消耗一击必杀次数
		if isOneBlow && r.oneBlowManager != nil {
			r.oneBlowManager.ConsumeOneBlow(session.Player.ID)
		}

		// 更新房间盈亏控制
		if r.profitControl != nil {
			r.profitControl.mu.Lock()
			r.profitControl.TotalBet += uint64(betAmount * multiple)
			r.profitControl.TotalWin += uint64(outcome.WinAmount)
			r.profitControl.mu.Unlock()
		}

		// 彩金池累积
		if r.jackpot != nil {
			r.jackpot.Accumulate(uint64(betAmount*multiple), session.Player.ID)

			// 尝试触发彩金
			if triggered, winAmount := r.jackpot.TryTrigger(session.Player.ID, uint64(betAmount*multiple)); triggered {
				outcome.JackpotWin = winAmount
				outcome.GoldAmount += uint32(winAmount)
			}
		}

		// 更新任务进度
		if r.taskManager != nil {
			// 击杀事件
			for _, killed := range outcome.KilledRoutes {
				r.taskManager.UpdateProgress(TaskEvent{
					Type:     "kill",
					Animal:   killed.Animal,
					RoomType: r.Type,
					Count:    1,
				})
			}

			// 下注事件
			r.taskManager.UpdateProgress(TaskEvent{
				Type:      "bet",
				BetAmount: betAmount * multiple,
				BetLevel:  betAmount,
				RoomType:  r.Type,
			})

			// 赢取事件
			if outcome.WinAmount > 0 {
				r.taskManager.UpdateProgress(TaskEvent{
					Type:      "win",
					WinAmount: outcome.WinAmount,
					RoomType:  r.Type,
					IsJackpot: outcome.JackpotWin > 0,
				})
			}
		}

		// 移除被击杀的动物
		for _, killedAnimal := range outcome.KilledRoutes {
			delete(r.animals, killedAnimal.ID)
		}
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