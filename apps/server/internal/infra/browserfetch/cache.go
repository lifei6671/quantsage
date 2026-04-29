package browserfetch

import (
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

type cookieCache struct {
	mu    sync.RWMutex
	items map[string]cookieCacheItem
}

type cookieCacheItem struct {
	header string
	expiry time.Time
}

func newCookieCache() *cookieCache {
	return &cookieCache{
		items: make(map[string]cookieCacheItem),
	}
}

func (c *cookieCache) get(key string, now time.Time) (string, bool) {
	if c == nil {
		return "", false
	}

	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || now.After(item.expiry) {
		if ok {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return "", false
	}

	return item.header, true
}

func (c *cookieCache) set(key, header string, expiry time.Time) {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.items[key] = cookieCacheItem{
		header: header,
		expiry: expiry,
	}
	c.mu.Unlock()
}

func (c *cookieCache) invalidate() {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.items = make(map[string]cookieCacheItem)
	c.mu.Unlock()
}

func cookieCacheKey(cfg normalizedConfig, pageURL string) string {
	parts := []string{
		cfg.BrowserPath,
		normalizePageURL(pageURL),
		strings.ToLower(strings.TrimSpace(boolString(cfg.Headless))),
		cfg.UserAgentMode,
		cfg.UserAgent,
		cfg.UserAgentPlatform,
		cfg.AcceptLanguage,
		cfg.WaitReadySelector,
		boolString(cfg.DisableImages),
		strings.Join(cfg.BlockedURLPatterns, ","),
	}
	return strings.Join(parts, "\x00")
}

func normalizePageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	parsed.Fragment = ""
	if parsed.Scheme != "" {
		parsed.Scheme = strings.ToLower(parsed.Scheme)
	}
	if parsed.Host != "" {
		parsed.Host = strings.ToLower(parsed.Host)
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	} else {
		parsed.Path = cleanURLPath(parsed.Path)
	}
	if parsed.RawQuery != "" {
		parsed.RawQuery = parsed.Query().Encode()
	}

	return parsed.String()
}

func cleanURLPath(raw string) string {
	if raw == "" {
		return "/"
	}

	cleaned := path.Clean(raw)
	if cleaned == "." {
		cleaned = "/"
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}

	return cleaned
}

func boolString(v bool) string {
	if v {
		return "true"
	}

	return "false"
}
