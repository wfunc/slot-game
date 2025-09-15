package websocket

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// ProtobufCodec 处理protobuf消息的编解码
type ProtobufCodec struct{}

// NewProtobufCodec 创建新的protobuf编解码器
func NewProtobufCodec() *ProtobufCodec {
	return &ProtobufCodec{}
}

// Encode 编码protobuf消息为二进制格式
// 格式: [4字节长度][2字节消息ID][protobuf数据]
func (c *ProtobufCodec) Encode(msgID uint16, msg proto.Message) ([]byte, error) {
	// 序列化protobuf消息
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal protobuf failed: %w", err)
	}

	// 计算总长度 (2字节消息ID + protobuf数据)
	totalLen := 2 + len(data)
	
	// 创建缓冲区
	buf := bytes.NewBuffer(make([]byte, 0, 4+totalLen))
	
	// 写入长度（4字节，大端序）
	if err := binary.Write(buf, binary.BigEndian, uint32(totalLen)); err != nil {
		return nil, fmt.Errorf("write length failed: %w", err)
	}
	
	// 写入消息ID（2字节，大端序）
	if err := binary.Write(buf, binary.BigEndian, msgID); err != nil {
		return nil, fmt.Errorf("write message ID failed: %w", err)
	}
	
	// 写入protobuf数据
	if _, err := buf.Write(data); err != nil {
		return nil, fmt.Errorf("write protobuf data failed: %w", err)
	}
	
	return buf.Bytes(), nil
}

// Decode 解码二进制数据为消息ID和protobuf数据
// 返回消息ID和protobuf数据部分
func (c *ProtobufCodec) Decode(data []byte) (uint16, []byte, error) {
	if len(data) < 6 {
		return 0, nil, fmt.Errorf("data too short: %d bytes", len(data))
	}
	
	reader := bytes.NewReader(data)
	
	// 读取长度（4字节）
	var length uint32
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return 0, nil, fmt.Errorf("read length failed: %w", err)
	}
	
	// 验证长度
	if int(length)+4 != len(data) {
		return 0, nil, fmt.Errorf("length mismatch: expected %d, got %d", length+4, len(data))
	}
	
	// 读取消息ID（2字节）
	var msgID uint16
	if err := binary.Read(reader, binary.BigEndian, &msgID); err != nil {
		return 0, nil, fmt.Errorf("read message ID failed: %w", err)
	}
	
	// 剩余的是protobuf数据
	protoData := data[6:]
	
	return msgID, protoData, nil
}

// DecodeHeader 只解码消息头部（长度和消息ID）
func (c *ProtobufCodec) DecodeHeader(data []byte) (uint32, uint16, error) {
	if len(data) < 6 {
		return 0, 0, fmt.Errorf("data too short for header: %d bytes", len(data))
	}
	
	reader := bytes.NewReader(data)
	
	// 读取长度（4字节）
	var length uint32
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return 0, 0, fmt.Errorf("read length failed: %w", err)
	}
	
	// 读取消息ID（2字节）
	var msgID uint16
	if err := binary.Read(reader, binary.BigEndian, &msgID); err != nil {
		return 0, 0, fmt.Errorf("read message ID failed: %w", err)
	}
	
	return length, msgID, nil
}