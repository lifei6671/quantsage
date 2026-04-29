package eastmoney

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

func TestClientGetHistoryReadsGzipResponse(t *testing.T) {
	t.Parallel()

	client := newClientWithHTTPClient(ClientConfig{
		Endpoint:      "https://hist.example.com",
		QuoteEndpoint: "https://quote.example.com",
		Timeout:       5 * time.Second,
		MaxRetries:    0,
		UserAgentMode: "stable",
	}, &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.URL.Path; got != historyKLinePath {
				t.Fatalf("request path = %q, want %q", got, historyKLinePath)
			}
			if got := r.URL.Query().Get("secid"); got != "0.000001" {
				t.Fatalf("secid = %q, want %q", got, "0.000001")
			}

			return newHTTPResponse(http.StatusOK, map[string]string{
				"Content-Encoding": "gzip",
				"Content-Type":     "application/json",
			}, gzipBytes(t, `{"data":{"code":"000001"}}`)), nil
		}),
	})

	body, err := client.GetHistory(context.Background(), historyKLinePath, url.Values{
		"secid": []string{"0.000001"},
	})
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if got := string(body); got != `{"data":{"code":"000001"}}` {
		t.Fatalf("GetHistory() body = %q, want %q", got, `{"data":{"code":"000001"}}`)
	}
}

func TestClientGetHistoryWrapsNon200Response(t *testing.T) {
	t.Parallel()

	client := newClientWithHTTPClient(ClientConfig{
		Endpoint:      "https://hist.example.com",
		QuoteEndpoint: "https://quote.example.com",
		Timeout:       5 * time.Second,
		MaxRetries:    0,
	}, &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return newHTTPResponse(http.StatusServiceUnavailable, map[string]string{
				"Content-Type": "text/plain",
			}, []byte("upstream failure")), nil
		}),
	})

	_, err := client.GetHistory(context.Background(), historyKLinePath, nil)
	if err == nil {
		t.Fatal("GetHistory() error = nil, want non-nil")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("CodeOf(error) = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
	if !strings.Contains(err.Error(), "http status 503") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "http status 503")
	}
}

func TestClientGetHistoryRejectsHTMLBotPage(t *testing.T) {
	t.Parallel()

	client := newClientWithHTTPClient(ClientConfig{
		Endpoint:      "https://hist.example.com",
		QuoteEndpoint: "https://quote.example.com",
		Timeout:       5 * time.Second,
		MaxRetries:    0,
	}, &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return newHTTPResponse(http.StatusOK, map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			}, []byte("<html><body>captcha verify</body></html>")), nil
		}),
	})

	_, err := client.GetHistory(context.Background(), historyKLinePath, nil)
	if err == nil {
		t.Fatal("GetHistory() error = nil, want non-nil")
	}
	if code := apperror.CodeOf(err); code != apperror.CodeDatasourceUnavailable {
		t.Fatalf("CodeOf(error) = %d, want %d", code, apperror.CodeDatasourceUnavailable)
	}
	if !strings.Contains(err.Error(), "html or anti-bot response") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "html or anti-bot response")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func gzipBytes(t *testing.T, payload string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	if _, err := gzipWriter.Write([]byte(payload)); err != nil {
		t.Fatalf("gzip write error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzip close error = %v", err)
	}

	return buffer.Bytes()
}

func newHTTPResponse(statusCode int, headers map[string]string, body []byte) *http.Response {
	header := make(http.Header, len(headers))
	for key, value := range headers {
		header.Set(key, value)
	}

	return &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func TestConvertTSCodeToSecID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		tsCode  string
		want    string
		wantErr error
	}{
		{name: "sz", tsCode: "000001.SZ", want: "0.000001"},
		{name: "sh", tsCode: "600000.SH", want: "1.600000"},
		{name: "bj", tsCode: "430001.BJ", want: "0.430001"},
		{name: "unsupported hk", tsCode: "000700.HK", wantErr: errors.New("unsupported eastmoney market")},
		{name: "invalid format", tsCode: "bad-code", wantErr: errors.New("unsupported eastmoney ts_code")},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ConvertTSCodeToSecID(tc.tsCode)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("ConvertTSCodeToSecID() error = nil, want contains %q", tc.wantErr.Error())
				}
				if !strings.Contains(err.Error(), tc.wantErr.Error()) {
					t.Fatalf("ConvertTSCodeToSecID() error = %q, want contains %q", err.Error(), tc.wantErr.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("ConvertTSCodeToSecID() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("ConvertTSCodeToSecID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapIntervalToEastMoneyKLT(t *testing.T) {
	t.Parallel()

	cases := []struct {
		interval Interval
		want     string
	}{
		{interval: Interval1Min, want: "1"},
		{interval: Interval5Min, want: "5"},
		{interval: Interval15Min, want: "15"},
		{interval: Interval30Min, want: "30"},
		{interval: Interval60Min, want: "60"},
		{interval: IntervalDay, want: "101"},
		{interval: IntervalWeek, want: "102"},
		{interval: IntervalMonth, want: "103"},
		{interval: IntervalQuarter, want: "104"},
		{interval: IntervalYear, want: "105"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.interval), func(t *testing.T) {
			t.Parallel()

			got, err := MapIntervalToEastMoneyKLT(tc.interval)
			if err != nil {
				t.Fatalf("MapIntervalToEastMoneyKLT() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("MapIntervalToEastMoneyKLT() = %q, want %q", got, tc.want)
			}
		})
	}

	if _, err := MapIntervalToEastMoneyKLT(Interval("2h")); err == nil {
		t.Fatal("MapIntervalToEastMoneyKLT() error = nil, want non-nil")
	}
}

func TestMapAdjustType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		adjust AdjustType
		want   string
	}{
		{name: "none", adjust: AdjustNone, want: "0"},
		{name: "qfq", adjust: AdjustQFQ, want: "1"},
		{name: "hfq", adjust: AdjustHFQ, want: "2"},
		{name: "unknown fallback", adjust: AdjustType("unexpected"), want: "0"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := MapAdjustType(tc.adjust); got != tc.want {
				t.Fatalf("MapAdjustType() = %q, want %q", got, tc.want)
			}
		})
	}
}
