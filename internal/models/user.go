package models

import (
	"time"
	
	"gorm.io/gorm"
)

// User 用户基础信息表
type User struct {
	BaseModel
	Username    string       `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Nickname    string       `gorm:"size:100" json:"nickname"`
	Phone       string       `gorm:"uniqueIndex;size:20" json:"phone"`
	Email       string       `gorm:"uniqueIndex;size:100" json:"email"`
	Avatar      string       `gorm:"size:255" json:"avatar"`
	Status      string       `gorm:"size:20;default:'active'" json:"status"` // active, frozen, banned
	Level       int          `gorm:"default:1" json:"level"`
	Experience  int          `gorm:"default:0" json:"experience"`
	VipLevel    int          `gorm:"default:0" json:"vip_level"`
	VipExpireAt *time.Time   `json:"vip_expire_at,omitempty"`
	LastLoginAt *time.Time   `json:"last_login_at,omitempty"`
	LastLoginIP string       `gorm:"size:50" json:"last_login_ip"`
	
	// 关联（注意：Wallet 不直接嵌入，避免循环依赖）
	Profile     UserProfile  `gorm:"foreignKey:UserID" json:"profile,omitempty"`
	Auth        UserAuth     `gorm:"foreignKey:UserID" json:"-"`
	Sessions    []UserSession `gorm:"foreignKey:UserID" json:"-"`
}

// UserProfile 用户详细信息表
type UserProfile struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	RealName    string    `gorm:"size:50" json:"real_name"`
	Gender      string    `gorm:"size:10" json:"gender"` // male, female, other
	Birthday    *time.Time `json:"birthday,omitempty"`
	Province    string    `gorm:"size:50" json:"province"`
	City        string    `gorm:"size:50" json:"city"`
	Address     string    `gorm:"size:255" json:"address"`
	DeviceID    string    `gorm:"size:100" json:"device_id"`
	DeviceType  string    `gorm:"size:50" json:"device_type"` // ios, android, web
	AppVersion  string    `gorm:"size:20" json:"app_version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserAuth 用户认证信息表
type UserAuth struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	Password          string    `gorm:"size:255;not null" json:"-"`
	Salt              string    `gorm:"size:32" json:"-"`
	PayPassword       string    `gorm:"size:255" json:"-"`
	PaySalt           string    `gorm:"size:32" json:"-"`
	TwoFactorEnabled  bool      `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret   string    `gorm:"size:255" json:"-"`
	SecurityQuestion  string    `gorm:"size:255" json:"security_question"`
	SecurityAnswer    string    `gorm:"size:255" json:"-"`
	LoginAttempts     int       `gorm:"default:0" json:"login_attempts"`
	LastAttemptAt     *time.Time `json:"last_attempt_at,omitempty"`
	LockedUntil       *time.Time `json:"locked_until,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// UserSession 用户会话表
type UserSession struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	SessionID   string    `gorm:"uniqueIndex;size:64;not null" json:"session_id"`
	Token       string    `gorm:"uniqueIndex;size:255;not null" json:"token"`
	RefreshToken string   `gorm:"size:255" json:"refresh_token"`
	IP          string    `gorm:"size:50" json:"ip"`
	UserAgent   string    `gorm:"size:255" json:"user_agent"`
	Platform    string    `gorm:"size:20" json:"platform"` // web, mobile, desktop
	DeviceID    string    `gorm:"size:100" json:"device_id"`
	IsOnline    bool      `gorm:"default:true" json:"is_online"`
	LastActiveAt time.Time `json:"last_active_at"`
	ExpireAt    time.Time `json:"expire_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// 关联（注意：不直接嵌入 User，避免循环依赖）
	// 查询时使用 Preload("User") 来加载用户信息
}

// TableName 指定User表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate 创建前的钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 设置默认昵称
	if u.Nickname == "" {
		u.Nickname = u.Username
	}
	// 设置默认状态
	if u.Status == "" {
		u.Status = "active"
	}
	return nil
}

// IsActive 检查用户是否激活
func (u *User) IsActive() bool {
	return u.Status == "active"
}

// IsVip 检查用户是否是VIP
func (u *User) IsVip() bool {
	if u.VipLevel == 0 {
		return false
	}
	if u.VipExpireAt == nil {
		return true
	}
	return u.VipExpireAt.After(time.Now())
}

// CanLogin 检查用户是否可以登录
func (u *User) CanLogin() bool {
	return u.Status == "active"
}

// UpdateLoginInfo 更新登录信息
func (u *User) UpdateLoginInfo(ip string) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ip
}