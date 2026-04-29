package job

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

// Runner 定义任务执行器契约。
type Runner interface {
	Run(ctx context.Context, jobName string, bizDate time.Time) error
}

// JobFunc 表示单个任务的执行函数。
type JobFunc func(ctx context.Context, bizDate time.Time) error

type registeredJob struct {
	fn           JobFunc
	dependencies []string
}

// RegisterOption 定义任务注册选项。
type RegisterOption func(*registeredJob)

// WithDependencies 声明当前任务在同一业务日下的前置依赖。
func WithDependencies(jobNames ...string) RegisterOption {
	return func(job *registeredJob) {
		job.dependencies = append(job.dependencies, jobNames...)
	}
}

// Registry 维护任务名称到执行函数的映射。
type Registry struct {
	mu     sync.RWMutex
	jobs   map[string]registeredJob
	locks  map[string]*sync.Mutex
	active map[string]map[string]struct{}
	done   map[string]map[string]struct{}
	waits  map[string][]string
}

// NewRegistry 创建任务注册表。
func NewRegistry() *Registry {
	return &Registry{
		jobs:   make(map[string]registeredJob),
		locks:  make(map[string]*sync.Mutex),
		active: make(map[string]map[string]struct{}),
		done:   make(map[string]map[string]struct{}),
		waits:  make(map[string][]string),
	}
}

// Register 注册单个任务。
func (r *Registry) Register(jobName string, fn JobFunc, options ...RegisterOption) error {
	if jobName == "" {
		return apperror.New(apperror.CodeBadRequest, errors.New("job name is required"))
	}
	if fn == nil {
		return apperror.New(apperror.CodeBadRequest, fmt.Errorf("job %s handler is nil", jobName))
	}

	job := registeredJob{fn: fn}
	for _, option := range options {
		if option != nil {
			option(&job)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[jobName]; exists {
		return apperror.New(apperror.CodeBadRequest, fmt.Errorf("job %s is already registered", jobName))
	}
	r.jobs[jobName] = job
	r.locks[jobName] = &sync.Mutex{}
	for _, dependency := range job.dependencies {
		r.waits[dependency] = append(r.waits[dependency], jobName)
	}
	return nil
}

// Run 执行指定名称的任务。
func (r *Registry) Run(ctx context.Context, jobName string, bizDate time.Time) error {
	r.mu.RLock()
	job, ok := r.jobs[jobName]
	lock := r.locks[jobName]
	r.mu.RUnlock()
	if !ok {
		return apperror.New(apperror.CodeNotFound, fmt.Errorf("job %s not found", jobName))
	}

	lock.Lock()
	defer lock.Unlock()

	if err := r.reserveRun(jobName, bizDate, job.dependencies); err != nil {
		return err
	}
	succeeded := false
	defer func() {
		r.finishRun(jobName, bizDate, succeeded)
	}()

	if err := job.fn(ctx, bizDate); err != nil {
		return fmt.Errorf("run job %s: %w", jobName, err)
	}
	succeeded = true

	return nil
}

// JobNames 返回当前已注册的任务名列表。
func (r *Registry) JobNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.jobs))
	for name := range r.jobs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MarkCompleted 将指定业务日的任务状态标记为已成功完成。
func (r *Registry) MarkCompleted(jobName string, bizDate time.Time) error {
	targetDate := runnerDateKey(bizDate)
	if targetDate == "" {
		return apperror.New(apperror.CodeBadRequest, errors.New("biz date is required"))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[jobName]; !exists {
		return apperror.New(apperror.CodeNotFound, fmt.Errorf("job %s not found", jobName))
	}
	if r.done[jobName] == nil {
		r.done[jobName] = make(map[string]struct{})
	}
	r.done[jobName][targetDate] = struct{}{}
	return nil
}

func (r *Registry) reserveRun(jobName string, bizDate time.Time, dependencies []string) error {
	targetDate := runnerDateKey(bizDate)

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, dependency := range dependencies {
		if activeDate, running := r.activeOnOrBeforeLocked(dependency, targetDate); running {
			return apperror.New(apperror.CodeJobRunning, fmt.Errorf("job %s is blocked by running dependency %s for %s", jobName, dependency, activeDate))
		}
		if !r.isDoneLocked(dependency, targetDate) {
			return apperror.New(apperror.CodeValidationFailed, fmt.Errorf("job %s prerequisite %s is not completed for %s", jobName, dependency, targetDate))
		}
	}

	if dependent, activeDate := r.findActiveDependentLocked(jobName, targetDate); dependent != "" {
		return apperror.New(apperror.CodeJobRunning, fmt.Errorf("job %s is blocked by running dependent %s for %s", jobName, dependent, activeDate))
	}

	if r.active[jobName] == nil {
		r.active[jobName] = make(map[string]struct{})
	}
	r.active[jobName][targetDate] = struct{}{}

	return nil
}

func (r *Registry) finishRun(jobName string, bizDate time.Time, succeeded bool) {
	targetDate := runnerDateKey(bizDate)

	r.mu.Lock()
	defer r.mu.Unlock()

	if activeDates := r.active[jobName]; activeDates != nil {
		delete(activeDates, targetDate)
		if len(activeDates) == 0 {
			delete(r.active, jobName)
		}
	}

	if succeeded {
		if r.done[jobName] == nil {
			r.done[jobName] = make(map[string]struct{})
		}
		r.done[jobName][targetDate] = struct{}{}
		r.invalidateDependentsLocked(jobName, targetDate)
	}
}

func (r *Registry) findActiveDependentLocked(jobName, targetDate string) (string, string) {
	visited := map[string]struct{}{jobName: {}}
	queue := append([]string(nil), r.waits[jobName]...)

	for len(queue) > 0 {
		dependent := queue[0]
		queue = queue[1:]
		if _, seen := visited[dependent]; seen {
			continue
		}
		visited[dependent] = struct{}{}

		if activeDate, running := r.activeOnOrAfterLocked(dependent, targetDate); running {
			return dependent, activeDate
		}

		queue = append(queue, r.waits[dependent]...)
	}

	return "", ""
}

func (r *Registry) activeOnOrBeforeLocked(jobName, targetDate string) (string, bool) {
	activeDates := r.active[jobName]
	for activeDate := range activeDates {
		if activeDate <= targetDate {
			return activeDate, true
		}
	}

	return "", false
}

func (r *Registry) activeOnOrAfterLocked(jobName, targetDate string) (string, bool) {
	activeDates := r.active[jobName]
	for activeDate := range activeDates {
		if activeDate >= targetDate {
			return activeDate, true
		}
	}

	return "", false
}

func (r *Registry) invalidateDependentsLocked(jobName, targetDate string) {
	visited := map[string]struct{}{jobName: {}}
	queue := append([]string(nil), r.waits[jobName]...)

	for len(queue) > 0 {
		dependent := queue[0]
		queue = queue[1:]
		if _, seen := visited[dependent]; seen {
			continue
		}
		visited[dependent] = struct{}{}

		if doneDates := r.done[dependent]; doneDates != nil {
			for doneDate := range doneDates {
				if doneDate >= targetDate {
					delete(doneDates, doneDate)
				}
			}
			if len(doneDates) == 0 {
				delete(r.done, dependent)
			}
		}

		queue = append(queue, r.waits[dependent]...)
	}
}

func (r *Registry) isDoneLocked(jobName, targetDate string) bool {
	doneDates := r.done[jobName]
	if doneDates == nil {
		return false
	}
	_, completed := doneDates[targetDate]
	return completed
}

func runnerDateKey(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	utcValue := value.UTC()
	return time.Date(utcValue.Year(), utcValue.Month(), utcValue.Day(), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}
