package slot

import (
	"math"
	"sync"
	"time"
)

// DynamicRTPController 动态RTP控制器
type DynamicRTPController struct {
	mu                  sync.RWMutex
	targetRTP           float64             // 目标RTP
	shortTermWindow     time.Duration       // 短期窗口
	longTermWindow      time.Duration       // 长期窗口
	shortTermHistory    *RTPHistory         // 短期历史
	longTermHistory     *RTPHistory         // 长期历史
	compensationFactor  float64             // 补偿因子
	volatilityFactor    float64             // 波动因子
	minRTP              float64             // 最小RTP
	maxRTP              float64             // 最大RTP
}

// RTPHistory RTP历史记录
type RTPHistory struct {
	totalBet       int64
	totalWin       int64
	windowStart    time.Time
	windowDuration time.Duration
	samples        []RTPSample
	maxSamples     int
}

// RTPSample RTP样本
type RTPSample struct {
	timestamp time.Time
	bet       int64
	win       int64
	rtp       float64
}

// NewDynamicRTPController 创建动态RTP控制器
func NewDynamicRTPController(targetRTP float64) *DynamicRTPController {
	return &DynamicRTPController{
		targetRTP:          targetRTP,
		shortTermWindow:    15 * time.Minute,     // 15分钟短期窗口
		longTermWindow:     24 * time.Hour,       // 24小时长期窗口
		shortTermHistory:   newRTPHistory(15 * time.Minute, 100),
		longTermHistory:    newRTPHistory(24 * time.Hour, 1000),
		compensationFactor: 1.5,                  // 补偿强度
		volatilityFactor:   0.1,                  // 波动因子
		minRTP:             targetRTP * 0.85,     // 最小85%目标RTP
		maxRTP:             targetRTP * 1.15,     // 最大115%目标RTP
	}
}

// newRTPHistory 创建RTP历史记录
func newRTPHistory(window time.Duration, maxSamples int) *RTPHistory {
	return &RTPHistory{
		windowStart:    time.Now(),
		windowDuration: window,
		samples:        make([]RTPSample, 0, maxSamples),
		maxSamples:     maxSamples,
	}
}

// AdjustOdds 根据当前RTP调整赔率
func (c *DynamicRTPController) AdjustOdds(currentRTP, targetRTP float64) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 计算RTP偏差
	deviation := targetRTP - currentRTP
	
	// 计算调整系数
	// 如果当前RTP低于目标，增加赔率；反之减少
	adjustmentFactor := 1.0 + (deviation * c.compensationFactor)
	
	// 应用波动性调整
	volatilityAdjustment := c.calculateVolatilityAdjustment()
	adjustmentFactor *= volatilityAdjustment
	
	// 限制调整范围
	adjustmentFactor = math.Max(0.5, math.Min(2.0, adjustmentFactor))
	
	return adjustmentFactor
}

// ShouldTriggerWin 判断是否应该触发中奖
func (c *DynamicRTPController) ShouldTriggerWin(currentRTP, targetRTP float64, betAmount int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 基础中奖概率
	baseWinProbability := targetRTP
	
	// 短期和长期RTP的加权
	shortTermRTP := c.shortTermHistory.getCurrentRTP()
	longTermRTP := c.longTermHistory.getCurrentRTP()
	
	// 计算加权RTP (短期权重30%, 长期权重70%)
	weightedRTP := shortTermRTP*0.3 + longTermRTP*0.7
	
	// 调整中奖概率
	if weightedRTP < targetRTP {
		// RTP低于目标，提高中奖概率
		compensation := (targetRTP - weightedRTP) * c.compensationFactor
		baseWinProbability += compensation
	} else if weightedRTP > targetRTP {
		// RTP高于目标，降低中奖概率
		reduction := (weightedRTP - targetRTP) * c.compensationFactor * 0.5
		baseWinProbability -= reduction
	}
	
	// 根据下注金额调整（大额下注略微降低中奖率）
	betFactor := 1.0 - (float64(betAmount) / 1000000.0) * 0.05
	baseWinProbability *= betFactor
	
	// 添加随机波动
	volatility := c.volatilityFactor * (0.5 - randomFloat()) // randomFloat() 返回0-1的随机数
	baseWinProbability += volatility
	
	// 限制概率范围
	baseWinProbability = math.Max(0.1, math.Min(0.9, baseWinProbability))
	
	// 生成随机数判断
	return randomFloat() < baseWinProbability
}

// CalculateRTP 计算当前RTP
func (c *DynamicRTPController) CalculateRTP(totalWin, totalBet int64) float64 {
	if totalBet == 0 {
		return 0
	}
	return float64(totalWin) / float64(totalBet)
}

// GetCompensationMultiplier 获取补偿倍率
func (c *DynamicRTPController) GetCompensationMultiplier(currentRTP, targetRTP float64) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 计算偏差百分比
	deviationPercent := (targetRTP - currentRTP) / targetRTP
	
	// 根据偏差计算补偿倍率
	var multiplier float64
	
	if deviationPercent > 0.1 {
		// RTP严重偏低，强补偿
		multiplier = 1.0 + (deviationPercent * 2.0)
	} else if deviationPercent > 0.05 {
		// RTP偏低，中等补偿
		multiplier = 1.0 + (deviationPercent * 1.5)
	} else if deviationPercent > -0.05 {
		// RTP接近目标，轻微调整
		multiplier = 1.0 + (deviationPercent * 1.0)
	} else if deviationPercent > -0.1 {
		// RTP偏高，中等抑制
		multiplier = 1.0 + (deviationPercent * 0.5)
	} else {
		// RTP严重偏高，强抑制
		multiplier = 1.0 + (deviationPercent * 0.3)
	}
	
	// 限制倍率范围
	multiplier = math.Max(0.5, math.Min(2.0, multiplier))
	
	return multiplier
}

// UpdateHistory 更新历史记录
func (c *DynamicRTPController) UpdateHistory(bet, win int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sample := RTPSample{
		timestamp: time.Now(),
		bet:       bet,
		win:       win,
		rtp:       c.CalculateRTP(win, bet),
	}
	
	c.shortTermHistory.addSample(sample)
	c.longTermHistory.addSample(sample)
}

// GetStatistics 获取统计信息
func (c *DynamicRTPController) GetStatistics() *RTPStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &RTPStatistics{
		TargetRTP:        c.targetRTP,
		ShortTermRTP:     c.shortTermHistory.getCurrentRTP(),
		LongTermRTP:      c.longTermHistory.getCurrentRTP(),
		ShortTermSamples: len(c.shortTermHistory.samples),
		LongTermSamples:  len(c.longTermHistory.samples),
		LastUpdate:       time.Now(),
	}
}

// calculateVolatilityAdjustment 计算波动性调整
func (c *DynamicRTPController) calculateVolatilityAdjustment() float64 {
	// 计算短期RTP的标准差
	samples := c.shortTermHistory.samples
	if len(samples) < 10 {
		return 1.0
	}
	
	// 计算平均RTP
	var sum float64
	for _, s := range samples {
		sum += s.rtp
	}
	mean := sum / float64(len(samples))
	
	// 计算标准差
	var variance float64
	for _, s := range samples {
		diff := s.rtp - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(samples)))
	
	// 根据标准差调整
	// 高波动性时减少调整幅度，低波动性时增加调整幅度
	if stdDev > 0.1 {
		return 0.8 // 高波动，减少调整
	} else if stdDev < 0.05 {
		return 1.2 // 低波动，增加调整
	}
	return 1.0
}

// addSample 添加样本到历史记录
func (h *RTPHistory) addSample(sample RTPSample) {
	h.totalBet += sample.bet
	h.totalWin += sample.win
	
	// 清理过期样本
	cutoff := time.Now().Add(-h.windowDuration)
	validSamples := make([]RTPSample, 0)
	
	for _, s := range h.samples {
		if s.timestamp.After(cutoff) {
			validSamples = append(validSamples, s)
		} else {
			// 从总计中减去过期样本
			h.totalBet -= s.bet
			h.totalWin -= s.win
		}
	}
	
	h.samples = validSamples
	
	// 添加新样本
	h.samples = append(h.samples, sample)
	
	// 限制样本数量
	if len(h.samples) > h.maxSamples {
		removed := h.samples[0]
		h.samples = h.samples[1:]
		h.totalBet -= removed.bet
		h.totalWin -= removed.win
	}
}

// getCurrentRTP 获取当前RTP
func (h *RTPHistory) getCurrentRTP() float64 {
	if h.totalBet == 0 {
		return 0
	}
	return float64(h.totalWin) / float64(h.totalBet)
}

// RTPStatistics RTP统计信息
type RTPStatistics struct {
	TargetRTP        float64   `json:"target_rtp"`
	ShortTermRTP     float64   `json:"short_term_rtp"`
	LongTermRTP      float64   `json:"long_term_rtp"`
	ShortTermSamples int       `json:"short_term_samples"`
	LongTermSamples  int       `json:"long_term_samples"`
	LastUpdate       time.Time `json:"last_update"`
}

// randomFloat 生成0-1的随机数
func randomFloat() float64 {
	rng := NewCryptoRandomGenerator()
	return rng.Next()
}