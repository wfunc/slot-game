package animal

import (
	"github.com/wfunc/slot-game/internal/pb"
)

// RoomTypeConfigs 房间类型配置
var RoomTypeConfigs = map[pb.EZooType]*RoomConfig{
	pb.EZooType_free: {
		BetValues:   []uint32{100, 500, 1000},
		MinVIP:      0,
		MaxPlayer:   100,
		UseFreeGold: true,
		OddsType:    "free",
	},
	pb.EZooType_civilian: {
		BetValues:   []uint32{100, 500, 1000},
		MinVIP:      0,
		MaxPlayer:   100,
		UseFreeGold: false,
		OddsType:    "normal",
	},
	pb.EZooType_petty: {
		BetValues:   []uint32{1000, 5000, 10000},
		MinVIP:      1,
		MaxPlayer:   80,
		UseFreeGold: false,
		OddsType:    "normal",
	},
	pb.EZooType_rich: {
		BetValues:   []uint32{10000, 50000, 100000},
		MinVIP:      5,
		MaxPlayer:   50,
		UseFreeGold: false,
		OddsType:    "normal",
	},
	pb.EZooType_gold: {
		BetValues:   []uint32{50000, 100000, 500000},
		MinVIP:      8,
		MaxPlayer:   30,
		UseFreeGold: false,
		OddsType:    "normal",
	},
	pb.EZooType_diamond: {
		BetValues:   []uint32{100000, 500000, 1000000},
		MinVIP:      10,
		MaxPlayer:   20,
		UseFreeGold: false,
		OddsType:    "normal",
	},
	pb.EZooType_single: {
		BetValues:   []uint32{100, 500, 1000},
		MinVIP:      0,
		MaxPlayer:   1,
		UseFreeGold: false,
		OddsType:    "normal",
	},
}

// GetRoomConfig 获取房间配置
func GetRoomConfig(roomType pb.EZooType) *RoomConfig {
	if config, exists := RoomTypeConfigs[roomType]; exists {
		// 返回配置的副本，避免被修改
		return &RoomConfig{
			BetValues:   append([]uint32{}, config.BetValues...),
			MinVIP:      config.MinVIP,
			MaxPlayer:   config.MaxPlayer,
			UseFreeGold: config.UseFreeGold,
			OddsType:    config.OddsType,
		}
	}

	// 默认配置
	return &RoomConfig{
		BetValues:   []uint32{100, 500, 1000},
		MinVIP:      0,
		MaxPlayer:   100,
		UseFreeGold: false,
		OddsType:    "normal",
	}
}

// CanPlayerEnterRoom 检查玩家是否可以进入房间
func CanPlayerEnterRoom(player *Player, roomType pb.EZooType) bool {
	config := GetRoomConfig(roomType)

	// 检查VIP等级
	if player.VIP < config.MinVIP {
		return false
	}

	// 检查金豆余额（至少能下注一次最小档位）
	minBet := config.BetValues[0]
	if config.UseFreeGold {
		// 体验场检查体验币
		if player.FreeGold < uint64(minBet) {
			return false
		}
	} else {
		// 正式场检查金豆
		if player.Balance < uint64(minBet) {
			return false
		}
	}

	return true
}

// GetRoomTypeInfo 获取所有房间类型信息（用于客户端显示）
func GetRoomTypeInfo() []*pb.PZooTypeInfo {
	var result []*pb.PZooTypeInfo

	for zooType, config := range RoomTypeConfigs {
		zt := zooType
		info := &pb.PZooTypeInfo{
			Type:      &zt,
			BetVal:    config.BetValues,
			Vip:       &config.MinVIP,
			MaxNum:    &config.MaxPlayer,
		}

		// 设置房间名称（PZooTypeInfo 没有 Name 字段，跳过）
		// switch zooType {
		// case pb.EZooType_free:
		// 	// 体验场
		// case pb.EZooType_civilian:
		// 	// 平民场
		// case pb.EZooType_petty:
		// 	// 小资场
		// case pb.EZooType_rich:
		// 	// 富豪场
		// case pb.EZooType_gold:
		// 	// 黄金场
		// case pb.EZooType_diamond:
		// 	// 钻石场
		// case pb.EZooType_single:
		// 	// 单人场
		// }

		result = append(result, info)
	}

	return result
}