package game

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// GameState 游戏状态枚举
type GameState string

const (
	StateIdle       GameState = "idle"        // 待机状态
	StateReady      GameState = "ready"       // 准备状态（已投币）
	StateSpinning   GameState = "spinning"    // 转动中
	StateCalculating GameState = "calculating" // 计算中奖
	StateWinning    GameState = "winning"     // 中奖展示
	StateSettlement GameState = "settlement"  // 结算状态
	StateError      GameState = "error"       // 错误状态
)

// StateTransition 状态转换定义
type StateTransition struct {
	From   GameState
	Event  string
	To     GameState
	Action func(ctx context.Context, sm *StateMachine) error
}

// StateMachine 游戏状态机
type StateMachine struct {
	mu          sync.RWMutex
	currentState GameState
	sessionID   string
	userID      uint
	transitions map[string][]StateTransition
	logger      *zap.Logger
	
	// 状态数据
	betAmount   int64     // 投注金额
	winAmount   int64     // 中奖金额
	startTime   time.Time // 游戏开始时间
	lastUpdate  time.Time // 最后更新时间
	errorMsg    string    // 错误信息
	
	// 回调函数
	onStateChange func(from, to GameState)
	onError       func(err error)
	
	// 持久化接口
	persister StatePersister
}

// StatePersister 状态持久化接口
type StatePersister interface {
	Save(ctx context.Context, sessionID string, state *StateMachineData) error
	Load(ctx context.Context, sessionID string) (*StateMachineData, error)
	Delete(ctx context.Context, sessionID string) error
}

// StateMachineData 状态机数据（用于持久化）
type StateMachineData struct {
	SessionID    string    `json:"session_id"`
	UserID       uint      `json:"user_id"`
	CurrentState GameState `json:"current_state"`
	BetAmount    int64     `json:"bet_amount"`
	WinAmount    int64     `json:"win_amount"`
	StartTime    time.Time `json:"start_time"`
	LastUpdate   time.Time `json:"last_update"`
	ErrorMsg     string    `json:"error_msg,omitempty"`
}

// NewStateMachine 创建新的状态机
func NewStateMachine(sessionID string, userID uint, logger *zap.Logger, persister StatePersister) *StateMachine {
	sm := &StateMachine{
		currentState: StateIdle,
		sessionID:    sessionID,
		userID:       userID,
		transitions:  make(map[string][]StateTransition),
		logger:       logger,
		lastUpdate:   time.Now(),
		persister:    persister,
	}
	
	// 初始化状态转换规则
	sm.initTransitions()
	
	return sm
}

// initTransitions 初始化状态转换规则
func (sm *StateMachine) initTransitions() {
	// 待机 -> 准备（投币）
	sm.addTransition(StateTransition{
		From:  StateIdle,
		Event: "insert_coin",
		To:    StateReady,
		Action: func(ctx context.Context, sm *StateMachine) error {
			sm.startTime = time.Now()
			sm.logger.Info("游戏开始", 
				zap.String("session_id", sm.sessionID),
				zap.Uint("user_id", sm.userID),
				zap.Int64("bet_amount", sm.betAmount))
			return nil
		},
	})
	
	// 准备 -> 转动中（开始游戏）
	sm.addTransition(StateTransition{
		From:  StateReady,
		Event: "start_spin",
		To:    StateSpinning,
		Action: func(ctx context.Context, sm *StateMachine) error {
			if sm.betAmount <= 0 {
				return errors.New("投注金额无效")
			}
			sm.logger.Info("开始转动", zap.String("session_id", sm.sessionID))
			return nil
		},
	})
	
	// 准备 -> 待机（取消）
	sm.addTransition(StateTransition{
		From:  StateReady,
		Event: "cancel",
		To:    StateIdle,
		Action: func(ctx context.Context, sm *StateMachine) error {
			sm.logger.Info("取消游戏", 
				zap.String("session_id", sm.sessionID),
				zap.Int64("bet_amount", sm.betAmount))
			sm.betAmount = 0
			return nil
		},
	})
	
	// 准备 -> 待机（超时）
	sm.addTransition(StateTransition{
		From:  StateReady,
		Event: "timeout",
		To:    StateIdle,
		Action: func(ctx context.Context, sm *StateMachine) error {
			sm.logger.Info("游戏超时", 
				zap.String("session_id", sm.sessionID),
				zap.Int64("bet_amount", sm.betAmount))
			sm.betAmount = 0
			return nil
		},
	})
	
	// 转动中 -> 计算中奖
	sm.addTransition(StateTransition{
		From:  StateSpinning,
		Event: "stop_spin",
		To:    StateCalculating,
		Action: func(ctx context.Context, sm *StateMachine) error {
			sm.logger.Info("停止转动，开始计算", zap.String("session_id", sm.sessionID))
			return nil
		},
	})
	
	// 计算中奖 -> 中奖展示（有奖）
	sm.addTransition(StateTransition{
		From:  StateCalculating,
		Event: "show_win",
		To:    StateWinning,
		Action: func(ctx context.Context, sm *StateMachine) error {
			if sm.winAmount <= 0 {
				return errors.New("中奖金额无效")
			}
			sm.logger.Info("展示中奖", 
				zap.String("session_id", sm.sessionID),
				zap.Int64("win_amount", sm.winAmount))
			return nil
		},
	})
	
	// 计算中奖 -> 结算（无奖）
	sm.addTransition(StateTransition{
		From:  StateCalculating,
		Event: "no_win",
		To:    StateSettlement,
		Action: func(ctx context.Context, sm *StateMachine) error {
			sm.logger.Info("未中奖，进入结算", zap.String("session_id", sm.sessionID))
			return nil
		},
	})
	
	// 中奖展示 -> 结算
	sm.addTransition(StateTransition{
		From:  StateWinning,
		Event: "settle",
		To:    StateSettlement,
		Action: func(ctx context.Context, sm *StateMachine) error {
			sm.logger.Info("中奖结算", 
				zap.String("session_id", sm.sessionID),
				zap.Int64("win_amount", sm.winAmount))
			return nil
		},
	})
	
	// 结算 -> 待机（游戏结束）
	sm.addTransition(StateTransition{
		From:  StateSettlement,
		Event: "finish",
		To:    StateIdle,
		Action: func(ctx context.Context, sm *StateMachine) error {
			duration := time.Since(sm.startTime)
			sm.logger.Info("游戏结束", 
				zap.String("session_id", sm.sessionID),
				zap.Duration("duration", duration),
				zap.Int64("bet", sm.betAmount),
				zap.Int64("win", sm.winAmount))
			
			// 重置游戏数据
			sm.betAmount = 0
			sm.winAmount = 0
			sm.startTime = time.Time{}
			return nil
		},
	})
	
	// 任何状态 -> 错误状态
	for _, state := range []GameState{StateIdle, StateReady, StateSpinning, StateCalculating, StateWinning, StateSettlement} {
		sm.addTransition(StateTransition{
			From:  state,
			Event: "error",
			To:    StateError,
			Action: func(ctx context.Context, sm *StateMachine) error {
				sm.logger.Error("游戏出错", 
					zap.String("session_id", sm.sessionID),
					zap.String("error", sm.errorMsg))
				return nil
			},
		})
	}
	
	// 错误状态 -> 待机（恢复）
	sm.addTransition(StateTransition{
		From:  StateError,
		Event: "recover",
		To:    StateIdle,
		Action: func(ctx context.Context, sm *StateMachine) error {
			sm.logger.Info("从错误恢复", zap.String("session_id", sm.sessionID))
			sm.errorMsg = ""
			sm.betAmount = 0
			sm.winAmount = 0
			return nil
		},
	})
}

// addTransition 添加状态转换
func (sm *StateMachine) addTransition(transition StateTransition) {
	key := sm.transitionKey(transition.From, transition.Event)
	sm.transitions[key] = append(sm.transitions[key], transition)
}

// transitionKey 生成转换键
func (sm *StateMachine) transitionKey(state GameState, event string) string {
	return fmt.Sprintf("%s:%s", state, event)
}

// Trigger 触发事件
func (sm *StateMachine) Trigger(ctx context.Context, event string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	key := sm.transitionKey(sm.currentState, event)
	transitions, exists := sm.transitions[key]
	if !exists || len(transitions) == 0 {
		return fmt.Errorf("无效的状态转换: 状态=%s, 事件=%s", sm.currentState, event)
	}
	
	// 执行第一个匹配的转换
	transition := transitions[0]
	oldState := sm.currentState
	
	// 执行转换动作
	if transition.Action != nil {
		if err := transition.Action(ctx, sm); err != nil {
			// 转换失败，保持原状态
			if sm.onError != nil {
				sm.onError(err)
			}
			return fmt.Errorf("状态转换失败: %w", err)
		}
	}
	
	// 更新状态
	sm.currentState = transition.To
	sm.lastUpdate = time.Now()
	
	// 触发状态变更回调
	if sm.onStateChange != nil {
		sm.onStateChange(oldState, sm.currentState)
	}
	
	// 持久化状态
	if sm.persister != nil {
		data := sm.toData()
		if err := sm.persister.Save(ctx, sm.sessionID, data); err != nil {
			sm.logger.Error("持久化状态失败", 
				zap.Error(err),
				zap.String("session_id", sm.sessionID))
		}
	}
	
	sm.logger.Info("状态转换", 
		zap.String("session_id", sm.sessionID),
		zap.String("from", string(oldState)),
		zap.String("to", string(sm.currentState)),
		zap.String("event", event))
	
	return nil
}

// GetState 获取当前状态
func (sm *StateMachine) GetState() GameState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// SetBetAmount 设置投注金额
func (sm *StateMachine) SetBetAmount(amount int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.betAmount = amount
}

// GetBetAmount 获取投注金额
func (sm *StateMachine) GetBetAmount() int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.betAmount
}

// SetWinAmount 设置中奖金额
func (sm *StateMachine) SetWinAmount(amount int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.winAmount = amount
}

// SetError 设置错误信息
func (sm *StateMachine) SetError(err string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.errorMsg = err
}

// OnStateChange 设置状态变更回调
func (sm *StateMachine) OnStateChange(fn func(from, to GameState)) {
	sm.onStateChange = fn
}

// OnError 设置错误回调
func (sm *StateMachine) OnError(fn func(err error)) {
	sm.onError = fn
}

// CanTransition 检查是否可以转换
func (sm *StateMachine) CanTransition(event string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	key := sm.transitionKey(sm.currentState, event)
	transitions, exists := sm.transitions[key]
	return exists && len(transitions) > 0
}

// GetValidEvents 获取当前状态下的有效事件
func (sm *StateMachine) GetValidEvents() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var events []string
	prefix := string(sm.currentState) + ":"
	
	for key := range sm.transitions {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			event := key[len(prefix):]
			events = append(events, event)
		}
	}
	
	return events
}

// toData 转换为持久化数据
func (sm *StateMachine) toData() *StateMachineData {
	return &StateMachineData{
		SessionID:    sm.sessionID,
		UserID:       sm.userID,
		CurrentState: sm.currentState,
		BetAmount:    sm.betAmount,
		WinAmount:    sm.winAmount,
		StartTime:    sm.startTime,
		LastUpdate:   sm.lastUpdate,
		ErrorMsg:     sm.errorMsg,
	}
}

// LoadFromData 从持久化数据加载
func (sm *StateMachine) LoadFromData(data *StateMachineData) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.sessionID = data.SessionID
	sm.userID = data.UserID
	sm.currentState = data.CurrentState
	sm.betAmount = data.BetAmount
	sm.winAmount = data.WinAmount
	sm.startTime = data.StartTime
	sm.lastUpdate = data.LastUpdate
	sm.errorMsg = data.ErrorMsg
}

// Reset 重置状态机
func (sm *StateMachine) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.currentState = StateIdle
	sm.betAmount = 0
	sm.winAmount = 0
	sm.startTime = time.Time{}
	sm.lastUpdate = time.Now()
	sm.errorMsg = ""
}