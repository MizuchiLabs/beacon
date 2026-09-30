package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/urfave/cli/v3"

	"github.com/mizuchilabs/kata/buildinfo"
	"github.com/mizuchilabs/kata/logx"
	"github.com/mizuchilabs/kata/sigx"

	"github.com/mizuchilabs/beacon/internal/api"
	"github.com/mizuchilabs/beacon/internal/checker"
	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/internal/db"
	"github.com/mizuchilabs/beacon/internal/incidents"
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
				Name:   "test-notify",
				Usage:  "Send a test notification to every webhook in the config",
				Action: testNotify,
			},
			incidentCommand(),
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
				Name:    "port",
				Usage:   "Server port",
				Value:   "3000",
				Sources: cli.EnvVars("BEACON_PORT"),
			},
			&cli.StringFlag{
				Name:    "data-dir",
				Usage:   "Directory for the database and local incident files",
				Value:   "data",
				Sources: cli.EnvVars("BEACON_DATA_DIR"),
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
	dataDir, path := cmd.String("data-dir"), cmd.String("config")

	q, err := db.Open(ctx, dataDir)
	if err != nil {
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
	if err := sched.Start(ctx); err != nil {
		return err
	}

	inc, err := incidents.New(dataDir, notifier)
	if err != nil {
		return err
	}

	// apply runs on startup and again whenever the config file changes.
	apply := func(cfg *config.Config) error {
		if err := notifier.SetWebhooks(cfg.Webhooks); err != nil {
			return err
		}
		monitors, err := config.Sync(ctx, q, cfg.Monitors)
		if err != nil {
			return err
		}
		sched.Schedule(ctx, monitors)
		inc.SetMonitors(cfg.MonitorNames())
		return nil
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	if err := apply(cfg); err != nil {
		return err
	}
	go config.Watch(ctx, path, apply)
	inc.Start(ctx)

	server, err := api.New(q, inc, sched)
	if err != nil {
		return err
	}
	return server.Start(ctx, cmd.String("port"))
}

func testNotify(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load(cmd.String("config"))
	if err != nil {
		return err
	}
	if len(cfg.Webhooks) == 0 {
		return errors.New("no webhooks in the config")
	}

	notifier := &notify.Service{}
	if err := notifier.SetWebhooks(cfg.Webhooks); err != nil {
		return err
	}
	if err := notifier.Test(ctx); err != nil {
		return err
	}
	fmt.Printf("Sent a test notification to %d webhook(s)\n", len(cfg.Webhooks))
	return nil
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
