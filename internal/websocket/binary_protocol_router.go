package websocket

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/wfunc/slot-game/internal/game/animal"
	"github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// BinaryProtocolRouter 二进制协议路由器，将前端的二进制协议转换为内部的protobuf消息并路由
// 实现 MessageHandler 接口
type BinaryProtocolRouter struct {
	slotHandler    *SlotHandler
	animalHandler  *AnimalHandler
	configHandler  *ConfigHandler
	codec          *ProtobufCodec
	clientManager  *ClientManager
	pushManager    *PushManager
	animalRooms    map[uint32]*animal.AnimalRoom // roomID -> room
	logger         *zap.Logger
	db             *gorm.DB
}

// 确保 BinaryProtocolRouter 实现了 MessageHandler 接口
var _ MessageHandler = (*BinaryProtocolRouter)(nil)

// NewBinaryProtocolRouter 创建二进制协议路由器
func NewBinaryProtocolRouter(db *gorm.DB, logger *zap.Logger) *BinaryProtocolRouter {
	clientManager := NewClientManager(logger)
	pushManager := NewPushManager(clientManager, logger)

	r := &BinaryProtocolRouter{
		slotHandler:   NewSlotHandler(db),
		animalHandler: NewAnimalHandler(db, logger),
		configHandler: NewConfigHandler(db, logger),
		codec:         NewProtobufCodec(),
		clientManager: clientManager,
		pushManager:   pushManager,
		animalRooms:   make(map[uint32]*animal.AnimalRoom),
		logger:        logger,
		db:            db,
	}

	// 初始化默认房间
	r.initDefaultAnimalRoom()

	return r
}

// HandleMessage 处理来自前端的二进制协议消息
func (r *BinaryProtocolRouter) HandleMessage(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 开始路由消息",
		zap.String("client_id", client.ID),
		zap.Uint16("cmd", msg.Cmd),
		zap.Uint32("flag", msg.Flag),
		zap.Uint8("data_status", msg.DataStatus),
		zap.Int("data_len", len(msg.Data)))

	// 打印接收到的数据
	if len(msg.Data) > 0 {
		r.logger.Info("[路由] 接收到的Data内容",
			zap.String("data_hex", fmt.Sprintf("%x", msg.Data)),
			zap.String("data_string", string(msg.Data)))
	}

	// 根据命令ID范围路由到不同的处理器
	switch {
	case msg.Cmd >= 1900 && msg.Cmd < 2000:
		// Slot游戏相关命令 (1900-1999)
		return r.routeToSlotHandler(client, msg)

	case msg.Cmd >= 1800 && msg.Cmd < 1900:
		// Animal游戏相关命令 (1800-1899)
		return r.routeToAnimalHandler(client, msg)

	case msg.Cmd >= 2000 && msg.Cmd < 2100:
		// 配置相关命令 (2000-2099)
		return r.routeToConfigHandler(client, msg)

	case msg.Cmd == 1002:
		// 心跳命令
		return r.handleHeartbeat(client, msg)

	default:
		r.logger.Warn("[路由] 未知命令",
			zap.Uint16("cmd", msg.Cmd),
			zap.Uint32("flag", msg.Flag))
		return client.GetProtocol().CreateErrorResponse(msg.Cmd, msg.Flag, 1000,
			fmt.Sprintf("未知命令: %d", msg.Cmd)), nil
	}
}

// routeToSlotHandler 路由到老虎机处理器
func (r *BinaryProtocolRouter) routeToSlotHandler(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	// 将前端的二进制消息转换为protobuf格式
	// 构建protobuf格式的消息（与SlotHandler期望的格式一致）
	protoData := r.buildProtobufMessage(msg.Cmd, msg.Data)

	// 创建一个临时的WebSocket连接包装器
	// 注意：这里需要一个更好的方式来处理这个转换
	// TODO: 重构SlotHandler以支持通用的消息接口

	r.logger.Info("转发到Slot处理器",
		zap.Uint16("cmd", msg.Cmd),
		zap.Int("proto_len", len(protoData)))

	// 目前返回一个模拟响应
	// TODO: 实际调用SlotHandler的相应方法
	return r.createSlotResponse(client, msg)
}

// routeToAnimalHandler 路由到动物游戏处理器
func (r *BinaryProtocolRouter) routeToAnimalHandler(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 转发到Animal处理器",
		zap.Uint16("cmd", msg.Cmd),
		zap.Uint32("flag", msg.Flag),
		zap.Int("data_len", len(msg.Data)))

	// 根据不同的命令处理
	switch msg.Cmd {
	case 1801: // 进入房间
		return r.handleAnimalEnterRoom(client, msg)
	case 1802: // 离开房间
		return r.handleAnimalLeaveRoom(client, msg)
	case 1803: // 下注
		return r.handleAnimalBet(client, msg)
	case 1815: // 发射子弹
		return r.handleAnimalFireBullet(client, msg)
	default:
		// 其他命令暂时返回空响应
		response := &ServerMessage{
			ErrorID:    0,  // 成功
			DataStatus: 0,
			Flag:       msg.Flag,
			Cmd:        msg.Cmd,
			Data:       []byte{}, // 空的protobuf数据
		}

		r.logger.Info("[路由] Animal响应准备完毕",
			zap.Uint16("cmd", response.Cmd),
			zap.Uint32("flag", response.Flag),
			zap.Uint16("error_id", response.ErrorID),
			zap.Int("data_len", len(response.Data)))

		return response, nil
	}
}

// routeToConfigHandler 路由到配置处理器
func (r *BinaryProtocolRouter) routeToConfigHandler(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("转发到Config处理器",
		zap.Uint16("cmd", msg.Cmd),
		zap.Int("data_len", len(msg.Data)))

	// 处理特定的配置命令
	switch msg.Cmd {
	case 2001:
		// 返回用户信息
		return r.handleGetUserInfo(client, msg)
	case 2099:
		// 返回配置信息
		return r.handleConfigQuery(client, msg)
	default:
		// TODO: 实际调用ConfigHandler的相应方法
		// 返回空的protobuf数据
		return &ServerMessage{
			ErrorID:    0,  // 成功
			DataStatus: 0,
			Flag:       msg.Flag,
			Cmd:        msg.Cmd,
			Data:       []byte{}, // 空的protobuf数据
		}, nil
	}
}

// handleHeartbeat 处理心跳
func (r *BinaryProtocolRouter) handleHeartbeat(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 处理心跳命令")
	response := client.GetProtocol().CreateSuccessResponse(msg.Cmd, msg.Flag, []byte("pong"))

	r.logger.Info("[路由] 心跳响应准备完毕",
		zap.String("data", string(response.Data)))

	return response, nil
}

// handleConfigQuery 处理配置查询
func (r *BinaryProtocolRouter) handleConfigQuery(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 处理2099命令 - 配置查询")

	// 返回空数据的成功响应（ErrorID=0，无数据）
	// 这样前端就不会报错了
	response := &ServerMessage{
		ErrorID:    0,     // 成功
		DataStatus: 0,
		Flag:       msg.Flag,
		Cmd:        msg.Cmd,
		Data:       []byte{}, // 空数据
	}

	r.logger.Info("[路由] 配置响应准备完毕",
		zap.Uint16("cmd", response.Cmd),
		zap.Uint32("flag", response.Flag),
		zap.Uint16("error_id", response.ErrorID),
		zap.Int("data_len", len(response.Data)))

	return response, nil
}

// buildProtobufMessage 构建protobuf消息
func (r *BinaryProtocolRouter) buildProtobufMessage(cmd uint16, data []byte) []byte {
	// 构建格式：[4字节长度][2字节消息ID][protobuf数据]
	encoded, err := r.codec.Encode(cmd, nil)
	if err != nil {
		r.logger.Error("编码protobuf失败", zap.Error(err))
		return nil
	}
	return encoded
}

// createSlotResponse 创建老虎机响应
func (r *BinaryProtocolRouter) createSlotResponse(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 创建Slot响应",
		zap.Uint16("cmd", msg.Cmd),
		zap.Uint32("flag", msg.Flag),
		zap.Int("data_len", len(msg.Data)))

	// 前端发送的数据是protobuf格式，需要返回protobuf格式
	// 目前先返回空的protobuf数据，避免前端解析错误
	var response *ServerMessage

	switch msg.Cmd {
	case 1901: // 进入房间
		// 创建正确的protobuf响应
		r.logger.Info("[路由] 处理1901命令 - 进入房间")

		devID := "SLOT001"
		devNo := uint32(1)
		cfg := &pb.PConfig{
			DevId: &devID,
			DevNo: &devNo,
		}

		respProto := &pb.M_1901Toc{
			BetVal: []uint32{100, 200, 500, 1000, 2000}, // 下注档位
			Cfg:    cfg,  // required配置
		}

		// 序列化protobuf
		responseData, err := proto.Marshal(respProto)
		if err != nil {
			r.logger.Error("[路由] 序列化1901响应失败", zap.Error(err))
			responseData = []byte{}
		}

		response = &ServerMessage{
			ErrorID:    0,  // 成功
			DataStatus: 0,
			Flag:       msg.Flag,
			Cmd:        msg.Cmd,
			Data:       responseData,
		}

	case 1902: // 开始游戏
		// TODO: 实际应该返回 m_1902_toc protobuf消息
		r.logger.Info("[路由] 处理1902命令 - 开始游戏")
		response = &ServerMessage{
			ErrorID:    0,  // 成功
			DataStatus: 0,
			Flag:       msg.Flag,
			Cmd:        msg.Cmd,
			Data:       []byte{}, // 空的protobuf数据
		}

	default:
		r.logger.Info("[路由] 处理其他Slot命令",
			zap.Uint16("cmd", msg.Cmd))
		response = &ServerMessage{
			ErrorID:    0,  // 成功
			DataStatus: 0,
			Flag:       msg.Flag,
			Cmd:        msg.Cmd,
			Data:       []byte{}, // 空的protobuf数据
		}
	}

	r.logger.Info("[路由] Slot响应准备完毕",
		zap.Uint16("cmd", response.Cmd),
		zap.Uint32("flag", response.Flag),
		zap.Uint16("error_id", response.ErrorID),
		zap.Int("data_len", len(response.Data)))

	return response, nil
}

// GetSlotHandler 获取Slot处理器
func (r *BinaryProtocolRouter) GetSlotHandler() *SlotHandler {
	return r.slotHandler
}

// GetAnimalHandler 获取Animal处理器
func (r *BinaryProtocolRouter) GetAnimalHandler() *AnimalHandler {
	return r.animalHandler
}

// handleAnimalEnterRoom 处理进入动物园房间
func (r *BinaryProtocolRouter) handleAnimalEnterRoom(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 处理1801命令 - 进入动物园房间",
		zap.String("client_id", client.ID),
		zap.Int("data_len", len(msg.Data)))

	// TODO: 解析protobuf请求数据
	// 暂时使用默认的平民场
	// var req pb.M_1801Tos
	// zooType := pb.EZooType_civilian

	// 将客户端加入到管理器
	r.clientManager.AddClient(client)

	// 模拟加入房间ID 1（默认房间）
	roomID := uint32(1)
	playerID := uint32(1) // TODO: 从用户会话获取真实玩家ID

	// 将客户端加入到房间
	r.clientManager.JoinRoom(client.ID, roomID, playerID)

	// 创建正确的protobuf响应
	redState := false
	time := uint32(30)
	respProto := &pb.M_1801Toc{
		RedState: &redState,  // required bool
		Time:     &time,      // required uint32
		BetVal:   []uint32{100, 200, 500, 1000}, // 下注档位
	}

	// 序列化protobuf
	responseData, err := proto.Marshal(respProto)
	if err != nil {
		r.logger.Error("[路由] 序列化1801响应失败", zap.Error(err))
		return nil, err
	}

	response := &ServerMessage{
		ErrorID:    0,  // 成功
		DataStatus: 0,
		Flag:       msg.Flag,
		Cmd:        msg.Cmd,
		Data:       responseData,
	}

	r.logger.Info("[路由] 进入动物园响应准备完毕",
		zap.Uint16("cmd", response.Cmd),
		zap.Uint32("flag", response.Flag),
		zap.Int("data_len", len(response.Data)))

	return response, nil
}

// handleAnimalLeaveRoom 处理离开动物园房间
func (r *BinaryProtocolRouter) handleAnimalLeaveRoom(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 处理1802命令 - 离开动物园房间",
		zap.String("client_id", client.ID))

	// 离开房间
	r.clientManager.LeaveRoom(client.ID)

	response := &ServerMessage{
		ErrorID:    0,  // 成功
		DataStatus: 0,
		Flag:       msg.Flag,
		Cmd:        msg.Cmd,
		Data:       []byte{},
	}

	return response, nil
}

// handleAnimalBet 处理下注（子弹击中动物）
func (r *BinaryProtocolRouter) handleAnimalBet(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 处理1803命令 - 下注（子弹击中）",
		zap.String("client_id", client.ID),
		zap.Int("data_len", len(msg.Data)))

	// 解析请求
	req := &pb.M_1803Tos{}
	if err := proto.Unmarshal(msg.Data, req); err != nil {
		r.logger.Error("[路由] 解析1803请求失败", zap.Error(err))
		return nil, err
	}

	animalID := req.GetId()
	bulletID := req.GetBulletId()

	r.logger.Info("[路由] 子弹击中动物",
		zap.Uint32("animal_id", animalID),
		zap.String("bullet_id", bulletID))

	// 模拟游戏结果（实际应该调用游戏逻辑）
	// 随机生成结果
	isHit := animalID%2 == 0 // 偶数ID的动物被击中
	winAmount := uint32(0)
	redBagAmount := uint32(0)

	if isHit {
		winAmount = 100 * (1 + animalID%5) // 根据动物ID生成赢取金额
		if animalID%10 == 0 {
			redBagAmount = 10 // 10%概率获得红包
		}
	}

	// 模拟玩家余额（实际应该从数据库获取）
	currentBalance := uint64(999900)
	if winAmount > 0 {
		currentBalance += uint64(winAmount)
	}

	// 构造响应
	respProto := &pb.M_1803Toc{
		Balance:  proto.Uint64(currentBalance),        // required: 当前余额
		Win:      proto.Uint32(winAmount),             // required: 赢得金额
		RedBag:   proto.Uint32(redBagAmount),          // required: 红包金额
		TotalWin: proto.Uint64(uint64(winAmount)),     // required: 累计赢取
		// Skill 和 FreeGold 是可选的，暂时不填
	}

	// 序列化protobuf
	responseData, err := proto.Marshal(respProto)
	if err != nil {
		r.logger.Error("[路由] 序列化1803响应失败", zap.Error(err))
		return nil, err
	}

	response := &ServerMessage{
		ErrorID:    0,  // 成功
		DataStatus: 0,
		Flag:       msg.Flag,
		Cmd:        msg.Cmd,
		Data:       responseData,
	}

	r.logger.Info("[路由] 下注响应准备完毕",
		zap.Uint16("cmd", response.Cmd),
		zap.Uint32("flag", response.Flag),
		zap.Int("data_len", len(response.Data)),
		zap.Uint32("animal_id", animalID),
		zap.Uint32("win", winAmount),
		zap.Uint32("red_bag", redBagAmount),
		zap.Uint64("balance", currentBalance))

	// 如果击中，推送动物死亡消息（可选）
	if isHit && winAmount > 0 {
		// 这里可以通过PushManager推送1884消息（动物死亡）
		// 暂时跳过推送逻辑
	}

	return response, nil
}

// handleAnimalFireBullet 处理发射子弹
func (r *BinaryProtocolRouter) handleAnimalFireBullet(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 处理1815命令 - 发射子弹",
		zap.String("client_id", client.ID),
		zap.Int("data_len", len(msg.Data)))

	// 解析请求
	req := &pb.M_1815Tos{}
	if err := proto.Unmarshal(msg.Data, req); err != nil {
		r.logger.Error("[路由] 解析1815请求失败", zap.Error(err))
		// 使用默认值
		req.BetVal = proto.Uint32(100)
	}

	betVal := req.GetBetVal()
	if betVal == 0 {
		betVal = 100
	}

	// 生成子弹ID
	bulletID := "bullet_" + uuid.New().String()

	// 模拟余额（实际应该从数据库获取）
	balance := uint64(999900) // 假设扣除了100后的余额

	// 构造响应
	respProto := &pb.M_1815Toc{
		BulletId: proto.String(bulletID),
		Balance:  proto.Uint64(balance),
	}

	// 序列化protobuf
	responseData, err := proto.Marshal(respProto)
	if err != nil {
		r.logger.Error("[路由] 序列化1815响应失败", zap.Error(err))
		return nil, err
	}

	response := &ServerMessage{
		ErrorID:    0,  // 成功
		DataStatus: 0,
		Flag:       msg.Flag,
		Cmd:        msg.Cmd,
		Data:       responseData,
	}

	r.logger.Info("[路由] 发射子弹响应准备完毕",
		zap.Uint16("cmd", response.Cmd),
		zap.Uint32("flag", response.Flag),
		zap.Int("data_len", len(response.Data)),
		zap.String("bullet_id", bulletID),
		zap.Uint32("bet_val", betVal))

	return response, nil
}

// handleGetUserInfo 处理获取用户信息 (2001命令)
func (r *BinaryProtocolRouter) handleGetUserInfo(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	r.logger.Info("[路由] 处理2001命令 - 获取用户信息",
		zap.String("client_id", client.ID),
		zap.Int("data_len", len(msg.Data)))

	// 创建正确的protobuf响应，包含必需的字段
	// m_2001_toc 要求 role_id 和 balance
	roleID := uint32(10001)    // 模拟用户ID
	balance := uint64(1000000)  // 模拟余额（10000.00）
	
	respProto := &pb.M_2001Toc{
		RoleId:  &roleID,   // required uint32
		Balance: &balance,  // required uint64
	}

	// 序列化protobuf
	responseData, err := proto.Marshal(respProto)
	if err != nil {
		r.logger.Error("[路由] 序列化2001响应失败", zap.Error(err))
		return nil, err
	}

	response := &ServerMessage{
		ErrorID:    0,  // 成功
		DataStatus: 0,
		Flag:       msg.Flag,
		Cmd:        msg.Cmd,
		Data:       responseData,
	}

	r.logger.Info("[路由] 用户信息响应准备完毕",
		zap.Uint16("cmd", response.Cmd),
		zap.Uint32("flag", response.Flag),
		zap.Int("data_len", len(response.Data)),
		zap.Uint32("role_id", roleID),
		zap.Uint64("balance", balance))

	return response, nil
}

// initDefaultAnimalRoom 初始化默认动物房间
func (r *BinaryProtocolRouter) initDefaultAnimalRoom() {
	roomID := uint32(1)
	zooType := pb.EZooType_civilian // 平民场

	// 创建推送回调函数
	pushCallback := r.pushManager.CreatePushCallback(roomID)

	// 创建房间
	room := animal.NewAnimalRoom(roomID, zooType, r.logger, pushCallback)
	room.Start()

	r.animalRooms[roomID] = room

	r.logger.Info("[路由] 默认动物房间已初始化",
		zap.Uint32("room_id", roomID),
		zap.String("zoo_type", zooType.String()))
}

// GetAnimalRoom 获取动物房间
func (r *BinaryProtocolRouter) GetAnimalRoom(roomID uint32) *animal.AnimalRoom {
	return r.animalRooms[roomID]
}

// GetClientManager 获取客户端管理器
func (r *BinaryProtocolRouter) GetClientManager() *ClientManager {
	return r.clientManager
}