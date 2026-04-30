package sina

import (
	"sort"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
)

type collector struct {
	query datasource.KLineQuery
	items map[string]datasource.KLine
}

func newCollector(query datasource.KLineQuery) *collector {
	return &collector{
		query: query,
		items: make(map[string]datasource.KLine),
	}
}

func (c *collector) Add(items []datasource.KLine) {
	if c == nil {
		return
	}
	for _, item := range items {
		if item.TSCode != c.query.TSCode {
			continue
		}
		key := item.TradeTime.UTC().Format(timeKeyLayout)
		existing, ok := c.items[key]
		if !ok || shouldReplaceSnapshot(existing, item) {
			c.items[key] = item
		}
	}
}

func (c *collector) Finalize() []datasource.KLine {
	if c == nil {
		return nil
	}

	result := make([]datasource.KLine, 0, len(c.items))
	for _, item := range c.items {
		if c.query.Limit > 0 {
			if !item.TradeTime.After(c.query.EndTime) {
				result = append(result, item)
			}
			continue
		}
		if item.TradeTime.Before(c.query.StartTime) || item.TradeTime.After(c.query.EndTime) {
			continue
		}
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TradeTime.Before(result[j].TradeTime)
	})

	return datasource.TrimKLinesByLimit(result, c.query.Limit)
}

const timeKeyLayout = "2006-01-02T15:04:05.000000000Z07:00"

func shouldReplaceSnapshot(existing, incoming datasource.KLine) bool {
	if existing.TradeTime.Before(incoming.TradeTime) {
		return true
	}
	if incoming.TradeTime.Before(existing.TradeTime) {
		return false
	}

	existingEmpty := isEmptySnapshot(existing)
	incomingEmpty := isEmptySnapshot(incoming)
	if existingEmpty != incomingEmpty {
		return !incomingEmpty
	}

	// 相同时间戳的多次推送默认以后到快照为准，避免把初始化阶段的旧 candle 留到最终结果里。
	return true
}

func isEmptySnapshot(item datasource.KLine) bool {
	return item.Open.IsZero() &&
		item.High.IsZero() &&
		item.Low.IsZero() &&
		item.Close.IsZero() &&
		item.Vol.IsZero() &&
		item.Amount.IsZero()
}
