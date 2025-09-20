package slot

import (
	"fmt"
)

// TriggerType 触发类型
type TriggerType string

const (
	TriggerTypeNone         TriggerType = "none"
	TriggerTypeAnimalNormal TriggerType = "animal_normal" // 普通Animal游戏
	TriggerTypeAnimalSuper  TriggerType = "animal_super"  // 超级Animal游戏
	TriggerTypeFreeSpins    TriggerType = "free_spins"    // 免费旋转
)

// AnimalTriggerConfig Animal游戏触发配置
type AnimalTriggerConfig struct {
	Enabled bool `json:"enabled"` // 是否启用Animal触发

	// 普通Animal游戏触发条件
	NormalTrigger struct {
		SymbolID      int     `json:"symbol_id"`      // 触发符号ID (SYMBOL_ANIMAL_WILD)
		RequiredCount int     `json:"required_count"` // 需要的符号数量
		FreeRounds    int     `json:"free_rounds"`    // 免费游戏轮数
		Multiplier    float64 `json:"multiplier"`     // 倍率加成
	} `json:"normal_trigger"`

	// 超级Animal游戏触发条件
	SuperTrigger struct {
		SymbolID      int     `json:"symbol_id"`      // 触发符号ID (SYMBOL_ANIMAL_BONUS)
		RequiredCount int     `json:"required_count"` // 需要的符号数量
		FreeRounds    int     `json:"free_rounds"`    // 免费游戏轮数
		Multiplier    float64 `json:"multiplier"`     // 倍率加成
		BonusPool     int64   `json:"bonus_pool"`     // 额外奖池
	} `json:"super_trigger"`
}

// AnimalTriggerData Animal游戏触发数据
type AnimalTriggerData struct {
	Type        TriggerType    `json:"type"`         // 触发类型
	SymbolID    int            `json:"symbol_id"`    // 触发符号ID
	SymbolCount int            `json:"symbol_count"` // 触发符号数量
	Positions   []GamePosition `json:"positions"`    // 触发符号位置
	FreeRounds  int            `json:"free_rounds"`  // 免费轮数
	Multiplier  float64        `json:"multiplier"`   // 倍率
	BonusPool   int64          `json:"bonus_pool"`   // 奖池金额
	TriggerGrid [][]int        `json:"trigger_grid"` // 触发时的网格状态
}

// GetDefaultAnimalTriggerConfig 获取默认的Animal触发配置
func GetDefaultAnimalTriggerConfig() *AnimalTriggerConfig {
	config := &AnimalTriggerConfig{
		Enabled: true,
	}

	// 普通Animal游戏配置
	config.NormalTrigger.SymbolID = SYMBOL_ANIMAL_WILD
	config.NormalTrigger.RequiredCount = 3
	config.NormalTrigger.FreeRounds = 15
	config.NormalTrigger.Multiplier = 1.5

	// 超级Animal游戏配置
	config.SuperTrigger.SymbolID = SYMBOL_ANIMAL_BONUS
	config.SuperTrigger.RequiredCount = 5
	config.SuperTrigger.FreeRounds = 30
	config.SuperTrigger.Multiplier = 3.0
	config.SuperTrigger.BonusPool = 10000

	return config
}

// TriggerDetector Animal触发检测器
type TriggerDetector struct {
	config *AnimalTriggerConfig
}

// NewTriggerDetector 创建触发检测器
func NewTriggerDetector(config *AnimalTriggerConfig) *TriggerDetector {
	if config == nil {
		config = GetDefaultAnimalTriggerConfig()
	}
	return &TriggerDetector{
		config: config,
	}
}

// DetectAnimalTrigger 检测是否触发Animal游戏
func (d *TriggerDetector) DetectAnimalTrigger(grid [][]int) *AnimalTriggerData {
	if !d.config.Enabled {
		return nil
	}

	// 统计符号出现次数和位置
	symbolCounts := make(map[int]int)
	symbolPositions := make(map[int][]GamePosition)

	for row := 0; row < len(grid); row++ {
		for col := 0; col < len(grid[row]); col++ {
			symbolID := grid[row][col]
			if IsAnimalTriggerSymbol(symbolID) {
				symbolCounts[symbolID]++
				symbolPositions[symbolID] = append(symbolPositions[symbolID], GamePosition{
					Row:  row,
					Reel: col,
				})
			}
		}
	}

	// 检查超级Animal游戏触发（优先级更高）
	if count := symbolCounts[d.config.SuperTrigger.SymbolID]; count >= d.config.SuperTrigger.RequiredCount {
		return &AnimalTriggerData{
			Type:        TriggerTypeAnimalSuper,
			SymbolID:    d.config.SuperTrigger.SymbolID,
			SymbolCount: count,
			Positions:   symbolPositions[d.config.SuperTrigger.SymbolID],
			FreeRounds:  d.config.SuperTrigger.FreeRounds,
			Multiplier:  d.config.SuperTrigger.Multiplier,
			BonusPool:   d.config.SuperTrigger.BonusPool,
			TriggerGrid: copyGrid(grid),
		}
	}

	// 检查普通Animal游戏触发
	if count := symbolCounts[d.config.NormalTrigger.SymbolID]; count >= d.config.NormalTrigger.RequiredCount {
		return &AnimalTriggerData{
			Type:        TriggerTypeAnimalNormal,
			SymbolID:    d.config.NormalTrigger.SymbolID,
			SymbolCount: count,
			Positions:   symbolPositions[d.config.NormalTrigger.SymbolID],
			FreeRounds:  d.config.NormalTrigger.FreeRounds + (count-d.config.NormalTrigger.RequiredCount)*5, // 每多一个符号增加5轮
			Multiplier:  d.config.NormalTrigger.Multiplier,
			BonusPool:   0,
			TriggerGrid: copyGrid(grid),
		}
	}

	return nil
}

// CountTriggerSymbols 统计触发符号数量
func (d *TriggerDetector) CountTriggerSymbols(grid [][]int) map[int]int {
	counts := make(map[int]int)
	for row := 0; row < len(grid); row++ {
		for col := 0; col < len(grid[row]); col++ {
			symbolID := grid[row][col]
			if IsAnimalTriggerSymbol(symbolID) {
				counts[symbolID]++
			}
		}
	}
	return counts
}

// GetTriggerProbability 获取触发概率（用于RTP计算）
func (d *TriggerDetector) GetTriggerProbability(symbolWeights [][]int) float64 {
	// 简化计算：假设每个位置独立
	// 实际应该根据符号权重计算准确概率
	totalWeight := 0
	triggerWeight := 0

	for _, reelWeights := range symbolWeights {
		for symbolID, weight := range reelWeights {
			totalWeight += weight
			if IsAnimalTriggerSymbol(symbolID) {
				triggerWeight += weight
			}
		}
	}

	if totalWeight == 0 {
		return 0
	}

	// 简化概率计算
	singleProb := float64(triggerWeight) / float64(totalWeight)

	// 计算至少3个符号出现的概率（二项分布）
	// 这里使用简化计算，实际应该更精确
	gridSize := len(symbolWeights) * 4 // 5列 x 4行
	minRequired := d.config.NormalTrigger.RequiredCount

	// 使用近似公式
	prob := 1.0
	for i := 0; i < minRequired; i++ {
		prob *= singleProb * float64(gridSize-i) / float64(i+1)
	}

	return prob
}

// ValidateTriggerConfig 验证触发配置
func (d *TriggerDetector) ValidateTriggerConfig() error {
	if d.config == nil {
		return fmt.Errorf("触发配置为空")
	}

	if d.config.Enabled {
		// 验证普通触发配置
		if d.config.NormalTrigger.RequiredCount <= 0 {
			return fmt.Errorf("普通触发所需符号数量必须大于0")
		}
		if d.config.NormalTrigger.FreeRounds <= 0 {
			return fmt.Errorf("普通触发免费轮数必须大于0")
		}
		if d.config.NormalTrigger.Multiplier <= 0 {
			return fmt.Errorf("普通触发倍率必须大于0")
		}

		// 验证超级触发配置
		if d.config.SuperTrigger.RequiredCount <= 0 {
			return fmt.Errorf("超级触发所需符号数量必须大于0")
		}
		if d.config.SuperTrigger.FreeRounds <= 0 {
			return fmt.Errorf("超级触发免费轮数必须大于0")
		}
		if d.config.SuperTrigger.Multiplier <= 0 {
			return fmt.Errorf("超级触发倍率必须大于0")
		}
	}

	return nil
}

// copyGrid 复制网格
func copyGrid(grid [][]int) [][]int {
	newGrid := make([][]int, len(grid))
	for i := range grid {
		newGrid[i] = make([]int, len(grid[i]))
		copy(newGrid[i], grid[i])
	}
	return newGrid
}
