package dto

// SignalItem 表示信号列表接口的响应项。
type SignalItem struct {
	StrategyCode          string `json:"strategy_code"`
	StrategyVersion       string `json:"strategy_version"`
	TSCode                string `json:"ts_code"`
	TradeDate             string `json:"trade_date"`
	SignalType            string `json:"signal_type"`
	SignalStrength        string `json:"signal_strength"`
	SignalLevel           string `json:"signal_level"`
	BuyPriceRef           string `json:"buy_price_ref"`
	StopLossRef           string `json:"stop_loss_ref"`
	TakeProfitRef         string `json:"take_profit_ref"`
	InvalidationCondition string `json:"invalidation_condition"`
	Reason                string `json:"reason"`
}
