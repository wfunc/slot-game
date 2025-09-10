package slot

import (
	"fmt"
)

// Line1024Matcher 1024线匹配器（All-Ways-Win）
type Line1024Matcher struct {
	config *CascadeConfig
}

// NewLine1024Matcher 创建1024线匹配器
func NewLine1024Matcher(config *CascadeConfig) *Line1024Matcher {
	return &Line1024Matcher{
		config: config,
	}
}

// LineMatch 1024线匹配结果
type LineMatch struct {
	SymbolID  int            `json:"symbol_id"`  // 符号ID
	Positions []GamePosition `json:"positions"`  // 匹配位置
	Length    int            `json:"length"`     // 连线长度
	Count     int            `json:"count"`      // 符号数量
	Payout    int64          `json:"payout"`     // 赔付金额
}

// Find1024LineMatches 查找所有1024线匹配
func (m *Line1024Matcher) Find1024LineMatches(grid [][]int, payTable map[int][]int64) []LineMatch {
	matches := []LineMatch{}
	
	// 检查每种符号的最长连线
	symbolMatches := make(map[int]LineMatch)
	
	// 从第1列开始，检查每个符号
	for row := 0; row < len(grid); row++ {
		symbolID := grid[row][0]
		if symbolID < 0 { // 跳过空位
			continue
		}
		
		// 查找这个符号的最长连线（必须从第1列开始连续）
		longestMatch := m.findConsecutiveLine(grid, symbolID, payTable)
		
		// 如果找到了更长的连线，或者这是第一次找到这个符号
		if longestMatch.Length >= 3 { // 至少3连
			existing, exists := symbolMatches[symbolID]
			if !exists || longestMatch.Length > existing.Length || 
				(longestMatch.Length == existing.Length && longestMatch.Count > existing.Count) {
				symbolMatches[symbolID] = longestMatch
			}
		}
	}
	
	// 转换为切片
	for _, match := range symbolMatches {
		matches = append(matches, match)
	}
	
	return matches
}

// findConsecutiveLine 查找连续匹配的1024线
func (m *Line1024Matcher) findConsecutiveLine(grid [][]int, targetSymbol int, payTable map[int][]int64) LineMatch {
	positions := []GamePosition{}
	
	// 检查第1列是否有目标符号
	firstColHasSymbol := false
	for row := 0; row < len(grid); row++ {
		if grid[row][0] == targetSymbol {
			firstColHasSymbol = true
			break
		}
	}
	
	if !firstColHasSymbol {
		return LineMatch{} // 第1列没有目标符号，无法形成连线
	}
	
	// 从第1列开始，检查每列是否有连续的目标符号
	for col := 0; col < len(grid[0]); col++ {
		colHasSymbol := false
		colPositions := []GamePosition{}
		
		// 检查当前列的所有行
		for row := 0; row < len(grid); row++ {
			if grid[row][col] == targetSymbol {
				colHasSymbol = true
				colPositions = append(colPositions, GamePosition{Row: row, Reel: col})
			}
		}
		
		if !colHasSymbol {
			// 当前列没有目标符号，连线中断
			break
		}
		
		// 添加当前列的所有匹配位置
		positions = append(positions, colPositions...)
	}
	
	// 如果连线长度不足3列，返回空
	lineLength := 0
	if len(positions) > 0 {
		maxCol := 0
		for _, pos := range positions {
			if pos.Reel > maxCol {
				maxCol = pos.Reel
			}
		}
		lineLength = maxCol + 1 // 列数从0开始，所以+1
	}
	
	if lineLength < 3 {
		return LineMatch{} // 连线长度不足
	}
	
	// 计算赔付
	payout := m.calculateLinePayout(targetSymbol, lineLength, len(positions), payTable)
	
	return LineMatch{
		SymbolID:  targetSymbol,
		Positions: positions,
		Length:    lineLength,
		Count:     len(positions),
		Payout:    payout,
	}
}

// findLongestLine 查找从指定位置开始的最长连线（旧版本）
func (m *Line1024Matcher) findLongestLine(grid [][]int, targetSymbol int, startRow, startCol int) LineMatch {
	if startCol >= len(grid[0]) {
		return LineMatch{}
	}
	
	positions := []GamePosition{}
	currentCol := startCol
	totalCount := 0
	
	// 从起始列开始，逐列检查
	for currentCol < len(grid[0]) {
		// 在当前列查找匹配的符号
		colMatches := []GamePosition{}
		for row := 0; row < len(grid); row++ {
			if grid[row][currentCol] == targetSymbol {
				colMatches = append(colMatches, GamePosition{Row: row, Reel: currentCol})
			}
		}
		
		if len(colMatches) == 0 {
			// 当前列没有匹配符号，连线中断
			break
		}
		
		// 添加当前列的所有匹配位置
		positions = append(positions, colMatches...)
		totalCount += len(colMatches)
		currentCol++
	}
	
	return LineMatch{
		SymbolID:  targetSymbol,
		Positions: positions,
		Length:    currentCol - startCol, // 连线长度（列数）
		Count:     totalCount,           // 符号总数
	}
}

// calculateLinePayout 计算连线赔付
func (m *Line1024Matcher) calculateLinePayout(symbolID int, lineLength, symbolCount int, payTable map[int][]int64) int64 {
	if payouts, exists := payTable[symbolID]; exists {
		if lineLength > 0 && lineLength <= len(payouts) {
			basePayout := payouts[lineLength-1]
			
			// 根据符号数量调整赔付
			// 1024线的特点：更多符号 = 更高倍数
			multiplier := float64(symbolCount) / float64(lineLength)
			return int64(float64(basePayout) * multiplier)
		}
	}
	return 0
}

// Find1024LineMatchesWithWild 支持Wild的1024线匹配
func (m *Line1024Matcher) Find1024LineMatchesWithWild(grid [][]int, payTable map[int][]int64, wildSymbolID int) []LineMatch {
	matches := []LineMatch{}
	symbolMatches := make(map[int]LineMatch)
	
	// 遍历每种可能的符号（包括通过Wild形成的）
	for targetSymbol := 0; targetSymbol < len(payTable); targetSymbol++ {
		if targetSymbol == wildSymbolID {
			continue // 跳过Wild本身
		}
		
		// 检查这个符号的连续匹配（包括Wild替换）
		longestMatch := m.findConsecutiveLineWithWild(grid, targetSymbol, wildSymbolID, payTable)
		
		if longestMatch.Length >= 3 {
			existing, exists := symbolMatches[targetSymbol]
			if !exists || longestMatch.Length > existing.Length || 
				(longestMatch.Length == existing.Length && longestMatch.Count > existing.Count) {
				symbolMatches[targetSymbol] = longestMatch
			}
		}
	}
	
	// 转换为切片
	for _, match := range symbolMatches {
		matches = append(matches, match)
	}
	
	return matches
}

// findConsecutiveLineWithWild 查找支持Wild的连续匹配1024线
func (m *Line1024Matcher) findConsecutiveLineWithWild(grid [][]int, targetSymbol int, wildSymbolID int, payTable map[int][]int64) LineMatch {
	positions := []GamePosition{}
	
	// 检查第1列是否有目标符号或Wild
	firstColHasSymbol := false
	for row := 0; row < len(grid); row++ {
		symbol := grid[row][0]
		if symbol == targetSymbol || symbol == wildSymbolID {
			firstColHasSymbol = true
			break
		}
	}
	
	if !firstColHasSymbol {
		return LineMatch{} // 第1列没有目标符号或Wild，无法形成连线
	}
	
	// 从第1列开始，检查每列是否有连续的目标符号或Wild
	for col := 0; col < len(grid[0]); col++ {
		colHasSymbol := false
		colPositions := []GamePosition{}
		
		// 检查当前列的所有行
		for row := 0; row < len(grid); row++ {
			symbol := grid[row][col]
			if symbol == targetSymbol || symbol == wildSymbolID {
				colHasSymbol = true
				colPositions = append(colPositions, GamePosition{Row: row, Reel: col})
			}
		}
		
		if !colHasSymbol {
			// 当前列没有目标符号或Wild，连线中断
			break
		}
		
		// 添加当前列的所有匹配位置
		positions = append(positions, colPositions...)
	}
	
	// 如果连线长度不足3列，返回空
	lineLength := 0
	if len(positions) > 0 {
		maxCol := 0
		for _, pos := range positions {
			if pos.Reel > maxCol {
				maxCol = pos.Reel
			}
		}
		lineLength = maxCol + 1 // 列数从0开始，所以+1
	}
	
	if lineLength < 3 {
		return LineMatch{} // 连线长度不足
	}
	
	// 计算赔付
	payout := m.calculateLinePayout(targetSymbol, lineLength, len(positions), payTable)
	
	return LineMatch{
		SymbolID:  targetSymbol,
		Positions: positions,
		Length:    lineLength,
		Count:     len(positions),
		Payout:    payout,
	}
}

// findLongestLineWithWild 支持Wild的最长连线查找（旧版本）
func (m *Line1024Matcher) findLongestLineWithWild(grid [][]int, targetSymbol int, startRow, startCol int, wildSymbolID int) LineMatch {
	if startCol >= len(grid[0]) {
		return LineMatch{}
	}
	
	positions := []GamePosition{}
	currentCol := startCol
	totalCount := 0
	
	// 从起始列开始，逐列检查
	for currentCol < len(grid[0]) {
		// 在当前列查找匹配的符号（包括Wild）
		colMatches := []GamePosition{}
		for row := 0; row < len(grid); row++ {
			symbol := grid[row][currentCol]
			if symbol == targetSymbol || symbol == wildSymbolID {
				colMatches = append(colMatches, GamePosition{Row: row, Reel: currentCol})
			}
		}
		
		if len(colMatches) == 0 {
			// 当前列没有匹配符号，连线中断
			break
		}
		
		// 添加当前列的所有匹配位置
		positions = append(positions, colMatches...)
		totalCount += len(colMatches)
		currentCol++
	}
	
	return LineMatch{
		SymbolID:  targetSymbol,
		Positions: positions,
		Length:    currentCol - startCol, // 连线长度（列数）
		Count:     totalCount,           // 符号总数
	}
}

// ConvertToMatchGroups 将LineMatch转换为MatchGroup格式
func (m *Line1024Matcher) ConvertToMatchGroups(lineMatches []LineMatch) []MatchGroup {
	matchGroups := make([]MatchGroup, len(lineMatches))
	
	for i, lineMatch := range lineMatches {
		matchGroups[i] = MatchGroup{
			SymbolID:  lineMatch.SymbolID,
			Positions: lineMatch.Positions,
			Count:     lineMatch.Count,
			Payout:    lineMatch.Payout,
		}
	}
	
	return matchGroups
}

// Debug1024Lines 调试1024线匹配（显示详细信息）
func (m *Line1024Matcher) Debug1024Lines(grid [][]int, matches []LineMatch) {
	fmt.Println("=== 1024线匹配调试信息 ===")
	fmt.Printf("网格: %d行 x %d列\n", len(grid), len(grid[0]))
	
	// 显示网格
	for row := 0; row < len(grid); row++ {
		fmt.Printf("行%d: ", row)
		for col := 0; col < len(grid[0]); col++ {
			if grid[row][col] == -1 {
				fmt.Printf("[空] ")
			} else {
				fmt.Printf("[%d] ", grid[row][col])
			}
		}
		fmt.Println()
	}
	
	// 显示匹配结果
	fmt.Printf("找到 %d 个连线匹配:\n", len(matches))
	for i, match := range matches {
		fmt.Printf("匹配%d: 符号%d, 长度%d列, %d个符号, 赔付%d\n", 
			i+1, match.SymbolID, match.Length, match.Count, match.Payout)
		
		fmt.Print("  位置: ")
		for j, pos := range match.Positions {
			fmt.Printf("(%d,%d)", pos.Row+1, pos.Reel+1)
			if j < len(match.Positions)-1 {
				fmt.Print(", ")
			}
		}
		fmt.Println()
	}
}