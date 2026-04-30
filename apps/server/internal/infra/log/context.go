package log

import (
	"context"
	"log/slog"
	"time"

	"github.com/lifei6671/logit"
)

// WithRequestInfo 向上下文注入请求级日志字段收集器。
func WithRequestInfo(ctx context.Context) context.Context {
	return logit.WithContext(ctx)
}

// AddInfo 向请求级收集器追加字段；同名 key 以后写入的值覆盖先前的值。
func AddInfo(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}

	ctx = WithRequestInfo(ctx)
	logit.AddFields(ctx, attrsToFields(attrs)...)
	return ctx
}

// Fields 返回当前上下文中已收集字段的快照。
func Fields(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}

	attrs := make([]slog.Attr, 0, 8)
	logit.Range(ctx, func(field logit.Field) bool {
		attrs = append(attrs, fieldToAttr(field))
		return true
	})
	return attrs
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

func attrsToFields(attrs []slog.Attr) []logit.Field {
	fields := make([]logit.Field, 0, len(attrs))
	for _, attr := range attrs {
		fields = append(fields, attrToFields("", attr)...)
	}
	return fields
}

func attrToFields(prefix string, attr slog.Attr) []logit.Field {
	attr.Value = attr.Value.Resolve()
	key := groupedKey(prefix, attr.Key)

	switch attr.Value.Kind() {
	case slog.KindGroup:
		groupPrefix := key
		if groupPrefix == "" {
			groupPrefix = prefix
		}
		group := attr.Value.Group()
		fields := make([]logit.Field, 0, len(group))
		for _, item := range group {
			fields = append(fields, attrToFields(groupPrefix, item)...)
		}
		return fields
	case slog.KindString:
		return []logit.Field{logit.String(key, attr.Value.String())}
	case slog.KindInt64:
		return []logit.Field{logit.Int64(key, attr.Value.Int64())}
	case slog.KindUint64:
		return []logit.Field{logit.Uint64(key, attr.Value.Uint64())}
	case slog.KindFloat64:
		return []logit.Field{logit.Float64(key, attr.Value.Float64())}
	case slog.KindBool:
		return []logit.Field{logit.Bool(key, attr.Value.Bool())}
	case slog.KindDuration:
		return []logit.Field{logit.Duration(key, attr.Value.Duration())}
	case slog.KindTime:
		return []logit.Field{logit.Time(key, attr.Value.Time())}
	case slog.KindAny:
		if errValue, ok := attr.Value.Any().(error); ok {
			return []logit.Field{logit.Error(key, errValue)}
		}
		return []logit.Field{logit.Any(key, attr.Value.Any())}
	default:
		return []logit.Field{logit.Any(key, attr.Value.Any())}
	}
}

func groupedKey(prefix, key string) string {
	switch {
	case prefix == "":
		return key
	case key == "":
		return prefix
	default:
		return prefix + "." + key
	}
}

func fieldToAttr(field logit.Field) slog.Attr {
	switch field.Kind() {
	case logit.StringKind:
		return slog.String(field.Key(), field.Value().(string))
	case logit.BoolKind:
		return slog.Bool(field.Key(), field.Value().(bool))
	case logit.Int64Kind:
		return slog.Int64(field.Key(), field.Value().(int64))
	case logit.Uint64Kind:
		return slog.Uint64(field.Key(), field.Value().(uint64))
	case logit.Float64Kind:
		return slog.Float64(field.Key(), field.Value().(float64))
	case logit.DurationKind:
		return slog.Duration(field.Key(), field.Value().(time.Duration))
	case logit.TimeKind:
		return slog.Time(field.Key(), field.Value().(time.Time))
	case logit.BytesKind, logit.ErrorKind, logit.AnyKind:
		return slog.Any(field.Key(), field.Value())
	default:
		return slog.Any(field.Key(), field.Value())
	}
}
