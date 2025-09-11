# STM32硬件控制通信协议文档

## 1. 概述

### 1.1 系统架构
```
Golang系统（Ubuntu）[主导] <-串口通信-> STM32[被动执行]
         |                                    |
    游戏逻辑/算法/MQTT                    硬件控制/传感器
```

### 1.2 职责划分
- **Golang系统（主导方）**：
  - 游戏逻辑处理
  - 算法执行
  - 用户账户管理
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
| **系统指令** |
| 0x31 | 心跳包 | 双向 | 保持连接 |
| 0x80 | ACK确认 | 双向 | 确认收到 |
| 0x81 | NACK拒绝 | 双向 | 执行失败 |

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
[0xAA][长度][0x05][序列号][模式][颜色RGB][亮度][CRC16][0x55]
```

**数据字段**：
- **模式**：uint8
  - 0x01：常亮
  - 0x02：闪烁
  - 0x03：呼吸
  - 0x04：流水
  - 0x05：关闭
- **颜色**：3字节（R,G,B）
- **亮度**：uint8（0-255）

## 4. 硬件事件上报详细定义

### 4.1 投币检测（0x11）

**功能**：检测到用户投币，立即上报

**STM32→Golang（投币事件）**：
```
[0xAA][长度][0x11][序列号][数量][币种][CRC16][0x55]
```

**数据字段**：
- **数量**：uint8，投币数量（通常为1）
- **币种**：uint8
  - 0x01：1元硬币
  - 0x02：游戏币
  - 0x03：代币

**Golang→STM32（ACK确认）**：
```
[0xAA][长度][0x80][序列号][0x11][0x00][CRC16][0x55]
```
- 序列号：投币事件的序列号
- 0x11：原始命令码（投币检测）
- 0x00：状态码（成功确认）

**重要性**：P0级别，必须100ms内响应

### 4.2 回币检测（0x12）

**功能**：检测到游戏币被推出

**STM32→Golang（回币事件）**：
```
[0xAA][长度][0x12][序列号][数量][位置][CRC16][0x55]
```

**数据字段**：
- **数量**：uint8，回币数量
- **位置**：uint8
  - 0x01：前方出币口（玩家获得）
  - 0x02：左侧出币口
  - 0x03：右侧出币口

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
  - 功能：用户按下后，Golang将投币的游戏币通过上币接口落币，开始游戏
  - 响应：Golang收到后执行 `上币控制(0x01)` 指令
  
- **0x02：彩票按键**
  - 功能：用户按下后，Golang调用彩票接口出票
  - 响应：Golang收到后执行 `彩票发放(0x03)` 指令

#### 配置按键码（类型=0x02时）：
- **0x11：上键**（↑）
- **0x12：下键**（↓）
- **0x13：左键**（←）
- **0x14：右键**（→）
- **0x15：确认键**（OK/Enter）
- **0x16：取消键**（ESC/Cancel）

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

2. **彩票按键长按2秒**：
```
[0xAA][0x00][0x0E][0x13][序列号][0x01][0x02][0x03][0x07][0xD0][CRC16][0x55]
                                   游戏类 彩票键 长按  2000ms
```

3. **配置上键按下**：
```
[0xAA][0x00][0x0C][0x13][序列号][0x02][0x11][0x01][CRC16][0x55]
                                   配置类 上键  按下
```

4. **配置组合键（上+确认）**：
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

## 6. 系统指令

### 6.1 心跳包（0x31）

**功能**：保持连接活跃，检测通信状态

**Golang→STM32（心跳请求）**：
```
[0xAA][长度][0x31][序列号][时间戳:4字节][CRC16][0x55]
```

**STM32→Golang（心跳响应）**：
```
[0xAA][长度][0x31][序列号][时间戳:4字节][运行时间:4字节][CRC16][0x55]
```

**心跳策略**：
- 发送间隔：30秒
- 超时时间：5秒
- 重试次数：3次
- 失败处理：标记离线，尝试重连

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

### 7.1 典型游戏流程
```
1. 用户投币
   STM32: [投币检测] → Golang
   Golang: [ACK确认] → STM32
   
2. Golang计算游戏结果（内部处理，不涉及STM32）
   
3. 执行上币
   Golang: [上币指令(20个)] → STM32
   STM32: [ACK确认] → Golang
   STM32: [执行进度(10/20)] → Golang
   STM32: [执行进度(20/20)] → Golang
   
4. 推币动作
   Golang: [推币控制(启动)] → STM32
   STM32: [ACK确认] → Golang
   
5. 检测回币
   STM32: [回币检测(前方5个)] → Golang
   Golang: [ACK确认] → STM32
   
6. 发放彩票
   Golang: [彩票发放(10张)] → STM32
   STM32: [ACK确认] → Golang
   STM32: [执行进度(10/10)] → Golang
```

### 7.2 故障处理流程
```
1. STM32检测到故障
   STM32: [故障上报(彩票缺纸)] → Golang
   Golang: [ACK确认] → STM32
   
2. Golang处理
   - 记录日志
   - 停止彩票相关操作
   - 通知维护（如需要）
   
3. 故障恢复
   维护人员添加彩票纸
   Golang: [状态查询] → STM32
   STM32: [状态上报(正常)] → Golang
```

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
} KeyConfig;

// 按键配置表
const KeyConfig key_map[] = {
    // 游戏按键
    {GPIOA, GPIO_PIN_0, 0x01, 0x01},  // 开始按键
    {GPIOA, GPIO_PIN_1, 0x02, 0x01},  // 彩票按键
    
    // 配置按键组
    {GPIOB, GPIO_PIN_0, 0x11, 0x02},  // 上键
    {GPIOB, GPIO_PIN_1, 0x12, 0x02},  // 下键
    {GPIOB, GPIO_PIN_2, 0x13, 0x02},  // 左键
    {GPIOB, GPIO_PIN_3, 0x14, 0x02},  // 右键
    {GPIOB, GPIO_PIN_4, 0x15, 0x02},  // 确认键
    {GPIOB, GPIO_PIN_5, 0x16, 0x02},  // 取消键
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

// 特殊组合键检测（配置菜单进入）
bool check_config_entry(void) {
    // 同时按住确认键+取消键3秒进入工程模式
    if (is_key_pressed(KEY_OK) && is_key_pressed(KEY_CANCEL)) {
        if (get_combo_press_time() >= 3000) {
            return true;
        }
    }
    return false;
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
        
    case 0x02: // 彩票按键
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
        case 0x15: // 确认键
            c.configMenu.SelectItem()
        case 0x16: // 取消键
            c.configMenu.Close()
        }
        
    case MENU_SETTING:
        switch keyCode {
        case 0x13: // 左键
            c.configMenu.DecrValue()
        case 0x14: // 右键
            c.configMenu.IncrValue()
        case 0x15: // 确认键
            c.configMenu.SaveSetting()
        case 0x16: // 取消键
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

## 11. 附录

### 11.1 CRC16计算
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

### 11.2 常见问题

**Q: STM32需要保存游戏状态吗？**
A: 不需要。所有游戏状态由Golang系统管理，STM32断电重启后等待Golang指令即可。

**Q: 如何处理指令执行中断电？**
A: STM32上电后上报当前硬件状态，Golang根据状态决定后续操作。

**Q: 投币信号抖动如何处理？**
A: STM32硬件层面消抖（建议20ms），确保上报的都是有效投币。

**Q: 多个事件同时发生怎么办？**
A: STM32使用事件队列，按时间顺序依次上报，Golang依次处理。

---

## 文档信息

**版本**：v1.0  
**创建日期**：2025-01-11  
**适用对象**：STM32硬件工程师  
**上位机环境**：Ubuntu + Golang  
**通信方式**：串口（UART）  
**协议特点**：Golang主导，STM32被动响应

**修订记录**：
- v1.0：初始版本，定义基础硬件控制协议

**联系方式**：
- Golang开发：[系统开发团队]
- STM32开发：[硬件开发团队]