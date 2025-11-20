package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mizuchilabs/beacon/internal/api"
	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/urfave/cli/v3"
)

var (
	Version = "unknown"
	Commit  string
	Date    string
	Dirty   string
)

func main() {
	cmd := &cli.Command{
		EnableShellCompletion: true,
		Suggest:               true,
		Name:                  "beacon",
		Version:               Version,
		Usage:                 "beacon [command]",
		Description:           `Beacon is a simple uptime monitoring tool for websites.`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config.New(ctx, cmd)
			return api.NewServer(cfg).Start(ctx)
		},
		Commands: []*cli.Command{},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Print version information",
			},
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Server port",
				Sources: cli.EnvVars("BEACON_PORT"),
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to monitors config file",
				Sources: cli.EnvVars("BEACON_CONFIG"),
			},
		},
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := cmd.Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
