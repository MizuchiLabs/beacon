package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/util"
)

type PushSubscriptionRequest struct {
	Endpoint string               `json:"endpoint"`
	Keys     PushSubscriptionKeys `json:"keys"`
}

type PushSubscriptionKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// GetVAPIDPublicKey returns the VAPID public key for client subscription
func (s *Server) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	keys, err := s.cfg.Conn.Queries.GetVAPIDKeys(r.Context())
	if err != nil {
		http.Error(w, "Failed to get VAPID public key", http.StatusInternalServerError)
		return
	}

	util.RespondJSON(w, http.StatusOK, map[string]string{
		"publicKey": keys.PublicKey,
	})
}

// SubscribeToPushNotifications subscribes a user to push notifications for a monitor
func (s *Server) SubscribeToPushNotifications(w http.ResponseWriter, r *http.Request) {
	monitorIDStr := chi.URLParam(r, "id")
	monitorID, err := strconv.ParseInt(monitorIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
		return
	}

	var req PushSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Store subscription
	err = s.cfg.Conn.Queries.CreatePushSubscription(r.Context(), &db.CreatePushSubscriptionParams{
		MonitorID: monitorID,
		Endpoint:  req.Endpoint,
		P256dhKey: req.Keys.P256dh,
		AuthKey:   req.Keys.Auth,
	})
	if err != nil {
		http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
		return
	}

	util.RespondJSON(w, http.StatusCreated, map[string]string{
		"message": "Subscribed successfully",
	})
}

// UnsubscribeFromPushNotifications unsubscribes a user from push notifications
func (s *Server) UnsubscribeFromPushNotifications(w http.ResponseWriter, r *http.Request) {
	monitorIDStr := chi.URLParam(r, "id")
	monitorID, err := strconv.ParseInt(monitorIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" {
		http.Error(w, "Missing endpoint", http.StatusBadRequest)
		return
	}

	err = s.cfg.Conn.Queries.DeletePushSubscription(r.Context(), &db.DeletePushSubscriptionParams{
		Endpoint:  req.Endpoint,
		MonitorID: monitorID,
	})
	if err != nil {
		http.Error(w, "Failed to unsubscribe", http.StatusInternalServerError)
		return
	}

	util.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Unsubscribed successfully",
	})
}
