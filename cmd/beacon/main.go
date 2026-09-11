package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/mizuchilabs/kata/buildinfo"
	"github.com/mizuchilabs/kata/logx"
	"github.com/mizuchilabs/kata/sigx"
	"github.com/urfave/cli/v3"

	"github.com/mizuchilabs/beacon/internal/api"
	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/incidents"
	"github.com/mizuchilabs/beacon/internal/monitors"
	"github.com/mizuchilabs/beacon/internal/notify"
	"github.com/mizuchilabs/beacon/internal/scheduler"
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
		Action: run,
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
				Action: openapi,
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
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to monitors config file",
				Value:   "config.yaml",
				Sources: cli.EnvVars("BEACON_CONFIG"),
			},
		},
	}

	if err := cmd.Run(sigx.NotifyContext(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", cmd.Name, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	q, err := db.Open(ctx)
	if err != nil {
		return err
	}

	// Sync monitors to DB before starting background jobs
	if err := monitors.Sync(ctx, q, cmd.String("config")); err != nil {
		return err
	}

	chk, err := checker.New()
	if err != nil {
		return err
	}

	notifier, err := notify.New(ctx, q)
	if err != nil {
		return err
	}

	sched, err := scheduler.New(q, chk, notifier)
	if err != nil {
		return err
	}
	sched.Start(ctx)

	inc, err := incidents.New()
	if err != nil {
		return err
	}
	inc.Start(ctx)

	server, err := api.NewServer(ctx, q, inc)
	if err != nil {
		return err
	}
	return server.Start()
}

func openapi(_ context.Context, cmd *cli.Command) error {
	spec := api.Spec()

	out := cmd.String("output")
	var b []byte
	var err error
	if ext := filepath.Ext(out); ext == ".yaml" || ext == ".yml" {
		b, err = spec.YAML()
	} else {
		b, err = spec.MarshalJSON()
	}
	if err != nil {
		return err
	}

	if err := os.WriteFile(out, b, 0o600); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	slog.Info("OpenAPI spec written", "path", out)
	return nil
}
