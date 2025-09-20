package slot

import (
	"encoding/json"
	"fmt"
)

// ThemeRenderer 主题渲染器 - 将抽象结果转换为具体图案
type ThemeRenderer interface {
	// 渲染抽象结果为主题化结果
	RenderResult(abstract *AbstractGameResult, themeID string) (*ThemedGameResult, error)

	// 主题管理
	GetTheme(themeID string) (*Theme, error)
	ListThemes() []string
	RegisterTheme(theme *Theme) error
}

// ThemedGameResult 主题化游戏结果
type ThemedGameResult struct {
	// 继承抽象结果的所有数值属性
	*AbstractGameResult

	// 主题化的视觉内容
	ThemeID       string          `json:"theme_id"`
	ThemeName     string          `json:"theme_name"`
	ReelSymbols   [][]ThemeSymbol `json:"reel_symbols"`   // 实际显示的符号
	WinAnimations []Animation     `json:"win_animations"` // 获胜动画
	SoundEffects  []SoundEffect   `json:"sound_effects"`  // 音效
	VisualEffects []VisualEffect  `json:"visual_effects"` // 视觉特效
}

// ThemeSymbol 主题符号（重命名避免冲突）
type ThemeSymbol struct {
	ID         int                    `json:"id"`         // 抽象符号ID
	Name       string                 `json:"name"`       // 符号名称
	ImageURL   string                 `json:"image_url"`  // 图片URL
	Animation  string                 `json:"animation"`  // 动画名称
	Rarity     SymbolRarity           `json:"rarity"`     // 稀有度
	Properties map[string]interface{} `json:"properties"` // 符号属性
}

// Animation 动画配置
type Animation struct {
	Type       AnimationType          `json:"type"`
	Target     []Position             `json:"target"`   // 动画目标位置
	Duration   int                    `json:"duration"` // 持续时间(ms)
	Sequence   []AnimationFrame       `json:"sequence"` // 动画帧序列
	Properties map[string]interface{} `json:"properties"`
}

// SoundEffect 音效配置
type SoundEffect struct {
	Type    SoundType `json:"type"`
	FileURL string    `json:"file_url"`
	Volume  float64   `json:"volume"`
	Loop    bool      `json:"loop"`
	Delay   int       `json:"delay"`
}

// VisualEffect 视觉特效
type VisualEffect struct {
	Type       EffectType             `json:"type"`
	Target     []Position             `json:"target"`
	Duration   int                    `json:"duration"`
	Intensity  float64                `json:"intensity"`
	Properties map[string]interface{} `json:"properties"`
}

// 枚举定义
type SymbolRarity int

const (
	RarityCommon SymbolRarity = iota
	RarityRare
	RarityEpic
	RarityLegendary
)

type AnimationType int

const (
	AnimationTypeWinLine AnimationType = iota
	AnimationTypeSymbolWin
	AnimationTypeFeatureTrigger
	AnimationTypeBigWin
	AnimationTypeJackpot
)

type SoundType int

const (
	SoundTypeSpin SoundType = iota
	SoundTypeWin
	SoundTypeBigWin
	SoundTypeJackpot
	SoundTypeFeature
	SoundTypeAmbient
)

type EffectType int

const (
	EffectTypeParticle EffectType = iota
	EffectTypeFlash
	EffectTypeShake
	EffectTypeGlow
	EffectTypeExplosion
)

type AnimationFrame struct {
	ImageURL   string                 `json:"image_url"`
	Duration   int                    `json:"duration"`
	Transform  Transform              `json:"transform"`
	Properties map[string]interface{} `json:"properties"`
}

type Transform struct {
	Scale    float64 `json:"scale"`
	Rotation float64 `json:"rotation"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Alpha    float64 `json:"alpha"`
}

// Theme 游戏主题配置
type Theme struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	// 符号映射：抽象ID -> 主题符号
	SymbolMap map[int]ThemeSymbol `json:"symbol_map"`

	// 背景配置
	Background BackgroundConfig `json:"background"`

	// 音效映射
	Sounds map[SoundType]SoundEffect `json:"sounds"`

	// 动画配置
	Animations map[AnimationType][]Animation `json:"animations"`

	// 特效配置
	Effects map[EffectType]VisualEffect `json:"effects"`

	// UI配置
	UI UIConfig `json:"ui"`

	// 扩展属性
	Properties map[string]interface{} `json:"properties"`
}

type BackgroundConfig struct {
	ImageURL    string  `json:"image_url"`
	Color       string  `json:"color"`
	Animation   string  `json:"animation"`
	MusicURL    string  `json:"music_url"`
	MusicVolume float64 `json:"music_volume"`
}

type UIConfig struct {
	ReelFrame string                  `json:"reel_frame"`
	Buttons   map[string]ButtonConfig `json:"buttons"`
	Colors    map[string]string       `json:"colors"`
	Fonts     map[string]FontConfig   `json:"fonts"`
}

type ButtonConfig struct {
	ImageURL      string `json:"image_url"`
	HoverImageURL string `json:"hover_image_url"`
	Sound         string `json:"sound"`
}

type FontConfig struct {
	Family string `json:"family"`
	Size   int    `json:"size"`
	Color  string `json:"color"`
	Style  string `json:"style"`
}

// DefaultThemeRenderer 默认主题渲染器实现
type DefaultThemeRenderer struct {
	themes map[string]*Theme
}

// NewDefaultThemeRenderer 创建默认主题渲染器
func NewDefaultThemeRenderer() *DefaultThemeRenderer {
	renderer := &DefaultThemeRenderer{
		themes: make(map[string]*Theme),
	}

	// 注册内置主题
	renderer.registerBuiltinThemes()
	return renderer
}

// RenderResult 渲染抽象结果
func (r *DefaultThemeRenderer) RenderResult(abstract *AbstractGameResult, themeID string) (*ThemedGameResult, error) {
	theme, err := r.GetTheme(themeID)
	if err != nil {
		return nil, fmt.Errorf("theme not found: %s", themeID)
	}

	// 1. 转换转轴符号
	reelSymbols := r.convertReelSymbols(abstract.ReelResults, theme)

	// 2. 生成获胜动画
	winAnimations := r.generateWinAnimations(abstract, theme)

	// 3. 生成音效
	soundEffects := r.generateSoundEffects(abstract, theme)

	// 4. 生成视觉特效
	visualEffects := r.generateVisualEffects(abstract, theme)

	return &ThemedGameResult{
		AbstractGameResult: abstract,
		ThemeID:            theme.ID,
		ThemeName:          theme.Name,
		ReelSymbols:        reelSymbols,
		WinAnimations:      winAnimations,
		SoundEffects:       soundEffects,
		VisualEffects:      visualEffects,
	}, nil
}

// convertReelSymbols 转换转轴符号
func (r *DefaultThemeRenderer) convertReelSymbols(reelResults [][]int, theme *Theme) [][]ThemeSymbol {
	reelSymbols := make([][]ThemeSymbol, len(reelResults))

	for reelIndex, reel := range reelResults {
		reelSymbols[reelIndex] = make([]ThemeSymbol, len(reel))

		for rowIndex, symbolID := range reel {
			// 从主题符号映射中获取具体符号
			if symbol, exists := theme.SymbolMap[symbolID]; exists {
				reelSymbols[reelIndex][rowIndex] = symbol
			} else {
				// 使用默认符号
				reelSymbols[reelIndex][rowIndex] = r.getDefaultSymbol(symbolID)
			}
		}
	}

	return reelSymbols
}

// generateWinAnimations 生成获胜动画
func (r *DefaultThemeRenderer) generateWinAnimations(abstract *AbstractGameResult, theme *Theme) []Animation {
	var animations []Animation

	// 根据赢取类型选择动画
	animationType := r.getAnimationTypeForWin(abstract.WinType)

	// 获取主题动画配置
	if themeAnimations, exists := theme.Animations[animationType]; exists {
		// 为每条获胜线生成动画
		for _, winLine := range abstract.WinLines {
			for _, baseAnim := range themeAnimations {
				anim := baseAnim
				// 转换GamePosition为Position
				positions := make([]Position, len(winLine.Positions))
				for i, pos := range winLine.Positions {
					positions[i] = Position{Reel: pos.Reel, Row: pos.Row}
				}
				anim.Target = positions
				animations = append(animations, anim)
			}
		}
	}

	// 特殊功能动画
	for _, feature := range abstract.Features {
		if featureAnims, exists := theme.Animations[AnimationTypeFeatureTrigger]; exists {
			for _, baseAnim := range featureAnims {
				anim := baseAnim
				// 转换GamePosition为Position
				positions := make([]Position, len(feature.Trigger))
				for i, pos := range feature.Trigger {
					positions[i] = Position{Reel: pos.Reel, Row: pos.Row}
				}
				anim.Target = positions
				animations = append(animations, anim)
			}
		}
	}

	return animations
}

// generateSoundEffects 生成音效
func (r *DefaultThemeRenderer) generateSoundEffects(abstract *AbstractGameResult, theme *Theme) []SoundEffect {
	var sounds []SoundEffect

	// 基础旋转音效
	if spinSound, exists := theme.Sounds[SoundTypeSpin]; exists {
		sounds = append(sounds, spinSound)
	}

	// 获胜音效
	if abstract.IsWin {
		soundType := r.getSoundTypeForWin(abstract.WinType)
		if winSound, exists := theme.Sounds[soundType]; exists {
			sounds = append(sounds, winSound)
		}
	}

	// 特殊功能音效
	for range abstract.Features {
		if featureSound, exists := theme.Sounds[SoundTypeFeature]; exists {
			sounds = append(sounds, featureSound)
		}
	}

	return sounds
}

// generateVisualEffects 生成视觉特效
func (r *DefaultThemeRenderer) generateVisualEffects(abstract *AbstractGameResult, theme *Theme) []VisualEffect {
	var effects []VisualEffect

	// 获胜特效
	if abstract.IsWin {
		effectType := r.getEffectTypeForWin(abstract.WinType)
		if baseEffect, exists := theme.Effects[effectType]; exists {
			// 为每条获胜线添加特效
			for _, winLine := range abstract.WinLines {
				effect := baseEffect
				// 转换GamePosition为Position
				positions := make([]Position, len(winLine.Positions))
				for i, pos := range winLine.Positions {
					positions[i] = Position{Reel: pos.Reel, Row: pos.Row}
				}
				effect.Target = positions
				effect.Intensity = r.calculateEffectIntensity(abstract.WinType)
				effects = append(effects, effect)
			}
		}
	}

	return effects
}

// 辅助方法
func (r *DefaultThemeRenderer) getAnimationTypeForWin(winType WinType) AnimationType {
	switch winType {
	case WinTypeJackpot:
		return AnimationTypeJackpot
	case WinTypeBig:
		return AnimationTypeBigWin
	default:
		return AnimationTypeWinLine
	}
}

func (r *DefaultThemeRenderer) getSoundTypeForWin(winType WinType) SoundType {
	switch winType {
	case WinTypeJackpot:
		return SoundTypeJackpot
	case WinTypeBig:
		return SoundTypeBigWin
	default:
		return SoundTypeWin
	}
}

func (r *DefaultThemeRenderer) getEffectTypeForWin(winType WinType) EffectType {
	switch winType {
	case WinTypeJackpot:
		return EffectTypeExplosion
	case WinTypeBig:
		return EffectTypeFlash
	default:
		return EffectTypeGlow
	}
}

func (r *DefaultThemeRenderer) calculateEffectIntensity(winType WinType) float64 {
	switch winType {
	case WinTypeJackpot:
		return 1.0
	case WinTypeBig:
		return 0.8
	case WinTypeMedium:
		return 0.6
	default:
		return 0.4
	}
}

func (r *DefaultThemeRenderer) getDefaultSymbol(symbolID int) ThemeSymbol {
	return ThemeSymbol{
		ID:         symbolID,
		Name:       fmt.Sprintf("Symbol_%d", symbolID),
		ImageURL:   fmt.Sprintf("/images/symbols/default_%d.png", symbolID),
		Rarity:     RarityCommon,
		Properties: make(map[string]interface{}),
	}
}

// 主题管理方法
func (r *DefaultThemeRenderer) GetTheme(themeID string) (*Theme, error) {
	theme, exists := r.themes[themeID]
	if !exists {
		return nil, fmt.Errorf("theme not found: %s", themeID)
	}
	return theme, nil
}

func (r *DefaultThemeRenderer) ListThemes() []string {
	themes := make([]string, 0, len(r.themes))
	for themeID := range r.themes {
		themes = append(themes, themeID)
	}
	return themes
}

func (r *DefaultThemeRenderer) RegisterTheme(theme *Theme) error {
	if theme.ID == "" {
		return fmt.Errorf("theme ID cannot be empty")
	}
	r.themes[theme.ID] = theme
	return nil
}

// registerBuiltinThemes 注册内置主题
func (r *DefaultThemeRenderer) registerBuiltinThemes() {
	// 经典老虎机主题
	classicTheme := &Theme{
		ID:          "classic",
		Name:        "经典老虎机",
		Description: "传统的老虎机主题，包含经典符号",
		Version:     "1.0.0",
		SymbolMap: map[int]ThemeSymbol{
			0: {ID: 0, Name: "樱桃", ImageURL: "/images/classic/cherry.png", Rarity: RarityCommon},
			1: {ID: 1, Name: "柠檬", ImageURL: "/images/classic/lemon.png", Rarity: RarityCommon},
			2: {ID: 2, Name: "橙子", ImageURL: "/images/classic/orange.png", Rarity: RarityCommon},
			3: {ID: 3, Name: "李子", ImageURL: "/images/classic/plum.png", Rarity: RarityRare},
			4: {ID: 4, Name: "葡萄", ImageURL: "/images/classic/grape.png", Rarity: RarityRare},
			5: {ID: 5, Name: "西瓜", ImageURL: "/images/classic/watermelon.png", Rarity: RarityEpic},
			6: {ID: 6, Name: "铃铛", ImageURL: "/images/classic/bell.png", Rarity: RarityEpic},
			7: {ID: 7, Name: "七", ImageURL: "/images/classic/seven.png", Rarity: RarityLegendary},
			8: {ID: 8, Name: "钻石", ImageURL: "/images/classic/diamond.png", Rarity: RarityLegendary},
			9: {ID: 9, Name: "百搭", ImageURL: "/images/classic/wild.png", Rarity: RarityLegendary},
		},
		Background: BackgroundConfig{
			ImageURL:    "/images/classic/background.jpg",
			Color:       "#1a1a2e",
			MusicURL:    "/sounds/classic/ambient.mp3",
			MusicVolume: 0.3,
		},
		Sounds: map[SoundType]SoundEffect{
			SoundTypeSpin:    {Type: SoundTypeSpin, FileURL: "/sounds/classic/spin.wav", Volume: 0.8},
			SoundTypeWin:     {Type: SoundTypeWin, FileURL: "/sounds/classic/win.wav", Volume: 0.9},
			SoundTypeBigWin:  {Type: SoundTypeBigWin, FileURL: "/sounds/classic/bigwin.wav", Volume: 1.0},
			SoundTypeJackpot: {Type: SoundTypeJackpot, FileURL: "/sounds/classic/jackpot.wav", Volume: 1.0},
		},
		Animations: map[AnimationType][]Animation{
			AnimationTypeWinLine: {
				{
					Type:     AnimationTypeWinLine,
					Duration: 1500,
					Sequence: []AnimationFrame{
						{ImageURL: "/images/effects/glow_1.png", Duration: 250, Transform: Transform{Scale: 1.0, Alpha: 0.8}},
						{ImageURL: "/images/effects/glow_2.png", Duration: 250, Transform: Transform{Scale: 1.1, Alpha: 1.0}},
						{ImageURL: "/images/effects/glow_3.png", Duration: 250, Transform: Transform{Scale: 1.2, Alpha: 0.8}},
					},
				},
			},
			AnimationTypeBigWin: {
				{
					Type:     AnimationTypeBigWin,
					Duration: 3000,
					Sequence: []AnimationFrame{
						{ImageURL: "/images/effects/bigwin_1.png", Duration: 500, Transform: Transform{Scale: 0.8, Alpha: 0.0}},
						{ImageURL: "/images/effects/bigwin_2.png", Duration: 500, Transform: Transform{Scale: 1.0, Alpha: 1.0}},
						{ImageURL: "/images/effects/bigwin_3.png", Duration: 500, Transform: Transform{Scale: 1.2, Alpha: 1.0}},
					},
				},
			},
		},
		Effects: map[EffectType]VisualEffect{
			EffectTypeGlow:      {Type: EffectTypeGlow, Duration: 1000, Intensity: 0.8},
			EffectTypeFlash:     {Type: EffectTypeFlash, Duration: 500, Intensity: 1.0},
			EffectTypeExplosion: {Type: EffectTypeExplosion, Duration: 2000, Intensity: 1.0},
		},
		UI: UIConfig{
			ReelFrame: "/images/classic/reel_frame.png",
			Buttons: map[string]ButtonConfig{
				"spin":     {ImageURL: "/images/classic/btn_spin.png", HoverImageURL: "/images/classic/btn_spin_hover.png"},
				"maxbet":   {ImageURL: "/images/classic/btn_maxbet.png", HoverImageURL: "/images/classic/btn_maxbet_hover.png"},
				"autoplay": {ImageURL: "/images/classic/btn_auto.png", HoverImageURL: "/images/classic/btn_auto_hover.png"},
			},
			Colors: map[string]string{
				"primary":   "#ffd700",
				"secondary": "#ff6b35",
				"text":      "#ffffff",
			},
			Fonts: map[string]FontConfig{
				"title":  {Family: "Arial Black", Size: 24, Color: "#ffd700", Style: "bold"},
				"normal": {Family: "Arial", Size: 16, Color: "#ffffff", Style: "normal"},
			},
		},
	}

	r.themes["classic"] = classicTheme

	// 可以继续添加其他主题...
	// 水果主题、埃及主题、海洋主题等
}

// ThemeManager 主题管理器
type ThemeManager struct {
	renderer ThemeRenderer
}

// NewThemeManager 创建主题管理器
func NewThemeManager() *ThemeManager {
	return &ThemeManager{
		renderer: NewDefaultThemeRenderer(),
	}
}

// ProcessGameResult 处理游戏结果，应用主题渲染
func (tm *ThemeManager) ProcessGameResult(abstract *AbstractGameResult, themeID string) (*ThemedGameResult, error) {
	if themeID == "" {
		themeID = "classic" // 默认主题
	}

	return tm.renderer.RenderResult(abstract, themeID)
}

// GetAvailableThemes 获取可用主题列表
func (tm *ThemeManager) GetAvailableThemes() []string {
	return tm.renderer.ListThemes()
}

// LoadThemeFromJSON 从JSON加载主题配置
func (tm *ThemeManager) LoadThemeFromJSON(jsonData []byte) error {
	var theme Theme
	if err := json.Unmarshal(jsonData, &theme); err != nil {
		return fmt.Errorf("failed to parse theme JSON: %w", err)
	}

	return tm.renderer.RegisterTheme(&theme)
}
