package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mizuchilabs/beacon/internal/scheduler"
)

type HeartbeatInput struct {
	Token  string `path:"token" doc:"Token from the push://<token> url of the monitor"`
	Status string `             doc:"Report a failure with down"                       query:"status" enum:"up,down" default:"up"`
	Msg    string `             doc:"Why the job failed, sent with the notification"   query:"msg"                                maxLength:"500"`
}

type HeartbeatOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

type heartbeatHandlers struct {
	sched *scheduler.Service
}

// registerHeartbeat lets cron jobs and scripts report in, e.g.
// curl -fsS https://status.example.com/api/push/<token>.
func registerHeartbeat(api huma.API, sched *scheduler.Service) {
	h := &heartbeatHandlers{sched: sched}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		huma.Register(api, huma.Operation{
			OperationID: "heartbeat-" + strings.ToLower(method),
			Method:      method,
			Path:        "/api/push/{token}",
			Summary:     "Send a heartbeat for a push monitor",
			Tags:        []string{"Heartbeat"},
		}, h.heartbeat)
	}
}

func (h *heartbeatHandlers) heartbeat(ctx context.Context, in *HeartbeatInput) (*HeartbeatOutput, error) {
	err := h.sched.Heartbeat(ctx, in.Token, in.Status != statusDown, in.Msg)
	if errors.Is(err, scheduler.ErrUnknownHeartbeat) {
		return nil, huma.Error404NotFound("no push monitor uses this token")
	}
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to record heartbeat")
	}

	out := &HeartbeatOutput{}
	out.Body.OK = true
	return out, nil
}
