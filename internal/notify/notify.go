// Package notify provides functionality for sending notifications
package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mizuchilabs/beacon/internal/db"
)

type Notifier struct {
	q         *db.Queries
	vapidKeys *db.VapidKey
}

type NotificationPayload struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	MonitorID int64  `json:"monitorId"`
}

func New(ctx context.Context, q *db.Queries) *Notifier {
	result, err := q.VAPIDKeysExist(ctx)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to check VAPID keys: %w", err))
	}

	// Generate VAPID keys if missing
	if result == 0 {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			log.Fatal(fmt.Errorf("failed to generate VAPID keys: %w", err))
		}
		if err := q.CreateVAPIDKeys(ctx, &db.CreateVAPIDKeysParams{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		}); err != nil {
			log.Fatal(fmt.Errorf("failed to store VAPID keys: %w", err))
		}
	}

	vapidKeys, err := q.GetVAPIDKeys(ctx)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get VAPID keys: %w", err))
	}

	return &Notifier{
		q:         q,
		vapidKeys: vapidKeys,
	}
}

// SendMonitorNotification sends push notifications to all subscribers of a
// monitor after an up/down state transition.
func (n *Notifier) SendMonitorNotification(
	ctx context.Context,
	monitor *db.Monitor,
	up bool,
	reason string,
) error {
	if monitor == nil {
		return nil
	}

	subscriptions, err := n.q.GetPushSubscriptionsByMonitor(ctx, monitor.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		slog.Debug("No subscriptions found for monitor", "monitor_id", monitor.ID)
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

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	for _, sub := range subscriptions {
		status, err := n.sendPushNotification(sub, payloadBytes)
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
			if deleteErr := n.q.DeletePushSubscriptionByEndpoint(
				ctx,
				sub.Endpoint,
			); deleteErr != nil {
				slog.Error("Failed to delete invalid subscription", "error", deleteErr)
			}
		}
	}

	return nil
}

// sendPushNotification returns the HTTP status reported by the push service,
// or 0 if the request failed before a response was received.
func (n *Notifier) sendPushNotification(
	subscription *db.PushSubscription,
	payload []byte,
) (int, error) {
	sub := &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			P256dh: subscription.P256dhKey,
			Auth:   subscription.AuthKey,
		},
	}

	resp, err := webpush.SendNotification(payload, sub, &webpush.Options{
		Subscriber:      "mailto:beacon@mizuchi.dev", // Contact email for push notifications
		VAPIDPublicKey:  n.vapidKeys.PublicKey,
		VAPIDPrivateKey: n.vapidKeys.PrivateKey,
		TTL:             30, // Time to live in seconds
	})
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
