package finscope

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/lifei6671/quantsage/apps/server/internal/infra/browserfetch"
	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

type constituentBodyCapturer interface {
	CaptureConstituentBodies(ctx context.Context, cfg Config, query constituentQuery) ([][]byte, error)
}

type capturedConstituentResponse struct {
	URL  string `json:"url"`
	Body string `json:"body"`
}

type constituentCaptureStatus struct {
	Pending               int   `json:"pending"`
	LastActivityUnixMilli int64 `json:"lastActivityUnixMilli"`
}

var constituentCapturePollInterval = 300 * time.Millisecond
var readCapturedConstituentResponsesFunc = readCapturedConstituentResponses
var readCapturedConstituentStatusFunc = readCapturedConstituentStatus
var waitForCapturedConstituentResponsesFunc = waitForCapturedConstituentResponses
var waitForCapturedConstituentResponsesStableFunc = waitForCapturedConstituentResponsesToStabilize

func (s *Source) watchConstituentPageBodies(ctx context.Context, query constituentQuery) ([][]byte, error) {
	if s == nil || s.browser == nil {
		return nil, browserUnavailableError()
	}
	if capturer, ok := s.browser.(constituentBodyCapturer); ok {
		return capturer.CaptureConstituentBodies(ctx, s.cfg, query)
	}

	var captured []capturedConstituentResponse
	err := s.browser.RunWithActions(
		ctx,
		s.cfg.BasePageURL,
		s.buildConstituentBeforeNavigateActions(),
		s.buildConstituentAfterReadyActions(&captured),
		browserfetch.WithRunPrimaryPageTarget(true),
		browserfetch.WithRunRawPageNavigate(true),
	)
	if err != nil {
		return nil, fmt.Errorf("run finscope constituent watcher: %w", err)
	}

	result := make([][]byte, 0, len(captured))
	seen := make(map[string]struct{}, len(captured))
	for _, item := range captured {
		if !query.matches(item.URL, s.cfg.ConstituentAPIURL) {
			continue
		}
		body := item.Body
		if body == "" {
			continue
		}
		key := item.URL + "\x00" + body
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, []byte(body))
	}
	if len(result) == 0 {
		return nil, apperror.New(
			apperror.CodeDatasourceUnavailable,
			errors.New("finscope datasource observed no constituent response body"),
		)
	}

	return result, nil
}

func (s *Source) buildConstituentBeforeNavigateActions() []chromedp.Action {
	return []chromedp.Action{
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(constituentCaptureScript()).Do(actionCtx)
			if err != nil {
				return fmt.Errorf("install finscope constituent capture script: %w", err)
			}
			return nil
		}),
	}
}

func (s *Source) buildConstituentAfterReadyActions(target *[]capturedConstituentResponse) []chromedp.Action {
	if s == nil {
		return nil
	}

	idleTimeout := s.observeIdleTimeout()
	pause := s.cfg.ConstituentScrollPause
	if pause <= 0 {
		pause = defaultConstituentScrollPause
	}
	maxRounds := s.cfg.ConstituentMaxScrollRounds
	if maxRounds <= 0 {
		maxRounds = defaultConstituentMaxScrollRounds
	}
	stableRounds := s.cfg.ConstituentStableScrollRounds
	if stableRounds <= 0 {
		stableRounds = defaultConstituentStableScrollLimit
	}

	return []chromedp.Action{
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			return waitForCapturedConstituentResponsesFunc(actionCtx, 1, idleTimeout)
		}),
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			return scrollPageUntilStable(actionCtx, pause, maxRounds, stableRounds)
		}),
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			return waitForCapturedConstituentResponsesStableFunc(actionCtx, idleTimeout)
		}),
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			items, err := readCapturedConstituentResponsesFunc(actionCtx)
			if err != nil {
				return err
			}
			if target != nil {
				*target = items
			}
			return nil
		}),
	}
}

func (s *Source) observeIdleTimeout() time.Duration {
	if s == nil || s.cfg.ObserveIdleTimeout <= 0 {
		return defaultObserveIdleTimeout
	}

	return s.cfg.ObserveIdleTimeout
}

func constituentCaptureScript() string {
	return `(function () {
		const storeKey = "__qsFinscopeConstituentBodies";
		const stateKey = "__qsFinscopeConstituentState";
		const matches = function (url) {
			return typeof url === "string" && url.indexOf("/sapi/v1/constituents") >= 0;
		};
		const ensureState = function () {
			if (!window[stateKey] || typeof window[stateKey] !== "object") {
				window[stateKey] = { pending: 0, lastActivityUnixMilli: Date.now() };
			}
			return window[stateKey];
		};
		const markActivity = function () {
			ensureState().lastActivityUnixMilli = Date.now();
		};
		const changePending = function (delta) {
			const state = ensureState();
			state.pending = Math.max(0, Number(state.pending || 0) + delta);
			state.lastActivityUnixMilli = Date.now();
		};
		const push = function (url, body) {
			try {
				if (!Array.isArray(window[storeKey])) {
					window[storeKey] = [];
				}
				window[storeKey].push({ url: String(url || ""), body: String(body || "") });
				markActivity();
			} catch (_) {}
		};

		window[storeKey] = window[storeKey] || [];
		ensureState();

		if (typeof window.fetch === "function" && !window.fetch.__qsWrapped) {
			const originalFetch = window.fetch;
			const wrappedFetch = function () {
				const request = arguments[0];
				const requestURL = request && request.url ? request.url : String(request || "");
				const tracked = matches(requestURL);
				if (tracked) {
					changePending(1);
				}
				return originalFetch.apply(this, arguments).then(function (response) {
					const responseURL = response && response.url ? response.url : requestURL;
					if (!tracked && !matches(responseURL)) {
						return response;
					}
					if (response && typeof response.clone === "function") {
						return response.clone().text().then(function (body) {
							push(responseURL, body);
							return response;
						}).catch(function () {
							return response;
						}).finally(function () {
							changePending(-1);
						});
					}
					changePending(-1);
					return response;
				}).catch(function (err) {
					if (tracked) {
						changePending(-1);
					}
					throw err;
				});
			};
			wrappedFetch.__qsWrapped = true;
			window.fetch = wrappedFetch;
		}

		if (window.XMLHttpRequest && !window.XMLHttpRequest.prototype.__qsWrapped) {
			const originalOpen = window.XMLHttpRequest.prototype.open;
			const originalSend = window.XMLHttpRequest.prototype.send;
			window.XMLHttpRequest.prototype.open = function (method, url) {
				this.__qsURL = url;
				this.__qsTracked = matches(url);
				return originalOpen.apply(this, arguments);
			};
			window.XMLHttpRequest.prototype.send = function () {
				if (this.__qsTracked) {
					changePending(1);
				}
				this.addEventListener("load", function () {
					try {
						const responseURL = this.responseURL || this.__qsURL || "";
						if (matches(responseURL) && typeof this.responseText === "string") {
							push(responseURL, this.responseText);
						}
					} catch (_) {}
				});
				this.addEventListener("loadend", function () {
					if (this.__qsTracked) {
						changePending(-1);
					}
				});
				return originalSend.apply(this, arguments);
			};
			window.XMLHttpRequest.prototype.__qsWrapped = true;
		}
	})();`
}

func waitForCapturedConstituentResponses(ctx context.Context, minCount int, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultObserveIdleTimeout
	}
	deadline := time.Now().Add(timeout)
	for {
		items, err := readCapturedConstituentResponsesFunc(ctx)
		if err != nil {
			return err
		}
		if len(items) >= minCount {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("wait finscope constituent response: timeout after %s", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(constituentCapturePollInterval):
		}
	}
}

func waitForCapturedConstituentResponsesToStabilize(ctx context.Context, idleTimeout time.Duration) error {
	if idleTimeout <= 0 {
		idleTimeout = defaultObserveIdleTimeout
	}

	for {
		status, err := readCapturedConstituentStatusFunc(ctx)
		if err != nil {
			return err
		}
		now := time.Now()
		lastActivity := now
		if status.LastActivityUnixMilli > 0 {
			lastActivity = time.UnixMilli(status.LastActivityUnixMilli)
		}
		if status.Pending == 0 && now.Sub(lastActivity) >= idleTimeout {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(constituentCapturePollInterval):
		}
	}
}

func readCapturedConstituentResponses(ctx context.Context) ([]capturedConstituentResponse, error) {
	var items []capturedConstituentResponse
	err := chromedp.Evaluate(`window.__qsFinscopeConstituentBodies || []`, &items).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("read finscope captured constituent responses: %w", err)
	}

	return items, nil
}

func readCapturedConstituentStatus(ctx context.Context) (constituentCaptureStatus, error) {
	var status constituentCaptureStatus
	err := chromedp.Evaluate(`window.__qsFinscopeConstituentState || { pending: 0, lastActivityUnixMilli: Date.now() }`, &status).Do(ctx)
	if err != nil {
		return constituentCaptureStatus{}, fmt.Errorf("read finscope constituent capture status: %w", err)
	}

	return status, nil
}

func scrollPageUntilStable(
	ctx context.Context,
	pause time.Duration,
	maxRounds int,
	stableRounds int,
) error {
	lastHeight := int64(0)
	unchanged := 0
	for round := 0; round < maxRounds; round++ {
		if err := scrollToPageBottom(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pause):
		}

		height, err := currentPageScrollHeight(ctx)
		if err != nil {
			return err
		}
		if height <= lastHeight {
			unchanged++
			if unchanged >= stableRounds {
				return nil
			}
			continue
		}
		lastHeight = height
		unchanged = 0
	}

	return nil
}

func scrollToPageBottom(ctx context.Context) error {
	return chromedp.Evaluate(`(() => {
		const root = document.scrollingElement || document.documentElement || document.body;
		if (!root) {
			return 0;
		}
		window.scrollTo(0, root.scrollHeight);
		return root.scrollHeight;
	})()`, nil).Do(ctx)
}

func currentPageScrollHeight(ctx context.Context) (int64, error) {
	var height int64
	err := chromedp.Evaluate(`(() => {
		const root = document.scrollingElement || document.documentElement || document.body;
		if (!root) {
			return 0;
		}
		return root.scrollHeight;
	})()`, &height).Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("read finscope page scroll height: %w", err)
	}

	return height, nil
}

func browserUnavailableError() error {
	return apperror.New(
		apperror.CodeDatasourceUnavailable,
		errors.New("finscope datasource browser watcher is not configured"),
	)
}
