package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
)

// TaskSpec 定义单个调度任务。
type TaskSpec struct {
	JobName string
	Spec    string
}

// Scheduler 封装 cron 和任务注册信息。
type Scheduler struct {
	logger *slog.Logger
	runner jobdomain.Runner
	cron   *cron.Cron
	now    func() time.Time

	mu    sync.RWMutex
	tasks map[string]string
}

// New 创建任务调度器。
func New(logger *slog.Logger, runner jobdomain.Runner) *Scheduler {
	return &Scheduler{
		logger: logger,
		runner: runner,
		cron:   cron.New(cron.WithSeconds()),
		now:    time.Now,
		tasks:  make(map[string]string),
	}
}

// Register 注册任务；若提供 cron 表达式则同步加入调度器。
func (s *Scheduler) Register(task TaskSpec) error {
	if task.JobName == "" {
		return fmt.Errorf("register scheduler task: job name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.JobName]; exists {
		return fmt.Errorf("register scheduler task %s: already exists", task.JobName)
	}
	s.tasks[task.JobName] = task.Spec

	if task.Spec == "" {
		return nil
	}

	jobName := task.JobName
	spec := task.Spec
	if _, err := s.cron.AddFunc(spec, func() {
		if runErr := s.runner.Run(context.Background(), jobName, schedulerBizDate(s.now())); runErr != nil {
			s.logger.Error("定时任务执行失败", "job_name", jobName, "error", runErr)
			return
		}
		s.logger.Info("定时任务执行完成", "job_name", jobName)
	}); err != nil {
		delete(s.tasks, task.JobName)
		return fmt.Errorf("register cron task %s: %w", task.JobName, err)
	}

	return nil
}

// Start 启动调度器。
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop 停止调度器。
func (s *Scheduler) Stop() context.Context {
	return s.cron.Stop()
}

// TaskNames 返回已注册任务列表。
func (s *Scheduler) TaskNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.tasks))
	for name := range s.tasks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func schedulerBizDate(now time.Time) time.Time {
	utcNow := now.UTC()
	return time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
}
