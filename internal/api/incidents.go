package api

import (
	"context"
	"net/http"

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

type IncidentService struct {
	incidents *incidents.IncidentManager
}

func NewIncidentService(api huma.API, inc *incidents.IncidentManager) *IncidentService {
	svc := &IncidentService{incidents: inc}
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
	return svc
}

func (s *IncidentService) getIncidents(
	_ context.Context,
	_ *struct{},
) (*GetIncidentsOutput, error) {
	if s.incidents == nil {
		return nil, huma.Error404NotFound("incidents not configured")
	}

	return &GetIncidentsOutput{Body: s.incidents.GetIncidents()}, nil
}

func (s *IncidentService) getIncident(
	_ context.Context,
	in *GetIncidentInput,
) (*GetIncidentOutput, error) {
	if s.incidents == nil {
		return nil, huma.Error404NotFound("incidents not configured")
	}

	incident, ok := s.incidents.GetIncident(in.ID)
	if !ok {
		return nil, huma.Error404NotFound("incident not found")
	}

	return &GetIncidentOutput{Body: *incident}, nil
}
