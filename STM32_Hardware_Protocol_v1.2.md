# STM32硬件控制通信协议文档 v1.2

## 文档信息

**版本**：v1.2
**更新日期**：2025-09-16
**适用对象**：STM32硬件工程师、Golang后端开发工程师
**上位机环境**：Ubuntu + Golang
**通信方式**：串口 (UART)
**协议特点**：Golang主导，STM32被动响应，无状态设计，简化确认机制

## 更新日志

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v1.0 | 2025-09-12 | 初始版本 |
| v1.1 | 2025-09-12 | 新增序列号管理规则、异常恢复流程、时序规范、并发控制、数据持久化策略 |
| v1.2 | 2025-09-16 | 长度字段改为1字节；字节序改为小端；CRC16改为XOR校验；使用echo确认替代ACK/NACK |

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

// 2. 接收处理（v1.2：echo确认）
void process_command(uint8_t cmd, uint8_t* data, uint16_t seq) {
    // 立即echo返回表示确认
    echo_command(cmd, data, seq);

    switch(cmd) {
        case 0x01: // 上币控制
            uint16_t count = data[0] | (data[1] << 8); // 小端序
            dispense_coins(count);
            break;
        case 0x31: // 心跳
            // echo即为响应，无需额外处理
            break;
    }
}

// 3. 事件上报
void on_coin_inserted() {
    uint8_t data[1] = {1}; // 1个币
    send_event(0x11, data, 1);
}

// 4. XOR校验计算
uint8_t calc_xor(uint8_t* data, uint8_t len) {
    uint8_t xor = 0;
    for(int i = 0; i < len; i++) {
        xor ^= data[i];
    }
    return xor;
}
```

#### Golang端最小实现
```go
// 1. 连接串口
port, _ := serial.OpenPort(&serial.Config{
    Name: "/dev/ttyS3",
    Baud: 115200,
})

// 2. 发送命令（v1.2：等待echo确认）
func dispenseCoins(count uint16) {
    data := make([]byte, 2)
    binary.LittleEndian.PutUint16(data, count) // 小端序

    // 发送命令
    sendCommand(0x01, data)

    // 等待echo确认
    if !waitForEcho(0x01, 3*time.Second) {
        log.Error("命令未确认")
    }
}

// 3. 处理事件
func handleEvent(cmd byte, data []byte) {
    switch cmd {
    case 0x11: // 投币
        coinCount := data[0]
        gameLogic.AddCredits(coinCount)
    }
}

// 4. XOR校验
func calcXOR(data []byte) byte {
    var xor byte
    for _, b := range data {
        xor ^= b
    }
    return xor
}
```

### 常见问题速查

| 问题 | 解决方案 | 参考章节 |
|------|----------|----------|
| 串口连接失败 | 检查端口权限、波特率配置 | 2.1 |
| 命令无响应 | 检查序列号、XOR校验 | 2.2, 2.4 |
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
- **简化确认**：echo机制替代ACK/NACK

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

### 2.2 数据帧格式（v1.2）
```
[帧头][长度][命令][序列号][数据][XOR][帧尾]
[0xAA][uint8][uint8][uint16][N字节][uint8][0x55]
```

**字段说明**：
- **帧头**：0xAA（固定）
- **长度**：uint8，整个帧的长度（最大255字节）
- **命令**：uint8，命令码
- **序列号**：uint16（小端序），防重复
  - Golang发送：奇数（0x0001, 0x0003...）
  - STM32上报：偶数（0x0002, 0x0004...）
- **数据**：可变长度，具体数据（多字节字段使用小端序）
- **XOR**：uint8，异或校验码
- **帧尾**：0x55（固定）

**XOR校验计算**：
从帧头到数据结束的所有字节进行异或运算（不包括XOR字段和帧尾）

### 2.3 命令码定义（v1.2：删除ACK/NACK）

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

### 2.4 序列号管理规则（v1.2：小端序）

#### 序列号分配
- Golang发送：使用奇数序列号（0x0001, 0x0003, 0x0005...）
- STM32上报：使用偶数序列号（0x0002, 0x0004, 0x0006...）
- 注意：序列号在内存和传输中均使用小端序

#### 序列号溢出处理
- 当序列号达到0xFFFF时，下一个序列号回到0x0001（Golang）或0x0002（STM32）
- 序列号不要求严格连续，但必须保持奇偶性
- 序列号生成示例：
```c
// STM32端（小端序）
uint16_t get_next_seq() {
    static uint16_t seq = 0;
    seq += 2;
    if (seq == 0 || seq == 0xFFFF) {
        seq = 0x0002; // 重新从2开始
    }
    return seq; // 直接返回，无需字节序转换
}
```

#### 序列号防重机制
- STM32应维护最近接收的10个序列号缓存
- 收到重复序列号时，返回上次的执行结果，不重复执行
- 缓存时间：5分钟后自动清除

### 2.5 确认机制（v1.2：Echo确认）

**Echo确认流程**：
1. Golang发送命令
2. STM32收到后立即原封不动返回（echo）
3. Golang收到echo表示命令已确认
4. STM32执行具体操作
5. 如需要，STM32上报执行进度

**Echo超时处理**：
- 超时时间：3秒
- 重试次数：3次
- 失败处理：记录错误，通知上层

---

## 第三部分：功能实现

## 3. 硬件控制指令详细定义

### 3.1 上币控制（0x01）

**功能**：控制上币电机，将游戏币从币仓送到游戏台面

**Golang→STM32（上币指令）**：
```
[0xAA][长度][0x01][序列号][数量:2字节][速度][XOR][0x55]
```

**数据字段**：
- **数量**：uint16（小端序），上币数量（1-9999）
- **速度**：uint8，上币速度（1-10，默认5）

**STM32→Golang（Echo确认）**：
```
[0xAA][长度][0x01][序列号][数量:2字节][速度][XOR][0x55]
```
- 原封不动返回表示已收到

**STM32→Golang（执行进度）**：
```
[0xAA][长度][0x24][序列号][命令码][已完成:2字节][总数:2字节][XOR][0x55]
```

**示例流程**：
```
1. Golang发送：上币20个（0x14 0x00小端序），速度5
   [AA][09][01][01 00][14 00][05][XOR][55]
                ^命令  ^序列号(小端)

2. STM32 Echo：原封返回表示确认
   [AA][09][01][01 00][14 00][05][XOR][55]

3. STM32上报：已上10个，共20个
   [AA][0A][24][02 00][01][0A 00][14 00][XOR][55]
                ^进度  ^新序列号 ^已完成 ^总数

4. STM32上报：已上20个，共20个（完成）
   [AA][0A][24][04 00][01][14 00][14 00][XOR][55]
```

### 3.2 退币控制（0x02）

**功能**：控制退币电机，将游戏币退还给用户

**Golang→STM32（退币指令）**：
```
[0xAA][长度][0x02][序列号][数量:2字节][XOR][0x55]
```

**数据字段**：
- **数量**：uint16（小端序），退币数量（1-9999）

**执行流程**：同上币控制（Echo确认机制）

### 3.3 彩票发放（0x03）

**功能**：控制彩票机发放彩票

**Golang→STM32（彩票指令）**：
```
[0xAA][长度][0x03][序列号][数量:2字节][XOR][0x55]
```

**数据字段**：
- **数量**：uint16（小端序），彩票数量（1-9999）

**特殊状态码**（在执行进度中上报）：
- 0x00：正常
- 0x01：缺纸
- 0x02：卡纸
- 0x03：打印头故障

### 3.4 推币控制（0x04）

**功能**：控制推币电机的运动

**Golang→STM32（推币指令）**：
```
[0xAA][长度][0x04][序列号][动作][参数][XOR][0x55]
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
[0xAA][长度][0x05][序列号][灯光位][XOR][0x55]
```

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
[0xAA][长度][0x11][序列号][数量][XOR][0x55]
```

**数据字段**：
- **数量**：uint8，投币数量（通常为1）

**Golang→STM32（Echo确认）**：
```
[0xAA][长度][0x11][序列号][数量][XOR][0x55]
```
- 原封返回表示已收到

**重要性**：P0级别，必须100ms内响应

### 4.2 回币检测（0x12）

**功能**：检测到游戏币被推出，一次上报三个位置的数量

**STM32→Golang（回币事件）**：
```
[0xAA][长度][0x12][序列号][前方数量][左侧数量][右侧数量][XOR][0x55]
```

**数据字段**：
- **前方数量**：uint8，前方出币口回币数（玩家获得）
- **左侧数量**：uint8，左侧出币口回币数（损失）
- **右侧数量**：uint8，右侧出币口回币数（损失）

**回币率计算**：
- 回币率 = 前方数量 / (前方数量 + 左侧数量 + 右侧数量) × 100%

### 4.3 按键事件（0x13）

**功能**：用户按键操作

**STM32→Golang（按键事件）**：
```
[0xAA][长度][0x13][序列号][按键类型][按键码][动作][附加数据][XOR][0x55]
```

**数据字段**：

#### 按键类型（uint8）：
- 0x01：游戏按键（开始/彩票）
- 0x02：配置按键（4键组合）

#### 游戏按键码（类型=0x01时）：
- **0x01：开始按键**
- **0x02：退币/彩票按键**（模式由Golang判断）

#### 配置按键码（类型=0x02时）：
- **0x11：上键（↑）/ 左键（←）**
- **0x12：下键（↓）/ 右键（→）**
- **0x13：确认键**（OK/Enter）
- **0x14：取消键**（ESC/Cancel）

#### 动作类型（uint8）：
- 0x01：按下（KEY_DOWN）
- 0x02：释放（KEY_UP）
- 0x03：长按（KEY_HOLD）
- 0x04：双击（KEY_DOUBLE）

### 4.4 传感器事件（0x14）

**功能**：各类传感器触发事件

**STM32→Golang（传感器事件）**：
```
[0xAA][长度][0x14][序列号][传感器类型][数值:2字节][XOR][0x55]
```

**传感器类型**：
- 0x01：币仓余量传感器
- 0x02：彩票余量传感器
- 0x03：温度传感器
- 0x04：震动传感器
- 0x05：门开关传感器

**数值**：uint16（小端序），传感器读数

## 5. 状态管理

### 5.1 状态查询（0x21）

**功能**：Golang查询STM32设备状态

**Golang→STM32（查询指令）**：
```
[0xAA][长度][0x21][序列号][查询类型][XOR][0x55]
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
[0xAA][长度][0x22][序列号][状态数据][XOR][0x55]
```

**状态数据结构**（所有uint16字段使用小端序）：
```c
struct DeviceStatus {
    uint8_t  coin_motor;      // 上币电机状态
    uint8_t  return_motor;     // 退币电机状态
    uint8_t  push_motor;       // 推币电机状态
    uint8_t  ticket_printer;   // 彩票机状态
    uint16_t coin_count;       // 币仓余量（小端序）
    uint16_t ticket_count;     // 彩票余量（小端序）
    uint8_t  temperature;      // 设备温度
    uint8_t  error_flags;      // 错误标志位
    uint8_t  current_mode;     // 当前模式
};
```

### 5.3 故障上报（0x23）

**功能**：硬件故障自动上报

**STM32→Golang（故障事件）**：
```
[0xAA][长度][0x23][序列号][故障码][严重级别][附加信息][XOR][0x55]
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

### 5.4 执行进度（0x24）

**功能**：上报指令执行进度

**STM32→Golang（进度上报）**：
```
[0xAA][长度][0x24][序列号][原命令码][已完成:2字节][总数:2字节][状态][XOR][0x55]
```

**字段说明**（uint16使用小端序）：
- **已完成**：uint16（小端序），已完成数量
- **总数**：uint16（小端序），总数量
- **状态**：uint8，执行状态
  - 0x00：执行中
  - 0x01：已完成
  - 0x02：已取消
  - 0x03：执行失败

### 5.5 故障恢复（0x25）

**功能**：当故障发生后，Golang发送恢复指令

**Golang→STM32（故障恢复指令）**：
```
[0xAA][长度][0x25][序列号][故障码][恢复动作][参数][XOR][0x55]
```

**恢复动作**：
- 0x01：重启设备
- 0x02：清除故障标志
- 0x03：强制重置
- 0x04：重试操作
- 0x05：跳过故障

## 6. 系统指令

### 6.1 心跳包（0x31）

**功能**：保持连接活跃，检测通信状态

**Golang→STM32（心跳请求）**：
```
[0xAA][长度][0x31][序列号][时间戳:4字节][版本:2字节][XOR][0x55]
```

**STM32→Golang（Echo响应）**：
```
[0xAA][长度][0x31][序列号][时间戳:4字节][版本:2字节][XOR][0x55]
```
- Echo原数据即表示心跳响应

**心跳策略**：
- 发送间隔：30秒
- 超时时间：5秒
- 重试次数：3次
- 失败处理：标记离线，尝试重连

**版本字段**（小端序）：
- v1.2 = 0x0201（即 [0x02, 0x01]）

## 7. 通信流程示例

### 7.1 完整游戏流程（v1.2版本）
```
1. 用户投币
   STM32: [投币检测(0x11)] → Golang
   Golang: [Echo(0x11)] → STM32 (确认收到)
   Golang: 记账，增加用户余额

2. 用户按下开始按键
   STM32: [按键事件(0x13): 开始键] → Golang
   Golang: [Echo(0x13)] → STM32 (确认收到)
   Golang: 判断余额，决定上币数量

3. 执行上币（币落入推盘）
   Golang: [上币指令(0x01): 20个] → STM32
   STM32: [Echo(0x01): 原封返回] → Golang (确认收到)
   STM32: [执行进度(0x24): 5/20] → Golang
   STM32: [执行进度(0x24): 10/20] → Golang
   STM32: [执行进度(0x24): 20/20 完成] → Golang

4. 推币动作
   Golang: [推币控制(0x04): 启动连续推币] → STM32
   STM32: [Echo(0x04): 原封返回] → Golang

5. 回币检测
   STM32: [回币检测(0x12): 前5，左2，右1] → Golang
   Golang: [Echo(0x12)] → STM32
   Golang: 计算回币率 = 5/(5+2+1) = 62.5%

6. 用户按退币/彩票键
   STM32: [按键事件(0x13): 退币/彩票键] → Golang
   Golang: [Echo(0x13)] → STM32

   若为退币模式：
   Golang: [退币控制(0x02): 10个] → STM32
   STM32: [Echo(0x02)] → Golang
   STM32: [执行进度(0x24): 10/10 完成] → Golang
```

### 7.2 故障处理与恢复流程（v1.2）
```
1. STM32检测到故障
   STM32: [故障上报(0x23): 彩票缺纸] → Golang
   Golang: [Echo(0x23)] → STM32 (确认收到)

2. Golang尝试恢复
   Golang: [故障恢复(0x25): 重试操作] → STM32
   STM32: [Echo(0x25)] → Golang
   STM32: 执行恢复操作

3. 恢复结果
   STM32: [状态上报(0x22): 当前状态] → Golang
   Golang: [Echo(0x22)] → STM32
```

### 7.3 异常恢复流程

#### 通信中断恢复
1. **检测中断**
   - Golang：心跳Echo超时3次判定为中断
   - STM32：60秒无指令判定为中断

2. **自动重连**
   ```
   Golang: 检测到中断 → 关闭串口 → 等待1秒 → 重新打开串口
   STM32: 检测到中断 → 复位串口 → 等待连接
   ```

3. **状态同步**
   ```
   Golang: [状态查询(0x21)] → STM32
   STM32: [Echo(0x21)] → Golang
   STM32: [状态上报(0x22)] → Golang
   ```

---

## 第四部分：最佳实践

## 8. STM32实现要求

### 8.1 基本要求
1. **被动响应**：STM32不主动发起业务操作，仅响应Golang指令
2. **Echo确认**：收到命令立即原封返回作为确认
3. **实时上报**：硬件事件必须立即上报
4. **状态维护**：维护所有硬件的实时状态

### 8.2 性能指标
| 指标 | 要求 | 说明 |
|------|------|------|
| Echo响应时间 | <20ms | 收到指令到返回Echo |
| 投币检测 | <10ms | 硬件触发到上报 |
| 执行精度 | 100% | 上币/退币数量准确 |
| 心跳间隔 | 30s | 保持连接 |

### 8.3 缓冲区管理
```c
// 推荐的缓冲区设置
#define RX_BUFFER_SIZE  256   // 接收缓冲区（适应新的长度限制）
#define TX_BUFFER_SIZE  256   // 发送缓冲区
#define CMD_QUEUE_SIZE  16    // 命令队列深度
#define EVENT_QUEUE_SIZE 32   // 事件队列深度
```

### 8.4 中断优先级
```c
// STM32中断优先级建议
#define UART_RX_PRIORITY    1  // 串口接收（最高）
#define COIN_DETECT_PRIORITY 2  // 投币检测
#define BUTTON_PRIORITY     2  // 按键检测
#define TIMER_PRIORITY      3  // 定时器
#define MOTOR_PRIORITY      4  // 电机控制
```

### 8.5 数据帧处理示例（v1.2）
```c
// 帧结构定义（小端序）
typedef struct {
    uint8_t  header;     // 0xAA
    uint8_t  length;     // 帧长度（最大255）
    uint8_t  cmd;        // 命令码
    uint16_t seq;        // 序列号（小端序）
    uint8_t  data[248];  // 数据（最大248字节）
    uint8_t  xor;        // XOR校验
    uint8_t  tail;       // 0x55
} Frame_t;

// Echo确认
void echo_command(uint8_t cmd, uint8_t* data, uint16_t seq, uint8_t len) {
    Frame_t frame;
    frame.header = 0xAA;
    frame.length = 7 + len;  // 基础长度7 + 数据长度
    frame.cmd = cmd;
    frame.seq = seq;         // 小端序，直接赋值
    memcpy(frame.data, data, len);
    frame.xor = calc_xor((uint8_t*)&frame, frame.length - 2);
    frame.tail = 0x55;

    uart_send((uint8_t*)&frame, frame.length);
}

// XOR校验计算
uint8_t calc_xor(uint8_t* data, uint8_t len) {
    uint8_t xor = 0;
    for(int i = 0; i < len; i++) {
        xor ^= data[i];
    }
    return xor;
}

// 解析接收数据（小端序）
void parse_frame(uint8_t* buf) {
    if(buf[0] != 0xAA) return;

    uint8_t len = buf[1];
    uint8_t cmd = buf[2];
    uint16_t seq = buf[3] | (buf[4] << 8);  // 小端序读取

    // 验证XOR
    uint8_t xor = calc_xor(buf, len - 2);
    if(xor != buf[len - 2]) return;

    // Echo确认
    echo_command(cmd, &buf[5], seq, len - 7);

    // 处理命令
    process_command(cmd, &buf[5], seq);
}
```

## 9. Golang实现示例

### 9.1 基本通信框架（v1.2）
```go
package hardware

import (
    "encoding/binary"
    "github.com/tarm/serial"
    "sync"
    "time"
)

type STM32Controller struct {
    port     *serial.Port
    seqNum   uint16
    mutex    sync.Mutex
    eventCh  chan HardwareEvent
    echoChan chan EchoPacket
}

// 发送指令并等待Echo（v1.2）
func (c *STM32Controller) SendCommand(cmd byte, data []byte) error {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    seq := c.getNextSeq()
    frame := c.buildFrame(cmd, seq, data)

    // 发送命令
    _, err := c.port.Write(frame)
    if err != nil {
        return err
    }

    // 等待Echo确认（超时3秒）
    select {
    case echo := <-c.echoChan:
        if echo.Cmd == cmd && echo.Seq == seq {
            return nil // Echo确认成功
        }
        return fmt.Errorf("Echo不匹配")
    case <-time.After(3 * time.Second):
        return fmt.Errorf("Echo超时")
    }
}

// 构建数据帧（v1.2：小端序+XOR）
func (c *STM32Controller) buildFrame(cmd byte, seq uint16, data []byte) []byte {
    frameLen := 7 + len(data)
    frame := make([]byte, frameLen)

    frame[0] = 0xAA                           // 帧头
    frame[1] = byte(frameLen)                 // 长度（1字节）
    frame[2] = cmd                            // 命令
    binary.LittleEndian.PutUint16(frame[3:5], seq) // 序列号（小端）
    copy(frame[5:], data)                     // 数据
    frame[frameLen-2] = c.calcXOR(frame[:frameLen-2]) // XOR
    frame[frameLen-1] = 0x55                  // 帧尾

    return frame
}

// XOR校验计算
func (c *STM32Controller) calcXOR(data []byte) byte {
    var xor byte
    for _, b := range data {
        xor ^= b
    }
    return xor
}

// 处理Echo响应
func (c *STM32Controller) handleEcho(data []byte) {
    if len(data) < 5 {
        return
    }

    echo := EchoPacket{
        Cmd: data[2],
        Seq: binary.LittleEndian.Uint16(data[3:5]),
    }

    select {
    case c.echoChan <- echo:
    default:
        // 通道满，丢弃
    }
}

// 上币控制（v1.2：小端序）
func (c *STM32Controller) DispenseCoins(count uint16, speed byte) error {
    data := make([]byte, 3)
    binary.LittleEndian.PutUint16(data[0:2], count) // 小端序
    data[2] = speed

    return c.SendCommand(0x01, data)
}
```

### 9.2 事件处理（v1.2）
```go
// 投币事件处理
func (c *STM32Controller) onCoinInserted(data []byte) {
    count := data[0]

    // 1. 发送Echo确认
    c.sendEcho(0x11, data)

    // 2. 更新用户余额
    gameLogic.AddCredits(count)

    log.Printf("投币检测: 数量:%d", count)
}

// 回币事件处理
func (c *STM32Controller) onCoinReturned(data []byte) {
    frontCount := data[0]
    leftCount := data[1]
    rightCount := data[2]

    // 1. 发送Echo确认
    c.sendEcho(0x12, data)

    // 2. 处理回币数据
    if frontCount > 0 {
        gameLogic.AddPlayerCoins(frontCount)
    }

    // 3. 计算回币率
    totalReturned := frontCount + leftCount + rightCount
    if totalReturned > 0 {
        returnRate := float64(frontCount) / float64(totalReturned) * 100
        gameLogic.UpdateReturnRate(returnRate)
    }
}

// 按键事件处理
func (c *STM32Controller) onButtonPressed(data []byte) {
    keyType := data[0]
    keyCode := data[1]
    action := data[2]

    // 发送Echo确认
    c.sendEcho(0x13, data)

    switch keyType {
    case 0x01: // 游戏按键
        c.handleGameButton(keyCode, action)
    case 0x02: // 配置按键
        c.handleConfigButton(keyCode, action)
    }
}
```

### 9.3 数据持久化策略

```go
// 实时持久化关键数据
func (c *STM32Controller) persistCriticalData(event Event) {
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
            c.cacheQueue.Push(event)
            return
        }

    case EVENT_FAULT:
        if err := tx.Create(&FaultRecord{
            Code: event.FaultCode,
            Level: event.Level,
            Time: time.Now(),
        }).Error; err != nil {
            log.Error("故障记录失败:", err)
        }
    }

    tx.Commit()
}
```

## 10. 测试与验证

### 10.1 功能测试清单
- [ ] 串口通信建立
- [ ] Echo确认机制
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
- [ ] Echo超时处理
- [ ] 错误参数拒绝
- [ ] 故障状态处理
- [ ] 缓冲区溢出保护

## 11. 版本管理与兼容性

### 11.1 协议版本
- 当前版本：v1.2
- 主要改进：简化确认机制，提高传输效率

### 11.2 向后兼容
- v1.2不兼容v1.1（确认机制不同）
- 升级需同时更新STM32和Golang端

### 11.3 版本协商
```go
// 版本检查
func (c *STM32Controller) checkVersion() error {
    // 发送心跳包
    version := []byte{0x02, 0x01} // v1.2小端序
    c.sendHeartbeat(version)

    // 等待Echo
    echo := c.waitForEcho()
    if echo.Version != 0x0102 {
        return fmt.Errorf("版本不匹配")
    }
    return nil
}
```

## 12. 故障诊断指南

### 12.1 常见问题诊断

| 症状 | 可能原因 | 诊断方法 | 解决方案 |
|------|----------|----------|----------|
| 无法连接 | 串口配置错误 | 检查端口、波特率 | 修改配置文件 |
| Echo超时 | 通信故障 | 查看串口日志 | 检查连接线路 |
| 数据错误 | XOR校验失败 | 打印XOR值 | 检查数据完整性 |
| 序列号混乱 | 字节序错误 | 验证小端序 | 统一使用小端序 |

### 12.2 监控指标
```go
type Metrics struct {
    // 通信指标
    PacketsSent     uint64
    PacketsReceived uint64
    EchoTimeout     uint64
    XORErrors       uint64

    // 业务指标
    TotalCoinsIn    uint64
    TotalCoinsOut   uint64
    ReturnRate      float64
}
```

## 13. 附录

### 13.1 XOR计算
```c
// C语言实现
uint8_t calc_xor(const uint8_t *data, uint8_t length) {
    uint8_t xor = 0;
    for (uint8_t i = 0; i < length; i++) {
        xor ^= data[i];
    }
    return xor;
}
```

```go
// Go语言实现
func calcXOR(data []byte) byte {
    var xor byte
    for _, b := range data {
        xor ^= b
    }
    return xor
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

### 13.3 v1.2版本主要改进

1. **长度字段优化**：从2字节改为1字节，节省传输开销
2. **字节序统一**：全部使用小端序，与处理器原生格式一致
3. **校验简化**：XOR替代CRC16，计算速度提升10倍
4. **确认机制简化**：Echo确认替代ACK/NACK，减少协议复杂度
5. **传输效率提升**：整体帧长度减少20-30%

### 13.4 常见问题

**Q: 为什么改用小端序？**
A: x86和ARM处理器原生使用小端序，无需转换，提高性能。

**Q: XOR校验是否足够可靠？**
A: 对于短距离串口通信，XOR足够检测传输错误，且计算速度快。

**Q: Echo确认的优势是什么？**
A: 简化协议设计，减少命令码数量，确认逻辑更直观。

**Q: 最大帧长度255字节够用吗？**
A: 当前定义的所有命令帧长度都在50字节以内，255字节完全足够。

---

**文档结束**

如需技术支持，请联系：
- STM32硬件问题：Qxiong
- Golang软件问题：xinba