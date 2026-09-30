package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/internal/incidents"
)

func incidentCommand() *cli.Command {
	return &cli.Command{
		Name:  "incident",
		Usage: "Write and check incident files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dir",
				Usage:   "Incident directory, defaults to <data-dir>/incidents",
				Sources: cli.EnvVars("BEACON_INCIDENT_PATH"),
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "new",
				Usage:     "Open a new incident or schedule maintenance",
				ArgsUsage: "<title>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "severity",
						Aliases: []string{"s"},
						Usage:   "critical, major, minor or maintenance",
						Value:   "minor",
					},
					&cli.StringSliceFlag{
						Name:    "monitor",
						Aliases: []string{"m"},
						Usage:   "Affected monitor, repeat for more, none means all",
					},
					&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "Longer description"},
					&cli.StringFlag{Name: "message", Usage: "First update, defaults to a generic line"},
					&cli.StringFlag{
						Name:  "at",
						Usage: "Start time (RFC 3339 or \"2006-01-02 15:04\"), defaults to now",
					},
					&cli.StringFlag{Name: "until", Usage: "Planned end of a maintenance window"},
				},
				Action: incidentNew,
			},
			{
				Name:      "update",
				Usage:     "Post an update on an incident",
				ArgsUsage: "<id>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "status",
						Aliases:  []string{"s"},
						Usage:    "investigating, identified, monitoring or resolved",
						Required: true,
					},
					&cli.StringFlag{Name: "message", Aliases: []string{"m"}, Required: true},
				},
				Action: incidentUpdate,
			},
			{
				Name:      "resolve",
				Usage:     "Resolve an incident",
				ArgsUsage: "<id>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "message", Aliases: []string{"m"}, Value: "This incident has been resolved."},
				},
				Action: incidentResolve,
			},
			{
				Name:   "lint",
				Usage:  "Check every incident file, and the monitor names against the config if there is one",
				Action: incidentLint,
			},
		},
	}
}

func incidentDir(cmd *cli.Command) string {
	if dir := cmd.String("dir"); dir != "" {
		return dir
	}
	return filepath.Join(cmd.String("data-dir"), "incidents")
}

func incidentNew(_ context.Context, cmd *cli.Command) error {
	title := strings.TrimSpace(strings.Join(cmd.Args().Slice(), " "))
	if title == "" {
		return errors.New("usage: beacon incident new <title>")
	}

	now := time.Now().UTC().Truncate(time.Second)
	started, err := parseTime(cmd.String("at"), now)
	if err != nil {
		return fmt.Errorf("--at: %w", err)
	}
	inc := incidents.Incident{
		Title:            title,
		Description:      cmd.String("description"),
		Severity:         cmd.String("severity"),
		AffectedMonitors: cmd.StringSlice("monitor"),
		StartedAt:        started,
	}
	if until := cmd.String("until"); until != "" {
		ends, err := parseTime(until, now)
		if err != nil {
			return fmt.Errorf("--until: %w", err)
		}
		inc.EndsAt = &ends
	}

	status, message := "investigating", "We are investigating this issue."
	if inc.Severity == "maintenance" {
		status, message = "scheduled", "Maintenance is scheduled."
	}
	if m := cmd.String("message"); m != "" {
		message = m
	}
	inc.Updates = []incidents.IncidentUpdate{{Message: message, Status: status, CreatedAt: now}}

	path, err := incidents.Create(incidentDir(cmd), inc)
	if err != nil {
		return err
	}
	fmt.Println("Created", path)
	return nil
}

func incidentUpdate(_ context.Context, cmd *cli.Command) error {
	return addUpdate(cmd, cmd.String("status"), cmd.String("message"))
}

func incidentResolve(_ context.Context, cmd *cli.Command) error {
	return addUpdate(cmd, "resolved", cmd.String("message"))
}

func addUpdate(cmd *cli.Command, status, message string) error {
	id := cmd.Args().First()
	if id == "" {
		return fmt.Errorf("usage: beacon incident %s <id>", cmd.Name)
	}
	path, err := incidents.AddUpdate(incidentDir(cmd), id, incidents.IncidentUpdate{
		Message:   message,
		Status:    status,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	})
	if err != nil {
		return err
	}
	fmt.Println("Updated", path)
	return nil
}

func incidentLint(_ context.Context, cmd *cli.Command) error {
	cfg, err := config.Load(cmd.String("config"))
	if err != nil {
		return err
	}
	var monitors []string
	if len(cfg.Monitors) > 0 {
		monitors = cfg.MonitorNames()
	} else {
		fmt.Println("No monitors configured, skipping the monitor name check")
	}

	dir := incidentDir(cmd)
	problems, err := incidents.Lint(dir, monitors)
	if err != nil {
		return err
	}
	for _, p := range problems {
		fmt.Println(p)
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d problem(s) in %s", len(problems), dir)
	}
	fmt.Println("All incident files in", dir, "look good")
	return nil
}

// parseTime reads RFC 3339, or a plain local date and time for typing by hand.
func parseTime(s string, fallback time.Time) (time.Time, error) {
	if s == "" {
		return fallback, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.ParseInLocation("2006-01-02 15:04", s, time.Local)
}
