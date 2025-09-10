# 金色Wild麻将拉霸机 API 接口文档

## 🎯 概述

这是一套完整的拉霸机游戏API，支持金色Wild麻将主题，提供RESTful API和WebSocket实时通信。

## 🚀 快速开始

### 启动服务器
```bash
cd /Users/mini/Documents/GitHub/wfunc/slot-game
go run cmd/api/main.go
```

服务器将在 `http://localhost:8080` 启动

### 运行测试客户端
```bash
go run test/api_test_client.go
```

## 📋 API 接口

### 基础信息
- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json`
- **WebSocket**: `ws://localhost:8080/ws/{sessionId}`

### 1. 健康检查
```http
GET /health
```

**响应**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0.0",
  "sessions": 5
}
```

### 2. 创建游戏会话
```http
POST /api/v1/session
```

**请求体**:
```json
{
  "player_id": "player_001",
  "initial_balance": 10000,
  "settings": {
    "bet_amount": 100,
    "auto_spin": false,
    "auto_spin_count": 0,
    "sound_enabled": true,
    "language": "zh-CN"
  }
}
```

**响应**:
```json
{
  "success": true,
  "message": "Session created successfully",
  "session": {
    "session_id": "session_player_001_1234567890",
    "player_id": "player_001",
    "balance": 10000,
    "total_bet": 0,
    "total_win": 0,
    "game_count": 0,
    "win_count": 0,
    "golden_count": 0,
    "wild_count": 0,
    "created_at": "2024-01-01T12:00:00Z",
    "last_played_at": "2024-01-01T12:00:00Z",
    "settings": { ... },
    "wild_state": []
  }
}
```

### 3. 获取游戏会话
```http
GET /api/v1/session/{sessionId}
```

**响应**: 与创建会话响应格式相同

### 4. 删除游戏会话
```http
DELETE /api/v1/session/{sessionId}
```

**响应**:
```json
{
  "success": true,
  "message": "Session deleted successfully"
}
```

### 5. 游戏旋转
```http
POST /api/v1/spin
```

**请求体**:
```json
{
  "session_id": "session_player_001_1234567890",
  "bet_amount": 100
}
```

**响应**:
```json
{
  "success": true,
  "message": "Spin completed successfully",
  "result": {
    "result_id": "cascade_session_player_001_1234567890",
    "bet_amount": 100,
    "total_win": 230,
    "is_win": true,
    "reel_results": [[...]], // 最终网格状态
    "multiplier": 1.5,
    "cascade_count": 2,
    "total_removed": 12,
    "cascade_details": [...], // 详细消除步骤
    "final_multiplier": 1.5,
    "initial_grid": [[...]], // 初始网格状态
    "golden_symbols": [...], // 金色符号信息
    "wild_positions": [...], // Wild位置
    "wild_transitions": [...], // Wild转换历史
    "final_wild_count": 1
  },
  "session": { ... }, // 更新后的会话信息
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 6. 获取游戏统计
```http
GET /api/v1/stats/{sessionId}
```

**响应**:
```json
{
  "success": true,
  "message": "Stats retrieved successfully",
  "total_bets": 1000,
  "total_wins": 850,
  "rtp": 0.85,
  "hit_rate": 0.45,
  "avg_cascade": 2.3
}
```

## 🌐 WebSocket 接口

### 连接
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/session_id_here');
```

### 消息格式
```json
{
  "type": "spin|result|error|heartbeat|connected|session_end",
  "session_id": "session_id",
  "data": { ... },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 发送旋转请求
```json
{
  "type": "spin",
  "session_id": "session_id",
  "data": {
    "session_id": "session_id",
    "bet_amount": 100
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 接收旋转结果
```json
{
  "type": "result",
  "session_id": "session_id", 
  "data": {
    "success": true,
    "result": { ... }, // 同HTTP API的result格式
    "session": { ... }
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 心跳检测
```javascript
// 发送心跳
ws.send(JSON.stringify({
  "type": "heartbeat",
  "session_id": "session_id",
  "data": {"message": "ping"},
  "timestamp": new Date().toISOString()
}));

// 接收心跳响应
{
  "type": "heartbeat",
  "session_id": "session_id",
  "data": {"message": "pong"},
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## 🎮 游戏逻辑说明

### 符号系统
- **ID 0-7**: 发财、红中、白板、八万、六筒、六条、三筒、二条
- **ID -1**: Wild符号，可以替代任何符号

### 金色符号机制
- **出现概率**: 12%
- **转换规则**: 金色符号被消除后变成Wild
- **持续性**: Wild只有参与消除时才会消失

### 1024线匹配规则
- **左到右连续**: 必须从第1列开始连续匹配
- **最少3个**: 需要至少3列连续有相同符号
- **All-Ways-Win**: 每列可以有多个相同符号

### 消除连锁系统
- **重力下落**: 消除符号后，上方符号下落填补空位
- **新符号生成**: 顶部生成新符号填补缺失位置
- **连锁倍数**: [1.0, 1.5, 2.0, 3.0, 5.0, 8.0, 12.0, 18.0, 25.0, 40.0]

## 🧪 前端集成示例

### JavaScript/TypeScript

```javascript
class SlotGameAPI {
  constructor(baseURL = 'http://localhost:8080') {
    this.baseURL = baseURL;
    this.sessionId = null;
  }

  async createSession(playerId, initialBalance = 10000) {
    const response = await fetch(`${this.baseURL}/api/v1/session`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        player_id: playerId,
        initial_balance: initialBalance
      })
    });
    
    const data = await response.json();
    if (data.success) {
      this.sessionId = data.session.session_id;
      return data.session;
    }
    throw new Error(data.message);
  }

  async spin(betAmount) {
    if (!this.sessionId) throw new Error('No active session');
    
    const response = await fetch(`${this.baseURL}/api/v1/spin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        session_id: this.sessionId,
        bet_amount: betAmount
      })
    });
    
    const data = await response.json();
    if (data.success) {
      return { result: data.result, session: data.session };
    }
    throw new Error(data.message);
  }

  connectWebSocket() {
    if (!this.sessionId) throw new Error('No active session');
    
    const ws = new WebSocket(`ws://localhost:8080/ws/${this.sessionId}`);
    
    ws.onopen = () => console.log('WebSocket connected');
    
    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      this.handleWebSocketMessage(msg);
    };
    
    return ws;
  }

  handleWebSocketMessage(msg) {
    switch (msg.type) {
      case 'connected':
        console.log('WebSocket connection confirmed');
        break;
      case 'result':
        const spinResult = JSON.parse(msg.data);
        this.onSpinResult(spinResult);
        break;
      case 'error':
        console.error('WebSocket error:', msg.data);
        break;
    }
  }

  onSpinResult(result) {
    // 处理旋转结果，更新UI
    if (result.result.is_win) {
      console.log(`🎉 中奖! 赢取 ${result.result.total_win} coins`);
    }
  }
}

// 使用示例
const game = new SlotGameAPI();
await game.createSession('player_123');
const result = await game.spin(100);
console.log(result);
```

### React Hook 示例

```jsx
import { useState, useEffect } from 'react';

const useSlotGame = (baseURL = 'http://localhost:8080') => {
  const [session, setSession] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [wsConnection, setWsConnection] = useState(null);

  const createSession = async (playerId, initialBalance = 10000) => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`${baseURL}/api/v1/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          player_id: playerId,
          initial_balance: initialBalance
        })
      });
      
      const data = await response.json();
      if (data.success) {
        setSession(data.session);
      } else {
        setError(data.message);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const spin = async (betAmount) => {
    if (!session) throw new Error('No active session');
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`${baseURL}/api/v1/spin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: session.session_id,
          bet_amount: betAmount
        })
      });
      
      const data = await response.json();
      if (data.success) {
        setSession(data.session);
        return data.result;
      } else {
        setError(data.message);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const connectWebSocket = () => {
    if (!session) return;
    
    const ws = new WebSocket(`ws://localhost:8080/ws/${session.session_id}`);
    
    ws.onopen = () => {
      setWsConnection(ws);
      console.log('WebSocket connected');
    };
    
    ws.onclose = () => {
      setWsConnection(null);
      console.log('WebSocket disconnected');
    };
    
    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      handleWebSocketMessage(msg);
    };
    
    return ws;
  };

  const handleWebSocketMessage = (msg) => {
    switch (msg.type) {
      case 'result':
        const spinResult = JSON.parse(msg.data);
        setSession(spinResult.session);
        // 触发结果回调
        break;
    }
  };

  return {
    session,
    loading,
    error,
    wsConnection,
    createSession,
    spin,
    connectWebSocket
  };
};

// 使用示例
function SlotGameComponent() {
  const { session, loading, error, createSession, spin } = useSlotGame();
  
  useEffect(() => {
    createSession('player_123', 10000);
  }, []);

  const handleSpin = async () => {
    const result = await spin(100);
    if (result && result.is_win) {
      alert(`🎉 中奖! 赢取 ${result.total_win} coins`);
    }
  };

  return (
    <div>
      {session && (
        <div>
          <p>余额: {session.balance} coins</p>
          <button onClick={handleSpin} disabled={loading}>
            {loading ? '旋转中...' : '开始游戏'}
          </button>
        </div>
      )}
    </div>
  );
}
```

## 🔧 错误处理

### HTTP 状态码
- **200**: 成功
- **201**: 创建成功
- **400**: 请求错误
- **404**: 资源不存在
- **500**: 服务器内部错误

### 错误响应格式
```json
{
  "success": false,
  "message": "错误描述",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 常见错误
- `Session not found`: 会话不存在或已过期
- `Insufficient balance`: 余额不足
- `Invalid bet amount`: 下注金额无效
- `Game execution failed`: 游戏执行失败

## 📊 性能说明

- **并发支持**: 支持多会话并发游戏
- **WebSocket**: 提供实时游戏体验
- **内存会话**: 当前使用内存存储，重启后会话丢失
- **建议**: 生产环境应集成数据库持久化存储

## 🛡️ 安全考虑

- **CORS**: 开发环境允许所有域名，生产环境需要配置
- **会话管理**: 建议添加会话超时和清理机制
- **输入验证**: 已实现基础参数验证
- **日志记录**: 包含基础访问日志

## 📈 扩展建议

1. **数据库集成**: 添加PostgreSQL/MySQL持久化存储
2. **认证系统**: 集成JWT或OAuth2认证
3. **限流机制**: 防止API滥用
4. **监控告警**: 添加性能监控和错误告警
5. **负载均衡**: 支持多实例部署
6. **缓存系统**: Redis缓存热门数据