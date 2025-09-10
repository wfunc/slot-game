package slot

import (
	"context"
	"fmt"
	"time"
)

// SlotGameService 老虎机游戏服务 - API集成层
type SlotGameService struct {
	engine       *CompositeSlotEngine
	configLoader *ConfigLoader
}

// ConfigLoader 配置加载器 - 兼容现有配置系统
type ConfigLoader struct {
	configPath string
}

// NewSlotGameService 创建游戏服务
func NewSlotGameService(configPath string) (*SlotGameService, error) {
	// 1. 加载配置
	configLoader := &ConfigLoader{configPath: configPath}
	algorithmConfig, err := configLoader.LoadAlgorithmConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load algorithm config: %w", err)
	}
	
	// 2. 创建组合引擎
	engine := NewCompositeSlotEngine(algorithmConfig, "classic")
	
	// 3. 加载主题配置
	if err := configLoader.LoadThemes(engine); err != nil {
		// 主题加载失败不影响核心功能，只记录警告
		fmt.Printf("Warning: failed to load themes: %v\n", err)
	}
	
	return &SlotGameService{
		engine:       engine,
		configLoader: configLoader,
	}, nil
}

// LoadAlgorithmConfig 加载算法配置 - 从现有SlotConfig转换
func (cl *ConfigLoader) LoadAlgorithmConfig() (*AlgorithmConfig, error) {
	// 加载现有的SlotConfig
	slotConfig := GetDefaultConfig() // 使用现有的配置函数
	
	// 转换为新的AlgorithmConfig格式
	algorithmConfig := &AlgorithmConfig{
		ReelCount:   slotConfig.Reels,
		RowCount:    slotConfig.Rows,
		SymbolCount: getSymbolCount(slotConfig),
		TargetRTP:   slotConfig.TargetRTP,
		MinRTP:      slotConfig.TargetRTP - 0.05,  // 允许5%的RTP波动
		MaxRTP:      slotConfig.TargetRTP + 0.05,
		
		// 转换符号权重
		SymbolWeights: extractSymbolWeights(slotConfig),
		
		// 转换赔付表
		PayTable: convertPayTable(slotConfig.PayTables),
		
		// 特殊符号配置
		WildSymbols:    []int{9},  // 假设符号9是百搭
		ScatterSymbols: []int{8},  // 假设符号8是散布
		BonusSymbols:   []int{7},  // 假设符号7是奖励
		
		// 算法设置
		Algorithm:    AlgorithmTypeClassic,
		Volatility:   0.5,  // 中等波动率
		HitFrequency: 0.25, // 25%命中率
		
		// 特性配置
		FeatureConfigs: make(map[AbstractFeatureType]*AbstractFeatureConfig),
	}
	
	// 配置免费旋转特性
	algorithmConfig.FeatureConfigs[AbstractFeatureTypeFreeSpin] = &AbstractFeatureConfig{
		TriggerSymbols: []int{8}, // 散布符号
		MinCount:       3,
		Probability:    0.05, // 5%概率
		Properties: map[string]interface{}{
			"free_spins": 10,
			"multiplier": 2.0,
		},
	}
	
	return algorithmConfig, nil
}

// getSymbolCount 获取符号数量
func getSymbolCount(config *SlotConfig) int {
	if len(config.ReelStrips) == 0 {
		return 10 // 默认符号数量
	}
	// 从第一个卷轴条获取符号数量
	return len(config.ReelStrips[0].Symbols)
}

// extractSymbolWeights 从SlotConfig提取符号权重
func extractSymbolWeights(config *SlotConfig) [][]int {
	if len(config.ReelStrips) == 0 {
		// 返回默认权重
		defaultWeights := []int{10, 20, 25, 15, 10, 8, 6, 4, 1, 1}
		weights := make([][]int, config.Reels)
		for i := range weights {
			weights[i] = defaultWeights
		}
		return weights
	}
	
	// 从ReelStrips提取权重
	weights := make([][]int, len(config.ReelStrips))
	for i, strip := range config.ReelStrips {
		if len(strip.Weights) > 0 {
			weights[i] = strip.Weights
		} else {
			// 如果没有权重，使用均等权重
			uniformWeight := make([]int, len(strip.Symbols))
			for j := range uniformWeight {
				uniformWeight[j] = 1
			}
			weights[i] = uniformWeight
		}
	}
	return weights
}

// convertSymbolWeights 转换符号权重格式（保留用于向后兼容）
func convertSymbolWeights(oldWeights map[int][]int) [][]int {
	if len(oldWeights) == 0 {
		// 提供默认权重配置
		return [][]int{
			{10, 20, 25, 15, 10, 8, 6, 4, 1, 1}, // 轮1权重
			{10, 20, 25, 15, 10, 8, 6, 4, 1, 1}, // 轮2权重
			{10, 20, 25, 15, 10, 8, 6, 4, 1, 1}, // 轮3权重
			{10, 20, 25, 15, 10, 8, 6, 4, 1, 1}, // 轮4权重
			{10, 20, 25, 15, 10, 8, 6, 4, 1, 1}, // 轮5权重
		}
	}
	
	// 转换现有格式
	maxReel := 0
	for reelIndex := range oldWeights {
		if reelIndex > maxReel {
			maxReel = reelIndex
		}
	}
	
	result := make([][]int, maxReel+1)
	for reelIndex := 0; reelIndex <= maxReel; reelIndex++ {
		if weights, exists := oldWeights[reelIndex]; exists {
			result[reelIndex] = weights
		} else {
			// 使用默认权重
			result[reelIndex] = []int{10, 20, 25, 15, 10, 8, 6, 4, 1, 1}
		}
	}
	
	return result
}

// convertPayTable 转换赔付表格式
func convertPayTable(payTables []PayTable) map[int][]int64 {
	result := make(map[int][]int64)
	
	if len(payTables) == 0 {
		// 提供默认赔付表
		for i := 0; i < 10; i++ {
			result[i] = getDefaultPayouts(i)
		}
		return result
	}
	
	// 从PayTables转换为新格式
	symbolPayouts := make(map[Symbol]map[int]float64)
	
	// 将PayTable数组转换为map结构便于处理
	for _, payTable := range payTables {
		if symbolPayouts[payTable.Symbol] == nil {
			symbolPayouts[payTable.Symbol] = make(map[int]float64)
		}
		symbolPayouts[payTable.Symbol][payTable.Count] = payTable.Multiplier
	}
	
	// 将符号映射为数字ID并创建赔付数组
	symbolToID := map[Symbol]int{
		SymbolCherry:     0, SymbolLemon:      1, SymbolOrange:     2,
		SymbolPlum:       3, SymbolGrape:      4, SymbolWatermelon: 5,
		SymbolBar:        6, SymbolBonus:      7, SymbolScatter:    8,
		SymbolSeven:      9, SymbolWild:       9, // Wild和Seven共用ID
	}
	
	for symbol, payouts := range symbolPayouts {
		if symbolID, exists := symbolToID[symbol]; exists {
			maxCount := 0
			for count := range payouts {
				if count > maxCount {
					maxCount = count
				}
			}
			
			payoutArray := make([]int64, maxCount)
			for count, multiplier := range payouts {
				if count > 0 && count <= maxCount {
					payoutArray[count-1] = int64(multiplier * 100) // 转换为固定奖金
				}
			}
			result[symbolID] = payoutArray
		}
	}
	
	// 填充缺失的符号赔付
	for i := 0; i < 10; i++ {
		if _, exists := result[i]; !exists {
			result[i] = getDefaultPayouts(i)
		}
	}
	
	return result
}

// getDefaultPayouts 获取默认赔付配置
func getDefaultPayouts(symbolID int) []int64 {
	// 根据符号稀有度提供不同的赔付
	switch symbolID {
	case 9: // 百搭符号
		return []int64{0, 0, 50, 200, 1000}
	case 8: // 散布符号
		return []int64{0, 0, 25, 100, 500}
	case 7: // 奖励符号
		return []int64{0, 0, 20, 80, 400}
	case 6: // 高价值符号
		return []int64{0, 0, 15, 60, 300}
	case 5:
		return []int64{0, 0, 10, 40, 200}
	case 4:
		return []int64{0, 0, 8, 30, 150}
	case 3:
		return []int64{0, 0, 6, 20, 100}
	case 2:
		return []int64{0, 0, 4, 15, 75}
	case 1:
		return []int64{0, 0, 3, 10, 50}
	default: // 低价值符号
		return []int64{0, 0, 2, 5, 25}
	}
}

// LoadThemes 加载主题配置
func (cl *ConfigLoader) LoadThemes(engine *CompositeSlotEngine) error {
	// 这里可以从文件系统、数据库或配置文件加载主题
	// 目前使用内置主题，已在ThemeRenderer中注册
	
	// 可以扩展为从外部文件加载：
	// themeFiles := []string{"themes/classic.json", "themes/fruit.json", "themes/egyptian.json"}
	// for _, file := range themeFiles {
	//     data, err := ioutil.ReadFile(file)
	//     if err != nil {
	//         continue
	//     }
	//     engine.LoadThemeFromJSON(data)
	// }
	
	return nil
}

// APISpinResult API响应结果 - 兼容现有API格式
type APISpinResult struct {
	// 基础信息
	Success     bool   `json:"success"`
	Message     string `json:"message,omitempty"`
	ResultID    string `json:"result_id"`
	
	// 游戏结果
	BetAmount   int64  `json:"bet_amount"`
	TotalWin    int64  `json:"total_win"`
	NetWin      int64  `json:"net_win"`
	Balance     int64  `json:"balance"`
	
	// 转轴结果
	Reels       [][]interface{} `json:"reels"`        // 兼容格式：可以是数字或符号对象
	
	// 获胜信息
	IsWin       bool            `json:"is_win"`
	WinLines    []WinLineResult `json:"win_lines"`
	Multiplier  float64         `json:"multiplier"`
	WinType     string          `json:"win_type"`
	
	// 特殊功能
	Features    []FeatureResult `json:"features,omitempty"`
	FreeSpins   int             `json:"free_spins,omitempty"`
	
	// 统计信息
	Statistics  map[string]interface{} `json:"statistics,omitempty"`
	
	// 主题信息（新增）
	ThemeID     string          `json:"theme_id,omitempty"`
	Animations  []interface{}   `json:"animations,omitempty"`
	SoundEffects []interface{}  `json:"sound_effects,omitempty"`
	
	// 性能信息
	ProcessingTime int64        `json:"processing_time"`
}

// WinLineResult 获胜线结果
type WinLineResult struct {
	LineID      int         `json:"line_id"`
	Symbol      interface{} `json:"symbol"`        // 可以是数字ID或符号对象
	Count       int         `json:"count"`
	Payout      int64       `json:"payout"`
	Positions   []Position  `json:"positions"`
	Multiplier  float64     `json:"multiplier"`
}

// FeatureResult 特性结果
type FeatureResult struct {
	Type        string      `json:"type"`
	Value       int64       `json:"value"`
	Description string      `json:"description"`
	Positions   []Position  `json:"positions"`
}

// Spin API端点实现
func (sgs *SlotGameService) Spin(ctx context.Context, userID uint, sessionID string, betAmount int64, themeID string) (*SpinResult, error) {
	// 1. 构建请求
	request := &SpinRequest{
		GameRequest: &GameRequest{
			SessionID: sessionID,
			BetAmount: betAmount,
			SpinCount: 1,
			Metadata:  map[string]interface{}{
				"user_id": userID,
				"include_abstract": false, // 不返回抽象结果以节省带宽
			},
		},
		ThemeID:     themeID,
		EnableTheme: themeID != "" && themeID != "abstract",
	}
	
	// 2. 执行旋转
	response, err := sgs.engine.Spin(ctx, request)
	if err != nil {
		// 返回错误格式的SpinResult
		return &SpinResult{
			ID:        "error_" + sessionID,
			SessionID: sessionID,
			UserID:    userID,
			BetAmount: betAmount,
			WinAmount: 0,
			Timestamp: time.Now(),
		}, err
	}
	
	// 3. 转换为API格式
	result := sgs.convertToAPIResult(response, userID)
	
	return result, nil
}

// convertToAPIResult 转换为API结果格式
func (sgs *SlotGameService) convertToAPIResult(response *SpinResponse, userID uint) *SpinResult {
	// 创建SpinResult基础结构
	result := &SpinResult{
		UserID:    userID,
		Timestamp: time.Now(),
	}
	
	// 使用主题化结果或抽象结果
	var abstractResult *AbstractGameResult
	if response.ThemedResult != nil {
		abstractResult = response.ThemedResult.AbstractGameResult
		// SpinResult没有ThemeID, Animations, SoundEffects字段，只设置基础字段
		result.Reels = sgs.convertThemedReelsToSymbols(response.ThemedResult.ReelSymbols)
	} else if response.AbstractResult != nil {
		abstractResult = response.AbstractResult
		result.Reels = sgs.convertAbstractReelsToSymbols(abstractResult.ReelResults)
	} else {
		return &SpinResult{
			ID:        "error_no_result",
			UserID:    userID,
			WinAmount: 0,
			Timestamp: time.Now(),
		}
	}
	
	// 填充基础信息
	result.ID = abstractResult.ResultID
	result.BetAmount = abstractResult.BetAmount
	result.WinAmount = abstractResult.TotalWin
	result.TotalPayout = abstractResult.TotalWin
	result.IsJackpot = abstractResult.WinType == WinTypeJackpot
	result.Multiplier = abstractResult.Multiplier
	
	// 转换获胜线
	result.WinLines = sgs.convertAbstractWinLinesToWinLines(abstractResult.WinLines)
	
	// 转换特性
	result.Features = sgs.convertAbstractFeaturesToFeatures(abstractResult.Features)
	
	// 计算免费旋转
	for _, feature := range abstractResult.Features {
		if feature.Type == AbstractFeatureTypeFreeSpin {
			result.FreeSpins = feature.Duration
		}
	}
	
	// 计算RTP（SpinResult没有Statistics字段，使用RTP字段）
	if abstractResult.Statistics != nil {
		result.RTP = abstractResult.Statistics.CurrentRTP
	}
	
	return result
}

// 转换辅助方法
func (sgs *SlotGameService) convertThemedReels(reelSymbols [][]Symbol) [][]interface{} {
	result := make([][]interface{}, len(reelSymbols))
	for i, reel := range reelSymbols {
		result[i] = make([]interface{}, len(reel))
		for j, symbol := range reel {
			result[i][j] = map[string]interface{}{
				"symbol": string(symbol), // Symbol是字符串类型，直接转换
			}
		}
	}
	return result
}

func (sgs *SlotGameService) convertAbstractReels(reelResults [][]int) [][]interface{} {
	result := make([][]interface{}, len(reelResults))
	for i, reel := range reelResults {
		result[i] = make([]interface{}, len(reel))
		for j, symbolID := range reel {
			result[i][j] = symbolID
		}
	}
	return result
}

func (sgs *SlotGameService) convertWinLines(winLines []AbstractWinLine) []WinLineResult {
	result := make([]WinLineResult, len(winLines))
	for i, line := range winLines {
		// 转换GamePosition为Position
		positions := make([]Position, len(line.Positions))
		for j, pos := range line.Positions {
			positions[j] = Position{Reel: pos.Reel, Row: pos.Row}
		}
		
		result[i] = WinLineResult{
			LineID:     line.LineID,
			Symbol:     line.SymbolID,
			Count:      line.Count,
			Payout:     line.Payout,
			Positions:  positions,
			Multiplier: line.Multiplier,
		}
	}
	return result
}

func (sgs *SlotGameService) convertFeatures(features []AbstractFeature) []FeatureResult {
	result := make([]FeatureResult, len(features))
	for i, feature := range features {
		// 转换GamePosition为Position
		positions := make([]Position, len(feature.Trigger))
		for j, pos := range feature.Trigger {
			positions[j] = Position{Reel: pos.Reel, Row: pos.Row}
		}
		
		result[i] = FeatureResult{
			Type:        sgs.convertAbstractFeatureTypeToString(feature.Type),
			Value:       feature.Value,
			Description: sgs.getAbstractFeatureDescription(feature.Type),
			Positions:   positions,
		}
	}
	return result
}

func (sgs *SlotGameService) convertWinType(winType WinType) string {
	switch winType {
	case WinTypeJackpot:
		return "jackpot"
	case WinTypeBig:
		return "big_win"
	case WinTypeMedium:
		return "medium_win"
	case WinTypeSmall:
		return "small_win"
	default:
		return "no_win"
	}
}

func (sgs *SlotGameService) convertFeatureType(featureType FeatureType) string {
	switch featureType {
	case FeatureTypeFreeSpins: // 使用正确的常量名
		return "free_spin"
	case FeatureTypeBonus:
		return "bonus"
	case FeatureTypeExpandWild: // 使用正确的常量名
		return "wild"
	case FeatureTypeMultiplier:
		return "multiplier"
	case FeatureTypeRespin: // 使用正确的常量名
		return "respins"
	default:
		return "unknown"
	}
}

// convertAbstractFeatureTypeToString 转换抽象特性类型为字符串
func (sgs *SlotGameService) convertAbstractFeatureTypeToString(featureType AbstractFeatureType) string {
	switch featureType {
	case AbstractFeatureTypeFreeSpin:
		return "free_spin"
	case AbstractFeatureTypeBonus:
		return "bonus"
	case AbstractFeatureTypeWild:
		return "wild"
	case AbstractFeatureTypeMultiplier:
		return "multiplier"
	default:
		return "unknown"
	}
}

func (sgs *SlotGameService) getFeatureDescription(featureType FeatureType) string {
	switch featureType {
	case FeatureTypeFreeSpins: // 使用正确的常量名
		return "免费旋转触发"
	case FeatureTypeBonus:
		return "奖励游戏触发"
	case FeatureTypeExpandWild: // 使用正确的常量名
		return "百搭符号扩展"
	case FeatureTypeMultiplier:
		return "倍数奖励"
	case FeatureTypeRespin: // 使用正确的常量名
		return "重转奖励"
	default:
		return "特殊功能"
	}
}

func (sgs *SlotGameService) convertAnimations(animations []Animation) []interface{} {
	result := make([]interface{}, len(animations))
	for i, anim := range animations {
		result[i] = map[string]interface{}{
			"type":     anim.Type,
			"target":   anim.Target,
			"duration": anim.Duration,
			"sequence": anim.Sequence,
		}
	}
	return result
}

func (sgs *SlotGameService) convertSoundEffects(sounds []SoundEffect) []interface{} {
	result := make([]interface{}, len(sounds))
	for i, sound := range sounds {
		result[i] = map[string]interface{}{
			"type":     sound.Type,
			"file_url": sound.FileURL,
			"volume":   sound.Volume,
			"loop":     sound.Loop,
			"delay":    sound.Delay,
		}
	}
	return result
}

// 批量旋转API
func (sgs *SlotGameService) BatchSpin(ctx context.Context, userID uint, sessionID string, betAmount int64, count int) (*BatchSpinResponse, error) {
	request := &SpinRequest{
		GameRequest: &GameRequest{
			SessionID: sessionID,
			BetAmount: betAmount,
			SpinCount: count,
			Metadata: map[string]interface{}{
				"user_id": userID,
			},
		},
		ThemeID:     "abstract", // 批量旋转通常不需要主题
		EnableTheme: false,
	}
	
	return sgs.engine.BatchSpin(ctx, request, count)
}

// RTP控制API
func (sgs *SlotGameService) SetTargetRTP(rtp float64) error {
	return sgs.engine.SetTargetRTP(rtp)
}

func (sgs *SlotGameService) GetCurrentRTP() float64 {
	return sgs.engine.GetCurrentRTP()
}

func (sgs *SlotGameService) GetStatistics() *AbstractStatistics {
	return sgs.engine.GetStatistics()
}

// 主题管理API
func (sgs *SlotGameService) GetAvailableThemes() []string {
	return sgs.engine.GetAvailableThemes()
}

func (sgs *SlotGameService) SetDefaultTheme(themeID string) error {
	return sgs.engine.SetDefaultTheme(themeID)
}

// 引擎管理API
func (sgs *SlotGameService) Start() error {
	return sgs.engine.Start()
}

func (sgs *SlotGameService) Stop() error {
	return sgs.engine.Stop()
}

func (sgs *SlotGameService) IsRunning() bool {
	return sgs.engine.IsRunning()
}

// 新增的正确类型转换函数

// convertThemedReelsToSymbols 转换主题化卷轴为Symbol数组
func (sgs *SlotGameService) convertThemedReelsToSymbols(reelSymbols [][]ThemeSymbol) [][]Symbol {
	result := make([][]Symbol, len(reelSymbols))
	for i, reel := range reelSymbols {
		result[i] = make([]Symbol, len(reel))
		for j, themeSymbol := range reel {
			// 将ThemeSymbol的ID映射为标准Symbol常量
			result[i][j] = sgs.symbolIDToSymbol(themeSymbol.ID)
		}
	}
	return result
}

// convertAbstractReelsToSymbols 转换抽象卷轴为Symbol数组
func (sgs *SlotGameService) convertAbstractReelsToSymbols(reelResults [][]int) [][]Symbol {
	result := make([][]Symbol, len(reelResults))
	for i, reel := range reelResults {
		result[i] = make([]Symbol, len(reel))
		for j, symbolID := range reel {
			// 将符号ID映射为标准Symbol常量
			result[i][j] = sgs.symbolIDToSymbol(symbolID)
		}
	}
	return result
}

// convertAbstractWinLinesToWinLines 转换抽象获胜线为WinLine数组
func (sgs *SlotGameService) convertAbstractWinLinesToWinLines(abstractLines []AbstractWinLine) []WinLine {
	result := make([]WinLine, len(abstractLines))
	for i, line := range abstractLines {
		// 转换Position数组
		positions := make([]Position, len(line.Positions))
		for j, pos := range line.Positions {
			positions[j] = Position{
				Reel: pos.Reel,
				Row:  pos.Row,
			}
		}

		result[i] = WinLine{
			LineID:     line.LineID,
			LineType:   sgs.abstractLineTypeToLineType(line.LineType),
			Symbol:     sgs.symbolIDToSymbol(line.SymbolID),
			Count:      line.Count,
			Positions:  positions,
			WinAmount:  line.Payout,
			Multiplier: line.Multiplier,
		}
	}
	return result
}

// convertAbstractFeaturesToFeatures 转换抽象特性为Feature数组
func (sgs *SlotGameService) convertAbstractFeaturesToFeatures(abstractFeatures []AbstractFeature) []Feature {
	result := make([]Feature, len(abstractFeatures))
	for i, feature := range abstractFeatures {
		// 转换触发位置
		positions := make([]Position, len(feature.Trigger))
		for j, pos := range feature.Trigger {
			positions[j] = Position{
				Reel: pos.Reel,
				Row:  pos.Row,
			}
		}

		result[i] = Feature{
			Type:        sgs.abstractFeatureTypeToFeatureType(feature.Type),
			TriggerPos:  positions,
			Value:       feature.Value,
			Description: sgs.getAbstractFeatureDescription(feature.Type),
		}
	}
	return result
}

// 辅助转换函数

// symbolIDToSymbol 将符号ID转换为Symbol常量
func (sgs *SlotGameService) symbolIDToSymbol(id int) Symbol {
	symbols := []Symbol{
		SymbolCherry, SymbolLemon, SymbolOrange, SymbolPlum,
		SymbolGrape, SymbolWatermelon, SymbolBar, SymbolBonus,
		SymbolScatter, SymbolSeven, // 假设9是Seven，也可以是Wild
	}
	if id >= 0 && id < len(symbols) {
		return symbols[id]
	}
	return SymbolCherry // 默认符号
}

// abstractLineTypeToLineType 转换抽象线型为LineType
func (sgs *SlotGameService) abstractLineTypeToLineType(abstractType AbstractLineType) LineType {
	switch abstractType {
	case AbstractLineTypeHorizontal:
		return LineTypeHorizontal
	case AbstractLineTypeDiagonal:
		return LineTypeDiagonal
	default:
		return LineTypeHorizontal
	}
}

// abstractFeatureTypeToFeatureType 转换抽象特性类型为FeatureType
func (sgs *SlotGameService) abstractFeatureTypeToFeatureType(abstractType AbstractFeatureType) FeatureType {
	switch abstractType {
	case AbstractFeatureTypeFreeSpin:
		return FeatureTypeFreeSpins
	case AbstractFeatureTypeBonus:
		return FeatureTypeBonus
	case AbstractFeatureTypeMultiplier:
		return FeatureTypeMultiplier
	case AbstractFeatureTypeWild:
		return FeatureTypeExpandWild
	default:
		return FeatureTypeFreeSpins
	}
}

// getAbstractFeatureDescription 获取抽象特性描述
func (sgs *SlotGameService) getAbstractFeatureDescription(featureType AbstractFeatureType) string {
	switch featureType {
	case AbstractFeatureTypeFreeSpin:
		return "免费旋转触发"
	case AbstractFeatureTypeBonus:
		return "奖励游戏触发" 
	case AbstractFeatureTypeWild:
		return "百搭符号扩展"
	case AbstractFeatureTypeMultiplier:
		return "倍数奖励"
	default:
		return "特殊功能"
	}
}