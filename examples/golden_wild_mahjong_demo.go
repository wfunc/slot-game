package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/wfunc/slot-game/internal/game/slot"
)

// 金色Wild麻将拉霸机演示
func main() {
	fmt.Println("🀄✨ 金色Wild麻将来了 - 消除式拉霸机演示")
	fmt.Println(strings.Repeat("=", 60))

	// 1. 创建消除式配置
	cascadeConfig := slot.GetDefaultCascadeConfig()
	cascadeConfig.GridWidth = 5    // 5列
	cascadeConfig.GridHeight = 4   // 4行
	cascadeConfig.MinMatch = 3     // 最少3连
	cascadeConfig.MaxCascades = 10 // 最多10连锁

	// 更激进的连锁倍数
	cascadeConfig.CascadeMultipliers = []float64{
		1.0, 1.5, 2.0, 3.0, 5.0, 8.0, 12.0, 18.0, 25.0, 40.0,
	}

	// 2. 创建算法配置（专门的麻将符号）
	algorithmConfig := &slot.AlgorithmConfig{
		ReelCount:   5,
		RowCount:    4,
		SymbolCount: 8, // 8种麻将符号
		TargetRTP:   0.96,
		MinRTP:      0.94,
		MaxRTP:      0.98,

		// 符号权重配置（平衡分布，便于形成连线）
		SymbolWeights: [][]int{
			{18, 16, 14, 12, 12, 10, 8, 6}, // 列1：发财权重最高
			{16, 18, 14, 12, 12, 10, 8, 6}, // 列2：红中权重最高
			{14, 16, 18, 12, 12, 10, 8, 6}, // 列3：白板权重最高
			{12, 14, 16, 18, 12, 10, 8, 6}, // 列4：八万权重最高
			{12, 12, 14, 16, 18, 12, 8, 6}, // 列5：筒条权重最高
		},

		// 赔付表（按符号价值递增）
		PayTable: map[int][]int64{
			0: {0, 0, 20, 60, 200}, // 发财
			1: {0, 0, 25, 75, 250}, // 红中
			2: {0, 0, 30, 90, 300}, // 白板
			3: {0, 0, 15, 45, 150}, // 八万
			4: {0, 0, 12, 36, 120}, // 六筒
			5: {0, 0, 10, 30, 100}, // 六条
			6: {0, 0, 8, 24, 80},   // 三筒
			7: {0, 0, 6, 18, 60},   // 二条
		},

		Algorithm:    slot.AlgorithmTypeClassic,
		Volatility:   0.55, // 中等波动率
		HitFrequency: 0.4,  // 适中命中率
	}

	// 3. 创建金色Wild消除式引擎
	goldenEngine := slot.NewGoldenWildCascadeEngine(algorithmConfig, cascadeConfig)

	// 4. 演示游戏
	ctx := context.Background()

	fmt.Println("🎮 开始金色Wild演示...")
	fmt.Println("\n💡 游戏规则:")
	fmt.Println("  • 普通符号：发财/红中/白板/八万/六筒/六条/三筒/二条")
	fmt.Println("  • 金色符号：12%概率出现，消除后变成Wild(野)")
	fmt.Println("  • Wild符号：可以当做任何符号参与连线")
	fmt.Println("  • Wild持续：只有参与消除时Wild才会消失")
	fmt.Println()

	totalWin := int64(0)
	totalBet := int64(0)
	winCount := 0
	goldenCount := 0
	wildCount := 0

	for gameCount := 1; gameCount <= 8; gameCount++ {
		fmt.Printf("🎯 第%d局游戏\n", gameCount)
		betAmount := int64(100)
		fmt.Printf("下注金额: %d coins\n", betAmount)
		totalBet += betAmount

		// 创建旋转请求
		request := &slot.SpinRequest{
			GameRequest: &slot.GameRequest{
				SessionID: fmt.Sprintf("golden_wild_%d", gameCount),
				BetAmount: betAmount,
				Metadata: map[string]interface{}{
					"game_type":  "golden_wild_mahjong",
					"debug_mode": true,
				},
			},
			ThemeID:     "mahjong",
			EnableTheme: false,
		}

		// 执行金色Wild旋转
		result, err := goldenEngine.SpinWithGoldenWild(ctx, request)
		if err != nil {
			log.Printf("❌ 游戏执行失败: %v", err)
			continue
		}

		// 显示初始网格（含金色符号）
		fmt.Println("📋 初始网格:")
		displayGridWithGolden(result.InitialGrid, result.GoldenSymbols)

		// 统计金色符号
		goldenInThisRound := len(result.GoldenSymbols)
		if goldenInThisRound > 0 {
			goldenCount += goldenInThisRound
			fmt.Printf("✨ 本局金色符号: %d个\n", goldenInThisRound)
		}

		// 显示结果
		if result.IsWin {
			winCount++
			totalWin += result.TotalWin
			displayGoldenWildResult(result)

			// 统计Wild生成
			if len(result.WildPositions) > 0 {
				wildCount += len(result.WildPositions)
			}
		} else {
			fmt.Println("❌ 未中奖")

			// 即使不中奖，也可能有Wild残留
			if len(result.WildPositions) > 0 {
				fmt.Printf("🎭 剩余Wild: %d个 (位置: ", len(result.WildPositions))
				for i, pos := range result.WildPositions {
					fmt.Printf("(%d,%d)", pos.Row+1, pos.Reel+1)
					if i < len(result.WildPositions)-1 {
						fmt.Print(", ")
					}
				}
				fmt.Println(")")
			}
		}

		// 显示运行统计
		if gameCount%4 == 0 {
			fmt.Printf("\n📊 阶段统计（前%d局）:\n", gameCount)
			fmt.Printf("  总下注: %d coins\n", totalBet)
			fmt.Printf("  总赢取: %d coins\n", totalWin)
			fmt.Printf("  中奖率: %d/%d (%.1f%%)\n", winCount, gameCount, float64(winCount)/float64(gameCount)*100)
			fmt.Printf("  金色符号出现: %d个\n", goldenCount)
			fmt.Printf("  Wild符号生成: %d个\n", wildCount)
			if totalBet > 0 {
				fmt.Printf("  当前RTP: %.3f\n", float64(totalWin)/float64(totalBet))
			}
		}

		fmt.Println(strings.Repeat("-", 50))
	}

	fmt.Println("\n🏆 演示完成！")
	fmt.Printf("\n📈 最终统计:\n")
	fmt.Printf("  总下注: %d coins\n", totalBet)
	fmt.Printf("  总赢取: %d coins\n", totalWin)
	fmt.Printf("  净结果: %+d coins\n", totalWin-totalBet)
	fmt.Printf("  中奖率: %.1f%%\n", float64(winCount)/8.0*100)
	fmt.Printf("  金色符号总数: %d个\n", goldenCount)
	fmt.Printf("  Wild符号总数: %d个\n", wildCount)
	if totalBet > 0 {
		fmt.Printf("  实际RTP: %.3f\n", float64(totalWin)/float64(totalBet))
	}

	fmt.Println("\n✨ 金色Wild特性总结:")
	fmt.Println("  • 金色符号12%概率出现")
	fmt.Println("  • 消除金色符号 → 变成Wild")
	fmt.Println("  • Wild可替代任何符号")
	fmt.Println("  • Wild只在参与消除时消失")
	fmt.Println("  • 连锁越多，倍数越高")
}

// displayGridWithGolden 显示网格（含金色符号标记）
func displayGridWithGolden(grid [][]int, goldenSymbols []slot.GoldenSymbolInfo) {
	symbolNames := []string{
		"发财", "红中", "白板", "八万", // 0-3
		"六筒", "六条", "三筒", "二条", // 4-7
	}

	// 创建金色符号位置映射
	goldenMap := make(map[string]bool)
	for _, golden := range goldenSymbols {
		key := fmt.Sprintf("%d_%d", golden.Position.Row, golden.Position.Reel)
		goldenMap[key] = golden.IsGolden
	}

	for row := 0; row < len(grid); row++ {
		fmt.Print("  ")
		for col := 0; col < len(grid[row]); col++ {
			symbolID := grid[row][col]
			key := fmt.Sprintf("%d_%d", row, col)

			var symbolDisplay string
			if symbolID == -1 {
				symbolDisplay = "[野]" // Wild符号
			} else if symbolID >= 0 && symbolID < len(symbolNames) {
				symbolDisplay = fmt.Sprintf("[%s]", symbolNames[symbolID])
			} else {
				symbolDisplay = fmt.Sprintf("[%d]", symbolID)
			}

			// 如果是金色符号，添加✨标记
			if goldenMap[key] {
				fmt.Printf("✨%s ", symbolDisplay)
			} else {
				fmt.Printf("%s ", symbolDisplay)
			}
		}
		fmt.Println()
	}
}

// displayGoldenWildResult 显示金色Wild结果
func displayGoldenWildResult(result *slot.GoldenWildResult) {
	fmt.Printf("🎊 中奖！总赢取: %d coins\n", result.TotalWin)
	fmt.Printf("🔗 连锁次数: %d\n", result.CascadeCount)
	fmt.Printf("📈 最终倍数: %.1fx\n", result.FinalMultiplier)
	fmt.Printf("🗑️  总消除数: %d个符号\n", result.TotalRemoved)

	// 显示金色符号转换
	goldenTurnedWild := 0
	for _, golden := range result.GoldenSymbols {
		if golden.BecameWild {
			goldenTurnedWild++
		}
	}
	if goldenTurnedWild > 0 {
		fmt.Printf("✨ 金色→Wild转换: %d个\n", goldenTurnedWild)
	}

	// 显示最终Wild状态
	if result.FinalWildCount > 0 {
		fmt.Printf("🎭 剩余Wild数量: %d个\n", result.FinalWildCount)
		fmt.Print("🎭 Wild位置: ")
		for i, pos := range result.WildPositions {
			fmt.Printf("(%d,%d)", pos.Row+1, pos.Reel+1)
			if i < len(result.WildPositions)-1 {
				fmt.Print(", ")
			}
		}
		fmt.Println()
	}

	// 显示Wild转换历史
	if len(result.WildTransitions) > 0 {
		fmt.Println("\n🎭 Wild转换记录:")
		for _, transition := range result.WildTransitions {
			symbolName := getMahjongSymbolName(transition.FromSymbol)
			if transition.ToWild {
				fmt.Printf("  步骤%d: %s → Wild (位置 %d,%d)\n",
					transition.Step, symbolName, transition.Position.Row+1, transition.Position.Reel+1)
			} else if transition.Disappeared {
				fmt.Printf("  步骤%d: Wild消失 (原%s, 位置 %d,%d)\n",
					transition.Step, symbolName, transition.Position.Row+1, transition.Position.Reel+1)
			}
		}
	}

	// 显示每步消除详情
	if len(result.CascadeDetails) > 0 {
		fmt.Println("\n📋 消除详情:")
		for _, step := range result.CascadeDetails {
			fmt.Printf("\n🔄 步骤%d: %d个匹配组, 赢取%d coins (%.1fx倍数)\n",
				step.StepNumber, len(step.RemovedGroups), step.StepWin, step.Multiplier)

			// 显示匹配组详情
			for i, group := range step.RemovedGroups {
				symbolName := getMahjongSymbolName(group.SymbolID)
				fmt.Printf("    匹配%d: %s x%d个 -> %d coins\n",
					i+1, symbolName, group.Count, group.Payout)

				// 显示匹配位置
				fmt.Print("      位置: ")
				for j, pos := range group.Positions {
					fmt.Printf("(%d,%d)", pos.Row+1, pos.Reel+1)
					if j < len(group.Positions)-1 {
						fmt.Print(", ")
					}
				}
				fmt.Println()
			}

			// 显示消除前网格
			fmt.Printf("    📋 消除前网格:\n")
			displaySimpleGrid(step.GridBefore)

			// 显示消除后网格（有空位）
			fmt.Printf("    🗑️  消除后网格（空位用[空]表示）:\n")
			displayGridWithEmpty(step.GridAfterRemove)

			// 显示重力填充后网格
			fmt.Printf("    ⬇️  重力填充后网格:\n")
			displaySimpleGrid(step.GridAfter)
		}
	}

	// 检查超级连击
	if result.CascadeCount >= 6 {
		fmt.Println("🔥 超级连击！连庄大胡！")
	}
	if result.CascadeCount >= 8 {
		fmt.Println("🌟 传奇连击！金龙献瑞！")
	}
}

// displaySimpleGrid 显示简单网格
func displaySimpleGrid(grid [][]int) {
	symbolNames := []string{
		"发财", "红中", "白板", "八万", // 0-3
		"六筒", "六条", "三筒", "二条", // 4-7
	}

	for row := 0; row < len(grid); row++ {
		fmt.Print("      ")
		for col := 0; col < len(grid[row]); col++ {
			symbolID := grid[row][col]
			if symbolID == -1 {
				fmt.Printf("[野] ")
			} else if symbolID >= 0 && symbolID < len(symbolNames) {
				fmt.Printf("[%s] ", symbolNames[symbolID])
			} else {
				fmt.Printf("[%d] ", symbolID)
			}
		}
		fmt.Println()
	}
}

// displayGridWithEmpty 显示带空位的网格
func displayGridWithEmpty(grid [][]int) {
	symbolNames := []string{
		"发财", "红中", "白板", "八万", // 0-3
		"六筒", "六条", "三筒", "二条", // 4-7
	}

	for row := 0; row < len(grid); row++ {
		fmt.Print("      ")
		for col := 0; col < len(grid[row]); col++ {
			symbolID := grid[row][col]
			if symbolID == -1 {
				// -1可能是Wild或空位，根据上下文判断
				fmt.Printf("[空空] ")
			} else if symbolID >= 0 && symbolID < len(symbolNames) {
				fmt.Printf("[%s] ", symbolNames[symbolID])
			} else {
				fmt.Printf("[%d] ", symbolID)
			}
		}
		fmt.Println()
	}
}

// getMahjongSymbolName 获取麻将符号名称
func getMahjongSymbolName(symbolID int) string {
	names := []string{
		"发财", "红中", "白板", "八万", // 0-3
		"六筒", "六条", "三筒", "二条", // 4-7
	}

	if symbolID >= 0 && symbolID < len(names) {
		return names[symbolID]
	}
	return fmt.Sprintf("符号%d", symbolID)
}
