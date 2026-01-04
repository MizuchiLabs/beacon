// Package checker provides functionality for checking HTTP endpoints.
package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mizuchilabs/beacon/internal/db"
)

type Checker struct {
	client *http.Client
}

func New(timeout time.Duration, insecure bool) *Checker {
	if timeout < 5*time.Second {
		timeout = 30 * time.Second
	}
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig:   &tls.Config{InsecureSkipVerify: insecure},
				DisableKeepAlives: true,
			},
		},
	}
}

func (c *Checker) Check(ctx context.Context, url string) *db.CreateCheckParams {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return checkErr(err)
	}
	req.Header.Set("User-Agent", "Beacon/1.0")

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
	return &db.CreateCheckParams{
		IsUp:         resp.StatusCode >= 200 && resp.StatusCode < 400,
		StatusCode:   &code,
		ResponseTime: &ms,
	}
}

func checkErr(err error, ms ...int64) *db.CreateCheckParams {
	msg := fmt.Sprintf("request failed: %v", err)
	p := &db.CreateCheckParams{IsUp: false, Error: &msg}
	if len(ms) > 0 {
		p.ResponseTime = &ms[0]
	}
	return p
}
