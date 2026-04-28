package job

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

const (
	defaultJobPage     = 1
	defaultJobPageSize = 20
	maxJobPageSize     = 100
)

// JobRun 表示一次任务执行记录。
type JobRun struct {
	ID           int64
	JobName      string
	BizDate      time.Time
	Status       string
	StartedAt    time.Time
	FinishedAt   time.Time
	ErrorCode    int
	ErrorMessage string
	CreatedAt    time.Time
}

// QueryParams 定义任务记录查询条件。
type QueryParams struct {
	JobName  string
	BizDate  time.Time
	Page     int
	PageSize int
}

// QueryResult 定义任务记录分页结果。
type QueryResult struct {
	Items    []JobRun
	Page     int
	PageSize int
}

// QueryService 定义任务记录查询服务契约。
type QueryService interface {
	ListJobRuns(ctx context.Context, params QueryParams) (QueryResult, error)
}

// JobRunReader 定义任务记录读取接口。
type JobRunReader interface {
	ListJobRuns(ctx context.Context, params QueryParams) ([]JobRun, error)
}

type queryService struct {
	reader JobRunReader
}

// NewQueryService 创建任务记录查询服务。
func NewQueryService(reader JobRunReader) QueryService {
	return &queryService{reader: reader}
}

// ListJobRuns 查询任务记录。
func (s *queryService) ListJobRuns(ctx context.Context, params QueryParams) (QueryResult, error) {
	if s.reader == nil {
		return QueryResult{}, apperror.New(apperror.CodeInternal, errors.New("job query service is not configured"))
	}

	page, pageSize := sanitizeJobPage(params.Page, params.PageSize)
	items, err := s.reader.ListJobRuns(ctx, QueryParams{
		JobName:  params.JobName,
		BizDate:  params.BizDate,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return QueryResult{}, apperror.New(apperror.CodeDatabaseError, fmt.Errorf("list job runs: %w", err))
	}

	return QueryResult{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func sanitizeJobPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultJobPage
	}
	if pageSize <= 0 {
		pageSize = defaultJobPageSize
	}
	if pageSize > maxJobPageSize {
		pageSize = maxJobPageSize
	}

	return page, pageSize
}
