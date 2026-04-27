package apperror

var messages = map[int]struct {
	Errmsg string
	Toast  string
}{
	CodeOK:                    {"", ""},
	CodeBadRequest:            {"bad request", "请求参数不正确"},
	CodeUnauthorized:          {"unauthorized", "请先登录"},
	CodeForbidden:             {"forbidden", "没有操作权限"},
	CodeNotFound:              {"not found", "数据不存在"},
	CodeValidationFailed:      {"validation failed", "提交内容不符合要求"},
	CodeDatasourceUnavailable: {"datasource unavailable", "数据源暂时不可用，请稍后重试"},
	CodeJobRunning:            {"job already running", "任务正在执行，请稍后查看结果"},
	CodeDatabaseError:         {"database error", "数据服务异常，请稍后重试"},
	CodeInternal:              {"internal error", "系统异常，请稍后重试"},
}
