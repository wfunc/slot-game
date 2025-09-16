package slot

import (
	"context"
	"fmt"
)

// GoldenWildCascadeEngine 带金色Wild的消除式引擎
type GoldenWildCascadeEngine struct {
	*CascadeEngine
	
	// 金色Wild配置
	goldenWildConfig *GoldenWildConfig
	
	// Wild状态追踪
	wildTracker *WildTracker
	
	// 1024线匹配器
	line1024Matcher *Line1024Matcher
	
	// Animal触发检测器
	triggerDetector *TriggerDetector
}

// GoldenWildConfig 金色Wild配置
type GoldenWildConfig struct {
	// 金色符号概率（每个普通符号变金色的概率）
	GoldenProbability float64 `json:"golden_probability"` // 默认0.1 (10%)
	
	// 哪些符号可以变成金色
	GoldenEnabledSymbols []int `json:"golden_enabled_symbols"` // [0,1,2,3,4,5]
	
	// Wild符号ID
	WildSymbolID int `json:"wild_symbol_id"` // 默认-1表示Wild
	
	// Wild持续规则
	WildPersistence bool `json:"wild_persistence"` // Wild是否持续到下次消除
	
	// Wild显示符号
	WildDisplayChar string `json:"wild_display_char"` // "W" 或 "野"
}

// WildTracker Wild状态跟踪器
type WildTracker struct {
	// 当前网格中的Wild位置
	ActiveWilds map[string]GamePosition `json:"active_wilds"` // key: "row_col"
	
	// Wild的来源符号ID（用于显示）
	WildSources map[string]int `json:"wild_sources"` // key: "row_col", value: original_symbol_id
	
	// 本轮新生成的Wild
	NewWilds map[string]GamePosition `json:"new_wilds"`
}

// GoldenWildResult 金色Wild结果
type GoldenWildResult struct {
	*CascadeResult
	
	// Wild特定数据
	InitialGrid      [][]int            `json:"initial_grid"`      // 真正的初始网格
	GoldenSymbols    []GoldenSymbolInfo `json:"golden_symbols"`    // 金色符号信息
	WildTransitions  []WildTransition   `json:"wild_transitions"`  // Wild转换记录
	FinalWildCount   int                `json:"final_wild_count"`  // 最终Wild数量
	WildPositions    []GamePosition     `json:"wild_positions"`    // Wild位置
	
	// Animal触发数据
	AnimalTrigger    *AnimalTriggerData `json:"animal_trigger,omitempty"` // Animal游戏触发数据
}

// GoldenSymbolInfo 金色符号信息
type GoldenSymbolInfo struct {
	Position     GamePosition `json:"position"`      // 位置
	OriginalID   int          `json:"original_id"`   // 原始符号ID
	IsGolden     bool         `json:"is_golden"`     // 是否为金色
	BecameWild   bool         `json:"became_wild"`   // 是否变成了Wild
}

// WildTransition Wild转换记录
type WildTransition struct {
	Step         int          `json:"step"`          // 发生在哪一步
	Position     GamePosition `json:"position"`      // 位置
	FromSymbol   int          `json:"from_symbol"`   // 原始符号
	ToWild       bool         `json:"to_wild"`       // 是否变成Wild
	UsedInMatch  bool         `json:"used_in_match"` // 是否参与匹配
	Disappeared  bool         `json:"disappeared"`   // 是否消失
}

// NewGoldenWildCascadeEngine 创建金色Wild消除式引擎
func NewGoldenWildCascadeEngine(algorithmConfig *AlgorithmConfig, cascadeConfig *CascadeConfig) *GoldenWildCascadeEngine {
	baseEngine := NewCascadeEngine(algorithmConfig, cascadeConfig)
	
	goldenConfig := &GoldenWildConfig{
		GoldenProbability:    0.12, // 12%概率变金色
		GoldenEnabledSymbols: []int{0, 1, 2, 3, 4}, // 前5个符号可以变金色
		WildSymbolID:         -1,   // 用-1表示Wild
		WildPersistence:      true, // Wild持续存在
		WildDisplayChar:      "野",
	}
	
	return &GoldenWildCascadeEngine{
		CascadeEngine:    baseEngine,
		goldenWildConfig: goldenConfig,
		wildTracker: &WildTracker{
			ActiveWilds: make(map[string]GamePosition),
			WildSources: make(map[string]int),
			NewWilds:    make(map[string]GamePosition),
		},
		line1024Matcher: NewLine1024Matcher(cascadeConfig),
		triggerDetector: NewTriggerDetector(GetDefaultAnimalTriggerConfig()),
	}
}

// SpinWithGoldenWild 执行带金色Wild的消除式旋转
func (e *GoldenWildCascadeEngine) SpinWithGoldenWild(ctx context.Context, request *SpinRequest) (*GoldenWildResult, error) {
	// 1. 生成初始网格（包含金色符号）
	initialGrid, goldenSymbols := e.generateGridWithGolden()
	
	// 保存真正的初始网格副本
	originalInitialGrid := make([][]int, len(initialGrid))
	for i := range initialGrid {
		originalInitialGrid[i] = make([]int, len(initialGrid[i]))
		copy(originalInitialGrid[i], initialGrid[i])
	}
	
	// 2. 执行连锁消除（支持Wild）
	cascadeSteps := []CascadeStep{}
	wildTransitions := []WildTransition{}
	currentGrid := initialGrid
	totalWin := int64(0)
	stepNumber := 1
	
	// 重置Wild跟踪器
	e.wildTracker.NewWilds = make(map[string]GamePosition)
	
	for stepNumber <= e.cascadeConfig.MaxCascades {
		// 查找1024线匹配（支持Wild替换）
		lineMatches := e.line1024Matcher.Find1024LineMatchesWithWild(
			currentGrid, 
			e.abstractEngine.GetAlgorithmConfig().PayTable,
			e.goldenWildConfig.WildSymbolID,
		)
		matches := e.line1024Matcher.ConvertToMatchGroups(lineMatches)
		if len(matches) == 0 {
			break // 没有更多匹配，结束连锁
		}
		
		// 保存消除前网格状态
		gridBefore := make([][]int, len(currentGrid))
		for i := range currentGrid {
			gridBefore[i] = make([]int, len(currentGrid[i]))
			copy(gridBefore[i], currentGrid[i])
		}
		
		// 记录Wild使用情况
		wildUsage := e.trackWildUsage(matches, stepNumber)
		wildTransitions = append(wildTransitions, wildUsage...)
		
		// 计算本步赢取
		stepWin := e.calculateStepWin(matches, stepNumber)
		
		// 消除匹配符号，处理Golden→Wild转换
		gridAfterRemove, newWilds := e.removeMatchesWithGoldenWild(currentGrid, matches, &goldenSymbols)
		
		// 记录新生成的Wild
		for _, wildPos := range newWilds {
			posKey := fmt.Sprintf("%d_%d", wildPos.Row, wildPos.Reel)
			e.wildTracker.ActiveWilds[posKey] = wildPos
			wildTransitions = append(wildTransitions, WildTransition{
				Step:       stepNumber,
				Position:   wildPos,
				FromSymbol: e.wildTracker.WildSources[posKey],
				ToWild:     true,
			})
		}
		
		// 重力下落和符号补充（保持Wild位置）
		gridAfterFill := e.applyGravityWithWild(gridAfterRemove)
		
		// 记录步骤（包含所有网格状态）
		step := CascadeStep{
			StepNumber:      stepNumber,
			RemovedGroups:   matches,
			StepWin:         stepWin,
			Multiplier:      e.getCascadeMultiplier(stepNumber),
			GridBefore:      gridBefore,      // 消除前
			GridAfterRemove: gridAfterRemove, // 消除后（有空位-1）
			GridAfter:       gridAfterFill,   // 重力填充后
		}
		cascadeSteps = append(cascadeSteps, step)
		
		totalWin += stepWin
		currentGrid = gridAfterFill
		stepNumber++
	}
	
	// 3. 检测Animal游戏触发
	var animalTrigger *AnimalTriggerData
	if e.triggerDetector != nil {
		// 使用原始网格检测触发（在消除之前的状态）
		animalTrigger = e.triggerDetector.DetectAnimalTrigger(originalInitialGrid)
	}
	
	// 4. 构建结果
	result := &GoldenWildResult{
		CascadeResult: &CascadeResult{
			AbstractGameResult: &AbstractGameResult{
				ResultID:    fmt.Sprintf("golden_wild_%s", request.GameRequest.SessionID),
				BetAmount:   request.GameRequest.BetAmount,
				TotalWin:    totalWin,
				IsWin:       totalWin > 0,
				ReelResults: currentGrid,
				Multiplier:  e.getCascadeMultiplier(len(cascadeSteps)),
			},
			CascadeCount:    len(cascadeSteps),
			TotalRemoved:    e.countTotalRemoved(cascadeSteps),
			CascadeDetails:  cascadeSteps,
			FinalMultiplier: e.getCascadeMultiplier(len(cascadeSteps)),
		},
		InitialGrid:     originalInitialGrid, // 保存真正的初始网格
		GoldenSymbols:   goldenSymbols,
		WildTransitions: wildTransitions,
		FinalWildCount:  len(e.wildTracker.ActiveWilds),
		WildPositions:   e.getWildPositions(),
		AnimalTrigger:   animalTrigger, // 添加触发数据
	}
	
	return result, nil
}

// generateGridWithGolden 生成包含金色符号和Animal触发符号的初始网格
func (e *GoldenWildCascadeEngine) generateGridWithGolden() ([][]int, []GoldenSymbolInfo) {
	grid := make([][]int, e.cascadeConfig.GridHeight)
	goldenSymbols := []GoldenSymbolInfo{}
	
	// Animal触发符号出现概率
	animalWildProb := 0.02    // 2%概率出现Animal Wild
	animalBonusProb := 0.01   // 1%概率出现Animal Bonus
	
	for i := range grid {
		grid[i] = make([]int, e.cascadeConfig.GridWidth)
		for j := range grid[i] {
			// 先检查是否生成特殊符号
			rand := e.abstractEngine.(*AbstractSlotEngine).randomGen.Next()
			
			if rand < animalBonusProb {
				// 生成Animal Bonus符号
				grid[i][j] = SYMBOL_ANIMAL_BONUS
			} else if rand < animalBonusProb + animalWildProb {
				// 生成Animal Wild符号
				grid[i][j] = SYMBOL_ANIMAL_WILD
			} else {
				// 生成普通符号
				symbolID := e.abstractEngine.(*AbstractSlotEngine).randomGen.NextInt(0, e.abstractEngine.GetAlgorithmConfig().SymbolCount)
				grid[i][j] = symbolID
				
				// 检查是否变成金色
				if e.canBeGolden(symbolID) && e.shouldBeGolden() {
					goldenInfo := GoldenSymbolInfo{
						Position:   GamePosition{Row: i, Reel: j},
						OriginalID: symbolID,
						IsGolden:   true,
						BecameWild: false, // 初始时还不是Wild
					}
					goldenSymbols = append(goldenSymbols, goldenInfo)
				}
			}
		}
	}
	
	return grid, goldenSymbols
}

// findMatchesWithWild 查找包含Wild的匹配
func (e *GoldenWildCascadeEngine) findMatchesWithWild(grid [][]int) []MatchGroup {
	matches := []MatchGroup{}
	visited := make([][]bool, len(grid))
	for i := range visited {
		visited[i] = make([]bool, len(grid[i]))
	}
	
	// 遍历每个位置
	for row := 0; row < len(grid); row++ {
		for col := 0; col < len(grid[row]); col++ {
			if !visited[row][col] {
				// 使用支持Wild的DFS查找
				positions := e.dfsMatchWithWild(grid, visited, row, col, grid[row][col])
				if len(positions) >= e.cascadeConfig.MinMatch {
					match := MatchGroup{
						SymbolID:  e.getEffectiveSymbolID(grid, row, col), // 获取有效符号ID
						Positions: positions,
						Count:     len(positions),
						Payout:    e.calculateMatchPayout(e.getEffectiveSymbolID(grid, row, col), len(positions)),
					}
					matches = append(matches, match)
				}
			}
		}
	}
	
	return matches
}

// dfsMatchWithWild 支持Wild的深度优先搜索
func (e *GoldenWildCascadeEngine) dfsMatchWithWild(grid [][]int, visited [][]bool, row, col, targetSymbol int) []GamePosition {
	if row < 0 || row >= len(grid) || col < 0 || col >= len(grid[0]) || visited[row][col] {
		return []GamePosition{}
	}
	
	currentSymbol := grid[row][col]
	
	// 检查是否匹配（考虑Wild）
	if !e.symbolsMatch(currentSymbol, targetSymbol) {
		return []GamePosition{}
	}
	
	visited[row][col] = true
	positions := []GamePosition{{Reel: col, Row: row}}
	
	// 相邻方向搜索
	directions := [][]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	if !e.cascadeConfig.AdjacentOnly {
		directions = append(directions, [][]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}...)
	}
	
	for _, dir := range directions {
		newRow, newCol := row+dir[0], col+dir[1]
		adjacent := e.dfsMatchWithWild(grid, visited, newRow, newCol, targetSymbol)
		positions = append(positions, adjacent...)
	}
	
	return positions
}

// symbolsMatch 检查符号是否匹配（支持Wild）
func (e *GoldenWildCascadeEngine) symbolsMatch(symbol1, symbol2 int) bool {
	// Wild可以匹配任何符号
	if symbol1 == e.goldenWildConfig.WildSymbolID || symbol2 == e.goldenWildConfig.WildSymbolID {
		return true
	}
	// 普通符号必须完全相同
	return symbol1 == symbol2
}

// getEffectiveSymbolID 获取有效符号ID（Wild的情况下返回匹配的符号）
func (e *GoldenWildCascadeEngine) getEffectiveSymbolID(grid [][]int, row, col int) int {
	symbol := grid[row][col]
	if symbol == e.goldenWildConfig.WildSymbolID {
		// Wild符号，需要找到周围匹配的符号
		return e.findMatchingSymbolForWild(grid, row, col)
	}
	return symbol
}

// findMatchingSymbolForWild 为Wild找到匹配的符号
func (e *GoldenWildCascadeEngine) findMatchingSymbolForWild(grid [][]int, row, col int) int {
	directions := [][]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	
	for _, dir := range directions {
		newRow, newCol := row+dir[0], col+dir[1]
		if newRow >= 0 && newRow < len(grid) && newCol >= 0 && newCol < len(grid[0]) {
			neighborSymbol := grid[newRow][newCol]
			if neighborSymbol != e.goldenWildConfig.WildSymbolID {
				return neighborSymbol
			}
		}
	}
	
	// 如果周围都是Wild，返回第一个可用符号
	return 0
}

// removeMatchesWithGoldenWild 消除匹配并处理金色→Wild转换
func (e *GoldenWildCascadeEngine) removeMatchesWithGoldenWild(grid [][]int, matches []MatchGroup, goldenSymbols *[]GoldenSymbolInfo) ([][]int, []GamePosition) {
	newGrid := make([][]int, len(grid))
	for i := range newGrid {
		newGrid[i] = make([]int, len(grid[i]))
		copy(newGrid[i], grid[i])
	}
	
	// 标记要消除的位置
	toRemove := make([][]bool, len(grid))
	for i := range toRemove {
		toRemove[i] = make([]bool, len(grid[i]))
	}
	
	// 记录哪些位置将被消除
	for _, match := range matches {
		for _, pos := range match.Positions {
			toRemove[pos.Row][pos.Reel] = true
		}
	}
	
	// 处理金色符号→Wild转换
	newWilds := []GamePosition{}
	for i, golden := range *goldenSymbols {
		pos := golden.Position
		if toRemove[pos.Row][pos.Reel] && golden.IsGolden {
			// 金色符号被消除，变成Wild
			newGrid[pos.Row][pos.Reel] = e.goldenWildConfig.WildSymbolID
			toRemove[pos.Row][pos.Reel] = false // 不消除，保留Wild
			
			// 记录Wild信息
			posKey := fmt.Sprintf("%d_%d", pos.Row, pos.Reel)
			e.wildTracker.WildSources[posKey] = golden.OriginalID
			newWilds = append(newWilds, pos)
			
			// 更新金色符号状态
			(*goldenSymbols)[i].BecameWild = true
		}
	}
	
	// 消除其他匹配符号
	for row := 0; row < len(newGrid); row++ {
		for col := 0; col < len(newGrid[row]); col++ {
			if toRemove[row][col] {
				newGrid[row][col] = -1 // 空位
				
				// 如果消除的是Wild，从追踪器中移除
				posKey := fmt.Sprintf("%d_%d", row, col)
				if _, exists := e.wildTracker.ActiveWilds[posKey]; exists {
					delete(e.wildTracker.ActiveWilds, posKey)
					delete(e.wildTracker.WildSources, posKey)
				}
			}
		}
	}
	
	return newGrid, newWilds
}

// applyGravityWithWild 应用重力（保持Wild位置）
func (e *GoldenWildCascadeEngine) applyGravityWithWild(grid [][]int) [][]int {
	newGrid := make([][]int, len(grid))
	for i := range newGrid {
		newGrid[i] = make([]int, len(grid[i]))
	}
	
	// 对每列应用重力
	for col := 0; col < len(grid[0]); col++ {
		// 收集非空符号（包括Wild）
		symbols := []int{}
		wildPositions := []int{} // 记录Wild在新位置的索引
		
		for row := len(grid) - 1; row >= 0; row-- {
			if grid[row][col] != -1 {
				symbols = append(symbols, grid[row][col])
				// 如果是Wild，记录其新位置
				if grid[row][col] == e.goldenWildConfig.WildSymbolID {
					wildPositions = append(wildPositions, len(symbols)-1)
				}
			}
		}
		
		// 在顶部生成新符号填补空位
		for len(symbols) < len(grid) {
			newSymbol := e.abstractEngine.(*AbstractSlotEngine).randomGen.NextInt(0, e.abstractEngine.GetAlgorithmConfig().SymbolCount)
			symbols = append(symbols, newSymbol)
		}
		
		// 填充列
		for row := 0; row < len(grid); row++ {
			newGrid[len(grid)-1-row][col] = symbols[row]
		}
		
		// 更新Wild追踪器的位置信息
		for _, wildIdx := range wildPositions {
			oldRow := len(grid) - 1 - wildIdx
			newRow := len(grid) - 1 - wildIdx
			oldKey := fmt.Sprintf("%d_%d", oldRow, col)
			newKey := fmt.Sprintf("%d_%d", newRow, col)
			
			if oldKey != newKey {
				if source, exists := e.wildTracker.WildSources[oldKey]; exists {
					e.wildTracker.WildSources[newKey] = source
					delete(e.wildTracker.WildSources, oldKey)
				}
				if _, exists := e.wildTracker.ActiveWilds[oldKey]; exists {
					e.wildTracker.ActiveWilds[newKey] = GamePosition{Row: newRow, Reel: col}
					delete(e.wildTracker.ActiveWilds, oldKey)
				}
			}
		}
	}
	
	return newGrid
}

// 辅助函数

// canBeGolden 检查符号是否可以变成金色
func (e *GoldenWildCascadeEngine) canBeGolden(symbolID int) bool {
	for _, enabledSymbol := range e.goldenWildConfig.GoldenEnabledSymbols {
		if symbolID == enabledSymbol {
			return true
		}
	}
	return false
}

// shouldBeGolden 检查是否应该变成金色
func (e *GoldenWildCascadeEngine) shouldBeGolden() bool {
	randomValue := e.abstractEngine.(*AbstractSlotEngine).randomGen.Next()
	return randomValue < e.goldenWildConfig.GoldenProbability
}

// trackWildUsage 跟踪Wild使用情况
func (e *GoldenWildCascadeEngine) trackWildUsage(matches []MatchGroup, step int) []WildTransition {
	transitions := []WildTransition{}
	
	for _, match := range matches {
		for _, pos := range match.Positions {
			posKey := fmt.Sprintf("%d_%d", pos.Row, pos.Reel)
			if _, isWild := e.wildTracker.ActiveWilds[posKey]; isWild {
				transition := WildTransition{
					Step:        step,
					Position:    pos,
					FromSymbol:  e.wildTracker.WildSources[posKey],
					ToWild:      false,
					UsedInMatch: true,
					Disappeared: true,
				}
				transitions = append(transitions, transition)
			}
		}
	}
	
	return transitions
}

// getWildPositions 获取当前Wild位置
func (e *GoldenWildCascadeEngine) getWildPositions() []GamePosition {
	positions := []GamePosition{}
	for _, pos := range e.wildTracker.ActiveWilds {
		positions = append(positions, pos)
	}
	return positions
}