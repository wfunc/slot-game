package slot

// GetDefaultConfig 获取默认配置（经典水果机）
func GetDefaultConfig() *SlotConfig {
	return &SlotConfig{
		MachineID:  "classic_fruit",
		Name:       "经典水果机",
		Rows:       3,
		Reels:      5,
		PayLines:   20,
		MinBet:     10,
		MaxBet:     10000,
		DefaultBet: 100,
		TargetRTP:  0.96, // 96% RTP
		Volatility: VolatilityMedium,
		
		// 卷轴条配置
		ReelStrips: []ReelStrip{
			// 第1卷轴
			{
				ReelID: 0,
				Symbols: []Symbol{
					SymbolCherry, SymbolLemon, SymbolOrange, SymbolPlum,
					SymbolGrape, SymbolWatermelon, SymbolBar, SymbolSeven,
					SymbolWild, SymbolScatter, SymbolCherry, SymbolLemon,
					SymbolOrange, SymbolPlum, SymbolGrape, SymbolWatermelon,
				},
				Weights: []int{
					10, 10, 10, 10, // 基础水果
					8, 8, 5, 3,     // 高级符号
					2, 2, 10, 10,   // 特殊+基础
					10, 10, 8, 8,   // 基础水果
				},
			},
			// 第2卷轴
			{
				ReelID: 1,
				Symbols: []Symbol{
					SymbolLemon, SymbolOrange, SymbolPlum, SymbolGrape,
					SymbolWatermelon, SymbolBar, SymbolSeven, SymbolWild,
					SymbolScatter, SymbolCherry, SymbolLemon, SymbolOrange,
					SymbolPlum, SymbolGrape, SymbolWatermelon, SymbolBar,
				},
				Weights: []int{
					10, 10, 10, 8,
					8, 5, 3, 2,
					2, 10, 10, 10,
					10, 8, 8, 5,
				},
			},
			// 第3卷轴
			{
				ReelID: 2,
				Symbols: []Symbol{
					SymbolOrange, SymbolPlum, SymbolGrape, SymbolWatermelon,
					SymbolBar, SymbolSeven, SymbolWild, SymbolScatter,
					SymbolBonus, SymbolCherry, SymbolLemon, SymbolOrange,
					SymbolPlum, SymbolGrape, SymbolWatermelon, SymbolBar,
				},
				Weights: []int{
					10, 10, 8, 8,
					5, 3, 2, 2,
					1, 10, 10, 10,
					10, 8, 8, 5,
				},
			},
			// 第4卷轴
			{
				ReelID: 3,
				Symbols: []Symbol{
					SymbolPlum, SymbolGrape, SymbolWatermelon, SymbolBar,
					SymbolSeven, SymbolWild, SymbolScatter, SymbolCherry,
					SymbolLemon, SymbolOrange, SymbolPlum, SymbolGrape,
					SymbolWatermelon, SymbolBar, SymbolSeven, SymbolWild,
				},
				Weights: []int{
					10, 8, 8, 5,
					3, 2, 2, 10,
					10, 10, 10, 8,
					8, 5, 3, 2,
				},
			},
			// 第5卷轴
			{
				ReelID: 4,
				Symbols: []Symbol{
					SymbolGrape, SymbolWatermelon, SymbolBar, SymbolSeven,
					SymbolWild, SymbolScatter, SymbolBonus, SymbolCherry,
					SymbolLemon, SymbolOrange, SymbolPlum, SymbolGrape,
					SymbolWatermelon, SymbolBar, SymbolSeven, SymbolCherry,
				},
				Weights: []int{
					8, 8, 5, 3,
					2, 2, 1, 10,
					10, 10, 10, 8,
					8, 5, 3, 10,
				},
			},
		},
		
		// 赔率表（降低倍率以控制RTP）
		PayTables: []PayTable{
			// 樱桃
			{Symbol: SymbolCherry, Count: 3, Multiplier: 1},
			{Symbol: SymbolCherry, Count: 4, Multiplier: 2},
			{Symbol: SymbolCherry, Count: 5, Multiplier: 5},
			
			// 柠檬
			{Symbol: SymbolLemon, Count: 3, Multiplier: 1.5},
			{Symbol: SymbolLemon, Count: 4, Multiplier: 3},
			{Symbol: SymbolLemon, Count: 5, Multiplier: 7},
			
			// 橙子
			{Symbol: SymbolOrange, Count: 3, Multiplier: 2},
			{Symbol: SymbolOrange, Count: 4, Multiplier: 5},
			{Symbol: SymbolOrange, Count: 5, Multiplier: 10},
			
			// 李子
			{Symbol: SymbolPlum, Count: 3, Multiplier: 2.5},
			{Symbol: SymbolPlum, Count: 4, Multiplier: 6},
			{Symbol: SymbolPlum, Count: 5, Multiplier: 12},
			
			// 葡萄
			{Symbol: SymbolGrape, Count: 3, Multiplier: 3},
			{Symbol: SymbolGrape, Count: 4, Multiplier: 7},
			{Symbol: SymbolGrape, Count: 5, Multiplier: 15},
			
			// 西瓜
			{Symbol: SymbolWatermelon, Count: 3, Multiplier: 4},
			{Symbol: SymbolWatermelon, Count: 4, Multiplier: 10},
			{Symbol: SymbolWatermelon, Count: 5, Multiplier: 20},
			
			// BAR
			{Symbol: SymbolBar, Count: 3, Multiplier: 5},
			{Symbol: SymbolBar, Count: 4, Multiplier: 15},
			{Symbol: SymbolBar, Count: 5, Multiplier: 30},
			
			// 7
			{Symbol: SymbolSeven, Count: 3, Multiplier: 10},
			{Symbol: SymbolSeven, Count: 4, Multiplier: 25},
			{Symbol: SymbolSeven, Count: 5, Multiplier: 50},
			
			// Wild（百搭）
			{Symbol: SymbolWild, Count: 3, Multiplier: 7},
			{Symbol: SymbolWild, Count: 4, Multiplier: 20},
			{Symbol: SymbolWild, Count: 5, Multiplier: 40},
			
			// Scatter（分散）
			{Symbol: SymbolScatter, Count: 3, Multiplier: 2},
			{Symbol: SymbolScatter, Count: 4, Multiplier: 10},
			{Symbol: SymbolScatter, Count: 5, Multiplier: 25},
			
			// Bonus（奖励）
			{Symbol: SymbolBonus, Count: 3, Multiplier: 0}, // 触发奖励游戏
		},
		
		// 特殊符号
		WildSymbols:    []Symbol{SymbolWild},
		ScatterSymbols: []Symbol{SymbolScatter},
		BonusSymbols:   []Symbol{SymbolBonus},
		
		// 特殊功能配置
		Features: []FeatureConfig{
			// 免费旋转
			{
				Type:           FeatureTypeFreeSpins,
				TriggerSymbols: []Symbol{SymbolScatter},
				TriggerCount:   3,
				Probability:    0.01, // 1%概率
				Value:          5,    // 5次免费旋转（降低）
			},
			// 奖励游戏
			{
				Type:           FeatureTypeBonus,
				TriggerSymbols: []Symbol{SymbolBonus},
				TriggerCount:   3,
				Probability:    0.005, // 0.5%概率
				Value:          "bonus_wheel",
			},
			// 倍率增加
			{
				Type:           FeatureTypeMultiplier,
				TriggerSymbols: []Symbol{SymbolWild},
				TriggerCount:   2,
				Probability:    0.05, // 5%概率
				Value:          1.5,  // 1.5倍（降低倍率）
			},
			// 扩展Wild
			{
				Type:           FeatureTypeExpandWild,
				TriggerSymbols: []Symbol{SymbolWild},
				TriggerCount:   1,
				Probability:    0.02, // 2%概率
				Value:          "expand_reel",
			},
			// 重新旋转
			{
				Type:           FeatureTypeRespin,
				TriggerSymbols: []Symbol{SymbolSeven},
				TriggerCount:   2,
				Probability:    0.03, // 3%概率
				Value:          1,    // 1次重新旋转
			},
		},
	}
}

// GetLuckySevenConfig 获取幸运7配置（高波动性）
func GetLuckySevenConfig() *SlotConfig {
	config := GetDefaultConfig()
	config.MachineID = "lucky_seven"
	config.Name = "幸运777"
	config.TargetRTP = 0.95
	config.Volatility = VolatilityHigh
	
	// 调整赔率表，增加7的赔率
	for i, payTable := range config.PayTables {
		if payTable.Symbol == SymbolSeven {
			config.PayTables[i].Multiplier *= 2 // 双倍赔率
		}
	}
	
	return config
}

// GetMegaFruitConfig 获取超级水果机配置（低波动性）
func GetMegaFruitConfig() *SlotConfig {
	config := GetDefaultConfig()
	config.MachineID = "mega_fruit"
	config.Name = "超级水果"
	config.TargetRTP = 0.97
	config.Volatility = VolatilityLow
	config.PayLines = 30 // 更多支付线
	
	// 降低赔率但提高中奖频率
	for i := range config.PayTables {
		config.PayTables[i].Multiplier *= 0.8
	}
	
	return config
}

// GetDiamondDeluxeConfig 获取钻石豪华版配置
func GetDiamondDeluxeConfig() *SlotConfig {
	return &SlotConfig{
		MachineID:  "diamond_deluxe",
		Name:       "钻石豪华版",
		Rows:       4,
		Reels:      6,
		PayLines:   50,
		MinBet:     20,
		MaxBet:     20000,
		DefaultBet: 200,
		TargetRTP:  0.96,
		Volatility: VolatilityHigh,
		
		// 使用钻石主题的符号...
		// 这里简化处理，实际应该有完整的钻石主题配置
		ReelStrips:     GetDefaultConfig().ReelStrips,
		PayTables:      GetDefaultConfig().PayTables,
		WildSymbols:    []Symbol{SymbolWild},
		ScatterSymbols: []Symbol{SymbolScatter},
		BonusSymbols:   []Symbol{SymbolBonus},
		Features:       GetDefaultConfig().Features,
	}
}

// ConfigPresets 预设配置集合
var ConfigPresets = map[string]*SlotConfig{
	"classic_fruit":  GetDefaultConfig(),
	"lucky_seven":    GetLuckySevenConfig(),
	"mega_fruit":     GetMegaFruitConfig(),
	"diamond_deluxe": GetDiamondDeluxeConfig(),
}

// GetConfigByID 根据ID获取配置
func GetConfigByID(machineID string) *SlotConfig {
	if config, exists := ConfigPresets[machineID]; exists {
		return config
	}
	return GetDefaultConfig()
}

// ValidateConfig 验证配置
func ValidateConfig(config *SlotConfig) error {
	if config.Rows < 3 || config.Rows > 5 {
		return ErrInvalidConfig
	}
	if config.Reels < 3 || config.Reels > 7 {
		return ErrInvalidConfig
	}
	if config.PayLines < 1 || config.PayLines > 100 {
		return ErrInvalidConfig
	}
	if config.TargetRTP < 0.8 || config.TargetRTP > 0.99 {
		return ErrInvalidRTP
	}
	if len(config.ReelStrips) != config.Reels {
		return ErrInvalidReelStrips
	}
	if len(config.PayTables) == 0 {
		return ErrInvalidPayTable
	}
	return nil
}