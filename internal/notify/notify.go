// Package notify provides functionality for sending notifications
package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/caarlos0/env/v11"

	"github.com/mizuchilabs/beacon/internal/db"
)

const (
	// pushTTL keeps an undelivered alert at the push service long enough for
	// a sleeping device to still receive it.
	pushTTL     = 12 * 60 * 60
	pushTimeout = 10 * time.Second
)

type Service struct {
	q         *db.Queries
	vapidKeys *db.VapidKey
	client    *http.Client

	Subscriber string `env:"BEACON_PUSH_SUBSCRIBER" envDefault:"mailto:beacon@mizuchi.dev"`
}

type NotificationPayload struct {
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

	result, err := q.VAPIDKeysExist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check VAPID keys: %w", err)
	}

	// Generate VAPID keys if missing
	if result == 0 {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			return nil, fmt.Errorf("failed to generate VAPID keys: %w", err)
		}
		if err := q.CreateVAPIDKeys(ctx, &db.CreateVAPIDKeysParams{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		}); err != nil {
			return nil, fmt.Errorf("failed to store VAPID keys: %w", err)
		}
	}

	n.vapidKeys, err = q.GetVAPIDKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get VAPID keys: %w", err)
	}
	n.client = &http.Client{Timeout: pushTimeout}

	return &n, nil
}

// SendMonitorNotification sends push notifications to all subscribers of a
// monitor after an up/down state transition.
func (n *Service) SendMonitorNotification(
	ctx context.Context,
	monitor *db.Monitor,
	up bool,
	reason string,
) error {
	if monitor == nil {
		return nil
	}

	payload := NotificationPayload{
		URL:       "/",
		MonitorID: monitor.ID,
	}
	if up {
		payload.Title = fmt.Sprintf("✅ %s is Back Up", monitor.Name)
		payload.Body = fmt.Sprintf("%s is now responding normally.", monitor.Url)
	} else {
		payload.Title = fmt.Sprintf("🔴 %s is Down", monitor.Name)
		payload.Body = fmt.Sprintf("%s is currently unreachable. Reason: %s", monitor.Url, reason)
	}

	return n.deliver(ctx, monitor, payload)
}

// SendCertificateExpiryNotification warns subscribers once a monitor's
// certificate is inside the expiry warning window.
func (n *Service) SendCertificateExpiryNotification(ctx context.Context, monitor *db.Monitor, days int64) error {
	if monitor == nil {
		return nil
	}

	payload := NotificationPayload{
		Title:     fmt.Sprintf("⚠️ %s certificate expires soon", monitor.Name),
		Body:      fmt.Sprintf("The certificate for %s expires in %d days.", monitor.Url, days),
		URL:       "/",
		MonitorID: monitor.ID,
	}

	return n.deliver(ctx, monitor, payload)
}

// deliver pushes one payload to every subscriber of the monitor, dropping
// subscriptions the push service reports as gone.
func (n *Service) deliver(ctx context.Context, monitor *db.Monitor, payload NotificationPayload) error {
	subscriptions, err := n.q.GetPushSubscriptionsByMonitor(ctx, monitor.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}
	if len(subscriptions) == 0 {
		slog.Debug("No subscriptions found for monitor", "monitor_id", monitor.ID)
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	for _, sub := range subscriptions {
		status, err := n.sendPushNotification(ctx, sub, payloadBytes)
		if err == nil {
			continue
		}

		slog.Error("Failed to send push notification",
			"monitor_id", monitor.ID,
			"subscription_id", sub.ID,
			"error", err,
		)

		// 404/410 mean the endpoint is gone and will never succeed again
		if status == http.StatusNotFound || status == http.StatusGone {
			if err = n.q.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint); err != nil {
				slog.Error("Failed to delete invalid subscription", "error", err)
			}
		}
	}

	return nil
}

// sendPushNotification returns the HTTP status reported by the push service,
// or 0 if the request failed before a response was received.
func (n *Service) sendPushNotification(
	ctx context.Context,
	subscription *db.PushSubscription,
	payload []byte,
) (int, error) {
	resp, err := webpush.SendNotificationWithContext(
		ctx,
		payload,
		&webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				P256dh: subscription.P256dhKey,
				Auth:   subscription.AuthKey,
			},
		},
		&webpush.Options{
			HTTPClient:      n.client,
			Subscriber:      n.Subscriber,
			VAPIDPublicKey:  n.vapidKeys.PublicKey,
			VAPIDPrivateKey: n.vapidKeys.PrivateKey,
			TTL:             pushTTL,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to send push: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return resp.StatusCode, fmt.Errorf("push service returned status %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}
