package slot

import (
	"testing"
)

func TestGetDefaultAnimalTriggerConfig(t *testing.T) {
	config := GetDefaultAnimalTriggerConfig()
	
	if !config.Enabled {
		t.Error("默认配置应该启用Animal触发")
	}
	
	// 测试普通触发配置
	if config.NormalTrigger.SymbolID != SYMBOL_ANIMAL_WILD {
		t.Errorf("普通触发符号ID = %v, want %v", config.NormalTrigger.SymbolID, SYMBOL_ANIMAL_WILD)
	}
	if config.NormalTrigger.RequiredCount != 3 {
		t.Errorf("普通触发需要符号数 = %v, want %v", config.NormalTrigger.RequiredCount, 3)
	}
	if config.NormalTrigger.FreeRounds != 15 {
		t.Errorf("普通触发免费轮数 = %v, want %v", config.NormalTrigger.FreeRounds, 15)
	}
	if config.NormalTrigger.Multiplier != 1.5 {
		t.Errorf("普通触发倍率 = %v, want %v", config.NormalTrigger.Multiplier, 1.5)
	}
	
	// 测试超级触发配置
	if config.SuperTrigger.SymbolID != SYMBOL_ANIMAL_BONUS {
		t.Errorf("超级触发符号ID = %v, want %v", config.SuperTrigger.SymbolID, SYMBOL_ANIMAL_BONUS)
	}
	if config.SuperTrigger.RequiredCount != 5 {
		t.Errorf("超级触发需要符号数 = %v, want %v", config.SuperTrigger.RequiredCount, 5)
	}
	if config.SuperTrigger.FreeRounds != 30 {
		t.Errorf("超级触发免费轮数 = %v, want %v", config.SuperTrigger.FreeRounds, 30)
	}
	if config.SuperTrigger.Multiplier != 3.0 {
		t.Errorf("超级触发倍率 = %v, want %v", config.SuperTrigger.Multiplier, 3.0)
	}
	if config.SuperTrigger.BonusPool != 10000 {
		t.Errorf("超级触发奖池 = %v, want %v", config.SuperTrigger.BonusPool, 10000)
	}
}

func TestDetectAnimalTrigger(t *testing.T) {
	detector := NewTriggerDetector(nil) // 使用默认配置
	
	tests := []struct {
		name         string
		grid         [][]int
		expectType   TriggerType
		expectRounds int
	}{
		{
			name: "无触发",
			grid: [][]int{
				{0, 1, 2, 3, 4},
				{1, 2, 3, 4, 5},
				{2, 3, 4, 5, 6},
				{3, 4, 5, 6, 7},
			},
			expectType:   TriggerTypeNone,
			expectRounds: 0,
		},
		{
			name: "3个Animal Wild触发普通游戏",
			grid: [][]int{
				{SYMBOL_ANIMAL_WILD, 1, 2, 3, 4},
				{1, SYMBOL_ANIMAL_WILD, 3, 4, 5},
				{2, 3, SYMBOL_ANIMAL_WILD, 5, 6},
				{3, 4, 5, 6, 7},
			},
			expectType:   TriggerTypeAnimalNormal,
			expectRounds: 15,
		},
		{
			name: "4个Animal Wild触发普通游戏（增加轮数）",
			grid: [][]int{
				{SYMBOL_ANIMAL_WILD, 1, SYMBOL_ANIMAL_WILD, 3, 4},
				{1, SYMBOL_ANIMAL_WILD, 3, 4, 5},
				{2, 3, SYMBOL_ANIMAL_WILD, 5, 6},
				{3, 4, 5, 6, 7},
			},
			expectType:   TriggerTypeAnimalNormal,
			expectRounds: 20, // 15 + 5
		},
		{
			name: "5个Animal Bonus触发超级游戏",
			grid: [][]int{
				{SYMBOL_ANIMAL_BONUS, 1, SYMBOL_ANIMAL_BONUS, 3, 4},
				{1, SYMBOL_ANIMAL_BONUS, 3, 4, 5},
				{SYMBOL_ANIMAL_BONUS, 3, SYMBOL_ANIMAL_BONUS, 5, 6},
				{3, 4, 5, 6, 7},
			},
			expectType:   TriggerTypeAnimalSuper,
			expectRounds: 30,
		},
		{
			name: "混合符号（Bonus优先）",
			grid: [][]int{
				{SYMBOL_ANIMAL_WILD, SYMBOL_ANIMAL_BONUS, SYMBOL_ANIMAL_WILD, 3, 4},
				{SYMBOL_ANIMAL_BONUS, SYMBOL_ANIMAL_WILD, SYMBOL_ANIMAL_BONUS, 4, 5},
				{SYMBOL_ANIMAL_BONUS, 3, SYMBOL_ANIMAL_BONUS, 5, 6},
				{3, 4, 5, 6, 7},
			},
			expectType:   TriggerTypeAnimalSuper,
			expectRounds: 30,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := detector.DetectAnimalTrigger(tt.grid)
			
			if tt.expectType == TriggerTypeNone {
				if trigger != nil {
					t.Errorf("期望无触发，但得到了触发: %v", trigger.Type)
				}
				return
			}
			
			if trigger == nil {
				t.Errorf("期望触发类型 %v，但没有触发", tt.expectType)
				return
			}
			
			if trigger.Type != tt.expectType {
				t.Errorf("触发类型 = %v, want %v", trigger.Type, tt.expectType)
			}
			
			if trigger.FreeRounds != tt.expectRounds {
				t.Errorf("免费轮数 = %v, want %v", trigger.FreeRounds, tt.expectRounds)
			}
		})
	}
}

func TestCountTriggerSymbols(t *testing.T) {
	detector := NewTriggerDetector(nil)
	
	grid := [][]int{
		{SYMBOL_ANIMAL_WILD, 1, SYMBOL_ANIMAL_BONUS, 3, 4},
		{1, SYMBOL_ANIMAL_WILD, 3, 4, 5},
		{SYMBOL_ANIMAL_BONUS, 3, SYMBOL_ANIMAL_WILD, 5, 6},
		{3, 4, 5, 6, 7},
	}
	
	counts := detector.CountTriggerSymbols(grid)
	
	if counts[SYMBOL_ANIMAL_WILD] != 3 {
		t.Errorf("Animal Wild数量 = %v, want %v", counts[SYMBOL_ANIMAL_WILD], 3)
	}
	
	if counts[SYMBOL_ANIMAL_BONUS] != 2 {
		t.Errorf("Animal Bonus数量 = %v, want %v", counts[SYMBOL_ANIMAL_BONUS], 2)
	}
}

func TestValidateTriggerConfig(t *testing.T) {
	detector := NewTriggerDetector(nil)
	
	// 默认配置应该有效
	if err := detector.ValidateTriggerConfig(); err != nil {
		t.Errorf("默认配置验证失败: %v", err)
	}
	
	// 测试无效配置
	invalidConfig := &AnimalTriggerConfig{
		Enabled: true,
	}
	invalidConfig.NormalTrigger.RequiredCount = 0
	
	detector = NewTriggerDetector(invalidConfig)
	if err := detector.ValidateTriggerConfig(); err == nil {
		t.Error("无效配置应该验证失败")
	}
}

func TestTriggerDetectorDisabled(t *testing.T) {
	config := &AnimalTriggerConfig{
		Enabled: false,
	}
	detector := NewTriggerDetector(config)
	
	// 即使有触发符号，禁用时也不应该触发
	grid := [][]int{
		{SYMBOL_ANIMAL_WILD, SYMBOL_ANIMAL_WILD, SYMBOL_ANIMAL_WILD, 3, 4},
		{1, 2, 3, 4, 5},
		{2, 3, 4, 5, 6},
		{3, 4, 5, 6, 7},
	}
	
	trigger := detector.DetectAnimalTrigger(grid)
	if trigger != nil {
		t.Error("禁用时不应该检测到触发")
	}
}

func TestCopyGrid(t *testing.T) {
	original := [][]int{
		{1, 2, 3},
		{4, 5, 6},
	}
	
	copied := copyGrid(original)
	
	// 验证内容相同
	for i := range original {
		for j := range original[i] {
			if copied[i][j] != original[i][j] {
				t.Errorf("复制的网格在位置[%d][%d]不匹配", i, j)
			}
		}
	}
	
	// 验证是独立的副本
	copied[0][0] = 999
	if original[0][0] == 999 {
		t.Error("修改副本不应该影响原始网格")
	}
}