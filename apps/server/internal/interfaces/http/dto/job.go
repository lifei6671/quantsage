package dto

// RunJobRequest 定义手动触发任务请求。
type RunJobRequest struct {
	BizDate string `json:"biz_date"`
}

// RunJobResponse 定义手动触发任务响应。
type RunJobResponse struct {
	JobName string `json:"job_name"`
	Status  string `json:"status"`
}

// JobRunItem 定义任务记录接口响应项。
type JobRunItem struct {
	ID           int64  `json:"id"`
	JobName      string `json:"job_name"`
	BizDate      string `json:"biz_date"`
	Status       string `json:"status"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}
