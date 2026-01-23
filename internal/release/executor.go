package release

import (
	"context"
	"log"
	"time"

	"go_cmdb/internal/agentclient"
	"go_cmdb/internal/model"
	"gorm.io/gorm"
)

// Executor 发布执行器
type Executor struct {
	db          *gorm.DB
	agentClient *agentclient.Client
	interval    time.Duration
}

// NewExecutor 创建发布执行器
func NewExecutor(db *gorm.DB, agentClient *agentclient.Client, interval time.Duration) *Executor {
	return &Executor{
		db:          db,
		agentClient: agentClient,
		interval:    interval,
	}
}

// RunOnce 执行一次扫描
func (e *Executor) RunOnce() error {
	log.Println("[Executor] Running once...")

	// 查询可执行的release_task（status=pending或running）
	var tasks []model.ReleaseTask
	if err := e.db.Where("status IN ?", []string{
		string(model.ReleaseTaskStatusPending),
		string(model.ReleaseTaskStatusRunning),
	}).Order("id ASC").Find(&tasks).Error; err != nil {
		log.Printf("[Executor] Failed to query tasks: %v", err)
		return err
	}

	if len(tasks) == 0 {
		log.Println("[Executor] No tasks to execute")
		return nil
	}

	log.Printf("[Executor] Found %d tasks to execute", len(tasks))

	// P0简化：只处理第一个任务（避免并发发布）
	task := &tasks[0]
	log.Printf("[Executor] Processing task %d (status=%s)", task.ID, task.Status)

	// 状态抢占：尝试将pending状态的任务标记为running
	if task.Status == model.ReleaseTaskStatusPending {
		result := e.db.Model(&model.ReleaseTask{}).
			Where("id = ? AND status = ?", task.ID, model.ReleaseTaskStatusPending).
			Update("status", model.ReleaseTaskStatusRunning)

		if result.Error != nil {
			log.Printf("[Executor] Failed to update task status: %v", result.Error)
			return result.Error
		}

		if result.RowsAffected == 0 {
			// 已被其他进程抢占，跳过
			log.Printf("[Executor] Task %d already taken by another process", task.ID)
			return nil
		}

		log.Printf("[Executor] Task %d status updated to running", task.ID)
		task.Status = model.ReleaseTaskStatusRunning
	}

	// 创建Runner并执行
	runner := NewRunner(e.db, e.agentClient, task)
	if err := runner.Run(); err != nil {
		log.Printf("[Executor] Task %d execution failed: %v", task.ID, err)
		return err
	}

	log.Printf("[Executor] Task %d execution completed", task.ID)
	return nil
}

// RunLoop 循环执行
func (e *Executor) RunLoop(ctx context.Context) {
	log.Printf("[Executor] Starting executor loop (interval=%v)", e.interval)

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	// 立即执行一次
	if err := e.RunOnce(); err != nil {
		log.Printf("[Executor] Initial run failed: %v", err)
	}

	// 循环执行
	for {
		select {
		case <-ctx.Done():
			log.Println("[Executor] Executor loop stopped")
			return
		case <-ticker.C:
			if err := e.RunOnce(); err != nil {
				log.Printf("[Executor] Run failed: %v", err)
			}
		}
	}
}
