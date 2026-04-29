package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	jobdomain "github.com/lifei6671/quantsage/apps/server/internal/domain/job"
)

const (
	dateLayout         = "2006-01-02"
	localAPIJobTimeout = 30 * time.Minute
)

type responseEnvelope[T any] struct {
	Code   int    `json:"code"`
	Errmsg string `json:"errmsg"`
	Toast  string `json:"toast"`
	Data   T      `json:"data"`
}

type runJobRequest struct {
	BizDate string `json:"biz_date"`
}

type runJobResponse struct {
	JobName string `json:"job_name"`
	Status  string `json:"status"`
}

// localAPIRunner 通过调用 server 的任务接口来触发本地 cron 任务。
// 这样 local 模式下只有 server 进程持有 sample runtime，worker 不会再维护第二份内存态。
type localAPIRunner struct {
	baseURL string
	client  *http.Client
}

func newLocalAPIRunner(serverAddr string) (jobdomain.Runner, error) {
	baseURL, err := resolveServerBaseURL(serverAddr)
	if err != nil {
		return nil, err
	}

	return &localAPIRunner{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: localAPIJobTimeout,
		},
	}, nil
}

func resolveServerBaseURL(serverAddr string) (string, error) {
	serverAddr = strings.TrimSpace(serverAddr)
	if serverAddr == "" {
		return "", errors.New("server addr is required")
	}

	if strings.HasPrefix(serverAddr, "http://") || strings.HasPrefix(serverAddr, "https://") {
		return strings.TrimRight(serverAddr, "/"), nil
	}

	host, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		return "", fmt.Errorf("parse server addr %q: %w", serverAddr, err)
	}

	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}

	return fmt.Sprintf("http://%s", net.JoinHostPort(host, port)), nil
}

func (r *localAPIRunner) Run(ctx context.Context, jobName string, bizDate time.Time) error {
	requestBody, err := json.Marshal(runJobRequest{
		BizDate: bizDate.UTC().Format(dateLayout),
	})
	if err != nil {
		return fmt.Errorf("marshal run job request: %w", err)
	}

	requestURL := fmt.Sprintf("%s/internal/jobs/%s/run", r.baseURL, url.PathEscape(jobName))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("build run job request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("call run job api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("run job api http %d and read response body: %w", resp.StatusCode, readErr)
		}
		return fmt.Errorf("run job api http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var envelope responseEnvelope[runJobResponse]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode run job response: %w", err)
	}
	if envelope.Code != 0 {
		message := envelope.Errmsg
		if message == "" {
			message = envelope.Toast
		}
		if message == "" {
			message = "run job request failed"
		}
		return fmt.Errorf("run job api business error %d: %s", envelope.Code, message)
	}

	return nil
}
