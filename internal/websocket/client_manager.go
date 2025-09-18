package websocket

import (
	"sync"

	"go.uber.org/zap"
)

// ClientManager 客户端管理器
type ClientManager struct {
	clients map[string]*ManagedClient // clientID -> client
	rooms   map[uint32][]*ManagedClient // roomID -> clients
	mu      sync.RWMutex
	logger  *zap.Logger
}

// ManagedClient 被管理的客户端
type ManagedClient struct {
	Client   *ProtocolClient
	RoomID   uint32
	PlayerID uint32
}

// NewClientManager 创建客户端管理器
func NewClientManager(logger *zap.Logger) *ClientManager {
	return &ClientManager{
		clients: make(map[string]*ManagedClient),
		rooms:   make(map[uint32][]*ManagedClient),
		logger:  logger,
	}
}

// AddClient 添加客户端
func (m *ClientManager) AddClient(client *ProtocolClient) {
	m.mu.Lock()
	defer m.mu.Unlock()

	managedClient := &ManagedClient{
		Client:   client,
		RoomID:   0,
		PlayerID: 0,
	}

	m.clients[client.ID] = managedClient

	m.logger.Info("[ClientManager] 客户端已添加",
		zap.String("client_id", client.ID))
}

// RemoveClient 移除客户端
func (m *ClientManager) RemoveClient(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	managedClient, exists := m.clients[clientID]
	if !exists {
		return
	}

	// 从房间中移除
	if managedClient.RoomID > 0 {
		m.removeFromRoom(managedClient)
	}

	delete(m.clients, clientID)

	m.logger.Info("[ClientManager] 客户端已移除",
		zap.String("client_id", clientID))
}

// JoinRoom 加入房间
func (m *ClientManager) JoinRoom(clientID string, roomID uint32, playerID uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	managedClient, exists := m.clients[clientID]
	if !exists {
		return
	}

	// 如果已经在其他房间，先离开
	if managedClient.RoomID > 0 && managedClient.RoomID != roomID {
		m.removeFromRoom(managedClient)
	}

	managedClient.RoomID = roomID
	managedClient.PlayerID = playerID

	// 添加到新房间
	if m.rooms[roomID] == nil {
		m.rooms[roomID] = make([]*ManagedClient, 0)
	}
	m.rooms[roomID] = append(m.rooms[roomID], managedClient)

	m.logger.Info("[ClientManager] 客户端加入房间",
		zap.String("client_id", clientID),
		zap.Uint32("room_id", roomID),
		zap.Uint32("player_id", playerID))
}

// LeaveRoom 离开房间
func (m *ClientManager) LeaveRoom(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	managedClient, exists := m.clients[clientID]
	if !exists || managedClient.RoomID == 0 {
		return
	}

	m.removeFromRoom(managedClient)
}

// removeFromRoom 从房间中移除（内部方法，需要持有锁）
func (m *ClientManager) removeFromRoom(managedClient *ManagedClient) {
	roomID := managedClient.RoomID
	if roomID == 0 {
		return
	}

	clients := m.rooms[roomID]
	for i, c := range clients {
		if c == managedClient {
			// 从切片中移除
			m.rooms[roomID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	// 如果房间为空，删除房间
	if len(m.rooms[roomID]) == 0 {
		delete(m.rooms, roomID)
	}

	managedClient.RoomID = 0
	managedClient.PlayerID = 0

	m.logger.Info("[ClientManager] 客户端离开房间",
		zap.String("client_id", managedClient.Client.ID),
		zap.Uint32("room_id", roomID))
}

// BroadcastToRoom 向房间广播消息
func (m *ClientManager) BroadcastToRoom(roomID uint32, msgID uint16, data []byte) {
	m.mu.RLock()
	clients := m.rooms[roomID]
	m.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	m.logger.Info("[ClientManager] 向房间广播消息",
		zap.Uint32("room_id", roomID),
		zap.Uint16("msg_id", msgID),
		zap.Int("client_count", len(clients)),
		zap.Int("data_len", len(data)))

	// 记录需要清理的客户端
	var disconnectedClients []string

	for _, managedClient := range clients {
		// 检查客户端是否连接
		if !managedClient.Client.IsConnected() {
			m.logger.Warn("[ClientManager] 客户端已断开，跳过广播",
				zap.String("client_id", managedClient.Client.ID))
			disconnectedClients = append(disconnectedClients, managedClient.Client.ID)
			continue
		}

		// 构建服务端消息
		msg := &ServerMessage{
			ErrorID:    0,
			DataStatus: 0,
			Flag:       0, // 推送消息的flag通常为0
			Cmd:        msgID,
			Data:       data,
		}

		// 发送消息
		if err := managedClient.Client.SendMessage(msg); err != nil {
			m.logger.Error("[ClientManager] 发送广播消息失败",
				zap.String("client_id", managedClient.Client.ID),
				zap.Error(err))
			// 如果发送失败，可能客户端已断开
			if err.Error() == "客户端已断开" || err.Error() == "客户端已关闭" {
				disconnectedClients = append(disconnectedClients, managedClient.Client.ID)
			}
		}
	}

	// 清理已断开的客户端
	for _, clientID := range disconnectedClients {
		m.RemoveClient(clientID)
	}
}

// GetClient 获取客户端
func (m *ClientManager) GetClient(clientID string) *ManagedClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.clients[clientID]
}

// GetRoomClients 获取房间内的所有客户端
func (m *ClientManager) GetRoomClients(roomID uint32) []*ManagedClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := m.rooms[roomID]
	result := make([]*ManagedClient, len(clients))
	copy(result, clients)
	return result
}