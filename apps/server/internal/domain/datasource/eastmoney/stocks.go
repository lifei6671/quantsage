package eastmoney

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

// ListStocks 读取 A 股股票基础信息。
func (s *Source) ListStocks(ctx context.Context) ([]datasource.StockBasic, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	body, err := s.fallbackClient.GetQuote(ctx, stockListPath, buildStockListQuery())
	if err != nil {
		return nil, err
	}

	var response stockListAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, datasourceUnavailable(fmt.Errorf("decode eastmoney stock list response: %w", err))
	}
	if response.RC != 0 {
		return nil, datasourceUnavailable(
			fmt.Errorf("eastmoney stock list rc=%d message=%q", response.RC, strings.TrimSpace(response.Message)),
		)
	}

	items := make([]datasource.StockBasic, 0, len(response.Data.Diff))
	for _, rawItem := range response.Data.Diff {
		if strings.TrimSpace(rawItem.Symbol) == "" || strings.TrimSpace(rawItem.Name) == "" {
			continue
		}

		tsCode, exchange, market, err := mapStockIdentity(rawItem.Market, rawItem.Symbol)
		if err != nil {
			return nil, datasourceUnavailable(fmt.Errorf("map eastmoney stock %s: %w", rawItem.Symbol, err))
		}

		items = append(items, datasource.StockBasic{
			TSCode:   tsCode,
			Symbol:   strings.TrimSpace(rawItem.Symbol),
			Name:     strings.TrimSpace(rawItem.Name),
			Industry: strings.TrimSpace(rawItem.Industry),
			Market:   market,
			Exchange: exchange,
			ListDate: parseOptionalStockListDate(rawItem.ListDate),
			Source:   sourceName,
		})
	}

	return items, nil
}

func buildStockListQuery() url.Values {
	return url.Values{
		"pn":     []string{"1"},
		"pz":     []string{"10000"},
		"po":     []string{"1"},
		"np":     []string{"1"},
		"fltt":   []string{"2"},
		"invt":   []string{"2"},
		"fid":    []string{"f3"},
		"fs":     []string{defaultStockListFS},
		"fields": []string{defaultStockListFields},
	}
}
