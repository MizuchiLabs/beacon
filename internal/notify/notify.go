// Package notify provides functionality for sending notifications
package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mizuchilabs/beacon/internal/db"
)

type Notifier struct {
	conn      *db.Connection
	vapidKeys *db.VapidKey
}

type NotificationPayload struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	MonitorID int64  `json:"monitorId"`
}

func New(ctx context.Context, conn *db.Connection) *Notifier {
	result, err := conn.Queries.VAPIDKeysExist(ctx)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to check VAPID keys: %w", err))
	}

	// Generate VAPID keys if missing
	if result == 0 {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			log.Fatal(fmt.Errorf("failed to generate VAPID keys: %w", err))
		}
		if err := conn.Queries.CreateVAPIDKeys(ctx, &db.CreateVAPIDKeysParams{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		}); err != nil {
			log.Fatal(fmt.Errorf("failed to store VAPID keys: %w", err))
		}
	}

	vapidKeys, err := conn.Queries.GetVAPIDKeys(ctx)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get VAPID keys: %w", err))
	}

	return &Notifier{
		conn:      conn,
		vapidKeys: vapidKeys,
	}
}

// SendMonitorDownNotification sends push notifications to all subscribers when a monitor goes down
func (n *Notifier) SendMonitorDownNotification(
	ctx context.Context,
	monitor *db.Monitor,
	reason string,
) error {
	if monitor == nil {
		return nil
	}

	// Get all subscriptions for this monitor
	subscriptions, err := n.conn.Queries.GetPushSubscriptionsByMonitor(ctx, monitor.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		slog.Debug("No subscriptions found for monitor", "monitor_id", monitor.ID)
		return nil
	}

	// Create notification payload
	payload := NotificationPayload{
		Title:     fmt.Sprintf("🔴 %s is Down", monitor.Name),
		Body:      fmt.Sprintf("%s is currently unreachable. Reason: %s", monitor.Url, reason),
		URL:       "/", // Could be a link to specific monitor page
		MonitorID: monitor.ID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Send to all subscribers
	for _, sub := range subscriptions {
		if err := n.sendPushNotification(sub, payloadBytes); err != nil {
			slog.Error("Failed to send push notification",
				"monitor_id", monitor.ID,
				"subscription_id", sub.ID,
				"error", err,
			)

			if isSubscriptionError(err) {
				if deleteErr := n.conn.Queries.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint); deleteErr != nil {
					slog.Error("Failed to delete invalid subscription", "error", deleteErr)
				}
			}
		}
	}

	return nil
}

// SendMonitorUpNotification sends notifications when a monitor comes back up
func (n *Notifier) SendMonitorUpNotification(ctx context.Context, monitor *db.Monitor) error {
	if monitor == nil {
		return nil
	}
	subscriptions, err := n.conn.Queries.GetPushSubscriptionsByMonitor(ctx, monitor.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		return nil
	}

	payload := NotificationPayload{
		Title:     fmt.Sprintf("✅ %s is Back Up", monitor.Name),
		Body:      fmt.Sprintf("%s is now responding normally.", monitor.Url),
		URL:       "/",
		MonitorID: monitor.ID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	for _, sub := range subscriptions {
		if err := n.sendPushNotification(sub, payloadBytes); err != nil {
			slog.Error("Failed to send recovery notification", "error", err)

			if isSubscriptionError(err) {
				if err := n.conn.Queries.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint); err != nil {
					slog.Error("Failed to delete invalid subscription", "error", err)
				}
			}
		}
	}

	return nil
}

func (n *Notifier) sendPushNotification(
	subscription *db.PushSubscription,
	payload []byte,
) error {
	// Create push subscription object
	sub := &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			P256dh: subscription.P256dhKey,
			Auth:   subscription.AuthKey,
		},
	}

	// Send the notification
	resp, err := webpush.SendNotification(payload, sub, &webpush.Options{
		Subscriber:      "mailto:beacon@mizuchi.dev", // Contact email for push notifications
		VAPIDPublicKey:  n.vapidKeys.PublicKey,
		VAPIDPrivateKey: n.vapidKeys.PrivateKey,
		TTL:             30, // Time to live in seconds
	})
	if err != nil {
		return fmt.Errorf("failed to send push: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close response body", "error", err)
		}
	}()

	// Check response status
	if resp.StatusCode != 201 {
		return fmt.Errorf("push service returned status %d", resp.StatusCode)
	}

	return nil
}

// isSubscriptionError checks if the error indicates an invalid/expired subscription
func isSubscriptionError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "410") ||
		strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "expired") ||
		strings.Contains(errStr, "invalid")
}
