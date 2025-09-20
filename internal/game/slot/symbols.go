package slot

// 符号ID定义
const (
	// 基础符号 (0-7)
	SYMBOL_MAHJONG_1 = 0 // 一筒
	SYMBOL_MAHJONG_2 = 1 // 二筒
	SYMBOL_MAHJONG_3 = 2 // 三筒
	SYMBOL_MAHJONG_4 = 3 // 四筒
	SYMBOL_MAHJONG_5 = 4 // 五筒
	SYMBOL_MAHJONG_6 = 5 // 六筒
	SYMBOL_MAHJONG_7 = 6 // 七筒
	SYMBOL_MAHJONG_8 = 7 // 八筒

	// 特殊符号
	SYMBOL_WILD         = -1 // Wild符号（金色Wild）
	SYMBOL_SCATTER      = 8  // Scatter符号（免费游戏）
	SYMBOL_ANIMAL_WILD  = 9  // 动物Wild符号（触发Animal游戏）
	SYMBOL_ANIMAL_BONUS = 10 // 动物Bonus符号（触发超级Animal游戏）

	// 金色符号范围 (16-23)
	SYMBOL_GOLDEN_BASE = 16
	SYMBOL_GOLDEN_1    = 16 // 金色一筒
	SYMBOL_GOLDEN_2    = 17 // 金色二筒
	SYMBOL_GOLDEN_3    = 18 // 金色三筒
	SYMBOL_GOLDEN_4    = 19 // 金色四筒
	SYMBOL_GOLDEN_5    = 20 // 金色五筒
	SYMBOL_GOLDEN_6    = 21 // 金色六筒
	SYMBOL_GOLDEN_7    = 22 // 金色七筒
	SYMBOL_GOLDEN_8    = 23 // 金色八筒
)

// SymbolInfo 符号信息
type SymbolInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`    // normal, wild, scatter, bonus
	Display     string `json:"display"` // 显示字符
	Description string `json:"description"`
	CanBeGolden bool   `json:"can_be_golden"` // 是否可以变成金色
	IsSpecial   bool   `json:"is_special"`    // 是否为特殊符号
}

// GetSymbolInfo 获取符号信息
func GetSymbolInfo(symbolID int) *SymbolInfo {
	symbolMap := map[int]*SymbolInfo{
		SYMBOL_MAHJONG_1: {
			ID:          SYMBOL_MAHJONG_1,
			Name:        "一筒",
			Type:        "normal",
			Display:     "①",
			Description: "基础符号 - 一筒",
			CanBeGolden: true,
			IsSpecial:   false,
		},
		SYMBOL_MAHJONG_2: {
			ID:          SYMBOL_MAHJONG_2,
			Name:        "二筒",
			Type:        "normal",
			Display:     "②",
			Description: "基础符号 - 二筒",
			CanBeGolden: true,
			IsSpecial:   false,
		},
		SYMBOL_MAHJONG_3: {
			ID:          SYMBOL_MAHJONG_3,
			Name:        "三筒",
			Type:        "normal",
			Display:     "③",
			Description: "基础符号 - 三筒",
			CanBeGolden: true,
			IsSpecial:   false,
		},
		SYMBOL_MAHJONG_4: {
			ID:          SYMBOL_MAHJONG_4,
			Name:        "四筒",
			Type:        "normal",
			Display:     "④",
			Description: "基础符号 - 四筒",
			CanBeGolden: true,
			IsSpecial:   false,
		},
		SYMBOL_MAHJONG_5: {
			ID:          SYMBOL_MAHJONG_5,
			Name:        "五筒",
			Type:        "normal",
			Display:     "⑤",
			Description: "基础符号 - 五筒",
			CanBeGolden: true,
			IsSpecial:   false,
		},
		SYMBOL_MAHJONG_6: {
			ID:          SYMBOL_MAHJONG_6,
			Name:        "六筒",
			Type:        "normal",
			Display:     "⑥",
			Description: "基础符号 - 六筒",
			CanBeGolden: true,
			IsSpecial:   false,
		},
		SYMBOL_MAHJONG_7: {
			ID:          SYMBOL_MAHJONG_7,
			Name:        "七筒",
			Type:        "normal",
			Display:     "⑦",
			Description: "基础符号 - 七筒",
			CanBeGolden: true,
			IsSpecial:   false,
		},
		SYMBOL_MAHJONG_8: {
			ID:          SYMBOL_MAHJONG_8,
			Name:        "八筒",
			Type:        "normal",
			Display:     "⑧",
			Description: "基础符号 - 八筒",
			CanBeGolden: true,
			IsSpecial:   false,
		},
		SYMBOL_WILD: {
			ID:          SYMBOL_WILD,
			Name:        "Wild",
			Type:        "wild",
			Display:     "🀄",
			Description: "Wild符号 - 可替代任意基础符号",
			CanBeGolden: false,
			IsSpecial:   true,
		},
		SYMBOL_SCATTER: {
			ID:          SYMBOL_SCATTER,
			Name:        "Scatter",
			Type:        "scatter",
			Display:     "🎰",
			Description: "Scatter符号 - 触发免费游戏",
			CanBeGolden: false,
			IsSpecial:   true,
		},
		SYMBOL_ANIMAL_WILD: {
			ID:          SYMBOL_ANIMAL_WILD,
			Name:        "动物Wild",
			Type:        "bonus",
			Display:     "🐼",
			Description: "动物Wild符号 - 3个触发Animal游戏",
			CanBeGolden: false,
			IsSpecial:   true,
		},
		SYMBOL_ANIMAL_BONUS: {
			ID:          SYMBOL_ANIMAL_BONUS,
			Name:        "动物Bonus",
			Type:        "bonus",
			Display:     "🎯",
			Description: "动物Bonus符号 - 5个触发超级Animal游戏",
			CanBeGolden: false,
			IsSpecial:   true,
		},
	}

	// 添加金色符号
	for i := 0; i < 8; i++ {
		goldenID := SYMBOL_GOLDEN_BASE + i
		baseInfo := symbolMap[i]
		if baseInfo != nil {
			symbolMap[goldenID] = &SymbolInfo{
				ID:          goldenID,
				Name:        "金色" + baseInfo.Name,
				Type:        "golden",
				Display:     "✨" + baseInfo.Display,
				Description: "金色符号 - " + baseInfo.Description,
				CanBeGolden: false,
				IsSpecial:   true,
			}
		}
	}

	if info, exists := symbolMap[symbolID]; exists {
		return info
	}
	return &SymbolInfo{
		ID:          symbolID,
		Name:        "未知符号",
		Type:        "unknown",
		Display:     "?",
		Description: "未知符号",
		CanBeGolden: false,
		IsSpecial:   false,
	}
}

// IsSpecialSymbol 判断是否为特殊符号
func IsSpecialSymbol(symbolID int) bool {
	return symbolID == SYMBOL_WILD ||
		symbolID == SYMBOL_SCATTER ||
		symbolID == SYMBOL_ANIMAL_WILD ||
		symbolID == SYMBOL_ANIMAL_BONUS ||
		(symbolID >= SYMBOL_GOLDEN_BASE && symbolID <= SYMBOL_GOLDEN_8)
}

// IsAnimalTriggerSymbol 判断是否为Animal触发符号
func IsAnimalTriggerSymbol(symbolID int) bool {
	return symbolID == SYMBOL_ANIMAL_WILD || symbolID == SYMBOL_ANIMAL_BONUS
}

// CanBeGolden 判断符号是否可以变成金色
func CanBeGolden(symbolID int) bool {
	return symbolID >= SYMBOL_MAHJONG_1 && symbolID <= SYMBOL_MAHJONG_8
}

// ConvertToGolden 将普通符号转换为金色符号
func ConvertToGolden(symbolID int) int {
	if CanBeGolden(symbolID) {
		return SYMBOL_GOLDEN_BASE + symbolID
	}
	return symbolID
}

// IsGoldenSymbol 判断是否为金色符号
func IsGoldenSymbol(symbolID int) bool {
	return symbolID >= SYMBOL_GOLDEN_1 && symbolID <= SYMBOL_GOLDEN_8
}

// GetBaseSymbol 获取金色符号对应的基础符号
func GetBaseSymbol(goldenSymbolID int) int {
	if IsGoldenSymbol(goldenSymbolID) {
		return goldenSymbolID - SYMBOL_GOLDEN_BASE
	}
	return goldenSymbolID
}
