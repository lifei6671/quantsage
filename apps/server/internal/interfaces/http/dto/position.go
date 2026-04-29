package dto

// PositionItem 定义持仓响应项。
type PositionItem struct {
	ID           int64  `json:"id"`
	TSCode       string `json:"ts_code"`
	PositionDate string `json:"position_date"`
	Quantity     string `json:"quantity"`
	CostPrice    string `json:"cost_price"`
	Note         string `json:"note"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// CreatePositionRequest 定义新增持仓请求。
type CreatePositionRequest struct {
	TSCode       string `json:"ts_code"`
	PositionDate string `json:"position_date"`
	Quantity     string `json:"quantity"`
	CostPrice    string `json:"cost_price"`
	Note         string `json:"note"`
}

// UpdatePositionRequest 定义更新持仓请求。
type UpdatePositionRequest struct {
	TSCode       string `json:"ts_code"`
	PositionDate string `json:"position_date"`
	Quantity     string `json:"quantity"`
	CostPrice    string `json:"cost_price"`
	Note         string `json:"note"`
}
