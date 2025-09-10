package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wfunc/slot-game/api"
)

// APITestClient API测试客户端
type APITestClient struct {
	BaseURL    string
	HTTPClient *http.Client
	SessionID  string
}

// NewAPITestClient 创建测试客户端
func NewAPITestClient(baseURL string) *APITestClient {
	return &APITestClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TestCreateSession 测试创建会话
func (c *APITestClient) TestCreateSession() error {
	fmt.Println("🎯 测试创建游戏会话...")

	request := api.CreateSessionRequest{
		PlayerID:       "test_player_001",
		InitialBalance: 10000,
		Settings: &api.GameSettings{
			BetAmount:     100,
			AutoSpin:      false,
			AutoSpinCount: 0,
			SoundEnabled:  true,
			Language:      "zh-CN",
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/v1/session", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var response api.CreateSessionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("创建会话失败: %s", response.Message)
	}

	c.SessionID = response.Session.SessionID
	fmt.Printf("✅ 会话创建成功: %s\n", c.SessionID)
	fmt.Printf("   玩家ID: %s\n", response.Session.PlayerID)
	fmt.Printf("   初始余额: %d coins\n", response.Session.Balance)
	
	return nil
}

// TestGetSession 测试获取会话
func (c *APITestClient) TestGetSession() error {
	fmt.Println("\n🔍 测试获取游戏会话...")

	if c.SessionID == "" {
		return fmt.Errorf("会话ID为空，请先创建会话")
	}

	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/v1/session/" + c.SessionID)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var response api.GetSessionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("获取会话失败: %s", response.Message)
	}

	fmt.Printf("✅ 会话信息获取成功\n")
	fmt.Printf("   余额: %d coins\n", response.Session.Balance)
	fmt.Printf("   游戏次数: %d\n", response.Session.GameCount)
	fmt.Printf("   中奖次数: %d\n", response.Session.WinCount)

	return nil
}

// TestSpin 测试游戏旋转
func (c *APITestClient) TestSpin(betAmount int64) error {
	fmt.Printf("\n🎰 测试游戏旋转 (下注: %d coins)...\n", betAmount)

	if c.SessionID == "" {
		return fmt.Errorf("会话ID为空，请先创建会话")
	}

	request := api.SpinGameRequest{
		SessionID: c.SessionID,
		BetAmount: betAmount,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/v1/spin", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var response api.SpinGameResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("游戏旋转失败: %s", response.Message)
	}

	result := response.Result
	session := response.Session

	fmt.Printf("🎊 旋转完成!\n")
	fmt.Printf("   结果: %s\n", map[bool]string{true: "中奖 🎉", false: "未中奖"}[result.IsWin])
	
	if result.IsWin {
		fmt.Printf("   赢取金额: %d coins\n", result.TotalWin)
		fmt.Printf("   连锁次数: %d\n", result.CascadeCount)
		fmt.Printf("   最终倍数: %.1fx\n", result.FinalMultiplier)
		fmt.Printf("   消除符号数: %d个\n", result.TotalRemoved)
		
		if len(result.GoldenSymbols) > 0 {
			goldenCount := 0
			for _, g := range result.GoldenSymbols {
				if g.IsGolden {
					goldenCount++
				}
			}
			fmt.Printf("   金色符号: %d个\n", goldenCount)
		}

		if len(result.WildPositions) > 0 {
			fmt.Printf("   Wild符号: %d个\n", len(result.WildPositions))
		}
	}

	fmt.Printf("   当前余额: %d coins\n", session.Balance)
	fmt.Printf("   总游戏次数: %d\n", session.GameCount)
	fmt.Printf("   中奖率: %.1f%%\n", float64(session.WinCount)/float64(session.GameCount)*100)

	return nil
}

// TestGetStats 测试获取统计
func (c *APITestClient) TestGetStats() error {
	fmt.Println("\n📊 测试获取游戏统计...")

	if c.SessionID == "" {
		return fmt.Errorf("会话ID为空，请先创建会话")
	}

	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/v1/stats/" + c.SessionID)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var response api.GetStatsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("获取统计失败: %s", response.Message)
	}

	fmt.Printf("✅ 统计信息获取成功\n")
	fmt.Printf("   总下注: %d coins\n", response.TotalBets)
	fmt.Printf("   总赢取: %d coins\n", response.TotalWins)
	fmt.Printf("   实际RTP: %.3f\n", response.RTP)
	fmt.Printf("   命中率: %.1f%%\n", response.HitRate*100)

	return nil
}

// TestWebSocket 测试WebSocket连接
func (c *APITestClient) TestWebSocket() error {
	fmt.Println("\n🌐 测试WebSocket连接...")

	if c.SessionID == "" {
		return fmt.Errorf("会话ID为空，请先创建会话")
	}

	// 构建WebSocket URL
	u := url.URL{
		Scheme: "ws",
		Host:   "localhost:8080",
		Path:   "/ws/" + c.SessionID,
	}

	fmt.Printf("连接到WebSocket: %s\n", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("WebSocket连接失败: %v", err)
	}
	defer conn.Close()

	// 设置中断信号处理
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 启动消息接收goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var msg api.WebSocketMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("WebSocket读取失败:", err)
				return
			}

			fmt.Printf("📨 收到消息 [%s]: ", msg.Type)
			
			switch msg.Type {
			case "connected":
				fmt.Println("WebSocket连接确认")
			case api.WSMsgResult:
				var spinResponse api.SpinGameResponse
				if err := json.Unmarshal(msg.Data, &spinResponse); err == nil {
					result := spinResponse.Result
					if result.IsWin {
						fmt.Printf("中奖! 赢取 %d coins, 连锁 %d 次\n", result.TotalWin, result.CascadeCount)
					} else {
						fmt.Println("未中奖")
					}
				}
			case api.WSMsgHeartbeat:
				fmt.Println("心跳响应")
			case api.WSMsgError:
				fmt.Printf("错误: %s\n", string(msg.Data))
			default:
				fmt.Printf("未知类型: %s\n", string(msg.Data))
			}
		}
	}()

	// 发送几个测试旋转
	for i := 0; i < 3; i++ {
		spinRequest := api.SpinGameRequest{
			SessionID: c.SessionID,
			BetAmount: 100,
		}
		
		spinData, _ := json.Marshal(spinRequest)
		msg := api.WebSocketMessage{
			Type:      api.WSMsgSpin,
			SessionID: c.SessionID,
			Data:      spinData,
			Timestamp: time.Now(),
		}

		fmt.Printf("🎰 发送WebSocket旋转请求 #%d\n", i+1)
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("WebSocket发送失败: %v", err)
			break
		}

		time.Sleep(2 * time.Second)
	}

	// 发送心跳测试
	heartbeatMsg := api.WebSocketMessage{
		Type:      api.WSMsgHeartbeat,
		SessionID: c.SessionID,
		Data:      json.RawMessage(`{"message": "ping"}`),
		Timestamp: time.Now(),
	}
	
	fmt.Println("💓 发送心跳测试")
	if err := conn.WriteJSON(heartbeatMsg); err != nil {
		log.Printf("心跳发送失败: %v", err)
	}

	// 等待中断或完成
	select {
	case <-done:
		fmt.Println("WebSocket连接关闭")
	case <-interrupt:
		fmt.Println("收到中断信号")
		// 优雅关闭连接
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("WebSocket关闭错误:", err)
			return err
		}
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}

	fmt.Println("✅ WebSocket测试完成")
	return nil
}

// TestHealthCheck 测试健康检查
func (c *APITestClient) TestHealthCheck() error {
	fmt.Println("🏥 测试健康检查...")

	resp, err := c.HTTPClient.Get(c.BaseURL + "/health")
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	fmt.Printf("✅ 服务器健康状态: %v\n", response["status"])
	fmt.Printf("   版本: %v\n", response["version"])
	fmt.Printf("   活跃会话: %.0f\n", response["sessions"])

	return nil
}

// RunAllTests 运行所有测试
func (c *APITestClient) RunAllTests() {
	fmt.Println("🚀 开始API全面测试...")
	fmt.Println(fmt.Sprintf("🎯 目标服务器: %s", c.BaseURL))
	fmt.Println(strings.Repeat("=", 60))

	tests := []struct {
		name string
		fn   func() error
	}{
		{"健康检查", c.TestHealthCheck},
		{"创建会话", c.TestCreateSession},
		{"获取会话", c.TestGetSession},
		{"游戏旋转 #1", func() error { return c.TestSpin(100) }},
		{"游戏旋转 #2", func() error { return c.TestSpin(200) }},
		{"游戏旋转 #3", func() error { return c.TestSpin(100) }},
		{"获取统计", c.TestGetStats},
	}

	successCount := 0
	for _, test := range tests {
		if err := test.fn(); err != nil {
			fmt.Printf("❌ %s失败: %v\n", test.name, err)
		} else {
			fmt.Printf("✅ %s成功\n", test.name)
			successCount++
		}
		time.Sleep(500 * time.Millisecond) // 短暂延迟
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("📊 测试结果: %d/%d 通过\n", successCount, len(tests))
	
	if successCount == len(tests) {
		fmt.Println("🎉 所有API测试通过!")
		fmt.Println("\n🌐 是否要测试WebSocket? (按Ctrl+C退出WebSocket测试)")
		if err := c.TestWebSocket(); err != nil {
			fmt.Printf("❌ WebSocket测试失败: %v\n", err)
		}
	} else {
		fmt.Printf("⚠️  有 %d 个测试失败，请检查服务器状态\n", len(tests)-successCount)
	}
}

func main() {
	fmt.Println("🀄✨ 金色Wild麻将拉霸机 API 测试客户端")
	fmt.Println("================================")

	// 创建测试客户端
	client := NewAPITestClient("http://localhost:8080")

	// 运行所有测试
	client.RunAllTests()
}