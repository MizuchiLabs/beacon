package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		Version:               fmt.Sprintf("%s (%s)", Version, Commit),
		Usage:                 "beacon [command]",
		Description:           `Beacon is a simple uptime monitoring tool for websites.`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return nil
		},
		Commands: []*cli.Command{},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Print version information",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug logging",
				Sources: cli.EnvVars("DEBUG"),
			},
			&cli.StringFlag{
				Name:    "host",
				Aliases: []string{"h"},
				Usage:   "Server host",
				Value:   "0.0.0.0",
				Sources: cli.EnvVars("BEACON_HOST"),
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Server port",
				Value:   3000,
				Sources: cli.EnvVars("BEACON_PORT"),
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Aliases: []string{"t"},
				Usage:   "Checker timeout",
				Value:   30 * time.Second,
				Sources: cli.EnvVars("BEACON_TIMEOUT"),
			},
			&cli.IntFlag{
				Name:    "retention-days",
				Aliases: []string{"r"},
				Usage:   "Checker retention days",
				Value:   30,
				Sources: cli.EnvVars("BEACON_RETENTION_DAYS"),
			},
			&cli.BoolFlag{
				Name:    "insecure",
				Aliases: []string{"i"},
				Usage:   "Disable TLS certificate verification",
				Value:   false,
				Sources: cli.EnvVars("BEACON_INSECURE"),
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
