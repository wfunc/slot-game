package hardware

import "io"

// SerialPort 串口接口（用于测试）
type SerialPort interface {
	io.ReadWriteCloser
	Flush() error
}