package slot

import (
	"testing"
)

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	if config == nil {
		t.Fatal("GetDefaultConfig returned nil")
	}

	// 验证基本配置
	if config.MachineID != "classic_fruit" {
		t.Errorf("MachineID = %v, want classic_fruit", config.MachineID)
	}

	if config.Name != "经典水果机" {
		t.Errorf("Name = %v, want 经典水果机", config.Name)
	}

	if config.Rows != 3 {
		t.Errorf("Rows = %v, want 3", config.Rows)
	}

	if config.Reels != 5 {
		t.Errorf("Reels = %v, want 5", config.Reels)
	}

	if config.PayLines != 20 {
		t.Errorf("PayLines = %v, want 20", config.PayLines)
	}

	if config.TargetRTP != 0.96 {
		t.Errorf("TargetRTP = %v, want 0.96", config.TargetRTP)
	}

	if config.Volatility != VolatilityMedium {
		t.Errorf("Volatility = %v, want VolatilityMedium", config.Volatility)
	}

	// 验证卷轴条数量
	if len(config.ReelStrips) != config.Reels {
		t.Errorf("ReelStrips count = %v, want %v", len(config.ReelStrips), config.Reels)
	}

	// 验证每个卷轴条
	for i, strip := range config.ReelStrips {
		if strip.ReelID != i {
			t.Errorf("ReelStrip[%d].ReelID = %v, want %v", i, strip.ReelID, i)
		}
		if len(strip.Symbols) != len(strip.Weights) {
			t.Errorf("ReelStrip[%d] symbols/weights mismatch: %v vs %v",
				i, len(strip.Symbols), len(strip.Weights))
		}
		if len(strip.Symbols) == 0 {
			t.Errorf("ReelStrip[%d] has no symbols", i)
		}
	}

	// 验证赔率表
	if len(config.PayTables) == 0 {
		t.Error("PayTables is empty")
	}

	// 验证特殊符号
	if len(config.WildSymbols) == 0 {
		t.Error("WildSymbols is empty")
	}

	if len(config.ScatterSymbols) == 0 {
		t.Error("ScatterSymbols is empty")
	}

	if len(config.BonusSymbols) == 0 {
		t.Error("BonusSymbols is empty")
	}

	// 验证特殊功能
	if len(config.Features) == 0 {
		t.Error("Features is empty")
	}
}

func TestGetLuckySevenConfig(t *testing.T) {
	config := GetLuckySevenConfig()

	if config.MachineID != "lucky_seven" {
		t.Errorf("MachineID = %v, want lucky_seven", config.MachineID)
	}

	if config.Name != "幸运777" {
		t.Errorf("Name = %v, want 幸运777", config.Name)
	}

	if config.TargetRTP != 0.95 {
		t.Errorf("TargetRTP = %v, want 0.95", config.TargetRTP)
	}

	if config.Volatility != VolatilityHigh {
		t.Errorf("Volatility = %v, want VolatilityHigh", config.Volatility)
	}

	// 验证7符号的赔率是否翻倍
	foundSevenPayTable := false
	for _, payTable := range config.PayTables {
		if payTable.Symbol == SymbolSeven && payTable.Count == 5 {
			if payTable.Multiplier != 100 { // 原始50 * 2
				t.Errorf("Seven multiplier = %v, want 100", payTable.Multiplier)
			}
			foundSevenPayTable = true
			break
		}
	}
	if !foundSevenPayTable {
		t.Error("Seven pay table not found")
	}
}

func TestGetMegaFruitConfig(t *testing.T) {
	config := GetMegaFruitConfig()

	if config.MachineID != "mega_fruit" {
		t.Errorf("MachineID = %v, want mega_fruit", config.MachineID)
	}

	if config.Name != "超级水果" {
		t.Errorf("Name = %v, want 超级水果", config.Name)
	}

	if config.TargetRTP != 0.97 {
		t.Errorf("TargetRTP = %v, want 0.97", config.TargetRTP)
	}

	if config.Volatility != VolatilityLow {
		t.Errorf("Volatility = %v, want VolatilityLow", config.Volatility)
	}

	if config.PayLines != 30 {
		t.Errorf("PayLines = %v, want 30", config.PayLines)
	}

	// 验证赔率是否降低
	defaultConfig := GetDefaultConfig()
	for i, payTable := range config.PayTables {
		expectedMultiplier := defaultConfig.PayTables[i].Multiplier * 0.8
		if payTable.Multiplier != expectedMultiplier {
			t.Errorf("PayTable[%d] multiplier = %v, want %v",
				i, payTable.Multiplier, expectedMultiplier)
		}
	}
}

func TestGetDiamondDeluxeConfig(t *testing.T) {
	config := GetDiamondDeluxeConfig()

	if config.MachineID != "diamond_deluxe" {
		t.Errorf("MachineID = %v, want diamond_deluxe", config.MachineID)
	}

	if config.Name != "钻石豪华版" {
		t.Errorf("Name = %v, want 钻石豪华版", config.Name)
	}

	if config.Rows != 4 {
		t.Errorf("Rows = %v, want 4", config.Rows)
	}

	if config.Reels != 6 {
		t.Errorf("Reels = %v, want 6", config.Reels)
	}

	if config.PayLines != 50 {
		t.Errorf("PayLines = %v, want 50", config.PayLines)
	}

	if config.MinBet != 20 {
		t.Errorf("MinBet = %v, want 20", config.MinBet)
	}

	if config.MaxBet != 20000 {
		t.Errorf("MaxBet = %v, want 20000", config.MaxBet)
	}

	if config.Volatility != VolatilityHigh {
		t.Errorf("Volatility = %v, want VolatilityHigh", config.Volatility)
	}
}

func TestConfigPresets(t *testing.T) {
	expectedPresets := []string{
		"classic_fruit",
		"lucky_seven",
		"mega_fruit",
		"diamond_deluxe",
	}

	for _, preset := range expectedPresets {
		if _, exists := ConfigPresets[preset]; !exists {
			t.Errorf("ConfigPresets missing %v", preset)
		}
	}

	// 验证每个预设都有效
	for name, config := range ConfigPresets {
		if config == nil {
			t.Errorf("ConfigPresets[%v] is nil", name)
			continue
		}
		if config.MachineID == "" {
			t.Errorf("ConfigPresets[%v] has empty MachineID", name)
		}
		if config.Name == "" {
			t.Errorf("ConfigPresets[%v] has empty Name", name)
		}
	}
}

func TestGetConfigByID(t *testing.T) {
	tests := []struct {
		machineID string
		expected  string
	}{
		{
			machineID: "classic_fruit",
			expected:  "classic_fruit",
		},
		{
			machineID: "lucky_seven",
			expected:  "lucky_seven",
		},
		{
			machineID: "unknown_machine",
			expected:  "classic_fruit", // 应该返回默认配置
		},
		{
			machineID: "",
			expected:  "classic_fruit", // 空ID应该返回默认配置
		},
	}

	for _, tt := range tests {
		t.Run(tt.machineID, func(t *testing.T) {
			config := GetConfigByID(tt.machineID)
			if config.MachineID != tt.expected {
				t.Errorf("GetConfigByID(%v) returned MachineID %v, want %v",
					tt.machineID, config.MachineID, tt.expected)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *SlotConfig
		wantErr bool
		errType error
	}{
		{
			name:    "有效配置",
			config:  GetDefaultConfig(),
			wantErr: false,
		},
		{
			name: "行数过少",
			config: &SlotConfig{
				Rows:       2,
				Reels:      5,
				PayLines:   20,
				TargetRTP:  0.96,
				ReelStrips: GetDefaultConfig().ReelStrips,
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidConfig,
		},
		{
			name: "行数过多",
			config: &SlotConfig{
				Rows:       6,
				Reels:      5,
				PayLines:   20,
				TargetRTP:  0.96,
				ReelStrips: GetDefaultConfig().ReelStrips,
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidConfig,
		},
		{
			name: "卷轴数过少",
			config: &SlotConfig{
				Rows:       3,
				Reels:      2,
				PayLines:   20,
				TargetRTP:  0.96,
				ReelStrips: GetDefaultConfig().ReelStrips[:2],
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidConfig,
		},
		{
			name: "卷轴数过多",
			config: &SlotConfig{
				Rows:       3,
				Reels:      8,
				PayLines:   20,
				TargetRTP:  0.96,
				ReelStrips: GetDefaultConfig().ReelStrips,
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidConfig,
		},
		{
			name: "支付线过少",
			config: &SlotConfig{
				Rows:       3,
				Reels:      5,
				PayLines:   0,
				TargetRTP:  0.96,
				ReelStrips: GetDefaultConfig().ReelStrips,
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidConfig,
		},
		{
			name: "支付线过多",
			config: &SlotConfig{
				Rows:       3,
				Reels:      5,
				PayLines:   101,
				TargetRTP:  0.96,
				ReelStrips: GetDefaultConfig().ReelStrips,
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidConfig,
		},
		{
			name: "RTP过低",
			config: &SlotConfig{
				Rows:       3,
				Reels:      5,
				PayLines:   20,
				TargetRTP:  0.79,
				ReelStrips: GetDefaultConfig().ReelStrips,
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidRTP,
		},
		{
			name: "RTP过高",
			config: &SlotConfig{
				Rows:       3,
				Reels:      5,
				PayLines:   20,
				TargetRTP:  1.0,
				ReelStrips: GetDefaultConfig().ReelStrips,
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidRTP,
		},
		{
			name: "卷轴条数量不匹配",
			config: &SlotConfig{
				Rows:       3,
				Reels:      5,
				PayLines:   20,
				TargetRTP:  0.96,
				ReelStrips: GetDefaultConfig().ReelStrips[:3], // 只有3个，需要5个
				PayTables:  GetDefaultConfig().PayTables,
			},
			wantErr: true,
			errType: ErrInvalidReelStrips,
		},
		{
			name: "空赔率表",
			config: &SlotConfig{
				Rows:       3,
				Reels:      5,
				PayLines:   20,
				TargetRTP:  0.96,
				ReelStrips: GetDefaultConfig().ReelStrips,
				PayTables:  []PayTable{},
			},
			wantErr: true,
			errType: ErrInvalidPayTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("ValidateConfig() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

func TestPayTableCompleteness(t *testing.T) {
	config := GetDefaultConfig()

	// 收集所有使用的符号
	usedSymbols := make(map[Symbol]bool)
	for _, strip := range config.ReelStrips {
		for _, symbol := range strip.Symbols {
			usedSymbols[symbol] = true
		}
	}

	// 收集赔率表中的符号
	payTableSymbols := make(map[Symbol]bool)
	for _, payTable := range config.PayTables {
		payTableSymbols[payTable.Symbol] = true
	}

	// 验证主要符号都有赔率表
	importantSymbols := []Symbol{
		SymbolCherry, SymbolLemon, SymbolOrange, SymbolPlum,
		SymbolGrape, SymbolWatermelon, SymbolBar, SymbolSeven,
		SymbolWild, SymbolScatter,
	}

	for _, symbol := range importantSymbols {
		if !payTableSymbols[symbol] {
			t.Errorf("Symbol %v missing from pay table", symbol)
		}
	}
}

func TestFeatureConfiguration(t *testing.T) {
	config := GetDefaultConfig()

	// 验证特殊功能配置
	featureTypes := make(map[FeatureType]bool)
	for _, feature := range config.Features {
		featureTypes[feature.Type] = true

		// 验证触发符号
		if len(feature.TriggerSymbols) == 0 {
			t.Errorf("Feature %v has no trigger symbols", feature.Type)
		}

		// 验证触发数量
		if feature.TriggerCount <= 0 {
			t.Errorf("Feature %v has invalid trigger count: %v",
				feature.Type, feature.TriggerCount)
		}

		// 验证概率
		if feature.Probability < 0 || feature.Probability > 1 {
			t.Errorf("Feature %v has invalid probability: %v",
				feature.Type, feature.Probability)
		}
	}

	// 验证关键特性存在
	expectedFeatures := []FeatureType{
		FeatureTypeFreeSpins,
		FeatureTypeBonus,
		FeatureTypeMultiplier,
	}

	for _, expected := range expectedFeatures {
		if !featureTypes[expected] {
			t.Errorf("Missing expected feature: %v", expected)
		}
	}
}