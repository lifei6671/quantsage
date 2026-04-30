package browserfetch

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func TestBaiduFinanceWaitReadySmoke(t *testing.T) {
	if os.Getenv("FINSCOPE_STOCKS_SMOKE") != "1" {
		t.Skip("set FINSCOPE_STOCKS_SMOKE=1 to run real browser smoke")
	}

	const pageURL = "https://finance.baidu.com/index/ab-000001?mainTab=%E6%88%90%E5%88%86%E8%82%A1"

	testCases := []struct {
		name    string
		actions []chromedp.Action
	}{
		{
			name: "navigate_only",
			actions: []chromedp.Action{
				chromedp.Navigate(pageURL),
			},
		},
		{
			name: "navigate_then_wait_body",
			actions: []chromedp.Action{
				chromedp.Navigate(pageURL),
				chromedp.WaitReady("body", chromedp.ByQuery),
			},
		},
		{
			name: "navigate_then_wait_app",
			actions: []chromedp.Action{
				chromedp.Navigate(pageURL),
				chromedp.WaitReady("#app", chromedp.ByQuery),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Timeout: 90 * time.Second,
			}
			if path := strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_BROWSER_PATH")); path != "" {
				cfg.BrowserPath = path
			}

			r := New(cfg)
			defer func() {
				_ = r.Close(context.Background())
			}()

			runnerImpl, ok := r.(*runner)
			if !ok {
				t.Fatal("runner type assertion failed")
			}

			err := runnerImpl.runInTab(
				context.Background(),
				runnerImpl.cfg,
				pageURL,
				func(pageURL string) []chromedp.Action {
					return appendBrowserActions(runnerImpl.cfg, tc.actions, runnerImpl.resolveUserAgent)
				},
			)
			if err != nil {
				t.Fatalf("runInTab() error = %v", err)
			}
		})
	}
}

func TestBaiduFinanceDirectChromedpSmoke(t *testing.T) {
	if os.Getenv("FINSCOPE_STOCKS_SMOKE") != "1" {
		t.Skip("set FINSCOPE_STOCKS_SMOKE=1 to run real browser smoke")
	}

	const pageURL = "https://finance.baidu.com/index/ab-000001?mainTab=%E6%88%90%E5%88%86%E8%82%A1"

	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if path := strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_BROWSER_PATH")); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	ctx, cancel := context.WithTimeout(browserCtx, 90*time.Second)
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.Navigate(pageURL)); err != nil {
		t.Fatalf("chromedp.Run() error = %v", err)
	}
}

func TestBaiduFinanceDirectPageNavigateSmoke(t *testing.T) {
	if os.Getenv("FINSCOPE_STOCKS_SMOKE") != "1" {
		t.Skip("set FINSCOPE_STOCKS_SMOKE=1 to run real browser smoke")
	}

	const pageURL = "https://finance.baidu.com/index/ab-000001?mainTab=%E6%88%90%E5%88%86%E8%82%A1"

	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if path := strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_BROWSER_PATH")); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	ctx, cancel := context.WithTimeout(browserCtx, 90*time.Second)
	defer cancel()

	var (
		title      string
		readyState string
	)
	if err := chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			_, _, errorText, _, err := page.Navigate(pageURL).Do(actionCtx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(errorText) != "" {
				return context.Canceled
			}
			return nil
		}),
		chromedp.Sleep(8*time.Second),
		chromedp.Evaluate(`document.readyState`, &readyState),
		chromedp.Title(&title),
	); err != nil {
		t.Fatalf("chromedp.Run() with page.Navigate error = %v", err)
	}

	t.Logf("readyState=%q title=%q", readyState, title)
}

func TestBaiduFinanceChildTabPageNavigateSmoke(t *testing.T) {
	if os.Getenv("FINSCOPE_STOCKS_SMOKE") != "1" {
		t.Skip("set FINSCOPE_STOCKS_SMOKE=1 to run real browser smoke")
	}

	const pageURL = "https://finance.baidu.com/index/ab-000001?mainTab=%E6%88%90%E5%88%86%E8%82%A1"

	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if path := strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_BROWSER_PATH")); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	initCtx, initCancel := context.WithTimeout(browserCtx, 30*time.Second)
	if err := chromedp.Run(initCtx); err != nil {
		initCancel()
		t.Fatalf("initial chromedp.Run() error = %v", err)
	}
	initCancel()

	tabCtx, tabCancel := chromedp.NewContext(browserCtx)
	defer tabCancel()

	ctx, cancel := context.WithTimeout(tabCtx, 90*time.Second)
	defer cancel()

	var (
		title      string
		readyState string
	)
	if err := chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			_, _, errorText, _, err := page.Navigate(pageURL).Do(actionCtx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(errorText) != "" {
				return context.Canceled
			}
			return nil
		}),
		chromedp.Sleep(8*time.Second),
		chromedp.Evaluate(`document.readyState`, &readyState),
		chromedp.Title(&title),
	); err != nil {
		t.Fatalf("child tab chromedp.Run() error = %v", err)
	}

	t.Logf("readyState=%q title=%q", readyState, title)
}

func TestBaiduFinanceRootPageNavigateAfterInitSmoke(t *testing.T) {
	if os.Getenv("FINSCOPE_STOCKS_SMOKE") != "1" {
		t.Skip("set FINSCOPE_STOCKS_SMOKE=1 to run real browser smoke")
	}

	const pageURL = "https://finance.baidu.com/index/ab-000001?mainTab=%E6%88%90%E5%88%86%E8%82%A1"

	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if path := strings.TrimSpace(os.Getenv("FINSCOPE_SMOKE_BROWSER_PATH")); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	initCtx, initCancel := context.WithTimeout(browserCtx, 30*time.Second)
	if err := chromedp.Run(initCtx); err != nil {
		initCancel()
		t.Fatalf("initial chromedp.Run() error = %v", err)
	}
	initCancel()

	ctx, cancel := context.WithTimeout(browserCtx, 90*time.Second)
	defer cancel()

	var (
		title      string
		readyState string
	)
	if err := chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			_, _, errorText, _, err := page.Navigate(pageURL).Do(actionCtx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(errorText) != "" {
				return context.Canceled
			}
			return nil
		}),
		chromedp.Sleep(8*time.Second),
		chromedp.Evaluate(`document.readyState`, &readyState),
		chromedp.Title(&title),
	); err != nil {
		t.Fatalf("root page chromedp.Run() after init error = %v", err)
	}

	t.Logf("readyState=%q title=%q", readyState, title)
}
