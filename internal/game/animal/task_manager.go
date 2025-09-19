package animal

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
)

// TaskManager 任务管理器（已在 types.go 定义，这里实现方法）

// NewTaskManager 创建任务管理器
func NewTaskManager() *TaskManager {
	tm := &TaskManager{
		dailyTasks:   make(map[uint32]*ZooTask),
		weeklyTasks:  make(map[uint32]*ZooTask),
		freeTasks:    make(map[uint32]*ZooTask),
		achievements: make(map[uint32]*ZooTask),
		lastReset:    time.Now(),
	}

	// 初始化任务
	tm.initDailyTasks()
	tm.initWeeklyTasks()
	tm.initFreeTasks()
	tm.initAchievements()

	return tm
}

// initDailyTasks 初始化每日任务
func (tm *TaskManager) initDailyTasks() {
	tasks := []*ZooTask{
		{
			ID:          1,
			Type:        "daily",
			Description: "击杀10只乌龟",
			Target:      10,
			Progress:    0,
			Reward:      1000,
			Status:      "active",
			Animal:      pb.EAnimal_turtle,
		},
		{
			ID:          2,
			Type:        "daily",
			Description: "击杀5只熊猫",
			Target:      5,
			Progress:    0,
			Reward:      5000,
			Status:      "active",
			Animal:      pb.EAnimal_panda,
		},
		{
			ID:          3,
			Type:        "daily",
			Description: "累计下注10000金豆",
			Target:      10000,
			Progress:    0,
			Reward:      2000,
			Status:      "active",
		},
		{
			ID:          4,
			Type:        "daily",
			Description: "使用10次技能",
			Target:      10,
			Progress:    0,
			Reward:      1500,
			Status:      "active",
		},
		{
			ID:          5,
			Type:        "daily",
			Description: "触发3次闪电链",
			Target:      3,
			Progress:    0,
			Reward:      3000,
			Status:      "active",
			Animal:      pb.EAnimal_pikachu,
		},
	}

	for _, task := range tasks {
		tm.dailyTasks[task.ID] = task
	}
}

// initWeeklyTasks 初始化每周任务
func (tm *TaskManager) initWeeklyTasks() {
	tasks := []*ZooTask{
		{
			ID:          101,
			Type:        "weekly",
			Description: "击杀100只动物",
			Target:      100,
			Progress:    0,
			Reward:      20000,
			Status:      "active",
		},
		{
			ID:          102,
			Type:        "weekly",
			Description: "击杀10只大象",
			Target:      10,
			Progress:    0,
			Reward:      50000,
			Status:      "active",
			Animal:      pb.EAnimal_elephant,
		},
		{
			ID:          103,
			Type:        "weekly",
			Description: "累计赢取100000金豆",
			Target:      100000,
			Progress:    0,
			Reward:      30000,
			Status:      "active",
		},
	}

	for _, task := range tasks {
		tm.weeklyTasks[task.ID] = task
	}
}

// initFreeTasks 初始化体验场任务
func (tm *TaskManager) initFreeTasks() {
	tasks := []*ZooTask{
		{
			ID:          201,
			Type:        "free",
			Description: "体验场击杀20只动物",
			Target:      20,
			Progress:    0,
			Reward:      5000, // 体验币
			Status:      "active",
		},
		{
			ID:          202,
			Type:        "free",
			Description: "体验场使用100档位下注10次",
			Target:      10,
			Progress:    0,
			Reward:      2000,
			Status:      "active",
			BetLevel:    100,
		},
		{
			ID:          203,
			Type:        "free",
			Description: "体验场使用500档位下注5次",
			Target:      5,
			Progress:    0,
			Reward:      3000,
			Status:      "active",
			BetLevel:    500,
		},
		{
			ID:          204,
			Type:        "free",
			Description: "体验场使用1000档位下注3次",
			Target:      3,
			Progress:    0,
			Reward:      5000,
			Status:      "active",
			BetLevel:    1000,
		},
	}

	for _, task := range tasks {
		tm.freeTasks[task.ID] = task
	}
}

// initAchievements 初始化成就任务
func (tm *TaskManager) initAchievements() {
	achievements := []*ZooTask{
		{
			ID:          1001,
			Type:        "achievement",
			Description: "累计击杀1000只动物",
			Target:      1000,
			Progress:    0,
			Reward:      100000,
			Status:      "active",
		},
		{
			ID:          1002,
			Type:        "achievement",
			Description: "触发彩金池",
			Target:      1,
			Progress:    0,
			Reward:      50000,
			Status:      "active",
		},
		{
			ID:          1003,
			Type:        "achievement",
			Description: "单次击杀获得10000金豆",
			Target:      1,
			Progress:    0,
			Reward:      20000,
			Status:      "active",
		},
	}

	for _, task := range achievements {
		tm.achievements[task.ID] = task
	}
}

// UpdateProgress 更新任务进度
func (tm *TaskManager) UpdateProgress(event TaskEvent) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查是否需要重置任务
	tm.checkReset()

	// 更新相关任务
	tm.updateDailyTasks(event)
	tm.updateWeeklyTasks(event)
	tm.updateFreeTasks(event)
	tm.updateAchievements(event)
}

// TaskEvent 任务事件
type TaskEvent struct {
	Type      string       // "kill", "bet", "skill", "win"
	Animal    pb.EAnimal   // 击杀的动物类型
	BetAmount uint32       // 下注金额
	WinAmount uint32       // 赢取金额
	RoomType  pb.EZooType  // 房间类型
	BetLevel  uint32       // 下注档位
	IsJackpot bool         // 是否触发彩金
	Count     uint32       // 数量
}

// updateDailyTasks 更新每日任务
func (tm *TaskManager) updateDailyTasks(event TaskEvent) {
	for _, task := range tm.dailyTasks {
		if task.Status != "active" {
			continue
		}

		updated := false

		switch event.Type {
		case "kill":
			// 击杀特定动物的任务
			if task.Animal != 0 && task.Animal == event.Animal {
				task.Progress += event.Count
				updated = true
			}
			// 击杀任意动物的任务
			if task.Animal == 0 && task.ID == 3 {
				task.Progress += event.Count
				updated = true
			}

		case "bet":
			// 累计下注任务
			if task.ID == 3 {
				task.Progress += event.BetAmount
				updated = true
			}

		case "skill":
			// 使用技能任务
			if task.ID == 4 {
				task.Progress += event.Count
				updated = true
			}
		}

		if updated && task.Progress >= task.Target {
			task.Status = "completed"
		}
	}
}

// updateWeeklyTasks 更新每周任务
func (tm *TaskManager) updateWeeklyTasks(event TaskEvent) {
	for _, task := range tm.weeklyTasks {
		if task.Status != "active" {
			continue
		}

		updated := false

		switch event.Type {
		case "kill":
			// 击杀特定动物
			if task.Animal != 0 && task.Animal == event.Animal {
				task.Progress += event.Count
				updated = true
			}
			// 击杀任意动物
			if task.ID == 101 {
				task.Progress += event.Count
				updated = true
			}

		case "win":
			// 累计赢取
			if task.ID == 103 {
				task.Progress += event.WinAmount
				updated = true
			}
		}

		if updated && task.Progress >= task.Target {
			task.Status = "completed"
		}
	}
}

// updateFreeTasks 更新体验场任务
func (tm *TaskManager) updateFreeTasks(event TaskEvent) {
	// 只在体验场更新
	if event.RoomType != pb.EZooType_free {
		return
	}

	for _, task := range tm.freeTasks {
		if task.Status != "active" {
			continue
		}

		updated := false

		switch event.Type {
		case "kill":
			// 体验场击杀任务
			if task.ID == 201 {
				task.Progress += event.Count
				updated = true
			}

		case "bet":
			// 特定档位下注任务
			if task.BetLevel != 0 && task.BetLevel == event.BetLevel {
				task.Progress++
				updated = true
			}
		}

		if updated && task.Progress >= task.Target {
			task.Status = "completed"
		}
	}
}

// updateAchievements 更新成就
func (tm *TaskManager) updateAchievements(event TaskEvent) {
	for _, task := range tm.achievements {
		if task.Status != "active" {
			continue
		}

		updated := false

		switch task.ID {
		case 1001:
			// 累计击杀
			if event.Type == "kill" {
				task.Progress += event.Count
				updated = true
			}

		case 1002:
			// 触发彩金
			if event.IsJackpot {
				task.Progress = 1
				updated = true
			}

		case 1003:
			// 单次大奖
			if event.Type == "win" && event.WinAmount >= 10000 {
				task.Progress = 1
				updated = true
			}
		}

		if updated && task.Progress >= task.Target {
			task.Status = "completed"
		}
	}
}

// checkReset 检查是否需要重置任务
func (tm *TaskManager) checkReset() {
	now := time.Now()

	// 每日重置（每天0点）
	if now.Day() != tm.lastReset.Day() {
		tm.resetDailyTasks()
	}

	// 每周重置（每周一0点）
	if now.Weekday() == time.Monday && now.Day() != tm.lastReset.Day() {
		tm.resetWeeklyTasks()
	}

	tm.lastReset = now
}

// resetDailyTasks 重置每日任务
func (tm *TaskManager) resetDailyTasks() {
	for _, task := range tm.dailyTasks {
		task.Progress = 0
		task.Status = "active"
	}
}

// resetWeeklyTasks 重置每周任务
func (tm *TaskManager) resetWeeklyTasks() {
	for _, task := range tm.weeklyTasks {
		task.Progress = 0
		task.Status = "active"
	}
}

// GetActiveTasks 获取活跃任务
func (tm *TaskManager) GetActiveTasks(roomType pb.EZooType) []*ZooTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tasks []*ZooTask

	// 每日任务
	for _, task := range tm.dailyTasks {
		if task.Status == "active" {
			tasks = append(tasks, task)
		}
	}

	// 每周任务
	for _, task := range tm.weeklyTasks {
		if task.Status == "active" {
			tasks = append(tasks, task)
		}
	}

	// 体验场任务（只在体验场显示）
	if roomType == pb.EZooType_free {
		for _, task := range tm.freeTasks {
			if task.Status == "active" {
				tasks = append(tasks, task)
			}
		}
	}

	// 成就
	for _, task := range tm.achievements {
		if task.Status == "active" {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// ClaimReward 领取任务奖励
func (tm *TaskManager) ClaimReward(taskID uint32) (uint64, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 查找任务
	var task *ZooTask
	if t, ok := tm.dailyTasks[taskID]; ok {
		task = t
	} else if t, ok := tm.weeklyTasks[taskID]; ok {
		task = t
	} else if t, ok := tm.freeTasks[taskID]; ok {
		task = t
	} else if t, ok := tm.achievements[taskID]; ok {
		task = t
	}

	if task == nil {
		return 0, fmt.Errorf("task not found")
	}

	if task.Status != "completed" {
		return 0, fmt.Errorf("task not completed")
	}

	task.Status = "claimed"
	return task.Reward, nil
}

// TaskLog 任务日志（代替 PZooTaskLog）
type TaskLog struct {
	Time   uint32
	First  uint32
	Second uint32
	Third  uint32
}

// GetTaskLog 获取任务完成记录（用于协议 4107/4108）
func (tm *TaskManager) GetTaskLog(isFree bool) []*TaskLog {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 模拟历史记录（实际应该从数据库读取）
	var logs []*TaskLog

	for i := 0; i < 7; i++ {
		date := time.Now().AddDate(0, 0, -i)
		log := &TaskLog{
			Time:   uint32(date.Unix()),
			First:  uint32(rand.Intn(10)),  // 第一档完成数量
			Second: uint32(rand.Intn(8)),   // 第二档完成数量
			Third:  uint32(rand.Intn(5)),   // 第三档完成数量
		}
		logs = append(logs, log)
	}

	return logs
}