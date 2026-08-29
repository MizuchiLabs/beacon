package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mizuchilabs/beacon/internal/api"
	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/kata/buildinfo"
	"github.com/mizuchilabs/kata/logx"
	"github.com/mizuchilabs/kata/sigx"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		EnableShellCompletion: true,
		Suggest:               true,
		Name:                  "beacon",
		Version:               buildinfo.String(),
		Usage:                 "monitoring your websites",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			logx.Init(cmd.Bool("debug"))
			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := config.New(ctx, cmd)
			if err != nil {
				return err
			}
			return api.NewServer(ctx, cfg).Start()
		},
		Commands: []*cli.Command{
			{
				Name:    "openapi",
				Aliases: []string{"o"},
				Usage:   "Generate the OpenAPI spec without starting the server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file path; .yaml/.yml writes YAML, anything else JSON",
						Value:   "web/openapi.json",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) (err error) {
					srv := api.NewServer(ctx, &config.Config{})

					out := cmd.String("output")
					var b []byte
					if ext := filepath.Ext(out); ext == ".yaml" || ext == ".yml" {
						b, err = srv.OpenAPI().YAML()
					} else {
						b, err = srv.OpenAPI().MarshalJSON()
					}
					if err != nil {
						return err
					}

					if err := os.WriteFile(out, b, 0o644); err != nil {
						return fmt.Errorf("writing spec: %w", err)
					}
					slog.Info("OpenAPI spec written", "path", out)
					return nil
				},
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug logging",
				Sources: cli.EnvVars("BEACON_DEBUG"),
			},
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Server port",
				Value:   "3000",
				Sources: cli.EnvVars("BEACON_PORT"),
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to monitors config file",
				Value:   "config.yaml",
				Sources: cli.EnvVars("BEACON_CONFIG"),
			},
			&cli.StringFlag{
				Name:    "chart-type",
				Aliases: []string{"t"},
				Usage:   "Chart type (bars or area)",
				Value:   "area",
				Sources: cli.EnvVars("BEACON_CHART_TYPE"),
			},
		},
	}

	if err := cmd.Run(sigx.NotifyContext(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "beacon: %v\n", err)
		os.Exit(1)
	}
}
