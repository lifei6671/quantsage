package dto

// StockItem 表示股票列表与详情接口的响应项。
type StockItem struct {
	TSCode   string `json:"ts_code"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Industry string `json:"industry"`
	Exchange string `json:"exchange"`
	IsActive bool   `json:"is_active"`
}

// DailyBarItem 表示股票日线接口的响应项。
type DailyBarItem struct {
	TSCode    string `json:"ts_code"`
	TradeDate string `json:"trade_date"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	PctChg    string `json:"pct_chg"`
	Vol       string `json:"vol"`
	Amount    string `json:"amount"`
}
