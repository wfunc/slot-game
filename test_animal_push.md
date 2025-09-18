# Animal Push System Test Guide

## Current Status
✅ Push system is implemented and working
✅ Animals are generating with enter (1887) and leave (1888) messages
❌ No clients currently in rooms to receive pushes

## Problem
The frontend client connects but doesn't join a room, so it can't receive push messages.

## Solution
Frontend must send command **1801** to join the animal room:

```javascript
// Frontend needs to send this message to join room:
{
  cmd: 1801,     // 进入房间命令
  flag: 1,       // 请求标识
  data: protobuf // m_1801_tos protobuf data (can be empty for default room)
}
```

## Test Flow
1. Client connects via WebSocket
2. Client sends 1801 to join room → Server adds client to room
3. Server pushes 1887 when animals enter → Client receives
4. Server pushes 1888 when animals leave → Client receives
5. Client sends 1802 to leave room → Stop receiving pushes

## Current Log Analysis
- Animals are generating every 1-2 seconds
- Push messages are being created with proper protobuf
- BroadcastToRoom is called but finds no clients in room
- Solution: Frontend must stay in room (don't send 1802 immediately after 1801)