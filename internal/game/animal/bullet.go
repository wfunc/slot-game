package animal

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Bullet 子弹信息
type Bullet struct {
	ID        string    // 子弹ID
	PlayerID  uint32    // 玩家ID
	BetValue  uint32    // 下注金额
	Multiple  uint32    // 倍数
	CreatedAt time.Time // 创建时间
	Used      bool      // 是否已使用
	ExpiredAt time.Time // 过期时间
}

// BulletManager 子弹管理器
type BulletManager struct {
	mu      sync.RWMutex
	bullets map[string]*Bullet  // bulletID -> Bullet
	playerBullets map[uint32][]*Bullet // playerID -> Bullets
}

// NewBulletManager 创建子弹管理器
func NewBulletManager() *BulletManager {
	bm := &BulletManager{
		bullets: make(map[string]*Bullet),
		playerBullets: make(map[uint32][]*Bullet),
	}

	// 启动清理协程，清理过期子弹
	go bm.cleanupExpiredBullets()

	return bm
}

// CreateBullet 创建子弹
func (bm *BulletManager) CreateBullet(playerID uint32, betValue uint32, multiple uint32) *Bullet {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bullet := &Bullet{
		ID:        uuid.New().String(),
		PlayerID:  playerID,
		BetValue:  betValue,
		Multiple:  multiple,
		CreatedAt: time.Now(),
		Used:      false,
		ExpiredAt: time.Now().Add(30 * time.Second), // 子弹30秒后过期
	}

	bm.bullets[bullet.ID] = bullet
	bm.playerBullets[playerID] = append(bm.playerBullets[playerID], bullet)

	return bullet
}

// GetBullet 获取子弹
func (bm *BulletManager) GetBullet(bulletID string) (*Bullet, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bullet, exists := bm.bullets[bulletID]
	if !exists {
		return nil, fmt.Errorf("子弹不存在: %s", bulletID)
	}

	if bullet.Used {
		return nil, fmt.Errorf("子弹已使用: %s", bulletID)
	}

	if time.Now().After(bullet.ExpiredAt) {
		return nil, fmt.Errorf("子弹已过期: %s", bulletID)
	}

	return bullet, nil
}

// UseBullet 使用子弹
func (bm *BulletManager) UseBullet(bulletID string) (*Bullet, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bullet, exists := bm.bullets[bulletID]
	if !exists {
		return nil, fmt.Errorf("子弹不存在: %s", bulletID)
	}

	if bullet.Used {
		return nil, fmt.Errorf("子弹已使用: %s", bulletID)
	}

	if time.Now().After(bullet.ExpiredAt) {
		return nil, fmt.Errorf("子弹已过期: %s", bulletID)
	}

	bullet.Used = true
	return bullet, nil
}

// GetPlayerBullets 获取玩家的所有有效子弹
func (bm *BulletManager) GetPlayerBullets(playerID uint32) []*Bullet {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bullets := bm.playerBullets[playerID]
	validBullets := make([]*Bullet, 0)

	now := time.Now()
	for _, bullet := range bullets {
		if !bullet.Used && now.Before(bullet.ExpiredAt) {
			validBullets = append(validBullets, bullet)
		}
	}

	return validBullets
}

// GetOldestPlayerBullet 获取玩家最旧的有效子弹（FIFO）
func (bm *BulletManager) GetOldestPlayerBullet(playerID uint32) *Bullet {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bullets := bm.playerBullets[playerID]
	now := time.Now()

	for _, bullet := range bullets {
		if !bullet.Used && now.Before(bullet.ExpiredAt) {
			return bullet
		}
	}

	return nil
}

// UseOldestPlayerBullet 使用玩家最旧的子弹
func (bm *BulletManager) UseOldestPlayerBullet(playerID uint32) (*Bullet, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bullets := bm.playerBullets[playerID]
	now := time.Now()

	for _, bullet := range bullets {
		if !bullet.Used && now.Before(bullet.ExpiredAt) {
			bullet.Used = true
			return bullet, nil
		}
	}

	return nil, fmt.Errorf("玩家没有有效子弹")
}

// cleanupExpiredBullets 清理过期子弹
func (bm *BulletManager) cleanupExpiredBullets() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		bm.mu.Lock()
		now := time.Now()

		// 清理过期子弹
		for id, bullet := range bm.bullets {
			if now.After(bullet.ExpiredAt.Add(1 * time.Minute)) {
				delete(bm.bullets, id)

				// 从玩家子弹列表中移除
				if playerBullets, exists := bm.playerBullets[bullet.PlayerID]; exists {
					newList := make([]*Bullet, 0)
					for _, b := range playerBullets {
						if b.ID != id {
							newList = append(newList, b)
						}
					}
					if len(newList) > 0 {
						bm.playerBullets[bullet.PlayerID] = newList
					} else {
						delete(bm.playerBullets, bullet.PlayerID)
					}
				}
			}
		}

		bm.mu.Unlock()
	}
}

// GetBulletCount 获取玩家的有效子弹数量
func (bm *BulletManager) GetBulletCount(playerID uint32) int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bullets := bm.playerBullets[playerID]
	count := 0
	now := time.Now()

	for _, bullet := range bullets {
		if !bullet.Used && now.Before(bullet.ExpiredAt) {
			count++
		}
	}

	return count
}

// ClearPlayerBullets 清空玩家的所有子弹（玩家离开房间时调用）
func (bm *BulletManager) ClearPlayerBullets(playerID uint32) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bullets := bm.playerBullets[playerID]
	for _, bullet := range bullets {
		delete(bm.bullets, bullet.ID)
	}
	delete(bm.playerBullets, playerID)
}