package tests

import (
	"testing"
)

// 示例测试函数
func TestExample(t *testing.T) {
	// 这是一个示例测试
	// 实际项目中应该测试真实的功能
	result := 2 + 2
	expected := 4
	
	if result != expected {
		t.Errorf("计算错误: 期望 %d, 得到 %d", expected, result)
	}
}

// 测试串口配置验证
func TestSerialConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		baudRate int
		valid    bool
	}{
		{"标准波特率9600", 9600, true},
		{"标准波特率115200", 115200, true},
		{"无效波特率", 0, false},
		{"负数波特率", -1, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证波特率是否有效
			isValid := tt.baudRate > 0 && (tt.baudRate == 9600 || tt.baudRate == 115200 || 
				tt.baudRate == 19200 || tt.baudRate == 38400 || tt.baudRate == 57600)
			
			if isValid != tt.valid {
				t.Errorf("测试 %s 失败: 期望 %v, 得到 %v", tt.name, tt.valid, isValid)
			}
		})
	}
}