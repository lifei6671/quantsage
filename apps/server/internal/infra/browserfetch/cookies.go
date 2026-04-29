package browserfetch

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type cookieFetcher func(context.Context, Config, string) (string, error)

func (r *runner) fetchCookieHeaderWithTab(ctx context.Context, pageURL string) (string, error) {
	if strings.TrimSpace(pageURL) == "" {
		return "", fmt.Errorf("fetch cookie header: page URL is required")
	}

	var cookies []*network.Cookie
	if err := r.runInTab(ctx, r.cfg, pageURL, func(pageURL string) []chromedp.Action {
		return appendBrowserActions(r.cfg, []chromedp.Action{
			chromedp.Navigate(pageURL),
			chromedp.WaitReady(r.cfg.WaitReadySelector, chromedp.ByQuery),
			chromedp.ActionFunc(func(actionCtx context.Context) error {
				var inner error
				cookies, inner = network.GetCookies().WithURLs([]string{pageURL}).Do(actionCtx)
				return inner
			}),
		}, r.resolveUserAgent)
	}); err != nil {
		return "", fmt.Errorf("fetch cookie header for %s: %w", pageURL, err)
	}

	return cookieHeaderFromNetwork(cookies), nil
}

func cookieHeaderFromNetwork(cookies []*network.Cookie) string {
	if len(cookies) == 0 {
		return ""
	}

	var builder strings.Builder
	first := true
	for _, cookie := range cookies {
		if cookie == nil || cookie.Name == "" {
			continue
		}

		if !first {
			builder.WriteString("; ")
		}
		first = false
		builder.WriteString(cookie.Name)
		builder.WriteByte('=')
		builder.WriteString(cookie.Value)
	}

	return builder.String()
}
