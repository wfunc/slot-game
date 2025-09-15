package websocket

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wfunc/slot-game/internal/game/slot"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/pb"
	"github.com/wfunc/slot-game/internal/repository"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// SlotSessionSimple 表示一个玩家的游戏会话
type SlotSessionSimple struct {
	ID          string
	UserID      uint  // 添加用户ID字段
	Conn        *websocket.Conn
	Engine      *slot.GoldenWildCascadeEngine // 使用具体的引擎类型
	Codec       *ProtobufCodec
	CurrentBet  uint32
	Balance     int64
	GameState   string // "idle", "playing", "free_spin"
	FreeSpins   int
	TotalWin    int64           // 单次中奖金额
	TotalDownCoins int64        // 累计落币数（总落币）
	LastSync    time.Time
	mu          sync.RWMutex
}

// SlotHandler 处理老虎机游戏的WebSocket连接
type SlotHandler struct {
	sessions       map[string]*SlotSessionSimple
	mu             sync.RWMutex
	db             *gorm.DB
	walletRepo     repository.WalletRepository
	jackpotRepo    *repository.JackpotRepository
	gameID         uint  // 当前游戏ID（老虎机）
}

// NewSlotHandler 创建新的老虎机处理器
func NewSlotHandler(db *gorm.DB) *SlotHandler {
	// 初始化仓储
	walletRepo := repository.NewWalletRepository(db)
	jackpotRepo := repository.NewJackpotRepository(db)
	
	// 获取或创建游戏记录
	var game models.Game
	err := db.Where("type = ?", "slot").First(&game).Error
	if err == gorm.ErrRecordNotFound {
		game = models.Game{
			Name:        "Golden Wild Mahjong",
			Type:        "slot",
			Description: "金色Wild麻将老虎机",
			Status:      "active",
			MinBet:      10,
			MaxBet:      1000,
			RTP:         96.5,
		}
		db.Create(&game)
	}
	
	// 初始化JP池
	if err := jackpotRepo.InitializeJackpots(game.ID); err != nil {
		log.Printf("[SlotHandler] 初始化JP池失败: %v", err)
	}
	
	return &SlotHandler{
		sessions:    make(map[string]*SlotSessionSimple),
		db:          db,
		walletRepo:  walletRepo,
		jackpotRepo: jackpotRepo,
		gameID:      game.ID,
	}
}

// HandleConnection 处理新的WebSocket连接
func (h *SlotHandler) HandleConnection(conn *websocket.Conn) {
	sessionID := uuid.New().String()
	
	// 创建临时用户ID（实际应该从认证系统获取）
	userID := h.getOrCreateTestUser()
	
	// 创建会话
	session := &SlotSessionSimple{
		ID:        sessionID,
		UserID:    userID,
		Conn:      conn,
		Codec:     NewProtobufCodec(),
		Balance:   10000, // 初始金币
		GameState: "idle",
		TotalDownCoins: 0, // 初始化落币数
		LastSync:  time.Now(),
	}
	
	// 保存会话
	h.mu.Lock()
	h.sessions[sessionID] = session
	h.mu.Unlock()
	
	log.Printf("[SlotHandler] 新玩家连接: %s", sessionID)
	
	// 处理消息
	h.handleMessages(session)
	
	// 清理会话
	h.mu.Lock()
	delete(h.sessions, sessionID)
	h.mu.Unlock()
	
	log.Printf("[SlotHandler] 玩家断开连接: %s", sessionID)
}

// handleMessages 处理客户端消息
func (h *SlotHandler) handleMessages(session *SlotSessionSimple) {
	defer session.Conn.Close()
	
	for {
		// 读取消息
		msgType, data, err := session.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[SlotHandler] 读取消息错误: %v", err)
			}
			break
		}
		
		// 只处理二进制消息（protobuf）
		if msgType != websocket.BinaryMessage {
			log.Printf("[SlotHandler] 忽略非二进制消息")
			continue
		}
		
		// 解码消息
		msgID, protoData, err := session.Codec.Decode(data)
		if err != nil {
			log.Printf("[SlotHandler] 解码消息失败: %v", err)
			continue
		}
		
		// 处理不同的消息类型
		switch msgID {
		case 1901: // 进入房间
			h.handleEnterRoom(session, protoData)
		case 1902: // 开始游戏
			h.handleStartGame(session, protoData)
		default:
			log.Printf("[SlotHandler] 未知消息ID: %d", msgID)
		}
	}
}

// handleEnterRoom 处理进入房间请求
func (h *SlotHandler) handleEnterRoom(session *SlotSessionSimple, data []byte) {
	// 解析请求
	req := &pb.M_1901Tos{}
	if err := proto.Unmarshal(data, req); err != nil {
		log.Printf("[SlotHandler] 解析进入房间请求失败: %v", err)
		return
	}
	
	log.Printf("[SlotHandler] 玩家 %s 进入房间，类型: %v", session.ID, req.GetType())
	
	// 创建游戏引擎配置
	cascadeConfig := slot.GetDefaultCascadeConfig()
	cascadeConfig.GridWidth = 5
	cascadeConfig.GridHeight = 4
	cascadeConfig.MinMatch = 3
	cascadeConfig.MaxCascades = 10
	
	algorithmConfig := &slot.AlgorithmConfig{
		ReelCount:   5,
		RowCount:    4,
		SymbolCount: 8,
		TargetRTP:   0.96,
		MinRTP:      0.94,
		MaxRTP:      0.98,
		
		// 符号权重配置
		SymbolWeights: [][]int{
			{18, 16, 14, 12, 12, 10, 8, 6},
			{16, 18, 14, 12, 12, 10, 8, 6},
			{14, 16, 18, 12, 12, 10, 8, 6},
			{12, 14, 16, 18, 12, 10, 8, 6},
			{12, 12, 14, 16, 18, 12, 8, 6},
		},
		
		// 赔付表
		PayTable: map[int][]int64{
			0: {0, 0, 20, 60, 200},
			1: {0, 0, 25, 75, 250},
			2: {0, 0, 30, 90, 300},
			3: {0, 0, 15, 45, 150},
			4: {0, 0, 12, 36, 120},
			5: {0, 0, 10, 30, 100},
			6: {0, 0, 8, 24, 80},
			7: {0, 0, 6, 18, 60},
		},
		
		Algorithm:    slot.AlgorithmTypeClassic,
		Volatility:   0.55,
		HitFrequency: 0.4,
	}
	
	// 创建游戏引擎
	session.mu.Lock()
	session.Engine = slot.NewGoldenWildCascadeEngine(algorithmConfig, cascadeConfig)
	session.mu.Unlock()
	
	// 构造响应
	resp := &pb.M_1901Toc{
		BetVal: []uint32{10, 20, 50, 100, 200, 500, 1000}, // 下注档位
		Odds: []*pb.PSlotOdds{
			{Odds: proto.Uint32(1), Val: proto.Uint32(10)},
			{Odds: proto.Uint32(2), Val: proto.Uint32(20)},
			{Odds: proto.Uint32(5), Val: proto.Uint32(50)},
			{Odds: proto.Uint32(10), Val: proto.Uint32(100)},
			{Odds: proto.Uint32(20), Val: proto.Uint32(200)},
			{Odds: proto.Uint32(50), Val: proto.Uint32(500)},
			{Odds: proto.Uint32(100), Val: proto.Uint32(1000)},
		},
	}
	
	// 发送响应
	if err := h.sendMessage(session, 1901, resp); err != nil {
		log.Printf("[SlotHandler] 发送进入房间响应失败: %v", err)
	}
	
	// 发送初始数据推送
	h.pushGameData(session)
}

// handleStartGame 处理开始游戏请求
func (h *SlotHandler) handleStartGame(session *SlotSessionSimple, data []byte) {
	// 解析请求
	req := &pb.M_1902Tos{}
	if err := proto.Unmarshal(data, req); err != nil {
		log.Printf("[SlotHandler] 解析开始游戏请求失败: %v", err)
		return
	}
	
	betAmount := req.GetBetVal()
	log.Printf("[SlotHandler] 玩家 %s 开始游戏，下注: %d", session.ID, betAmount)
	
	session.mu.Lock()
	
	// 检查余额
	if session.Balance < int64(betAmount) {
		session.mu.Unlock()
		log.Printf("[SlotHandler] 玩家余额不足: %d < %d", session.Balance, betAmount)
		return
	}
	
	// 检查引擎
	if session.Engine == nil {
		session.mu.Unlock()
		log.Printf("[SlotHandler] 游戏引擎未初始化")
		return
	}
	
	// 扣除下注金额
	session.Balance -= int64(betAmount)
	session.CurrentBet = betAmount
	session.GameState = "playing"
	
	engine := session.Engine
	session.mu.Unlock()
	
	// 创建旋转请求
	spinReq := &slot.SpinRequest{
		GameRequest: &slot.GameRequest{
			SessionID: session.ID,
			BetAmount: int64(betAmount),
			Metadata: map[string]interface{}{
				"game_type": "golden_wild_mahjong",
			},
		},
		ThemeID:     "mahjong",
		EnableTheme: false,
	}
	
	// 执行游戏
	ctx := context.Background()
	result, err := engine.SpinWithGoldenWild(ctx, spinReq)
	if err != nil {
		log.Printf("[SlotHandler] 游戏执行失败: %v", err)
		return
	}
	
	// 更新余额和落币数
	session.mu.Lock()
	// 余额不变（不加中奖金额）
	// session.Balance 保持不变，中奖金额累计到落币数
	session.TotalWin = result.TotalWin  // 记录本次中奖
	session.TotalDownCoins += result.TotalWin  // 累计到总落币数
	
	// 检查免费游戏
	isFree := false
	currentFree := uint32(0)
	totalFree := uint32(0)
	if result.CascadeCount >= 3 {
		isFree = true
		currentFree = 1
		totalFree = uint32(result.CascadeCount)
		session.FreeSpins = result.CascadeCount - 1
	}
	
	session.GameState = "idle"
	userIDNum := session.UserID
	session.mu.Unlock()
	
	// 在事务中更新JP池和用户资产
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 累计JP池
		if err := h.jackpotRepo.AccumulateJackpot(tx, h.gameID, int64(betAmount)); err != nil {
			return fmt.Errorf("累计JP池失败: %w", err)
		}
		
		// 更新用户钱包统计（包括投币数和落币数）
		coinsIn := int64(betAmount)  // 投币数 = 下注金额
		coinsOut := result.TotalWin  // 落币数 = 赢取金额
		
		if err := h.walletRepo.UpdateGameStatsTx(tx, userIDNum, int64(betAmount), result.TotalWin, coinsIn, coinsOut); err != nil {
			return fmt.Errorf("更新用户资产失败: %w", err)
		}
		
		// 获取最终网格（如果有消除步骤，使用最后一步的网格）
		var finalGrid [][]int
		if len(result.CascadeDetails) > 0 {
			finalGrid = result.CascadeDetails[len(result.CascadeDetails)-1].GridAfter
		} else {
			finalGrid = result.InitialGrid
		}
		
		// 创建游戏结果记录
		gameResult := &models.GameResult{
			UserID:    userIDNum,
			GameID:    h.gameID,
			SessionID: 0, // 暂时使用0，实际应该从游戏会话获取
			RoundID:   uuid.New().String(),
			BetAmount: int64(betAmount),
			WinAmount: result.TotalWin,
			Multiplier: float64(result.TotalWin) / float64(betAmount),
			Result: models.JSONMap{
				"cascade_count": result.CascadeCount,
				"initial_grid": result.InitialGrid,
				"final_grid": finalGrid,
			},
			IsJackpot: false,
			IsBonus:   isFree,
			PlayedAt:  time.Now(),
		}
		
		if err := tx.Create(gameResult).Error; err != nil {
			return fmt.Errorf("创建游戏结果失败: %w", err)
		}
		
		return nil
	})
	
	if err != nil {
		log.Printf("[SlotHandler] 数据库操作失败: %v", err)
	}
	
	// 构造游戏结果
	slotResult := &pb.PSlotResult{
		Line1:   convertSymbols(result.InitialGrid[0]),
		Line2:   convertSymbols(result.InitialGrid[1]),
		Line3:   convertSymbols(result.InitialGrid[2]),
		Line4:   convertSymbols(result.InitialGrid[3]),
		Line5:   []pb.ESlotBetType{}, // 5行布局，但实际只有4行
		Rewards: convertRewards(result),
	}
	
	// 构造响应
	resp := &pb.M_1902Toc{
		BetVal:      proto.Uint32(betAmount),
		Win:         proto.Uint32(uint32(result.TotalWin)),
		TotalWin:    proto.Uint32(uint32(result.TotalWin)),
		IsFree:      proto.Bool(isFree),
		CurrentFree: proto.Uint32(currentFree),
		TotalFree:   proto.Uint32(totalFree),
		Result:      slotResult,
	}
	
	// 发送响应
	if err := h.sendMessage(session, 1902, resp); err != nil {
		log.Printf("[SlotHandler] 发送游戏结果失败: %v", err)
	}
	
	// 推送最新数据
	h.pushGameData(session)
}

// pushGameData 推送游戏数据
func (h *SlotHandler) pushGameData(session *SlotSessionSimple) {
	session.mu.RLock()
	balance := session.Balance
	totalDownCoins := session.TotalDownCoins  // 使用累计落币数
	session.mu.RUnlock()
	
	// 获取真实的JP池数据
	jackpots, err := h.jackpotRepo.GetAllJackpots(h.gameID)
	var jp1, jp2, jp3, jpAll uint32
	if err == nil {
		for _, jp := range jackpots {
			switch jp.Type {
			case "JP1":
				jp1 = uint32(jp.Amount)
			case "JP2":
				jp2 = uint32(jp.Amount)
			case "JP3":
				jp3 = uint32(jp.Amount)
			case "JPALL":
				jpAll = uint32(jp.Amount)
			}
		}
	} else {
		// 使用默认值
		jp1 = 1000
		jp2 = 5000
		jp3 = 10000
		jpAll = 20000
	}
	
	// 构造推送数据
	push := &pb.M_1903Toc{
		Coins:     proto.Uint32(uint32(balance)),  // 当前金币
		Remain:    proto.Uint32(uint32(balance)),  // 余分（等于余额）
		DownCoins: proto.Uint32(uint32(totalDownCoins)), // 落币（累计总落币数）
		Jp1:       proto.Uint32(jp1),              // JP1奖池
		Jp2:       proto.Uint32(jp2),              // JP2奖池
		Jp3:       proto.Uint32(jp3),              // JP3奖池
		JpAll:     proto.Uint32(jpAll),            // 总奖池
	}
	
	// 发送推送
	if err := h.sendMessage(session, 1903, push); err != nil {
		log.Printf("[SlotHandler] 推送游戏数据失败: %v", err)
	}
	
	// 更新同步时间
	session.mu.Lock()
	session.LastSync = time.Now()
	session.mu.Unlock()
}

// sendMessage 发送protobuf消息
func (h *SlotHandler) sendMessage(session *SlotSessionSimple, msgID uint16, msg proto.Message) error {
	// 编码消息
	data, err := session.Codec.Encode(msgID, msg)
	if err != nil {
		return fmt.Errorf("encode message failed: %w", err)
	}
	
	// 发送消息
	if err := session.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("write message failed: %w", err)
	}
	
	log.Printf("[SlotHandler] 发送消息 ID=%d, 长度=%d字节", msgID, len(data))
	return nil
}

// convertSymbols 转换符号ID为枚举类型
func convertSymbols(row []int) []pb.ESlotBetType {
	result := make([]pb.ESlotBetType, len(row))
	for i, symbolID := range row {
		if symbolID == -1 {
			result[i] = pb.ESlotBetType_e_slot_bet_type_wild
		} else if symbolID >= 0 && symbolID <= 7 {
			result[i] = pb.ESlotBetType(symbolID)
		} else {
			result[i] = pb.ESlotBetType_e_slot_bet_type_0
		}
	}
	return result
}

// convertRewards 转换游戏奖励
func convertRewards(result *slot.GoldenWildResult) []*pb.PSlotReward {
	var rewards []*pb.PSlotReward
	
	// 转换每步的消除奖励
	for _, step := range result.CascadeDetails {
		for _, group := range step.RemovedGroups {
			// 转换位置
			positions := make([]uint32, 0)
			for _, pos := range group.Positions {
				// 将行列位置转换为单一索引 (row * 5 + col)
				positions = append(positions, uint32(pos.Row*5+pos.Reel))
			}
			
			// 创建奖励
			betType := pb.ESlotBetType(group.SymbolID)
			reward := &pb.PSlotReward{
				Type: &betType,
				Val:  proto.Uint32(uint32(group.Payout)),
				Pos:  positions,
			}
			rewards = append(rewards, reward)
		}
	}
	
	return rewards
}

// getOrCreateTestUser 获取或创建测试用户（临时实现）
func (h *SlotHandler) getOrCreateTestUser() uint {
	// 在实际环境中，应该从认证系统获取用户ID
	// 这里使用简单的测试实现
	var user models.User
	var wallet models.Wallet
	
	// 查找或创建测试用户
	if err := h.db.Where("username = ?", "test_slot_user").First(&user).Error; err == gorm.ErrRecordNotFound {
		user = models.User{
			Username: "test_slot_user",
			Nickname: "测试玩家",
			Status:   "active",
			Level:    1,
		}
		h.db.Create(&user)
		
		// 创建钱包
		wallet = models.Wallet{
			UserID:  user.ID,
			Balance: 0,
			Coins:   1000000, // 初始100万游戏币
		}
		h.db.Create(&wallet)
	} else {
		// 确保钱包存在
		if err := h.db.Where("user_id = ?", user.ID).First(&wallet).Error; err == gorm.ErrRecordNotFound {
			wallet = models.Wallet{
				UserID:  user.ID,
				Balance: 0,
				Coins:   1000000,
			}
			h.db.Create(&wallet)
		}
	}
	
	return user.ID
}