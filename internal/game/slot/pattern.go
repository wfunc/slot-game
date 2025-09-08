package slot

import (
	"math"
)

// AdvancedPatternMatcher 高级图案匹配器
type AdvancedPatternMatcher struct {
	config          *SlotConfig
	paylinePatterns [][]Position // 支付线模式
}

// NewAdvancedPatternMatcher 创建高级图案匹配器
func NewAdvancedPatternMatcher(config *SlotConfig) *AdvancedPatternMatcher {
	matcher := &AdvancedPatternMatcher{
		config: config,
	}
	matcher.initPaylinePatterns()
	return matcher
}

// initPaylinePatterns 初始化支付线模式
func (m *AdvancedPatternMatcher) initPaylinePatterns() {
	rows := m.config.Rows
	reels := m.config.Reels
	
	m.paylinePatterns = [][]Position{
		// 水平线
		m.createHorizontalLine(0, reels),            // 上线
		m.createHorizontalLine(rows/2, reels),       // 中线
		m.createHorizontalLine(rows-1, reels),       // 下线
		
		// 对角线
		m.createDiagonalLine(reels, rows, true),     // 左上到右下
		m.createDiagonalLine(reels, rows, false),    // 左下到右上
		
		// V型
		m.createVLine(reels, rows, true),            // V型
		m.createVLine(reels, rows, false),           // 倒V型
		
		// 之字形
		m.createZigzagLine(reels, rows, true),       // 之字形1
		m.createZigzagLine(reels, rows, false),      // 之字形2
		
		// W型
		m.createWLine(reels, rows),                  // W型
		
		// M型
		m.createMLine(reels, rows),                  // M型
	}
	
	// 限制到配置的支付线数
	if len(m.paylinePatterns) > m.config.PayLines {
		m.paylinePatterns = m.paylinePatterns[:m.config.PayLines]
	}
}

// FindWinningLines 查找中奖线
func (m *AdvancedPatternMatcher) FindWinningLines(reels [][]Symbol, config *SlotConfig) []WinLine {
	var winLines []WinLine
	
	// 检查每条支付线
	for lineID, pattern := range m.paylinePatterns {
		// 获取支付线上的符号
		symbols := m.getSymbolsOnLine(reels, pattern)
		
		// 检查连续符号
		if winInfo := m.checkConsecutiveSymbols(symbols, lineID, pattern); winInfo != nil {
			winLines = append(winLines, *winInfo)
		}
		
		// 检查分散符号（Scatter不需要在支付线上）
		if scatterWin := m.checkScatterSymbols(reels); scatterWin != nil {
			winLines = append(winLines, *scatterWin)
		}
	}
	
	// 去重（避免重复计算Scatter）
	winLines = m.deduplicateWinLines(winLines)
	
	return winLines
}

// CalculatePayout 计算赔付
func (m *AdvancedPatternMatcher) CalculatePayout(winLines []WinLine, betAmount int64) int64 {
	var totalPayout int64
	
	for _, winLine := range winLines {
		// 从赔率表查找倍率
		multiplier := m.getMultiplier(winLine.Symbol, winLine.Count)
		
		// 计算单线赔付
		linePayout := int64(float64(betAmount) * multiplier)
		
		// 应用额外倍率（如Wild加成）
		if winLine.Multiplier > 1.0 {
			linePayout = int64(float64(linePayout) * winLine.Multiplier)
		}
		
		winLine.WinAmount = linePayout
		totalPayout += linePayout
	}
	
	return totalPayout
}

// DetectFeatures 检测特殊功能
func (m *AdvancedPatternMatcher) DetectFeatures(reels [][]Symbol, config *SlotConfig) []Feature {
	var features []Feature
	
	// 检测免费旋转
	if freeSpins := m.detectFreeSpins(reels); freeSpins != nil {
		features = append(features, *freeSpins)
	}
	
	// 检测奖励游戏
	if bonus := m.detectBonusGame(reels); bonus != nil {
		features = append(features, *bonus)
	}
	
	// 检测扩展Wild
	if expandWild := m.detectExpandingWild(reels); expandWild != nil {
		features = append(features, *expandWild)
	}
	
	// 检测级联消除
	if cascade := m.detectCascade(reels); cascade != nil {
		features = append(features, *cascade)
	}
	
	return features
}

// getSymbolsOnLine 获取支付线上的符号
func (m *AdvancedPatternMatcher) getSymbolsOnLine(reels [][]Symbol, pattern []Position) []Symbol {
	symbols := make([]Symbol, len(pattern))
	for i, pos := range pattern {
		if pos.Reel < len(reels) && pos.Row < len(reels[pos.Reel]) {
			symbols[i] = reels[pos.Reel][pos.Row]
		}
	}
	return symbols
}

// checkConsecutiveSymbols 检查连续符号
func (m *AdvancedPatternMatcher) checkConsecutiveSymbols(symbols []Symbol, lineID int, pattern []Position) *WinLine {
	if len(symbols) == 0 {
		return nil
	}
	
	// 获取第一个非Wild符号
	firstSymbol := symbols[0]
	if m.isWild(firstSymbol) && len(symbols) > 1 {
		for _, s := range symbols[1:] {
			if !m.isWild(s) {
				firstSymbol = s
				break
			}
		}
	}
	
	// 计算连续符号数
	count := 1
	positions := []Position{pattern[0]}
	wildMultiplier := 1.0
	
	for i := 1; i < len(symbols); i++ {
		currentSymbol := symbols[i]
		
		// Wild可以替代任何符号（除了Scatter和Bonus）
		if m.isWild(currentSymbol) {
			if !m.isScatter(firstSymbol) && !m.isBonus(firstSymbol) {
				count++
				positions = append(positions, pattern[i])
				wildMultiplier *= 1.5 // Wild加倍（降低倍率）
				continue
			}
		}
		
		// 检查是否相同符号
		if currentSymbol == firstSymbol {
			count++
			positions = append(positions, pattern[i])
		} else {
			break // 连续符号中断
		}
	}
	
	// 检查是否达到最小连续数（通常是3个）
	if count >= 3 {
		return &WinLine{
			LineID:     lineID,
			LineType:   m.getLineType(pattern),
			Symbol:     firstSymbol,
			Count:      count,
			Positions:  positions,
			Multiplier: wildMultiplier,
		}
	}
	
	return nil
}

// checkScatterSymbols 检查分散符号
func (m *AdvancedPatternMatcher) checkScatterSymbols(reels [][]Symbol) *WinLine {
	var scatterPositions []Position
	
	// 扫描所有位置寻找Scatter
	for r := 0; r < len(reels); r++ {
		for c := 0; c < len(reels[r]); c++ {
			if m.isScatter(reels[r][c]) {
				scatterPositions = append(scatterPositions, Position{Reel: r, Row: c})
			}
		}
	}
	
	// Scatter通常需要3个或以上
	if len(scatterPositions) >= 3 {
		return &WinLine{
			LineID:     -1, // 特殊ID表示Scatter
			LineType:   -1,
			Symbol:     SymbolScatter,
			Count:      len(scatterPositions),
			Positions:  scatterPositions,
			Multiplier: 1.0,
		}
	}
	
	return nil
}

// detectFreeSpins 检测免费旋转
func (m *AdvancedPatternMatcher) detectFreeSpins(reels [][]Symbol) *Feature {
	scatterCount := 0
	var positions []Position
	
	for r := 0; r < len(reels); r++ {
		for c := 0; c < len(reels[r]); c++ {
			if m.isScatter(reels[r][c]) {
				scatterCount++
				positions = append(positions, Position{Reel: r, Row: c})
			}
		}
	}
	
	// 3个Scatter触发免费旋转
	if scatterCount >= 3 {
		freeSpins := scatterCount * 2 // 每个Scatter给2次免费旋转（降低）
		return &Feature{
			Type:        FeatureTypeFreeSpins,
			TriggerPos:  positions,
			Value:       freeSpins,
			Description: "触发免费旋转",
		}
	}
	
	return nil
}

// detectBonusGame 检测奖励游戏
func (m *AdvancedPatternMatcher) detectBonusGame(reels [][]Symbol) *Feature {
	bonusCount := 0
	var positions []Position
	
	for r := 0; r < len(reels); r++ {
		for c := 0; c < len(reels[r]); c++ {
			if m.isBonus(reels[r][c]) {
				bonusCount++
				positions = append(positions, Position{Reel: r, Row: c})
			}
		}
	}
	
	// 3个Bonus触发奖励游戏
	if bonusCount >= 3 {
		return &Feature{
			Type:        FeatureTypeBonus,
			TriggerPos:  positions,
			Value:       "bonus_game",
			Description: "触发奖励游戏",
		}
	}
	
	return nil
}

// detectExpandingWild 检测扩展Wild
func (m *AdvancedPatternMatcher) detectExpandingWild(reels [][]Symbol) *Feature {
	var wildPositions []Position
	
	// 检查每个卷轴是否有Wild
	for r := 0; r < len(reels); r++ {
		for c := 0; c < len(reels[r]); c++ {
			if m.isWild(reels[r][c]) {
				wildPositions = append(wildPositions, Position{Reel: r, Row: c})
			}
		}
	}
	
	// 如果中间卷轴有Wild，触发扩展
	for _, pos := range wildPositions {
		if pos.Reel == len(reels)/2 {
			return &Feature{
				Type:        FeatureTypeExpandWild,
				TriggerPos:  []Position{pos},
				Value:       "expand_reel",
				Description: "Wild扩展整个卷轴",
			}
		}
	}
	
	return nil
}

// detectCascade 检测级联消除
func (m *AdvancedPatternMatcher) detectCascade(reels [][]Symbol) *Feature {
	// 简化实现：如果有任何中奖线，触发级联
	winLines := m.FindWinningLines(reels, m.config)
	if len(winLines) > 0 {
		return &Feature{
			Type:        FeatureTypeCascade,
			TriggerPos:  winLines[0].Positions,
			Value:       len(winLines),
			Description: "触发级联消除",
		}
	}
	
	return nil
}

// Helper functions

func (m *AdvancedPatternMatcher) isWild(symbol Symbol) bool {
	for _, wild := range m.config.WildSymbols {
		if symbol == wild {
			return true
		}
	}
	return false
}

func (m *AdvancedPatternMatcher) isScatter(symbol Symbol) bool {
	for _, scatter := range m.config.ScatterSymbols {
		if symbol == scatter {
			return true
		}
	}
	return false
}

func (m *AdvancedPatternMatcher) isBonus(symbol Symbol) bool {
	for _, bonus := range m.config.BonusSymbols {
		if symbol == bonus {
			return true
		}
	}
	return false
}

func (m *AdvancedPatternMatcher) getMultiplier(symbol Symbol, count int) float64 {
	for _, payTable := range m.config.PayTables {
		if payTable.Symbol == symbol && payTable.Count == count {
			return payTable.Multiplier
		}
	}
	// 默认倍率（降低以控制RTP）
	return float64(count) * 0.2
}

func (m *AdvancedPatternMatcher) getLineType(pattern []Position) LineType {
	// 简化判断线型
	if len(pattern) < 2 {
		return LineTypeHorizontal
	}
	
	// 检查是否水平
	isHorizontal := true
	firstRow := pattern[0].Row
	for _, pos := range pattern[1:] {
		if pos.Row != firstRow {
			isHorizontal = false
			break
		}
	}
	if isHorizontal {
		return LineTypeHorizontal
	}
	
	// 其他类型的判断...
	return LineTypeZigzag
}

func (m *AdvancedPatternMatcher) deduplicateWinLines(lines []WinLine) []WinLine {
	seen := make(map[string]bool)
	var result []WinLine
	
	for _, line := range lines {
		key := string(line.Symbol) + string(rune(line.Count))
		if !seen[key] {
			seen[key] = true
			result = append(result, line)
		}
	}
	
	return result
}

// Line pattern creators

func (m *AdvancedPatternMatcher) createHorizontalLine(row, reels int) []Position {
	line := make([]Position, reels)
	for i := 0; i < reels; i++ {
		line[i] = Position{Reel: i, Row: row}
	}
	return line
}

func (m *AdvancedPatternMatcher) createDiagonalLine(reels, rows int, downward bool) []Position {
	line := make([]Position, 0)
	for i := 0; i < reels && i < rows; i++ {
		row := i
		if !downward {
			row = rows - 1 - i
		}
		line = append(line, Position{Reel: i, Row: row})
	}
	return line
}

func (m *AdvancedPatternMatcher) createVLine(reels, rows int, upward bool) []Position {
	line := make([]Position, reels)
	mid := reels / 2
	
	for i := 0; i < reels; i++ {
		var row int
		if i <= mid {
			if upward {
				row = rows - 1 - i
			} else {
				row = i
			}
		} else {
			if upward {
				row = rows - 1 - (reels - 1 - i)
			} else {
				row = reels - 1 - i
			}
		}
		row = int(math.Max(0, math.Min(float64(rows-1), float64(row))))
		line[i] = Position{Reel: i, Row: row}
	}
	return line
}

func (m *AdvancedPatternMatcher) createZigzagLine(reels, rows int, pattern1 bool) []Position {
	line := make([]Position, reels)
	for i := 0; i < reels; i++ {
		var row int
		if pattern1 {
			if i%2 == 0 {
				row = 0
			} else {
				row = rows - 1
			}
		} else {
			if i%2 == 0 {
				row = rows - 1
			} else {
				row = 0
			}
		}
		line[i] = Position{Reel: i, Row: row}
	}
	return line
}

func (m *AdvancedPatternMatcher) createWLine(reels, rows int) []Position {
	line := make([]Position, reels)
	quarter := reels / 4
	
	for i := 0; i < reels; i++ {
		var row int
		if i < quarter {
			row = 0
		} else if i < quarter*2 {
			row = rows - 1
		} else if i < quarter*3 {
			row = 0
		} else {
			row = rows - 1
		}
		line[i] = Position{Reel: i, Row: row}
	}
	return line
}

func (m *AdvancedPatternMatcher) createMLine(reels, rows int) []Position {
	line := make([]Position, reels)
	quarter := reels / 4
	
	for i := 0; i < reels; i++ {
		var row int
		if i < quarter {
			row = rows - 1
		} else if i < quarter*2 {
			row = 0
		} else if i < quarter*3 {
			row = rows - 1
		} else {
			row = 0
		}
		line[i] = Position{Reel: i, Row: row}
	}
	return line
}