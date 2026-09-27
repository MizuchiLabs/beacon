// Package notify sends monitor events to browser push subscribers and to the
// webhooks from the config.
package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/caarlos0/env/v11"

	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
)

const (
	// pushTTL keeps an undelivered alert at the push service long enough for
	// a sleeping device to still receive it.
	pushTTL     = 12 * 60 * 60
	sendTimeout = 10 * time.Second
)

var httpClient = &http.Client{Timeout: sendTimeout}

// Event kinds.
const (
	EventDown       = "down"
	EventUp         = "up"
	EventCertExpiry = "cert_expiry"
	EventTest       = "test"
)

// Event is what a webhook body template renders, and the JSON body when the
// webhook has no template.
type Event struct {
	Event   string    `json:"event"`
	Monitor string    `json:"monitor"`
	URL     string    `json:"url,omitempty"`
	Title   string    `json:"title"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

type Service struct {
	q         *db.Queries
	vapidKeys *db.VapidKey

	mu       sync.RWMutex
	webhooks []webhook

	Subscriber string `env:"BEACON_PUSH_SUBSCRIBER" envDefault:"mailto:beacon@mizuchi.dev"`
}

// pushPayload is what the service worker reads to show a notification.
type pushPayload struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	MonitorID int64  `json:"monitorId"`
}

func New(ctx context.Context, q *db.Queries) (*Service, error) {
	n, err := env.ParseAs[Service]()
	if err != nil {
		return nil, err
	}
	n.q = q

	count, err := q.VAPIDKeysExist(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking VAPID keys: %w", err)
	}
	if count == 0 {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			return nil, fmt.Errorf("generating VAPID keys: %w", err)
		}
		if err := q.CreateVAPIDKeys(ctx, &db.CreateVAPIDKeysParams{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		}); err != nil {
			return nil, fmt.Errorf("storing VAPID keys: %w", err)
		}
	}

	n.vapidKeys, err = q.GetVAPIDKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading VAPID keys: %w", err)
	}
	return &n, nil
}

// MonitorDown tells everyone a monitor went down.
func (n *Service) MonitorDown(ctx context.Context, m *db.Monitor, reason string) {
	n.send(ctx, m, Event{
		Event:   EventDown,
		Title:   fmt.Sprintf("🔴 %s is down", m.Name),
		Message: fmt.Sprintf("%s is down: %s", m.Name, reason),
	})
}

// MonitorUp tells everyone a monitor recovered.
func (n *Service) MonitorUp(ctx context.Context, m *db.Monitor) {
	n.send(ctx, m, Event{
		Event:   EventUp,
		Title:   fmt.Sprintf("✅ %s is back up", m.Name),
		Message: fmt.Sprintf("%s is responding normally again.", m.Name),
	})
}

// CertExpiring warns that a monitor's certificate expires soon.
func (n *Service) CertExpiring(ctx context.Context, m *db.Monitor, days int64) {
	n.send(ctx, m, Event{
		Event:   EventCertExpiry,
		Title:   fmt.Sprintf("⚠️ %s certificate expires soon", m.Name),
		Message: fmt.Sprintf("The certificate for %s expires in %d days.", m.Name, days),
	})
}

// Test sends a sample event to every webhook and returns the first failure.
func (n *Service) Test(ctx context.Context) error {
	event := Event{
		Event:   EventTest,
		Monitor: "Beacon",
		Title:   "🔔 Beacon test notification",
		Message: "If you can read this, your webhook works.",
		Time:    time.Now(),
	}
	for _, w := range n.hooks() {
		if err := n.sendWebhook(ctx, w, event); err != nil {
			return fmt.Errorf("webhook %s: %w", w.url, err)
		}
	}
	return nil
}

func (n *Service) send(ctx context.Context, m *db.Monitor, event Event) {
	event.Monitor = m.Name
	event.URL = PublicURL(m)
	event.Time = time.Now()

	for _, w := range n.hooks() {
		if err := n.sendWebhook(ctx, w, event); err != nil {
			slog.Error("Failed to send webhook", "url", w.url, "monitor", m.Name, "error", err)
		}
	}
	n.sendPush(ctx, m.ID, event)
}

// PublicURL is the monitor url that is safe to show. A push url holds the
// heartbeat token, so it stays private.
func PublicURL(m *db.Monitor) string {
	if m.Type == checker.TypePush {
		return ""
	}
	return m.Url
}

func (n *Service) sendPush(ctx context.Context, monitorID int64, event Event) {
	subs, err := n.q.GetPushSubscriptionsByMonitor(ctx, monitorID)
	if err != nil {
		slog.Error("Failed to load push subscriptions", "monitor_id", monitorID, "error", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	payload, err := json.Marshal(pushPayload{Title: event.Title, Body: event.Message, URL: "/", MonitorID: monitorID})
	if err != nil {
		slog.Error("Failed to marshal push payload", "error", err)
		return
	}

	for _, sub := range subs {
		status, err := n.pushTo(ctx, sub, payload)
		if err == nil {
			continue
		}
		slog.Error("Failed to send push notification", "monitor_id", monitorID, "subscription_id", sub.ID, "error", err)

		// 404/410 mean the endpoint is gone and will never succeed again
		if status == http.StatusNotFound || status == http.StatusGone {
			if err := n.q.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint); err != nil {
				slog.Error("Failed to delete invalid subscription", "error", err)
			}
		}
	}
}

// pushTo returns the HTTP status reported by the push service, or 0 if the
// request failed before a response was received.
func (n *Service) pushTo(ctx context.Context, sub *db.PushSubscription, payload []byte) (int, error) {
	resp, err := webpush.SendNotificationWithContext(
		ctx,
		payload,
		&webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dhKey, Auth: sub.AuthKey},
		},
		&webpush.Options{
			HTTPClient:      httpClient,
			Subscriber:      n.Subscriber,
			VAPIDPublicKey:  n.vapidKeys.PublicKey,
			VAPIDPrivateKey: n.vapidKeys.PrivateKey,
			TTL:             pushTTL,
		},
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return resp.StatusCode, fmt.Errorf("push service returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}
