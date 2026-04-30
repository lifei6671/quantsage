package browserfetch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const defaultObserveIdleTimeout = 5 * time.Second

var errObserveNoMatchingResponse = errors.New("no matching response observed before idle timeout")

// ResponseMetadata 描述一次页面响应的基础元信息。
type ResponseMetadata struct {
	URL          string
	Status       int
	MIMEType     string
	ResourceType string
}

// ResponseStreamItem 表示一次被监听到的原始响应。
type ResponseStreamItem struct {
	URL        string
	ReceivedAt time.Time
	Body       []byte
	Err        error
}

// ResponseStream 表示持续监听页面响应的流式结果。
type ResponseStream struct {
	Responses <-chan ResponseStreamItem
	Done      <-chan error
	Close     func()
}

// ObserveOption 调整单次响应监听行为。
type ObserveOption func(*observeOptions)

type observeOptions struct {
	idleTimeout   time.Duration
	resourceTypes map[string]struct{}
	matchers      []func(ResponseMetadata) bool
}

// WithObserveIdleTimeout 覆盖页面响应监听的空闲收口时间。
func WithObserveIdleTimeout(timeout time.Duration) ObserveOption {
	return func(opts *observeOptions) {
		opts.idleTimeout = timeout
	}
}

// WithObserveURLContains 要求命中的响应 URL 必须包含指定片段。
func WithObserveURLContains(fragment string) ObserveOption {
	fragment = strings.TrimSpace(fragment)
	return func(opts *observeOptions) {
		if fragment == "" {
			return
		}
		opts.matchers = append(opts.matchers, func(meta ResponseMetadata) bool {
			return strings.Contains(meta.URL, fragment)
		})
	}
}

// WithObserveResourceTypes 限制只监听指定资源类型。
func WithObserveResourceTypes(resourceTypes ...network.ResourceType) ObserveOption {
	return func(opts *observeOptions) {
		if opts.resourceTypes == nil {
			opts.resourceTypes = make(map[string]struct{}, len(resourceTypes))
		}
		for _, resourceType := range resourceTypes {
			trimmed := strings.TrimSpace(string(resourceType))
			if trimmed == "" {
				continue
			}
			opts.resourceTypes[trimmed] = struct{}{}
		}
	}
}

// WithObserveMatch 允许调用方按响应元信息追加自定义过滤条件。
func WithObserveMatch(match func(ResponseMetadata) bool) ObserveOption {
	return func(opts *observeOptions) {
		if match == nil {
			return
		}
		opts.matchers = append(opts.matchers, match)
	}
}

// ObserveResponses 持续监听页面命中的原始响应，直到 ctx 取消或进入空闲超时。
func (r *runner) ObserveResponses(ctx context.Context, pageURL string, opts ...ObserveOption) (*ResponseStream, error) {
	if r == nil {
		return nil, fmt.Errorf("observe browser responses: runner is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(pageURL) == "" {
		return nil, fmt.Errorf("observe browser responses: page URL is required")
	}

	options := normalizeObserveOptions(opts...)
	streamCtx, cancel := context.WithCancel(ctx)
	responses := make(chan ResponseStreamItem, 16)
	done := make(chan error, 1)
	stream := &ResponseStream{
		Responses: responses,
		Done:      done,
		Close:     cancel,
	}

	go func() {
		defer cancel()
		err := r.observeResponses(streamCtx, r.cfg, pageURL, options, responses)
		close(responses)
		done <- err
		close(done)
	}()

	return stream, nil
}

func normalizeObserveOptions(opts ...ObserveOption) observeOptions {
	options := observeOptions{
		idleTimeout: defaultObserveIdleTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.idleTimeout <= 0 {
		options.idleTimeout = defaultObserveIdleTimeout
	}

	return options
}

func (o observeOptions) matches(meta ResponseMetadata) bool {
	if len(o.resourceTypes) > 0 {
		if _, ok := o.resourceTypes[meta.ResourceType]; !ok {
			return false
		}
	}
	for _, matcher := range o.matchers {
		if matcher != nil && !matcher(meta) {
			return false
		}
	}

	return true
}

func (r *runner) observeResponses(
	ctx context.Context,
	cfg normalizedConfig,
	pageURL string,
	options observeOptions,
	out chan<- ResponseStreamItem,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	lifecycleCtx, lifecycleCancel := context.WithCancel(ctx)
	idleDone := make(chan struct{})
	defer func() {
		lifecycleCancel()
		<-idleDone
	}()

	stopCh := make(chan struct{})
	touchCh := make(chan struct{}, 1)
	idleStartCh := make(chan struct{}, 1)
	pendingDeltaCh := make(chan int, 8)
	var stopOnce sync.Once
	var idleStartOnce sync.Once
	var stopErr error
	stop := func(err error) {
		stopOnce.Do(func() {
			stopErr = err
			close(stopCh)
		})
	}
	startIdle := func() {
		idleStartOnce.Do(func() {
			idleStartCh <- struct{}{}
		})
	}
	touch := func() {
		select {
		case touchCh <- struct{}{}:
		default:
		}
	}

	go func() {
		defer close(idleDone)
		var (
			timer           *time.Timer
			timerCh         <-chan time.Time
			pendingActivity bool
			pendingCount    int
		)
		for {
			select {
			case <-lifecycleCtx.Done():
				stop(lifecycleCtx.Err())
				return
			case <-idleStartCh:
				if timer == nil {
					timer = time.NewTimer(options.idleTimeout)
					timerCh = timer.C
					if pendingActivity && pendingCount == 0 {
						resetObserveTimer(timer, options.idleTimeout)
						pendingActivity = false
					}
				}
			case delta := <-pendingDeltaCh:
				pendingCount += delta
				if pendingCount < 0 {
					pendingCount = 0
				}
				if pendingCount > 0 {
					if timer != nil && !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timerCh = nil
					continue
				}
				if timer == nil {
					continue
				}
				timerCh = timer.C
				resetObserveTimer(timer, options.idleTimeout)
				pendingActivity = false
			case <-touchCh:
				if timer == nil {
					pendingActivity = true
					continue
				}
				if pendingCount > 0 {
					pendingActivity = true
					continue
				}
				resetObserveTimer(timer, options.idleTimeout)
			case <-timerCh:
				stop(nil)
				return
			}
		}
	}()

	pending := make(map[network.RequestID]ResponseMetadata)
	var pendingMu sync.Mutex
	var fetchWG sync.WaitGroup
	var capturedMu sync.Mutex
	capturedCount := 0

	err := r.runInTabWithHooks(
		ctx,
		cfg,
		pageURL,
		func(pageURL string) []chromedp.Action {
			return appendBrowserActions(cfg, []chromedp.Action{
				chromedp.Navigate(pageURL),
				chromedp.WaitReady(cfg.WaitReadySelector, chromedp.ByQuery),
			}, r.resolveUserAgent)
		},
		func(tabCtx context.Context, runCtx context.Context) error {
			listenTargetFunc(tabCtx, func(event any) {
				switch event := event.(type) {
				case *network.EventResponseReceived:
					if event == nil {
						return
					}

					meta := ResponseMetadata{ResourceType: string(event.Type)}
					if event.Response != nil {
						meta.URL = strings.TrimSpace(event.Response.URL)
						meta.Status = int(event.Response.Status)
						meta.MIMEType = strings.TrimSpace(event.Response.MimeType)
					}
					if !options.matches(meta) {
						return
					}

					touch()
					pendingMu.Lock()
					pending[event.RequestID] = meta
					pendingMu.Unlock()
					select {
					case pendingDeltaCh <- 1:
					case <-lifecycleCtx.Done():
					}
				case *network.EventLoadingFinished:
					if event == nil {
						return
					}

					meta, ok := takePendingResponse(pending, &pendingMu, event.RequestID)
					if !ok {
						return
					}
					select {
					case pendingDeltaCh <- -1:
					case <-lifecycleCtx.Done():
					}

					touch()
					fetchWG.Add(1)
					go func(meta ResponseMetadata, requestID network.RequestID) {
						defer fetchWG.Done()

						body, err := getResponseBodyFunc(runCtx, requestID)
						if err != nil {
							err = fmt.Errorf("get response body for %s: %w", meta.URL, err)
						}
						item := ResponseStreamItem{
							URL:        meta.URL,
							ReceivedAt: time.Now().UTC(),
							Body:       body,
							Err:        err,
						}
						if emitErr := emitObservedResponse(ctx, out, item); emitErr != nil {
							stop(emitErr)
							return
						}
						capturedMu.Lock()
						capturedCount++
						capturedMu.Unlock()
						if err != nil {
							stop(err)
							return
						}
						touch()
					}(meta, event.RequestID)
				case *network.EventLoadingFailed:
					if event == nil || event.Canceled {
						return
					}

					meta, ok := takePendingResponse(pending, &pendingMu, event.RequestID)
					if !ok {
						return
					}
					select {
					case pendingDeltaCh <- -1:
					case <-lifecycleCtx.Done():
					}
					err := fmt.Errorf("load response body for %s: %s", meta.URL, strings.TrimSpace(event.ErrorText))
					item := ResponseStreamItem{
						URL:        meta.URL,
						ReceivedAt: time.Now().UTC(),
						Err:        err,
					}
					if emitErr := emitObservedResponse(ctx, out, item); emitErr != nil {
						stop(emitErr)
						return
					}
					stop(err)
				}
			})

			return nil
		},
		func(context.Context, context.Context) error {
			startIdle()
			select {
			case <-stopCh:
			case <-ctx.Done():
				stop(ctx.Err())
				<-stopCh
			}

			fetchWG.Wait()
			if stopErr != nil {
				return stopErr
			}

			capturedMu.Lock()
			defer capturedMu.Unlock()
			if capturedCount == 0 {
				return errObserveNoMatchingResponse
			}

			return nil
		},
	)
	fetchWG.Wait()
	if err != nil {
		return fmt.Errorf("observe browser responses for %s: %w", pageURL, err)
	}

	return nil
}

func takePendingResponse(
	pending map[network.RequestID]ResponseMetadata,
	mu *sync.Mutex,
	requestID network.RequestID,
) (ResponseMetadata, bool) {
	mu.Lock()
	defer mu.Unlock()

	meta, ok := pending[requestID]
	if !ok {
		return ResponseMetadata{}, false
	}
	delete(pending, requestID)

	return meta, true
}

func emitObservedResponse(ctx context.Context, out chan<- ResponseStreamItem, item ResponseStreamItem) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- item:
		return nil
	}
}

func resetObserveTimer(timer *time.Timer, timeout time.Duration) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(timeout)
}
