package slot

import (
	"context"
	"fmt"
)

// CascadeEngine 消除式拉霸机引擎 - 1024线玩法
type CascadeEngine struct {
	*CompositeSlotEngine
	
	// 消除式特定配置
	cascadeConfig *CascadeConfig
	
	// 消除统计
	cascadeStats *CascadeStatistics
}

// CascadeConfig 消除式配置
type CascadeConfig struct {
	GridWidth    int     `json:"grid_width"`     // 网格宽度（默认5）
	GridHeight   int     `json:"grid_height"`    // 网格高度（默认4）
	MinMatch     int     `json:"min_match"`      // 最少连线数量（默认3）
	MaxCascades  int     `json:"max_cascades"`   // 最大连锁次数（默认10）
	
	// 倍数设置
	CascadeMultipliers []float64 `json:"cascade_multipliers"` // [1.0, 2.0, 3.0, 5.0, 8.0...]
	
	// 连线方向
	AdjacentOnly bool `json:"adjacent_only"` // 仅相邻连线（不包括对角线）
}

// CascadeResult 消除结果
type CascadeResult struct {
	*AbstractGameResult
	
	// 消除特定数据
	CascadeCount    int                    `json:"cascade_count"`     // 连锁次数
	TotalRemoved    int                    `json:"total_removed"`     // 总消除数
	CascadeDetails  []CascadeStep          `json:"cascade_details"`   // 每步详情
	FinalMultiplier float64                `json:"final_multiplier"`  // 最终倍数
}

// CascadeStep 单次消除步骤
type CascadeStep struct {
	StepNumber       int          `json:"step_number"`        // 步骤编号
	RemovedGroups    []MatchGroup `json:"removed_groups"`     // 消除的组
	StepWin          int64        `json:"step_win"`           // 本步赢取
	Multiplier       float64      `json:"multiplier"`         // 本步倍数
	GridBefore       [][]int      `json:"grid_before"`        // 消除前网格
	GridAfterRemove  [][]int      `json:"grid_after_remove"`  // 消除后网格（有空位-1）
	GridAfter        [][]int      `json:"grid_after"`         // 重力填充后网格
}

// MatchGroup 匹配组
type MatchGroup struct {
	SymbolID   int            `json:"symbol_id"`   // 符号ID  
	Positions  []GamePosition `json:"positions"`   // 匹配位置
	Count      int            `json:"count"`       // 匹配数量
	Payout     int64          `json:"payout"`      // 赔付金额
}

// CascadeStatistics 消除统计
type CascadeStatistics struct {
	*AbstractStatistics
	
	AverageCascades    float64 `json:"average_cascades"`    // 平均连锁数
	MaxCascadesHit     int     `json:"max_cascades_hit"`    // 最高连锁数
	CascadeDistribution map[int]int `json:"cascade_distribution"` // 连锁分布
}

// NewCascadeEngine 创建消除式引擎
func NewCascadeEngine(algorithmConfig *AlgorithmConfig, cascadeConfig *CascadeConfig) *CascadeEngine {
	// 调整算法配置以适应消除式玩法
	algorithmConfig.ReelCount = cascadeConfig.GridWidth
	algorithmConfig.RowCount = cascadeConfig.GridHeight
	
	composite := NewCompositeSlotEngine(algorithmConfig, "mahjong")
	
	return &CascadeEngine{
		CompositeSlotEngine: composite,
		cascadeConfig:       cascadeConfig,
		cascadeStats:        &CascadeStatistics{
			AbstractStatistics:  &AbstractStatistics{},
			CascadeDistribution: make(map[int]int),
		},
	}
}

// SpinCascade 执行消除式旋转
func (e *CascadeEngine) SpinCascade(ctx context.Context, request *SpinRequest) (*CascadeResult, error) {
	// 1. 生成初始网格
	initialGrid := e.generateInitialGrid()
	
	// 2. 执行连锁消除
	cascadeSteps := []CascadeStep{}
	currentGrid := initialGrid
	totalWin := int64(0)
	stepNumber := 1
	
	for stepNumber <= e.cascadeConfig.MaxCascades {
		// 查找匹配组
		matches := e.findMatches(currentGrid)
		if len(matches) == 0 {
			break // 没有更多匹配，结束连锁
		}
		
		// 计算本步赢取
		stepWin := e.calculateStepWin(matches, stepNumber)
		
		// 消除匹配符号
		newGrid := e.removeMatches(currentGrid, matches)
		
		// 重力下落
		newGrid = e.applyGravity(newGrid)
		
		// 记录步骤
		step := CascadeStep{
			StepNumber:    stepNumber,
			RemovedGroups: matches,
			StepWin:       stepWin,
			Multiplier:    e.getCascadeMultiplier(stepNumber),
			GridAfter:     newGrid,
		}
		cascadeSteps = append(cascadeSteps, step)
		
		totalWin += stepWin
		currentGrid = newGrid
		stepNumber++
	}
	
	// 3. 构建结果
	result := &CascadeResult{
		AbstractGameResult: &AbstractGameResult{
			ResultID:    fmt.Sprintf("cascade_%s", request.GameRequest.SessionID),
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
	}
	
	// 4. 更新统计
	e.updateCascadeStats(result)
	
	return result, nil
}

// generateInitialGrid 生成初始网格
func (e *CascadeEngine) generateInitialGrid() [][]int {
	grid := make([][]int, e.cascadeConfig.GridHeight)
	for i := range grid {
		grid[i] = make([]int, e.cascadeConfig.GridWidth)
		for j := range grid[i] {
			// 使用抽象引擎的随机生成器
			grid[i][j] = e.abstractEngine.(*AbstractSlotEngine).randomGen.NextInt(0, e.abstractEngine.GetAlgorithmConfig().SymbolCount)
		}
	}
	return grid
}

// findMatches 查找1024线匹配（相邻连线）
func (e *CascadeEngine) findMatches(grid [][]int) []MatchGroup {
	matches := []MatchGroup{}
	visited := make([][]bool, len(grid))
	for i := range visited {
		visited[i] = make([]bool, len(grid[i]))
	}
	
	// 遍历每个位置
	for row := 0; row < len(grid); row++ {
		for col := 0; col < len(grid[row]); col++ {
			if !visited[row][col] {
				// 使用DFS查找连通的相同符号
				positions := e.dfsMatch(grid, visited, row, col, grid[row][col])
				if len(positions) >= e.cascadeConfig.MinMatch {
					match := MatchGroup{
						SymbolID:  grid[row][col],
						Positions: positions,
						Count:     len(positions),
						Payout:    e.calculateMatchPayout(grid[row][col], len(positions)),
					}
					matches = append(matches, match)
				}
			}
		}
	}
	
	return matches
}

// dfsMatch 深度优先搜索匹配
func (e *CascadeEngine) dfsMatch(grid [][]int, visited [][]bool, row, col, symbolID int) []GamePosition {
	if row < 0 || row >= len(grid) || col < 0 || col >= len(grid[0]) ||
		visited[row][col] || grid[row][col] != symbolID {
		return []GamePosition{}
	}
	
	visited[row][col] = true
	positions := []GamePosition{{Reel: col, Row: row}}
	
	// 相邻方向（上下左右）
	directions := [][]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	if !e.cascadeConfig.AdjacentOnly {
		// 包含对角线
		directions = append(directions, [][]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}...)
	}
	
	for _, dir := range directions {
		newRow, newCol := row+dir[0], col+dir[1]
		adjacent := e.dfsMatch(grid, visited, newRow, newCol, symbolID)
		positions = append(positions, adjacent...)
	}
	
	return positions
}

// calculateMatchPayout 计算匹配赔付
func (e *CascadeEngine) calculateMatchPayout(symbolID, count int) int64 {
	// 基础赔付 * 连线数量的指数
	basePayout := int64(10) // 可以从配置中获取
	if symbolID >= 7 { // 高价值符号
		basePayout = 50
	}
	
	// 连线数量越多，赔付指数增长
	multiplier := 1.0
	if count >= 5 {
		multiplier = 2.0
	}
	if count >= 8 {
		multiplier = 5.0
	}
	if count >= 12 {
		multiplier = 10.0
	}
	
	return int64(float64(basePayout) * multiplier * float64(count))
}

// removeMatches 消除匹配符号
func (e *CascadeEngine) removeMatches(grid [][]int, matches []MatchGroup) [][]int {
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
	
	for _, match := range matches {
		for _, pos := range match.Positions {
			toRemove[pos.Row][pos.Reel] = true
		}
	}
	
	// 消除符号（设置为-1表示空位）
	for row := 0; row < len(newGrid); row++ {
		for col := 0; col < len(newGrid[row]); col++ {
			if toRemove[row][col] {
				newGrid[row][col] = -1
			}
		}
	}
	
	return newGrid
}

// applyGravity 应用重力，符号下落
func (e *CascadeEngine) applyGravity(grid [][]int) [][]int {
	newGrid := make([][]int, len(grid))
	for i := range newGrid {
		newGrid[i] = make([]int, len(grid[i]))
	}
	
	// 对每列应用重力
	for col := 0; col < len(grid[0]); col++ {
		// 收集非空符号
		symbols := []int{}
		for row := len(grid) - 1; row >= 0; row-- {
			if grid[row][col] != -1 {
				symbols = append(symbols, grid[row][col])
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
	}
	
	return newGrid
}

// calculateStepWin 计算单步赢取
func (e *CascadeEngine) calculateStepWin(matches []MatchGroup, stepNumber int) int64 {
	totalWin := int64(0)
	for _, match := range matches {
		totalWin += match.Payout
	}
	
	// 应用连锁倍数
	multiplier := e.getCascadeMultiplier(stepNumber)
	return int64(float64(totalWin) * multiplier)
}

// getCascadeMultiplier 获取连锁倍数
func (e *CascadeEngine) getCascadeMultiplier(cascadeStep int) float64 {
	if cascadeStep <= 0 || cascadeStep > len(e.cascadeConfig.CascadeMultipliers) {
		return 1.0
	}
	return e.cascadeConfig.CascadeMultipliers[cascadeStep-1]
}

// countTotalRemoved 统计总消除数
func (e *CascadeEngine) countTotalRemoved(steps []CascadeStep) int {
	total := 0
	for _, step := range steps {
		for _, group := range step.RemovedGroups {
			total += group.Count
		}
	}
	return total
}

// updateCascadeStats 更新消除统计
func (e *CascadeEngine) updateCascadeStats(result *CascadeResult) {
	e.cascadeStats.CascadeDistribution[result.CascadeCount]++
	
	if result.CascadeCount > e.cascadeStats.MaxCascadesHit {
		e.cascadeStats.MaxCascadesHit = result.CascadeCount
	}
	
	// 更新平均连锁数（简化计算）
	totalSpins := 0
	totalCascades := 0
	for cascadeCount, frequency := range e.cascadeStats.CascadeDistribution {
		totalSpins += frequency
		totalCascades += cascadeCount * frequency
	}
	
	if totalSpins > 0 {
		e.cascadeStats.AverageCascades = float64(totalCascades) / float64(totalSpins)
	}
}

// GetDefaultCascadeConfig 获取默认消除式配置
func GetDefaultCascadeConfig() *CascadeConfig {
	return &CascadeConfig{
		GridWidth:    5,
		GridHeight:   4, 
		MinMatch:     3,
		MaxCascades:  10,
		CascadeMultipliers: []float64{1.0, 2.0, 3.0, 5.0, 8.0, 12.0, 18.0, 25.0, 35.0, 50.0},
		AdjacentOnly: true, // 仅相邻，不包括对角线
	}
}