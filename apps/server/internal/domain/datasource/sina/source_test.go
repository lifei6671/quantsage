package sina

import (
	"context"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/infra/browserfetch"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestSourceListKLinesConsumesWatcherStream(t *testing.T) {
	t.Parallel()

	responses := make(chan browserfetch.ResponseStreamItem, 2)
	responses <- browserfetch.ResponseStreamItem{
		Body: []byte(`callback_1([{"day":"2026-04-29 09:35:00","open":"10.2","high":"10.5","low":"10.1","close":"10.3","volume":"1200"}])`),
	}
	responses <- browserfetch.ResponseStreamItem{
		Body: []byte(`callback_2([{"day":"2026-04-29 09:30:00","open":"10.1","high":"10.4","low":"10.0","close":"10.2","volume":"1000"}])`),
	}
	close(responses)

	watcher := &stubWatcher{
		stream: responses,
		done:   closedDone(nil),
	}
	source := newSourceWithWatcher(watcher)

	items, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("ListKLines() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if !items[0].TradeTime.Before(items[1].TradeTime) {
		t.Fatalf("items not sorted ascending: %+v", items)
	}
	if watcher.closeCalls != 1 {
		t.Fatalf("watcher.closeCalls = %d, want %d", watcher.closeCalls, 1)
	}
}

func TestSourceStreamKLinesEmitsParsedBatch(t *testing.T) {
	t.Parallel()

	responses := make(chan browserfetch.ResponseStreamItem, 1)
	responses <- browserfetch.ResponseStreamItem{
		Body: []byte(`callback_1([{"day":"2026-04-29 09:30:00","open":"10.1","high":"10.4","low":"10.0","close":"10.2","volume":"1000"}])`),
	}
	close(responses)

	source := newSourceWithWatcher(&stubWatcher{
		stream: responses,
		done:   closedDone(nil),
	})

	stream, err := source.StreamKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("StreamKLines() error = %v", err)
	}

	item, ok := <-stream
	if !ok {
		t.Fatal("stream closed before first item")
	}
	if item.Err != nil {
		t.Fatalf("item.Err = %v, want nil", item.Err)
	}
	if len(item.Items) != 1 {
		t.Fatalf("len(item.Items) = %d, want %d", len(item.Items), 1)
	}
	if got := item.Items[0].TradeTime; !got.Equal(time.Date(2026, 4, 29, 1, 30, 0, 0, time.UTC)) {
		t.Fatalf("item.Items[0].TradeTime = %s, want %s", got, time.Date(2026, 4, 29, 1, 30, 0, 0, time.UTC))
	}
}

func TestSourceUnsupportedBatchMethodsReturnUnavailable(t *testing.T) {
	t.Parallel()

	source := New(nil)
	if _, err := source.ListStocks(context.Background()); apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListStocks() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
	if _, err := source.ListTradeCalendar(context.Background(), "", time.Time{}, time.Time{}); apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListTradeCalendar() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
	if _, err := source.ListDailyBars(context.Background(), time.Time{}, time.Time{}); apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListDailyBars() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
}

func TestSourceRejectsUnsupportedInterval(t *testing.T) {
	t.Parallel()

	source := newSourceWithWatcher(&stubWatcher{})
	_, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.IntervalMonth,
		Limit:    1,
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
	if apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListKLines() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
}

func TestSourceRejectsHistoricalTimeRangeQuery(t *testing.T) {
	t.Parallel()

	source := newSourceWithWatcher(&stubWatcher{})
	_, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:    "000001.SZ",
		Interval:  datasource.Interval5Min,
		StartTime: time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
	if apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListKLines() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
}

func TestSourceRejectsExplicitEndTimeInLatestQuery(t *testing.T) {
	t.Parallel()

	source := newSourceWithWatcher(&stubWatcher{})
	_, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		EndTime:  time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC),
		Limit:    10,
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
	if apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListKLines() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
}

func TestSourceRejectsOversizedLatestLimit(t *testing.T) {
	t.Parallel()

	source := newSourceWithWatcher(&stubWatcher{})
	_, err := source.ListKLines(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    maxKLineLimit + 1,
	})
	if err == nil {
		t.Fatal("ListKLines() error = nil, want non-nil")
	}
	if apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListKLines() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
}

func TestWatchKLineResponsesUsesDirectKLineRequest(t *testing.T) {
	t.Parallel()

	watcher := &stubWatcher{
		stream: make(chan browserfetch.ResponseStreamItem),
		done:   closedDone(nil),
	}
	source := newSourceWithWatcher(watcher)

	stream, err := source.watchKLineResponses(context.Background(), datasource.KLineQuery{
		TSCode:   "000001.SZ",
		Interval: datasource.Interval5Min,
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("watchKLineResponses() error = %v", err)
	}
	stream.Close()

	if !strings.Contains(watcher.pageURL, "CN_MarketDataService.getKLineData") {
		t.Fatalf("pageURL = %q, want direct kline endpoint", watcher.pageURL)
	}
	parsedURL, err := url.Parse(watcher.pageURL)
	if err != nil {
		t.Fatalf("url.Parse(pageURL) error = %v", err)
	}
	params := parsedURL.Query()
	if got := params.Get("symbol"); got != "sz000001" {
		t.Fatalf("symbol = %q, want %q", got, "sz000001")
	}
	if got := params.Get("scale"); got != "5" {
		t.Fatalf("scale = %q, want %q", got, "5")
	}
	if got := params.Get("ma"); got != "no" {
		t.Fatalf("ma = %q, want %q", got, "no")
	}
	if got := params.Get("datalen"); got != "1" {
		t.Fatalf("datalen = %q, want %q", got, "1")
	}
	if resourceTypes := resourceTypesFromObserveOptions(t, watcher.opts...); len(resourceTypes) != 0 {
		t.Fatalf("resourceTypes = %v, want empty to allow JSONP script responses", resourceTypes)
	}
}

func TestParseSinaTradeTimeUsesShanghaiForMinuteBars(t *testing.T) {
	t.Parallel()

	got, err := parseSinaTradeTime(datasource.Interval5Min, "2026-04-29 09:30:00")
	if err != nil {
		t.Fatalf("parseSinaTradeTime() error = %v", err)
	}

	want := time.Date(2026, 4, 29, 1, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parseSinaTradeTime() = %s, want %s", got, want)
	}
}

func TestParseJSONPResponseAcceptsEmptyArrayPayload(t *testing.T) {
	t.Parallel()

	items, err := parseJSONPResponse([]byte(`callback_1([])`))
	if err != nil {
		t.Fatalf("parseJSONPResponse() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want %d", len(items), 0)
	}
}

type stubWatcher struct {
	stream     <-chan browserfetch.ResponseStreamItem
	done       <-chan error
	err        error
	closeCalls int
	pageURL    string
	opts       []browserfetch.ObserveOption
}

func (s *stubWatcher) ObserveResponses(_ context.Context, pageURL string, opts ...browserfetch.ObserveOption) (*browserfetch.ResponseStream, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.pageURL = pageURL
	s.opts = append([]browserfetch.ObserveOption(nil), opts...)

	done := s.done
	if done == nil {
		done = closedDone(nil)
	}

	return &browserfetch.ResponseStream{
		Responses: s.stream,
		Done:      done,
		Close: func() {
			s.closeCalls++
		},
	}, nil
}

func closedDone(err error) <-chan error {
	done := make(chan error, 1)
	if err != nil {
		done <- err
	}
	close(done)

	return done
}

func resourceTypesFromObserveOptions(t *testing.T, opts ...browserfetch.ObserveOption) map[string]struct{} {
	t.Helper()

	if len(opts) == 0 {
		return nil
	}

	optionsPtr := newObserveOptionsValue(t, opts[0])
	for _, opt := range opts {
		reflect.ValueOf(opt).Call([]reflect.Value{optionsPtr})
	}

	resourceTypes := optionsPtr.Elem().FieldByName("resourceTypes")
	if !resourceTypes.IsValid() {
		t.Fatal("observeOptions.resourceTypes field not found")
	}

	result := make(map[string]struct{}, resourceTypes.Len())
	for _, key := range resourceTypes.MapKeys() {
		result[key.String()] = struct{}{}
	}

	return result
}

func newObserveOptionsValue(t *testing.T, opt browserfetch.ObserveOption) reflect.Value {
	t.Helper()

	optValue := reflect.ValueOf(opt)
	if !optValue.IsValid() || optValue.Kind() != reflect.Func {
		t.Fatal("observe option must be a function")
	}
	optType := optValue.Type()
	if optType.NumIn() != 1 {
		t.Fatalf("observe option input count = %d, want %d", optType.NumIn(), 1)
	}

	optionsType := optType.In(0)
	if optionsType.Kind() != reflect.Pointer {
		t.Fatalf("observe option param kind = %s, want pointer", optionsType.Kind())
	}

	return reflect.New(optionsType.Elem())
}
