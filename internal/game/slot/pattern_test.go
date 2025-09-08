package slot

import (
	"reflect"
	"testing"
)

func TestNewAdvancedPatternMatcher(t *testing.T) {
	config := GetDefaultConfig()
	matcher := NewAdvancedPatternMatcher(config)

	if matcher == nil {
		t.Fatal("NewAdvancedPatternMatcher returned nil")
	}

	if matcher.config != config {
		t.Error("Config not properly set")
	}

	// 验证支付线模式已初始化
	if len(matcher.paylinePatterns) == 0 {
		t.Error("No payline patterns initialized")
	}

	// 支付线数量不应超过配置
	if len(matcher.paylinePatterns) > config.PayLines {
		t.Errorf("Payline patterns %v exceeds configured %v",
			len(matcher.paylinePatterns), config.PayLines)
	}
}

func TestAdvancedPatternMatcher_FindWinningLines(t *testing.T) {
	config := GetDefaultConfig()
	matcher := NewAdvancedPatternMatcher(config)

	tests := []struct {
		name     string
		reels    [][]Symbol
		minWins  int
		maxWins  int
	}{
		{
			name: "三个相同符号水平线",
			reels: [][]Symbol{
				{SymbolCherry, SymbolLemon, SymbolOrange},
				{SymbolCherry, SymbolPlum, SymbolGrape},
				{SymbolCherry, SymbolWatermelon, SymbolBar},
				{SymbolLemon, SymbolSeven, SymbolWild},
				{SymbolOrange, SymbolScatter, SymbolBonus},
			},
			minWins: 1,
			maxWins: 5,
		},
		{
			name: "包含Wild的中奖线",
			reels: [][]Symbol{
				{SymbolCherry, SymbolLemon, SymbolOrange},
				{SymbolWild, SymbolPlum, SymbolGrape},
				{SymbolCherry, SymbolWatermelon, SymbolBar},
				{SymbolLemon, SymbolSeven, SymbolWild},
				{SymbolOrange, SymbolScatter, SymbolBonus},
			},
			minWins: 1,
			maxWins: 5,
		},
		{
			name: "三个Scatter触发",
			reels: [][]Symbol{
				{SymbolScatter, SymbolLemon, SymbolOrange},
				{SymbolCherry, SymbolScatter, SymbolGrape},
				{SymbolCherry, SymbolWatermelon, SymbolScatter},
				{SymbolLemon, SymbolSeven, SymbolWild},
				{SymbolOrange, SymbolPlum, SymbolBonus},
			},
			minWins: 1,
			maxWins: 10,
		},
		{
			name: "无中奖组合",
			reels: [][]Symbol{
				{SymbolCherry, SymbolLemon, SymbolOrange},
				{SymbolPlum, SymbolGrape, SymbolWatermelon},
				{SymbolBar, SymbolSeven, SymbolCherry},
				{SymbolLemon, SymbolOrange, SymbolPlum},
				{SymbolGrape, SymbolWatermelon, SymbolBar},
			},
			minWins: 0,
			maxWins: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winLines := matcher.FindWinningLines(tt.reels, config)
			winCount := len(winLines)
			
			if winCount < tt.minWins || winCount > tt.maxWins {
				t.Errorf("Found %v win lines, expected [%v, %v]",
					winCount, tt.minWins, tt.maxWins)
			}

			// 验证每条中奖线的基本属性
			for _, line := range winLines {
				if line.Count < 3 && line.Symbol != SymbolScatter {
					t.Errorf("Win line with less than 3 symbols: %+v", line)
				}
				if len(line.Positions) != line.Count {
					t.Errorf("Position count mismatch: %v vs %v",
						len(line.Positions), line.Count)
				}
			}
		})
	}
}

func TestAdvancedPatternMatcher_CalculatePayout(t *testing.T) {
	config := GetDefaultConfig()
	matcher := NewAdvancedPatternMatcher(config)

	tests := []struct {
		name      string
		winLines  []WinLine
		betAmount int64
		minPayout int64
		maxPayout int64
	}{
		{
			name: "单条中奖线",
			winLines: []WinLine{
				{
					Symbol:     SymbolCherry,
					Count:      3,
					Multiplier: 1.0,
				},
			},
			betAmount: 100,
			minPayout: 100,  // 至少1倍
			maxPayout: 1000, // 最多10倍
		},
		{
			name: "多条中奖线",
			winLines: []WinLine{
				{
					Symbol:     SymbolCherry,
					Count:      3,
					Multiplier: 1.0,
				},
				{
					Symbol:     SymbolLemon,
					Count:      4,
					Multiplier: 1.0,
				},
			},
			betAmount: 100,
			minPayout: 200,
			maxPayout: 2000,
		},
		{
			name: "带Wild倍率的中奖线",
			winLines: []WinLine{
				{
					Symbol:     SymbolSeven,
					Count:      3,
					Multiplier: 2.0, // Wild加倍
				},
			},
			betAmount: 100,
			minPayout: 2000,
			maxPayout: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payout := matcher.CalculatePayout(tt.winLines, tt.betAmount)
			
			if payout < tt.minPayout || payout > tt.maxPayout {
				t.Errorf("Payout %v outside expected range [%v, %v]",
					payout, tt.minPayout, tt.maxPayout)
			}
		})
	}
}

func TestAdvancedPatternMatcher_DetectFeatures(t *testing.T) {
	config := GetDefaultConfig()
	matcher := NewAdvancedPatternMatcher(config)

	tests := []struct {
		name            string
		reels           [][]Symbol
		expectedFeature FeatureType
		shouldDetect    bool
	}{
		{
			name: "检测免费旋转",
			reels: [][]Symbol{
				{SymbolScatter, SymbolLemon, SymbolOrange},
				{SymbolCherry, SymbolScatter, SymbolGrape},
				{SymbolCherry, SymbolWatermelon, SymbolScatter},
				{SymbolLemon, SymbolSeven, SymbolWild},
				{SymbolOrange, SymbolPlum, SymbolBonus},
			},
			expectedFeature: FeatureTypeFreeSpins,
			shouldDetect:    true,
		},
		{
			name: "检测奖励游戏",
			reels: [][]Symbol{
				{SymbolBonus, SymbolLemon, SymbolOrange},
				{SymbolCherry, SymbolBonus, SymbolGrape},
				{SymbolCherry, SymbolWatermelon, SymbolBonus},
				{SymbolLemon, SymbolSeven, SymbolWild},
				{SymbolOrange, SymbolPlum, SymbolScatter},
			},
			expectedFeature: FeatureTypeBonus,
			shouldDetect:    true,
		},
		{
			name: "检测扩展Wild",
			reels: [][]Symbol{
				{SymbolCherry, SymbolLemon, SymbolOrange},
				{SymbolPlum, SymbolGrape, SymbolWatermelon},
				{SymbolBar, SymbolWild, SymbolCherry}, // Wild在中间卷轴
				{SymbolLemon, SymbolOrange, SymbolPlum},
				{SymbolGrape, SymbolWatermelon, SymbolBar},
			},
			expectedFeature: FeatureTypeExpandWild,
			shouldDetect:    true,
		},
		{
			name: "无特殊功能",
			reels: [][]Symbol{
				{SymbolCherry, SymbolLemon, SymbolOrange},
				{SymbolPlum, SymbolGrape, SymbolWatermelon},
				{SymbolBar, SymbolSeven, SymbolCherry},
				{SymbolLemon, SymbolOrange, SymbolPlum},
				{SymbolGrape, SymbolWatermelon, SymbolBar},
			},
			expectedFeature: FeatureTypeFreeSpins,
			shouldDetect:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := matcher.DetectFeatures(tt.reels, config)
			
			found := false
			for _, feature := range features {
				if feature.Type == tt.expectedFeature {
					found = true
					break
				}
			}
			
			if found != tt.shouldDetect {
				t.Errorf("Feature %v detection = %v, expected %v",
					tt.expectedFeature, found, tt.shouldDetect)
			}
		})
	}
}

func TestAdvancedPatternMatcher_checkConsecutiveSymbols(t *testing.T) {
	config := GetDefaultConfig()
	matcher := NewAdvancedPatternMatcher(config)

	pattern := []Position{
		{Reel: 0, Row: 0},
		{Reel: 1, Row: 0},
		{Reel: 2, Row: 0},
		{Reel: 3, Row: 0},
		{Reel: 4, Row: 0},
	}

	tests := []struct {
		name      string
		symbols   []Symbol
		expectWin bool
		minCount  int
	}{
		{
			name:      "五个相同符号",
			symbols:   []Symbol{SymbolCherry, SymbolCherry, SymbolCherry, SymbolCherry, SymbolCherry},
			expectWin: true,
			minCount:  5,
		},
		{
			name:      "三个相同符号",
			symbols:   []Symbol{SymbolCherry, SymbolCherry, SymbolCherry, SymbolLemon, SymbolOrange},
			expectWin: true,
			minCount:  3,
		},
		{
			name:      "Wild替代",
			symbols:   []Symbol{SymbolCherry, SymbolWild, SymbolCherry, SymbolLemon, SymbolOrange},
			expectWin: true,
			minCount:  3,
		},
		{
			name:      "两个相同符号（不中奖）",
			symbols:   []Symbol{SymbolCherry, SymbolCherry, SymbolLemon, SymbolOrange, SymbolPlum},
			expectWin: false,
			minCount:  0,
		},
		{
			name:      "全是Wild",
			symbols:   []Symbol{SymbolWild, SymbolWild, SymbolWild, SymbolWild, SymbolWild},
			expectWin: true,
			minCount:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winLine := matcher.checkConsecutiveSymbols(tt.symbols, 0, pattern)
			
			if tt.expectWin && winLine == nil {
				t.Error("Expected win but got nil")
			} else if !tt.expectWin && winLine != nil {
				t.Errorf("Expected no win but got: %+v", winLine)
			}
			
			if winLine != nil && winLine.Count < tt.minCount {
				t.Errorf("Win count %v less than expected %v", winLine.Count, tt.minCount)
			}
		})
	}
}

func TestAdvancedPatternMatcher_LinePatterns(t *testing.T) {
	config := &SlotConfig{
		Rows:      3,
		Reels:     5,
		PayLines:  20,
		TargetRTP: 0.96,
	}
	matcher := NewAdvancedPatternMatcher(config)

	// 测试水平线
	horizontalLine := matcher.createHorizontalLine(1, 5)
	expectedHorizontal := []Position{
		{Reel: 0, Row: 1},
		{Reel: 1, Row: 1},
		{Reel: 2, Row: 1},
		{Reel: 3, Row: 1},
		{Reel: 4, Row: 1},
	}
	if !reflect.DeepEqual(horizontalLine, expectedHorizontal) {
		t.Errorf("Horizontal line = %v, want %v", horizontalLine, expectedHorizontal)
	}

	// 测试对角线
	diagonalLine := matcher.createDiagonalLine(3, 3, true)
	if len(diagonalLine) != 3 {
		t.Errorf("Diagonal line length = %v, want 3", len(diagonalLine))
	}

	// 测试V型线
	vLine := matcher.createVLine(5, 3, true)
	if len(vLine) != 5 {
		t.Errorf("V line length = %v, want 5", len(vLine))
	}

	// 测试之字形
	zigzagLine := matcher.createZigzagLine(5, 3, true)
	if len(zigzagLine) != 5 {
		t.Errorf("Zigzag line length = %v, want 5", len(zigzagLine))
	}
	
	// 验证之字形模式
	for i, pos := range zigzagLine {
		if i%2 == 0 && pos.Row != 0 {
			t.Errorf("Zigzag pattern error at position %v", i)
		}
		if i%2 == 1 && pos.Row != 2 {
			t.Errorf("Zigzag pattern error at position %v", i)
		}
	}
}

func TestAdvancedPatternMatcher_Helpers(t *testing.T) {
	config := GetDefaultConfig()
	matcher := NewAdvancedPatternMatcher(config)

	// 测试isWild
	if !matcher.isWild(SymbolWild) {
		t.Error("isWild(SymbolWild) should return true")
	}
	if matcher.isWild(SymbolCherry) {
		t.Error("isWild(SymbolCherry) should return false")
	}

	// 测试isScatter
	if !matcher.isScatter(SymbolScatter) {
		t.Error("isScatter(SymbolScatter) should return true")
	}
	if matcher.isScatter(SymbolCherry) {
		t.Error("isScatter(SymbolCherry) should return false")
	}

	// 测试isBonus
	if !matcher.isBonus(SymbolBonus) {
		t.Error("isBonus(SymbolBonus) should return true")
	}
	if matcher.isBonus(SymbolCherry) {
		t.Error("isBonus(SymbolCherry) should return false")
	}

	// 测试getMultiplier
	multiplier := matcher.getMultiplier(SymbolCherry, 3)
	if multiplier != 1.0 {
		t.Errorf("getMultiplier(Cherry, 3) = %v, want 1.0", multiplier)
	}
	
	multiplier = matcher.getMultiplier(SymbolSeven, 5)
	if multiplier != 50.0 {
		t.Errorf("getMultiplier(Seven, 5) = %v, want 50.0", multiplier)
	}
}

func TestAdvancedPatternMatcher_deduplicateWinLines(t *testing.T) {
	config := GetDefaultConfig()
	matcher := NewAdvancedPatternMatcher(config)

	lines := []WinLine{
		{Symbol: SymbolCherry, Count: 3},
		{Symbol: SymbolCherry, Count: 3}, // 重复
		{Symbol: SymbolLemon, Count: 4},
		{Symbol: SymbolCherry, Count: 4},  // 不同数量，不算重复
	}

	deduplicated := matcher.deduplicateWinLines(lines)
	
	if len(deduplicated) != 3 {
		t.Errorf("Deduplicated lines = %v, want 3", len(deduplicated))
	}
	
	// 验证每个独特的组合都存在
	combinations := make(map[string]bool)
	for _, line := range deduplicated {
		key := string(line.Symbol) + string(rune(line.Count))
		if combinations[key] {
			t.Errorf("Found duplicate after deduplication: %+v", line)
		}
		combinations[key] = true
	}
}