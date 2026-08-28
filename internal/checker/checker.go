// Package checker provides functionality for checking HTTP endpoints.
package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Checker struct {
	client *http.Client
}

// Result is the outcome of one HTTP check.
type Result struct {
	StatusCode   int64
	ResponseTime int64 // in ms
	Error        *string
	IsUp         bool
}

const (
	minTimeout     = 5 * time.Second
	defaultTimeout = 30 * time.Second
)

func New(timeout time.Duration, insecure bool) *Checker {
	if timeout < minTimeout {
		timeout = defaultTimeout
	}
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig:   &tls.Config{InsecureSkipVerify: insecure}, // #nosec G402
				DisableKeepAlives: true,
			},
		},
	}
}

func (c *Checker) Check(ctx context.Context, url string) Result {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return checkErr(err, 0)
	}
	req.Header.Set("User-Agent", "Beacon/1.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "close")

	resp, err := c.client.Do(req)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		return checkErr(err, ms)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	code := int64(resp.StatusCode)
	return Result{
		IsUp:         resp.StatusCode >= 200 && resp.StatusCode < 400,
		StatusCode:   code,
		ResponseTime: ms,
	}
}

func checkErr(err error, responseTime int64) Result {
	msg := fmt.Sprintf("request failed: %v", err)
	return Result{
		IsUp:         false,
		Error:        &msg,
		ResponseTime: responseTime,
	}
}
