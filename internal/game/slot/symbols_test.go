package slot

import (
	"testing"
)

func TestGetSymbolInfo(t *testing.T) {
	tests := []struct {
		name     string
		symbolID int
		wantName string
		wantType string
		isSpecial bool
	}{
		{
			name:     "基础符号一筒",
			symbolID: SYMBOL_MAHJONG_1,
			wantName: "一筒",
			wantType: "normal",
			isSpecial: false,
		},
		{
			name:     "Wild符号",
			symbolID: SYMBOL_WILD,
			wantName: "Wild",
			wantType: "wild",
			isSpecial: true,
		},
		{
			name:     "Animal Wild符号",
			symbolID: SYMBOL_ANIMAL_WILD,
			wantName: "动物Wild",
			wantType: "bonus",
			isSpecial: true,
		},
		{
			name:     "Animal Bonus符号",
			symbolID: SYMBOL_ANIMAL_BONUS,
			wantName: "动物Bonus",
			wantType: "bonus",
			isSpecial: true,
		},
		{
			name:     "金色符号",
			symbolID: SYMBOL_GOLDEN_1,
			wantName: "金色一筒",
			wantType: "golden",
			isSpecial: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetSymbolInfo(tt.symbolID)
			
			if info.Name != tt.wantName {
				t.Errorf("GetSymbolInfo() Name = %v, want %v", info.Name, tt.wantName)
			}
			
			if info.Type != tt.wantType {
				t.Errorf("GetSymbolInfo() Type = %v, want %v", info.Type, tt.wantType)
			}
			
			if info.IsSpecial != tt.isSpecial {
				t.Errorf("GetSymbolInfo() IsSpecial = %v, want %v", info.IsSpecial, tt.isSpecial)
			}
		})
	}
}

func TestIsAnimalTriggerSymbol(t *testing.T) {
	tests := []struct {
		name     string
		symbolID int
		want     bool
	}{
		{
			name:     "Animal Wild符号应该触发",
			symbolID: SYMBOL_ANIMAL_WILD,
			want:     true,
		},
		{
			name:     "Animal Bonus符号应该触发",
			symbolID: SYMBOL_ANIMAL_BONUS,
			want:     true,
		},
		{
			name:     "普通符号不应该触发",
			symbolID: SYMBOL_MAHJONG_1,
			want:     false,
		},
		{
			name:     "Wild符号不应该触发",
			symbolID: SYMBOL_WILD,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAnimalTriggerSymbol(tt.symbolID); got != tt.want {
				t.Errorf("IsAnimalTriggerSymbol() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanBeGolden(t *testing.T) {
	tests := []struct {
		name     string
		symbolID int
		want     bool
	}{
		{
			name:     "基础符号可以变金色",
			symbolID: SYMBOL_MAHJONG_1,
			want:     true,
		},
		{
			name:     "Wild符号不能变金色",
			symbolID: SYMBOL_WILD,
			want:     false,
		},
		{
			name:     "Animal符号不能变金色",
			symbolID: SYMBOL_ANIMAL_WILD,
			want:     false,
		},
		{
			name:     "已经是金色符号不能再变金色",
			symbolID: SYMBOL_GOLDEN_1,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanBeGolden(tt.symbolID); got != tt.want {
				t.Errorf("CanBeGolden() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToGolden(t *testing.T) {
	tests := []struct {
		name     string
		symbolID int
		want     int
	}{
		{
			name:     "转换一筒为金色",
			symbolID: SYMBOL_MAHJONG_1,
			want:     SYMBOL_GOLDEN_1,
		},
		{
			name:     "转换八筒为金色",
			symbolID: SYMBOL_MAHJONG_8,
			want:     SYMBOL_GOLDEN_8,
		},
		{
			name:     "Wild符号不能转换",
			symbolID: SYMBOL_WILD,
			want:     SYMBOL_WILD,
		},
		{
			name:     "Animal符号不能转换",
			symbolID: SYMBOL_ANIMAL_WILD,
			want:     SYMBOL_ANIMAL_WILD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToGolden(tt.symbolID); got != tt.want {
				t.Errorf("ConvertToGolden() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsGoldenSymbol(t *testing.T) {
	tests := []struct {
		name     string
		symbolID int
		want     bool
	}{
		{
			name:     "金色一筒是金色符号",
			symbolID: SYMBOL_GOLDEN_1,
			want:     true,
		},
		{
			name:     "金色八筒是金色符号",
			symbolID: SYMBOL_GOLDEN_8,
			want:     true,
		},
		{
			name:     "普通一筒不是金色符号",
			symbolID: SYMBOL_MAHJONG_1,
			want:     false,
		},
		{
			name:     "Wild不是金色符号",
			symbolID: SYMBOL_WILD,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGoldenSymbol(tt.symbolID); got != tt.want {
				t.Errorf("IsGoldenSymbol() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBaseSymbol(t *testing.T) {
	tests := []struct {
		name           string
		goldenSymbolID int
		want           int
	}{
		{
			name:           "金色一筒的基础符号是一筒",
			goldenSymbolID: SYMBOL_GOLDEN_1,
			want:           SYMBOL_MAHJONG_1,
		},
		{
			name:           "金色八筒的基础符号是八筒",
			goldenSymbolID: SYMBOL_GOLDEN_8,
			want:           SYMBOL_MAHJONG_8,
		},
		{
			name:           "普通符号返回自身",
			goldenSymbolID: SYMBOL_MAHJONG_1,
			want:           SYMBOL_MAHJONG_1,
		},
		{
			name:           "Wild符号返回自身",
			goldenSymbolID: SYMBOL_WILD,
			want:           SYMBOL_WILD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBaseSymbol(tt.goldenSymbolID); got != tt.want {
				t.Errorf("GetBaseSymbol() = %v, want %v", got, tt.want)
			}
		})
	}
}