package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mizuchilabs/beacon/internal/incidents"
)

type GetIncidentsOutput struct {
	Body []incidents.Incident
}

type GetIncidentInput struct {
	ID string `path:"id" doc:"Incident ID"`
}

type GetIncidentOutput struct {
	Body incidents.Incident
}

// SyncIncidentsInput takes the token the way common git hosts send webhook
// secrets. It is never read from the query, which ends up in access logs.
type SyncIncidentsInput struct {
	Authorization string `header:"Authorization"       doc:"Bearer <token>, e.g. Gitea, Forgejo or curl"`
	GitlabToken   string `header:"X-Gitlab-Token"      doc:"GitLab webhook secret token"`
	GithubSig     string `header:"X-Hub-Signature-256" doc:"GitHub webhook signature, the secret is the token"`
	RawBody       []byte
}

type SyncIncidentsOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

type incidentHandlers struct {
	incidents *incidents.Service
}

func registerIncidents(api huma.API, inc *incidents.Service) {
	svc := &incidentHandlers{incidents: inc}
	huma.Register(api, huma.Operation{
		OperationID: "get-incidents",
		Method:      http.MethodGet,
		Path:        "/api/incidents",
		Summary:     "List incidents",
		Tags:        []string{"Incidents"},
	}, svc.getIncidents)
	huma.Register(api, huma.Operation{
		OperationID: "get-incident",
		Method:      http.MethodGet,
		Path:        "/api/incidents/{id}",
		Summary:     "Get an incident",
		Tags:        []string{"Incidents"},
	}, svc.getIncident)
	syncBody := &huma.RequestBody{}
	huma.Register(api, huma.Operation{
		OperationID:   "sync-incidents",
		Method:        http.MethodPost,
		Path:          "/api/incidents/sync",
		Summary:       "Reload incidents now",
		Description:   "Point a git push webhook here so updates show up without waiting for the sync interval. Needs BEACON_INCIDENT_SYNC_TOKEN as the webhook secret.",
		Tags:          []string{"Incidents"},
		DefaultStatus: http.StatusAccepted,
		RequestBody:   syncBody,
	}, svc.syncIncidents)
	// Huma always requires a RawBody. Only GitHub's signature needs it, other
	// senders may post none.
	syncBody.Required = false
}

func (s *incidentHandlers) getIncidents(
	_ context.Context,
	_ *struct{},
) (*GetIncidentsOutput, error) {
	if s.incidents == nil {
		return &GetIncidentsOutput{Body: []incidents.Incident{}}, nil
	}

	return &GetIncidentsOutput{Body: s.incidents.GetIncidents()}, nil
}

func (s *incidentHandlers) getIncident(
	_ context.Context,
	in *GetIncidentInput,
) (*GetIncidentOutput, error) {
	if s.incidents == nil {
		return nil, huma.Error404NotFound("incident not found")
	}

	incident, ok := s.incidents.GetIncident(in.ID)
	if !ok {
		return nil, huma.Error404NotFound("incident not found")
	}

	return &GetIncidentOutput{Body: *incident}, nil
}

func (s *incidentHandlers) syncIncidents(
	_ context.Context,
	in *SyncIncidentsInput,
) (*SyncIncidentsOutput, error) {
	if s.incidents == nil || s.incidents.SyncToken == "" {
		return nil, huma.Error404NotFound("incident sync is not enabled")
	}
	if !validSyncToken(in, s.incidents.SyncToken) {
		return nil, huma.Error401Unauthorized("invalid sync token")
	}

	s.incidents.Sync()
	out := &SyncIncidentsOutput{}
	out.Body.OK = true
	return out, nil
}

func validSyncToken(in *SyncIncidentsInput, token string) bool {
	if sig, ok := strings.CutPrefix(in.GithubSig, "sha256="); ok {
		mac := hmac.New(sha256.New, []byte(token))
		mac.Write(in.RawBody)
		want := hex.EncodeToString(mac.Sum(nil))
		return subtle.ConstantTimeCompare([]byte(sig), []byte(want)) == 1
	}
	got := in.GitlabToken
	if bearer, ok := strings.CutPrefix(in.Authorization, "Bearer "); ok {
		got = bearer
	}
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}
