package websocket

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

// TestClientMessageDecode 测试客户端消息解码
func TestClientMessageDecode(t *testing.T) {
	protocol := NewProtocol()
	protocol.Debug = true

	// 准备测试数据
	testData := []byte(`{"action":"login","username":"test"}`)

	// 构建客户端消息（9字节头）
	buf := new(bytes.Buffer)

	// DataSize (2字节)
	binary.Write(buf, binary.BigEndian, uint16(len(testData)))
	// DataStatus (1字节)
	binary.Write(buf, binary.BigEndian, uint8(0))
	// Flag (4字节)
	binary.Write(buf, binary.BigEndian, uint32(12345))
	// Cmd (2字节)
	binary.Write(buf, binary.BigEndian, uint16(1001))
	// Data
	buf.Write(testData)

	// 解码消息
	msg, err := protocol.DecodeClientMessage(buf.Bytes())
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	// 验证结果
	if msg.Cmd != 1001 {
		t.Errorf("Cmd不匹配，期望1001，实际%d", msg.Cmd)
	}
	if msg.Flag != 12345 {
		t.Errorf("Flag不匹配，期望12345，实际%d", msg.Flag)
	}
	if msg.DataSize != uint16(len(testData)) {
		t.Errorf("DataSize不匹配，期望%d，实际%d", len(testData), msg.DataSize)
	}
	if !bytes.Equal(msg.Data, testData) {
		t.Errorf("Data不匹配，期望%s，实际%s", string(testData), string(msg.Data))
	}

	t.Logf("解码成功: Cmd=%d, Flag=%d, DataSize=%d, Data=%s",
		msg.Cmd, msg.Flag, msg.DataSize, string(msg.Data))
}

// TestServerMessageEncode 测试服务端消息编码
func TestServerMessageEncode(t *testing.T) {
	protocol := NewProtocol()
	protocol.Debug = true

	// 准备测试数据
	responseData := map[string]interface{}{
		"status":  "success",
		"message": "登录成功",
		"user_id": 12345,
	}
	testData, _ := json.Marshal(responseData)

	// 创建服务端消息
	msg := &ServerMessage{
		ErrorID:    0,
		DataStatus: 0,
		Flag:       67890,
		Cmd:        1001,
		Data:       testData,
	}

	// 编码消息
	encoded, err := protocol.EncodeServerMessage(msg)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	// 验证长度
	expectedLen := HeaderSizeToClient + len(testData)
	if len(encoded) != expectedLen {
		t.Errorf("编码长度不匹配，期望%d，实际%d", expectedLen, len(encoded))
	}

	// 解析编码结果验证
	reader := bytes.NewReader(encoded)

	var errorID uint16
	binary.Read(reader, binary.BigEndian, &errorID)
	if errorID != 0 {
		t.Errorf("ErrorID不匹配，期望0，实际%d", errorID)
	}

	var dataSize uint16
	binary.Read(reader, binary.BigEndian, &dataSize)
	if dataSize != uint16(len(testData)) {
		t.Errorf("DataSize不匹配，期望%d，实际%d", len(testData), dataSize)
	}

	var dataStatus uint8
	binary.Read(reader, binary.BigEndian, &dataStatus)
	if dataStatus != 0 {
		t.Errorf("DataStatus不匹配，期望0，实际%d", dataStatus)
	}

	var flag uint32
	binary.Read(reader, binary.BigEndian, &flag)
	if flag != 67890 {
		t.Errorf("Flag不匹配，期望67890，实际%d", flag)
	}

	var cmd uint16
	binary.Read(reader, binary.BigEndian, &cmd)
	if cmd != 1001 {
		t.Errorf("Cmd不匹配，期望1001，实际%d", cmd)
	}

	// 读取数据部分
	data := make([]byte, dataSize)
	reader.Read(data)
	if !bytes.Equal(data, testData) {
		t.Errorf("Data不匹配，期望%s，实际%s", string(testData), string(data))
	}

	t.Logf("编码成功: 总长度=%d, ErrorID=%d, DataSize=%d, Flag=%d, Cmd=%d",
		len(encoded), errorID, dataSize, flag, cmd)
}

// TestEmptyMessage 测试空消息
func TestEmptyMessage(t *testing.T) {
	protocol := NewProtocol()

	// 测试客户端空消息
	clientBuf := new(bytes.Buffer)
	binary.Write(clientBuf, binary.BigEndian, uint16(0))  // DataSize = 0
	binary.Write(clientBuf, binary.BigEndian, uint8(0))   // DataStatus
	binary.Write(clientBuf, binary.BigEndian, uint32(0))  // Flag
	binary.Write(clientBuf, binary.BigEndian, uint16(1002)) // Cmd (心跳)

	clientMsg, err := protocol.DecodeClientMessage(clientBuf.Bytes())
	if err != nil {
		t.Fatalf("解码空消息失败: %v", err)
	}
	if len(clientMsg.Data) != 0 {
		t.Errorf("空消息Data长度应该为0，实际%d", len(clientMsg.Data))
	}

	// 测试服务端空消息
	serverMsg := &ServerMessage{
		ErrorID:    0,
		DataStatus: 0,
		Flag:       0,
		Cmd:        1002,
		Data:       []byte{},
	}

	encoded, err := protocol.EncodeServerMessage(serverMsg)
	if err != nil {
		t.Fatalf("编码空消息失败: %v", err)
	}
	if len(encoded) != HeaderSizeToClient {
		t.Errorf("空消息长度应该为%d，实际%d", HeaderSizeToClient, len(encoded))
	}
}

// TestErrorResponse 测试错误响应
func TestErrorResponse(t *testing.T) {
	protocol := NewProtocol()

	// 创建错误响应
	errMsg := protocol.CreateErrorResponse(1001, 12345, 2001, "参数错误")

	// 编码
	encoded, err := protocol.EncodeServerMessage(errMsg)
	if err != nil {
		t.Fatalf("编码错误响应失败: %v", err)
	}

	// 解析验证
	reader := bytes.NewReader(encoded)

	var errorID uint16
	binary.Read(reader, binary.BigEndian, &errorID)
	if errorID != 2001 {
		t.Errorf("ErrorID不匹配，期望2001，实际%d", errorID)
	}

	t.Logf("错误响应创建成功: ErrorID=%d, Cmd=%d, Message=%s",
		errMsg.ErrorID, errMsg.Cmd, string(errMsg.Data))
}

// TestProtocolCompatibility 测试协议兼容性
func TestProtocolCompatibility(t *testing.T) {
	protocol := NewProtocol()

	// 模拟前端发来的数据
	// 根据前端代码：dataView.setUint16(0, dataSize);
	// dataView.setUint8(2, 0);
	// dataView.setUint32(3, flag_id);
	// dataView.setUint16(7, cmd);

	testCases := []struct {
		name     string
		cmd      uint16
		flag     uint32
		data     []byte
	}{
		{"登录请求", 1001, 123, []byte(`{"username":"test","password":"123456"}`)},
		{"心跳包", 1002, 0, []byte{}},
		{"游戏请求", 2001, 456, []byte(`{"bet":100,"lines":20}`)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 构建客户端消息（模拟前端）
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.BigEndian, uint16(len(tc.data))) // DataSize
			binary.Write(buf, binary.BigEndian, uint8(0))              // DataStatus
			binary.Write(buf, binary.BigEndian, tc.flag)               // Flag
			binary.Write(buf, binary.BigEndian, tc.cmd)                // Cmd
			buf.Write(tc.data)                                         // Data

			// 解码
			msg, err := protocol.DecodeClientMessage(buf.Bytes())
			if err != nil {
				t.Fatalf("解码失败: %v", err)
			}

			// 验证
			if msg.Cmd != tc.cmd {
				t.Errorf("Cmd不匹配，期望%d，实际%d", tc.cmd, msg.Cmd)
			}
			if msg.Flag != tc.flag {
				t.Errorf("Flag不匹配，期望%d，实际%d", tc.flag, msg.Flag)
			}
			if !bytes.Equal(msg.Data, tc.data) {
				t.Errorf("Data不匹配")
			}

			// 创建响应（模拟服务端）
			response := protocol.CreateSuccessResponse(tc.cmd, tc.flag, []byte(`{"status":"ok"}`))

			// 编码响应
			encoded, err := protocol.EncodeServerMessage(response)
			if err != nil {
				t.Fatalf("编码响应失败: %v", err)
			}

			// 验证响应格式（前端期望的格式）
			// ErrorID(2) + DataSize(2) + DataStatus(1) + Flag(4) + Cmd(2) + Data
			if len(encoded) < HeaderSizeToClient {
				t.Errorf("响应长度不足")
			}

			t.Logf("%s - 请求: Cmd=%d, Flag=%d, Data=%s", tc.name, msg.Cmd, msg.Flag, string(msg.Data))
			t.Logf("%s - 响应: 长度=%d", tc.name, len(encoded))
		})
	}
}

// BenchmarkDecode 性能测试 - 解码
func BenchmarkDecode(b *testing.B) {
	protocol := NewProtocol()

	// 准备测试数据
	buf := new(bytes.Buffer)
	testData := []byte(`{"action":"test","value":12345}`)
	binary.Write(buf, binary.BigEndian, uint16(len(testData)))
	binary.Write(buf, binary.BigEndian, uint8(0))
	binary.Write(buf, binary.BigEndian, uint32(12345))
	binary.Write(buf, binary.BigEndian, uint16(1001))
	buf.Write(testData)

	data := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = protocol.DecodeClientMessage(data)
	}
}

// BenchmarkEncode 性能测试 - 编码
func BenchmarkEncode(b *testing.B) {
	protocol := NewProtocol()

	msg := &ServerMessage{
		ErrorID:    0,
		DataStatus: 0,
		Flag:       12345,
		Cmd:        1001,
		Data:       []byte(`{"status":"success","data":{"id":123}}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = protocol.EncodeServerMessage(msg)
	}
}