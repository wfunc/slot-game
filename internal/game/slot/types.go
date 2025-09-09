package slot

import (
	"fmt"
	"time"
)

// Symbol 游戏符号
type Symbol string

const (
	SymbolCherry     Symbol = "CHERRY"     // 樱桃
	SymbolLemon      Symbol = "LEMON"      // 柠檬
	SymbolOrange     Symbol = "ORANGE"     // 橙子
	SymbolPlum       Symbol = "PLUM"       // 李子
	SymbolGrape      Symbol = "GRAPE"      // 葡萄
	SymbolWatermelon Symbol = "WATERMELON" // 西瓜
	SymbolBar        Symbol = "BAR"        // BAR
	SymbolSeven      Symbol = "SEVEN"      // 7
	SymbolWild       Symbol = "WILD"       // 百搭
	SymbolScatter    Symbol = "SCATTER"    // 分散
	SymbolBonus      Symbol = "BONUS"      // 奖励
)

// LineType 连线类型
type LineType int

const (
	LineTypeHorizontal LineType = iota // 水平线
	LineTypeDiagonal                   // 对角线
	LineTypeV                           // V型
	LineTypeZigzag                      // 之字形
)

// SpinResult 旋转结果
type SpinResult struct {
	ID           string       `json:"id"`            // 结果ID
	SessionID    string       `json:"session_id"`    // 会话ID
	UserID       uint         `json:"user_id"`       // 用户ID
	BetAmount    int64        `json:"bet_amount"`    // 下注金额
	WinAmount    int64        `json:"win_amount"`    // 中奖金额
	TotalPayout  int64        `json:"total_payout"`  // 总赔付金额
	Multiplier   float64      `json:"multiplier"`    // 倍率
	Reels        [][]Symbol   `json:"reels"`         // 卷轴结果
	WinLines     []WinLine    `json:"win_lines"`     // 中奖线
	Features     []Feature    `json:"features"`      // 特殊功能
	FreeSpins    int          `json:"free_spins"`    // 免费旋转次数
	IsJackpot    bool         `json:"is_jackpot"`    // 是否中大奖
	RTP          float64      `json:"rtp"`           // 实际RTP
	Timestamp    time.Time    `json:"timestamp"`     // 时间戳
}

// GetTotalPayout 获取总赔付（兼容性方法）
func (s *SpinResult) GetTotalPayout() int64 {
	if s.TotalPayout > 0 {
		return s.TotalPayout
	}
	return s.WinAmount
}

// GetWinDescription 获取中奖描述
func (s *SpinResult) GetWinDescription() string {
	if s.IsJackpot {
		return "Jackpot!"
	}
	if len(s.WinLines) > 0 {
		return fmt.Sprintf("%d条中奖线", len(s.WinLines))
	}
	return "未中奖"
}

// ToJSON 转换为JSON map
func (s *SpinResult) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"id":           s.ID,
		"session_id":   s.SessionID,
		"user_id":      s.UserID,
		"bet_amount":   s.BetAmount,
		"win_amount":   s.WinAmount,
		"total_payout": s.GetTotalPayout(),
		"multiplier":   s.Multiplier,
		"reels":        s.Reels,
		"win_lines":    s.WinLines,
		"features":     s.Features,
		"free_spins":   s.FreeSpins,
		"is_jackpot":   s.IsJackpot,
		"rtp":          s.RTP,
		"timestamp":    s.Timestamp,
	}
}

// WinLine 中奖线
type WinLine struct {
	LineID      int        `json:"line_id"`      // 线ID
	LineType    LineType   `json:"line_type"`    // 线类型
	Symbol      Symbol     `json:"symbol"`       // 中奖符号
	Count       int        `json:"count"`        // 连续个数
	Positions   []Position `json:"positions"`    // 位置
	WinAmount   int64      `json:"win_amount"`   // 中奖金额
	Multiplier  float64    `json:"multiplier"`   // 倍率
}

// Position 符号位置
type Position struct {
	Reel int `json:"reel"` // 卷轴索引 (0-based)
	Row  int `json:"row"`  // 行索引 (0-based)
}

// Feature 特殊功能
type Feature struct {
	Type        FeatureType `json:"type"`        // 功能类型
	TriggerPos  []Position  `json:"trigger_pos"` // 触发位置
	Value       interface{} `json:"value"`       // 功能值
	Description string      `json:"description"` // 描述
}

// FeatureType 特殊功能类型
type FeatureType string

const (
	FeatureTypeFreeSpins   FeatureType = "FREE_SPINS"   // 免费旋转
	FeatureTypeBonus       FeatureType = "BONUS"        // 奖励游戏
	FeatureTypeMultiplier  FeatureType = "MULTIPLIER"   // 倍率增加
	FeatureTypeExpandWild  FeatureType = "EXPAND_WILD"  // 扩展百搭
	FeatureTypeRespin      FeatureType = "RESPIN"       // 重新旋转
	FeatureTypeCascade     FeatureType = "CASCADE"      // 消除
)

// PayTable 赔率表
type PayTable struct {
	Symbol     Symbol  `json:"symbol"`      // 符号
	Count      int     `json:"count"`       // 连续个数
	Multiplier float64 `json:"multiplier"`  // 赔率倍数
}

// ReelStrip 卷轴条
type ReelStrip struct {
	ReelID  int      `json:"reel_id"`  // 卷轴ID
	Symbols []Symbol `json:"symbols"`  // 符号序列
	Weights []int    `json:"weights"`  // 权重
}

// SlotConfig 老虎机配置
type SlotConfig struct {
	MachineID       string      `json:"machine_id"`       // 机器ID
	Name            string      `json:"name"`             // 名称
	Rows            int         `json:"rows"`             // 行数
	Reels           int         `json:"reels"`            // 卷轴数
	PayLines        int         `json:"pay_lines"`        // 支付线数
	MinBet          int64       `json:"min_bet"`          // 最小下注
	MaxBet          int64       `json:"max_bet"`          // 最大下注
	DefaultBet      int64       `json:"default_bet"`      // 默认下注
	TargetRTP       float64     `json:"target_rtp"`       // 目标RTP
	Volatility      Volatility  `json:"volatility"`       // 波动性
	ReelStrips      []ReelStrip `json:"reel_strips"`     // 卷轴条配置
	PayTables       []PayTable  `json:"pay_tables"`       // 赔率表
	WildSymbols     []Symbol    `json:"wild_symbols"`     // 百搭符号
	ScatterSymbols  []Symbol    `json:"scatter_symbols"`  // 分散符号
	BonusSymbols    []Symbol    `json:"bonus_symbols"`    // 奖励符号
	Features        []FeatureConfig `json:"features"`     // 特殊功能配置
}

// FeatureConfig 特殊功能配置
type FeatureConfig struct {
	Type            FeatureType    `json:"type"`             // 功能类型
	TriggerSymbols  []Symbol       `json:"trigger_symbols"`  // 触发符号
	TriggerCount    int            `json:"trigger_count"`    // 触发数量
	Probability     float64        `json:"probability"`      // 触发概率
	Value           interface{}    `json:"value"`            // 功能值
}

// Volatility 波动性
type Volatility int

const (
	VolatilityLow    Volatility = iota // 低波动
	VolatilityMedium                   // 中波动
	VolatilityHigh                      // 高波动
)

// RTPController RTP控制器接口
type RTPController interface {
	// AdjustOdds 根据当前RTP调整赔率
	AdjustOdds(currentRTP, targetRTP float64) float64
	
	// ShouldTriggerWin 判断是否应该触发中奖
	ShouldTriggerWin(currentRTP, targetRTP float64, betAmount int64) bool
	
	// CalculateRTP 计算当前RTP
	CalculateRTP(totalWin, totalBet int64) float64
	
	// GetCompensationMultiplier 获取补偿倍率
	GetCompensationMultiplier(currentRTP, targetRTP float64) float64
}

// PatternMatcher 图案匹配器接口
type PatternMatcher interface {
	// FindWinningLines 查找中奖线
	FindWinningLines(reels [][]Symbol, config *SlotConfig) []WinLine
	
	// CalculatePayout 计算赔付
	CalculatePayout(winLines []WinLine, betAmount int64) int64
	
	// DetectFeatures 检测特殊功能
	DetectFeatures(reels [][]Symbol, config *SlotConfig) []Feature
}

// RandomGenerator 随机数生成器接口
type RandomGenerator interface {
	// Next 生成下一个随机数
	Next() float64
	
	// NextInt 生成指定范围内的随机整数
	NextInt(min, max int) int
	
	// Seed 设置种子
	Seed(seed int64)
}

// Engine 游戏引擎接口
type Engine interface {
	// Spin 执行旋转
	Spin(userID uint, sessionID string, betAmount int64) (*SpinResult, error)
	
	// GetConfig 获取配置
	GetConfig() *SlotConfig
	
	// SetRTPController 设置RTP控制器
	SetRTPController(controller RTPController)
	
	// GetStatistics 获取统计数据
	GetStatistics() *Statistics
	
	// Reset 重置引擎
	Reset()
}

// Statistics 统计数据
type Statistics struct {
	TotalSpins      int64     `json:"total_spins"`      // 总旋转次数
	TotalBet        int64     `json:"total_bet"`         // 总下注
	TotalWin        int64     `json:"total_win"`         // 总赢取
	CurrentRTP      float64   `json:"current_rtp"`       // 当前RTP
	BigWins         int       `json:"big_wins"`          // 大奖次数
	JackpotHits     int       `json:"jackpot_hits"`      // 头奖次数
	FreeSpinsTotal  int       `json:"free_spins_total"`  // 免费旋转总数
	LastUpdate      time.Time `json:"last_update"`       // 最后更新时间
}