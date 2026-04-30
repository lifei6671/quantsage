package browserfetch

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func TestNormalizeConfigDefaults(t *testing.T) {
	t.Parallel()

	got := normalizeConfig(Config{})

	if got.BrowserPath != "" {
		t.Fatalf("BrowserPath = %q, want empty", got.BrowserPath)
	}
	if !got.Headless {
		t.Fatal("Headless = false, want true")
	}
	if got.UserAgentMode != UserAgentModeDefault {
		t.Fatalf("UserAgentMode = %q, want %q", got.UserAgentMode, UserAgentModeDefault)
	}
	if got.Timeout != defaultBrowserTimeout {
		t.Fatalf("Timeout = %s, want %s", got.Timeout, defaultBrowserTimeout)
	}
	if got.CookieCacheTTL != defaultCookieCacheTTL {
		t.Fatalf("CookieCacheTTL = %s, want %s", got.CookieCacheTTL, defaultCookieCacheTTL)
	}
	if got.BrowserCount != defaultBrowserCount {
		t.Fatalf("BrowserCount = %d, want %d", got.BrowserCount, defaultBrowserCount)
	}
	if got.TabsPerBrowser != defaultMaxConcurrentTabs {
		t.Fatalf("TabsPerBrowser = %d, want %d", got.TabsPerBrowser, defaultMaxConcurrentTabs)
	}
	if got.RecycleAfterTabs != defaultRecycleAfterTabs {
		t.Fatalf("RecycleAfterTabs = %d, want %d", got.RecycleAfterTabs, defaultRecycleAfterTabs)
	}
	if got.MaxConcurrentTabs != defaultMaxConcurrentTabs {
		t.Fatalf("MaxConcurrentTabs = %d, want %d", got.MaxConcurrentTabs, defaultMaxConcurrentTabs)
	}
	if got.WaitReadySelector != defaultWaitReadySelector {
		t.Fatalf("WaitReadySelector = %q, want %q", got.WaitReadySelector, defaultWaitReadySelector)
	}
	if got.AcceptLanguage != defaultAcceptLanguage {
		t.Fatalf("AcceptLanguage = %q, want %q", got.AcceptLanguage, defaultAcceptLanguage)
	}
}

func TestCookieCacheUsesNormalizedURL(t *testing.T) {
	t.Parallel()

	cache := newCookieCache()
	cfg := normalizeConfig(Config{
		BrowserPath:   " /usr/bin/google-chrome ",
		UserAgentMode: UserAgentModeCustom,
		UserAgent:     " TestUA ",
	})

	key := cookieCacheKey(cfg, "https://EXAMPLE.com/path/?b=2&a=1#frag")
	cache.set(key, "sid=1", time.Now().Add(time.Hour))

	got, ok := cache.get(cookieCacheKey(cfg, "https://example.com/path?a=1&b=2"), time.Now())
	if !ok {
		t.Fatal("cache miss, want hit")
	}
	if got != "sid=1" {
		t.Fatalf("header = %q, want %q", got, "sid=1")
	}
}

func TestInvalidateCookiesClearsCache(t *testing.T) {
	t.Parallel()

	fetchCalls := 0
	r := &runner{
		cfg:   normalizeConfig(Config{}),
		cache: newCookieCache(),
		now:   time.Now,
		fetchCookieHeader: func(context.Context, Config, string) (string, error) {
			fetchCalls++
			return "sid=1", nil
		},
	}

	ctx := context.Background()
	if _, err := r.FetchCookieHeader(ctx, "https://example.com/page"); err != nil {
		t.Fatalf("FetchCookieHeader() error = %v", err)
	}
	r.InvalidateCookies()
	if _, err := r.FetchCookieHeader(ctx, "https://example.com/page"); err != nil {
		t.Fatalf("FetchCookieHeader() after invalidate error = %v", err)
	}
	if fetchCalls != 2 {
		t.Fatalf("fetchCalls = %d, want %d", fetchCalls, 2)
	}
}

func TestFetchCookieHeaderUsesCache(t *testing.T) {
	t.Parallel()

	fetchCalls := 0
	r := &runner{
		cfg:   normalizeConfig(Config{CookieCacheTTL: time.Hour}),
		cache: newCookieCache(),
		now:   time.Now,
		fetchCookieHeader: func(context.Context, Config, string) (string, error) {
			fetchCalls++
			return "sid=1", nil
		},
	}

	ctx := context.Background()
	got1, err := r.FetchCookieHeader(ctx, "https://example.com/page?a=1&b=2")
	if err != nil {
		t.Fatalf("FetchCookieHeader() error = %v", err)
	}
	got2, err := r.FetchCookieHeader(ctx, "https://EXAMPLE.com/page?b=2&a=1")
	if err != nil {
		t.Fatalf("FetchCookieHeader() second call error = %v", err)
	}

	if got1 != "sid=1" || got2 != "sid=1" {
		t.Fatalf("headers = %q, %q, want sid=1", got1, got2)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetchCalls = %d, want %d", fetchCalls, 1)
	}
}

func TestFetchCookieHeaderWritesCacheAfterSuccess(t *testing.T) {
	t.Parallel()

	fetchCalls := 0
	r := &runner{
		cfg:   normalizeConfig(Config{CookieCacheTTL: time.Hour}),
		cache: newCookieCache(),
		now:   time.Now,
		fetchCookieHeader: func(context.Context, Config, string) (string, error) {
			fetchCalls++
			return "sid=1", nil
		},
	}

	ctx := context.Background()
	pageURL := "https://example.com/page?b=2&a=1"
	got, err := r.FetchCookieHeader(ctx, pageURL)
	if err != nil {
		t.Fatalf("FetchCookieHeader() error = %v", err)
	}
	if got != "sid=1" {
		t.Fatalf("header = %q, want %q", got, "sid=1")
	}

	key := cookieCacheKey(r.cfg, pageURL)
	cached, ok := r.cache.get(key, time.Now())
	if !ok {
		t.Fatal("cache miss after successful fetch, want hit")
	}
	if cached != "sid=1" {
		t.Fatalf("cached header = %q, want %q", cached, "sid=1")
	}
}

func TestFetchCookieHeaderDoesNotCacheEmptyHeader(t *testing.T) {
	t.Parallel()

	fetchCalls := 0
	r := &runner{
		cfg:   normalizeConfig(Config{CookieCacheTTL: time.Hour}),
		cache: newCookieCache(),
		now:   time.Now,
		fetchCookieHeader: func(context.Context, Config, string) (string, error) {
			fetchCalls++
			if fetchCalls == 1 {
				return "", nil
			}
			return "sid=1", nil
		},
	}

	ctx := context.Background()
	got1, err := r.FetchCookieHeader(ctx, "https://example.com/page")
	if err != nil {
		t.Fatalf("FetchCookieHeader() first call error = %v", err)
	}
	if got1 != "" {
		t.Fatalf("first header = %q, want empty", got1)
	}

	got2, err := r.FetchCookieHeader(ctx, "https://example.com/page")
	if err != nil {
		t.Fatalf("FetchCookieHeader() second call error = %v", err)
	}
	if got2 != "sid=1" {
		t.Fatalf("second header = %q, want %q", got2, "sid=1")
	}
	if fetchCalls != 2 {
		t.Fatalf("fetchCalls = %d, want %d", fetchCalls, 2)
	}
}

func TestFetchCookieHeaderRefetchesAfterCacheExpiry(t *testing.T) {
	t.Parallel()

	fetchCalls := 0
	now := time.Unix(1_700_000_000, 0)
	r := &runner{
		cfg:   normalizeConfig(Config{CookieCacheTTL: time.Second}),
		cache: newCookieCache(),
		now: func() time.Time {
			return now
		},
		fetchCookieHeader: func(context.Context, Config, string) (string, error) {
			fetchCalls++
			return "sid=1", nil
		},
	}

	ctx := context.Background()
	if _, err := r.FetchCookieHeader(ctx, "https://example.com/page"); err != nil {
		t.Fatalf("FetchCookieHeader() first call error = %v", err)
	}
	now = now.Add(2 * time.Second)
	if _, err := r.FetchCookieHeader(ctx, "https://example.com/page"); err != nil {
		t.Fatalf("FetchCookieHeader() second call error = %v", err)
	}

	if fetchCalls != 2 {
		t.Fatalf("fetchCalls = %d, want %d", fetchCalls, 2)
	}
}

func TestObserveResponsesEmitsMatchedBodiesAndStopsOnIdle(t *testing.T) {
	stubObserveHooks(t)

	var listener func(any)
	listenerReady := make(chan struct{}, 1)
	listenTargetFunc = func(_ context.Context, fn func(any)) {
		listener = fn
		listenerReady <- struct{}{}
	}
	getResponseBodyFunc = func(_ context.Context, requestID network.RequestID) ([]byte, error) {
		if requestID != network.RequestID("req-1") {
			t.Fatalf("requestID = %q, want %q", requestID, "req-1")
		}
		return []byte(`{"ok":true}`), nil
	}

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://quote.example.com/stock/000001",
		WithObserveIdleTimeout(10*time.Millisecond),
		WithObserveURLContains("/stock/kline"),
		WithObserveResourceTypes(network.ResourceTypeXHR, network.ResourceTypeFetch),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	select {
	case <-listenerReady:
	case <-time.After(time.Second):
		t.Fatal("listener was not registered")
	}
	if listener == nil {
		t.Fatal("listener = nil, want non-nil after registration")
	}

	listener(&network.EventResponseReceived{
		RequestID: network.RequestID("req-1"),
		Type:      network.ResourceTypeXHR,
		Response: &network.Response{
			URL:      "https://api.example.com/stock/kline",
			Status:   200,
			MimeType: "application/json",
		},
	})
	listener(&network.EventLoadingFinished{RequestID: network.RequestID("req-1")})

	select {
	case item := <-stream.Responses:
		if got := string(item.Body); got != `{"ok":true}` {
			t.Fatalf("item.Body = %q, want %q", got, `{"ok":true}`)
		}
		if item.URL != "https://api.example.com/stock/kline" {
			t.Fatalf("item.URL = %q, want %q", item.URL, "https://api.example.com/stock/kline")
		}
		if item.Err != nil {
			t.Fatalf("item.Err = %v, want nil", item.Err)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive observed response item")
	}

	select {
	case err := <-stream.Done:
		if err != nil {
			t.Fatalf("stream.Done = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after idle timeout")
	}
}

func TestObserveResponsesSkipsNonMatchingResponses(t *testing.T) {
	stubObserveHooks(t)

	var listener func(any)
	listenerReady := make(chan struct{}, 1)
	listenTargetFunc = func(_ context.Context, fn func(any)) {
		listener = fn
		listenerReady <- struct{}{}
	}
	getResponseBodyFunc = func(context.Context, network.RequestID) ([]byte, error) {
		t.Fatal("getResponseBodyFunc called for non-matching response")
		return nil, nil
	}

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://quote.example.com/stock/000001",
		WithObserveIdleTimeout(10*time.Millisecond),
		WithObserveURLContains("/stock/kline"),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	select {
	case <-listenerReady:
	case <-time.After(time.Second):
		t.Fatal("listener was not registered")
	}
	if listener == nil {
		t.Fatal("listener = nil, want non-nil after registration")
	}

	listener(&network.EventResponseReceived{
		RequestID: network.RequestID("req-2"),
		Type:      network.ResourceTypeXHR,
		Response: &network.Response{
			URL:      "https://api.example.com/other",
			Status:   200,
			MimeType: "application/json",
		},
	})
	listener(&network.EventLoadingFinished{RequestID: network.RequestID("req-2")})

	select {
	case item, ok := <-stream.Responses:
		if ok {
			t.Fatalf("unexpected response item: %+v", item)
		}
	case err := <-stream.Done:
		if !errors.Is(err, errObserveNoMatchingResponse) {
			t.Fatalf("stream.Done = %v, want %v", err, errObserveNoMatchingResponse)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after idle timeout")
	}
}

func TestObserveResponsesDoesNotTimeOutBeforePageReady(t *testing.T) {
	stubObserveHooks(t)

	var listener func(any)
	listenerReady := make(chan struct{}, 1)
	listenTargetFunc = func(_ context.Context, fn func(any)) {
		listener = fn
		listenerReady <- struct{}{}
	}
	runBlocked := make(chan struct{})
	releaseRun := make(chan struct{})
	runActionsFunc = func(context.Context, ...chromedp.Action) error {
		close(runBlocked)
		<-releaseRun
		return nil
	}
	getResponseBodyFunc = func(_ context.Context, requestID network.RequestID) ([]byte, error) {
		if requestID != network.RequestID("req-delayed") {
			t.Fatalf("requestID = %q, want %q", requestID, "req-delayed")
		}
		return []byte(`{"ok":"delayed"}`), nil
	}

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://quote.example.com/stock/000001",
		WithObserveIdleTimeout(10*time.Millisecond),
		WithObserveURLContains("/stock/kline"),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	select {
	case <-listenerReady:
	case <-time.After(time.Second):
		t.Fatal("listener was not registered")
	}
	select {
	case <-runBlocked:
	case <-time.After(time.Second):
		t.Fatal("runActions was not blocked")
	}

	select {
	case err := <-stream.Done:
		t.Fatalf("stream.Done = %v, want no timeout before page ready", err)
	case <-time.After(30 * time.Millisecond):
	}

	listener(&network.EventResponseReceived{
		RequestID: network.RequestID("req-delayed"),
		Type:      network.ResourceTypeScript,
		Response: &network.Response{
			URL:      "https://api.example.com/stock/kline",
			Status:   200,
			MimeType: "application/javascript",
		},
	})
	listener(&network.EventLoadingFinished{RequestID: network.RequestID("req-delayed")})
	close(releaseRun)

	select {
	case item := <-stream.Responses:
		if got := string(item.Body); got != `{"ok":"delayed"}` {
			t.Fatalf("item.Body = %q, want %q", got, `{"ok":"delayed"}`)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive delayed observed response item")
	}

	select {
	case err := <-stream.Done:
		if err != nil {
			t.Fatalf("stream.Done = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after idle timeout once page became ready")
	}
}

func TestObserveResponsesExecutesObserveActionsBeforeIdleStarts(t *testing.T) {
	stubObserveHooks(t)

	actionRan := make(chan struct{}, 1)
	runActionsFunc = func(ctx context.Context, actions ...chromedp.Action) error {
		if len(actions) == 0 {
			t.Fatal("len(actions) = 0, want observe actions to be present")
		}
		if err := actions[len(actions)-1].Do(ctx); err != nil {
			return err
		}
		return nil
	}

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://quote.example.com/stock/000001",
		WithObserveIdleTimeout(10*time.Millisecond),
		WithObserveActions(chromedp.ActionFunc(func(context.Context) error {
			actionRan <- struct{}{}
			return nil
		})),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	select {
	case <-actionRan:
	case <-time.After(time.Second):
		t.Fatal("observe action was not executed")
	}

	select {
	case err := <-stream.Done:
		if !errors.Is(err, errObserveNoMatchingResponse) {
			t.Fatalf("stream.Done = %v, want %v", err, errObserveNoMatchingResponse)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after idle timeout")
	}
}

func TestObserveResponsesWaitsForBodyWorkersBeforeClosingStreamOnRunError(t *testing.T) {
	stubObserveHooks(t)

	var listener func(any)
	listenerReady := make(chan struct{}, 1)
	listenTargetFunc = func(_ context.Context, fn func(any)) {
		listener = fn
		listenerReady <- struct{}{}
	}

	bodyStarted := make(chan struct{}, 1)
	releaseBody := make(chan struct{})
	getResponseBodyFunc = func(_ context.Context, requestID network.RequestID) ([]byte, error) {
		if requestID != network.RequestID("req-run-error") {
			t.Fatalf("requestID = %q, want %q", requestID, "req-run-error")
		}
		bodyStarted <- struct{}{}
		<-releaseBody
		return []byte(`{"ok":"late"}`), nil
	}
	runActionsFunc = func(context.Context, ...chromedp.Action) error {
		select {
		case <-listenerReady:
		case <-time.After(time.Second):
			return errors.New("listener was not registered")
		}

		listener(&network.EventResponseReceived{
			RequestID: network.RequestID("req-run-error"),
			Type:      network.ResourceTypeXHR,
			Response: &network.Response{
				URL:      "https://api.example.com/stock/kline",
				Status:   200,
				MimeType: "application/json",
			},
		})
		listener(&network.EventLoadingFinished{RequestID: network.RequestID("req-run-error")})

		return errors.New("run failed")
	}

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://quote.example.com/stock/000001",
		WithObserveIdleTimeout(time.Second),
		WithObserveURLContains("/stock/kline"),
		WithObserveResourceTypes(network.ResourceTypeXHR),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	select {
	case <-bodyStarted:
	case <-time.After(time.Second):
		t.Fatal("response body worker did not start")
	}
	close(releaseBody)

	select {
	case item := <-stream.Responses:
		if got := string(item.Body); got != `{"ok":"late"}` {
			t.Fatalf("item.Body = %q, want %q", got, `{"ok":"late"}`)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive observed response item before stream closed")
	}

	select {
	case err := <-stream.Done:
		if err == nil || !strings.Contains(err.Error(), "run failed") {
			t.Fatalf("stream.Done = %v, want contains %q", err, "run failed")
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after run error")
	}
}

func TestObserveResponsesWaitsForMatchingRequestToFinishBeforeIdleTimeout(t *testing.T) {
	stubObserveHooks(t)

	var listener func(any)
	listenerReady := make(chan struct{}, 1)
	listenTargetFunc = func(_ context.Context, fn func(any)) {
		listener = fn
		listenerReady <- struct{}{}
	}
	runBlocked := make(chan struct{})
	releaseRun := make(chan struct{})
	runActionsFunc = func(context.Context, ...chromedp.Action) error {
		close(runBlocked)
		<-releaseRun
		return nil
	}

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://quote.example.com/stock/000001",
		WithObserveIdleTimeout(10*time.Millisecond),
		WithObserveURLContains("/stock/kline"),
		WithObserveResourceTypes(network.ResourceTypeXHR),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	select {
	case <-listenerReady:
	case <-time.After(time.Second):
		t.Fatal("listener was not registered")
	}
	select {
	case <-runBlocked:
	case <-time.After(time.Second):
		t.Fatal("runActions was not blocked")
	}

	listener(&network.EventResponseReceived{
		RequestID: network.RequestID("req-pending"),
		Type:      network.ResourceTypeXHR,
		Response: &network.Response{
			URL:      "https://api.example.com/stock/kline",
			Status:   200,
			MimeType: "application/json",
		},
	})
	close(releaseRun)

	select {
	case err := <-stream.Done:
		t.Fatalf("stream.Done = %v, want still waiting for matching request", err)
	case <-time.After(30 * time.Millisecond):
	}

	stream.Close()
	select {
	case <-stream.Done:
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after close")
	}
}

func TestRunOptionAppliesToRunConfig(t *testing.T) {
	originalRunPageFunc := runPageFunc
	t.Cleanup(func() {
		runPageFunc = originalRunPageFunc
	})

	var gotConfig normalizedConfig
	var gotPageURL string
	runPageFunc = func(ctx context.Context, r *runner, cfg normalizedConfig, pageURL string) error {
		gotConfig = cfg
		gotPageURL = pageURL
		return nil
	}

	r := &runner{cfg: normalizeConfig(Config{})}
	if err := r.Run(
		context.Background(),
		"https://example.com/page",
		WithRunHeadless(false),
		WithRunTimeout(42*time.Second),
		WithRunWaitReadySelector("#app"),
		WithRunPrimaryPageTarget(true),
		WithRunRawPageNavigate(true),
		WithRunDisableImages(true),
	); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if gotConfig.Headless {
		t.Fatal("Headless = true, want false")
	}
	if gotConfig.Timeout != 42*time.Second {
		t.Fatalf("Timeout = %s, want %s", gotConfig.Timeout, 42*time.Second)
	}
	if gotConfig.WaitReadySelector != "#app" {
		t.Fatalf("WaitReadySelector = %q, want %q", gotConfig.WaitReadySelector, "#app")
	}
	if !gotConfig.UsePrimaryPageTarget {
		t.Fatal("UsePrimaryPageTarget = false, want true")
	}
	if !gotConfig.UseRawPageNavigate {
		t.Fatal("UseRawPageNavigate = false, want true")
	}
	if !gotConfig.DisableImages {
		t.Fatal("DisableImages = false, want true")
	}
	if gotPageURL != "https://example.com/page" {
		t.Fatalf("pageURL = %q, want %q", gotPageURL, "https://example.com/page")
	}
}

func TestRunPrependsNetworkEnableByDefault(t *testing.T) {
	stubObserveHooks(t)

	actionCountCh := make(chan int, 1)
	runActionsFunc = func(_ context.Context, actions ...chromedp.Action) error {
		actionCountCh <- len(actions)
		return nil
	}

	r := New(Config{})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if err := r.Run(context.Background(), "https://example.com/page"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	select {
	case got := <-actionCountCh:
		if got != 3 {
			t.Fatalf("len(actions) = %d, want %d (network enable + navigate + wait ready)", got, 3)
		}
	case <-time.After(time.Second):
		t.Fatal("did not capture run action count")
	}
}

func TestRunRawPageNavigateSkipsWaitReadyAction(t *testing.T) {
	stubObserveHooks(t)

	actionCountCh := make(chan int, 1)
	runActionsFunc = func(_ context.Context, actions ...chromedp.Action) error {
		actionCountCh <- len(actions)
		return nil
	}

	r := New(Config{})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if err := r.Run(
		context.Background(),
		"https://example.com/page",
		WithRunRawPageNavigate(true),
	); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	select {
	case got := <-actionCountCh:
		if got != 2 {
			t.Fatalf("len(actions) = %d, want %d (network enable + raw page navigate)", got, 2)
		}
	case <-time.After(time.Second):
		t.Fatal("did not capture run action count")
	}
}

func TestAppendBrowserActionsPutsNetworkEnableBeforeBlockedURLs(t *testing.T) {
	t.Parallel()

	actions := appendBrowserActions(
		normalizeConfig(Config{DisableImages: true}),
		buildNavigateActions(normalizeConfig(Config{}), "https://example.com/page"),
		nil,
	)
	if len(actions) < 4 {
		t.Fatalf("len(actions) = %d, want at least 4", len(actions))
	}
	if _, ok := actions[0].(*network.EnableParams); !ok {
		t.Fatalf("actions[0] = %T, want *network.EnableParams", actions[0])
	}
	if _, ok := actions[1].(*network.SetBlockedURLsParams); !ok {
		t.Fatalf("actions[1] = %T, want *network.SetBlockedURLsParams", actions[1])
	}
}

func TestRunPrimaryPageTargetSkipsChildTabCreation(t *testing.T) {
	restore := stubBrowserProcessHooks(t)

	var newTabCalls int
	originalNewTab := newTabContextFunc
	newTabContextFunc = func(parent context.Context, opts ...chromedp.ContextOption) (context.Context, context.CancelFunc) {
		newTabCalls++
		return originalNewTab(parent, opts...)
	}
	t.Cleanup(func() {
		newTabContextFunc = originalNewTab
	})

	runCalls := 0
	restore.onRun = func() {
		runCalls++
	}

	r := New(Config{})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if err := r.Run(
		context.Background(),
		"https://example.com/page",
		WithRunPrimaryPageTarget(true),
	); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if newTabCalls != 0 {
		t.Fatalf("newTabCalls = %d, want 0", newTabCalls)
	}
	if runCalls != 1 {
		t.Fatalf("runCalls = %d, want %d (page run only in stubbed process)", runCalls, 1)
	}
}

func TestRunRestartsWorkerBrowserAfterInitialStartupFailure(t *testing.T) {
	restore := stubBrowserProcessHooks(t)

	startCalls := 0
	restore.onStart = func() {
		startCalls++
	}

	runCalls := 0
	runActionsFunc = func(_ context.Context, actions ...chromedp.Action) error {
		runCalls++
		if runCalls == 1 {
			return errors.New("startup failed")
		}
		return nil
	}

	r := New(Config{
		BrowserCount:   1,
		TabsPerBrowser: 1,
	})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if err := r.Run(context.Background(), "https://example.com/page"); err == nil {
		t.Fatal("first Run() error = nil, want startup failure")
	}
	if err := r.Run(context.Background(), "https://example.com/page"); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if startCalls != 2 {
		t.Fatalf("startCalls = %d, want %d after restart on first-run failure", startCalls, 2)
	}
}

func TestObserveResponsesPrependsNetworkEnable(t *testing.T) {
	stubObserveHooks(t)

	actionCountCh := make(chan int, 1)
	runActionsFunc = func(_ context.Context, actions ...chromedp.Action) error {
		actionCountCh <- len(actions)
		return nil
	}

	stream, err := New(Config{}).ObserveResponses(
		context.Background(),
		"https://example.com/page",
		WithObserveIdleTimeout(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("ObserveResponses() error = %v", err)
	}

	select {
	case got := <-actionCountCh:
		if got != 3 {
			t.Fatalf("len(actions) = %d, want %d (network enable + navigate + wait ready)", got, 3)
		}
	case <-time.After(time.Second):
		t.Fatal("did not capture observe action count")
	}

	select {
	case err := <-stream.Done:
		if !errors.Is(err, errObserveNoMatchingResponse) {
			t.Fatalf("stream.Done = %v, want %v", err, errObserveNoMatchingResponse)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after idle timeout")
	}
}

func TestRunWithActionsExecutesCustomActions(t *testing.T) {
	stubObserveHooks(t)

	beforeRan := false
	afterRan := false
	runActionsFunc = func(ctx context.Context, actions ...chromedp.Action) error {
		if len(actions) < 3 {
			t.Fatalf("len(actions) = %d, want at least 3", len(actions))
		}
		if err := actions[1].Do(ctx); err != nil {
			return err
		}
		if err := actions[len(actions)-1].Do(ctx); err != nil {
			return err
		}
		return nil
	}

	r := New(Config{WaitReadySelector: "body"})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	err := r.RunWithActions(
		context.Background(),
		"https://example.com/page",
		[]chromedp.Action{
			chromedp.ActionFunc(func(context.Context) error {
				beforeRan = true
				return nil
			}),
		},
		[]chromedp.Action{
			chromedp.ActionFunc(func(context.Context) error {
				afterRan = true
				return nil
			}),
		},
	)
	if err != nil {
		t.Fatalf("RunWithActions() error = %v", err)
	}
	if !beforeRan {
		t.Fatal("beforeRan = false, want true")
	}
	if !afterRan {
		t.Fatal("afterRan = false, want true")
	}
}

func TestRunWithActionsRawNavigateAddsReadyBarrierBeforeAfterReady(t *testing.T) {
	stubObserveHooks(t)

	actionCountCh := make(chan int, 1)
	runActionsFunc = func(_ context.Context, actions ...chromedp.Action) error {
		actionCountCh <- len(actions)
		return nil
	}

	r := New(Config{WaitReadySelector: "body"})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	err := r.RunWithActions(
		context.Background(),
		"https://example.com/page",
		nil,
		[]chromedp.Action{
			chromedp.ActionFunc(func(context.Context) error { return nil }),
		},
		WithRunRawPageNavigate(true),
	)
	if err != nil {
		t.Fatalf("RunWithActions() error = %v", err)
	}

	select {
	case got := <-actionCountCh:
		if got != 4 {
			t.Fatalf("len(actions) = %d, want %d (network enable + raw navigate + wait ready + afterReady)", got, 4)
		}
	case <-time.After(time.Second):
		t.Fatal("did not capture run action count")
	}
}

func TestRunAllowsStandaloneProcessLevelOverrides(t *testing.T) {
	restore := stubBrowserProcessHooks(t)

	var startCalls int
	var closeCalls int
	var runCalls int
	restore.onStart = func() {
		startCalls++
	}
	restore.onClose = func() {
		closeCalls++
	}
	restore.onRun = func() {
		runCalls++
	}

	r := New(Config{})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if err := r.Run(
		context.Background(),
		"https://example.com/page",
		WithRunHeadless(false),
	); err != nil {
		t.Fatalf("Run() with process-level override error = %v", err)
	}

	if startCalls != 1 {
		t.Fatalf("startCalls = %d, want %d", startCalls, 1)
	}
	if closeCalls != 1 {
		t.Fatalf("closeCalls = %d, want %d", closeCalls, 1)
	}
	if runCalls != 1 {
		t.Fatalf("runCalls = %d, want %d", runCalls, 1)
	}
}

func TestCloseClosesStandaloneProcessBeforeRunReturns(t *testing.T) {
	restore := stubBrowserProcessHooks(t)

	runEntered := make(chan struct{}, 1)
	releaseRun := make(chan struct{})
	closeObserved := make(chan struct{}, 1)
	var closeCalls int
	restore.onRun = func() {
		runEntered <- struct{}{}
		<-releaseRun
	}
	restore.onClose = func() {
		closeCalls++
		select {
		case closeObserved <- struct{}{}:
		default:
		}
	}

	r := New(Config{})
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- r.Run(
			context.Background(),
			"https://example.com/page",
			WithRunHeadless(false),
		)
	}()

	<-runEntered

	closeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	closeErr := r.Close(closeCtx)
	if !errors.Is(closeErr, context.DeadlineExceeded) {
		t.Fatalf("Close() error = %v, want deadline exceeded", closeErr)
	}

	select {
	case <-closeObserved:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("standalone process was not closed during Close()")
	}

	close(releaseRun)
	if err := <-runErrCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if closeCalls != 1 {
		t.Fatalf("closeCalls = %d, want %d", closeCalls, 1)
	}
}

func TestResolveUserAgentUsesFakeGeneratorBeforeFallback(t *testing.T) {
	t.Parallel()

	got := resolveUserAgent(normalizeConfig(Config{
		UserAgentMode: UserAgentModeFake,
		UserAgent:     "fallback",
	}), func(normalizedConfig) (string, error) {
		return "generated", nil
	})

	if got != "generated" {
		t.Fatalf("userAgent = %q, want generated", got)
	}
}

func TestResolveUserAgentFallsBackWhenFakeGeneratorFails(t *testing.T) {
	t.Parallel()

	got := resolveUserAgent(normalizeConfig(Config{
		UserAgentMode: UserAgentModeFake,
		UserAgent:     "fallback",
	}), func(normalizedConfig) (string, error) {
		return "", errors.New("boom")
	})

	if got != "fallback" {
		t.Fatalf("userAgent = %q, want fallback", got)
	}
}

func TestCloseIsIdempotentBeforeBrowserStarts(t *testing.T) {
	t.Parallel()

	r := New(Config{})
	if err := r.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := r.Close(context.Background()); err != nil {
		t.Fatalf("Close() second call error = %v", err)
	}
}

func TestRunnerRecyclesBrowserAfterTabThreshold(t *testing.T) {
	restore := stubBrowserProcessHooks(t)

	var startCalls int
	var closeCalls int
	restore.onStart = func() {
		startCalls++
	}
	restore.onClose = func() {
		closeCalls++
	}

	r := New(Config{
		BrowserCount:     1,
		TabsPerBrowser:   1,
		RecycleAfterTabs: 2,
	})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	for index := 0; index < 2; index++ {
		if err := r.Run(context.Background(), "https://example.com/page"); err != nil {
			t.Fatalf("Run(%d) error = %v", index, err)
		}
	}
	if startCalls != 1 {
		t.Fatalf("startCalls after recycle threshold = %d, want 1", startCalls)
	}
	if closeCalls != 1 {
		t.Fatalf("closeCalls after recycle threshold = %d, want 1", closeCalls)
	}

	if err := r.Run(context.Background(), "https://example.com/page"); err != nil {
		t.Fatalf("Run() after recycle error = %v", err)
	}
	if startCalls != 2 {
		t.Fatalf("startCalls after next run = %d, want 2", startCalls)
	}
}

func TestRunnerPoolConcurrencyIsBrowserCountTimesTabsPerBrowser(t *testing.T) {
	restore := stubBrowserProcessHooks(t)

	var startCalls int
	var active int
	var maxActive int
	runEntered := make(chan struct{}, 4)
	releaseRuns := make(chan struct{})
	var activeMu sync.Mutex
	restore.onStart = func() {
		startCalls++
	}
	restore.onRun = func() {
		activeMu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		activeMu.Unlock()
		runEntered <- struct{}{}
		<-releaseRuns
		activeMu.Lock()
		active--
		activeMu.Unlock()
	}

	r := New(Config{
		BrowserCount:     2,
		TabsPerBrowser:   2,
		RecycleAfterTabs: 100,
	})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	errCh := make(chan error, 4)
	for index := 0; index < 4; index++ {
		go func() {
			errCh <- r.Run(context.Background(), "https://example.com/page")
		}()
	}
	for index := 0; index < 4; index++ {
		<-runEntered
	}
	for index := 0; index < 4; index++ {
		releaseRuns <- struct{}{}
	}
	for index := 0; index < 4; index++ {
		if err := <-errCh; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}

	if startCalls != 2 {
		t.Fatalf("startCalls = %d, want 2", startCalls)
	}
	if maxActive != 4 {
		t.Fatalf("maxActive = %d, want 4", maxActive)
	}
}

func TestRunnerDoesNotOpenNewTabWhileWorkerIsRecycling(t *testing.T) {
	restore := stubBrowserProcessHooks(t)

	closeEntered := make(chan struct{})
	releaseClose := make(chan struct{})
	var closeOnce sync.Once
	var runCalls int
	var callsMu sync.Mutex
	restore.onClose = func() {
		closeOnce.Do(func() {
			close(closeEntered)
			<-releaseClose
		})
	}
	restore.onRun = func() {
		callsMu.Lock()
		runCalls++
		callsMu.Unlock()
	}

	r := New(Config{
		BrowserCount:     1,
		TabsPerBrowser:   2,
		RecycleAfterTabs: 1,
	})
	defer func() {
		if err := r.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	firstErr := make(chan error, 1)
	go func() {
		firstErr <- r.Run(context.Background(), "https://example.com/page")
	}()
	<-closeEntered

	secondCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	secondErr := r.Run(secondCtx, "https://example.com/page")
	if !errors.Is(secondErr, context.DeadlineExceeded) {
		t.Fatalf("Run() while recycling error = %v, want deadline exceeded", secondErr)
	}

	close(releaseClose)
	if err := <-firstErr; err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	callsMu.Lock()
	defer callsMu.Unlock()
	if runCalls != 1 {
		t.Fatalf("runCalls = %d, want 1", runCalls)
	}
}

func TestFetchCookieHeaderReturnsFetcherError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	r := &runner{
		cfg:   normalizeConfig(Config{}),
		cache: newCookieCache(),
		now:   time.Now,
		fetchCookieHeader: func(context.Context, Config, string) (string, error) {
			return "", wantErr
		},
	}

	_, err := r.FetchCookieHeader(context.Background(), "https://example.com/page")
	if !errors.Is(err, wantErr) {
		t.Fatalf("FetchCookieHeader() error = %v, want %v", err, wantErr)
	}
}

type browserHookRestore struct {
	mu      sync.Mutex
	onStart func()
	onClose func()
	onRun   func()
}

func stubBrowserProcessHooks(t *testing.T) *browserHookRestore {
	t.Helper()

	restore := &browserHookRestore{}
	originalStart := startBrowserProcessFunc
	originalClose := closeBrowserProcessFunc
	originalNewTab := newTabContextFunc
	originalRun := runActionsFunc

	startBrowserProcessFunc = func(context.Context, normalizedConfig) (browserProcess, error) {
		restore.mu.Lock()
		if restore.onStart != nil {
			restore.onStart()
		}
		restore.mu.Unlock()

		ctx, cancel := context.WithCancel(context.Background())
		return browserProcess{
			ctx:           ctx,
			browserCancel: cancel,
			allocCancel:   cancel,
		}, nil
	}
	closeBrowserProcessFunc = func(_ context.Context, process browserProcess) error {
		restore.mu.Lock()
		if restore.onClose != nil {
			restore.onClose()
		}
		restore.mu.Unlock()
		if process.browserCancel != nil {
			process.browserCancel()
		}
		if process.allocCancel != nil {
			process.allocCancel()
		}
		return nil
	}
	newTabContextFunc = func(parent context.Context, _ ...chromedp.ContextOption) (context.Context, context.CancelFunc) {
		return context.WithCancel(parent)
	}
	runActionsFunc = func(context.Context, ...chromedp.Action) error {
		restore.mu.Lock()
		onRun := restore.onRun
		restore.mu.Unlock()
		if onRun == nil {
			return nil
		}
		onRun()
		return nil
	}

	t.Cleanup(func() {
		startBrowserProcessFunc = originalStart
		closeBrowserProcessFunc = originalClose
		newTabContextFunc = originalNewTab
		runActionsFunc = originalRun
	})

	return restore
}

func stubObserveHooks(t *testing.T) {
	t.Helper()

	stubBrowserProcessHooks(t)

	originalListen := listenTargetFunc
	originalGetResponseBody := getResponseBodyFunc
	listenTargetFunc = func(context.Context, func(any)) {}
	getResponseBodyFunc = func(context.Context, network.RequestID) ([]byte, error) {
		return nil, nil
	}

	t.Cleanup(func() {
		listenTargetFunc = originalListen
		getResponseBodyFunc = originalGetResponseBody
	})
}
