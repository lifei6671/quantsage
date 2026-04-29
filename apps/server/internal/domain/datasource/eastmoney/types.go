package eastmoney

import (
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/consts"
)

const (
	defaultEndpoint              = "https://push2his.eastmoney.com"
	defaultQuoteEndpoint         = "https://push2his.eastmoney.com"
	defaultTimeout               = 30 * time.Second
	defaultUserAgentMode         = "stable"
	responseBodyLimit            = 32 << 20
	defaultDailyFetchConcurrency = 8
	sourceName                   = consts.DatasourceEastMoney
	historyKLinePath             = "/api/qt/stock/kline/get"
	stockListPath                = "/api/qt/clist/get"
	defaultKLineFields1          = "f1,f2,f3,f4,f5,f6"
	defaultKLineFields2          = "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61"
	defaultStockListFS           = "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23"
	defaultStockListFields       = "f12,f13,f14,f100,f26"
	eastMoneyDateLayout          = "2006-01-02"
	eastMoneyMinuteLayout        = "2006-01-02 15:04"
	eastMoneyDateCompactLayout   = "20060102"
)

// Interval 定义东财 K 线查询支持的周期枚举。
type Interval string

const (
	Interval1Min    Interval = "1m"
	Interval5Min    Interval = "5m"
	Interval15Min   Interval = "15m"
	Interval30Min   Interval = "30m"
	Interval60Min   Interval = "60m"
	IntervalDay     Interval = "1d"
	IntervalWeek    Interval = "1w"
	IntervalMonth   Interval = "1mo"
	IntervalQuarter Interval = "1q"
	IntervalYear    Interval = "1y"
)

// AdjustType 定义东财复权参数。
type AdjustType string

const (
	AdjustNone AdjustType = "none"
	AdjustQFQ  AdjustType = "qfq"
	AdjustHFQ  AdjustType = "hfq"
)

// FetchMode 定义东财抓取时的请求模式。
type FetchMode string

const (
	FetchModeHTTP     FetchMode = "http"
	FetchModeAuto     FetchMode = "auto"
	FetchModeChromedp FetchMode = "chromedp"
)

// FallbackConfig 定义东财浏览器回退抓取的最小配置。
type FallbackConfig struct {
	Mode         FetchMode
	QuotePageURL string
}

// KLineAPIResponse 预留给后续任务 3 解析东财 K 线接口响应。
type KLineAPIResponse struct {
	RC      int          `json:"rc"`
	RT      int          `json:"rt"`
	SVR     int          `json:"svr"`
	LT      int          `json:"lt"`
	Full    int          `json:"full"`
	DLT     int          `json:"dlt"`
	Message string       `json:"message"`
	Data    KLineAPIData `json:"data"`
}

// KLineAPIData 表示东财 K 线接口 data 负载。
type KLineAPIData struct {
	Code    string   `json:"code"`
	Market  int      `json:"market"`
	Name    string   `json:"name"`
	Decimal int      `json:"decimal"`
	KLines  []string `json:"klines"`
}

// StockListAPIResponse 表示东财全市场列表接口响应。
type stockListAPIResponse struct {
	RC      int              `json:"rc"`
	Message string           `json:"message"`
	Data    stockListAPIData `json:"data"`
}

type stockListAPIData struct {
	Total int             `json:"total"`
	Diff  []stockListItem `json:"diff"`
}

type stockListItem struct {
	Symbol   string `json:"f12"`
	Market   int    `json:"f13"`
	Name     string `json:"f14"`
	Industry string `json:"f100"`
	ListDate string `json:"f26"`
}
