package finscope

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/lifei6671/quantsage/apps/server/internal/domain/datasource"
	"github.com/lifei6671/quantsage/apps/server/internal/infra/browserfetch"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestSourceImplementsDatasourceInterface(t *testing.T) {
	t.Parallel()

	var _ datasource.Source = (*Source)(nil)
}

func TestNewReturnsSource(t *testing.T) {
	t.Parallel()

	source := New(nil)
	if source == nil {
		t.Fatal("New(nil) = nil, want non-nil source")
	}
	if source.browser != nil {
		t.Fatal("source.browser != nil, want nil browser for nil input")
	}
	if source.cfg.ObserveIdleTimeout != defaultObserveIdleTimeout {
		t.Fatalf("ObserveIdleTimeout = %s, want %s", source.cfg.ObserveIdleTimeout, defaultObserveIdleTimeout)
	}
	if source.cfg.ConstituentScrollPause != defaultConstituentScrollPause {
		t.Fatalf("ConstituentScrollPause = %s, want %s", source.cfg.ConstituentScrollPause, defaultConstituentScrollPause)
	}
}

func TestSourceMethodsReturnNotImplemented(t *testing.T) {
	t.Parallel()

	source := New(nil)
	ctx := context.Background()

	if _, err := source.ListTradeCalendar(ctx, "SSE", time.Time{}, time.Time{}); apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListTradeCalendar() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
	if _, err := source.ListDailyBars(ctx, time.Time{}, time.Time{}); apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListDailyBars() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
	if _, err := source.ListKLines(ctx, datasource.KLineQuery{}); apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListKLines() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
	if _, err := source.StreamKLines(ctx, datasource.KLineQuery{}); apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("StreamKLines() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
}

func TestListStocksRejectsMissingBrowser(t *testing.T) {
	t.Parallel()

	source := New(nil)
	_, err := source.ListStocks(context.Background())
	if apperror.CodeOf(err) != apperror.CodeDatasourceUnavailable {
		t.Fatalf("ListStocks() code = %d, want %d", apperror.CodeOf(err), apperror.CodeDatasourceUnavailable)
	}
}

func TestParseStockBasicsFromConstituentResponseMapsItems(t *testing.T) {
	t.Parallel()

	items, err := parseStockBasicsFromConstituentResponse([]byte(`{
		"ResultCode": 0,
		"Result": {
			"list": {
				"body": [
					{"code": "601398", "exchange": "SH", "name": "工商银行", "market": "ab", "financeType": "stock"},
					{"code": "300750", "exchange": "SZ", "name": "宁德时代", "market": "ab", "financeType": "stock"}
				]
			}
		}
	}`))
	if err != nil {
		t.Fatalf("parseStockBasicsFromConstituentResponse() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want %d", len(items), 2)
	}
	if items[0].TSCode != "601398.SH" || items[0].Exchange != "SSE" || items[0].Market != "MAIN" {
		t.Fatalf("items[0] = %+v, want SH stock mapped to SSE/MAIN", items[0])
	}
	if items[1].TSCode != "300750.SZ" || items[1].Exchange != "SZSE" || items[1].Market != "GEM" {
		t.Fatalf("items[1] = %+v, want SZ stock mapped to SZSE/GEM", items[1])
	}
}

func TestParseStockBasicsFromConstituentResponseMaps301PrefixToGEM(t *testing.T) {
	t.Parallel()

	items, err := parseStockBasicsFromConstituentResponse([]byte(`{
		"ResultCode": 0,
		"Result": {
			"list": {
				"body": [
					{"code": "301183", "exchange": "SZ", "name": "东田微", "market": "ab", "financeType": "stock"}
				]
			}
		}
	}`))
	if err != nil {
		t.Fatalf("parseStockBasicsFromConstituentResponse() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	if items[0].Market != "GEM" {
		t.Fatalf("items[0].Market = %q, want %q", items[0].Market, "GEM")
	}
}

func TestListStocksUsesSHIndexConstituentPage(t *testing.T) {
	t.Parallel()

	watcher := &stubWatcher{
		capturedBodies: [][]byte{
			[]byte(`{
				"ResultCode": 0,
				"Result": {
					"list": {
						"body": [
							{"code": "601398", "exchange": "SH", "name": "工商银行", "market": "ab", "financeType": "stock"},
							{"code": "600519", "exchange": "SH", "name": "贵州茅台", "market": "ab", "financeType": "stock"}
						]
					}
				}
			}`),
			[]byte(`{
				"ResultCode": 0,
				"Result": {
					"list": {
						"body": [
							{"code": "600519", "exchange": "SH", "name": "贵州茅台", "market": "ab", "financeType": "stock"},
							{"code": "688981", "exchange": "SH", "name": "中芯国际", "market": "ab", "financeType": "stock"}
						]
					}
				}
			}`),
		},
	}
	source := New(watcher)

	items, err := source.ListStocks(context.Background())
	if err != nil {
		t.Fatalf("ListStocks() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want %d", len(items), 3)
	}
	if items[0].TSCode != "601398.SH" || items[0].Name != "工商银行" {
		t.Fatalf("items[0] = %+v, want first parsed stock", items[0])
	}
	if items[2].TSCode != "688981.SH" || items[2].Market != "STAR" {
		t.Fatalf("items[2] = %+v, want STAR market mapping", items[2])
	}
}

type stubWatcher struct {
	capturedBodies [][]byte
	err            error
}

func (s *stubWatcher) CaptureConstituentBodies(_ context.Context, _ Config, _ constituentQuery) ([][]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.capturedBodies, nil
}

func (s *stubWatcher) FetchCookieHeader(context.Context, string) (string, error) {
	return "", nil
}

func (s *stubWatcher) Run(context.Context, string, ...browserfetch.RunOption) error {
	return nil
}

func (s *stubWatcher) RunWithActions(context.Context, string, []chromedp.Action, []chromedp.Action, ...browserfetch.RunOption) error {
	return nil
}

func (s *stubWatcher) ObserveResponses(context.Context, string, ...browserfetch.ObserveOption) (*browserfetch.ResponseStream, error) {
	return nil, errors.New("not implemented in stub")
}

func (s *stubWatcher) Close(context.Context) error {
	return nil
}

func (s *stubWatcher) InvalidateCookies() {
}

func TestWatchConstituentPageBodiesPropagatesWatcherError(t *testing.T) {
	t.Parallel()

	source := New(&stubWatcher{err: errors.New("boom")})
	_, err := source.watchConstituentPageBodies(context.Background(), defaultSHIndexConstituentQuery())
	if err == nil {
		t.Fatal("watchConstituentPageBodies() error = nil, want non-nil")
	}
}

func TestConstituentQueryMatchesResponseURL(t *testing.T) {
	t.Parallel()

	query := defaultSHIndexConstituentQuery()
	if !query.matches(query.pageURL(defaultConstituentAPIURL), defaultConstituentAPIURL) {
		t.Fatal("query.matches() = false, want true for fully matched URL")
	}
	if query.matches("https://finance.pae.baidu.com/sapi/v1/constituents?market=ab&code=399001&financeType=index", defaultConstituentAPIURL) {
		t.Fatal("query.matches() = true, want false for different code")
	}
}

func TestBuildConstituentAfterReadyActionsUsesConfiguredObserveIdleTimeout(t *testing.T) {
	t.Parallel()

	source := New(nil, WithObserveIdleTimeout(37*time.Second))

	originalWait := waitForCapturedConstituentResponsesFunc
	originalStable := waitForCapturedConstituentResponsesStableFunc
	t.Cleanup(func() {
		waitForCapturedConstituentResponsesFunc = originalWait
		waitForCapturedConstituentResponsesStableFunc = originalStable
	})

	var initialTimeout time.Duration
	var stableTimeout time.Duration
	waitForCapturedConstituentResponsesFunc = func(_ context.Context, minCount int, timeout time.Duration) error {
		if minCount != 1 {
			t.Fatalf("minCount = %d, want %d", minCount, 1)
		}
		initialTimeout = timeout
		return nil
	}
	waitForCapturedConstituentResponsesStableFunc = func(_ context.Context, timeout time.Duration) error {
		stableTimeout = timeout
		return nil
	}

	actions := source.buildConstituentAfterReadyActions(nil)
	if len(actions) != 4 {
		t.Fatalf("len(actions) = %d, want %d", len(actions), 4)
	}
	if err := actions[0].Do(context.Background()); err != nil {
		t.Fatalf("actions[0].Do() error = %v", err)
	}
	if err := actions[2].Do(context.Background()); err != nil {
		t.Fatalf("actions[2].Do() error = %v", err)
	}
	if initialTimeout != 37*time.Second {
		t.Fatalf("initialTimeout = %s, want %s", initialTimeout, 37*time.Second)
	}
	if stableTimeout != 37*time.Second {
		t.Fatalf("stableTimeout = %s, want %s", stableTimeout, 37*time.Second)
	}
}

func TestWaitForCapturedConstituentResponsesToStabilizeWaitsForPendingRequests(t *testing.T) {
	originalPollInterval := constituentCapturePollInterval
	originalReadStatus := readCapturedConstituentStatusFunc
	t.Cleanup(func() {
		constituentCapturePollInterval = originalPollInterval
		readCapturedConstituentStatusFunc = originalReadStatus
	})

	constituentCapturePollInterval = time.Millisecond
	callCount := 0
	readCapturedConstituentStatusFunc = func(context.Context) (constituentCaptureStatus, error) {
		callCount++
		if callCount == 1 {
			return constituentCaptureStatus{
				Pending:               1,
				LastActivityUnixMilli: time.Now().Add(-time.Second).UnixMilli(),
			}, nil
		}
		return constituentCaptureStatus{
			Pending:               0,
			LastActivityUnixMilli: time.Now().Add(-time.Second).UnixMilli(),
		}, nil
	}

	if err := waitForCapturedConstituentResponsesToStabilize(context.Background(), 10*time.Millisecond); err != nil {
		t.Fatalf("waitForCapturedConstituentResponsesToStabilize() error = %v", err)
	}
	if callCount < 2 {
		t.Fatalf("callCount = %d, want at least 2 reads before stabilization", callCount)
	}
}

func TestListStocksRealSmoke(t *testing.T) {
	if os.Getenv("FINSCOPE_STOCKS_SMOKE") != "1" {
		t.Skip("set FINSCOPE_STOCKS_SMOKE=1 to run real browser smoke")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	runner := browserfetch.New(realSmokeBrowserConfig())
	defer func() {
		if err := runner.Close(context.Background()); err != nil {
			t.Fatalf("runner.Close() error = %v", err)
		}
	}()

	source := New(runner)
	items, err := source.ListStocks(ctx)
	if err != nil {
		t.Fatalf("ListStocks() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("len(items) = 0, want at least one stock from real smoke")
	}
	if items[0].TSCode == "" || items[0].Name == "" || items[0].Source != sourceName {
		t.Fatalf("items[0] = %+v, want populated first stock", items[0])
	}
}

func TestOpenSHIndexPageRealSmoke(t *testing.T) {
	if os.Getenv("FINSCOPE_STOCKS_SMOKE") != "1" {
		t.Skip("set FINSCOPE_STOCKS_SMOKE=1 to run real browser smoke")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	runner := browserfetch.New(realSmokeBrowserConfig())
	defer func() {
		_ = runner.Close(context.Background())
	}()

	err := runner.Run(
		ctx,
		defaultIndexPageURL,
		browserfetch.WithRunPrimaryPageTarget(true),
		browserfetch.WithRunRawPageNavigate(true),
	)
	if err != nil {
		t.Fatalf("runner.Run() error = %v", err)
	}
}

func realSmokeBrowserConfig() browserfetch.Config {
	cfg := browserfetch.Config{
		Timeout: 90 * time.Second,
	}

	if path := strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_BROWSER_PATH")); path != "" {
		cfg.BrowserPath = path
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_NO_SANDBOX")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_NO_SANDBOX")), "true") {
		cfg.NoSandbox = true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_HEADED")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_HEADED")), "true") {
		cfg.Headless = new(false)
	}

	return cfg
}
