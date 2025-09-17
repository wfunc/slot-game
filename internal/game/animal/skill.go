package animal

import (
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"google.golang.org/protobuf/proto"
)

// SkillType 技能类型
type SkillType int

const (
	SkillTypeIce        SkillType = iota + 1 // 冰冻技能
	SkillTypeLocking                          // 锁定技能
	SkillTypeOddsBoost                        // 倍率提升
	SkillTypeChain                            // 连锁效果
	SkillTypeExplosive                        // 爆炸效果
	SkillTypeOneBlow                          // 一击必杀
)

// Skill 技能
type Skill struct {
	Type      SkillType     // 技能类型
	PlayerID  uint32        // 玩家ID
	Level     int           // 技能等级
	Count     int           // 剩余次数
	Duration  time.Duration // 持续时间
	StartTime time.Time     // 开始时间
	EndTime   time.Time     // 结束时间
	Value     float32       // 技能数值（倍率、概率等）
	IsActive  bool          // 是否激活
	Cooldown  time.Duration // 冷却时间
	NextUse   time.Time     // 下次可用时间
}

// PlayerSkills 玩家技能管理
type PlayerSkills struct {
	PlayerID uint32
	Skills   map[SkillType]*Skill
	mu       sync.RWMutex
}

// SkillManager 技能管理器
type SkillManager struct {
	playerSkills map[uint32]*PlayerSkills
	skillPrices  map[SkillType]uint32
	mu           sync.RWMutex
}

// NewSkillManager 创建技能管理器
func NewSkillManager() *SkillManager {
	sm := &SkillManager{
		playerSkills: make(map[uint32]*PlayerSkills),
		skillPrices:  make(map[SkillType]uint32),
	}
	sm.initSkillPrices()
	return sm
}

// initSkillPrices 初始化技能价格
func (sm *SkillManager) initSkillPrices() {
	sm.skillPrices[SkillTypeIce] = 500          // 冰冻技能 500金币
	sm.skillPrices[SkillTypeLocking] = 800      // 锁定技能 800金币
	sm.skillPrices[SkillTypeOddsBoost] = 1000   // 倍率提升 1000金币
	sm.skillPrices[SkillTypeChain] = 1500       // 连锁效果 1500金币
	sm.skillPrices[SkillTypeExplosive] = 2000   // 爆炸效果 2000金币
	sm.skillPrices[SkillTypeOneBlow] = 5000     // 一击必杀 5000金币
}

// GetPlayerSkills 获取玩家技能
func (sm *SkillManager) GetPlayerSkills(playerID uint32) *PlayerSkills {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if ps, exists := sm.playerSkills[playerID]; exists {
		return ps
	}

	// 创建新的玩家技能
	ps := &PlayerSkills{
		PlayerID: playerID,
		Skills:   make(map[SkillType]*Skill),
	}
	sm.playerSkills[playerID] = ps
	return ps
}

// BuySkill 购买技能
func (sm *SkillManager) BuySkill(playerID uint32, skillType SkillType) (*Skill, uint32) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	price := sm.skillPrices[skillType]
	ps := sm.GetPlayerSkills(playerID)

	// 检查是否已有该技能
	if skill, exists := ps.Skills[skillType]; exists {
		// 增加技能次数
		skill.Count += sm.getSkillCount(skillType)
		return skill, price
	}

	// 创建新技能
	skill := &Skill{
		Type:     skillType,
		PlayerID: playerID,
		Level:    1,
		Count:    sm.getSkillCount(skillType),
		Duration: sm.getSkillDuration(skillType),
		Value:    sm.getSkillValue(skillType),
		Cooldown: sm.getSkillCooldown(skillType),
	}

	ps.mu.Lock()
	ps.Skills[skillType] = skill
	ps.mu.Unlock()

	return skill, price
}

// UseSkill 使用技能
func (sm *SkillManager) UseSkill(playerID uint32, skillType SkillType) (*Skill, error) {
	ps := sm.GetPlayerSkills(playerID)

	ps.mu.Lock()
	defer ps.mu.Unlock()

	skill, exists := ps.Skills[skillType]
	if !exists {
		return nil, ErrSkillNotFound
	}

	// 检查冷却
	if time.Now().Before(skill.NextUse) {
		return nil, ErrSkillCooldown
	}

	// 检查次数
	if skill.Count <= 0 {
		return nil, ErrSkillNoCount
	}

	// 使用技能
	skill.Count--
	skill.StartTime = time.Now()
	skill.EndTime = skill.StartTime.Add(skill.Duration)
	skill.IsActive = true
	skill.NextUse = time.Now().Add(skill.Cooldown)

	return skill, nil
}

// ApplySkillToAnimal 应用技能效果到动物
func (sm *SkillManager) ApplySkillToAnimal(skill *Skill, animal *Animal) {
	switch skill.Type {
	case SkillTypeIce:
		// 冰冻动物
		animal.Freeze(skill.Duration)

	case SkillTypeLocking:
		// 锁定动物
		animal.Lock(skill.PlayerID)

	case SkillTypeOddsBoost:
		// 提升赔率
		animal.ApplyOddsBoost(skill.Value)

	case SkillTypeOneBlow:
		// 一击必杀
		animal.TakeDamage(animal.MaxHP)
	}
}

// ApplySkillToBullet 应用技能效果到子弹
func (sm *SkillManager) ApplySkillToBullet(playerID uint32, bullet *Bullet) {
	ps := sm.GetPlayerSkills(playerID)

	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for skillType, skill := range ps.Skills {
		if !skill.IsActive {
			continue
		}

		// 检查技能是否过期
		if time.Now().After(skill.EndTime) {
			skill.IsActive = false
			continue
		}

		switch skillType {
		case SkillTypeIce:
			bullet.HasIceEffect = true
			bullet.IceDuration = skill.Duration

		case SkillTypeLocking:
			bullet.IsLocking = true
			bullet.LockStrength = skill.Value

		case SkillTypeOddsBoost:
			bullet.DamageMultiplier *= skill.Value

		case SkillTypeChain:
			bullet.HasChainEffect = true
			bullet.ChainProbability = skill.Value
			bullet.Penetration = 3

		case SkillTypeExplosive:
			bullet.ExplosiveRadius = 150
			bullet.Damage *= 2

		case SkillTypeOneBlow:
			bullet.Damage = 9999 // 超高伤害
		}
	}
}

// UpdateSkills 更新所有技能状态
func (sm *SkillManager) UpdateSkills() {
	sm.mu.RLock()
	players := make([]*PlayerSkills, 0, len(sm.playerSkills))
	for _, ps := range sm.playerSkills {
		players = append(players, ps)
	}
	sm.mu.RUnlock()

	now := time.Now()
	for _, ps := range players {
		ps.mu.Lock()
		for _, skill := range ps.Skills {
			if skill.IsActive && now.After(skill.EndTime) {
				skill.IsActive = false
			}
		}
		ps.mu.Unlock()
	}
}

// GetActiveSkills 获取玩家激活的技能
func (sm *SkillManager) GetActiveSkills(playerID uint32) []*Skill {
	ps := sm.GetPlayerSkills(playerID)

	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var activeSkills []*Skill
	for _, skill := range ps.Skills {
		if skill.IsActive {
			activeSkills = append(activeSkills, skill)
		}
	}
	return activeSkills
}

// GetSkillProtos 转换为Protobuf消息
func (sm *SkillManager) GetSkillProtos(playerID uint32) []*pb.PAnimalSkill {
	ps := sm.GetPlayerSkills(playerID)

	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var protos []*pb.PAnimalSkill
	for _, skill := range ps.Skills {
		if skill.Count > 0 {
			skillType := sm.skillTypeToPB(skill.Type)
			remainingTime := uint32(0)
			if skill.IsActive {
				remaining := skill.EndTime.Sub(time.Now())
				if remaining > 0 {
					remainingTime = uint32(remaining.Seconds())
				}
			}

			proto := &pb.PAnimalSkill{
				Type:  &skillType,
				Val:   proto.Uint32(uint32(skill.Count)),
				Time:  proto.Uint32(remainingTime),
				Count: proto.Uint32(uint32(skill.Level)),
			}
			protos = append(protos, proto)
		}
	}
	return protos
}

// 辅助函数

// getSkillCount 获取技能初始次数
func (sm *SkillManager) getSkillCount(skillType SkillType) int {
	counts := map[SkillType]int{
		SkillTypeIce:       10,
		SkillTypeLocking:   5,
		SkillTypeOddsBoost: 5,
		SkillTypeChain:     3,
		SkillTypeExplosive: 3,
		SkillTypeOneBlow:   1,
	}
	return counts[skillType]
}

// getSkillDuration 获取技能持续时间
func (sm *SkillManager) getSkillDuration(skillType SkillType) time.Duration {
	durations := map[SkillType]time.Duration{
		SkillTypeIce:       5 * time.Second,
		SkillTypeLocking:   10 * time.Second,
		SkillTypeOddsBoost: 15 * time.Second,
		SkillTypeChain:     10 * time.Second,
		SkillTypeExplosive: 0,
		SkillTypeOneBlow:   3 * time.Second,
	}
	return durations[skillType]
}

// getSkillValue 获取技能数值
func (sm *SkillManager) getSkillValue(skillType SkillType) float32 {
	values := map[SkillType]float32{
		SkillTypeIce:       1.0,
		SkillTypeLocking:   0.8, // 80%锁定强度
		SkillTypeOddsBoost: 2.0, // 2倍赔率
		SkillTypeChain:     0.3, // 30%连锁概率
		SkillTypeExplosive: 1.5, // 1.5倍伤害
		SkillTypeOneBlow:   1.0,
	}
	return values[skillType]
}

// getSkillCooldown 获取技能冷却时间
func (sm *SkillManager) getSkillCooldown(skillType SkillType) time.Duration {
	cooldowns := map[SkillType]time.Duration{
		SkillTypeIce:       2 * time.Second,
		SkillTypeLocking:   5 * time.Second,
		SkillTypeOddsBoost: 10 * time.Second,
		SkillTypeChain:     5 * time.Second,
		SkillTypeExplosive: 3 * time.Second,
		SkillTypeOneBlow:   30 * time.Second,
	}
	return cooldowns[skillType]
}

// skillTypeToPB 转换技能类型到Protobuf
func (sm *SkillManager) skillTypeToPB(skillType SkillType) pb.EAnimalSkillType {
	mapping := map[SkillType]pb.EAnimalSkillType{
		SkillTypeIce:        pb.EAnimalSkillType_skill_ice,
		SkillTypeLocking:    pb.EAnimalSkillType_locking,
		SkillTypeOddsBoost:  pb.EAnimalSkillType_improve_odds,
	}
	if pbType, exists := mapping[skillType]; exists {
		return pbType
	}
	return pb.EAnimalSkillType_skill_ice
}

// 错误定义
var (
	ErrSkillNotFound = &SkillError{Code: "SKILL_NOT_FOUND", Message: "技能未找到"}
	ErrSkillCooldown = &SkillError{Code: "SKILL_COOLDOWN", Message: "技能冷却中"}
	ErrSkillNoCount  = &SkillError{Code: "SKILL_NO_COUNT", Message: "技能次数不足"}
)

// SkillError 技能错误
type SkillError struct {
	Code    string
	Message string
}

func (e *SkillError) Error() string {
	return e.Message
}

