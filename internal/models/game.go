package models

import (
	"time"
)

// Game 游戏类型表
type Game struct {
	BaseModel
	Name        string    `gorm:"size:100;not null" json:"name"`
	Type        string    `gorm:"size:50;not null" json:"type"` // slot, pusher
	Description string    `gorm:"size:500" json:"description"`
	Icon        string    `gorm:"size:255" json:"icon"`
	Status      string    `gorm:"size:20;default:'active'" json:"status"` // active, maintenance, disabled
	MinBet      int       `gorm:"default:1" json:"min_bet"`
	MaxBet      int       `gorm:"default:100" json:"max_bet"`
	RTP         float64   `gorm:"default:96.5" json:"rtp"` // Return To Player percentage
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	Config      JSONMap   `gorm:"type:json" json:"config"`
}

// GameRoom 游戏房间表
type GameRoom struct {
	BaseModel
	GameID      uint      `gorm:"not null;index" json:"game_id"`
	RoomNumber  string    `gorm:"uniqueIndex;size:50;not null" json:"room_number"`
	Name        string    `gorm:"size:100" json:"name"`
	Type        string    `gorm:"size:50" json:"type"` // normal, vip, tournament
	Status      string    `gorm:"size:20;default:'open'" json:"status"` // open, full, closed, maintenance
	MaxPlayers  int       `gorm:"default:1" json:"max_players"`
	CurrentPlayers int    `gorm:"default:0" json:"current_players"`
	MinBet      int       `json:"min_bet"`
	MaxBet      int       `json:"max_bet"`
	MinLevel    int       `gorm:"default:1" json:"min_level"`
	Config      JSONMap   `gorm:"type:json" json:"config"`
	
	// 关联
	Game        Game      `gorm:"foreignKey:GameID" json:"game,omitempty"`
}

// GameSession 游戏会话表
type GameSession struct {
	BaseModel
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	GameID      uint      `gorm:"not null;index" json:"game_id"`
	RoomID      uint      `gorm:"index" json:"room_id"`
	SessionID   string    `gorm:"uniqueIndex;size:64;not null" json:"session_id"`
	Status      string    `gorm:"size:20;default:'playing'" json:"status"` // playing, paused, ended
	StartedAt   time.Time `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	Duration    int       `json:"duration"` // 秒
	TotalBet    int64     `gorm:"default:0" json:"total_bet"`
	TotalWin    int64     `gorm:"default:0" json:"total_win"`
	TotalRounds int       `gorm:"default:0" json:"total_rounds"`
	PeakWin     int64     `gorm:"default:0" json:"peak_win"`
	GameData    JSONMap   `gorm:"type:json" json:"game_data"`
	
	// 关联
	Game        Game      `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Room        *GameRoom `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

// GameRecord 是 GameResult 的别名，用于兼容性
type GameRecord = GameResult

// GameResult 游戏结果表
type GameResult struct {
	BaseModel
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	GameID      uint      `gorm:"not null;index" json:"game_id"`
	SessionID   uint      `gorm:"not null;index" json:"session_id"`
	RoundID     string    `gorm:"uniqueIndex;size:64;not null" json:"round_id"`
	BetAmount   int64     `gorm:"not null" json:"bet_amount"`
	WinAmount   int64     `gorm:"default:0" json:"win_amount"`
	Multiplier  float64   `gorm:"default:0" json:"multiplier"`
	Result      JSONMap   `gorm:"type:json" json:"result"`
	IsJackpot   bool      `gorm:"default:false" json:"is_jackpot"`
	IsBonus     bool      `gorm:"default:false" json:"is_bonus"`
	PlayedAt    time.Time `json:"played_at"`
	
	// 关联
	Game        Game        `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Session     GameSession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
}

// SlotMachine 老虎机配置表
type SlotMachine struct {
	BaseModel
	GameID      uint      `gorm:"uniqueIndex;not null" json:"game_id"`
	MachineID   string    `gorm:"uniqueIndex;size:50;not null" json:"machine_id"`
	Name        string    `gorm:"size:100" json:"name"`
	Reels       int       `gorm:"default:3" json:"reels"`
	Rows        int       `gorm:"default:3" json:"rows"`
	Paylines    int       `gorm:"default:5" json:"paylines"`
	Symbols     JSONMap   `gorm:"type:json" json:"symbols"`
	PayTable    JSONMap   `gorm:"type:json" json:"pay_table"`
	BonusConfig JSONMap   `gorm:"type:json" json:"bonus_config"`
	JackpotPool int64     `gorm:"default:0" json:"jackpot_pool"`
	Status      string    `gorm:"size:20;default:'active'" json:"status"`
	
	// 关联
	Game        Game      `gorm:"foreignKey:GameID" json:"game,omitempty"`
}

// SlotSpin 老虎机旋转记录表
type SlotSpin struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ResultID    uint      `gorm:"not null;index" json:"result_id"`
	MachineID   uint      `gorm:"not null;index" json:"machine_id"`
	SpinNumber  int       `json:"spin_number"`
	ReelStops   JSONMap   `gorm:"type:json" json:"reel_stops"`
	WinLines    JSONMap   `gorm:"type:json" json:"win_lines"`
	BonusWon    bool      `gorm:"default:false" json:"bonus_won"`
	FreeSpins   int       `gorm:"default:0" json:"free_spins"`
	Multiplier  float64   `gorm:"default:1" json:"multiplier"`
	CreatedAt   time.Time `json:"created_at"`
	
	// 关联
	Result      GameResult  `gorm:"foreignKey:ResultID" json:"result,omitempty"`
	Machine     SlotMachine `gorm:"foreignKey:MachineID" json:"machine,omitempty"`
}

// SlotWinLine 老虎机中奖线记录表
type SlotWinLine struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	SpinID      uint      `gorm:"not null;index" json:"spin_id"`
	LineNumber  int       `json:"line_number"`
	Symbol      string    `gorm:"size:50" json:"symbol"`
	Count       int       `json:"count"`
	WinAmount   int64     `json:"win_amount"`
	Positions   JSONMap   `gorm:"type:json" json:"positions"`
	CreatedAt   time.Time `json:"created_at"`
	
	// 关联
	Spin        SlotSpin  `gorm:"foreignKey:SpinID" json:"spin,omitempty"`
}

// PusherMachine 推币机配置表
type PusherMachine struct {
	BaseModel
	GameID      uint      `gorm:"uniqueIndex;not null" json:"game_id"`
	MachineID   string    `gorm:"uniqueIndex;size:50;not null" json:"machine_id"`
	Name        string    `gorm:"size:100" json:"name"`
	PlatformWidth  float64 `gorm:"default:100" json:"platform_width"`
	PlatformDepth  float64 `gorm:"default:50" json:"platform_depth"`
	PusherForce    float64 `gorm:"default:10" json:"pusher_force"`
	PusherSpeed    float64 `gorm:"default:1" json:"pusher_speed"`
	CoinValue      float64 `gorm:"default:0.1" json:"coin_value"`
	SpecialItems   JSONMap `gorm:"type:json" json:"special_items"`
	CurrentCoins   int     `gorm:"default:0" json:"current_coins"`
	Status         string  `gorm:"size:20;default:'active'" json:"status"`
	
	// 关联
	Game        Game      `gorm:"foreignKey:GameID" json:"game,omitempty"`
}

// PusherSession 推币机会话表
type PusherSession struct {
	BaseModel
	SessionID   uint      `gorm:"not null;index" json:"session_id"`
	MachineID   uint      `gorm:"not null;index" json:"machine_id"`
	CoinsInserted int     `gorm:"default:0" json:"coins_inserted"`
	CoinsWon      int     `gorm:"default:0" json:"coins_won"`
	ItemsWon      JSONMap `gorm:"type:json" json:"items_won"`
	PushCount     int     `gorm:"default:0" json:"push_count"`
	StartState    JSONMap `gorm:"type:json" json:"start_state"`
	EndState      JSONMap `gorm:"type:json" json:"end_state"`
	
	// 关联
	Session     GameSession    `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	Machine     PusherMachine  `gorm:"foreignKey:MachineID" json:"machine,omitempty"`
}

// CoinDrop 推币掉落记录表
type CoinDrop struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	SessionID   uint      `gorm:"not null;index" json:"session_id"`
	DropTime    time.Time `json:"drop_time"`
	CoinCount   int       `json:"coin_count"`
	ItemType    string    `gorm:"size:50" json:"item_type"` // coin, bonus, special
	ItemValue   int64     `json:"item_value"`
	Position    JSONMap   `gorm:"type:json" json:"position"`
	
	// 关联
	Session     PusherSession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
}

