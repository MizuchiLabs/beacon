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
		Commands: []*cli.Command{
			{
				Name:   "generate",
				Usage:  "Generate random test data (only for development)",
				Hidden: true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := config.New(ctx, cmd)
					cfg.GenerateRandomData(ctx, true)
					return nil
				},
			},
		},
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
		Aliases:                         []string{},
		UsageText:                       "",
		ArgsUsage:                       "",
		DefaultCommand:                  "",
		Category:                        "",
		HideHelp:                        false,
		HideHelpCommand:                 false,
		HideVersion:                     false,
		ShellCompletionCommandName:      "",
		ShellComplete:                   nil,
		ConfigureShellCompletionCommand: nil,
		Before:                          nil,
		After:                           nil,
		CommandNotFound:                 nil,
		OnUsageError:                    nil,
		InvalidFlagAccessHandler:        nil,
		Hidden:                          false,
		Authors:                         []any{},
		Copyright:                       "",
		Reader:                          nil,
		Writer:                          nil,
		ErrWriter:                       nil,
		ExitErrHandler:                  nil,
		Metadata:                        map[string]interface{}{},
		ExtraInfo: func() map[string]string {
			panic("TODO")
		},
		CustomRootCommandHelpTemplate: "",
		SliceFlagSeparator:            "",
		DisableSliceFlagSeparator:     false,
		MapFlagKeyValueSeparator:      "",
		UseShortOptionHandling:        false,
		AllowExtFlags:                 false,
		SkipFlagParsing:               false,
		CustomHelpTemplate:            "",
		PrefixMatchCommands:           false,
		SuggestCommandFunc:            nil,
		MutuallyExclusiveFlags:        []cli.MutuallyExclusiveFlags{},
		Arguments:                     []cli.Argument{},
		ReadArgsFromStdin:             false,
		StopOnNthArg:                  new(int),
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := cmd.Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
