# Serial Test Examples

这个目录包含了串口测试相关的示例代码和工具。

## 目录结构

```
examples/testserial/
├── README.md                # 本文件
├── Makefile                 # 构建配置文件
├── cmd/                     # 可执行程序源码目录
│   ├── Acmlib.java         # ACM库Java实现
│   ├── android_simulator/  # Android模拟器
│   │   └── main.go
│   ├── hardware_diagnose/   # 硬件诊断工具
│   │   └── main.go
│   ├── protocol_tester/     # 协议测试器
│   │   └── main.go
│   ├── serial_diagnose/     # 串口诊断工具
│   │   └── main.go
│   ├── serial_tester/       # 串口测试器
│   │   └── main.go
│   ├── stm32_protocol/      # STM32协议实现
│   │   └── main.go
│   └── stm32_simulator/     # STM32模拟器
│       └── main.go
├── bin/                     # 可执行文件目录（构建输出）
│   ├── protocol_tester_linux_arm
│   ├── serial_diagnose_linux_arm
│   └── serial_tester_linux_arm
├── scripts/                 # 脚本文件目录
│   ├── deploy.sh           # 部署脚本
│   ├── integration_test.sh # 集成测试脚本
│   └── quick_test.sh       # 快速测试脚本
├── tests/                   # 测试文件目录
│   └── serial_test.go      # 串口测试用例
└── docs/                    # 文档目录
    ├── README.md           # 详细说明文档
    ├── README_DEPLOY.md    # 部署说明文档
    └── STM32_Hardware_Protocol.md # STM32硬件协议文档
```

## 快速开始

1. 构建项目：
   ```bash
   make
   ```

2. 运行快速测试：
   ```bash
   ./scripts/quick_test.sh
   ```

3. 运行完整测试：
   ```bash
   ./scripts/integration_test.sh
   ```

## 文档

详细的文档位于 `docs/` 目录下：
- [详细说明](docs/README.md)
- [部署指南](docs/README_DEPLOY.md)
- [STM32硬件协议](docs/STM32_Hardware_Protocol.md)