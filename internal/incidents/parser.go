package incidents

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// Valid values for enums
var (
	ValidSeverities = []string{"critical", "major", "minor", "maintenance"}
	ValidStatuses   = []string{"investigating", "identified", "monitoring", "resolved"}
)

type Incident struct {
	ID               string           `yaml:"id"                          json:"id,omitempty"`
	Title            string           `yaml:"title"                       json:"title,omitempty"`
	Description      string           `yaml:"description"                 json:"description,omitempty"`
	Severity         string           `yaml:"severity"                    json:"severity,omitempty"`
	Status           string           `yaml:"status"                      json:"status,omitempty"`
	AffectedMonitors []string         `yaml:"affected_monitors,omitempty" json:"affected_monitors,omitempty"`
	StartedAt        time.Time        `yaml:"started_at"                  json:"started_at"`
	ResolvedAt       *time.Time       `yaml:"resolved_at,omitempty"       json:"resolved_at,omitempty"`
	Updates          []IncidentUpdate `yaml:"updates"                     json:"updates,omitempty"`
}

type IncidentUpdate struct {
	Message   string    `yaml:"message"    json:"message"`
	Status    string    `yaml:"status"     json:"status"`
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
}

// ParseIncidentsDir reads all .yaml or .yml files from a directory
func ParseIncidentsDir(dirPath string) ([]Incident, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var incidents []Incident
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			continue
		}

		var incident Incident
		if err := yaml.Unmarshal(data, &incident); err != nil {
			continue
		}

		// Validate the incident
		if err := incident.Validate(); err != nil {
			continue
		}

		incidents = append(incidents, incident)
	}

	// Sort by started_at descending (most recent first)
	sort.Slice(incidents, func(i, j int) bool {
		return incidents[i].StartedAt.After(incidents[j].StartedAt)
	})

	return incidents, nil
}

// Validate checks if the incident has valid enum values
func (i *Incident) Validate() error {
	if !slices.Contains(ValidSeverities, i.Severity) {
		return fmt.Errorf("invalid severity '%s': must be one of %v", i.Severity, ValidSeverities)
	}
	if !slices.Contains(ValidStatuses, i.Status) {
		return fmt.Errorf("invalid status '%s': must be one of %v", i.Status, ValidStatuses)
	}
	for _, update := range i.Updates {
		if !slices.Contains(ValidStatuses, update.Status) {
			return fmt.Errorf(
				"invalid update status '%s': must be one of %v",
				update.Status,
				ValidStatuses,
			)
		}
	}
	return nil
}
