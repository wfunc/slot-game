package slot

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	ErrInvalidBet         = errors.New("无效的下注金额")
	ErrInvalidConfig      = errors.New("无效的配置")
	ErrInvalidRTP         = errors.New("无效的RTP设置")
	ErrInvalidReelStrips  = errors.New("无效的卷轴条配置")
	ErrInvalidPayTable    = errors.New("无效的赔率表")
	ErrEngineNotReady     = errors.New("引擎未就绪")
)

// SlotEngine 老虎机游戏引擎
type SlotEngine struct {
	mu             sync.RWMutex
	config         *SlotConfig
	rtpController  RTPController
	patternMatcher PatternMatcher
	randomGen      RandomGenerator
	statistics     *Statistics
	sessionData    map[string]*SessionData
	isRunning      bool
}

// SessionData 会话数据
type SessionData struct {
	UserID         uint
	SessionID      string
	TotalBet       int64
	TotalWin       int64
	SpinCount      int
	FreeSpinsLeft  int
	LastSpinResult *SpinResult
	CreatedAt      time.Time
	LastActiveAt   time.Time
}

// NewSlotEngine 创建老虎机引擎
func NewSlotEngine(config *SlotConfig) (*SlotEngine, error) {
	// 验证配置
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}
	
	engine := &SlotEngine{
		config:         config,
		rtpController:  NewDynamicRTPController(config.TargetRTP),
		patternMatcher: NewAdvancedPatternMatcher(config),
		randomGen:      NewCryptoRandomGenerator(),
		statistics: &Statistics{
			LastUpdate: time.Now(),
		},
		sessionData: make(map[string]*SessionData),
		isRunning:   true,
	}
	
	return engine, nil
}

// Spin 执行旋转
func (e *SlotEngine) Spin(userID uint, sessionID string, betAmount int64) (*SpinResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// 检查引擎状态
	if !e.isRunning {
		return nil, ErrEngineNotReady
	}
	
	// 验证下注金额
	if betAmount < e.config.MinBet || betAmount > e.config.MaxBet {
		return nil, ErrInvalidBet
	}
	
	// 获取或创建会话数据
	session := e.getOrCreateSession(userID, sessionID)
	
	// 生成结果ID
	resultID := e.generateResultID()
	
	// 判断是否免费旋转
	isFreeSpins := session.FreeSpinsLeft > 0
	if isFreeSpins {
		session.FreeSpinsLeft--
		betAmount = 0 // 免费旋转不扣费
	}
	
	// 更新统计
	e.statistics.TotalSpins++
	e.statistics.TotalBet += betAmount
	session.TotalBet += betAmount
	session.SpinCount++
	
	// 获取当前RTP
	currentRTP := e.rtpController.CalculateRTP(e.statistics.TotalWin, e.statistics.TotalBet)
	
	// 判断是否应该触发中奖
	shouldWin := e.rtpController.ShouldTriggerWin(currentRTP, e.config.TargetRTP, betAmount)
	
	// 生成卷轴结果
	reels := e.generateReels(shouldWin)
	
	// 查找中奖线
	winLines := e.patternMatcher.FindWinningLines(reels, e.config)
	
	// 计算赔付
	winAmount := e.patternMatcher.CalculatePayout(winLines, betAmount)
	
	// 应用RTP补偿
	if shouldWin && winAmount == 0 {
		// 强制产生小奖（降低倍率以控制RTP）
		winAmount = betAmount / 2 // 0.5倍赔付
		winLines = e.forceSmallWin(reels)
	} else if !shouldWin && winAmount > 0 {
		// 减少赔付
		compensationMultiplier := e.rtpController.GetCompensationMultiplier(currentRTP, e.config.TargetRTP)
		winAmount = int64(float64(winAmount) * compensationMultiplier * 0.8) // 额外降低20%
	}
	
	// 检测特殊功能
	features := e.patternMatcher.DetectFeatures(reels, e.config)
	
	// 处理特殊功能
	freeSpinsAwarded := 0
	isJackpot := false
	for _, feature := range features {
		switch feature.Type {
		case FeatureTypeFreeSpins:
			if spins, ok := feature.Value.(int); ok {
				freeSpinsAwarded += spins
				session.FreeSpinsLeft += spins
			}
		case FeatureTypeMultiplier:
			if multiplier, ok := feature.Value.(float64); ok {
				winAmount = int64(float64(winAmount) * multiplier)
			}
		case FeatureTypeBonus:
			// 奖励游戏可以获得额外奖金
			bonusWin := e.calculateBonusWin(betAmount)
			winAmount += bonusWin
			if bonusWin > betAmount*50 { // 降低jackpot阈值
				isJackpot = true
			}
		}
	}
	
	// 检查是否中大奖
	if winAmount > betAmount*100 { // 提高jackpot阈值到100倍
		isJackpot = true
		e.statistics.JackpotHits++
	}
	if winAmount > betAmount*20 {
		e.statistics.BigWins++
	}
	
	// 更新统计
	e.statistics.TotalWin += winAmount
	session.TotalWin += winAmount
	e.statistics.FreeSpinsTotal += freeSpinsAwarded
	e.statistics.CurrentRTP = e.rtpController.CalculateRTP(e.statistics.TotalWin, e.statistics.TotalBet)
	e.statistics.LastUpdate = time.Now()
	
	// 更新RTP控制器历史
	if rtpCtrl, ok := e.rtpController.(*DynamicRTPController); ok {
		rtpCtrl.UpdateHistory(betAmount, winAmount)
	}
	
	// 创建旋转结果
	result := &SpinResult{
		ID:         resultID,
		SessionID:  sessionID,
		UserID:     userID,
		BetAmount:  betAmount,
		WinAmount:  winAmount,
		Multiplier: float64(winAmount) / float64(betAmount+1), // 避免除零
		Reels:      reels,
		WinLines:   winLines,
		Features:   features,
		FreeSpins:  freeSpinsAwarded,
		IsJackpot:  isJackpot,
		RTP:        e.statistics.CurrentRTP,
		Timestamp:  time.Now(),
	}
	
	// 保存结果到会话
	session.LastSpinResult = result
	session.LastActiveAt = time.Now()
	
	return result, nil
}

// generateReels 生成卷轴结果
func (e *SlotEngine) generateReels(favorWin bool) [][]Symbol {
	reels := make([][]Symbol, e.config.Reels)
	
	for i := 0; i < e.config.Reels; i++ {
		reels[i] = make([]Symbol, e.config.Rows)
		reelStrip := e.config.ReelStrips[i]
		
		for j := 0; j < e.config.Rows; j++ {
			// 根据权重选择符号
			symbol := e.selectSymbolByWeight(reelStrip, favorWin)
			reels[i][j] = symbol
		}
	}
	
	// 如果需要中奖，可能调整某些位置
	if favorWin {
		e.adjustForWin(reels)
	}
	
	return reels
}

// selectSymbolByWeight 根据权重选择符号
func (e *SlotEngine) selectSymbolByWeight(reelStrip ReelStrip, favorWin bool) Symbol {
	totalWeight := 0
	for _, weight := range reelStrip.Weights {
		totalWeight += weight
	}
	
	// 生成随机数
	randomValue := e.randomGen.NextInt(0, totalWeight)
	
	// 如果倾向于中奖，调整权重
	if favorWin {
		// 增加高价值符号的权重
		randomValue = e.adjustRandomForWin(randomValue, totalWeight)
	}
	
	// 选择符号
	currentWeight := 0
	for i, weight := range reelStrip.Weights {
		currentWeight += weight
		if randomValue < currentWeight {
			return reelStrip.Symbols[i]
		}
	}
	
	// 默认返回第一个符号
	return reelStrip.Symbols[0]
}

// adjustForWin 调整卷轴以产生中奖
func (e *SlotEngine) adjustForWin(reels [][]Symbol) {
	// 简单策略：在中间行放置相同的符号
	if e.config.Rows >= 3 && e.config.Reels >= 3 {
		middleRow := e.config.Rows / 2
		// 选择一个中等价值的符号
		winSymbol := SymbolOrange
		
		// 在前3个卷轴的中间行放置相同符号
		for i := 0; i < 3 && i < e.config.Reels; i++ {
			reels[i][middleRow] = winSymbol
		}
	}
}

// adjustRandomForWin 调整随机数以倾向中奖
func (e *SlotEngine) adjustRandomForWin(randomValue, totalWeight int) int {
	// 将随机值向高价值符号偏移
	adjustment := totalWeight / 10 // 10%的偏移
	randomValue -= adjustment
	if randomValue < 0 {
		randomValue = 0
	}
	return randomValue
}

// forceSmallWin 强制产生小奖
func (e *SlotEngine) forceSmallWin(reels [][]Symbol) []WinLine {
	// 创建一个虚拟的中奖线
	return []WinLine{
		{
			LineID:     0,
			LineType:   LineTypeHorizontal,
			Symbol:     SymbolCherry,
			Count:      3,
			Positions:  []Position{{0, 0}, {1, 0}, {2, 0}},
			WinAmount:  0,
			Multiplier: 0.5, // 降低倍率
		},
	}
}

// calculateBonusWin 计算奖励游戏赢取
func (e *SlotEngine) calculateBonusWin(betAmount int64) int64 {
	// 随机倍率 5-20倍（降低以维持合理RTP）
	multiplier := e.randomGen.NextInt(5, 20)
	return betAmount * int64(multiplier)
}

// getOrCreateSession 获取或创建会话
func (e *SlotEngine) getOrCreateSession(userID uint, sessionID string) *SessionData {
	if session, exists := e.sessionData[sessionID]; exists {
		return session
	}
	
	session := &SessionData{
		UserID:       userID,
		SessionID:    sessionID,
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	e.sessionData[sessionID] = session
	return session
}

// generateResultID 生成结果ID
func (e *SlotEngine) generateResultID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetConfig 获取配置
func (e *SlotEngine) GetConfig() *SlotConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// SetRTPController 设置RTP控制器
func (e *SlotEngine) SetRTPController(controller RTPController) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rtpController = controller
}

// GetStatistics 获取统计数据
func (e *SlotEngine) GetStatistics() *Statistics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.statistics
}

// Reset 重置引擎
func (e *SlotEngine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.statistics = &Statistics{
		LastUpdate: time.Now(),
	}
	e.sessionData = make(map[string]*SessionData)
}

// GetSession 获取会话数据
func (e *SlotEngine) GetSession(sessionID string) *SessionData {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.sessionData[sessionID]
}

// CleanupSessions 清理过期会话
func (e *SlotEngine) CleanupSessions(maxAge time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	for sessionID, session := range e.sessionData {
		if session.LastActiveAt.Before(cutoff) {
			delete(e.sessionData, sessionID)
		}
	}
}

// Stop 停止引擎
func (e *SlotEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.isRunning = false
}

// Start 启动引擎
func (e *SlotEngine) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.isRunning = true
}

// CryptoRandomGenerator 加密安全的随机数生成器
type CryptoRandomGenerator struct{}

// NewCryptoRandomGenerator 创建加密随机数生成器
func NewCryptoRandomGenerator() *CryptoRandomGenerator {
	return &CryptoRandomGenerator{}
}

// Next 生成下一个随机数 (0-1)
func (g *CryptoRandomGenerator) Next() float64 {
	max := big.NewInt(1000000)
	n, _ := rand.Int(rand.Reader, max)
	return float64(n.Int64()) / 1000000.0
}

// NextInt 生成指定范围内的随机整数
func (g *CryptoRandomGenerator) NextInt(min, max int) int {
	if min >= max {
		return min
	}
	diff := big.NewInt(int64(max - min))
	n, _ := rand.Int(rand.Reader, diff)
	return min + int(n.Int64())
}

// Seed 设置种子（加密随机数不需要种子）
func (g *CryptoRandomGenerator) Seed(seed int64) {
	// 加密随机数生成器不需要种子
}

// init函数已不需要，randomFloat函数在rtp.go中定义
func init() {
	// 初始化代码（如果需要）
}

// GetRTPInfo 获取RTP信息
func (e *SlotEngine) GetRTPInfo() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	info := map[string]interface{}{
		"target_rtp":  e.config.TargetRTP,
		"current_rtp": e.statistics.CurrentRTP,
		"total_spins": e.statistics.TotalSpins,
		"total_bet":   e.statistics.TotalBet,
		"total_win":   e.statistics.TotalWin,
	}
	
	// 如果是动态RTP控制器，获取更多信息
	if rtpCtrl, ok := e.rtpController.(*DynamicRTPController); ok {
		stats := rtpCtrl.GetStatistics()
		info["short_term_rtp"] = stats.ShortTermRTP
		info["long_term_rtp"] = stats.LongTermRTP
	}
	
	return info
}

// SimulateBatch 批量模拟（用于测试RTP）
func (e *SlotEngine) SimulateBatch(spins int, betAmount int64) *SimulationResult {
	totalBet := int64(0)
	totalWin := int64(0)
	bigWins := 0
	jackpots := 0
	
	for i := 0; i < spins; i++ {
		result, err := e.Spin(0, fmt.Sprintf("sim_%d", i), betAmount)
		if err != nil {
			continue
		}
		
		totalBet += result.BetAmount
		totalWin += result.WinAmount
		
		if result.WinAmount > result.BetAmount*20 {
			bigWins++
		}
		if result.IsJackpot {
			jackpots++
		}
	}
	
	return &SimulationResult{
		TotalSpins: spins,
		TotalBet:   totalBet,
		TotalWin:   totalWin,
		RTP:        float64(totalWin) / float64(totalBet),
		BigWins:    bigWins,
		Jackpots:   jackpots,
	}
}

// SimulationResult 模拟结果
type SimulationResult struct {
	TotalSpins int     `json:"total_spins"`
	TotalBet   int64   `json:"total_bet"`
	TotalWin   int64   `json:"total_win"`
	RTP        float64 `json:"rtp"`
	BigWins    int     `json:"big_wins"`
	Jackpots   int     `json:"jackpots"`
}