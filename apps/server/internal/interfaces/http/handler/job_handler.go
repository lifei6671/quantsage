package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
	infraLog "github.com/lifei6671/quantsage/apps/server/internal/infra/log"
	"github.com/lifei6671/quantsage/apps/server/internal/interfaces/http/dto"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/response"
)

// JobHandler 提供任务相关 HTTP 接口。
type JobHandler struct {
	runner       jobdomain.Runner
	queryService jobdomain.QueryService
}

// NewJobHandler 创建任务接口处理器。
func NewJobHandler(runner jobdomain.Runner, queryService jobdomain.QueryService) *JobHandler {
	return &JobHandler{
		runner:       runner,
		queryService: queryService,
	}
}

// Run 手动触发指定任务。
func (h *JobHandler) Run(c *gin.Context) {
	if err := h.ensureRunner(); err != nil {
		response.Fail(c, err)
		return
	}

	jobName := c.Param("job_name")
	c.Request = c.Request.WithContext(infraLog.AddInfo(c.Request.Context(), infraLog.String("job_name", jobName)))

	var req dto.RunJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind run job request: %w", err)))
		return
	}

	bizDate, err := time.Parse(dateLayout, req.BizDate)
	if err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("parse biz_date: %w", err)))
		return
	}

	if err := h.runner.Run(c.Request.Context(), jobName, bizDate); err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.RunJobResponse{
		JobName: jobName,
		Status:  "queued",
	})
}

// List 返回任务执行记录列表。
func (h *JobHandler) List(c *gin.Context) {
	if err := h.ensureQueryService(); err != nil {
		response.Fail(c, err)
		return
	}

	var req dto.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("bind job list query: %w", err)))
		return
	}

	var bizDate time.Time
	bizDateText := c.Query("biz_date")
	if bizDateText != "" {
		parsedDate, err := time.Parse(dateLayout, bizDateText)
		if err != nil {
			response.Fail(c, apperror.New(apperror.CodeBadRequest, fmt.Errorf("parse biz_date: %w", err)))
			return
		}
		bizDate = parsedDate
	}

	result, err := h.queryService.ListJobRuns(c.Request.Context(), jobdomain.QueryParams{
		JobName:  c.Query("job_name"),
		BizDate:  bizDate,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}

	items := make([]dto.JobRunItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dto.JobRunItem{
			ID:           item.ID,
			JobName:      item.JobName,
			BizDate:      formatDate(item.BizDate),
			Status:       item.Status,
			StartedAt:    formatTimestamp(item.StartedAt),
			FinishedAt:   formatTimestamp(item.FinishedAt),
			ErrorCode:    item.ErrorCode,
			ErrorMessage: item.ErrorMessage,
		})
	}

	response.OK(c, dto.PageResponse[dto.JobRunItem]{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func (h *JobHandler) ensureRunner() error {
	if h.runner != nil {
		return nil
	}

	return apperror.New(apperror.CodeInternal, errors.New("job runner is not configured"))
}

func (h *JobHandler) ensureQueryService() error {
	if h.queryService != nil {
		return nil
	}

	return apperror.New(apperror.CodeInternal, errors.New("job query service is not configured"))
}

// formatDate 仅输出日期字段。
func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format(dateLayout)
}

// formatTimestamp 始终按 RFC3339 输出时间戳字段。
func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
