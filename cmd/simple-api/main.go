package main

import (
	"github.com/wfunc/slot-game/api"
)

func main() {
	// 创建简化版游戏API
	gameAPI := api.NewSimpleGameAPI()

	// 启动服务器
	gameAPI.Start("8080")
}