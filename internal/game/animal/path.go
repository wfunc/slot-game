package animal

import (
	"math"
	"sync"
	"time"
)

// Path 动物移动路径
type Path struct {
	ID          uint32        // 路径ID
	Name        string        // 路径名称
	Points      []PathPoint   // 路径点
	Length      float32       // 路径总长度
	ControlType PathType      // 路径类型
	mu          sync.RWMutex  // 并发控制
}

// PathPoint 路径点
type PathPoint struct {
	X, Y      float32 // 坐标
	Direction float32 // 朝向角度
	Speed     float32 // 速度调节系数
}

// PathType 路径类型
type PathType int

const (
	PathTypeStraight PathType = iota // 直线路径
	PathTypeCurve                     // 曲线路径
	PathTypeBezier                    // 贝塞尔曲线
	PathTypeCircle                    // 圆形路径
	PathTypeZigzag                    // 之字形路径
)

// PathManager 路径管理器
type PathManager struct {
	paths map[uint32]*Path
	mu    sync.RWMutex
}

// NewPathManager 创建路径管理器
func NewPathManager() *PathManager {
	pm := &PathManager{
		paths: make(map[uint32]*Path),
	}
	pm.InitDefaultPaths()
	return pm
}

// InitDefaultPaths 初始化默认路径
func (pm *PathManager) InitDefaultPaths() {
	// 路径1: 上方水平移动
	pm.AddPath(&Path{
		ID:          1,
		Name:        "上方直线",
		ControlType: PathTypeStraight,
		Points: []PathPoint{
			{X: -50, Y: 150, Direction: 0, Speed: 1.0},
			{X: 100, Y: 150, Direction: 0, Speed: 1.0},
			{X: 300, Y: 150, Direction: 0, Speed: 1.0},
			{X: 500, Y: 150, Direction: 0, Speed: 1.0},
			{X: 700, Y: 150, Direction: 0, Speed: 1.0},
			{X: 850, Y: 150, Direction: 0, Speed: 1.0},
		},
	})

	// 路径2: 中部曲线
	pm.AddPath(&Path{
		ID:          2,
		Name:        "中部曲线",
		ControlType: PathTypeCurve,
		Points: []PathPoint{
			{X: -50, Y: 300, Direction: 0, Speed: 1.0},
			{X: 150, Y: 280, Direction: -10, Speed: 1.1},
			{X: 300, Y: 320, Direction: 10, Speed: 0.9},
			{X: 450, Y: 280, Direction: -10, Speed: 1.1},
			{X: 600, Y: 320, Direction: 10, Speed: 0.9},
			{X: 850, Y: 300, Direction: 0, Speed: 1.0},
		},
	})

	// 路径3: 下方之字形
	pm.AddPath(&Path{
		ID:          3,
		Name:        "下方之字形",
		ControlType: PathTypeZigzag,
		Points: []PathPoint{
			{X: -50, Y: 450, Direction: 0, Speed: 1.0},
			{X: 100, Y: 420, Direction: -20, Speed: 1.2},
			{X: 250, Y: 480, Direction: 20, Speed: 0.8},
			{X: 400, Y: 420, Direction: -20, Speed: 1.2},
			{X: 550, Y: 480, Direction: 20, Speed: 0.8},
			{X: 700, Y: 450, Direction: 0, Speed: 1.0},
			{X: 850, Y: 450, Direction: 0, Speed: 1.0},
		},
	})

	// 路径4: 圆形路径（环绕屏幕）
	pm.AddPath(&Path{
		ID:          4,
		Name:        "环形路径",
		ControlType: PathTypeCircle,
		Points:      generateCirclePath(400, 300, 200, 32),
	})

	// 路径5: 贝塞尔曲线（优美弧线）
	pm.AddPath(&Path{
		ID:          5,
		Name:        "贝塞尔弧线",
		ControlType: PathTypeBezier,
		Points:      generateBezierPath(-50, 400, 850, 200, 400, 100, 400, 500, 20),
	})

	// 计算所有路径的长度
	for _, path := range pm.paths {
		path.calculateLength()
	}
}

// AddPath 添加路径
func (pm *PathManager) AddPath(path *Path) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	path.calculateLength()
	pm.paths[path.ID] = path
}

// GetPath 获取路径
func (pm *PathManager) GetPath(id uint32) *Path {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.paths[id]
}

// GetRandomPathID 获取随机路径ID
func (pm *PathManager) GetRandomPathID() uint32 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 简单实现：返回1-5的随机路径
	return uint32((time.Now().UnixNano() % 5) + 1)
}

// GetPosition 根据进度获取位置
func (p *Path) GetPosition(progress float32) (x, y float32) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.Points) == 0 {
		return 0, 0
	}

	// 限制进度在0-1之间
	if progress <= 0 {
		return p.Points[0].X, p.Points[0].Y
	}
	if progress >= 1 {
		last := p.Points[len(p.Points)-1]
		return last.X, last.Y
	}

	// 根据路径类型使用不同的插值方法
	switch p.ControlType {
	case PathTypeBezier:
		return p.getBezierPosition(progress)
	case PathTypeCircle:
		return p.getCirclePosition(progress)
	default:
		return p.getLinearPosition(progress)
	}
}

// GetDirection 根据进度获取朝向
func (p *Path) GetDirection(progress float32) float32 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.Points) < 2 {
		return 0
	}

	// 获取当前位置和稍后位置
	x1, y1 := p.GetPosition(progress)
	x2, y2 := p.GetPosition(progress + 0.01) // 向前看一点点

	// 计算方向角度
	dx := x2 - x1
	dy := y2 - y1
	return float32(math.Atan2(float64(dy), float64(dx)) * 180 / math.Pi)
}

// getLinearPosition 线性插值获取位置
func (p *Path) getLinearPosition(progress float32) (x, y float32) {
	if len(p.Points) < 2 {
		return p.Points[0].X, p.Points[0].Y
	}

	// 计算总距离对应的进度
	totalDist := p.Length * progress
	currentDist := float32(0)

	// 找到对应的线段
	for i := 0; i < len(p.Points)-1; i++ {
		p1 := p.Points[i]
		p2 := p.Points[i+1]

		segmentDist := distance(p1.X, p1.Y, p2.X, p2.Y)

		if currentDist+segmentDist >= totalDist {
			// 在这个线段上
			t := (totalDist - currentDist) / segmentDist
			x = p1.X + (p2.X-p1.X)*t
			y = p1.Y + (p2.Y-p1.Y)*t
			return x, y
		}

		currentDist += segmentDist
	}

	// 到达终点
	last := p.Points[len(p.Points)-1]
	return last.X, last.Y
}

// getBezierPosition 贝塞尔曲线插值
func (p *Path) getBezierPosition(t float32) (x, y float32) {
	if len(p.Points) < 4 {
		return p.getLinearPosition(t)
	}

	// 使用前4个点作为控制点的三次贝塞尔曲线
	p0, p1, p2, p3 := p.Points[0], p.Points[1], p.Points[2], p.Points[3]

	// 三次贝塞尔曲线公式
	mt := 1 - t
	mt2 := mt * mt
	mt3 := mt2 * mt
	t2 := t * t
	t3 := t2 * t

	x = mt3*p0.X + 3*mt2*t*p1.X + 3*mt*t2*p2.X + t3*p3.X
	y = mt3*p0.Y + 3*mt2*t*p1.Y + 3*mt*t2*p2.Y + t3*p3.Y

	return x, y
}

// getCirclePosition 圆形路径插值
func (p *Path) getCirclePosition(progress float32) (x, y float32) {
	if len(p.Points) == 0 {
		return 0, 0
	}

	// 使用进度计算索引
	index := int(progress * float32(len(p.Points)))
	if index >= len(p.Points) {
		index = len(p.Points) - 1
	}

	return p.Points[index].X, p.Points[index].Y
}

// calculateLength 计算路径总长度
func (p *Path) calculateLength() {
	p.Length = 0
	if len(p.Points) < 2 {
		return
	}

	for i := 0; i < len(p.Points)-1; i++ {
		p1 := p.Points[i]
		p2 := p.Points[i+1]
		p.Length += distance(p1.X, p1.Y, p2.X, p2.Y)
	}
}

// 辅助函数

// distance 计算两点间距离
func distance(x1, y1, x2, y2 float32) float32 {
	dx := x2 - x1
	dy := y2 - y1
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// generateCirclePath 生成圆形路径点
func generateCirclePath(centerX, centerY, radius float32, segments int) []PathPoint {
	points := make([]PathPoint, segments)
	angleStep := 2 * math.Pi / float64(segments)

	for i := 0; i < segments; i++ {
		angle := float64(i) * angleStep
		x := centerX + radius*float32(math.Cos(angle))
		y := centerY + radius*float32(math.Sin(angle))
		direction := float32(angle*180/math.Pi + 90) // 切线方向

		points[i] = PathPoint{
			X:         x,
			Y:         y,
			Direction: direction,
			Speed:     1.0,
		}
	}

	return points
}

// generateBezierPath 生成贝塞尔曲线路径点
func generateBezierPath(x0, y0, x3, y3, x1, y1, x2, y2 float32, segments int) []PathPoint {
	points := make([]PathPoint, segments)

	for i := 0; i < segments; i++ {
		t := float32(i) / float32(segments-1)
		mt := 1 - t
		mt2 := mt * mt
		mt3 := mt2 * mt
		t2 := t * t
		t3 := t2 * t

		// 三次贝塞尔曲线
		x := mt3*x0 + 3*mt2*t*x1 + 3*mt*t2*x2 + t3*x3
		y := mt3*y0 + 3*mt2*t*y1 + 3*mt*t2*y2 + t3*y3

		// 计算切线方向
		dx := -3*mt2*x0 + 3*(mt2-2*mt*t)*x1 + 3*(2*mt*t-t2)*x2 + 3*t2*x3
		dy := -3*mt2*y0 + 3*(mt2-2*mt*t)*y1 + 3*(2*mt*t-t2)*y2 + 3*t2*y3
		direction := float32(math.Atan2(float64(dy), float64(dx)) * 180 / math.Pi)

		points[i] = PathPoint{
			X:         x,
			Y:         y,
			Direction: direction,
			Speed:     1.0,
		}
	}

	return points
}

