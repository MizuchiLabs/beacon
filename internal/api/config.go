package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/danielgtaylor/huma/v2"

	"github.com/mizuchilabs/beacon/internal/incidents"
)

// Config holds the dashboard settings. Port and ChartType come from command
// flags, the rest from the environment.
type Config struct {
	Port        string `env:"BEACON_PORT"        envDefault:"3000"`
	ChartType   string `env:"BEACON_CHART_TYPE"  envDefault:"area"`
	Title       string `env:"BEACON_TITLE"       envDefault:"Beacon Dashboard"`
	Description string `env:"BEACON_DESCRIPTION" envDefault:"Track uptime and response times across all monitors"`
	Timezone    string `env:"BEACON_TIMEZONE"    envDefault:"Europe/Vienna"`
}

func loadConfig() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}

	if cfg.ChartType != "area" && cfg.ChartType != "bars" {
		return nil, fmt.Errorf("invalid chart type: %s", cfg.ChartType)
	}
	return &cfg, nil
}

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
	cfg       *Config
	incidents *incidents.IncidentManager
}

func NewConfigService(
	api huma.API,
	cfg *Config,
	inc *incidents.IncidentManager,
) *ConfigService {
	svc := &ConfigService{cfg: cfg, incidents: inc}
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

func (s *ConfigService) getConfig(_ context.Context, _ *struct{}) (*ConfigOutput, error) {
	return &ConfigOutput{Body: ConfigBody{
		Title:            s.cfg.Title,
		Description:      s.cfg.Description,
		Timezone:         s.cfg.Timezone,
		ChartType:        s.cfg.ChartType,
		IncidentsEnabled: s.incidents != nil,
	}}, nil
}
