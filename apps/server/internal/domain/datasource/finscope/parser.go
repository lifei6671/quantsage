package finscope

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type constituentResponse struct {
	ResultCode int `json:"ResultCode"`
	Result     struct {
		List struct {
			Body []constituentItem `json:"body"`
		} `json:"list"`
	} `json:"Result"`
}

type constituentItem struct {
	Code        string `json:"code"`
	Exchange    string `json:"exchange"`
	FinanceType string `json:"financeType"`
	Market      string `json:"market"`
	Name        string `json:"name"`
}

func parseStockBasicsFromConstituentResponse(body []byte) ([]datasource.StockBasic, error) {
	var response constituentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, apperror.New(apperror.CodeDatasourceUnavailable, fmt.Errorf("decode finscope constituent response: %w", err))
	}
	if response.ResultCode != 0 {
		return nil, apperror.New(
			apperror.CodeDatasourceUnavailable,
			fmt.Errorf("finscope constituent response code %d", response.ResultCode),
		)
	}

	result := make([]datasource.StockBasic, 0, len(response.Result.List.Body))
	for _, item := range response.Result.List.Body {
		stock, err := mapConstituentItemToStockBasic(item)
		if err != nil {
			return nil, err
		}
		result = append(result, stock)
	}

	return result, nil
}

func mapConstituentItemToStockBasic(item constituentItem) (datasource.StockBasic, error) {
	tsCode, exchange, market, err := buildTSCodeAndMarket(item.Code, item.Exchange)
	if err != nil {
		return datasource.StockBasic{}, apperror.New(
			apperror.CodeDatasourceUnavailable,
			fmt.Errorf("map finscope constituent %s.%s: %w", item.Code, item.Exchange, err),
		)
	}

	return datasource.StockBasic{
		TSCode:   tsCode,
		Symbol:   strings.TrimSpace(item.Code),
		Name:     strings.TrimSpace(item.Name),
		Market:   market,
		Exchange: exchange,
		Source:   sourceName,
	}, nil
}

func buildTSCodeAndMarket(code string, exchange string) (string, string, string, error) {
	symbol := strings.TrimSpace(code)
	marketCode := strings.ToUpper(strings.TrimSpace(exchange))
	if symbol == "" {
		return "", "", "", fmt.Errorf("code is required")
	}
	if _, err := strconv.Atoi(symbol); err != nil {
		return "", "", "", fmt.Errorf("code %q must be numeric", code)
	}

	switch marketCode {
	case "SH":
		return symbol + ".SH", "SSE", marketForSH(symbol), nil
	case "SZ":
		return symbol + ".SZ", "SZSE", marketForSZ(symbol), nil
	case "BJ":
		return symbol + ".BJ", "BSE", "BSE", nil
	default:
		return "", "", "", fmt.Errorf("unsupported exchange %q", exchange)
	}
}

func marketForSH(symbol string) string {
	if strings.HasPrefix(symbol, "688") {
		return "STAR"
	}

	return "MAIN"
}

func marketForSZ(symbol string) string {
	if strings.HasPrefix(symbol, "300") || strings.HasPrefix(symbol, "301") {
		return "GEM"
	}

	return "MAIN"
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
