package log

import (
	"context"
	"log/slog"
	"sync"
)

type requestInfoKey struct{}

type requestInfo struct {
	mu    sync.Mutex
	attrs []slog.Attr
	index map[string]int
}

func newRequestInfo() *requestInfo {
	return &requestInfo{
		attrs: make([]slog.Attr, 0, 8),
		index: make(map[string]int, 8),
	}
}

func (r *requestInfo) add(attrs ...slog.Attr) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, attr := range attrs {
		if i, ok := r.index[attr.Key]; ok {
			r.attrs[i] = attr
			continue
		}

		r.index[attr.Key] = len(r.attrs)
		r.attrs = append(r.attrs, attr)
	}
}

func (r *requestInfo) snapshot() []slog.Attr {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]slog.Attr, len(r.attrs))
	copy(out, r.attrs)
	return out
}

// WithRequestInfo 向上下文注入请求级日志字段收集器。
func WithRequestInfo(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(requestInfoKey{}).(*requestInfo); ok {
		return ctx
	}

	return context.WithValue(ctx, requestInfoKey{}, newRequestInfo())
}

// AddInfo 向请求级收集器追加字段；同名 key 以后写入的值覆盖先前的值。
func AddInfo(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}

	ctx = WithRequestInfo(ctx)
	info, _ := ctx.Value(requestInfoKey{}).(*requestInfo)
	if info == nil {
		return ctx
	}

	info.add(attrs...)
	return ctx
}

// Fields 返回当前上下文中已收集字段的快照。
func Fields(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}

	info, _ := ctx.Value(requestInfoKey{}).(*requestInfo)
	if info == nil {
		return nil
	}

	return info.snapshot()
}

// Any 封装 slog.Any，便于调用方统一从本地 log 包构造字段。
func Any(key string, value any) slog.Attr {
	return slog.Any(key, value)
}

// String 封装 slog.String，便于调用方统一从本地 log 包构造字段。
func String(key, value string) slog.Attr {
	return slog.String(key, value)
}

// Int 封装 slog.Int，便于调用方统一从本地 log 包构造字段。
func Int(key string, value int) slog.Attr {
	return slog.Int(key, value)
}

// Int64 封装 slog.Int64，便于调用方统一从本地 log 包构造字段。
func Int64(key string, value int64) slog.Attr {
	return slog.Int64(key, value)
}

// Bool 封装 slog.Bool，便于调用方统一从本地 log 包构造字段。
func Bool(key string, value bool) slog.Attr {
	return slog.Bool(key, value)
}
