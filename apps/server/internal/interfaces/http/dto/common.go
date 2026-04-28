package dto

// PageRequest 定义通用分页请求参数。
type PageRequest struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// PageResponse 定义通用分页响应结构。
type PageResponse[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
