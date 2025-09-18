package websocket

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// 服务端发给客户端的包头长度
	// <<ErrorID:16, DataSize:16, DataStatus:8, Flag:32, Cmd:16, Data>>
	HeaderSizeToClient = 11

	// 客户端发给服务端的包头长度
	// <<DataSize:16, DataStatus:8, Flag:32, Cmd:16, Data>>
	HeaderSizeFromClient = 9
)

// ClientMessage 客户端发来的消息结构（9字节头）
type ClientMessage struct {
	DataSize   uint16 // 2字节
	DataStatus uint8  // 1字节
	Flag       uint32 // 4字节
	Cmd        uint16 // 2字节
	Data       []byte // 可变长度
}

// ServerMessage 服务端发送的消息结构（11字节头）
type ServerMessage struct {
	ErrorID    uint16 // 2字节
	DataSize   uint16 // 2字节
	DataStatus uint8  // 1字节
	Flag       uint32 // 4字节
	Cmd        uint16 // 2字节
	Data       []byte // 可变长度
}

// Protocol WebSocket协议处理器
type Protocol struct {
	// 是否启用调试日志
	Debug bool
}

// NewProtocol 创建协议处理器
func NewProtocol() *Protocol {
	return &Protocol{
		Debug: false,
	}
}

// DecodeClientMessage 解码客户端消息（9字节头）
// 客户端格式：DataSize(2) + DataStatus(1) + Flag(4) + Cmd(2) + Data(n)
func (p *Protocol) DecodeClientMessage(data []byte) (*ClientMessage, error) {
	if len(data) < HeaderSizeFromClient {
		return nil, fmt.Errorf("数据长度不足，需要至少%d字节，实际%d字节", HeaderSizeFromClient, len(data))
	}

	reader := bytes.NewReader(data)
	msg := &ClientMessage{}

	// 读取DataSize (偏移0，2字节)
	if err := binary.Read(reader, binary.BigEndian, &msg.DataSize); err != nil {
		return nil, fmt.Errorf("读取DataSize失败: %w", err)
	}

	// 读取DataStatus (偏移2，1字节)
	if err := binary.Read(reader, binary.BigEndian, &msg.DataStatus); err != nil {
		return nil, fmt.Errorf("读取DataStatus失败: %w", err)
	}

	// 读取Flag (偏移3，4字节)
	if err := binary.Read(reader, binary.BigEndian, &msg.Flag); err != nil {
		return nil, fmt.Errorf("读取Flag失败: %w", err)
	}

	// 读取Cmd (偏移7，2字节)
	if err := binary.Read(reader, binary.BigEndian, &msg.Cmd); err != nil {
		return nil, fmt.Errorf("读取Cmd失败: %w", err)
	}

	// 验证数据长度
	expectedTotalLength := HeaderSizeFromClient + int(msg.DataSize)
	if len(data) < expectedTotalLength {
		return nil, fmt.Errorf("数据长度不匹配，期望%d字节，实际%d字节", expectedTotalLength, len(data))
	}

	// 读取Data (偏移9，n字节)
	if msg.DataSize > 0 {
		msg.Data = make([]byte, msg.DataSize)
		if _, err := io.ReadFull(reader, msg.Data); err != nil {
			return nil, fmt.Errorf("读取Data失败: %w", err)
		}
	} else {
		msg.Data = []byte{}
	}

	if p.Debug {
		fmt.Printf("[Protocol] 解码客户端消息: Cmd=%d, Flag=%d, DataSize=%d, DataStatus=%d\n",
			msg.Cmd, msg.Flag, msg.DataSize, msg.DataStatus)
		if msg.DataSize > 0 {
			fmt.Printf("[Protocol] 接收的原始数据 (前20字节): ")
			debugLen := len(data)
			if debugLen > 20 {
				debugLen = 20
			}
			for i := 0; i < debugLen; i++ {
				if i > 0 {
					fmt.Printf(",")
				}
				fmt.Printf("%d", data[i])
			}
			fmt.Printf("\n")
			fmt.Printf("[Protocol] Data部分内容: %x\n", msg.Data)
			if msg.DataSize < 100 {
				fmt.Printf("[Protocol] Data(string): %s\n", string(msg.Data))
			}
		}
	}

	return msg, nil
}

// EncodeServerMessage 编码服务端消息（11字节头）
// 服务端格式：ErrorID(2) + DataSize(2) + DataStatus(1) + Flag(4) + Cmd(2) + Data(n)
func (p *Protocol) EncodeServerMessage(msg *ServerMessage) ([]byte, error) {
	// 计算总长度
	totalLength := HeaderSizeToClient + len(msg.Data)
	buf := bytes.NewBuffer(make([]byte, 0, totalLength))

	// 写入ErrorID (偏移0，2字节)
	if err := binary.Write(buf, binary.BigEndian, msg.ErrorID); err != nil {
		return nil, fmt.Errorf("写入ErrorID失败: %w", err)
	}

	// 写入DataSize (偏移2，2字节)
	dataSize := uint16(len(msg.Data))
	if err := binary.Write(buf, binary.BigEndian, dataSize); err != nil {
		return nil, fmt.Errorf("写入DataSize失败: %w", err)
	}

	// 写入DataStatus (偏移4，1字节)
	if err := binary.Write(buf, binary.BigEndian, msg.DataStatus); err != nil {
		return nil, fmt.Errorf("写入DataStatus失败: %w", err)
	}

	// 写入Flag (偏移5，4字节)
	if err := binary.Write(buf, binary.BigEndian, msg.Flag); err != nil {
		return nil, fmt.Errorf("写入Flag失败: %w", err)
	}

	// 写入Cmd (偏移9，2字节)
	if err := binary.Write(buf, binary.BigEndian, msg.Cmd); err != nil {
		return nil, fmt.Errorf("写入Cmd失败: %w", err)
	}

	// 写入Data (偏移11，n字节)
	if len(msg.Data) > 0 {
		if _, err := buf.Write(msg.Data); err != nil {
			return nil, fmt.Errorf("写入Data失败: %w", err)
		}
	}

	if p.Debug {
		fmt.Printf("[Protocol] 编码服务端消息: Cmd=%d, Flag=%d, ErrorID=%d, DataSize=%d, DataStatus=%d\n",
			msg.Cmd, msg.Flag, msg.ErrorID, dataSize, msg.DataStatus)
		if len(msg.Data) > 0 {
			fmt.Printf("[Protocol] Data: %x\n", msg.Data)
			fmt.Printf("[Protocol] Data(string): %s\n", string(msg.Data))
		}
		// 打印前20字节的原始数据
		encoded := buf.Bytes()
		debugLen := len(encoded)
		if debugLen > 30 {
			debugLen = 30
		}
		fmt.Printf("[Protocol] 编码后的原始字节 (前%d字节): ", debugLen)
		for i := 0; i < debugLen; i++ {
			if i > 0 {
				fmt.Printf(",")
			}
			fmt.Printf("%d", encoded[i])
		}
		fmt.Printf("\n")
	}

	return buf.Bytes(), nil
}

// CreateErrorResponse 创建错误响应消息
func (p *Protocol) CreateErrorResponse(cmd uint16, flag uint32, errorID uint16, errorMsg string) *ServerMessage {
	return &ServerMessage{
		ErrorID:    errorID,
		DataStatus: 0,
		Flag:       flag,
		Cmd:        cmd,
		Data:       []byte(errorMsg),
	}
}

// CreateSuccessResponse 创建成功响应消息
func (p *Protocol) CreateSuccessResponse(cmd uint16, flag uint32, data []byte) *ServerMessage {
	return &ServerMessage{
		ErrorID:    0,
		DataStatus: 0,
		Flag:       flag,
		Cmd:        cmd,
		Data:       data,
	}
}

// ParseRawMessage 解析原始字节数据（用于判断消息来源）
func (p *Protocol) ParseRawMessage(data []byte) (isClientMessage bool, cmd uint16, err error) {
	// 尝试按客户端格式解析（9字节头）
	if len(data) >= HeaderSizeFromClient {
		// 客户端消息：Cmd在偏移7的位置
		reader := bytes.NewReader(data[7:9])
		var cmdValue uint16
		if err := binary.Read(reader, binary.BigEndian, &cmdValue); err == nil {
			return true, cmdValue, nil
		}
	}

	// 尝试按服务端格式解析（11字节头）
	if len(data) >= HeaderSizeToClient {
		// 服务端消息：Cmd在偏移9的位置
		reader := bytes.NewReader(data[9:11])
		var cmdValue uint16
		if err := binary.Read(reader, binary.BigEndian, &cmdValue); err == nil {
			return false, cmdValue, nil
		}
	}

	return false, 0, fmt.Errorf("无法解析消息格式")
}

// ValidateClientMessage 验证客户端消息的有效性
func (p *Protocol) ValidateClientMessage(data []byte) error {
	if len(data) < HeaderSizeFromClient {
		return fmt.Errorf("消息头长度不足")
	}

	// 读取DataSize
	reader := bytes.NewReader(data[0:2])
	var dataSize uint16
	if err := binary.Read(reader, binary.BigEndian, &dataSize); err != nil {
		return fmt.Errorf("无法读取DataSize: %w", err)
	}

	// 验证总长度
	expectedLength := HeaderSizeFromClient + int(dataSize)
	if len(data) != expectedLength {
		return fmt.Errorf("消息长度不匹配，期望%d，实际%d", expectedLength, len(data))
	}

	return nil
}

// GetMessageInfo 获取消息的基本信息（用于日志）
func (p *Protocol) GetMessageInfo(msg interface{}) string {
	switch m := msg.(type) {
	case *ClientMessage:
		return fmt.Sprintf("ClientMsg[Cmd:%d, Flag:%d, DataSize:%d]", m.Cmd, m.Flag, m.DataSize)
	case *ServerMessage:
		return fmt.Sprintf("ServerMsg[Cmd:%d, Flag:%d, ErrorID:%d, DataSize:%d]",
			m.Cmd, m.Flag, m.ErrorID, m.DataSize)
	default:
		return "UnknownMsg"
	}
}