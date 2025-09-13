package hardware

import (
	"sync"
	"time"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// GameMode 游戏模式
type GameMode byte

const (
	ModeCoinRefund GameMode = 0x01 // 退币模式
	ModeTicket     GameMode = 0x02 // 彩票模式
)

// DifficultyLevel 难度级别
type DifficultyLevel byte

const (
	DifficultyEasy   DifficultyLevel = 0x01
	DifficultyNormal DifficultyLevel = 0x02
	DifficultyHard   DifficultyLevel = 0x03
)

// GameLogicAdapter 游戏逻辑适配器
// 实现GameLogicInterface接口，连接游戏系统和硬件控制
type GameLogicAdapter struct {
	mu              sync.RWMutex
	logger          *zap.Logger
	
	// 游戏状态
	currentMode     GameMode          // 当前模式（退币/彩票）
	difficulty      DifficultyLevel   // 游戏难度
	credits         int64             // 游戏币余额
	tickets         int64             // 彩票余额
	playerCoins     int64             // 玩家币数
	pendingCoins    uint16            // 待上币数量
	returnRate      float64           // 回币率
	
	// 游戏配置
	minBet          int64             // 最小下注
	maxBet          int64             // 最大下注
	ticketRatio     int64             // 彩票兑换比例
	
	// 统计数据
	totalBets       int64             // 总下注
	totalWins       int64             // 总赢取
	gameCount       int64             // 游戏次数
	lastGameTime    time.Time         // 最后游戏时间
	
	// 回调函数
	onGameStart     func(coins uint16)
	onGameEnd       func(result *GameResult)
	onBalanceChange func(credits int64)
}

// GameResult 游戏结果
type GameResult struct {
	Won          bool
	WinAmount    int64
	ReturnRate   float64
	Timestamp    time.Time
}

// NewGameLogicAdapter 创建游戏逻辑适配器
func NewGameLogicAdapter() *GameLogicAdapter {
	return &GameLogicAdapter{
		logger:       logger.GetLogger(),
		currentMode:  ModeCoinRefund,
		difficulty:   DifficultyNormal,
		minBet:       1,
		maxBet:       100,
		ticketRatio:  10, // 10个币换1张彩票
	}
}

// GetCurrentMode 获取当前模式
func (g *GameLogicAdapter) GetCurrentMode() byte {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return byte(g.currentMode)
}

// SetCurrentMode 设置当前模式
func (g *GameLogicAdapter) SetCurrentMode(mode GameMode) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.currentMode = mode
	g.logger.Info("游戏模式切换", zap.Uint8("mode", uint8(mode)))
}

// HasCredits 是否有余额
func (g *GameLogicAdapter) HasCredits() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.credits > 0
}

// GetPendingCoins 获取待上币数量
func (g *GameLogicAdapter) GetPendingCoins() uint16 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pendingCoins
}

// AddCredits 增加余额（投币）
func (g *GameLogicAdapter) AddCredits(count byte) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.credits += int64(count)
	g.pendingCoins += uint16(count)
	
	g.logger.Info("增加游戏币", 
		zap.Uint8("count", count),
		zap.Int64("balance", g.credits))
	
	if g.onBalanceChange != nil {
		g.onBalanceChange(g.credits)
	}
}

// AddPlayerCoins 增加玩家币数（回币）
func (g *GameLogicAdapter) AddPlayerCoins(count byte) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.playerCoins += int64(count)
	g.totalWins += int64(count)
	
	g.logger.Info("玩家获得币", 
		zap.Uint8("count", count),
		zap.Int64("total", g.playerCoins))
}

// UpdateReturnRate 更新回币率
func (g *GameLogicAdapter) UpdateReturnRate(rate float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.returnRate = rate
	
	// 根据回币率自动调整难度
	switch {
	case rate < 30:
		g.difficulty = DifficultyEasy
		g.logger.Info("回币率过低，调整为简单模式", zap.Float64("rate", rate))
	case rate > 70:
		g.difficulty = DifficultyHard
		g.logger.Info("回币率过高，调整为困难模式", zap.Float64("rate", rate))
	default:
		g.difficulty = DifficultyNormal
	}
}

// GetRefundableCoins 获取可退币数
func (g *GameLogicAdapter) GetRefundableCoins() uint16 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	// 可退币数 = 余额 + 玩家获得的币
	total := g.credits + g.playerCoins
	if total > 9999 {
		return 9999 // 最大退币数限制
	}
	return uint16(total)
}

// GetAvailableTickets 获取可用彩票数
func (g *GameLogicAdapter) GetAvailableTickets() uint16 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	// 根据玩家币数计算可兑换彩票数
	if g.playerCoins < g.ticketRatio {
		return 0
	}
	
	tickets := g.playerCoins / g.ticketRatio
	if tickets > 9999 {
		return 9999 // 最大彩票数限制
	}
	return uint16(tickets)
}

// DeductCoins 扣除币数
func (g *GameLogicAdapter) DeductCoins(count uint16) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	deduct := int64(count)
	
	// 优先从余额扣除
	if g.credits >= deduct {
		g.credits -= deduct
	} else {
		// 余额不足，从玩家币扣除
		remaining := deduct - g.credits
		g.credits = 0
		g.playerCoins -= remaining
		if g.playerCoins < 0 {
			g.playerCoins = 0
		}
	}
	
	g.logger.Info("扣除币数", 
		zap.Uint16("count", count),
		zap.Int64("credits", g.credits),
		zap.Int64("playerCoins", g.playerCoins))
	
	if g.onBalanceChange != nil {
		g.onBalanceChange(g.credits)
	}
}

// RedeemTickets 兑换彩票
func (g *GameLogicAdapter) RedeemTickets(count uint16) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// 扣除对应的玩家币
	cost := int64(count) * g.ticketRatio
	g.playerCoins -= cost
	if g.playerCoins < 0 {
		g.playerCoins = 0
	}
	
	g.tickets += int64(count)
	
	g.logger.Info("兑换彩票", 
		zap.Uint16("count", count),
		zap.Int64("cost", cost),
		zap.Int64("playerCoins", g.playerCoins))
}

// StartGame 开始游戏
func (g *GameLogicAdapter) StartGame(coinCount uint16) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.pendingCoins = 0 // 清空待上币
	g.totalBets += int64(coinCount)
	g.gameCount++
	g.lastGameTime = time.Now()
	
	g.logger.Info("游戏开始", 
		zap.Uint16("coins", coinCount),
		zap.Int64("gameCount", g.gameCount))
	
	if g.onGameStart != nil {
		g.onGameStart(coinCount)
	}
}

// SetDifficulty 设置难度
func (g *GameLogicAdapter) SetDifficulty(level byte) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.difficulty = DifficultyLevel(level)
	g.logger.Info("难度设置", zap.Uint8("level", level))
}

// GetStatistics 获取统计数据
func (g *GameLogicAdapter) GetStatistics() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	rtp := float64(0)
	if g.totalBets > 0 {
		rtp = float64(g.totalWins) / float64(g.totalBets) * 100
	}
	
	return map[string]interface{}{
		"credits":      g.credits,
		"playerCoins":  g.playerCoins,
		"tickets":      g.tickets,
		"totalBets":    g.totalBets,
		"totalWins":    g.totalWins,
		"gameCount":    g.gameCount,
		"returnRate":   g.returnRate,
		"rtp":          rtp,
		"currentMode":  g.currentMode,
		"difficulty":   g.difficulty,
		"lastGameTime": g.lastGameTime,
	}
}

// SetCallbacks 设置回调函数
func (g *GameLogicAdapter) SetCallbacks(
	onGameStart func(coins uint16),
	onGameEnd func(result *GameResult),
	onBalanceChange func(credits int64)) {
	
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.onGameStart = onGameStart
	g.onGameEnd = onGameEnd
	g.onBalanceChange = onBalanceChange
}

// Reset 重置游戏状态
func (g *GameLogicAdapter) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.credits = 0
	g.playerCoins = 0
	g.tickets = 0
	g.pendingCoins = 0
	g.returnRate = 0
	
	g.logger.Info("游戏状态重置")
}