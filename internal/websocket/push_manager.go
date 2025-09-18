package websocket

import (
	"google.golang.org/protobuf/proto"
	"github.com/wfunc/slot-game/internal/game/animal"
	"github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
)

// PushManager 推送管理器
type PushManager struct {
	clientManager *ClientManager
	codec         *ProtobufCodec
	logger        *zap.Logger
}

// NewPushManager 创建推送管理器
func NewPushManager(clientManager *ClientManager, logger *zap.Logger) *PushManager {
	return &PushManager{
		clientManager: clientManager,
		codec:         NewProtobufCodec(),
		logger:        logger,
	}
}

// HandleAnimalPush 处理动物推送消息
func (pm *PushManager) HandleAnimalPush(msg *animal.PushMessage) {
	pm.logger.Info("[PushManager] 处理动物推送",
		zap.Uint32("room_id", msg.RoomID),
		zap.Uint16("msg_id", msg.MsgID),
		zap.String("zoo_type", msg.ZooType.String()))

	// 将protobuf消息编码为二进制数据
	data, err := pm.encodeProtobufMessage(msg.MsgID, msg.Message)
	if err != nil {
		pm.logger.Error("[PushManager] 编码protobuf消息失败",
			zap.Error(err),
			zap.Uint16("msg_id", msg.MsgID))
		return
	}

	// 根据消息类型记录详细信息
	switch msg.MsgID {
	case 1887: // m_1887_toc - 动物进入
		if enterMsg, ok := msg.Message.(*pb.M_1887Toc); ok {
			for _, route := range enterMsg.Animal {
				pm.logger.Info("[PushManager] 推送动物进入",
					zap.Uint32("animal_id", route.GetId()),
					zap.String("animal_type", route.GetBet().String()),
					zap.Uint32("line_id", route.GetLineId()),
					zap.Uint32("point", route.GetPoint()))
			}
		}
	case 1888: // m_1888_toc - 动物离开
		if leaveMsg, ok := msg.Message.(*pb.M_1888Toc); ok {
			pm.logger.Info("[PushManager] 推送动物离开",
				zap.Uint32("animal_id", leaveMsg.GetId()))
		}
	}

	// 广播给房间内的所有客户端
	pm.clientManager.BroadcastToRoom(msg.RoomID, msg.MsgID, data)
}

// encodeProtobufMessage 编码protobuf消息
func (pm *PushManager) encodeProtobufMessage(msgID uint16, message proto.Message) ([]byte, error) {
	// 序列化protobuf消息
	protoData, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	pm.logger.Info("[PushManager] Protobuf消息已序列化",
		zap.Uint16("msg_id", msgID),
		zap.Int("proto_len", len(protoData)))

	// 直接返回protobuf数据
	// 客户端管理器会负责添加协议头
	return protoData, nil
}

// CreatePushCallback 创建推送回调函数
func (pm *PushManager) CreatePushCallback(roomID uint32) func(*animal.PushMessage) {
	return func(msg *animal.PushMessage) {
		msg.RoomID = roomID
		pm.HandleAnimalPush(msg)
	}
}