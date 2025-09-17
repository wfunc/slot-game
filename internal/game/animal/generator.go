package animal

import (
	"math/rand"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
)

// AnimalGenerator 基于Erlang代码的动物生成器
type AnimalGenerator struct {
	mu     sync.RWMutex
	logger *zap.Logger
	rand   *rand.Rand

	// 房间信息
	roomID uint32

	// 动物权重配置（基于Erlang的appear_animal定义）
	animalWeights map[pb.EAnimal]int

	// 路径系统（基于Erlang的animal_dict）
	leftLines  []*PathLine
	rightLines []*PathLine

	// 特殊动物线路限制
	specialLineIDs []uint32

	// ID生成器
	nextAnimalID uint32
}

// PathLine 路径线定义（基于Erlang的line record）
type PathLine struct {
	ID    uint32  // 线路ID
	Point float32 // 路径点数
}

// NewAnimalGenerator 创建动物生成器
func NewAnimalGenerator(roomID uint32, logger *zap.Logger) *AnimalGenerator {
	gen := &AnimalGenerator{
		logger: logger,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		roomID: roomID,
		animalWeights: map[pb.EAnimal]int{
			pb.EAnimal_turtle:   12,
			pb.EAnimal_cock:     18,
			pb.EAnimal_dog:      14,
			pb.EAnimal_monkey:   10,
			pb.EAnimal_horse:    10,
			pb.EAnimal_ox:       10,
			pb.EAnimal_panda:    10,
			pb.EAnimal_pikachu:  10,
			pb.EAnimal_hippo:    5,
			pb.EAnimal_lion:     5,
			pb.EAnimal_elephant: 15,
			pb.EAnimal_bomber:   1,
			pb.EAnimal_baozi:    10,
			pb.EAnimal_hema:     3,
			pb.EAnimal_zhu:      2,
			pb.EAnimal_lv:       5,
			pb.EAnimal_tuzi:     7,
		},
		specialLineIDs: []uint32{22, 6, 8, 11, 15, 18, 19, 20, 28, 30, 31, 33, 34, 37, 40, 41, 42, 44},
		nextAnimalID:   1,
	}

	gen.initializeLines()
	return gen
}

// initializeLines 初始化路径线（基于Erlang的init_line逻辑）
func (g *AnimalGenerator) initializeLines() {
	// 基于Erlang的left_line和right_line定义
	leftLinePoints := []float32{36, 36, 40, 32, 42, 36, 38, 36, 40, 40, 36, 40, 40, 36, 34, 42, 34, 30, 36, 38, 34, 34}
	rightLinePoints := []float32{36, 36, 40, 32, 42, 36, 38, 36, 40, 40, 36, 40, 40, 36, 34, 42, 34, 30, 36, 38, 34, 34}

	// 创建左侧路径线
	g.leftLines = make([]*PathLine, len(leftLinePoints))
	for i, point := range leftLinePoints {
		g.leftLines[i] = &PathLine{
			ID:    uint32(i + 1), // ID从1开始
			Point: point,
		}
	}

	// 创建右侧路径线
	g.rightLines = make([]*PathLine, len(rightLinePoints))
	for i, point := range rightLinePoints {
		g.rightLines[i] = &PathLine{
			ID:    uint32(len(leftLinePoints) + i + 1), // 右侧ID从左侧末尾继续
			Point: point,
		}
	}

	g.logger.Info("[AnimalGenerator] 路径线初始化完成",
		zap.Int("left_lines", len(g.leftLines)),
		zap.Int("right_lines", len(g.rightLines)))
}

// SelectAnimal 选择动物类型（基于Erlang的alg_animal:appear()）
func (g *AnimalGenerator) SelectAnimal() pb.EAnimal {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// 计算总权重
	totalWeight := 0
	for _, weight := range g.animalWeights {
		totalWeight += weight
	}

	// 随机选择
	random := g.rand.Intn(totalWeight)
	current := 0

	for animal, weight := range g.animalWeights {
		current += weight
		if random < current {
			return animal
		}
	}

	// 默认返回乌龟
	return pb.EAnimal_turtle
}

// SelectAnimalExcluding 选择动物类型，排除指定类型（基于Erlang的appear_without_it）
func (g *AnimalGenerator) SelectAnimalExcluding(excludeTypes []pb.EAnimal) pb.EAnimal {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// 创建排除指定动物的权重表
	availableWeights := make(map[pb.EAnimal]int)
	for animal, weight := range g.animalWeights {
		excluded := false
		for _, excludeType := range excludeTypes {
			if animal == excludeType {
				excluded = true
				break
			}
		}
		if !excluded {
			availableWeights[animal] = weight
		}
	}

	// 如果没有可用动物，返回乌龟
	if len(availableWeights) == 0 {
		return pb.EAnimal_turtle
	}

	// 计算总权重
	totalWeight := 0
	for _, weight := range availableWeights {
		totalWeight += weight
	}

	// 随机选择
	random := g.rand.Intn(totalWeight)
	current := 0

	for animal, weight := range availableWeights {
		current += weight
		if random < current {
			return animal
		}
	}

	// 默认返回乌龟
	return pb.EAnimal_turtle
}

// SelectPathLine 选择路径线（基于Erlang的animal_status:add_animal逻辑）
func (g *AnimalGenerator) SelectPathLine(animalType pb.EAnimal) *PathLine {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var availableLines []*PathLine

	// 特殊动物只能使用特殊线路
	if g.isSpecialAnimal(animalType) {
		allLines := append(g.leftLines, g.rightLines...)
		for _, line := range allLines {
			for _, specialID := range g.specialLineIDs {
				if line.ID == specialID {
					availableLines = append(availableLines, line)
					break
				}
			}
		}
	} else {
		// 普通动物可以使用所有线路
		availableLines = append(g.leftLines, g.rightLines...)
	}

	// 如果没有可用线路，返回第一条线路
	if len(availableLines) == 0 {
		if len(g.leftLines) > 0 {
			return g.leftLines[0]
		}
		return nil
	}

	// 随机选择线路
	return availableLines[g.rand.Intn(len(availableLines))]
}

// isSpecialAnimal 判断是否为特殊动物（基于Erlang逻辑）
func (g *AnimalGenerator) isSpecialAnimal(animalType pb.EAnimal) bool {
	specialAnimals := []pb.EAnimal{
		pb.EAnimal_panda,
		pb.EAnimal_hippo,
		pb.EAnimal_lion,
		pb.EAnimal_elephant,
		pb.EAnimal_pikachu,
		pb.EAnimal_bomber,
	}

	for _, special := range specialAnimals {
		if animalType == special {
			return true
		}
	}
	return false
}

// GenerateAnimal 生成一个新动物（基于Erlang的add_animal逻辑）
func (g *AnimalGenerator) GenerateAnimal(excludeTypes []pb.EAnimal) *AnimalRoute {
	// 选择动物类型
	var animalType pb.EAnimal
	if len(excludeTypes) > 0 {
		animalType = g.SelectAnimalExcluding(excludeTypes)
	} else {
		animalType = g.SelectAnimal()
	}

	// 选择路径线
	pathLine := g.SelectPathLine(animalType)
	if pathLine == nil {
		g.logger.Error("[AnimalGenerator] 无法选择路径线", zap.String("animal", animalType.String()))
		return nil
	}

	// 生成唯一ID
	g.mu.Lock()
	animalID := g.nextAnimalID
	g.nextAnimalID++
	g.mu.Unlock()

	// 计算动物路径参数（基于Erlang逻辑）
	var newPoint float32
	var startPoint uint32

	switch animalType {
	case pb.EAnimal_elephant, pb.EAnimal_bomber:
		// 大象和炸弹人延迟出现
		newPoint = pathLine.Point + 5 // ElephantComingTime = 5秒
		startPoint = 1
	default:
		// 普通动物有40%概率立即出现，60%概率延迟5-10秒
		if g.rand.Float32() < 0.4 {
			newPoint = pathLine.Point
		} else {
			delay := float32(g.rand.Intn(6) + 5) // 5-10秒随机延迟
			newPoint = pathLine.Point - delay
		}
		startPoint = uint32(pathLine.Point - newPoint + 1)
	}

	// 计算结束时间
	endTime := time.Now().Add(time.Duration(newPoint) * time.Second)

	// 判断是否携带红包
	redState := g.shouldHaveRedBag(animalType)

	// 创建动物路线（基于Erlang原版，不包含HP）
	route := &AnimalRoute{
		ID:      animalID,
		Animal:  animalType,
		LineID:  pathLine.ID,
		Point:   startPoint,
		Red:     redState,
		State:   pb.EAnimalState_state_normal,
		SpawnAt: time.Now(),
	}

	g.logger.Info("[AnimalGenerator] 生成新动物",
		zap.Uint32("room_id", g.roomID),
		zap.Uint32("id", animalID),
		zap.String("type", animalType.String()),
		zap.Uint32("line_id", pathLine.ID),
		zap.Uint32("start_point", startPoint),
		zap.Bool("red_bag", redState),
		zap.Duration("duration", time.Until(endTime)))

	return route
}

// shouldHaveRedBag 判断动物是否应该携带红包（基于Erlang的get_animal_red_state）
func (g *AnimalGenerator) shouldHaveRedBag(animalType pb.EAnimal) bool {
	// 红包动物列表
	redBagAnimals := []pb.EAnimal{
		pb.EAnimal_baozi,    // 豹子
		pb.EAnimal_pikachu,  // 皮卡丘
		pb.EAnimal_hippo,    // 河马
		pb.EAnimal_lion,     // 狮子
		pb.EAnimal_elephant, // 大象
	}

	// 检查是否为红包动物
	for _, redAnimal := range redBagAnimals {
		if animalType == redAnimal {
			// 暂时不打开红包
			return false
		}
	}

	return false
}

// GetAllLines 获取所有路径线
func (g *AnimalGenerator) GetAllLines() []*PathLine {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var allLines []*PathLine
	allLines = append(allLines, g.leftLines...)
	allLines = append(allLines, g.rightLines...)
	return allLines
}

// GetLineByID 根据ID获取路径线
func (g *AnimalGenerator) GetLineByID(lineID uint32) *PathLine {
	g.mu.RLock()
	defer g.mu.RUnlock()

	allLines := append(g.leftLines, g.rightLines...)
	for _, line := range allLines {
		if line.ID == lineID {
			return line
		}
	}
	return nil
}

