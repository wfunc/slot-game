package animal

import (
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
)

// ActivityState 活动状态
type ActivityState string

const (
	ActivityStateIdle       ActivityState = "idle"       // 空闲
	ActivityStatePrepare    ActivityState = "prepare"    // 准备中
	ActivityStateActive     ActivityState = "active"     // 进行中
	ActivityStateSettlement ActivityState = "settlement" // 结算中
)

// ActivityConfig 活动配置（替代缺失的 pb.PActivityOpenData）
type ActivityConfig struct {
	ID       uint32
	PreTime  uint32   // 准备时间（秒）
	WorkTime uint32   // 活动时间（秒）
	SendGold uint32   // 活动赠送金豆
	MaxNum   uint32   // 最大参与人数
	BetVals  []uint32 // 下注档位
	Percent  uint32   // 打动物的概率 1000分比
	Rewards  []RewardConfig // 奖励配置
	AgentID  uint32   // 渠道ID
}

// RewardConfig 奖励配置
type RewardConfig struct {
	Rank uint32
	Gold uint32
}

// NewActivityManager 创建活动管理器
func NewActivityManager(logger *zap.Logger) *ActivityManager {
	return &ActivityManager{
		logger:  logger,
		history: make([]*Activity, 0, 100),
	}
}

// ScheduleActivity 安排新活动
func (am *ActivityManager) ScheduleActivity(config *ActivityConfig) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// 如果有正在进行的活动，返回错误
	if am.currentActivity != nil && am.currentActivity.Status != string(ActivityStateIdle) {
		return fmt.Errorf("活动正在进行中")
	}

	// 创建新活动
	activity := &Activity{
		ID:         config.ID,
		PreTime:    config.PreTime,
		WorkTime:   config.WorkTime,
		SendGold:   config.SendGold,
		MaxPlayers: config.MaxNum,
		BetValues:  config.BetVals,
		Percent:    config.Percent,
		Status:     string(ActivityStateIdle),
		Rankings:   make([]*PlayerRank, 0),
		Rewards:    make([]string, 0),
		AgentID:    config.AgentID,
	}

	// 将奖励配置转换为字符串（或根据实际需要处理）
	for _, reward := range config.Rewards {
		activity.Rewards = append(activity.Rewards, fmt.Sprintf("rank_%d:%d", reward.Rank, reward.Gold))
	}

	am.currentActivity = activity

	// 安排活动开始
	am.scheduleActivityStart()

	am.logger.Info("活动已安排",
		zap.Uint32("activity_id", activity.ID),
		zap.Uint32("pre_time", activity.PreTime),
		zap.Uint32("work_time", activity.WorkTime))

	return nil
}

// scheduleActivityStart 安排活动开始
func (am *ActivityManager) scheduleActivityStart() {
	if am.currentActivity == nil {
		return
	}

	// 立即进入准备阶段
	am.currentActivity.Status = string(ActivityStatePrepare)
	am.currentActivity.StartTime = time.Now()

	// 准备阶段定时器
	am.scheduler = time.AfterFunc(time.Duration(am.currentActivity.PreTime)*time.Second, func() {
		am.mu.Lock()
		defer am.mu.Unlock()

		// 进入活动阶段
		am.currentActivity.Status = string(ActivityStateActive)
		am.logger.Info("活动开始", zap.Uint32("activity_id", am.currentActivity.ID))

		// 活动结束定时器
		am.scheduler = time.AfterFunc(time.Duration(am.currentActivity.WorkTime)*time.Second, func() {
			am.mu.Lock()
			defer am.mu.Unlock()

			// 进入结算阶段
			am.currentActivity.Status = string(ActivityStateSettlement)
			am.currentActivity.EndTime = time.Now()
			am.logger.Info("活动结束，开始结算", zap.Uint32("activity_id", am.currentActivity.ID))

			// 结算奖励
			am.settleActivity()
		})
	})
}

// settleActivity 结算活动
func (am *ActivityManager) settleActivity() {
	if am.currentActivity == nil {
		return
	}

	// 排序玩家排行榜
	sort.Slice(am.currentActivity.Rankings, func(i, j int) bool {
		return am.currentActivity.Rankings[i].Value > am.currentActivity.Rankings[j].Value
	})

	// 更新排名
	for i, rank := range am.currentActivity.Rankings {
		rank.Rank = uint32(i + 1)
	}

	am.logger.Info("活动结算完成",
		zap.Uint32("activity_id", am.currentActivity.ID),
		zap.Int("player_count", len(am.currentActivity.Rankings)))

	// 活动加入历史
	am.history = append(am.history, am.currentActivity)

	// 保留最近100条历史
	if len(am.history) > 100 {
		am.history = am.history[1:]
	}

	// 清空当前活动
	am.currentActivity.Status = string(ActivityStateIdle)
	am.currentActivity = nil

	// 取消定时器
	if am.scheduler != nil {
		am.scheduler.Stop()
		am.scheduler = nil
	}
}

// GetCurrentActivity 获取当前活动
func (am *ActivityManager) GetCurrentActivity() *Activity {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.currentActivity == nil {
		return nil
	}

	// 返回副本，避免被修改
	return &Activity{
		ID:         am.currentActivity.ID,
		PreTime:    am.currentActivity.PreTime,
		WorkTime:   am.currentActivity.WorkTime,
		SendGold:   am.currentActivity.SendGold,
		MaxPlayers: am.currentActivity.MaxPlayers,
		BetValues:  append([]uint32{}, am.currentActivity.BetValues...),
		Percent:    am.currentActivity.Percent,
		Status:     am.currentActivity.Status,
		Rankings:   am.copyRankings(am.currentActivity.Rankings),
		Rewards:    append([]string{}, am.currentActivity.Rewards...),
		StartTime:  am.currentActivity.StartTime,
		EndTime:    am.currentActivity.EndTime,
		AgentID:    am.currentActivity.AgentID,
	}
}

// copyRankings 复制排行榜
func (am *ActivityManager) copyRankings(rankings []*PlayerRank) []*PlayerRank {
	result := make([]*PlayerRank, len(rankings))
	for i, rank := range rankings {
		result[i] = &PlayerRank{
			Rank:     rank.Rank,
			PlayerID: rank.PlayerID,
			Name:     rank.Name,
			Icon:     rank.Icon,
			Value:    rank.Value,
			VIP:      rank.VIP,
		}
	}
	return result
}

// UpdatePlayerActivity 更新玩家活动数据
func (am *ActivityManager) UpdatePlayerActivity(playerID uint32, name string, icon string, vip uint32, betAmount uint64) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.currentActivity == nil || am.currentActivity.Status != string(ActivityStateActive) {
		return
	}

	// 查找或创建玩家排名
	var playerRank *PlayerRank
	for _, rank := range am.currentActivity.Rankings {
		if rank.PlayerID == playerID {
			playerRank = rank
			break
		}
	}

	if playerRank == nil {
		// 检查是否超过最大人数
		if uint32(len(am.currentActivity.Rankings)) >= am.currentActivity.MaxPlayers {
			return
		}

		playerRank = &PlayerRank{
			PlayerID: playerID,
			Name:     name,
			Icon:     icon,
			VIP:      vip,
			Value:    0,
		}
		am.currentActivity.Rankings = append(am.currentActivity.Rankings, playerRank)
	}

	// 更新下注总额
	playerRank.Value += betAmount
}

// GetActivityStatus 获取活动状态
func (am *ActivityManager) GetActivityStatus() (ActivityState, *time.Time, *time.Time) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.currentActivity == nil {
		return ActivityStateIdle, nil, nil
	}

	state := ActivityState(am.currentActivity.Status)
	var startTime, endTime *time.Time

	if !am.currentActivity.StartTime.IsZero() {
		startTime = &am.currentActivity.StartTime
	}
	if !am.currentActivity.EndTime.IsZero() {
		endTime = &am.currentActivity.EndTime
	}

	return state, startTime, endTime
}

// GetActivityRankings 获取活动排行榜
func (am *ActivityManager) GetActivityRankings(limit int) []*PlayerRank {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.currentActivity == nil {
		return nil
	}

	// 排序
	rankings := am.copyRankings(am.currentActivity.Rankings)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Value > rankings[j].Value
	})

	// 限制返回数量
	if limit > 0 && len(rankings) > limit {
		rankings = rankings[:limit]
	}

	// 更新排名
	for i, rank := range rankings {
		rank.Rank = uint32(i + 1)
	}

	return rankings
}

// GetActivityHistory 获取历史活动记录
func (am *ActivityManager) GetActivityHistory(limit int) []*Activity {
	am.mu.RLock()
	defer am.mu.RUnlock()

	start := len(am.history) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*Activity, 0, limit)
	for i := start; i < len(am.history); i++ {
		result = append(result, am.history[i])
	}

	return result
}

// CancelActivity 取消当前活动
func (am *ActivityManager) CancelActivity() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.currentActivity == nil {
		return fmt.Errorf("没有正在进行的活动")
	}

	// 取消定时器
	if am.scheduler != nil {
		am.scheduler.Stop()
		am.scheduler = nil
	}

	// 结算活动
	if am.currentActivity.Status == string(ActivityStateActive) {
		am.currentActivity.Status = string(ActivityStateSettlement)
		am.currentActivity.EndTime = time.Now()
		am.settleActivity()
	} else {
		// 直接取消
		am.currentActivity.Status = string(ActivityStateIdle)
		am.currentActivity = nil
	}

	am.logger.Info("活动已取消")
	return nil
}

// IsActivityActive 检查活动是否进行中
func (am *ActivityManager) IsActivityActive() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return am.currentActivity != nil && am.currentActivity.Status == string(ActivityStateActive)
}

// GetActivityTimeRemaining 获取活动剩余时间
func (am *ActivityManager) GetActivityTimeRemaining() (prepareTime, activeTime int) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.currentActivity == nil {
		return 0, 0
	}

	now := time.Now()
	switch ActivityState(am.currentActivity.Status) {
	case ActivityStatePrepare:
		elapsed := now.Sub(am.currentActivity.StartTime).Seconds()
		prepareTime = int(float64(am.currentActivity.PreTime) - elapsed)
		if prepareTime < 0 {
			prepareTime = 0
		}
		activeTime = int(am.currentActivity.WorkTime)

	case ActivityStateActive:
		prepareTime = 0
		startActive := am.currentActivity.StartTime.Add(time.Duration(am.currentActivity.PreTime) * time.Second)
		elapsed := now.Sub(startActive).Seconds()
		activeTime = int(float64(am.currentActivity.WorkTime) - elapsed)
		if activeTime < 0 {
			activeTime = 0
		}

	default:
		prepareTime = 0
		activeTime = 0
	}

	return prepareTime, activeTime
}