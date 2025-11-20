// Package checker provides functionality for checking HTTP endpoints.
package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/mizuchilabs/beacon/internal/db"
)

type Result struct {
	IsUp           bool
	StatusCode     int
	ResponseTimeMs int64
	Error          error
}

type Checker struct {
	client  *http.Client
	Timeout time.Duration `env:"BEACON_TIMEOUT" envDefault:"30s"`
}

func New(insecure bool) *Checker {
	c, err := env.ParseAs[Checker]()
	if err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	// Set a sane default timeout
	if c.Timeout <= time.Second*5 {
		c.Timeout = 30 * time.Second
	}

	return &Checker{
		client: &http.Client{
			Timeout: c.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
				DisableKeepAlives:   false,
				DisableCompression:  false,
				ForceAttemptHTTP2:   true,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: insecure},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (c *Checker) Check(ctx context.Context, url string) *db.CreateCheckParams {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		msg := fmt.Sprintf("failed to create request: %v", err)
		return &db.CreateCheckParams{
			IsUp:  false,
			Error: &msg,
		}
	}

	// Set user agent
	req.Header.Set("User-Agent", "Beacon-Uptime-Monitor/1.0")

	resp, err := c.client.Do(req)
	responseTime := time.Since(start).Milliseconds()
	if err != nil {
		msg := fmt.Sprintf("failed to execute request: %v", err)
		return &db.CreateCheckParams{
			IsUp:         false,
			Error:        &msg,
			ResponseTime: &responseTime,
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()

	// Consider 2xx and 3xx as up
	isUp := resp.StatusCode >= 200 && resp.StatusCode < 400
	statusCode := int64(resp.StatusCode)
	return &db.CreateCheckParams{
		IsUp:         isUp,
		StatusCode:   &statusCode,
		ResponseTime: &responseTime,
		Error:        nil,
	}
}

func (c *Checker) Close() {
	c.client.CloseIdleConnections()
}
