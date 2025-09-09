package game

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// MemoryStatePersister 内存状态持久化（用于测试）
type MemoryStatePersister struct {
	mu     sync.RWMutex
	states map[string]*StateMachineData
}

// NewMemoryStatePersister 创建内存持久化器
func NewMemoryStatePersister() *MemoryStatePersister {
	return &MemoryStatePersister{
		states: make(map[string]*StateMachineData),
	}
}

// Save 保存状态
func (p *MemoryStatePersister) Save(ctx context.Context, sessionID string, state *StateMachineData) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// 深拷贝数据
	stateCopy := *state
	p.states[sessionID] = &stateCopy
	return nil
}

// Load 加载状态
func (p *MemoryStatePersister) Load(ctx context.Context, sessionID string) (*StateMachineData, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	state, exists := p.states[sessionID]
	if !exists {
		return nil, fmt.Errorf("状态不存在: %s", sessionID)
	}
	
	// 返回深拷贝
	stateCopy := *state
	return &stateCopy, nil
}

// Delete 删除状态
func (p *MemoryStatePersister) Delete(ctx context.Context, sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	delete(p.states, sessionID)
	return nil
}

// DatabaseStatePersister 数据库状态持久化
type DatabaseStatePersister struct {
	db *gorm.DB
}

// NewDatabaseStatePersister 创建数据库持久化器
func NewDatabaseStatePersister(db *gorm.DB) *DatabaseStatePersister {
	return &DatabaseStatePersister{
		db: db,
	}
}

// Save 保存状态到数据库
func (p *DatabaseStatePersister) Save(ctx context.Context, sessionID string, state *StateMachineData) error {
	// 将状态数据序列化为JSON
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}
	
	// 保存到游戏记录表
	gameState := &models.GameState{
		SessionID:    sessionID,
		UserID:       state.UserID,
		CurrentState: string(state.CurrentState),
		StateData:    string(stateJSON),
		UpdatedAt:    time.Now(),
	}
	
	// 使用 Upsert 操作（存在则更新，不存在则插入）
	result := p.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Assign(models.GameState{
			CurrentState: gameState.CurrentState,
			StateData:    gameState.StateData,
			UpdatedAt:    gameState.UpdatedAt,
		}).
		FirstOrCreate(&gameState)
	
	if result.Error != nil {
		return fmt.Errorf("保存状态失败: %w", result.Error)
	}
	
	return nil
}

// Load 从数据库加载状态
func (p *DatabaseStatePersister) Load(ctx context.Context, sessionID string) (*StateMachineData, error) {
	var gameState models.GameState
	
	result := p.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(&gameState)
	
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("游戏状态不存在: %s", sessionID)
		}
		return nil, fmt.Errorf("查询状态失败: %w", result.Error)
	}
	
	// 反序列化状态数据
	var state StateMachineData
	if err := json.Unmarshal([]byte(gameState.StateData), &state); err != nil {
		return nil, fmt.Errorf("反序列化状态失败: %w", err)
	}
	
	return &state, nil
}

// Delete 从数据库删除状态
func (p *DatabaseStatePersister) Delete(ctx context.Context, sessionID string) error {
	result := p.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.GameState{})
	
	if result.Error != nil {
		return fmt.Errorf("删除状态失败: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("状态不存在: %s", sessionID)
	}
	
	return nil
}

// RedisStatePersister Redis状态持久化（预留接口）
type RedisStatePersister struct {
	// TODO: 实现Redis持久化
	// client *redis.Client
	ttl time.Duration
}

// NewRedisStatePersister 创建Redis持久化器
func NewRedisStatePersister(ttl time.Duration) *RedisStatePersister {
	return &RedisStatePersister{
		ttl: ttl,
	}
}

// Save 保存状态到Redis
func (p *RedisStatePersister) Save(ctx context.Context, sessionID string, state *StateMachineData) error {
	// TODO: 实现Redis保存
	return fmt.Errorf("Redis持久化未实现")
}

// Load 从Redis加载状态
func (p *RedisStatePersister) Load(ctx context.Context, sessionID string) (*StateMachineData, error) {
	// TODO: 实现Redis加载
	return nil, fmt.Errorf("Redis持久化未实现")
}

// Delete 从Redis删除状态
func (p *RedisStatePersister) Delete(ctx context.Context, sessionID string) error {
	// TODO: 实现Redis删除
	return fmt.Errorf("Redis持久化未实现")
}

// CacheStatePersister 带缓存的持久化器（装饰器模式）
type CacheStatePersister struct {
	cache     StatePersister // 缓存层（如Redis）
	storage   StatePersister // 存储层（如数据库）
	cacheTTL  time.Duration
}

// NewCacheStatePersister 创建带缓存的持久化器
func NewCacheStatePersister(cache, storage StatePersister, cacheTTL time.Duration) *CacheStatePersister {
	return &CacheStatePersister{
		cache:    cache,
		storage:  storage,
		cacheTTL: cacheTTL,
	}
}

// Save 保存状态（同时保存到缓存和存储）
func (p *CacheStatePersister) Save(ctx context.Context, sessionID string, state *StateMachineData) error {
	// 先保存到存储层
	if err := p.storage.Save(ctx, sessionID, state); err != nil {
		return err
	}
	
	// 再保存到缓存层（缓存失败不影响主流程）
	_ = p.cache.Save(ctx, sessionID, state)
	
	return nil
}

// Load 加载状态（优先从缓存加载）
func (p *CacheStatePersister) Load(ctx context.Context, sessionID string) (*StateMachineData, error) {
	// 先从缓存加载
	if state, err := p.cache.Load(ctx, sessionID); err == nil {
		return state, nil
	}
	
	// 缓存未命中，从存储层加载
	state, err := p.storage.Load(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	// 更新缓存（缓存失败不影响主流程）
	_ = p.cache.Save(ctx, sessionID, state)
	
	return state, nil
}

// Delete 删除状态（同时删除缓存和存储）
func (p *CacheStatePersister) Delete(ctx context.Context, sessionID string) error {
	// 先删除缓存
	_ = p.cache.Delete(ctx, sessionID)
	
	// 再删除存储
	return p.storage.Delete(ctx, sessionID)
}