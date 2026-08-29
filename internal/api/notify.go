package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/internal/db"
)

// PushSubscriptionRequest is the Web Push subscription payload from the browser.
type PushSubscriptionRequest struct {
	Endpoint string               `json:"endpoint" format:"uri" doc:"Push service endpoint URL"`
	Keys     PushSubscriptionKeys `json:"keys"`
}

// PushSubscriptionKeys holds the browser-generated encryption keys.
type PushSubscriptionKeys struct {
	P256dh string `json:"p256dh" doc:"Client public key"`
	Auth   string `json:"auth"   doc:"Authentication secret"`
}

type SubscribeInput struct {
	ID   int64 `path:"id" doc:"Monitor ID"`
	Body PushSubscriptionRequest
}

type SubscribeOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type UnsubscribeInput struct {
	ID   int64 `path:"id" doc:"Monitor ID"`
	Body struct {
		Endpoint string `json:"endpoint" format:"uri" doc:"Push service endpoint URL to remove"`
	}
}

type UnsubscribeOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type VAPIDOutput struct {
	Body struct {
		PublicKey string `json:"publicKey" doc:"VAPID public key for push subscriptions"`
	}
}

type NotifyService struct {
	q *db.Queries
}

func NewNotifyService(api huma.API, cfg *config.Config) *NotifyService {
	svc := &NotifyService{q: cfg.Conn.Q}
	huma.Register(api, huma.Operation{
		OperationID: "get-vapid-public-key",
		Method:      http.MethodGet,
		Path:        "/api/vapid-public-key",
		Summary:     "Get VAPID public key",
		Tags:        []string{"Notifications"},
	}, svc.getVAPIDPublicKey)
	huma.Register(api, huma.Operation{
		OperationID:   "subscribe-to-monitor",
		Method:        http.MethodPost,
		Path:          "/api/monitors/{id}/subscribe",
		Summary:       "Subscribe to push notifications for a monitor",
		Tags:          []string{"Notifications"},
		DefaultStatus: http.StatusCreated,
	}, svc.subscribe)
	huma.Register(api, huma.Operation{
		OperationID: "unsubscribe-from-monitor",
		Method:      http.MethodPost,
		Path:        "/api/monitors/{id}/unsubscribe",
		Summary:     "Unsubscribe from push notifications for a monitor",
		Tags:        []string{"Notifications"},
	}, svc.unsubscribe)
	return svc
}

func (s *NotifyService) getVAPIDPublicKey(
	ctx context.Context,
	in *struct{},
) (*VAPIDOutput, error) {
	keys, err := s.q.GetVAPIDKeys(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get VAPID public key")
	}

	out := &VAPIDOutput{}
	out.Body.PublicKey = keys.PublicKey
	return out, nil
}

func (s *NotifyService) subscribe(
	ctx context.Context,
	in *SubscribeInput,
) (*SubscribeOutput, error) {
	if in.Body.Endpoint == "" || in.Body.Keys.P256dh == "" || in.Body.Keys.Auth == "" {
		return nil, huma.Error400BadRequest("missing required fields")
	}

	err := s.q.CreatePushSubscription(ctx, &db.CreatePushSubscriptionParams{
		MonitorID: in.ID,
		Endpoint:  in.Body.Endpoint,
		P256dhKey: in.Body.Keys.P256dh,
		AuthKey:   in.Body.Keys.Auth,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to subscribe")
	}

	out := &SubscribeOutput{}
	out.Body.Message = "Subscribed successfully"
	return out, nil
}

func (s *NotifyService) unsubscribe(
	ctx context.Context,
	in *UnsubscribeInput,
) (*UnsubscribeOutput, error) {
	if in.Body.Endpoint == "" {
		return nil, huma.Error400BadRequest("missing endpoint")
	}

	if err := s.q.DeletePushSubscription(ctx, &db.DeletePushSubscriptionParams{
		Endpoint:  in.Body.Endpoint,
		MonitorID: in.ID,
	}); err != nil {
		return nil, huma.Error500InternalServerError("failed to unsubscribe")
	}

	out := &UnsubscribeOutput{}
	out.Body.Message = "Unsubscribed successfully"
	return out, nil
}
