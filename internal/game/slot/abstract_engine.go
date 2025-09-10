package slot

import (
	"context"
	"fmt"
	"sync"
)

// AbstractGameResult 抽象游戏结果 - 纯数值逻辑，无具体图案
type AbstractGameResult struct {
	// 基础信息
	ResultID    string                 `json:"result_id"`
	BetAmount   int64                  `json:"bet_amount"`
	TotalWin    int64                  `json:"total_win"`
	IsWin       bool                   `json:"is_win"`
	WinType     WinType                `json:"win_type"`
	Multiplier  float64                `json:"multiplier"`
	
	// 抽象结果数据
	ReelResults [][]int                `json:"reel_results"`     // 数值化的转轴结果
	WinLines    []AbstractWinLine      `json:"win_lines"`        // 获胜线抽象描述
	Features    []AbstractFeature      `json:"features"`         // 特殊功能触发
	Statistics  *AbstractStatistics    `json:"statistics"`       // 统计数据
	Metadata    map[string]interface{} `json:"metadata"`         // 扩展数据
}

// AbstractWinLine 抽象获胜线
type AbstractWinLine struct {
	LineID      int                 `json:"line_id"`
	Positions   []GamePosition      `json:"positions"`       // 获胜位置
	SymbolID    int                 `json:"symbol_id"`       // 符号ID（数值）
	Count       int                 `json:"count"`           // 符号数量
	Payout      int64               `json:"payout"`          // 赔付金额
	Multiplier  float64             `json:"multiplier"`      // 倍数
	LineType    AbstractLineType    `json:"line_type"`       // 线型
	Properties  map[string]interface{} `json:"properties"`    // 额外属性
}

// AbstractFeature 抽象游戏特性
type AbstractFeature struct {
	FeatureID   int                    `json:"feature_id"`
	Type        AbstractFeatureType    `json:"type"`
	Trigger     []GamePosition         `json:"trigger"`         // 触发位置
	Value       int64                  `json:"value"`           // 特性值
	Duration    int                    `json:"duration"`        // 持续轮数
	Properties  map[string]interface{} `json:"properties"`      // 特性参数
}

// GamePosition 游戏位置信息（重命名避免冲突）
type GamePosition struct {
	Reel int `json:"reel"`
	Row  int `json:"row"`
}

// 枚举定义
type WinType int
const (
	WinTypeNone WinType = iota
	WinTypeSmall
	WinTypeMedium
	WinTypeBig
	WinTypeJackpot
)

type AbstractLineType int
const (
	AbstractLineTypeHorizontal AbstractLineType = iota
	AbstractLineTypeVertical
	AbstractLineTypeDiagonal
	AbstractLineTypeZigzag
	AbstractLineTypeCustom
)

type AbstractFeatureType int
const (
	AbstractFeatureTypeFreeSpin AbstractFeatureType = iota
	AbstractFeatureTypeBonus
	AbstractFeatureTypeWild
	AbstractFeatureTypeScatter
	AbstractFeatureTypeMultiplier
	AbstractFeatureTypeRespins
)

// AbstractStatistics 抽象统计数据
type AbstractStatistics struct {
	SpinCount       int64   `json:"spin_count"`
	TotalBet        int64   `json:"total_bet"`
	TotalWin        int64   `json:"total_win"`
	CurrentRTP      float64 `json:"current_rtp"`
	WinFrequency    float64 `json:"win_frequency"`
	BigWinCount     int64   `json:"big_win_count"`
	FeatureTriggers int64   `json:"feature_triggers"`
}

// AbstractGameEngine 抽象游戏引擎接口
type AbstractGameEngine interface {
	// 核心算法接口 - 只返回数值结果
	CalculateResult(ctx context.Context, request *GameRequest) (*AbstractGameResult, error)
	
	// 算法配置
	SetAlgorithmConfig(config *AlgorithmConfig) error
	GetAlgorithmConfig() *AlgorithmConfig
	
	// RTP控制
	SetTargetRTP(rtp float64) error
	GetCurrentRTP() float64
	
	// 统计信息
	GetStatistics() *AbstractStatistics
	ResetStatistics() error
	
	// 会话管理
	CreateSession(sessionID string) error
	GetSession(sessionID string) (*SessionData, error)
	CloseSession(sessionID string) error
}

// GameRequest 游戏请求
type GameRequest struct {
	SessionID    string                 `json:"session_id"`
	BetAmount    int64                  `json:"bet_amount"`
	SpinCount    int                    `json:"spin_count"`      // 批量旋转数量
	ForceResult  *int                   `json:"force_result"`    // 强制结果（测试用）
	Metadata     map[string]interface{} `json:"metadata"`        // 扩展参数
}

// AlgorithmConfig 算法配置
type AlgorithmConfig struct {
	// 基础参数
	ReelCount     int       `json:"reel_count"`
	RowCount      int       `json:"row_count"`
	SymbolCount   int       `json:"symbol_count"`
	
	// RTP设置
	TargetRTP     float64   `json:"target_rtp"`
	MinRTP        float64   `json:"min_rtp"`
	MaxRTP        float64   `json:"max_rtp"`
	
	// 符号权重 - 使用数值ID
	SymbolWeights [][]int   `json:"symbol_weights"`  // [reel][symbol_id] = weight
	
	// 赔付表 - 符号ID对应的赔付
	PayTable      map[int][]int64 `json:"pay_table"`   // symbol_id -> [count1_payout, count2_payout, ...]
	
	// 特殊符号配置
	WildSymbols    []int    `json:"wild_symbols"`
	ScatterSymbols []int    `json:"scatter_symbols"`
	BonusSymbols   []int    `json:"bonus_symbols"`
	
	// 特性配置
	FeatureConfigs map[AbstractFeatureType]*AbstractFeatureConfig `json:"feature_configs"`
	
	// 算法参数
	Algorithm      AlgorithmType `json:"algorithm"`
	Volatility     float64       `json:"volatility"`
	HitFrequency   float64       `json:"hit_frequency"`
}

type AlgorithmType int
const (
	AlgorithmTypeClassic AlgorithmType = iota
	AlgorithmTypeCascade
	AlgorithmTypeMegaways
	AlgorithmTypeCluster
)

// AbstractFeatureConfig 抽象特性配置
type AbstractFeatureConfig struct {
	TriggerSymbols []int     `json:"trigger_symbols"`
	MinCount       int       `json:"min_count"`
	BaseValue      int64     `json:"base_value"`
	Probability    float64   `json:"probability"`
	Properties     map[string]interface{} `json:"properties"`
}

// AbstractSlotEngine 抽象老虎机引擎实现
type AbstractSlotEngine struct {
	mu            sync.RWMutex
	config        *AlgorithmConfig
	rtpController *DynamicRTPController
	randomGen     RandomGenerator
	statistics    *AbstractStatistics
	sessions      map[string]*SessionData
	isRunning     bool
}

// NewAbstractSlotEngine 创建抽象游戏引擎
func NewAbstractSlotEngine(config *AlgorithmConfig) *AbstractSlotEngine {
	return &AbstractSlotEngine{
		config:        config,
		rtpController: NewDynamicRTPController(config.TargetRTP),
		randomGen:     NewCryptoRandomGenerator(),
		statistics:    &AbstractStatistics{},
		sessions:      make(map[string]*SessionData),
		isRunning:     true,
	}
}

// CalculateResult 核心算法 - 计算抽象结果
func (e *AbstractSlotEngine) CalculateResult(ctx context.Context, request *GameRequest) (*AbstractGameResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. 生成转轴结果（数值ID）
	reelResults := e.generateAbstractReels()
	
	// 2. 计算获胜线
	winLines := e.calculateWinLines(reelResults)
	
	// 3. 检测特殊功能
	features := e.detectFeatures(reelResults)
	
	// 4. 计算总赢取
	totalWin := e.calculateTotalWin(winLines, features)
	
	// 5. 应用RTP控制
	totalWin = e.applyRTPControl(totalWin, request.BetAmount)
	
	// 6. 更新统计
	e.updateStatistics(request.BetAmount, totalWin)
	
	// 7. 确定赢取类型
	winType := e.determineWinType(totalWin, request.BetAmount)
	
	return &AbstractGameResult{
		ResultID:    e.generateResultID(),
		BetAmount:   request.BetAmount,
		TotalWin:    totalWin,
		IsWin:       totalWin > 0,
		WinType:     winType,
		Multiplier:  float64(totalWin) / float64(request.BetAmount),
		ReelResults: reelResults,
		WinLines:    winLines,
		Features:    features,
		Statistics:  e.statistics,
		Metadata:    make(map[string]interface{}),
	}, nil
}

// generateAbstractReels 生成抽象转轴结果（只有数值ID）
func (e *AbstractSlotEngine) generateAbstractReels() [][]int {
	reelResults := make([][]int, e.config.ReelCount)
	
	for reelIndex := 0; reelIndex < e.config.ReelCount; reelIndex++ {
		reelResults[reelIndex] = make([]int, e.config.RowCount)
		
		for rowIndex := 0; rowIndex < e.config.RowCount; rowIndex++ {
			// 根据权重选择符号ID
			symbolID := e.selectSymbolByWeight(reelIndex)
			reelResults[reelIndex][rowIndex] = symbolID
		}
	}
	
	return reelResults
}

// selectSymbolByWeight 根据权重选择符号ID
func (e *AbstractSlotEngine) selectSymbolByWeight(reelIndex int) int {
	if reelIndex >= len(e.config.SymbolWeights) {
		return 0 // 默认符号
	}
	
	weights := e.config.SymbolWeights[reelIndex]
	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return 0
	}
	
	random := e.randomGen.NextInt(0, totalWeight)
	currentWeight := 0
	
	for symbolID, weight := range weights {
		currentWeight += weight
		if random < currentWeight {
			return symbolID
		}
	}
	
	return 0
}

// calculateWinLines 计算获胜线
func (e *AbstractSlotEngine) calculateWinLines(reelResults [][]int) []AbstractWinLine {
	var winLines []AbstractWinLine
	
	// 检查水平线
	for row := 0; row < e.config.RowCount; row++ {
		line := e.checkHorizontalLine(reelResults, row)
		if line != nil {
			winLines = append(winLines, *line)
		}
	}
	
	// 检查其他线型（对角线、之字形等）
	// ... 具体实现
	
	return winLines
}

// checkHorizontalLine 检查水平线
func (e *AbstractSlotEngine) checkHorizontalLine(reelResults [][]int, row int) *AbstractWinLine {
	if len(reelResults) == 0 || len(reelResults[0]) <= row {
		return nil
	}
	
	firstSymbol := reelResults[0][row]
	count := 1
	positions := []GamePosition{{Reel: 0, Row: row}}
	
	// 检查连续符号
	for reel := 1; reel < len(reelResults); reel++ {
		if len(reelResults[reel]) <= row {
			break
		}
		
		symbol := reelResults[reel][row]
		if symbol == firstSymbol || e.isWildSymbol(symbol) {
			count++
			positions = append(positions, GamePosition{Reel: reel, Row: row})
		} else {
			break
		}
	}
	
	// 计算赔付
	payout := e.calculateSymbolPayout(firstSymbol, count)
	if payout > 0 {
		return &AbstractWinLine{
			LineID:    row,
			Positions: positions,
			SymbolID:  firstSymbol,
			Count:     count,
			Payout:    payout,
			LineType:  AbstractLineTypeHorizontal,
		}
	}
	
	return nil
}

// 辅助方法
func (e *AbstractSlotEngine) isWildSymbol(symbolID int) bool {
	for _, wild := range e.config.WildSymbols {
		if wild == symbolID {
			return true
		}
	}
	return false
}

func (e *AbstractSlotEngine) calculateSymbolPayout(symbolID int, count int) int64 {
	payouts, exists := e.config.PayTable[symbolID]
	if !exists || count < 1 || count > len(payouts) {
		return 0
	}
	return payouts[count-1]
}

func (e *AbstractSlotEngine) detectFeatures(reelResults [][]int) []AbstractFeature {
	var features []AbstractFeature
	// 实现特殊功能检测逻辑
	return features
}

func (e *AbstractSlotEngine) calculateTotalWin(winLines []AbstractWinLine, features []AbstractFeature) int64 {
	total := int64(0)
	
	for _, line := range winLines {
		total += line.Payout
	}
	
	for _, feature := range features {
		total += feature.Value
	}
	
	return total
}

func (e *AbstractSlotEngine) applyRTPControl(totalWin, betAmount int64) int64 {
	// RTP控制逻辑
	currentRTP := e.rtpController.CalculateRTP(e.statistics.TotalWin, e.statistics.TotalBet)
	shouldTriggerWin := e.rtpController.ShouldTriggerWin(currentRTP, e.config.TargetRTP, betAmount)
	
	if !shouldTriggerWin && totalWin > 0 {
		// 降低赢取
		return int64(float64(totalWin) * 0.5)
	}
	
	if shouldTriggerWin && totalWin == 0 {
		// 强制小赢
		return betAmount / 2
	}
	
	return totalWin
}

func (e *AbstractSlotEngine) updateStatistics(betAmount, totalWin int64) {
	e.statistics.SpinCount++
	e.statistics.TotalBet += betAmount
	e.statistics.TotalWin += totalWin
	
	if e.statistics.TotalBet > 0 {
		e.statistics.CurrentRTP = float64(e.statistics.TotalWin) / float64(e.statistics.TotalBet)
	}
	
	if totalWin > 0 {
		e.statistics.WinFrequency = float64(e.statistics.SpinCount) / float64(e.statistics.SpinCount)
	}
	
	if totalWin > betAmount*10 {
		e.statistics.BigWinCount++
	}
}

func (e *AbstractSlotEngine) determineWinType(totalWin, betAmount int64) WinType {
	multiplier := float64(totalWin) / float64(betAmount)
	
	switch {
	case multiplier >= 100:
		return WinTypeJackpot
	case multiplier >= 20:
		return WinTypeBig
	case multiplier >= 5:
		return WinTypeMedium
	case multiplier > 0:
		return WinTypeSmall
	default:
		return WinTypeNone
	}
}

func (e *AbstractSlotEngine) generateResultID() string {
	return "result_" + e.generateRandomString(8)
}

// 其他接口方法实现...
func (e *AbstractSlotEngine) SetAlgorithmConfig(config *AlgorithmConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = config
	return nil
}

func (e *AbstractSlotEngine) GetAlgorithmConfig() *AlgorithmConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

func (e *AbstractSlotEngine) SetTargetRTP(rtp float64) error {
	// Note: Current RTPController interface doesn't support SetTargetRTP
	// Would need to recreate the controller or extend the interface
	return fmt.Errorf("SetTargetRTP not supported by current RTPController interface")
}

func (e *AbstractSlotEngine) GetCurrentRTP() float64 {
	return e.statistics.CurrentRTP
}

func (e *AbstractSlotEngine) GetStatistics() *AbstractStatistics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.statistics
}

func (e *AbstractSlotEngine) ResetStatistics() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.statistics = &AbstractStatistics{}
	return nil
}

func (e *AbstractSlotEngine) CreateSession(sessionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sessions[sessionID] = &SessionData{
		SessionID: sessionID,
		// ... 初始化会话数据
	}
	return nil
}

func (e *AbstractSlotEngine) GetSession(sessionID string) (*SessionData, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	session, exists := e.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (e *AbstractSlotEngine) CloseSession(sessionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.sessions, sessionID)
	return nil
}

var ErrSessionNotFound = fmt.Errorf("session not found")

// generateRandomString 生成随机字符串
func (e *AbstractSlotEngine) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[e.randomGen.NextInt(0, len(charset))]
	}
	return string(b)
}