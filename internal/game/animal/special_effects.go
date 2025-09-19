package animal

import (
	"math"
	"math/rand"

	"github.com/wfunc/slot-game/internal/pb"
)

// SpecialEffectProcessor 特殊动物效果处理器
type SpecialEffectProcessor struct {
	room *Room
}

// NewSpecialEffectProcessor 创建特殊效果处理器
func NewSpecialEffectProcessor(room *Room) *SpecialEffectProcessor {
	return &SpecialEffectProcessor{room: room}
}

// ProcessPikachuLightning 处理皮卡丘闪电链效果
// 基于 Erlang calcPikachu 函数实现
func (p *SpecialEffectProcessor) ProcessPikachuLightning(targetID uint32, targetAnimal *AnimalRoute, betVal uint32, multiple uint32, roleID uint32) *BetOutcome {
	outcome := &BetOutcome{
		EffectType: pb.EAnimalType_lightning,
		ChainKills: []uint32{},
	}

	// 计算主目标奖励
	oddsSystem := &OddsSystem{}
	odds := oddsSystem.CalculateDynamicOdds(
		targetAnimal.Animal,
		p.room.Type,
		0, // TODO: 获取玩家VIP等级
		p.room.profitControl,
	)

	mainWin := uint32(float32(betVal*multiple) * odds)

	// 处理红包
	redBag, goldAmount := oddsSystem.CalculateRedPacket(targetAnimal.Animal, mainWin, targetAnimal.Red)
	outcome.RedBag = redBag
	outcome.GoldAmount = goldAmount
	outcome.WinAmount = mainWin

	// 添加主目标到击杀列表
	outcome.KilledRoutes = append(outcome.KilledRoutes, targetAnimal)

	// 查找附近的动物进行闪电链
	chainTargets := p.findChainTargets(targetAnimal, 5) // 最多连锁5只
	chainKillCount := 0

	for _, chainAnimal := range chainTargets {
		// 闪电链触发概率（基于距离递减）
		distance := p.calculateDistance(targetAnimal, chainAnimal)
		chainChance := 0.5 - (distance / 1000.0) // 距离越远概率越低

		if rand.Float32() < float32(chainChance) && chainKillCount < 3 {
			// 计算连锁伤害（递减）
			chainDamageRatio := 1.0 - float32(chainKillCount)*0.2 // 每次递减20%
			chainOdds := oddsSystem.CalculateDynamicOdds(
				chainAnimal.Animal,
				p.room.Type,
				0,
				p.room.profitControl,
			)

			chainWin := uint32(float32(betVal*multiple) * chainOdds * chainDamageRatio)
			outcome.WinAmount += chainWin

			// 处理连锁目标的红包
			if chainAnimal.Red {
				chainRedBag, chainGold := oddsSystem.CalculateRedPacket(chainAnimal.Animal, chainWin, true)
				outcome.RedBag += chainRedBag
				outcome.GoldAmount += chainGold - chainWin // 只加红包转换的额外金豆
			} else {
				outcome.GoldAmount += chainWin
			}

			outcome.KilledRoutes = append(outcome.KilledRoutes, chainAnimal)
			outcome.ChainKills = append(outcome.ChainKills, chainAnimal.ID)
			chainKillCount++
		}
	}

	return outcome
}

// ProcessBomberExplosion 处理炸弹人全屏爆炸效果
// 基于 Erlang calcBomber 函数实现
func (p *SpecialEffectProcessor) ProcessBomberExplosion(targetID uint32, targetAnimal *AnimalRoute, betVal uint32, multiple uint32, roleID uint32) *BetOutcome {
	outcome := &BetOutcome{
		EffectType: pb.EAnimalType_boom,
		ChainKills: []uint32{},
	}

	oddsSystem := &OddsSystem{}
	totalWin := uint32(0)
	totalRedBag := uint32(0)
	totalGold := uint32(0)

	// 炸弹人本身没有赔率，只计算爆炸效果
	// 遍历所有动物，除了特殊动物外全部击杀
	for id, animal := range p.room.animals {
		// 跳过炸弹人自己
		if id == targetID {
			continue
		}

		// 特殊动物免疫爆炸（皮卡丘、其他炸弹人）
		if animal.Animal == pb.EAnimal_pikachu || animal.Animal == pb.EAnimal_bomber {
			continue
		}

		// 计算每个动物的奖励
		odds := oddsSystem.CalculateDynamicOdds(
			animal.Animal,
			p.room.Type,
			0,
			p.room.profitControl,
		)

		// 爆炸伤害有一定衰减（根据动物价值）
		damageRatio := p.getBomberDamageRatio(animal.Animal)
		win := uint32(float32(betVal*multiple) * odds * damageRatio)
		totalWin += win

		// 处理红包
		if animal.Red {
			redBag, gold := oddsSystem.CalculateRedPacket(animal.Animal, win, true)
			totalRedBag += redBag
			totalGold += gold
		} else {
			totalGold += win
		}

		outcome.KilledRoutes = append(outcome.KilledRoutes, animal)
		outcome.ChainKills = append(outcome.ChainKills, id)
	}

	// 添加炸弹人自己到击杀列表（虽然它没有赔率）
	outcome.KilledRoutes = append(outcome.KilledRoutes, targetAnimal)

	outcome.WinAmount = totalWin
	outcome.RedBag = totalRedBag
	outcome.GoldAmount = totalGold

	return outcome
}

// findChainTargets 查找闪电链目标
func (p *SpecialEffectProcessor) findChainTargets(source *AnimalRoute, maxTargets int) []*AnimalRoute {
	var targets []*AnimalRoute

	// 计算所有动物与源动物的距离
	type distanceAnimal struct {
		animal   *AnimalRoute
		distance float32
	}

	var candidates []distanceAnimal

	for id, animal := range p.room.animals {
		// 跳过源动物和已经死亡的动物
		if id == source.ID || animal == nil {
			continue
		}

		// 跳过被冰冻的动物（可选）
		if animal.State == pb.EAnimalState_state_ice {
			continue
		}

		dist := p.calculateDistance(source, animal)
		candidates = append(candidates, distanceAnimal{
			animal:   animal,
			distance: dist,
		})
	}

	// 按距离排序
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].distance > candidates[j].distance {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// 选择最近的目标
	for i := 0; i < len(candidates) && len(targets) < maxTargets; i++ {
		targets = append(targets, candidates[i].animal)
	}

	return targets
}

// calculateDistance 计算两个动物之间的距离
func (p *SpecialEffectProcessor) calculateDistance(a1, a2 *AnimalRoute) float32 {
	// 简化计算：基于路线ID和点位置
	// 同一条路线上的动物距离较近
	if a1.LineID == a2.LineID {
		return float32(math.Abs(float64(a1.Point - a2.Point)))
	}

	// 不同路线的动物距离较远（基础距离 + 点差）
	baseDist := float32(100.0)
	pointDiff := math.Abs(float64(a1.Point - a2.Point))
	return baseDist + float32(pointDiff)
}

// getBomberDamageRatio 获取炸弹人对不同动物的伤害系数
func (p *SpecialEffectProcessor) getBomberDamageRatio(animal pb.EAnimal) float32 {
	// 根据动物价值设置不同的伤害系数
	// 低价值动物受到全额伤害，高价值动物受到部分伤害
	switch animal {
	case pb.EAnimal_turtle, pb.EAnimal_cock, pb.EAnimal_dog:
		return 1.0 // 低价值动物，全额伤害
	case pb.EAnimal_monkey, pb.EAnimal_horse, pb.EAnimal_ox:
		return 0.8 // 中等价值动物，80%伤害
	case pb.EAnimal_panda, pb.EAnimal_hippo:
		return 0.6 // 高价值动物，60%伤害
	case pb.EAnimal_lion, pb.EAnimal_elephant:
		return 0.4 // 超高价值动物，40%伤害
	default:
		return 0.7 // 默认70%伤害
	}
}

// ProcessNormalKill 处理普通击杀
func (p *SpecialEffectProcessor) ProcessNormalKill(targetID uint32, targetAnimal *AnimalRoute, betVal uint32, multiple uint32, roleID uint32) *BetOutcome {
	outcome := &BetOutcome{
		EffectType: pb.EAnimalType_type_normal,
	}

	// 计算赔率
	oddsSystem := &OddsSystem{}
	odds := oddsSystem.CalculateDynamicOdds(
		targetAnimal.Animal,
		p.room.Type,
		0, // TODO: 获取玩家VIP等级
		p.room.profitControl,
	)

	// 计算基础奖励
	win := uint32(float32(betVal*multiple) * odds)

	// 处理红包
	redBag, goldAmount := oddsSystem.CalculateRedPacket(targetAnimal.Animal, win, targetAnimal.Red)

	outcome.WinAmount = win
	outcome.RedBag = redBag
	outcome.GoldAmount = goldAmount
	outcome.KilledRoutes = append(outcome.KilledRoutes, targetAnimal)

	// 检查是否触发彩金池（击杀大象有机会）
	if targetAnimal.Animal == pb.EAnimal_elephant && p.room.jackpot != nil {
		// 暂时注释彩金池触发，需要实现 TryTrigger 方法
		// if jackpotWin := p.room.jackpot.TryTrigger(roleID); jackpotWin > 0 {
		jackpotWin := uint64(0)
		if false {
			outcome.JackpotWin = jackpotWin
			outcome.GoldAmount += uint32(jackpotWin)
		}
	}

	return outcome
}