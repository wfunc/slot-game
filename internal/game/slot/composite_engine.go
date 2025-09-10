package slot

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CompositeSlotEngine 组合老虎机引擎 - 算法与主题分离的完整实现
type CompositeSlotEngine struct {
	mu            sync.RWMutex
	
	// 核心组件
	abstractEngine AbstractGameEngine  // 抽象算法引擎
	themeManager   *ThemeManager       // 主题管理器
	
	// 配置
	defaultTheme   string              // 默认主题
	enableThemes   bool                // 是否启用主题系统
	
	// 统计和会话
	isRunning      bool
}

// NewCompositeSlotEngine 创建组合引擎
func NewCompositeSlotEngine(config *AlgorithmConfig, defaultTheme string) *CompositeSlotEngine {
	return &CompositeSlotEngine{
		abstractEngine: NewAbstractSlotEngine(config),
		themeManager:   NewThemeManager(),
		defaultTheme:   defaultTheme,
		enableThemes:   true,
		isRunning:      true,
	}
}

// SpinRequest 旋转请求 - 包含主题信息
type SpinRequest struct {
	*GameRequest
	ThemeID      string `json:"theme_id"`       // 指定主题
	EnableTheme  bool   `json:"enable_theme"`   // 是否启用主题渲染
}

// SpinResponse 旋转响应 - 支持多种格式
type SpinResponse struct {
	// 基础信息
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
	
	// 结果数据 - 根据请求类型返回不同格式
	AbstractResult *AbstractGameResult  `json:"abstract_result,omitempty"`  // 纯算法结果
	ThemedResult   *ThemedGameResult    `json:"themed_result,omitempty"`    // 主题化结果
	
	// 性能信息
	ProcessingTime int64                `json:"processing_time"`  // 处理时间(ms)
	EngineVersion  string               `json:"engine_version"`
}

// Spin 主要的旋转接口 - 智能选择返回格式
func (e *CompositeSlotEngine) Spin(ctx context.Context, request *SpinRequest) (*SpinResponse, error) {
	e.mu.RLock()
	if !e.isRunning {
		e.mu.RUnlock()
		return &SpinResponse{
			Success: false,
			Message: "engine is not running",
		}, fmt.Errorf("engine is not running")
	}
	e.mu.RUnlock()
	
	startTime := getCurrentTimeMs()
	
	// 1. 执行抽象算法计算
	abstractResult, err := e.abstractEngine.CalculateResult(ctx, request.GameRequest)
	if err != nil {
		return &SpinResponse{
			Success: false,
			Message: fmt.Sprintf("algorithm calculation failed: %v", err),
		}, err
	}
	
	response := &SpinResponse{
		Success:        true,
		AbstractResult: abstractResult,
		ProcessingTime: getCurrentTimeMs() - startTime,
		EngineVersion:  "2.0.0-composite",
	}
	
	// 2. 如果需要主题渲染，则进行主题化
	if e.enableThemes && request.EnableTheme {
		themeID := request.ThemeID
		if themeID == "" {
			themeID = e.defaultTheme
		}
		
		themedResult, err := e.themeManager.ProcessGameResult(abstractResult, themeID)
		if err != nil {
			// 主题渲染失败不影响核心结果，只记录警告
			response.Message = fmt.Sprintf("theme rendering failed: %v", err)
		} else {
			response.ThemedResult = themedResult
			// 如果主题渲染成功，可以选择不返回抽象结果以节省带宽
			if !request.GameRequest.Metadata["include_abstract"].(bool) {
				response.AbstractResult = nil
			}
		}
	}
	
	response.ProcessingTime = getCurrentTimeMs() - startTime
	return response, nil
}

// SpinAbstract 仅执行算法计算，返回抽象结果
func (e *CompositeSlotEngine) SpinAbstract(ctx context.Context, request *GameRequest) (*AbstractGameResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if !e.isRunning {
		return nil, fmt.Errorf("engine is not running")
	}
	
	return e.abstractEngine.CalculateResult(ctx, request)
}

// SpinWithTheme 执行完整的算法+主题渲染
func (e *CompositeSlotEngine) SpinWithTheme(ctx context.Context, request *GameRequest, themeID string) (*ThemedGameResult, error) {
	// 1. 获取抽象结果
	abstractResult, err := e.SpinAbstract(ctx, request)
	if err != nil {
		return nil, err
	}
	
	// 2. 应用主题渲染
	if themeID == "" {
		themeID = e.defaultTheme
	}
	
	return e.themeManager.ProcessGameResult(abstractResult, themeID)
}

// BatchSpin 批量旋转 - 优化性能
func (e *CompositeSlotEngine) BatchSpin(ctx context.Context, request *SpinRequest, count int) (*BatchSpinResponse, error) {
	e.mu.RLock()
	if !e.isRunning {
		e.mu.RUnlock()
		return nil, fmt.Errorf("engine is not running")
	}
	e.mu.RUnlock()
	
	startTime := getCurrentTimeMs()
	
	// 批量执行抽象算法
	abstractResults := make([]*AbstractGameResult, 0, count)
	for i := 0; i < count; i++ {
		result, err := e.abstractEngine.CalculateResult(ctx, request.GameRequest)
		if err != nil {
			return nil, fmt.Errorf("batch spin failed at index %d: %w", i, err)
		}
		abstractResults = append(abstractResults, result)
	}
	
	// 聚合统计
	aggregated := e.aggregateResults(abstractResults)
	
	response := &BatchSpinResponse{
		Success:         true,
		Count:           count,
		AbstractResults: abstractResults,
		Aggregated:      aggregated,
		ProcessingTime:  getCurrentTimeMs() - startTime,
		EngineVersion:   "2.0.0-composite",
	}
	
	// 可选的主题渲染（通常批量旋转不需要主题渲染以提高性能）
	if e.enableThemes && request.EnableTheme {
		themeID := request.ThemeID
		if themeID == "" {
			themeID = e.defaultTheme
		}
		
		themedResults := make([]*ThemedGameResult, 0, count)
		for _, abstractResult := range abstractResults {
			themed, err := e.themeManager.ProcessGameResult(abstractResult, themeID)
			if err != nil {
				continue // 跳过失败的主题渲染
			}
			themedResults = append(themedResults, themed)
		}
		response.ThemedResults = themedResults
	}
	
	return response, nil
}

// BatchSpinResponse 批量旋转响应
type BatchSpinResponse struct {
	Success         bool                    `json:"success"`
	Count           int                     `json:"count"`
	AbstractResults []*AbstractGameResult   `json:"abstract_results,omitempty"`
	ThemedResults   []*ThemedGameResult     `json:"themed_results,omitempty"`
	Aggregated      *AggregatedResult       `json:"aggregated"`
	ProcessingTime  int64                   `json:"processing_time"`
	EngineVersion   string                  `json:"engine_version"`
}

// AggregatedResult 聚合结果统计
type AggregatedResult struct {
	TotalSpins     int                `json:"total_spins"`
	TotalBet       int64              `json:"total_bet"`
	TotalWin       int64              `json:"total_win"`
	NetResult      int64              `json:"net_result"`
	WinCount       int                `json:"win_count"`
	WinRate        float64            `json:"win_rate"`
	AverageWin     float64            `json:"average_win"`
	BigWinCount    int                `json:"big_win_count"`
	MaxWin         int64              `json:"max_win"`
	WinDistribution map[WinType]int   `json:"win_distribution"`
	FeatureCount   int                `json:"feature_count"`
}

// aggregateResults 聚合批量结果
func (e *CompositeSlotEngine) aggregateResults(results []*AbstractGameResult) *AggregatedResult {
	if len(results) == 0 {
		return &AggregatedResult{}
	}
	
	agg := &AggregatedResult{
		TotalSpins:      len(results),
		WinDistribution: make(map[WinType]int),
	}
	
	for _, result := range results {
		agg.TotalBet += result.BetAmount
		agg.TotalWin += result.TotalWin
		
		if result.IsWin {
			agg.WinCount++
		}
		
		if result.TotalWin > result.BetAmount*10 {
			agg.BigWinCount++
		}
		
		if result.TotalWin > agg.MaxWin {
			agg.MaxWin = result.TotalWin
		}
		
		agg.WinDistribution[result.WinType]++
		agg.FeatureCount += len(result.Features)
	}
	
	agg.NetResult = agg.TotalWin - agg.TotalBet
	agg.WinRate = float64(agg.WinCount) / float64(agg.TotalSpins)
	
	if agg.WinCount > 0 {
		agg.AverageWin = float64(agg.TotalWin) / float64(agg.WinCount)
	}
	
	return agg
}

// 配置管理接口
func (e *CompositeSlotEngine) SetAlgorithmConfig(config *AlgorithmConfig) error {
	return e.abstractEngine.SetAlgorithmConfig(config)
}

func (e *CompositeSlotEngine) GetAlgorithmConfig() *AlgorithmConfig {
	return e.abstractEngine.GetAlgorithmConfig()
}

func (e *CompositeSlotEngine) SetDefaultTheme(themeID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// 验证主题是否存在
	themes := e.themeManager.GetAvailableThemes()
	for _, theme := range themes {
		if theme == themeID {
			e.defaultTheme = themeID
			return nil
		}
	}
	
	return fmt.Errorf("theme not found: %s", themeID)
}

func (e *CompositeSlotEngine) GetDefaultTheme() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.defaultTheme
}

func (e *CompositeSlotEngine) GetAvailableThemes() []string {
	return e.themeManager.GetAvailableThemes()
}

func (e *CompositeSlotEngine) LoadThemeFromJSON(jsonData []byte) error {
	return e.themeManager.LoadThemeFromJSON(jsonData)
}

// RTP控制接口
func (e *CompositeSlotEngine) SetTargetRTP(rtp float64) error {
	return e.abstractEngine.SetTargetRTP(rtp)
}

func (e *CompositeSlotEngine) GetCurrentRTP() float64 {
	return e.abstractEngine.GetCurrentRTP()
}

// 统计接口
func (e *CompositeSlotEngine) GetStatistics() *AbstractStatistics {
	return e.abstractEngine.GetStatistics()
}

func (e *CompositeSlotEngine) ResetStatistics() error {
	return e.abstractEngine.ResetStatistics()
}

// 会话管理接口
func (e *CompositeSlotEngine) CreateSession(sessionID string) error {
	return e.abstractEngine.CreateSession(sessionID)
}

func (e *CompositeSlotEngine) GetSession(sessionID string) (*SessionData, error) {
	return e.abstractEngine.GetSession(sessionID)
}

func (e *CompositeSlotEngine) CloseSession(sessionID string) error {
	return e.abstractEngine.CloseSession(sessionID)
}

// 引擎控制接口
func (e *CompositeSlotEngine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.isRunning = true
	return nil
}

func (e *CompositeSlotEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.isRunning = false
	return nil
}

func (e *CompositeSlotEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

// 主题控制接口
func (e *CompositeSlotEngine) EnableThemes(enable bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enableThemes = enable
}

func (e *CompositeSlotEngine) IsThemesEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enableThemes
}

// 调试和测试接口
func (e *CompositeSlotEngine) SimulateWithConfig(ctx context.Context, config *SimulationConfig) (*CompositeSimulationResult, error) {
	// 暂时禁用主题渲染以提高性能
	originalThemeState := e.enableThemes
	e.EnableThemes(false)
	defer e.EnableThemes(originalThemeState)
	
	results := make([]*AbstractGameResult, 0, config.SpinCount)
	
	for i := 0; i < config.SpinCount; i++ {
		request := &GameRequest{
			SessionID: "simulation",
			BetAmount: config.BetAmount,
			Metadata:  make(map[string]interface{}),
		}
		
		result, err := e.abstractEngine.CalculateResult(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("simulation failed at spin %d: %w", i, err)
		}
		
		results = append(results, result)
	}
	
	// 计算仿真统计
	return e.calculateSimulationStats(results, config), nil
}

// SimulationConfig 仿真配置
type SimulationConfig struct {
	SpinCount int   `json:"spin_count"`
	BetAmount int64 `json:"bet_amount"`
	TargetRTP float64 `json:"target_rtp"`
}

// CompositeSimulationResult 组合引擎仿真结果
type CompositeSimulationResult struct {
	Config          *SimulationConfig `json:"config"`
	Results         []*AbstractGameResult `json:"results"`
	Aggregated      *AggregatedResult `json:"aggregated"`
	ActualRTP       float64           `json:"actual_rtp"`
	RTTDeviation    float64           `json:"rtp_deviation"`
	Confidence      float64           `json:"confidence"`
}

// calculateSimulationStats 计算仿真统计
func (e *CompositeSlotEngine) calculateSimulationStats(results []*AbstractGameResult, config *SimulationConfig) *CompositeSimulationResult {
	aggregated := e.aggregateResults(results)
	
	actualRTP := float64(0)
	if aggregated.TotalBet > 0 {
		actualRTP = float64(aggregated.TotalWin) / float64(aggregated.TotalBet)
	}
	
	rtpDeviation := actualRTP - config.TargetRTP
	confidence := calculateConfidenceLevel(len(results), actualRTP, config.TargetRTP)
	
	return &CompositeSimulationResult{
		Config:       config,
		Results:      results,
		Aggregated:   aggregated,
		ActualRTP:    actualRTP,
		RTTDeviation: rtpDeviation,
		Confidence:   confidence,
	}
}

// 工具函数
func getCurrentTimeMs() int64 {
	return time.Now().UnixMilli()
}

func calculateConfidenceLevel(sampleSize int, actualRTP, targetRTP float64) float64 {
	// 简化的置信度计算
	deviation := abs(actualRTP - targetRTP)
	sampleFactor := min(float64(sampleSize)/10000.0, 1.0)  // 样本越大置信度越高
	return max(0, 1.0 - deviation*10) * sampleFactor
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}