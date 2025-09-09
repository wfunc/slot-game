package models

import (
	"time"
)

// GameState 游戏状态模型（用于持久化游戏状态机）
type GameState struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	SessionID    string    `gorm:"uniqueIndex;size:64;not null" json:"session_id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	CurrentState string    `gorm:"size:20;not null" json:"current_state"`
	StateData    string    `gorm:"type:text" json:"state_data"` // JSON格式的状态数据
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (GameState) TableName() string {
	return "game_states"
}