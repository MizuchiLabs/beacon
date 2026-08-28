package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mizuchilabs/beacon/internal/config"
)

// ConfigOutput body mirrors the frontend Config interface.
type ConfigOutput struct {
	Body ConfigBody
}

// ConfigBody is the public dashboard configuration.
type ConfigBody struct {
	Title            string `json:"title"             doc:"Dashboard title"`
	Description      string `json:"description"       doc:"Dashboard description"`
	Timezone         string `json:"timezone"          doc:"IANA timezone for displaying timestamps"`
	ChartType        string `json:"chart_type"        doc:"Chart rendering style"                   enum:"area,bars"`
	IncidentsEnabled bool   `json:"incidents_enabled" doc:"Whether incident tracking is enabled"`
}

type ConfigService struct {
	cfg *config.Config
}

func NewConfigService(api huma.API, cfg *config.Config) *ConfigService {
	svc := &ConfigService{cfg: cfg}
	huma.Register(api, huma.Operation{
		OperationID: "get-config",
		Method:      http.MethodGet,
		Path:        "/api/config",
		Summary:     "Get dashboard config",
		Description: "Public configuration used by the dashboard frontend.",
		Tags:        []string{"Config"},
	}, svc.getConfig)
	return svc
}

func (s *ConfigService) getConfig(ctx context.Context, in *struct{}) (*ConfigOutput, error) {
	return &ConfigOutput{Body: ConfigBody{
		Title:            s.cfg.Title,
		Description:      s.cfg.Description,
		Timezone:         s.cfg.Timezone,
		ChartType:        s.cfg.ChartType,
		IncidentsEnabled: s.cfg.Incidents != nil,
	}}, nil
}
