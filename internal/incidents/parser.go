package incidents

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Incident struct {
	ID               string           `yaml:"id"`
	Title            string           `yaml:"title"`
	Description      string           `yaml:"description"`
	Severity         string           `yaml:"severity"` // critical, major, minor, maintenance
	Status           string           `yaml:"status"`   // investigating, identified, monitoring, resolved
	AffectedMonitors []string         `yaml:"affected_monitors,omitempty"`
	StartedAt        time.Time        `yaml:"started_at"`
	ResolvedAt       *time.Time       `yaml:"resolved_at,omitempty"`
	Updates          []IncidentUpdate `yaml:"updates"`
}

type IncidentUpdate struct {
	Message   string    `yaml:"message"`
	Status    string    `yaml:"status"`
	CreatedAt time.Time `yaml:"created_at"`
}

// ParseIncidentsDir reads all .yaml files from a directory
func ParseIncidentsDir(dirPath string) ([]Incident, error) {
	var incidents []Incident

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
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

		incidents = append(incidents, incident)
	}

	return incidents, nil
}
