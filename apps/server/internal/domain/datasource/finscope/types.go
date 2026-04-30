package finscope

import (
	"net/url"
	"strings"
	"time"
)

const sourceName = "finscope"

const defaultObserveIdleTimeout = 5 * time.Second

const defaultConstituentPageSize = 20
const defaultConstituentScrollPause = 1200 * time.Millisecond
const defaultConstituentMaxScrollRounds = 24
const defaultConstituentStableScrollLimit = 3

const (
	defaultConstituentMarket     = "ab"
	defaultConstituentCode       = "000001"
	defaultConstituentSortKey    = "marketValue"
	defaultConstituentStyle      = "heatmap"
	defaultConstituentClientType = "pc"
	defaultFinanceTypeIndex      = "index"
	defaultIndexPageURL          = "https://finance.baidu.com/index/ab-000001?mainTab=%E6%88%90%E5%88%86%E8%82%A1"
	defaultConstituentAPIURL     = "https://finance.pae.baidu.com/sapi/v1/constituents"
)

// Config 定义 Finscope 数据源的最小内部配置。
type Config struct {
	BasePageURL                   string
	ConstituentAPIURL             string
	ObserveIdleTimeout            time.Duration
	ConstituentScrollPause        time.Duration
	ConstituentMaxScrollRounds    int
	ConstituentStableScrollRounds int
}

// Option 用于覆盖 Finscope 数据源内部配置。
type Option func(*Config)

// WithBasePageURL 覆盖后续页面监听使用的基础页面地址。
func WithBasePageURL(pageURL string) Option {
	return func(cfg *Config) {
		cfg.BasePageURL = pageURL
	}
}

// WithObserveIdleTimeout 覆盖页面响应监听的空闲收口时间。
func WithObserveIdleTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.ObserveIdleTimeout = timeout
	}
}

func defaultConfig() Config {
	return Config{
		BasePageURL:                   defaultIndexPageURL,
		ConstituentAPIURL:             defaultConstituentAPIURL,
		ObserveIdleTimeout:            defaultObserveIdleTimeout,
		ConstituentScrollPause:        defaultConstituentScrollPause,
		ConstituentMaxScrollRounds:    defaultConstituentMaxScrollRounds,
		ConstituentStableScrollRounds: defaultConstituentStableScrollLimit,
	}
}

type constituentQuery struct {
	Market        string
	Code          string
	FinanceType   string
	SortKey       string
	Style         string
	PageSize      int
	PageNumber    int
	FinClientType string
}

func defaultSHIndexConstituentQuery() constituentQuery {
	return constituentQuery{
		Market:        defaultConstituentMarket,
		Code:          defaultConstituentCode,
		FinanceType:   defaultFinanceTypeIndex,
		SortKey:       defaultConstituentSortKey,
		Style:         defaultConstituentStyle,
		PageSize:      defaultConstituentPageSize,
		PageNumber:    0,
		FinClientType: defaultConstituentClientType,
	}
}

func (q constituentQuery) pageURL(baseURL string) string {
	params := url.Values{}
	params.Set("market", strings.TrimSpace(q.Market))
	params.Set("code", strings.TrimSpace(q.Code))
	params.Set("financeType", strings.TrimSpace(q.FinanceType))
	params.Set("sortKey", strings.TrimSpace(q.SortKey))
	params.Set("style", strings.TrimSpace(q.Style))
	params.Set("rn", intToString(q.PageSize))
	params.Set("pn", intToString(q.PageNumber))
	params.Set("finClientType", strings.TrimSpace(q.FinClientType))

	return strings.TrimSpace(baseURL) + "?" + params.Encode()
}

func (q constituentQuery) matches(rawURL string, baseURL string) bool {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}

	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsedURL.Scheme, base.Scheme) || !strings.EqualFold(parsedURL.Host, base.Host) || parsedURL.Path != base.Path {
		return false
	}

	params := parsedURL.Query()
	if strings.TrimSpace(params.Get("market")) != strings.TrimSpace(q.Market) {
		return false
	}
	if strings.TrimSpace(params.Get("code")) != strings.TrimSpace(q.Code) {
		return false
	}
	if strings.TrimSpace(params.Get("financeType")) != strings.TrimSpace(q.FinanceType) {
		return false
	}
	if strings.TrimSpace(params.Get("sortKey")) != strings.TrimSpace(q.SortKey) {
		return false
	}
	if strings.TrimSpace(params.Get("style")) != strings.TrimSpace(q.Style) {
		return false
	}
	if pageSize := strings.TrimSpace(params.Get("rn")); pageSize != "" && pageSize != intToString(q.PageSize) {
		return false
	}
	if clientType := strings.TrimSpace(params.Get("finClientType")); clientType != "" && clientType != strings.TrimSpace(q.FinClientType) {
		return false
	}

	return true
}
