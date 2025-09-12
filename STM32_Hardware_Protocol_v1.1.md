# STM32硬件控制通信协议文档 v1.1

## 文档信息

**版本**：v1.1  
**更新日期**：2025-09-12  
**适用对象**：STM32硬件工程师、Golang后端开发工程师  
**上位机环境**：Ubuntu + Golang  
**通信方式**：串口 (UART)  
**协议特点**：Golang主导，STM32被动响应，无状态设计

## 更新日志

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v1.0 | 2025-09-12 | 初始版本 |
| v1.1 | 2025-09-12 | 新增序列号管理规则、异常恢复流程、时序规范、并发控制、数据持久化策略 |

---

## 第一部分：快速开始

### 系统架构概览

```
┌─────────────────────────────────────────────────┐
│         Golang系统（Ubuntu）[主导方]              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ 游戏逻辑  │  │ 账目管理  │  │ MQTT通信 │      │
│  └──────────┘  └──────────┘  └──────────┘      │
│         ↓            ↓            ↓              │
│  ┌────────────────────────────────────┐         │
│  │      硬件控制协议层 (本文档)         │         │
│  └────────────────────────────────────┘         │
└─────────────────↓↑──────────────────────────────┘
              串口通信
              115200bps
┌─────────────────↓↑──────────────────────────────┐
│         STM32控制板 [被动执行方]                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ 电机控制  │  │ 传感器   │  │ 灯光控制  │      │
│  └──────────┘  └──────────┘  └──────────┘      │
│         ↓            ↓            ↓              │
│  ┌────────────────────────────────────┐         │
│  │         硬件设备层                  │         │
│  └────────────────────────────────────┘         │
└─────────────────────────────────────────────────┘
```

### 最小实现示例

#### STM32端最小实现
```c
// 1. 初始化串口
void init_uart() {
    // 配置: 115200, 8N2, 无流控
    HAL_UART_Init(&huart3);
}

// 2. 接收处理
void process_command(uint8_t cmd, uint8_t* data) {
    switch(cmd) {
        case 0x01: // 上币控制
            uint16_t count = (data[0] << 8) | data[1];
            dispense_coins(count);
            send_ack(0x00); // 成功
            break;
        case 0x31: // 心跳
            send_heartbeat_response();
            break;
    }
}

// 3. 事件上报
void on_coin_inserted() {
    uint8_t data[1] = {1}; // 1个币
    send_event(0x11, data, 1);
}
```

#### Golang端最小实现
```go
// 1. 连接串口
port, _ := serial.OpenPort(&serial.Config{
    Name: "/dev/ttyS3",
    Baud: 115200,
})

// 2. 发送命令
func dispenseCoins(count uint16) {
    data := make([]byte, 2)
    binary.BigEndian.PutUint16(data, count)
    sendCommand(0x01, data)
}

// 3. 处理事件
func handleEvent(cmd byte, data []byte) {
    switch cmd {
    case 0x11: // 投币
        coinCount := data[0]
        gameLogic.AddCredits(coinCount)
    }
}
```

### 常见问题速查

| 问题 | 解决方案 | 参考章节 |
|------|----------|----------|
| 串口连接失败 | 检查端口权限、波特率配置 | 2.1 |
| 命令无响应 | 检查序列号、CRC校验 | 2.2, 2.4 |
| 投币检测延迟 | 优化中断优先级 | 8.4 |
| 数据丢失 | 启用持久化策略 | 9.3 |
| 通信中断 | 查看异常恢复流程 | 7.5 |

---

## 第二部分：协议规范

## 1. 概述

### 1.1 系统架构
```
Golang系统（Ubuntu）[主导] <-串口通信-> STM32[被动执行]
         |                                    |
    游戏逻辑/MQTT                    硬件控制/传感器
```

### 1.2 职责划分
- **Golang系统（主导方）**：
  - 游戏逻辑处理
  - 账目管理
  - MQTT云端通信
  - 发送硬件控制指令
  - 接收硬件状态上报

- **STM32（被动方）**：
  - 执行硬件控制指令
  - 检测硬件传感器状态
  - 上报硬件事件
  - 故障检测和报警

### 1.3 设计原则
- **简单可靠**：STM32只负责硬件层，不涉及业务逻辑
- **响应式设计**：STM32被动响应Golang指令
- **实时反馈**：硬件事件立即上报
- **防重机制**：序列号确保指令不重复执行

## 2. 通信规范

### 2.1 串口配置
```
端口：/dev/ttyS3 (Ubuntu) / UART3 (STM32)
波特率：115200 bps
数据位：8位
停止位：2位
奇偶校验：无
流控：无
```

### 2.2 数据帧格式
```
[帧头][长度][命令][序列号][数据][CRC16][帧尾]
[0xAA][uint16][uint8][uint16][N字节][uint16][0x55]
```

**字段说明**：
- **帧头**：0xAA（固定）
- **长度**：uint16（大端序），整个帧的长度
- **命令**：uint8，命令码
- **序列号**：uint16（大端序），防重复
  - Golang发送：奇数（0x0001, 0x0003...）
  - STM32上报：偶数（0x0002, 0x0004...）
- **数据**：可变长度，具体数据
- **CRC16**：uint16（大端序），校验码
- **帧尾**：0x55（固定）

### 2.3 命令码定义

| 命令码 | 功能 | 方向 | 说明 |
|--------|------|------|------|
| **硬件控制指令（Golang→STM32）** |
| 0x01 | 上币控制 | Go→STM32 | 控制上币电机 |
| 0x02 | 退币控制 | Go→STM32 | 控制退币电机 |
| 0x03 | 彩票发放 | Go→STM32 | 控制彩票机 |
| 0x04 | 推币控制 | Go→STM32 | 控制推币电机 |
| 0x05 | 灯光控制 | Go→STM32 | 控制LED灯效 |
| **硬件事件上报（STM32→Golang）** |
| 0x11 | 投币检测 | STM32→Go | 检测到投币 |
| 0x12 | 回币检测 | STM32→Go | 检测到回币 |
| 0x13 | 按键事件 | STM32→Go | 用户按键 |
| 0x14 | 传感器事件 | STM32→Go | 各类传感器触发 |
| **状态管理** |
| 0x21 | 状态查询 | Go→STM32 | 查询设备状态 |
| 0x22 | 状态上报 | STM32→Go | 上报设备状态 |
| 0x23 | 故障上报 | STM32→Go | 硬件故障 |
| 0x24 | 执行进度 | STM32→Go | 指令执行进度 |
| 0x25 | 故障恢复 | Go→STM32 | 故障恢复指令 |
| **系统指令** |
| 0x31 | 心跳包 | 双向 | 保持连接 |
| 0x80 | ACK确认 | 双向 | 确认收到 |
| 0x81 | NACK拒绝 | 双向 | 执行失败 |

### 2.4 序列号管理规则

#### 序列号分配
- Golang发送：使用奇数序列号（0x0001, 0x0003, 0x0005...）
- STM32上报：使用偶数序列号（0x0002, 0x0004, 0x0006...）

#### 序列号溢出处理
- 当序列号达到0xFFFF时，下一个序列号回到0x0001（Golang）或0x0002（STM32）
- 序列号不要求严格连续，但必须保持奇偶性
- 序列号生成示例：
```c
// STM32端
uint16_t get_next_seq() {
    static uint16_t seq = 0;
    seq += 2;
    if (seq == 0 || seq == 0xFFFF) {
        seq = 0x0002; // 重新从2开始
    }
    return seq;
}
```

#### 序列号防重机制
- STM32应维护最近接收的10个序列号缓存
- 收到重复序列号时，返回上次的执行结果，不重复执行
- 缓存时间：5分钟后自动清除
```c
typedef struct {
    uint16_t seq;
    uint8_t cmd;
    uint8_t result;
    uint32_t timestamp;
} CmdCache;

CmdCache cmd_cache[10];
```

---

## 第三部分：功能实现

## 3. 硬件控制指令详细定义

### 3.1 上币控制（0x01）

**功能**：控制上币电机，将游戏币从币仓送到游戏台面

**Golang→STM32（上币指令）**：
```
[0xAA][长度][0x01][序列号][数量:2字节][速度][CRC16][0x55]
```

**数据字段**：
- **数量**：uint16，上币数量（1-9999）
- **速度**：uint8，上币速度（1-10，默认5）

**STM32→Golang（ACK确认）**：
```
[0xAA][长度][0x80][序列号][0x01][状态码][CRC16][0x55]
```
- 序列号：原始上币指令的序列号
- 0x01：原始命令码（上币控制）
- 状态码：0x00表示开始执行

**STM32→Golang（执行进度）**：
```
[0xAA][长度][0x24][序列号][命令码][已完成:2字节][总数:2字节][CRC16][0x55]
```

**示例流程**：
```
1. Golang发送：上币20个，速度5
   [AA][00 0C][01][00 01][00 14][05][CRC][55]
                  ^命令  ^序列号

2. STM32确认：开始执行
   [AA][00 0C][80][00 01][01][00][CRC][55]
                  ^ACK   ^原序列号 ^原命令 ^状态:开始

3. STM32上报：已上10个，共20个
   [AA][00 0E][24][00 02][01][00 0A][00 14][CRC][55]
                  ^进度  ^新序列号 ^命令 ^已完成 ^总数

4. STM32上报：已上20个，共20个（完成）
   [AA][00 0E][24][00 04][01][00 14][00 14][CRC][55]
```

### 3.2 退币控制（0x02）

**功能**：控制退币电机，将游戏币退还给用户

**Golang→STM32（退币指令）**：
```
[0xAA][长度][0x02][序列号][数量:2字节][CRC16][0x55]
```

**数据字段**：
- **数量**：uint16，退币数量（1-9999）

**执行流程**：同上币控制

### 3.3 彩票发放（0x03）

**功能**：控制彩票机发放彩票

**Golang→STM32（彩票指令）**：
```
[0xAA][长度][0x03][序列号][数量:2字节][CRC16][0x55]
```

**数据字段**：
- **数量**：uint16，彩票数量（1-9999）

**特殊状态码**：
- 0x00：正常
- 0x01：缺纸
- 0x02：卡纸
- 0x03：打印头故障

### 3.4 推币控制（0x04）

**功能**：控制推币电机的运动

**Golang→STM32（推币指令）**：
```
[0xAA][长度][0x04][序列号][动作][参数][CRC16][0x55]
```

**动作类型**：
- 0x01：启动连续推币
- 0x02：停止推币
- 0x03：单次推币
- 0x04：设置推币速度

**参数定义**：
- 动作0x01/0x02：无参数
- 动作0x03：推币次数（1字节）
- 动作0x04：速度值（1字节，1-10）

### 3.5 灯光控制（0x05）

**功能**：控制LED灯光效果

**Golang→STM32（灯光指令）**：
```
[0xAA][长度][0x05][序列号][灯光位][CRC16][0x55]
```

**数据字段**：
**灯光位定义（bitmask）**：
- `off`  = `0x20`   // 关
- `l1`   = `0x22`   // 灯1
- `l2`   = `0x30`   // 灯2
- `l3`   = `0x24`   // 灯3
- `l4`   = `0x28`   // 灯4
- `l5`   = `0x21`   // 灯5
- `lall` = `0x7F`   // 所有灯
- `loff` = `0x40`   // 所有灯关闭

## 4. 硬件事件上报详细定义

### 4.1 投币检测（0x11）

**功能**：检测到用户投币，立即上报

**STM32→Golang（投币事件）**：
```
[0xAA][长度][0x11][序列号][数量][CRC16][0x55]
```

**数据字段**：
- **数量**：uint8，投币数量（通常为1）

**Golang→STM32（ACK确认）**：
```
[0xAA][长度][0x80][序列号][0x11][0x00][CRC16][0x55]
```
- 序列号：投币事件的序列号
- 0x11：原始命令码（投币检测）
- 0x00：状态码（成功确认）

**重要性**：P0级别，必须100ms内响应

### 4.2 回币检测（0x12）（重要：用于回币率统计）

**功能**：检测到游戏币被推出，一次上报三个位置的数量

**STM32→Golang（回币事件）**：
```
[0xAA][长度][0x12][序列号][前方数量][左侧数量][右侧数量][CRC16][0x55]
```

**数据字段**（优化后）：
- **前方数量**：uint8，前方出币口回币数（玩家获得，进入回币马达）
- **左侧数量**：uint8，左侧出币口回币数（损失，不返还玩家）
- **右侧数量**：uint8，右侧出币口回币数（损失，不返还玩家）

**优点**：
- 一次通信完成所有位置的上报
- 减少通信次数，提高效率
- 数据结构更清晰，易于解析

**回币率计算**：
- 回币率 = 前方数量 / (前方数量 + 左侧数量 + 右侧数量) × 100%
- Golang根据回币率调整游戏难度和中奖概率

**Golang→STM32（ACK确认）**：
```
[0xAA][长度][0x80][序列号][0x12][0x00][CRC16][0x55]
```

### 4.3 按键事件（0x13）

**功能**：用户按键操作，分为游戏按键和配置按键

**STM32→Golang（按键事件）**：
```
[0xAA][长度][0x13][序列号][按键类型][按键码][动作][附加数据][CRC16][0x55]
```

**数据字段**：

#### 按键类型（uint8）：
- 0x01：游戏按键（开始/彩票）
- 0x02：配置按键（4键组合）

#### 游戏按键码（类型=0x01时）：
- **0x01：开始按键**
  - 功能：用户按下后，控制板通知Golang，Golang根据当前游戏币数量执行上币操作
  - 响应：Golang收到后执行 `上币控制(0x01)` 指令，将币落入推盘开始游戏
  
- **0x02：退币/彩票按键**（根据当前模式）
  - 功能：同一个物理按键，根据Golang设置的模式执行不同功能
  - 退币模式：Golang收到后执行 `退币控制(0x02)` 指令，退还游戏币
  - 彩票模式：Golang收到后执行 `彩票发放(0x03)` 指令，打印彩票
  - 注意：模式判断由Golang完成，STM32只负责上报按键事件

#### 配置按键码（类型=0x02时）：
- **0x11：上键（↑）/ 左键（←）**
- **0x12：下键（↓）/ 右键（→）**
- **0x13：确认键**（OK/Enter）
- **0x14：取消键**（ESC/Cancel）

#### 动作类型（uint8）：
- 0x01：按下（KEY_DOWN）
- 0x02：释放（KEY_UP）
- 0x03：长按（KEY_HOLD）- 持续按住超过1秒
- 0x04：双击（KEY_DOUBLE）- 500ms内按两次

#### 附加数据（可选）：
- 长按时：包含持续时间（uint16，单位ms）
- 组合键时：包含同时按下的其他键码

**示例**：

1. **开始按键按下**：
```
[0xAA][0x00][0x0C][0x13][序列号][0x01][0x01][0x01][CRC16][0x55]
                                   游戏类 开始键 按下
```

2. **退币/彩票按键按下**（模式由Golang判断）：
```
[0xAA][0x00][0x0C][0x13][序列号][0x01][0x02][0x01][CRC16][0x55]
                                   游戏类 退币/彩票键 按下
```

3. **退币/彩票按键长按2秒**（可用于特殊功能）：
```
[0xAA][0x00][0x0E][0x13][序列号][0x01][0x02][0x03][0x07][0xD0][CRC16][0x55]
                                   游戏类 退币/彩票键 长按  2000ms
```

4. **配置上键按下**：
```
[0xAA][0x00][0x0C][0x13][序列号][0x02][0x11][0x01][CRC16][0x55]
                                   配置类 上键  按下
```

5. **配置组合键（上+确认）**：
```
[0xAA][0x00][0x0D][0x13][序列号][0x02][0x11][0x01][0x15][CRC16][0x55]
                                   配置类 上键  按下  确认键同时按
```

### 4.4 传感器事件（0x14）

**功能**：各类传感器触发事件

**STM32→Golang（传感器事件）**：
```
[0xAA][长度][0x14][序列号][传感器类型][数值:2字节][CRC16][0x55]
```

**传感器类型**：
- 0x01：币仓余量传感器
- 0x02：彩票余量传感器
- 0x03：温度传感器
- 0x04：震动传感器
- 0x05：门开关传感器

## 5. 状态管理

### 5.1 状态查询（0x21）

**功能**：Golang查询STM32设备状态

**Golang→STM32（查询指令）**：
```
[0xAA][长度][0x21][序列号][查询类型][CRC16][0x55]
```

**查询类型**：
- 0x01：全部状态
- 0x02：硬件状态
- 0x03：传感器状态
- 0x04：故障状态

### 5.2 状态上报（0x22）

**功能**：STM32上报设备状态

**STM32→Golang（状态数据）**：
```
[0xAA][长度][0x22][序列号][状态数据][CRC16][0x55]
```

**状态数据结构**：
```c
struct DeviceStatus {
    uint8_t  coin_motor;      // 上币电机状态
    uint8_t  return_motor;     // 退币电机状态
    uint8_t  push_motor;       // 推币电机状态
    uint8_t  ticket_printer;   // 彩票机状态
    uint16_t coin_count;       // 币仓余量
    uint16_t ticket_count;     // 彩票余量
    uint8_t  temperature;      // 设备温度
    uint8_t  error_flags;      // 错误标志位
    uint8_t  current_mode;     // 当前模式（0x01:退币 0x02:彩票）
};
```

### 5.3 故障上报（0x23）

**功能**：硬件故障自动上报

**STM32→Golang（故障事件）**：
```
[0xAA][长度][0x23][序列号][故障码][严重级别][附加信息][CRC16][0x55]
```

**故障码定义**：
| 故障码 | 说明 | 处理建议 |
|--------|------|----------|
| 0x01 | 上币电机卡死 | 检查币道 |
| 0x02 | 退币电机故障 | 检查退币口 |
| 0x03 | 彩票机缺纸 | 添加彩票纸 |
| 0x04 | 彩票机卡纸 | 清理卡纸 |
| 0x05 | 推币电机过载 | 检查推板 |
| 0x06 | 投币器卡币 | 清理投币器 |
| 0x07 | 温度过高 | 设备散热 |
| 0x08 | 传感器异常 | 检查连接 |

**严重级别**：
- 0x01：提示（不影响运行）
- 0x02：警告（部分功能受限）
- 0x03：错误（需要处理）
- 0x04：严重（停机维护）

### 5.4 执行进度（0x24）

**功能**：上报指令执行进度

**STM32→Golang（进度上报）**：
```
[0xAA][长度][0x24][序列号][原命令码][已完成:2字节][总数:2字节][状态][CRC16][0x55]
```

**状态定义**：
- 0x00：执行中
- 0x01：已完成
- 0x02：已取消
- 0x03：执行失败

### 5.5 故障恢复（0x25）

**功能**：当故障发生后，Golang发送恢复指令

**Golang→STM32（故障恢复指令）**：
```
[0xAA][长度][0x25][序列号][故障码][恢复动作][参数][CRC16][0x55]
```

**数据字段**：
- **故障码**：uint8，要恢复的故障类型（同故障上报的故障码）
- **恢复动作**：uint8
  - 0x01：重启设备
  - 0x02：清除故障标志
  - 0x03：强制重置
  - 0x04：重试操作
  - 0x05：跳过故障（仅记录）
- **参数**：uint8，恢复动作的参数（如重试次数）

**STM32→Golang（恢复结果）**：
```
[0xAA][长度][0x80][序列号][0x25][恢复状态][CRC16][0x55]
```
- **恢复状态**：
  - 0x00：恢复成功
  - 0x01：恢复失败，需人工干预
  - 0x02：部分恢复
  - 0x03：正在恢复中

**自动恢复机制**：
- 卡币：自动反转电机尝试清除
- 卡纸：自动回收彩票并重新送纸
- 过载：自动减速或暂停休息
- 通信异常：自动重连3次

## 6. 系统指令

### 6.1 心跳包（0x31）

**功能**：保持连接活跃，检测通信状态，协商版本

**Golang→STM32（心跳请求）**：
```
[0xAA][长度][0x31][序列号][时间戳:4字节][版本:2字节][CRC16][0x55]
```

**STM32→Golang（心跳响应）**：
```
[0xAA][长度][0x31][序列号][时间戳:4字节][运行时间:4字节][版本:2字节][CRC16][0x55]
```

**心跳策略**：
- 发送间隔：30秒
- 超时时间：5秒
- 重试次数：3次
- 失败处理：标记离线，尝试重连

**版本协商**：
- 版本字段格式：主版本号(1字节) + 次版本号(1字节)
- 示例：v1.1 = 0x0101

### 6.2 ACK/NACK响应

**ACK确认（0x80）**：
```
[0xAA][长度][0x80][原序列号][原命令码][状态码][CRC16][0x55]
```

**数据字段**：
- **原序列号**：uint16，被确认的原始命令序列号
- **原命令码**：uint8，被确认的原始命令码
- **状态码**：uint8，执行状态
  - 0x00：成功接收，开始执行
  - 0x01：成功接收，已完成
  - 0x02：成功接收，排队等待

**NACK拒绝（0x81）**：
```
[0xAA][长度][0x81][原序列号][原命令码][错误码][CRC16][0x55]
```

**数据字段**：
- **原序列号**：uint16，被拒绝的原始命令序列号
- **原命令码**：uint8，被拒绝的原始命令码
- **错误码**：uint8，错误原因
  - 0x01：命令不支持
  - 0x02：参数错误
  - 0x03：设备忙
  - 0x04：硬件故障
  - 0x05：校验失败
  - 0x06：超出范围
  - 0x07：资源不足

**示例**：

1. **对上币指令的ACK确认**：
```
Golang发送：[AA][00 0C][01][00 01][00 14][05][CRC][55]  // 上币20个
STM32确认：[AA][00 0C][80][00 01][01][00][CRC][55]      // ACK：序列号0001，命令01，成功
```

2. **对彩票指令的NACK拒绝**：
```
Golang发送：[AA][00 0B][03][00 03][03 E8][CRC][55]      // 发1000张彩票
STM32拒绝：[AA][00 0C][81][00 03][03][02][CRC][55]      // NACK：序列号0003，命令03，参数错误
```

## 7. 通信流程示例

### 7.1 完整游戏流程（从投币到结算）
```
1. 用户投币
   STM32: [投币检测(0x11)] → Golang
   Golang: [ACK确认(0x80)] → STM32
   Golang: 记账，增加用户余额
   
2. 用户按下开始按键
   STM32: [按键事件(0x13): 开始键] → Golang
   Golang: [ACK确认(0x80)] → STM32
   Golang: 判断余额，决定上币数量
   
3. 执行上币（币落入推盘）
   Golang: [上币指令(0x01): 20个] → STM32
   STM32: [ACK确认(0x80): 开始执行] → Golang
   STM32: [执行进度(0x24): 5/20] → Golang
   STM32: [执行进度(0x24): 10/20] → Golang
   STM32: [执行进度(0x24): 15/20] → Golang
   STM32: [执行进度(0x24): 20/20 完成] → Golang
   
4. 推币动作
   Golang: [推币控制(0x04): 启动连续推币] → STM32
   STM32: [ACK确认(0x80)] → Golang
   
5. 回币检测（优化后：一次上报三个位置）
   STM32: [回币检测(0x12): 前方5个，左侧2个，右侧1个] → Golang
   Golang: [ACK确认(0x80)] → STM32
   Golang: 记账，前方增加玩家积分，两侧记录损失
   Golang: 计算回币率并持久化数据
   
6. 结算阶段
   Golang: 计算回币率 = 5/(5+2+1) = 62.5%
   Golang: 根据回币率调整游戏参数
   
7. 用户按下退币/彩票按键
   STM32: [按键事件(0x13): 退币/彩票键] → Golang
   Golang: [ACK确认(0x80)] → STM32
   Golang: 内部判断当前模式（STM32不保存模式状态）
   
   若Golang内部设置为退币模式：
   Golang: [退币控制(0x02): 10个] → STM32
   STM32: [ACK确认(0x80)] → Golang
   STM32: [执行进度(0x24): 10/10 完成] → Golang
   
   若Golang内部设置为彩票模式：
   Golang: [彩票发放(0x03): 10张] → STM32
   STM32: [ACK确认(0x80)] → Golang
   STM32: [执行进度(0x24): 10/10 完成] → Golang
```

### 7.2 模式管理说明（STM32无状态设计）
```
1. 模式完全由Golang管理
   - Golang内部保存当前模式（退币/彩票）
   - 根据配置或运营需求切换
   - STM32不知道当前模式
   
2. 按键处理流程
   - STM32: 仅上报按键事件
   - Golang: 根据内部模式决定执行操作
   - 执行: 发送对应指令给STM32
```

### 7.3 回币率统计方案（Go端实现）
```
1. 实时数据采集
   STM32: 检测到回币立即上报三个位置数量
   Golang: 接收后立即保存到数据库
   
2. 统计分析（Golang端）
   - 累计各位置回币数量
   - 计算回币率 = 前方/(前方+左侧+右侧)
   - 根据回币率调整游戏难度
   
3. 持久化存储
   - 每次接收到数据立即写入
   - 定期生成统计报表
   - 支持历史数据查询
```

### 7.4 故障处理与恢复流程
```
1. STM32检测到故障
   STM32: [故障上报(0x23): 彩票缺纸，严重级别2] → Golang
   Golang: [ACK确认(0x80)] → STM32
   
2. Golang处理决策
   - 记录故障日志
   - 判断严重级别
   - 决定恢复策略
   
3. 自动恢复尝试
   Golang: [故障恢复(0x25): 故障码，重试操作] → STM32
   STM32: 执行恢复动作
   STM32: [ACK确认(0x80): 恢复状态] → Golang
   
4. 恢复结果判断
   若成功：
   - 恢复正常操作
   - 记录恢复成功
   
   若失败：
   - 停止相关操作
   - 通知维护人员
   - 显示故障提示
   
5. 人工干预后
   维护人员处理（如添加彩票纸）
   Golang: [故障恢复(0x25): 清除故障标志] → STM32
   STM32: [ACK确认(0x80): 恢复成功] → Golang
   Golang: [状态查询(0x21)] → STM32
   STM32: [状态上报(0x22): 正常] → Golang
```

### 7.5 异常恢复流程

#### 通信中断恢复
1. **检测中断**
   - Golang：心跳超时3次判定为中断
   - STM32：60秒无指令判定为中断

2. **自动重连**
   ```
   Golang: 检测到中断 → 关闭串口 → 等待1秒 → 重新打开串口
   STM32: 检测到中断 → 复位串口 → 等待连接
   ```

3. **状态同步**
   ```
   Golang: [状态查询(0x21)] → STM32
   STM32: [状态上报(0x22)] → Golang
   Golang: 根据状态决定是否恢复未完成操作
   ```

#### 数据包处理
1. **粘包处理**：通过帧头(0xAA)和长度字段分割
   ```c
   // STM32端示例
   void process_rx_buffer(uint8_t* buf, uint16_t len) {
       static uint8_t frame_buf[512];
       static uint16_t frame_len = 0;
       
       // 查找帧头
       for (int i = 0; i < len; i++) {
           if (buf[i] == 0xAA) {
               // 解析长度
               uint16_t pkt_len = (buf[i+1] << 8) | buf[i+2];
               // 提取完整帧
               if (i + pkt_len <= len) {
                   process_frame(&buf[i], pkt_len);
                   i += pkt_len - 1;
               }
           }
       }
   }
   ```

2. **半包处理**：缓存不完整数据，等待后续数据
   ```c
   if (frame_len + recv_len < expected_len) {
       // 数据不完整，追加到缓冲区
       memcpy(&frame_buf[frame_len], recv_buf, recv_len);
       frame_len += recv_len;
       return; // 等待更多数据
   }
   ```

3. **错包处理**：CRC校验失败直接丢弃，等待重发
   ```c
   if (calc_crc != frame->crc) {
       // CRC错误，发送NACK
       send_nack(frame->seq, frame->cmd, 0x05);
       return;
   }
   ```

---

## 第四部分：最佳实践

## 8. STM32实现要求

### 8.1 基本要求
1. **被动响应**：STM32不主动发起业务操作，仅响应Golang指令
2. **实时上报**：硬件事件（投币、回币等）必须立即上报
3. **状态维护**：维护所有硬件的实时状态
4. **故障检测**：自动检测并上报硬件故障

### 8.2 性能指标
| 指标 | 要求 | 说明 |
|------|------|------|
| 响应时间 | <50ms | 收到指令到返回ACK |
| 投币检测 | <10ms | 硬件触发到上报 |
| 执行精度 | 100% | 上币/退币数量准确 |
| 心跳间隔 | 30s | 保持连接 |

### 8.3 缓冲区管理
```c
// 推荐的缓冲区设置
#define RX_BUFFER_SIZE  512   // 接收缓冲区
#define TX_BUFFER_SIZE  512   // 发送缓冲区
#define CMD_QUEUE_SIZE  16    // 命令队列深度
#define EVENT_QUEUE_SIZE 32   // 事件队列深度
```

### 8.4 中断优先级
```c
// STM32中断优先级建议
#define UART_RX_PRIORITY    1  // 串口接收（最高）
#define COIN_DETECT_PRIORITY 2  // 投币检测
#define BUTTON_PRIORITY     2  // 按键检测（同投币）
#define TIMER_PRIORITY      3  // 定时器
#define MOTOR_PRIORITY      4  // 电机控制
```

### 8.5 按键硬件设计建议
```c
// 按键消抖配置
#define DEBOUNCE_TIME_MS    20   // 消抖时间20ms
#define LONG_PRESS_TIME_MS  1000 // 长按判定1秒
#define DOUBLE_CLICK_TIME_MS 500 // 双击间隔500ms

// 按键GPIO配置示例（STM32）
typedef struct {
    GPIO_TypeDef* port;
    uint16_t pin;
    uint8_t key_code;
    uint8_t key_type;
    char* description;
} KeyConfig;

// 按键配置表（只有2个游戏按键）
const KeyConfig key_map[] = {
    // 游戏按键
    {GPIOA, GPIO_PIN_0, 0x01, 0x01, "开始按键"},     // 开始按键
    {GPIOA, GPIO_PIN_1, 0x02, 0x01, "退币/彩票键"},  // 退币/彩票按键（根据模式）
    
    // 注：配置按键可选，若需要可通过长按或组合键实现
    // 例如：同时按住两个按键3秒进入配置模式
};

// 按键状态机
typedef enum {
    KEY_IDLE,
    KEY_DEBOUNCE,
    KEY_PRESSED,
    KEY_HOLD,
    KEY_RELEASED
} KeyState;

// 按键扫描处理（建议10ms调用一次）
void scan_keys(void) {
    for (int i = 0; i < KEY_COUNT; i++) {
        KeyState state = get_key_state(i);
        uint32_t press_time = get_press_time(i);
        
        switch (state) {
        case KEY_PRESSED:
            // 发送按下事件
            send_key_event(key_map[i].key_type, 
                          key_map[i].key_code, 
                          0x01, 0);
            break;
            
        case KEY_HOLD:
            if (press_time >= LONG_PRESS_TIME_MS) {
                // 发送长按事件（只发一次）
                send_key_event(key_map[i].key_type,
                              key_map[i].key_code,
                              0x03, press_time);
            }
            break;
            
        case KEY_RELEASED:
            // 发送释放事件（可选）
            send_key_event(key_map[i].key_type,
                          key_map[i].key_code,
                          0x02, 0);
            break;
        }
    }
}

// 特殊组合键检测（可选功能）
bool check_config_entry(void) {
    // 同时按住两个按键3秒进入工程模式（可选）
    if (is_key_pressed(KEY_START) && is_key_pressed(KEY_REFUND_TICKET)) {
        if (get_combo_press_time() >= 3000) {
            // 发送特殊组合键事件给Golang
            send_combo_key_event(0x01, 0x02, 3000);
            return true;
        }
    }
    return false;
}

// 发送回币事件（优化后：一次发送三个位置数据）
void send_coin_return_event(uint8_t front, uint8_t left, uint8_t right) {
    // 新数据格式：[前方数量][左侧数量][右侧数量]
    uint8_t data[3] = {front, left, right};
    
    // 发送回币检测事件
    send_event(0x12, data, sizeof(data));
    
    // 注意：STM32不存储统计数据，所有统计由Golang完成
    
    // 调试输出
    #ifdef DEBUG
    printf("回币事件: 前方=%d, 左侧=%d, 右侧=%d\n", front, left, right);
    #endif
}

// 回币检测处理（定时调用或中断触发）
void process_coin_return_detection(void) {
    uint8_t front_coins = 0;
    uint8_t left_coins = 0;
    uint8_t right_coins = 0;
    
    // 检测各位置回币数量
    front_coins = detect_front_coins();
    left_coins = detect_left_coins();
    right_coins = detect_right_coins();
    
    // 如果有任何位置检测到回币，立即上报
    if (front_coins > 0 || left_coins > 0 || right_coins > 0) {
        send_coin_return_event(front_coins, left_coins, right_coins);
    }
}
```

### 8.6 时序规范汇总

| 操作类型 | 响应时间要求 | 超时处理 | 重试次数 |
|---------|-------------|---------|---------|
| 投币检测→ACK | <100ms | 记录错误，继续 | 0 |
| 命令→ACK | <50ms | 重发命令 | 3 |
| 心跳→响应 | <5s | 标记离线 | 3 |
| 上币完成 | <30s | 取消操作 | 0 |
| 退币完成 | <30s | 人工介入 | 0 |
| 彩票打印 | <60s | 故障上报 | 1 |
| 状态查询 | <200ms | 重发查询 | 2 |
| 故障恢复 | <10s | 人工介入 | 1 |

### 8.7 并发控制

#### 命令执行模式
- **串行执行**：同类型命令必须串行（如：两个上币命令）
- **并行执行**：不同类型命令可并行（如：上币+灯光控制）

#### 命令队列管理
```c
#define CMD_QUEUE_SIZE     16   // 命令队列深度
#define HIGH_PRIORITY_QUEUE 4   // 高优先级队列

// 优先级定义
typedef enum {
    PRIORITY_URGENT = 0,    // 紧急（故障恢复）
    PRIORITY_HIGH = 1,      // 高（投币响应）
    PRIORITY_NORMAL = 2,    // 普通（游戏操作）
    PRIORITY_LOW = 3        // 低（状态查询）
} CommandPriority;

// 命令队列结构
typedef struct {
    uint8_t cmd;
    uint16_t seq;
    uint8_t priority;
    uint8_t data[256];
    uint16_t data_len;
    uint32_t timestamp;
} Command;

// 队列操作
void enqueue_command(Command* cmd) {
    // 根据优先级插入队列
    if (cmd->priority == PRIORITY_URGENT) {
        // 插入高优先级队列头部
        insert_high_priority(cmd);
    } else {
        // 插入普通队列
        insert_normal_queue(cmd);
    }
}
```

#### 资源互斥
- 电机类操作需要互斥（上币、退币、推币）
- 传感器读取和电机操作可并行
```c
// 资源锁定标志
typedef struct {
    bool coin_motor_busy;    // 上币电机忙
    bool return_motor_busy;  // 退币电机忙
    bool push_motor_busy;    // 推币电机忙
    bool printer_busy;       // 彩票机忙
} ResourceLock;

// 检查资源是否可用
bool check_resource_available(uint8_t cmd) {
    switch (cmd) {
    case 0x01: // 上币
        return !resource_lock.coin_motor_busy;
    case 0x02: // 退币
        return !resource_lock.return_motor_busy;
    case 0x03: // 彩票
        return !resource_lock.printer_busy;
    case 0x04: // 推币
        return !resource_lock.push_motor_busy;
    default:
        return true; // 其他命令不需要资源锁
    }
}
```

## 9. Golang实现示例

### 9.1 基本通信框架
```go
package hardware

import (
    "encoding/binary"
    "github.com/tarm/serial"
    "sync"
)

type STM32Controller struct {
    port     *serial.Port
    seqNum   uint16
    mutex    sync.Mutex
    eventCh  chan HardwareEvent
}

// 发送指令并等待ACK
func (c *STM32Controller) SendCommand(cmd byte, data []byte) error {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    seq := c.getNextSeq()
    frame := c.buildFrame(cmd, seq, data)
    
    // 记录待确认命令
    c.pendingCmds[seq] = &PendingCommand{
        Cmd: cmd,
        Seq: seq,
        Time: time.Now(),
    }
    
    _, err := c.port.Write(frame)
    if err != nil {
        delete(c.pendingCmds, seq)
        return err
    }
    
    // 等待ACK（超时3秒）
    select {
    case ack := <-c.ackChan:
        if ack.OrigSeq == seq && ack.OrigCmd == cmd {
            if ack.Status == 0x00 {
                return nil // 成功
            }
            return fmt.Errorf("命令执行失败: %d", ack.Status)
        }
    case <-time.After(3 * time.Second):
        delete(c.pendingCmds, seq)
        return fmt.Errorf("等待ACK超时")
    }
    
    return nil
}

// 处理ACK响应
func (c *STM32Controller) handleAck(data []byte) {
    if len(data) < 4 {
        return
    }
    
    ack := AckResponse{
        OrigSeq: binary.BigEndian.Uint16(data[0:2]),
        OrigCmd: data[2],
        Status:  data[3],
    }
    
    // 查找对应的待确认命令
    if cmd, ok := c.pendingCmds[ack.OrigSeq]; ok {
        if cmd.Cmd == ack.OrigCmd {
            delete(c.pendingCmds, ack.OrigSeq)
            c.ackChan <- ack
        }
    }
}

// 上币控制
func (c *STM32Controller) DispenseCoins(count uint16, speed byte) error {
    data := make([]byte, 3)
    binary.BigEndian.PutUint16(data[0:2], count)
    data[2] = speed
    
    return c.SendCommand(0x01, data)
}

// 处理硬件事件
func (c *STM32Controller) HandleEvents() {
    for event := range c.eventCh {
        switch event.Type {
        case EventCoinIn:
            // 处理投币事件
            c.onCoinInserted(event.Data)
        case EventCoinOut:
            // 处理回币事件
            c.onCoinReturned(event.Data)
        case EventButton:
            // 处理按键事件
            c.onButtonPressed(event.Data)
        }
    }
}
```

### 9.2 事件处理
```go
// 投币事件处理
func (c *STM32Controller) onCoinInserted(data []byte) {
    count := data[0]
    coinType := data[1]
    
    // 1. 发送ACK确认
    c.sendAck(data)
    
    // 2. 更新用户余额（游戏逻辑）
    gameLogic.AddCredits(count)
    
    // 3. 等待用户按开始键
    // 不自动开始，由按键触发
    
    log.Printf("投币检测: 数量:%d 类型:%s", count, c.getCoinTypeName(coinType))
}

// 回币事件处理（优化后：一次接收三个位置数据）
func (c *STM32Controller) onCoinReturned(data []byte) {
    // 新数据格式：[前方数量][左侧数量][右侧数量]
    frontCount := data[0]
    leftCount := data[1]
    rightCount := data[2]
    
    // 1. 发送ACK确认
    c.sendAck(data)
    
    // 2. 处理回币数据
    if frontCount > 0 {
        gameLogic.AddPlayerCoins(frontCount)
        c.stats.CoinsReturnedFront += uint16(frontCount)
        log.Printf("回币检测[前方]: %d个 (玩家获得)", frontCount)
    }
    
    if leftCount > 0 {
        c.stats.CoinsReturnedLeft += uint16(leftCount)
        log.Printf("回币检测[左侧]: %d个 (损失)", leftCount)
    }
    
    if rightCount > 0 {
        c.stats.CoinsReturnedRight += uint16(rightCount)
        log.Printf("回币检测[右侧]: %d个 (损失)", rightCount)
    }
    
    // 3. 立即持久化数据（防断电丢失）
    c.saveStatisticsToDatabase(&c.stats)
    
    // 4. 实时计算回币率
    returnRate := c.calculateReturnRate(&c.stats)
    
    // 5. 通知游戏逻辑
    gameLogic.UpdateReturnRate(returnRate)
    
    log.Printf("回币统计更新: 前:%d 左:%d 右:%d 回币率:%.2f%%",
        frontCount, leftCount, rightCount, returnRate)
}

// 按键事件处理
func (c *STM32Controller) onButtonPressed(data []byte) {
    keyType := data[0]
    keyCode := data[1]
    action := data[2]
    
    // 发送ACK确认
    c.sendAck(data)
    
    switch keyType {
    case 0x01: // 游戏按键
        c.handleGameButton(keyCode, action)
    case 0x02: // 配置按键
        c.handleConfigButton(keyCode, action)
    }
}

// 游戏按键处理
func (c *STM32Controller) handleGameButton(keyCode, action byte) {
    if action != 0x01 { // 只响应按下事件，忽略释放
        return
    }
    
    switch keyCode {
    case 0x01: // 开始按键
        if gameLogic.HasCredits() {
            // 获取用户投币数
            coinCount := gameLogic.GetPendingCoins()
            
            // 通过上币接口落币，开始游戏
            c.DispenseCoins(coinCount, 5)
            
            // 开始游戏逻辑
            gameLogic.StartGame(coinCount)
            
            // 启动推币
            c.SendCommand(0x04, []byte{0x01}) // 启动连续推币
        } else {
            // 提示需要投币
            c.PlaySound(SOUND_NEED_COIN)
        }
        
    case 0x02: // 退币/彩票按键（根据内部模式）
        // 判断内部模式（STM32不存储模式）
        if gameLogic.GetCurrentMode() == MODE_COIN_REFUND { // 退币模式
            if gameLogic.HasRefundableCoins() {
                // 获取可退币数
                coinCount := gameLogic.GetRefundableCoins()
                
                // 发送退币指令
                c.RefundCoins(coinCount)
                
                // 扣除余额
                gameLogic.DeductCoins(coinCount)
            } else {
                // 提示无可退币
                c.PlaySound(SOUND_NO_COIN)
            }
        } else if gameLogic.GetCurrentMode() == MODE_TICKET { // 彩票模式
            if gameLogic.HasTickets() {
                // 获取可兑换彩票数
                ticketCount := gameLogic.GetAvailableTickets()
                
                // 发放彩票
                c.DispenseTickets(ticketCount)
                
                // 扣除彩票积分
                gameLogic.RedeemTickets(ticketCount)
            } else {
                // 提示无可用彩票
                c.PlaySound(SOUND_NO_TICKET)
            }
        }
    }
}

// 回币率统计（Go端实现）
type CoinStatistics struct {
    CoinsReturnedFront uint16  // 前方回币
    CoinsReturnedLeft  uint16  // 左侧回币
    CoinsReturnedRight uint16  // 右侧回币
    ReturnRate         float64 // 回币率
    Timestamp          time.Time
}

// 计算回币率
func (c *STM32Controller) calculateReturnRate(stats *CoinStatistics) float64 {
    totalReturned := stats.CoinsReturnedFront + 
                    stats.CoinsReturnedLeft + 
                    stats.CoinsReturnedRight
    
    if totalReturned == 0 {
        return 0
    }
    
    // 回币率 = 前方回币 / 总回币
    returnRate := float64(stats.CoinsReturnedFront) / float64(totalReturned) * 100
    
    log.Printf("回币率统计: 前方:%d 左侧:%d 右侧:%d 回币率:%.2f%%",
        stats.CoinsReturnedFront,
        stats.CoinsReturnedLeft,
        stats.CoinsReturnedRight,
        returnRate)
    
    return returnRate
}

// 根据回币率调整游戏难度
func (c *STM32Controller) adjustGameDifficulty(returnRate float64) {
    switch {
    case returnRate < 30:
        // 回币率太低，降低难度
        gameLogic.SetDifficulty(DIFFICULTY_EASY)
        log.Println("回币率过低，调整为简单模式")
        
    case returnRate > 70:
        // 回币率太高，增加难度
        gameLogic.SetDifficulty(DIFFICULTY_HARD)
        log.Println("回币率过高，调整为困难模式")
        
    default:
        // 回币率正常
        gameLogic.SetDifficulty(DIFFICULTY_NORMAL)
    }
}

// 配置按键处理
func (c *STM32Controller) handleConfigButton(keyCode, action byte) {
    // 配置菜单状态机
    switch c.configMenu.State {
    case MENU_CLOSED:
        if keyCode == 0x15 && action == 0x03 { // 长按确认键进入菜单
            c.configMenu.Open()
        }
        
    case MENU_MAIN:
        switch keyCode {
        case 0x11: // 上键
            c.configMenu.PrevItem()
        case 0x12: // 下键
            c.configMenu.NextItem()
        case 0x13: // 确认键
            c.configMenu.SelectItem()
        case 0x14: // 取消键
            c.configMenu.Close()
        }
        
    case MENU_SETTING:
        switch keyCode {
        case 0x11: // 左键
            c.configMenu.DecrValue()
        case 0x12: // 右键
            c.configMenu.IncrValue()
        case 0x13: // 确认键
            c.configMenu.SaveSetting()
        case 0x14: // 取消键
            c.configMenu.Back()
        }
    }
}

// 游戏序列（由开始按键触发）
func (c *STM32Controller) StartGameSequence(coinCount uint16) {
    // 1. 落币动画
    c.DispenseCoins(coinCount, 5)
    
    // 2. 等待落币完成
    c.WaitForCompletion(0x01)
    
    // 3. 计算游戏结果（纯软件逻辑）
    result := gameLogic.Calculate()
    
    // 4. 执行推币
    c.SendCommand(0x04, []byte{0x01}) // 启动推币
    
    // 5. 根据结果控制灯光
    if result.Won {
        c.SetLights(LIGHT_WINNING)
    }
}
```

### 9.3 数据持久化策略

#### 实时持久化（必须）
- 投币事件：立即写入数据库
- 退币/彩票：操作完成后立即记录
- 故障事件：立即记录并告警

```go
// 实时持久化示例
func (c *STM32Controller) persistCriticalData(event Event) {
    // 使用事务确保数据一致性
    tx := db.Begin()
    defer tx.Rollback()
    
    switch event.Type {
    case EVENT_COIN_IN:
        if err := tx.Create(&CoinRecord{
            Type: "insert",
            Count: event.Count,
            Time: time.Now(),
        }).Error; err != nil {
            log.Error("持久化失败:", err)
            // 写入本地缓存
            c.cacheQueue.Push(event)
            return
        }
        
    case EVENT_FAULT:
        // 故障立即记录并发送告警
        if err := tx.Create(&FaultRecord{
            Code: event.FaultCode,
            Level: event.Level,
            Time: time.Now(),
        }).Error; err != nil {
            log.Error("故障记录失败:", err)
        }
        // 发送告警
        c.sendAlert(event)
    }
    
    tx.Commit()
}
```

#### 批量持久化（可选）
- 回币统计：每10个币或每分钟批量写入
- 游戏日志：每局结束批量写入
- 性能数据：每5分钟批量更新

```go
// 批量持久化示例
type BatchPersister struct {
    buffer    []Event
    maxSize   int
    interval  time.Duration
    ticker    *time.Ticker
}

func (p *BatchPersister) Add(event Event) {
    p.buffer = append(p.buffer, event)
    
    // 达到批量大小，立即持久化
    if len(p.buffer) >= p.maxSize {
        p.Flush()
    }
}

func (p *BatchPersister) Flush() {
    if len(p.buffer) == 0 {
        return
    }
    
    // 批量写入
    if err := db.CreateInBatches(p.buffer, 100).Error; err != nil {
        log.Error("批量持久化失败:", err)
        // 降级到文件存储
        p.saveToFile(p.buffer)
    }
    
    p.buffer = p.buffer[:0]
}
```

#### 持久化失败处理
1. 本地缓存队列（最大1000条）
2. 重试机制（指数退避）
3. 降级到文件存储
4. 恢复后补充同步

```go
type PersistenceManager struct {
    primaryDB   *gorm.DB
    cacheQueue  *Queue
    fileBackup  *FileStorage
    retryPolicy *RetryPolicy
}

func (m *PersistenceManager) Persist(data interface{}) error {
    // 1. 尝试写入主数据库
    if err := m.primaryDB.Create(data).Error; err == nil {
        return nil
    }
    
    // 2. 写入缓存队列
    if m.cacheQueue.Size() < 1000 {
        m.cacheQueue.Push(data)
    }
    
    // 3. 降级到文件存储
    if err := m.fileBackup.Save(data); err != nil {
        log.Critical("数据持久化完全失败")
        return err
    }
    
    // 4. 启动重试任务
    go m.retryPolicy.Retry(func() error {
        return m.syncCachedData()
    })
    
    return nil
}
```

---

## 第五部分：测试与维护

## 10. 测试与验证

### 10.1 功能测试清单
- [ ] 串口通信建立
- [ ] 投币检测响应
- [ ] 上币精确控制
- [ ] 退币正确执行
- [ ] 彩票发放计数
- [ ] 推币电机控制
- [ ] 回币位置识别
- [ ] 灯光效果切换
- [ ] 故障检测上报
- [ ] 心跳保持连接

### 10.2 压力测试
- [ ] 连续投币100次
- [ ] 批量上币1000个
- [ ] 24小时稳定运行
- [ ] 断线重连测试
- [ ] 并发指令处理

### 10.3 异常测试
- [ ] 串口断开恢复
- [ ] 指令超时处理
- [ ] 错误参数拒绝
- [ ] 故障状态处理
- [ ] 缓冲区溢出保护

### 10.4 详细测试用例

#### TC001: 投币响应时序测试
**目的**：验证投币检测到ACK响应时间<100ms
**步骤**：
1. 模拟投币信号
2. 记录STM32上报时间T1
3. 记录Golang ACK时间T2
**预期**：T2-T1 < 100ms
**实际结果**：______

#### TC002: 命令防重测试
**目的**：验证相同序列号命令不会重复执行
**步骤**：
1. 发送上币命令(seq=0x0001, count=5)
2. 等待ACK确认
3. 再次发送相同命令(seq=0x0001, count=5)
**预期**：第二次返回上次结果，不执行上币
**实际结果**：______

#### TC003: 通信中断恢复测试
**目的**：验证通信中断后能自动恢复
**步骤**：
1. 建立正常通信
2. 断开串口连接
3. 等待检测到中断（心跳超时）
4. 重新连接串口
5. 发送状态查询
**预期**：自动重连成功，状态同步正常
**实际结果**：______

#### TC004: 并发命令测试
**目的**：验证不同类型命令可以并发执行
**步骤**：
1. 同时发送上币命令和灯光控制命令
2. 观察执行情况
**预期**：两个命令并行执行，互不影响
**实际结果**：______

#### TC005: 回币率统计准确性测试
**目的**：验证回币率计算的准确性
**步骤**：
1. 清零统计数据
2. 模拟回币事件：前方10个，左侧5个，右侧3个
3. 检查回币率计算结果
**预期**：回币率 = 10/(10+5+3) = 55.56%
**实际结果**：______

## 11. 版本管理与兼容性

### 11.1 协议版本
- 当前版本：v1.1
- 版本标识：在心跳包中携带版本号

### 11.2 向后兼容原则
- 新增命令码不影响旧版本
- 数据字段只能追加，不能修改
- 必须支持最近3个版本

### 11.3 版本协商
```
// 心跳包携带版本信息
Golang→STM32: [0x31][序列号][时间戳:4][版本:2][CRC]
STM32→Golang: [0x31][序列号][时间戳:4][运行时间:4][版本:2][CRC]
```

### 11.4 版本差异处理
```go
// 根据版本选择协议处理器
func (c *STM32Controller) selectProtocolHandler(version uint16) ProtocolHandler {
    switch version {
    case 0x0100: // v1.0
        return &ProtocolV10Handler{}
    case 0x0101: // v1.1
        return &ProtocolV11Handler{}
    default:
        // 使用最接近的兼容版本
        return &ProtocolV10Handler{}
    }
}
```

## 12. 故障诊断指南

### 12.1 常见问题诊断

| 症状 | 可能原因 | 诊断方法 | 解决方案 |
|------|----------|----------|----------|
| 无法连接 | 串口配置错误 | 检查端口、波特率 | 修改配置文件 |
| 命令无响应 | 序列号错误 | 查看日志序列号 | 检查序列号生成 |
| 投币不识别 | 传感器故障 | 状态查询检查传感器 | 更换传感器 |
| 数据丢失 | 缓冲区溢出 | 检查缓冲区大小 | 增大缓冲区 |
| 心跳超时 | 网络延迟 | ping测试延迟 | 调整超时时间 |

### 12.2 日志级别配置

```go
// 日志级别定义
const (
    LOG_DEBUG   = 0 // 调试信息
    LOG_INFO    = 1 // 普通信息
    LOG_WARNING = 2 // 警告信息
    LOG_ERROR   = 3 // 错误信息
    LOG_FATAL   = 4 // 致命错误
)

// 根据运行环境设置日志级别
func setLogLevel() {
    if isProduction() {
        logger.SetLevel(LOG_WARNING)
    } else {
        logger.SetLevel(LOG_DEBUG)
    }
}
```

### 12.3 监控指标

```go
// 关键监控指标
type Metrics struct {
    // 通信指标
    PacketsSent     uint64
    PacketsReceived uint64
    PacketsLost     uint64
    AvgResponseTime float64
    
    // 业务指标
    TotalCoinsIn    uint64
    TotalCoinsOut   uint64
    ReturnRate      float64
    FaultCount      uint32
    
    // 性能指标
    CPUUsage        float64
    MemoryUsage     float64
    QueueDepth      int
}

// 定期上报监控数据
func (c *STM32Controller) reportMetrics() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        metrics := c.collectMetrics()
        c.sendToMonitoringSystem(metrics)
    }
}
```

## 13. 附录

### 13.1 CRC16计算
```c
uint16_t crc16_xmodem(const uint8_t *data, uint16_t length) {
    uint16_t crc = 0x0000;
    for (uint16_t i = 0; i < length; i++) {
        crc ^= (uint16_t)data[i] << 8;
        for (uint8_t j = 0; j < 8; j++) {
            if (crc & 0x8000) {
                crc = (crc << 1) ^ 0x1021;
            } else {
                crc <<= 1;
            }
        }
    }
    return crc;
}
```

### 13.2 命令码速查表

| 分类 | 命令码 | 功能 | 方向 |
|------|--------|------|------|
| **控制** | 0x01 | 上币控制 | Go→STM32 |
| | 0x02 | 退币控制 | Go→STM32 |
| | 0x03 | 彩票发放 | Go→STM32 |
| | 0x04 | 推币控制 | Go→STM32 |
| | 0x05 | 灯光控制 | Go→STM32 |
| **事件** | 0x11 | 投币检测 | STM32→Go |
| | 0x12 | 回币检测 | STM32→Go |
| | 0x13 | 按键事件 | STM32→Go |
| | 0x14 | 传感器事件 | STM32→Go |
| **状态** | 0x21 | 状态查询 | Go→STM32 |
| | 0x22 | 状态上报 | STM32→Go |
| | 0x23 | 故障上报 | STM32→Go |
| | 0x24 | 执行进度 | STM32→Go |
| | 0x25 | 故障恢复 | Go→STM32 |
| **系统** | 0x31 | 心跳包 | 双向 |
| | 0x80 | ACK确认 | 双向 |
| | 0x81 | NACK拒绝 | 双向 |

### 13.3 错误码对照表

| 错误码 | 含义 | 处理建议 |
|--------|------|----------|
| 0x01 | 命令不支持 | 检查协议版本 |
| 0x02 | 参数错误 | 检查参数范围 |
| 0x03 | 设备忙 | 等待后重试 |
| 0x04 | 硬件故障 | 检查硬件状态 |
| 0x05 | 校验失败 | 重新发送 |
| 0x06 | 超出范围 | 调整参数 |
| 0x07 | 资源不足 | 补充资源 |

### 13.4 常见问题

**Q: STM32需要保存游戏状态吗？**
A: 不需要。所有游戏状态由Golang系统管理，STM32断电重启后等待Golang指令即可。

**Q: 如何处理指令执行中断电？**
A: STM32上电后上报当前硬件状态，Golang根据状态决定后续操作。

**Q: 投币信号抖动如何处理？**
A: STM32硬件层面消抖（建议20ms），确保上报的都是有效投币。

**Q: 多个事件同时发生怎么办？**
A: STM32使用事件队列，按时间顺序依次上报，Golang依次处理。

**Q: 序列号溢出如何处理？**
A: 当序列号达到0xFFFF时，下一个序列号回到0x0001（Golang）或0x0002（STM32）。

**Q: 如何保证数据不丢失？**
A: 关键数据实时持久化，批量数据定期保存，配合本地缓存和文件备份。

---

**文档结束**

如需技术支持，请联系：
- STM32硬件问题：Qxiong
- Golang软件问题：xinba