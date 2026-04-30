package log

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestNewLoggerIncludesContextFields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := newLogger(&buf)

	ctx := WithRequestInfo(context.Background())
	ctx = AddInfo(ctx,
		String("request_id", "req-1"),
		String("ts_code", "000001.SZ"),
	)

	logger.InfoContext(ctx, "probe", slog.String("event", "test"))

	output := buf.Bytes()
	if !bytes.Contains(output, []byte(`"request_id":"req-1"`)) {
		t.Fatalf("log output = %q, want request_id field", buf.String())
	}
	if !bytes.Contains(output, []byte(`"ts_code":"000001.SZ"`)) {
		t.Fatalf("log output = %q, want ts_code field", buf.String())
	}
	if !bytes.Contains(output, []byte(`"event":"test"`)) {
		t.Fatalf("log output = %q, want event field", buf.String())
	}
}
