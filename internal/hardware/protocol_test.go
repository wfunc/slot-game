package hardware

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFrame_NewFrame(t *testing.T) {
	cmd := byte(CmdCoinDispense)
	seq := uint16(123)
	data := []byte{0x01, 0x02, 0x03}

	frame := NewFrame(cmd, seq, data)

	assert.Equal(t, FrameHeader, frame.Header)
	assert.Equal(t, cmd, frame.Command)
	assert.Equal(t, seq, frame.Sequence)
	assert.Equal(t, data, frame.Data)
	assert.Equal(t, FrameTail, frame.Tail)
	assert.Equal(t, uint16(12), frame.Length) // 9 + 3 bytes of data
	assert.NotZero(t, frame.CRC16)
}

func TestFrame_ToBytes(t *testing.T) {
	frame := &Frame{
		Header:   FrameHeader,
		Length:   11,
		Command:  CmdHeartbeat,
		Sequence: 1001,
		Data:     []byte{0xAB, 0xCD},
		Tail:     FrameTail,
	}
	frame.CRC16 = frame.CalculateCRC()

	bytes := frame.ToBytes()

	// Verify frame structure
	assert.Equal(t, byte(0xAA), bytes[0]) // Header
	assert.Equal(t, uint16(11), binary.BigEndian.Uint16(bytes[1:3])) // Length
	assert.Equal(t, byte(CmdHeartbeat), bytes[3]) // Command
	assert.Equal(t, uint16(1001), binary.BigEndian.Uint16(bytes[4:6])) // Sequence
	assert.Equal(t, byte(0xAB), bytes[6]) // Data[0]
	assert.Equal(t, byte(0xCD), bytes[7]) // Data[1]
	// CRC at bytes[8:10]
	assert.Equal(t, byte(0x55), bytes[10]) // Tail
}

func TestFrame_FromBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid frame",
			data: func() []byte {
				f := NewFrame(CmdACK, 100, []byte{0x01, 0x02})
				return f.ToBytes()
			}(),
			wantErr: false,
		},
		{
			name:    "too short",
			data:    []byte{0xAA, 0x00},
			wantErr: true,
			errMsg:  "frame too short",
		},
		{
			name:    "invalid header",
			data:    []byte{0xFF, 0x00, 0x09, 0x01, 0x00, 0x64, 0x00, 0x00, 0x55},
			wantErr: true,
			errMsg:  "invalid frame header",
		},
		{
			name: "invalid tail",
			data: func() []byte {
				f := NewFrame(CmdACK, 100, nil)
				b := f.ToBytes()
				b[len(b)-1] = 0xFF // Corrupt tail
				return b
			}(),
			wantErr: true,
			errMsg:  "invalid frame tail",
		},
		{
			name: "invalid CRC",
			data: func() []byte {
				f := NewFrame(CmdACK, 100, nil)
				b := f.ToBytes()
				b[len(b)-3] = 0xFF // Corrupt CRC
				return b
			}(),
			wantErr: true,
			errMsg:  "CRC mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &Frame{}
			err := frame.FromBytes(tt.data)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCRC16XMODEM(t *testing.T) {
	tests := []struct {
		data     []byte
		expected uint16
	}{
		{[]byte{0x01, 0x02, 0x03}, 0x6131},  // Corrected CRC value
		{[]byte{0xFF, 0xFF, 0xFF}, 0xD26C},  // Corrected CRC value
		{[]byte{}, 0x0000},
		{[]byte{0x48, 0x65, 0x6C, 0x6C, 0x6F}, 0xCBD6}, // "Hello" - Corrected CRC value
	}

	for _, tt := range tests {
		result := CRC16XMODEM(tt.data)
		assert.Equal(t, tt.expected, result, "CRC mismatch for data %v", tt.data)
	}
}

func TestFormatTimestamp(t *testing.T) {
	testTime := time.Unix(1234567890, 0)
	bytes := FormatTimestamp(testTime)
	
	assert.Equal(t, 4, len(bytes))
	assert.Equal(t, uint32(1234567890), binary.BigEndian.Uint32(bytes))
}

func TestParseTimestamp(t *testing.T) {
	testTime := time.Unix(1234567890, 0)
	bytes := FormatTimestamp(testTime)
	
	parsed := ParseTimestamp(bytes)
	assert.Equal(t, testTime.Unix(), parsed.Unix())
	
	// Test with invalid data
	invalidParsed := ParseTimestamp([]byte{0x01})
	assert.True(t, invalidParsed.IsZero())
}

func TestCoinReturnData(t *testing.T) {
	data := &CoinReturnData{
		FrontCount: 10,
		LeftCount:  5,
		RightCount: 3,
	}
	
	assert.Equal(t, byte(10), data.FrontCount)
	assert.Equal(t, byte(5), data.LeftCount)
	assert.Equal(t, byte(3), data.RightCount)
}

func TestButtonEvent(t *testing.T) {
	event := &ButtonEvent{
		KeyType:   KeyTypeGame,
		KeyCode:   KeyStart,
		Action:    KeyActionDown,
		ExtraData: []byte{0x01, 0x02},
	}
	
	assert.Equal(t, KeyTypeGame, event.KeyType)
	assert.Equal(t, KeyStart, event.KeyCode)
	assert.Equal(t, KeyActionDown, event.Action)
	assert.Equal(t, []byte{0x01, 0x02}, event.ExtraData)
}

func TestFaultEvent(t *testing.T) {
	event := &FaultEvent{
		FaultCode: FaultCoinMotorStuck,
		Level:     FaultLevelError,
		ExtraInfo: []byte{0xAB},
	}
	
	assert.Equal(t, FaultCoinMotorStuck, event.FaultCode)
	assert.Equal(t, FaultLevelError, event.Level)
	assert.Equal(t, []byte{0xAB}, event.ExtraInfo)
}

func TestProgressReport(t *testing.T) {
	report := &ProgressReport{
		OriginalCmd: CmdCoinDispense,
		Completed:   50,
		Total:       100,
		Status:      StatusInProgress,
	}
	
	assert.Equal(t, byte(CmdCoinDispense), report.OriginalCmd)
	assert.Equal(t, uint16(50), report.Completed)
	assert.Equal(t, uint16(100), report.Total)
	assert.Equal(t, StatusInProgress, report.Status)
}

func TestDeviceStatus(t *testing.T) {
	status := &DeviceStatus{
		CoinMotor:     0x01,
		ReturnMotor:   0x02,
		PushMotor:     0x03,
		TicketPrinter: 0x04,
		CoinCount:     100,
		TicketCount:   50,
		Temperature:   25,
		ErrorFlags:    0x00,
	}
	
	assert.Equal(t, byte(0x01), status.CoinMotor)
	assert.Equal(t, byte(0x02), status.ReturnMotor)
	assert.Equal(t, byte(0x03), status.PushMotor)
	assert.Equal(t, byte(0x04), status.TicketPrinter)
	assert.Equal(t, uint16(100), status.CoinCount)
	assert.Equal(t, uint16(50), status.TicketCount)
	assert.Equal(t, byte(25), status.Temperature)
	assert.Equal(t, byte(0x00), status.ErrorFlags)
}

// Benchmark tests
func BenchmarkCRC16XMODEM(b *testing.B) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CRC16XMODEM(data)
	}
}

func BenchmarkFrame_ToBytes(b *testing.B) {
	frame := NewFrame(CmdCoinDispense, 1000, []byte{0x01, 0x02, 0x03, 0x04})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame.ToBytes()
	}
}

func BenchmarkFrame_FromBytes(b *testing.B) {
	data := NewFrame(CmdCoinDispense, 1000, []byte{0x01, 0x02, 0x03, 0x04}).ToBytes()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame := &Frame{}
		frame.FromBytes(data)
	}
}