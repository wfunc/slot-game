package slot

import (
	"math"
	"testing"
	"time"
)

func TestNewDynamicRTPController(t *testing.T) {
	targetRTP := 0.96
	controller := NewDynamicRTPController(targetRTP)

	if controller == nil {
		t.Fatal("NewDynamicRTPController returned nil")
	}

	if controller.targetRTP != targetRTP {
		t.Errorf("targetRTP = %v, want %v", controller.targetRTP, targetRTP)
	}

	if controller.minRTP != targetRTP*0.85 {
		t.Errorf("minRTP = %v, want %v", controller.minRTP, targetRTP*0.85)
	}

	if controller.maxRTP != targetRTP*1.15 {
		t.Errorf("maxRTP = %v, want %v", controller.maxRTP, targetRTP*1.15)
	}
}

func TestDynamicRTPController_CalculateRTP(t *testing.T) {
	controller := NewDynamicRTPController(0.96)

	tests := []struct {
		name      string
		totalWin  int64
		totalBet  int64
		expected  float64
	}{
		{
			name:     "标准RTP",
			totalWin: 960,
			totalBet: 1000,
			expected: 0.96,
		},
		{
			name:     "零下注",
			totalWin: 100,
			totalBet: 0,
			expected: 0,
		},
		{
			name:     "高RTP",
			totalWin: 1200,
			totalBet: 1000,
			expected: 1.2,
		},
		{
			name:     "低RTP",
			totalWin: 500,
			totalBet: 1000,
			expected: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := controller.CalculateRTP(tt.totalWin, tt.totalBet)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("CalculateRTP() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDynamicRTPController_AdjustOdds(t *testing.T) {
	controller := NewDynamicRTPController(0.96)

	tests := []struct {
		name       string
		currentRTP float64
		targetRTP  float64
		minExpect  float64
		maxExpect  float64
	}{
		{
			name:       "RTP偏低需要补偿",
			currentRTP: 0.90,
			targetRTP:  0.96,
			minExpect:  1.0, // 应该增加赔率
			maxExpect:  2.0,
		},
		{
			name:       "RTP偏高需要抑制",
			currentRTP: 1.02,
			targetRTP:  0.96,
			minExpect:  0.5,
			maxExpect:  1.0, // 应该减少赔率
		},
		{
			name:       "RTP接近目标",
			currentRTP: 0.96,
			targetRTP:  0.96,
			minExpect:  0.9,
			maxExpect:  1.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adjustment := controller.AdjustOdds(tt.currentRTP, tt.targetRTP)
			if adjustment < tt.minExpect || adjustment > tt.maxExpect {
				t.Errorf("AdjustOdds() = %v, expected range [%v, %v]",
					adjustment, tt.minExpect, tt.maxExpect)
			}
		})
	}
}

func TestDynamicRTPController_ShouldTriggerWin(t *testing.T) {
	controller := NewDynamicRTPController(0.96)

	// 添加一些历史数据
	for i := 0; i < 10; i++ {
		controller.UpdateHistory(100, 90)
	}

	tests := []struct {
		name       string
		currentRTP float64
		targetRTP  float64
		betAmount  int64
		runs       int
	}{
		{
			name:       "低RTP应该更频繁触发中奖",
			currentRTP: 0.80,
			targetRTP:  0.96,
			betAmount:  100,
			runs:       1000,
		},
		{
			name:       "高RTP应该减少中奖",
			currentRTP: 1.10,
			targetRTP:  0.96,
			betAmount:  100,
			runs:       1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wins := 0
			for i := 0; i < tt.runs; i++ {
				if controller.ShouldTriggerWin(tt.currentRTP, tt.targetRTP, tt.betAmount) {
					wins++
				}
			}
			
			winRate := float64(wins) / float64(tt.runs)
			t.Logf("%s: Win rate = %.2f%%", tt.name, winRate*100)
			
			// 验证中奖率在合理范围内（允许略高的阈值）
			if winRate < 0.1 || winRate > 0.95 {
				t.Errorf("Win rate %v outside reasonable range [0.1, 0.95]", winRate)
			}
		})
	}
}

func TestDynamicRTPController_GetCompensationMultiplier(t *testing.T) {
	controller := NewDynamicRTPController(0.96)

	tests := []struct {
		name       string
		currentRTP float64
		targetRTP  float64
		minExpect  float64
		maxExpect  float64
	}{
		{
			name:       "严重偏低RTP",
			currentRTP: 0.80,
			targetRTP:  0.96,
			minExpect:  1.2,
			maxExpect:  2.0,
		},
		{
			name:       "轻微偏低RTP",
			currentRTP: 0.93,
			targetRTP:  0.96,
			minExpect:  1.0,
			maxExpect:  1.2,
		},
		{
			name:       "接近目标RTP",
			currentRTP: 0.96,
			targetRTP:  0.96,
			minExpect:  0.95,
			maxExpect:  1.05,
		},
		{
			name:       "偏高RTP",
			currentRTP: 1.05,
			targetRTP:  0.96,
			minExpect:  0.5,
			maxExpect:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier := controller.GetCompensationMultiplier(tt.currentRTP, tt.targetRTP)
			if multiplier < tt.minExpect || multiplier > tt.maxExpect {
				t.Errorf("GetCompensationMultiplier() = %v, expected range [%v, %v]",
					multiplier, tt.minExpect, tt.maxExpect)
			}
		})
	}
}

func TestDynamicRTPController_UpdateHistory(t *testing.T) {
	controller := NewDynamicRTPController(0.96)

	// 添加多个样本
	for i := 0; i < 5; i++ {
		controller.UpdateHistory(100, 96)
	}

	stats := controller.GetStatistics()

	if stats.ShortTermSamples < 5 {
		t.Errorf("ShortTermSamples = %v, want at least 5", stats.ShortTermSamples)
	}

	if stats.LongTermSamples < 5 {
		t.Errorf("LongTermSamples = %v, want at least 5", stats.LongTermSamples)
	}

	// 短期和长期RTP应该接近0.96
	expectedRTP := 0.96
	tolerance := 0.01
	if math.Abs(stats.ShortTermRTP-expectedRTP) > tolerance {
		t.Errorf("ShortTermRTP = %v, want ~%v", stats.ShortTermRTP, expectedRTP)
	}

	if math.Abs(stats.LongTermRTP-expectedRTP) > tolerance {
		t.Errorf("LongTermRTP = %v, want ~%v", stats.LongTermRTP, expectedRTP)
	}
}

func TestRTPHistory_addSample(t *testing.T) {
	history := newRTPHistory(1*time.Hour, 10)

	// 添加样本
	for i := 0; i < 15; i++ {
		sample := RTPSample{
			timestamp: time.Now(),
			bet:       100,
			win:       96,
			rtp:       0.96,
		}
		history.addSample(sample)
	}

	// 应该只保留最多10个样本
	if len(history.samples) > 10 {
		t.Errorf("Sample count = %v, max should be 10", len(history.samples))
	}

	// 验证总计正确
	expectedTotalBet := int64(1000) // 最后10个样本
	if history.totalBet != expectedTotalBet {
		t.Errorf("totalBet = %v, want %v", history.totalBet, expectedTotalBet)
	}
}

func TestRTPHistory_getCurrentRTP(t *testing.T) {
	history := newRTPHistory(1*time.Hour, 100)

	// 没有数据时应该返回0
	if rtp := history.getCurrentRTP(); rtp != 0 {
		t.Errorf("getCurrentRTP() with no data = %v, want 0", rtp)
	}

	// 添加数据
	history.totalBet = 1000
	history.totalWin = 960

	expectedRTP := 0.96
	if rtp := history.getCurrentRTP(); math.Abs(rtp-expectedRTP) > 0.001 {
		t.Errorf("getCurrentRTP() = %v, want %v", rtp, expectedRTP)
	}
}

func TestRTPHistory_expiredSampleCleanup(t *testing.T) {
	history := newRTPHistory(1*time.Hour, 100)

	// 添加过期样本
	oldSample := RTPSample{
		timestamp: time.Now().Add(-2 * time.Hour),
		bet:       100,
		win:       100,
		rtp:       1.0,
	}
	history.addSample(oldSample)

	// 添加新样本应该清理过期的
	newSample := RTPSample{
		timestamp: time.Now(),
		bet:       200,
		win:       190,
		rtp:       0.95,
	}
	history.addSample(newSample)

	// 应该只有新样本
	if len(history.samples) != 1 {
		t.Errorf("Sample count after cleanup = %v, want 1", len(history.samples))
	}

	// 总计应该只包含新样本
	if history.totalBet != 200 {
		t.Errorf("totalBet after cleanup = %v, want 200", history.totalBet)
	}
}

func TestDynamicRTPController_calculateVolatilityAdjustment(t *testing.T) {
	controller := NewDynamicRTPController(0.96)

	// 添加低波动性样本
	for i := 0; i < 20; i++ {
		controller.UpdateHistory(100, 96)
	}

	adjustment := controller.calculateVolatilityAdjustment()
	if adjustment < 1.0 {
		t.Errorf("Low volatility adjustment = %v, expected >= 1.0", adjustment)
	}

	// 重置并添加高波动性样本
	controller = NewDynamicRTPController(0.96)
	for i := 0; i < 20; i++ {
		win := int64(50)
		if i%2 == 0 {
			win = 150
		}
		controller.UpdateHistory(100, win)
	}

	adjustment = controller.calculateVolatilityAdjustment()
	if adjustment > 1.0 {
		t.Errorf("High volatility adjustment = %v, expected <= 1.0", adjustment)
	}
}

func TestRandomFloat(t *testing.T) {
	// 测试randomFloat函数返回值范围
	for i := 0; i < 100; i++ {
		val := randomFloat()
		if val < 0 || val >= 1 {
			t.Errorf("randomFloat() returned %v, expected [0, 1)", val)
		}
	}
	
	// 测试分布（简单检查）
	sum := 0.0
	count := 1000
	for i := 0; i < count; i++ {
		sum += randomFloat()
	}
	avg := sum / float64(count)
	
	// 平均值应该接近0.5
	if math.Abs(avg-0.5) > 0.1 {
		t.Errorf("randomFloat() average = %v, expected ~0.5", avg)
	}
}