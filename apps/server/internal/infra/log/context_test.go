package log

import (
	"context"
	"testing"
)

func TestAddInfoWithoutCollector(t *testing.T) {
	t.Parallel()

	ctx := AddInfo(context.Background(), String("module", "http"), Int("status", 200))
	fields := Fields(ctx)
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want %d", len(fields), 2)
	}
	if fields[0].Key != "module" {
		t.Fatalf("fields[0].Key = %q, want %q", fields[0].Key, "module")
	}
	if fields[1].Key != "status" {
		t.Fatalf("fields[1].Key = %q, want %q", fields[1].Key, "status")
	}
}

func TestAddInfoAppends(t *testing.T) {
	t.Parallel()

	ctx := WithRequestInfo(context.Background())
	ctx = AddInfo(ctx, String("request_id", "abc"))
	ctx = AddInfo(ctx, String("ts_code", "000001.SZ"))

	fields := Fields(ctx)
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want %d", len(fields), 2)
	}
	if fields[0].Key != "request_id" || fields[1].Key != "ts_code" {
		t.Fatalf("keys = [%q %q], want [request_id ts_code]", fields[0].Key, fields[1].Key)
	}
}

func TestAddInfoOverwriteByKey(t *testing.T) {
	t.Parallel()

	ctx := WithRequestInfo(context.Background())
	ctx = AddInfo(ctx, String("request_id", "first"))
	ctx = AddInfo(ctx, String("request_id", "second"), String("path", "/api/healthz"))

	fields := Fields(ctx)
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want %d", len(fields), 2)
	}
	if fields[0].Key != "request_id" {
		t.Fatalf("fields[0].Key = %q, want %q", fields[0].Key, "request_id")
	}
	if got := fields[0].Value.String(); got != "second" {
		t.Fatalf("fields[0].Value = %q, want %q", got, "second")
	}
	if fields[1].Key != "path" {
		t.Fatalf("fields[1].Key = %q, want %q", fields[1].Key, "path")
	}
}
