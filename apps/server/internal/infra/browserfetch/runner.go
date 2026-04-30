package browserfetch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Runner 定义浏览器抓取基础设施的最小对外能力。
type Runner interface {
	FetchCookieHeader(ctx context.Context, pageURL string) (string, error)
	Run(ctx context.Context, pageURL string, opts ...RunOption) error
	ObserveResponses(ctx context.Context, pageURL string, opts ...ObserveOption) (*ResponseStream, error)
	Close(ctx context.Context) error
	InvalidateCookies()
}

// RunOption 允许调用方在单次 Run 中覆盖基础配置。
type RunOption func(*Config)

// WithRunBrowserPath 覆盖单次 Run 的浏览器路径。
func WithRunBrowserPath(browserPath string) RunOption {
	return func(cfg *Config) {
		cfg.BrowserPath = browserPath
	}
}

// WithRunHeadless 覆盖单次 Run 的无头模式。
func WithRunHeadless(headless bool) RunOption {
	return func(cfg *Config) {
		cfg.Headless = new(headless)
	}
}

// WithRunUserAgentMode 覆盖单次 Run 的 User-Agent 模式。
func WithRunUserAgentMode(mode string) RunOption {
	return func(cfg *Config) {
		cfg.UserAgentMode = mode
	}
}

// WithRunUserAgent 覆盖单次 Run 的 User-Agent。
func WithRunUserAgent(userAgent string) RunOption {
	return func(cfg *Config) {
		cfg.UserAgent = userAgent
	}
}

// WithRunTimeout 覆盖单次 Run 的超时时间。
func WithRunTimeout(timeout time.Duration) RunOption {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}

// WithRunWaitReadySelector 覆盖单次 Run 等待就绪的 CSS selector。
func WithRunWaitReadySelector(selector string) RunOption {
	return func(cfg *Config) {
		cfg.WaitReadySelector = selector
	}
}

// WithRunDisableImages 覆盖单次 Run 是否阻止常见图片资源。
func WithRunDisableImages(disabled bool) RunOption {
	return func(cfg *Config) {
		cfg.DisableImages = disabled
	}
}

type runner struct {
	cfg               normalizedConfig
	cache             *cookieCache
	now               func() time.Time
	fetchCookieHeader cookieFetcher
	resolveUserAgent  userAgentResolver

	mu                  sync.Mutex
	closed              bool
	nextWorker          int
	inflight            sync.WaitGroup
	workers             []*browserWorker
	standaloneProcesses map[*managedBrowserProcess]struct{}
}

type browserWorker struct {
	id               int
	slots            chan struct{}
	processKey       string
	recycleAfterTabs int

	mu               sync.Mutex
	allocCancel      context.CancelFunc
	browserCtx       context.Context
	browserCancel    context.CancelFunc
	activeTabs       int
	openedSinceStart int
	recycling        bool
}

type browserProcess struct {
	ctx           context.Context
	allocCancel   context.CancelFunc
	browserCancel context.CancelFunc
}

type managedBrowserProcess struct {
	process browserProcess
	once    sync.Once
}

var runPageFunc = runPage
var errRunnerClosed = errors.New("browser fetch runner is closed")
var errBrowserProcessConfigChanged = errors.New("browser process-level config changed after pool started")
var startBrowserProcessFunc = startBrowserProcess
var closeBrowserProcessFunc = closeBrowserProcess
var newTabContextFunc = chromedp.NewContext
var runActionsFunc = chromedp.Run
var listenTargetFunc = chromedp.ListenTarget
var getResponseBodyFunc = func(ctx context.Context, requestID network.RequestID) ([]byte, error) {
	return network.GetResponseBody(requestID).Do(ctx)
}

// New 创建一个基础浏览器 Runner。
func New(cfg Config) Runner {
	normalized := normalizeConfig(cfg)
	return newRunner(normalized)
}

func newRunner(cfg normalizedConfig) *runner {
	workers := make([]*browserWorker, 0, cfg.BrowserCount)
	for index := 0; index < cfg.BrowserCount; index++ {
		workers = append(workers, &browserWorker{
			id:               index,
			slots:            make(chan struct{}, cfg.TabsPerBrowser),
			processKey:       browserProcessKey(cfg),
			recycleAfterTabs: cfg.RecycleAfterTabs,
		})
	}

	return &runner{
		cfg:                 cfg,
		cache:               newCookieCache(),
		now:                 time.Now,
		resolveUserAgent:    defaultUserAgentResolver,
		workers:             workers,
		standaloneProcesses: make(map[*managedBrowserProcess]struct{}),
	}
}

// FetchCookieHeader 获取并缓存页面 Cookie Header。
func (r *runner) FetchCookieHeader(ctx context.Context, pageURL string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("fetch cookie header: runner is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(pageURL) == "" {
		return "", fmt.Errorf("fetch cookie header: page URL is required")
	}

	now := r.now
	if now == nil {
		now = time.Now
	}

	key := cookieCacheKey(r.cfg, pageURL)
	if header, ok := r.cache.get(key, now()); ok {
		return header, nil
	}

	fetcher := r.fetchCookieHeader
	if fetcher == nil {
		header, err := r.fetchCookieHeaderWithTab(ctx, pageURL)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(header) != "" {
			r.cache.set(key, header, now().Add(r.cfg.CookieCacheTTL))
		}
		return header, nil
	}

	header, err := fetcher(ctx, r.publicConfig(), pageURL)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(header) != "" {
		r.cache.set(key, header, now().Add(r.cfg.CookieCacheTTL))
	}
	return header, nil
}

// Run 使用浏览器访问页面并执行最小导航流程。
func (r *runner) Run(ctx context.Context, pageURL string, opts ...RunOption) error {
	if r == nil {
		return fmt.Errorf("run browser fetch: runner is nil")
	}

	cfg := r.publicConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	normalized := normalizeConfig(cfg)
	return runPageFunc(ctx, r, normalized, pageURL)
}

// Close 关闭浏览器进程并释放底层 chromedp 资源。
func (r *runner) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	workers := append([]*browserWorker(nil), r.workers...)
	standaloneProcesses := make([]*managedBrowserProcess, 0, len(r.standaloneProcesses))
	for process := range r.standaloneProcesses {
		standaloneProcesses = append(standaloneProcesses, process)
	}
	r.mu.Unlock()

	done := make(chan struct{})
	go func() {
		r.inflight.Wait()
		close(done)
	}()

	var waitErr error
	select {
	case <-done:
	case <-ctx.Done():
		waitErr = fmt.Errorf("wait browser tabs to close: %w", ctx.Err())
	}

	var closeErr error
	for _, worker := range workers {
		if err := worker.closeBrowser(ctx); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	}
	for _, process := range standaloneProcesses {
		if err := process.close(ctx); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	}
	if closeErr != nil {
		closeErr = fmt.Errorf("close browser fetch runner: %w", closeErr)
	}
	return errors.Join(waitErr, closeErr)
}

// InvalidateCookies 清空 Cookie 缓存。
func (r *runner) InvalidateCookies() {
	if r == nil || r.cache == nil {
		return
	}

	r.cache.invalidate()
}

func (r *runner) publicConfig() Config {
	headless := r.cfg.Headless
	return Config{
		BrowserPath:        r.cfg.BrowserPath,
		Headless:           new(headless),
		UserAgentMode:      r.cfg.UserAgentMode,
		UserAgent:          r.cfg.UserAgent,
		UserAgentPlatform:  r.cfg.UserAgentPlatform,
		AcceptLanguage:     r.cfg.AcceptLanguage,
		Timeout:            r.cfg.Timeout,
		CookieCacheTTL:     r.cfg.CookieCacheTTL,
		BrowserCount:       r.cfg.BrowserCount,
		TabsPerBrowser:     r.cfg.TabsPerBrowser,
		RecycleAfterTabs:   r.cfg.RecycleAfterTabs,
		MaxConcurrentTabs:  r.cfg.MaxConcurrentTabs,
		WaitReadySelector:  r.cfg.WaitReadySelector,
		DisableImages:      r.cfg.DisableImages,
		BlockedURLPatterns: append([]string(nil), r.cfg.BlockedURLPatterns...),
		NoSandbox:          r.cfg.NoSandbox,
		WindowWidth:        r.cfg.WindowWidth,
		WindowHeight:       r.cfg.WindowHeight,
		ExtraFlags:         append([]string(nil), r.cfg.ExtraFlags...),
	}
}

func runPage(ctx context.Context, r *runner, cfg normalizedConfig, pageURL string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(pageURL) == "" {
		return fmt.Errorf("run browser fetch: page URL is required")
	}

	return r.runInTab(ctx, cfg, pageURL, func(pageURL string) []chromedp.Action {
		return appendBrowserActions(cfg, []chromedp.Action{
			chromedp.Navigate(pageURL),
			chromedp.WaitReady(cfg.WaitReadySelector, chromedp.ByQuery),
		}, r.resolveUserAgent)
	})
}

func (r *runner) runInTab(ctx context.Context, cfg normalizedConfig, pageURL string, buildActions func(string) []chromedp.Action) error {
	return r.runInTabWithHooks(ctx, cfg, pageURL, buildActions, nil, nil)
}

func (r *runner) runInTabWithHooks(
	ctx context.Context,
	cfg normalizedConfig,
	pageURL string,
	buildActions func(string) []chromedp.Action,
	setup func(tabCtx context.Context, runCtx context.Context) error,
	afterRun func(tabCtx context.Context, runCtx context.Context) error,
) error {
	if r == nil {
		return fmt.Errorf("run browser tab: runner is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(pageURL) == "" {
		return fmt.Errorf("run browser tab: page URL is required")
	}
	if buildActions == nil {
		return fmt.Errorf("run browser tab: actions are required")
	}

	if err := r.addInflight(); err != nil {
		return err
	}
	defer r.inflight.Done()

	if browserProcessKey(cfg) != browserProcessKey(r.cfg) {
		return r.runInStandaloneProcess(ctx, cfg, pageURL, buildActions)
	}

	worker, releaseWorker, err := r.acquireWorker(ctx, cfg)
	if err != nil {
		return err
	}
	defer releaseWorker()

	browserCtx, err := worker.ensureBrowser(ctx, cfg)
	if err != nil {
		return err
	}
	worker.markTabOpened()

	tabCtx, tabCancel := newTabContextFunc(browserCtx)
	defer func() {
		if err := chromedp.Cancel(tabCtx); err != nil && !errors.Is(err, chromedp.ErrInvalidContext) {
			tabCancel()
		}
	}()

	runCtx := tabCtx
	var runCancel context.CancelFunc
	if cfg.Timeout > 0 {
		runCtx, runCancel = context.WithTimeout(tabCtx, cfg.Timeout)
		defer runCancel()
	}

	if setup != nil {
		if err := setup(tabCtx, runCtx); err != nil {
			return err
		}
	}
	if err := runActionsFunc(runCtx, buildActions(pageURL)...); err != nil {
		return fmt.Errorf("run browser for %s: %w", pageURL, err)
	}
	if afterRun != nil {
		if err := afterRun(tabCtx, runCtx); err != nil {
			return err
		}
	}

	return nil
}

func (r *runner) runInStandaloneProcess(ctx context.Context, cfg normalizedConfig, pageURL string, buildActions func(string) []chromedp.Action) error {
	process, err := startBrowserProcessFunc(ctx, cfg)
	if err != nil {
		return fmt.Errorf("start standalone browser fetch runner: %w", err)
	}
	managedProcess, err := r.trackStandaloneProcess(process)
	if err != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultBrowserCloseTimeout)
		_ = closeBrowserProcessFunc(closeCtx, process)
		cancel()
		return err
	}
	defer func() {
		r.untrackStandaloneProcess(managedProcess)
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultBrowserCloseTimeout)
		_ = managedProcess.close(closeCtx)
		cancel()
	}()

	tabCtx, tabCancel := newTabContextFunc(process.ctx)
	defer func() {
		if err := chromedp.Cancel(tabCtx); err != nil && !errors.Is(err, chromedp.ErrInvalidContext) {
			tabCancel()
		}
	}()

	runCtx := tabCtx
	var runCancel context.CancelFunc
	if cfg.Timeout > 0 {
		runCtx, runCancel = context.WithTimeout(tabCtx, cfg.Timeout)
		defer runCancel()
	}

	if err := runActionsFunc(runCtx, buildActions(pageURL)...); err != nil {
		return fmt.Errorf("run browser for %s: %w", pageURL, err)
	}

	return nil
}

func (r *runner) trackStandaloneProcess(process browserProcess) (*managedBrowserProcess, error) {
	managedProcess := &managedBrowserProcess{process: process}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, errRunnerClosed
	}
	r.standaloneProcesses[managedProcess] = struct{}{}

	return managedProcess, nil
}

func (r *runner) untrackStandaloneProcess(process *managedBrowserProcess) {
	if r == nil || process == nil {
		return
	}

	r.mu.Lock()
	delete(r.standaloneProcesses, process)
	r.mu.Unlock()
}

func (r *runner) acquireWorker(ctx context.Context, _ normalizedConfig) (*browserWorker, func(), error) {
	if r == nil {
		return nil, nil, fmt.Errorf("browser fetch runner is nil")
	}

	for {
		worker, release, err := r.tryAcquireWorker()
		if err != nil {
			return nil, nil, err
		}
		if worker != nil {
			return worker, release, nil
		}

		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, nil, fmt.Errorf("acquire browser worker tab: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func (r *runner) tryAcquireWorker() (*browserWorker, func(), error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, nil, errRunnerClosed
	}
	if len(r.workers) == 0 {
		return nil, nil, fmt.Errorf("browser worker pool is empty")
	}

	start := r.nextWorker
	for offset := range r.workers {
		index := (start + offset) % len(r.workers)
		worker := r.workers[index]
		if release, ok := worker.tryAcquire(); ok {
			r.nextWorker = (index + 1) % len(r.workers)
			return worker, release, nil
		}
	}

	return nil, nil, nil
}

func (r *runner) addInflight() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return errRunnerClosed
	}
	r.inflight.Add(1)
	return nil
}

func (w *browserWorker) tryAcquire() (func(), bool) {
	w.mu.Lock()
	if w.recycling {
		w.mu.Unlock()
		return nil, false
	}
	w.mu.Unlock()

	select {
	case w.slots <- struct{}{}:
		w.mu.Lock()
		if w.recycling {
			w.mu.Unlock()
			<-w.slots
			return nil, false
		}
		w.activeTabs++
		w.mu.Unlock()
		return func() {
			w.release()
		}, true
	default:
		return nil, false
	}
}

func (w *browserWorker) release() {
	shouldRecycle := false
	w.mu.Lock()
	if w.activeTabs > 0 {
		w.activeTabs--
	}
	if w.activeTabs == 0 {
		shouldRecycle = w.openedSinceStart >= w.recycleAfterTabs
		if shouldRecycle {
			w.recycling = true
		}
	}
	w.mu.Unlock()

	if shouldRecycle {
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultBrowserCloseTimeout)
		_ = w.closeBrowser(closeCtx)
		cancel()

		w.mu.Lock()
		w.recycling = false
		w.mu.Unlock()
	}

	select {
	case <-w.slots:
	default:
	}
}

func (w *browserWorker) markTabOpened() {
	w.mu.Lock()
	w.openedSinceStart++
	w.mu.Unlock()
}

func (w *browserWorker) ensureBrowser(ctx context.Context, cfg normalizedConfig) (context.Context, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	processKey := browserProcessKey(cfg)
	if w.browserCtx != nil {
		if w.processKey != processKey {
			return nil, errBrowserProcessConfigChanged
		}
		return w.browserCtx, nil
	}

	process, err := startBrowserProcessFunc(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("start browser fetch runner: %w", err)
	}

	w.processKey = processKey
	w.allocCancel = process.allocCancel
	w.browserCtx = process.ctx
	w.browserCancel = process.browserCancel
	w.openedSinceStart = 0

	return w.browserCtx, nil
}

func (w *browserWorker) closeBrowser(ctx context.Context) error {
	w.mu.Lock()
	process := browserProcess{
		ctx:           w.browserCtx,
		allocCancel:   w.allocCancel,
		browserCancel: w.browserCancel,
	}
	w.browserCtx = nil
	w.allocCancel = nil
	w.browserCancel = nil
	w.openedSinceStart = 0
	w.mu.Unlock()

	if process.ctx == nil && process.allocCancel == nil && process.browserCancel == nil {
		return nil
	}

	return closeBrowserProcessFunc(ctx, process)
}

func startBrowserProcess(ctx context.Context, cfg normalizedConfig) (browserProcess, error) {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if cfg.BrowserPath != "" {
		opts = append(opts, chromedp.ExecPath(cfg.BrowserPath))
	}
	opts = append(opts, buildExecAllocatorOptions(cfg)...)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	initCtx, initCancel := context.WithTimeout(browserCtx, cfg.Timeout)
	defer initCancel()
	if err := runActionsFunc(initCtx); err != nil {
		browserCancel()
		allocCancel()
		return browserProcess{}, err
	}

	select {
	case <-ctx.Done():
		closeCtx, closeCancel := context.WithTimeout(context.Background(), defaultBrowserCloseTimeout)
		_ = closeBrowserProcess(closeCtx, browserProcess{
			ctx:           browserCtx,
			allocCancel:   allocCancel,
			browserCancel: browserCancel,
		})
		closeCancel()
		return browserProcess{}, ctx.Err()
	default:
	}

	return browserProcess{
		ctx:           browserCtx,
		allocCancel:   allocCancel,
		browserCancel: browserCancel,
	}, nil
}

func closeBrowserProcess(ctx context.Context, process browserProcess) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if process.ctx == nil && process.allocCancel == nil && process.browserCancel == nil {
		return nil
	}

	done := make(chan error, 1)
	go func() {
		var closeErr error
		if process.ctx != nil {
			closeErr = chromedp.Cancel(process.ctx)
			if errors.Is(closeErr, chromedp.ErrInvalidContext) {
				closeErr = nil
				if process.browserCancel != nil {
					process.browserCancel()
				}
			}
		} else if process.browserCancel != nil {
			process.browserCancel()
		}
		if process.allocCancel != nil {
			process.allocCancel()
		}

		done <- closeErr
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if process.browserCancel != nil {
			process.browserCancel()
		}
		if process.allocCancel != nil {
			process.allocCancel()
		}
		return fmt.Errorf("close browser process: %w", ctx.Err())
	}
}

func (p *managedBrowserProcess) close(ctx context.Context) error {
	if p == nil {
		return nil
	}

	var closeErr error
	p.once.Do(func() {
		closeErr = closeBrowserProcessFunc(ctx, p.process)
	})

	return closeErr
}

func buildExecAllocatorOptions(cfg normalizedConfig) []chromedp.ExecAllocatorOption {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", cfg.Headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.WindowSize(cfg.WindowWidth, cfg.WindowHeight),
	}
	if cfg.NoSandbox {
		opts = append(opts, chromedp.NoSandbox)
	}
	for _, flag := range cfg.ExtraFlags {
		name, value := parseChromeFlag(flag)
		if name == "" {
			continue
		}
		opts = append(opts, chromedp.Flag(name, value))
	}

	return opts
}

func parseChromeFlag(flag string) (string, any) {
	flag = strings.TrimLeft(strings.TrimSpace(flag), "-")
	if flag == "" {
		return "", true
	}
	name, value, ok := strings.Cut(flag, "=")
	name = strings.TrimSpace(name)
	if name == "" {
		return "", true
	}
	if !ok {
		return name, true
	}

	return name, strings.TrimSpace(value)
}

func appendBrowserActions(cfg normalizedConfig, actions []chromedp.Action, resolve userAgentResolver) []chromedp.Action {
	preflight := []chromedp.Action{
		network.Enable(),
	}
	if userAgent := resolveUserAgent(cfg, resolve); userAgent != "" {
		override := emulation.SetUserAgentOverride(userAgent).
			WithAcceptLanguage(cfg.AcceptLanguage).
			WithPlatform(navigatorPlatform(cfg.UserAgentPlatform))
		preflight = append(preflight, override)
	}
	if patterns := buildBlockedURLPatterns(cfg); len(patterns) > 0 {
		preflight = append(preflight, network.SetBlockedURLs().WithURLPatterns(patterns))
	}

	return append(preflight, actions...)
}

func buildBlockedURLPatterns(cfg normalizedConfig) []*network.BlockPattern {
	patterns := make([]string, 0, len(cfg.BlockedURLPatterns)+5)
	if cfg.DisableImages {
		patterns = append(patterns, "*://*/*.png", "*://*/*.jpg", "*://*/*.jpeg", "*://*/*.gif", "*://*/*.webp")
	}
	patterns = append(patterns, cfg.BlockedURLPatterns...)
	if len(patterns) == 0 {
		return nil
	}

	blockPatterns := make([]*network.BlockPattern, 0, len(patterns))
	for _, pattern := range patterns {
		blockPatterns = append(blockPatterns, &network.BlockPattern{
			URLPattern: pattern,
			Block:      true,
		})
	}

	return blockPatterns
}

func resolveUserAgent(cfg normalizedConfig, resolve userAgentResolver) string {
	switch cfg.UserAgentMode {
	case UserAgentModeCustom:
		return strings.TrimSpace(cfg.UserAgent)
	case UserAgentModeFake:
		if resolve != nil {
			if userAgent, err := resolve(cfg); err == nil && strings.TrimSpace(userAgent) != "" {
				return strings.TrimSpace(userAgent)
			}
		}
		return strings.TrimSpace(cfg.UserAgent)
	default:
		return ""
	}
}

func navigatorPlatform(platform string) string {
	if platform == UserAgentPlatformMobile {
		return "iPhone"
	}

	return "Win32"
}
