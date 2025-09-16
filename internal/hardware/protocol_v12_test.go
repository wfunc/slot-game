package hardware

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestFrameV12 测试v1.2协议的帧格式
func TestFrameV12(t *testing.T) {
	tests := []struct {
		name     string
		cmd      byte
		seq      uint16
		data     []byte
		wantLen  uint8
	}{
		{
			name:    "上币命令",
			cmd:     CmdCoinDispense,
			seq:     0x0001,
			data:    []byte{0x14, 0x00, 0x05}, // 20个币(小端序), 速度5
			wantLen: 10, // 7 + 3
		},
		{
			name:    "心跳包",
			cmd:     CmdHeartbeat,
			seq:     0x0003,
			data:    []byte{0x01, 0x02, 0x03, 0x04}, // 时间戳(小端序)
			wantLen: 11, // 7 + 4
		},
		{
			name:    "最小帧",
			cmd:     CmdStatusQuery,
			seq:     0x0005,
			data:    []byte{},
			wantLen: 7, // 最小帧长度
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建帧
			frame := NewFrame(tt.cmd, tt.seq, tt.data)

			// 验证长度
			if frame.Length != tt.wantLen {
				t.Errorf("Length = %d, want %d", frame.Length, tt.wantLen)
			}

			// 转换为字节
			buf := frame.ToBytes()

			// 验证帧头帧尾
			if buf[0] != FrameHeader {
				t.Errorf("Header = 0x%02X, want 0x%02X", buf[0], FrameHeader)
			}
			if buf[len(buf)-1] != FrameTail {
				t.Errorf("Tail = 0x%02X, want 0x%02X", buf[len(buf)-1], FrameTail)
			}

			// 验证长度字段(v1.2: 1字节)
			if buf[1] != tt.wantLen {
				t.Errorf("Length field = %d, want %d", buf[1], tt.wantLen)
			}

			// 验证序列号(v1.2: 小端序)
			seqBytes := buf[3:5]
			gotSeq := binary.LittleEndian.Uint16(seqBytes)
			if gotSeq != tt.seq {
				t.Errorf("Sequence = 0x%04X, want 0x%04X", gotSeq, tt.seq)
			}

			// 验证XOR校验
			xorIdx := len(buf) - 2
			gotXOR := buf[xorIdx]
			wantXOR := calculateXORFromBytes(buf[:xorIdx])
			if gotXOR != wantXOR {
				t.Errorf("XOR = 0x%02X, want 0x%02X", gotXOR, wantXOR)
			}
		})
	}
}

// TestXORCalculation 测试XOR校验算法
func TestXORCalculation(t *testing.T) {
	frame := &Frame{
		Header:   FrameHeader,
		Length:   7,
		Command:  0x01,
		Sequence: 0x1234,
		Data:     []byte{},
		Tail:     FrameTail,
	}

	xor := frame.CalculateXOR()

	// 手动计算期望值
	expected := uint8(0)
	expected ^= FrameHeader
	expected ^= 7
	expected ^= 0x01
	expected ^= 0x34 // 序列号低字节
	expected ^= 0x12 // 序列号高字节

	if xor != expected {
		t.Errorf("CalculateXOR() = 0x%02X, want 0x%02X", xor, expected)
	}
}

// TestFromBytesV12 测试从字节数组解析v1.2帧
func TestFromBytesV12(t *testing.T) {
	// 构造一个完整的帧数据
	data := []byte{
		0xAA,       // 帧头
		0x0A,       // 长度(10字节)
		0x01,       // 命令(上币)
		0x01, 0x00, // 序列号(小端序: 0x0001)
		0x14, 0x00, // 数据: 20个币(小端序)
		0x05,       // 数据: 速度5
		0x00,       // XOR (需要计算)
		0x55,       // 帧尾
	}

	// 计算正确的XOR
	xor := calculateXORFromBytes(data[:8])
	data[8] = xor

	frame := &Frame{}
	err := frame.FromBytes(data)

	if err != nil {
		t.Fatalf("FromBytes() error = %v", err)
	}

	if frame.Command != 0x01 {
		t.Errorf("Command = 0x%02X, want 0x01", frame.Command)
	}

	if frame.Sequence != 0x0001 {
		t.Errorf("Sequence = 0x%04X, want 0x0001", frame.Sequence)
	}

	if !bytes.Equal(frame.Data, []byte{0x14, 0x00, 0x05}) {
		t.Errorf("Data = %v, want [0x14, 0x00, 0x05]", frame.Data)
	}
}

// TestEchoMechanism 测试Echo确认机制
func TestEchoMechanism(t *testing.T) {
	// 创建原始命令帧
	originalFrame := NewFrame(CmdCoinDispense, 0x0001, []byte{0x0A, 0x00})

	// 模拟Echo响应(原封不动返回)
	echoFrame := &Frame{
		Header:   originalFrame.Header,
		Length:   originalFrame.Length,
		Command:  originalFrame.Command,
		Sequence: originalFrame.Sequence,
		Data:     originalFrame.Data,
		XOR:      originalFrame.XOR,
		Tail:     originalFrame.Tail,
	}

	// 验证Echo帧与原始帧完全相同
	if !bytes.Equal(originalFrame.ToBytes(), echoFrame.ToBytes()) {
		t.Error("Echo frame doesn't match original frame")
	}
}

// TestLittleEndianConversion 测试小端序转换
func TestLittleEndianConversion(t *testing.T) {
	tests := []struct {
		name  string
		value uint16
		want  []byte
	}{
		{"值1", 0x0001, []byte{0x01, 0x00}},
		{"值20", 0x0014, []byte{0x14, 0x00}},
		{"值1234", 0x1234, []byte{0x34, 0x12}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, 2)
			binary.LittleEndian.PutUint16(buf, tt.value)

			if !bytes.Equal(buf, tt.want) {
				t.Errorf("LittleEndian.PutUint16(%04X) = %v, want %v",
					tt.value, buf, tt.want)
			}

			// 反向验证
			got := binary.LittleEndian.Uint16(tt.want)
			if got != tt.value {
				t.Errorf("LittleEndian.Uint16(%v) = %04X, want %04X",
					tt.want, got, tt.value)
			}
		})
	}
}

// 辅助函数：从字节数组计算XOR
func calculateXORFromBytes(data []byte) uint8 {
	var xor uint8 = 0
	for _, b := range data {
		xor ^= b
	}
	return xor
}