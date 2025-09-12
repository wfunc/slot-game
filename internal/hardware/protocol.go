package hardware

import (
	"encoding/binary"
	"fmt"
	"time"
)

// 帧定义
const (
	FrameHeader byte   = 0xAA
	FrameTail   byte   = 0x55
	MinFrameLen uint16 = 9 // 最小帧长度：帧头(1) + 长度(2) + 命令(1) + 序列号(2) + CRC(2) + 帧尾(1)
)

// 命令码定义
const (
	// 硬件控制指令（Golang→STM32）
	CmdCoinDispense  byte = 0x01 // 上币控制
	CmdCoinRefund    byte = 0x02 // 退币控制
	CmdTicketPrint   byte = 0x03 // 彩票发放
	CmdPushControl   byte = 0x04 // 推币控制
	CmdLightControl  byte = 0x05 // 灯光控制

	// 硬件事件上报（STM32→Golang）
	EventCoinInserted  byte = 0x11 // 投币检测
	EventCoinReturned  byte = 0x12 // 回币检测
	EventButtonPressed byte = 0x13 // 按键事件
	EventSensorTriggered byte = 0x14 // 传感器事件

	// 状态管理
	CmdStatusQuery   byte = 0x21 // 状态查询
	EventStatusReport byte = 0x22 // 状态上报
	EventFaultReport  byte = 0x23 // 故障上报
	EventProgress     byte = 0x24 // 执行进度
	CmdFaultRecovery byte = 0x25 // 故障恢复

	// 系统指令
	CmdHeartbeat byte = 0x31 // 心跳包
	CmdACK       byte = 0x80 // ACK确认
	CmdNACK      byte = 0x81 // NACK拒绝
)

// 按键类型
const (
	KeyTypeGame   byte = 0x01 // 游戏按键
	KeyTypeConfig byte = 0x02 // 配置按键
)

// 游戏按键码
const (
	KeyStart        byte = 0x01 // 开始按键
	KeyRefundTicket byte = 0x02 // 退币/彩票按键
)

// 按键动作
const (
	KeyActionDown   byte = 0x01 // 按下
	KeyActionUp     byte = 0x02 // 释放
	KeyActionHold   byte = 0x03 // 长按
	KeyActionDouble byte = 0x04 // 双击
)

// 推币动作
const (
	PushActionContinuous byte = 0x01 // 启动连续推币
	PushActionStop       byte = 0x02 // 停止推币
	PushActionSingle     byte = 0x03 // 单次推币
	PushActionSetSpeed   byte = 0x04 // 设置推币速度
)

// 灯光位定义
const (
	LightOff  byte = 0x20 // 关
	Light1    byte = 0x22 // 灯1
	Light2    byte = 0x30 // 灯2
	Light3    byte = 0x24 // 灯3
	Light4    byte = 0x28 // 灯4
	Light5    byte = 0x21 // 灯5
	LightAll  byte = 0x7F // 所有灯
	LightNone byte = 0x40 // 所有灯关闭
)

// 传感器类型
const (
	SensorCoinLevel   byte = 0x01 // 币仓余量传感器
	SensorTicketLevel byte = 0x02 // 彩票余量传感器
	SensorTemperature byte = 0x03 // 温度传感器
	SensorVibration   byte = 0x04 // 震动传感器
	SensorDoor        byte = 0x05 // 门开关传感器
)

// 故障码定义
const (
	FaultCoinMotorStuck    byte = 0x01 // 上币电机卡死
	FaultRefundMotorFault  byte = 0x02 // 退币电机故障
	FaultTicketNoPaper     byte = 0x03 // 彩票机缺纸
	FaultTicketPaperJam    byte = 0x04 // 彩票机卡纸
	FaultPushMotorOverload byte = 0x05 // 推币电机过载
	FaultCoinJam           byte = 0x06 // 投币器卡币
	FaultOverTemperature   byte = 0x07 // 温度过高
	FaultSensorAbnormal    byte = 0x08 // 传感器异常
)

// 故障严重级别
const (
	FaultLevelInfo     byte = 0x01 // 提示
	FaultLevelWarning  byte = 0x02 // 警告
	FaultLevelError    byte = 0x03 // 错误
	FaultLevelCritical byte = 0x04 // 严重
)

// 故障恢复动作
const (
	RecoveryRestart    byte = 0x01 // 重启设备
	RecoveryClearFlag  byte = 0x02 // 清除故障标志
	RecoveryForceReset byte = 0x03 // 强制重置
	RecoveryRetry      byte = 0x04 // 重试操作
	RecoverySkip       byte = 0x05 // 跳过故障
)

// 状态码定义
const (
	StatusSuccess    byte = 0x00 // 成功接收，开始执行
	StatusCompleted  byte = 0x01 // 成功接收，已完成
	StatusQueued     byte = 0x02 // 成功接收，排队等待
	StatusInProgress byte = 0x00 // 执行中
	StatusCancelled  byte = 0x02 // 已取消
	StatusFailed     byte = 0x03 // 执行失败
)

// 错误码定义
const (
	ErrorUnsupported byte = 0x01 // 命令不支持
	ErrorInvalidParam byte = 0x02 // 参数错误
	ErrorBusy        byte = 0x03 // 设备忙
	ErrorHardware    byte = 0x04 // 硬件故障
	ErrorChecksum    byte = 0x05 // 校验失败
	ErrorOutOfRange  byte = 0x06 // 超出范围
	ErrorNoResource  byte = 0x07 // 资源不足
	
	// 补充的故障码（用于传感器事件）
	FaultNoResource  byte = 0x09 // 资源不足（币/彩票）
)

// Frame 数据帧结构
type Frame struct {
	Header   byte   // 帧头
	Length   uint16 // 长度
	Command  byte   // 命令码
	Sequence uint16 // 序列号
	Data     []byte // 数据
	CRC16    uint16 // CRC校验
	Tail     byte   // 帧尾
}

// DeviceStatus 设备状态结构
type DeviceStatus struct {
	CoinMotor     byte   // 上币电机状态
	ReturnMotor   byte   // 退币电机状态
	PushMotor     byte   // 推币电机状态
	TicketPrinter byte   // 彩票机状态
	CoinCount     uint16 // 币仓余量
	TicketCount   uint16 // 彩票余量
	Temperature   byte   // 设备温度
	ErrorFlags    byte   // 错误标志位
}

// CoinReturnData 回币数据（优化后的格式）
type CoinReturnData struct {
	FrontCount byte // 前方数量
	LeftCount  byte // 左侧数量
	RightCount byte // 右侧数量
}

// ButtonEvent 按键事件
type ButtonEvent struct {
	KeyType   byte   // 按键类型
	KeyCode   byte   // 按键码
	Action    byte   // 动作类型
	ExtraData []byte // 附加数据
}

// FaultEvent 故障事件
type FaultEvent struct {
	FaultCode  byte   // 故障码
	Level      byte   // 严重级别
	ExtraInfo  []byte // 附加信息
}

// ProgressReport 执行进度报告
type ProgressReport struct {
	OriginalCmd byte   // 原命令码
	Completed   uint16 // 已完成
	Total       uint16 // 总数
	Status      byte   // 状态
}

// NewFrame 创建新的数据帧
func NewFrame(cmd byte, seq uint16, data []byte) *Frame {
	f := &Frame{
		Header:   FrameHeader,
		Command:  cmd,
		Sequence: seq,
		Data:     data,
		Tail:     FrameTail,
	}
	
	// 计算长度（整个帧的长度）
	f.Length = uint16(9 + len(data)) // 9 = 帧头(1) + 长度(2) + 命令(1) + 序列号(2) + CRC(2) + 帧尾(1)
	
	// 计算CRC
	f.CRC16 = f.CalculateCRC()
	
	return f
}

// ToBytes 将帧转换为字节数组
func (f *Frame) ToBytes() []byte {
	buf := make([]byte, f.Length)
	idx := 0
	
	// 帧头
	buf[idx] = f.Header
	idx++
	
	// 长度（大端序）
	binary.BigEndian.PutUint16(buf[idx:], f.Length)
	idx += 2
	
	// 命令
	buf[idx] = f.Command
	idx++
	
	// 序列号（大端序）
	binary.BigEndian.PutUint16(buf[idx:], f.Sequence)
	idx += 2
	
	// 数据
	if len(f.Data) > 0 {
		copy(buf[idx:], f.Data)
		idx += len(f.Data)
	}
	
	// CRC16（大端序）
	binary.BigEndian.PutUint16(buf[idx:], f.CRC16)
	idx += 2
	
	// 帧尾
	buf[idx] = f.Tail
	
	return buf
}

// FromBytes 从字节数组解析帧
func (f *Frame) FromBytes(data []byte) error {
	if len(data) < int(MinFrameLen) {
		return fmt.Errorf("frame too short: %d < %d", len(data), MinFrameLen)
	}
	
	// 检查帧头
	if data[0] != FrameHeader {
		return fmt.Errorf("invalid frame header: 0x%02X", data[0])
	}
	
	// 解析长度
	f.Header = data[0]
	f.Length = binary.BigEndian.Uint16(data[1:3])
	
	// 检查数据长度
	if len(data) < int(f.Length) {
		return fmt.Errorf("incomplete frame: %d < %d", len(data), f.Length)
	}
	
	// 检查帧尾
	if data[f.Length-1] != FrameTail {
		return fmt.Errorf("invalid frame tail: 0x%02X", data[f.Length-1])
	}
	
	// 解析字段
	f.Command = data[3]
	f.Sequence = binary.BigEndian.Uint16(data[4:6])
	
	// 解析数据
	dataLen := f.Length - 9
	if dataLen > 0 {
		f.Data = make([]byte, dataLen)
		copy(f.Data, data[6:6+dataLen])
	}
	
	// 解析CRC
	crcIdx := f.Length - 3
	f.CRC16 = binary.BigEndian.Uint16(data[crcIdx:crcIdx+2])
	f.Tail = data[f.Length-1]
	
	// 验证CRC
	calcCRC := f.CalculateCRC()
	if calcCRC != f.CRC16 {
		return fmt.Errorf("CRC mismatch: calc=0x%04X, recv=0x%04X", calcCRC, f.CRC16)
	}
	
	return nil
}

// CalculateCRC 计算CRC16校验值
func (f *Frame) CalculateCRC() uint16 {
	// 计算从命令码到数据的CRC
	data := make([]byte, 0, 3+len(f.Data))
	data = append(data, f.Command)
	data = append(data, byte(f.Sequence>>8), byte(f.Sequence&0xFF))
	if len(f.Data) > 0 {
		data = append(data, f.Data...)
	}
	return CRC16XMODEM(data)
}

// CRC16XMODEM CRC16-XMODEM算法
func CRC16XMODEM(data []byte) uint16 {
	crc := uint16(0x0000)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// FormatTimestamp 格式化时间戳为4字节
func FormatTimestamp(t time.Time) []byte {
	unix := uint32(t.Unix())
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, unix)
	return buf
}

// ParseTimestamp 解析4字节时间戳
func ParseTimestamp(data []byte) time.Time {
	if len(data) < 4 {
		return time.Time{}
	}
	unix := binary.BigEndian.Uint32(data)
	return time.Unix(int64(unix), 0)
}