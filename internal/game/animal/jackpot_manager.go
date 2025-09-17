package animal

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// JackpotConfig 彩金配置
type JackpotConfig struct {
	InitialPool    float64 // 初始奖池金额
	AccumulateRate float64 // 积累比例（从每次下注中抽取的比例）
	TriggerProb    float64 // 触发概率
	MinTrigger     float64 // 最小触发金额
	MaxTrigger     float64 // 最大触发金额
}

// JackpotManager 彩金管理器
type JackpotManager struct {
	mu           sync.RWMutex
	currentPool  float64         // 当前彩金池
	config       *JackpotConfig  // 彩金配置
	history      []*JackpotWin   // 中奖历史
	lastPushTime time.Time       // 上次推送时间
	logger       *zap.Logger
}

// JackpotWin 彩金中奖记录
type JackpotWin struct {
	PlayerID   uint32
	PlayerName string
	PlayerIcon string
	Amount     float64
	Time       time.Time
}

// NewJackpotManager 创建彩金管理器
func NewJackpotManager(logger *zap.Logger) *JackpotManager {
	return &JackpotManager{
		currentPool: 10000.0, // 初始奖池1万
		config: &JackpotConfig{
			InitialPool:    10000.0,
			AccumulateRate: 0.02,  // 2%进入奖池
			TriggerProb:    0.001, // 0.1%触发概率
			MinTrigger:     1000.0,
			MaxTrigger:     100000.0,
		},
		history:      make([]*JackpotWin, 0),
		lastPushTime: time.Now(),
		logger:       logger,
	}
}

// AccumulateFromBet 从下注中积累彩金
func (j *JackpotManager) AccumulateFromBet(betAmount uint32) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// 将下注金额的一定比例加入彩金池
	accumulate := float64(betAmount) * j.config.AccumulateRate
	j.currentPool += accumulate

	// 限制最大奖池
	if j.currentPool > j.config.MaxTrigger {
		j.currentPool = j.config.MaxTrigger
	}
}

// CheckJackpotTrigger 检查是否触发彩金
func (j *JackpotManager) CheckJackpotTrigger(playerID uint32, playerName, playerIcon string) (bool, float64) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// 检查是否满足最小触发金额
	if j.currentPool < j.config.MinTrigger {
		return false, 0
	}

	// 随机判断是否触发
	if rand.Float64() > j.config.TriggerProb {
		return false, 0
	}

	// 计算中奖金额（当前奖池的50%-100%）
	winRatio := 0.5 + rand.Float64()*0.5
	winAmount := j.currentPool * winRatio

	// 记录中奖
	win := &JackpotWin{
		PlayerID:   playerID,
		PlayerName: playerName,
		PlayerIcon: playerIcon,
		Amount:     winAmount,
		Time:       time.Now(),
	}
	j.history = append(j.history, win)

	// 重置奖池（保留剩余部分）
	j.currentPool = j.currentPool - winAmount + j.config.InitialPool*0.1

	j.logger.Info("[JackpotManager] 彩金触发",
		zap.Uint32("player_id", playerID),
		zap.String("player_name", playerName),
		zap.Float64("win_amount", winAmount),
		zap.Float64("remaining_pool", j.currentPool))

	return true, winAmount
}

// GetCurrentPool 获取当前奖池金额
func (j *JackpotManager) GetCurrentPool() float64 {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.currentPool
}

// GetJackpotHistory 获取彩金中奖历史
func (j *JackpotManager) GetJackpotHistory(limit int) []*pb.PCjLog {
	j.mu.RLock()
	defer j.mu.RUnlock()

	result := make([]*pb.PCjLog, 0)
	start := 0
	if len(j.history) > limit {
		start = len(j.history) - limit
	}

	for i := start; i < len(j.history); i++ {
		win := j.history[i]
		result = append(result, &pb.PCjLog{
			Id:    proto.Uint32(uint32(i + 1)),
			Icon:  proto.String(win.PlayerIcon),
			Name:  proto.String(win.PlayerName),
			Time:  proto.Uint32(uint32(win.Time.Unix())),
			Bonus: proto.String(formatMoney(win.Amount)),
		})
	}

	return result
}

// CreateJackpotPush 创建彩金推送消息 (1810)
func (j *JackpotManager) CreateJackpotPush() *pb.M_1810Toc {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return &pb.M_1810Toc{
		Bonus: proto.String(formatMoney(j.currentPool)),
	}
}

// CreateJackpotWinPush 创建彩金中奖推送 (1811)
func (j *JackpotManager) CreateJackpotWinPush(amount float64) *pb.M_1811Toc {
	return &pb.M_1811Toc{
		Bonus: proto.String(formatMoney(amount)),
	}
}

// CreateJackpotHistoryResponse 创建彩金历史响应 (1812)
func (j *JackpotManager) CreateJackpotHistoryResponse() *pb.M_1812Toc {
	history := j.GetJackpotHistory(20) // 获取最近20条记录
	return &pb.M_1812Toc{
		List: history,
	}
}

// ShouldPushJackpot 判断是否应该推送彩金池更新
func (j *JackpotManager) ShouldPushJackpot() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()

	// 每30秒推送一次彩金池更新
	return time.Since(j.lastPushTime) > 30*time.Second
}

// UpdateLastPushTime 更新最后推送时间
func (j *JackpotManager) UpdateLastPushTime() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.lastPushTime = time.Now()
}

// formatMoney 格式化金额显示
func formatMoney(amount float64) string {
	if amount >= 100000000 {
		// 亿
		return fmt.Sprintf("%.2f亿", amount/100000000)
	} else if amount >= 10000 {
		// 万
		return fmt.Sprintf("%.2f万", amount/10000)
	}
	return fmt.Sprintf("%.0f", amount)
}

// StartJackpotTicker 启动彩金定时推送
func (j *JackpotManager) StartJackpotTicker(pushFunc func(msg *pb.M_1810Toc)) {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			if j.ShouldPushJackpot() {
				push := j.CreateJackpotPush()
				pushFunc(push)
				j.UpdateLastPushTime()
			}
		}
	}()
}