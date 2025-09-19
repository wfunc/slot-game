package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/wfunc/slot-game/internal/pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

// DebugProtobufMessage 用于调试：将protobuf消息转换为可读的JSON格式
func DebugProtobufMessage(cmd uint16, data []byte) string {
	if len(data) == 0 {
		return "empty"
	}

	// 根据命令ID解析对应的消息类型
	var msg proto.Message
	switch cmd {

	// 动物园相关
	case 1801:
		msg = &pb.M_1801Toc{}
	case 1802:
		msg = &pb.M_1802Toc{}
	case 1803:
		msg = &pb.M_1803Toc{}
	case 1804:
		msg = &pb.M_1804Toc{}
	case 1805:
		msg = &pb.M_1805Toc{}
	case 1806:
		msg = &pb.M_1806Toc{}
	case 1807:
		msg = &pb.M_1807Toc{}
	case 1808:
		msg = &pb.M_1808Toc{}
	case 1809:
		msg = &pb.M_1809Toc{}
	case 1810:
		msg = &pb.M_1810Toc{}
	case 1811:
		msg = &pb.M_1811Toc{}
	case 1812:
		msg = &pb.M_1812Toc{}

	// 推送消息
	case 1882:
		msg = &pb.M_1882Toc{}
	case 1883:
		msg = &pb.M_1883Toc{}
	case 1884:
		msg = &pb.M_1884Toc{}
	case 1885:
		msg = &pb.M_1885Toc{}
	case 1886:
		msg = &pb.M_1886Toc{}
	case 1887:
		msg = &pb.M_1887Toc{}
	case 1888:
		msg = &pb.M_1888Toc{}
	case 1899:
		msg = &pb.M_1899Toc{}

	default:
		// 未知消息类型，返回十六进制
		return fmt.Sprintf("unknown_cmd_%d: %x", cmd, data)
	}

	// 解析protobuf
	if err := proto.Unmarshal(data, msg); err != nil {
		return fmt.Sprintf("unmarshal_error: %v, hex: %x", err, data)
	}

	// 转换为JSON格式以便阅读
	marshaler := protojson.MarshalOptions{
		Indent:          "  ",
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}

	jsonBytes, err := marshaler.Marshal(msg)
	if err != nil {
		// 如果无法转换为JSON，尝试使用标准JSON
		if jsonData, err2 := json.MarshalIndent(msg, "", "  "); err2 == nil {
			return string(jsonData)
		}
		return fmt.Sprintf("marshal_error: %v", err)
	}

	return string(jsonBytes)
}

// GetMessageTypeName 获取消息类型名称
func GetMessageTypeName(cmd uint16) string {
	switch cmd {
	case 1801:
		return "进入房间响应"
	case 1802:
		return "离开房间响应"
	case 1803:
		return "下注响应"
	case 1804:
		return "游戏记录响应"
	case 1805:
		return "大奖记录响应"
	case 1806:
		return "使用技能响应"
	case 1807:
		return "房间信息响应"
	case 1808:
		return "购买道具响应"
	case 1809:
		return "道具价格响应"
	case 1810:
		return "彩金池推送"
	case 1811:
		return "彩金中奖推送"
	case 1812:
		return "彩金记录响应"
	case 1882:
		return "玩家使用技能推送"
	case 1883:
		return "动物预告推送"
	case 1884:
		return "动物死亡推送"
	case 1885:
		return "玩家离开推送"
	case 1886:
		return "玩家进入推送"
	case 1887:
		return "动物进场推送"
	case 1888:
		return "动物离开推送"
	case 1899:
		return "打击事件推送"
	default:
		return fmt.Sprintf("未知消息_%d", cmd)
	}
}