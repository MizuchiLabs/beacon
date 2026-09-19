package api

import (
	"context"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/danielgtaylor/huma/v2"
)

// ConfigOutput body mirrors the frontend Config interface.
type ConfigOutput struct {
	Body ConfigBody
}

// ConfigBody is the public dashboard configuration.
type ConfigBody struct {
	Title   string `json:"title"              doc:"Dashboard title"`
	LogoURL string `json:"logo_url,omitempty" doc:"URL of a custom logo image shown in the header"`
}

type ConfigService struct {
	Title   string `env:"BEACON_TITLE"    envDefault:"Beacon"`
	LogoURL string `env:"BEACON_LOGO_URL"`
}

func NewConfigService(api huma.API) error {
	cfg, err := env.ParseAs[ConfigService]()
	if err != nil {
		return err
	}
	huma.Register(api, huma.Operation{
		OperationID: "get-config",
		Method:      http.MethodGet,
		Path:        "/api/config",
		Summary:     "Get dashboard config",
		Description: "Public configuration used by the dashboard frontend.",
		Tags:        []string{"Config"},
	}, cfg.getConfig)
	return nil
}

func (s *ConfigService) getConfig(_ context.Context, _ *struct{}) (*ConfigOutput, error) {
	return &ConfigOutput{Body: ConfigBody{Title: s.Title, LogoURL: s.LogoURL}}, nil
}
