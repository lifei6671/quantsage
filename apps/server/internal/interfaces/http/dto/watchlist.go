package dto

// WatchlistGroupItem 定义自选分组响应项。
type WatchlistGroupItem struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortOrder int32  `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateWatchlistGroupRequest 定义新增自选分组请求。
type CreateWatchlistGroupRequest struct {
	Name      string `json:"name"`
	SortOrder int32  `json:"sort_order"`
}

// UpdateWatchlistGroupRequest 定义更新自选分组请求。
type UpdateWatchlistGroupRequest struct {
	Name      string `json:"name"`
	SortOrder int32  `json:"sort_order"`
}

// WatchlistItem 定义自选股条目响应。
type WatchlistItem struct {
	ID        int64  `json:"id"`
	GroupID   int64  `json:"group_id"`
	TSCode    string `json:"ts_code"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

// CreateWatchlistItemRequest 定义新增分组内股票请求。
type CreateWatchlistItemRequest struct {
	TSCode string `json:"ts_code"`
	Note   string `json:"note"`
}
