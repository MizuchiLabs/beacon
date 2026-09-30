package incidents

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Valid values for enums.
var (
	ValidSeverities = []string{"critical", "major", "minor", "maintenance"}
	ValidStatuses   = []string{"scheduled", "investigating", "identified", "monitoring", "resolved"}
)

type Incident struct {
	ID               string           `yaml:"id,omitempty"                json:"id"`
	Title            string           `yaml:"title"                       json:"title"`
	Description      string           `yaml:"description,omitempty"       json:"description,omitempty"`
	Severity         string           `yaml:"severity"                    json:"severity"                    enum:"critical,major,minor,maintenance"`
	Status           string           `yaml:"status,omitempty"            json:"status"                      enum:"scheduled,investigating,identified,monitoring,resolved" doc:"Taken from the latest update unless the file sets it"`
	AffectedMonitors []string         `yaml:"affected_monitors,omitempty" json:"affected_monitors,omitempty"                                                               doc:"Monitor names, empty means every monitor"`
	StartedAt        time.Time        `yaml:"started_at"                  json:"started_at"`
	EndsAt           *time.Time       `yaml:"ends_at,omitempty"           json:"ends_at,omitempty"                                                                         doc:"Planned end of a maintenance window, it counts as resolved afterwards"`
	ResolvedAt       *time.Time       `yaml:"resolved_at,omitempty"       json:"resolved_at,omitempty"                                                                     doc:"Taken from the resolved update unless the file sets it"`
	Updates          []IncidentUpdate `yaml:"updates,omitempty"           json:"updates,omitempty"                                                                         doc:"Oldest first"`

	file string
}

type IncidentUpdate struct {
	Message   string    `yaml:"message"    json:"message"`
	Status    string    `yaml:"status"     json:"status"     enum:"scheduled,investigating,identified,monitoring,resolved"`
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
}

// ParseIncidentsDir reads all .yaml or .yml files from a directory. Files that
// fail to parse are logged and skipped.
func ParseIncidentsDir(dirPath string) ([]Incident, error) {
	incidents, errs, err := parseDir(dirPath)
	for _, err := range errs {
		slog.Warn("Skipping incident file", "error", err)
	}
	return incidents, err
}

// parseDir returns the valid incidents, newest first, and one error per file
// that could not be used.
func parseDir(dirPath string) ([]Incident, []error, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, nil, err
	}

	var incidents []Incident
	var errs []error
	for _, entry := range entries {
		if entry.IsDir() || !isIncidentFile(entry.Name()) {
			continue
		}

		incident, err := parseIncident(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", entry.Name(), err))
			continue
		}
		incidents = append(incidents, incident)
	}

	seen := map[string]string{}
	incidents = slices.DeleteFunc(incidents, func(inc Incident) bool {
		if other, ok := seen[inc.ID]; ok {
			errs = append(errs, fmt.Errorf("%s: duplicate id %q, already used by %s", inc.file, inc.ID, other))
			return true
		}
		seen[inc.ID] = inc.file
		return false
	})

	slices.SortFunc(incidents, func(a, b Incident) int {
		return b.StartedAt.Compare(a.StartedAt)
	})
	return incidents, errs, nil
}

func isIncidentFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

func parseIncident(path string) (Incident, error) {
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return Incident{}, err
	}
	return parseBytes(data, path)
}

func parseBytes(data []byte, path string) (Incident, error) {
	var incident Incident
	if err := yaml.Unmarshal(data, &incident); err != nil {
		return incident, err
	}
	incident.file = filepath.Base(path)
	if incident.ID == "" {
		incident.ID = strings.TrimSuffix(incident.file, filepath.Ext(incident.file))
	}
	if err := incident.Validate(); err != nil {
		return incident, err
	}
	incident.derive()
	return incident, nil
}

// Validate checks required fields and enum values.
func (i *Incident) Validate() error {
	if i.Title == "" {
		return errors.New("missing title")
	}
	if i.StartedAt.IsZero() {
		return errors.New("missing started_at")
	}
	if !slices.Contains(ValidSeverities, i.Severity) {
		return fmt.Errorf("invalid severity '%s': must be one of %v", i.Severity, ValidSeverities)
	}
	if i.Status != "" {
		if err := validateStatus("incident", i.Status); err != nil {
			return err
		}
	}
	if i.EndsAt != nil && !i.EndsAt.After(i.StartedAt) {
		return errors.New("ends_at must be after started_at")
	}
	for _, update := range i.Updates {
		if err := validateStatus(fmt.Sprintf("update %q", update.Message), update.Status); err != nil {
			return err
		}
		if update.CreatedAt.IsZero() {
			return fmt.Errorf("update %q: missing created_at", update.Message)
		}
	}
	return nil
}

func validateStatus(what, status string) error {
	if !slices.Contains(ValidStatuses, status) {
		return fmt.Errorf("%s: invalid status '%s': must be one of %v", what, status, ValidStatuses)
	}
	return nil
}

// derive fills status and resolved_at from the updates, so writing an
// incident is just appending updates. Fields set in the file win.
func (i *Incident) derive() {
	slices.SortStableFunc(i.Updates, func(a, b IncidentUpdate) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	if i.Status == "" {
		switch {
		case len(i.Updates) > 0:
			i.Status = i.Updates[len(i.Updates)-1].Status
		case i.Severity == "maintenance":
			i.Status = "scheduled"
		default:
			i.Status = "investigating"
		}
	}

	if i.ResolvedAt == nil && i.Status == "resolved" {
		for _, u := range slices.Backward(i.Updates) {
			if u.Status == "resolved" {
				i.ResolvedAt = new(u.CreatedAt)
				break
			}
		}
	}
}
