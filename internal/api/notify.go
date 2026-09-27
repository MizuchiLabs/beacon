package api

import (
	"context"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/danielgtaylor/huma/v2"

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

type SubscriptionsInput struct {
	Body struct {
		Endpoint string `json:"endpoint" format:"uri" doc:"Push service endpoint URL of this browser"`
	}
}

type SubscriptionsOutput struct {
	Body struct {
		MonitorIDs []int64 `json:"monitor_ids" doc:"Monitors this endpoint is subscribed to"`
	}
}

type VAPIDOutput struct {
	Body struct {
		PublicKey string `json:"publicKey" doc:"VAPID public key for push subscriptions"`
	}
}

type notifyHandlers struct {
	q *db.Queries
}

func registerNotify(api huma.API, q *db.Queries) {
	svc := &notifyHandlers{q: q}
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
	huma.Register(api, huma.Operation{
		OperationID: "list-subscriptions",
		Method:      http.MethodPost,
		Path:        "/api/subscriptions",
		Summary:     "List the monitors a browser is subscribed to",
		Description: "POST so the endpoint stays out of access logs.",
		Tags:        []string{"Notifications"},
	}, svc.listSubscriptions)
}

func (s *notifyHandlers) getVAPIDPublicKey(
	ctx context.Context,
	_ *struct{},
) (*VAPIDOutput, error) {
	keys, err := s.q.GetVAPIDKeys(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get VAPID public key")
	}

	out := &VAPIDOutput{}
	out.Body.PublicKey = keys.PublicKey
	return out, nil
}

func (s *notifyHandlers) subscribe(
	ctx context.Context,
	in *SubscribeInput,
) (*SubscribeOutput, error) {
	if in.Body.Endpoint == "" || in.Body.Keys.P256dh == "" || in.Body.Keys.Auth == "" {
		return nil, huma.Error400BadRequest("missing required fields")
	}
	// The server POSTs to this url later, so only allow what a real push
	// service hands out.
	if u, err := url.Parse(in.Body.Endpoint); err != nil || u.Scheme != "https" || !publicHost(u.Hostname()) {
		return nil, huma.Error400BadRequest("endpoint must be a public https url")
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

func (s *notifyHandlers) unsubscribe(
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

func (s *notifyHandlers) listSubscriptions(
	ctx context.Context,
	in *SubscriptionsInput,
) (*SubscriptionsOutput, error) {
	ids, err := s.q.GetSubscribedMonitorIDs(ctx, in.Body.Endpoint)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load subscriptions")
	}

	out := &SubscriptionsOutput{}
	out.Body.MonitorIDs = append([]int64{}, ids...)
	return out, nil
}

// publicHost rejects localhost and private or loopback ip literals.
func publicHost(host string) bool {
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return false
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return true
	}
	return ip.IsGlobalUnicast() && !ip.IsPrivate()
}
