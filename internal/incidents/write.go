package incidents

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

const schemaURL = "https://raw.githubusercontent.com/mizuchilabs/beacon/main/incident.schema.json"

// Create writes a new incident file named after its start date and title and
// returns its path.
func Create(dir string, inc Incident) (string, error) {
	if err := inc.Validate(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}

	path := filepath.Join(dir, inc.StartedAt.Format("2006-01-02")+"-"+slug(inc.Title)+".yaml")
	var buf bytes.Buffer
	buf.WriteString("# yaml-language-server: $schema=" + schemaURL + "\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(inc); err != nil {
		return "", err
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304
	if errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("%s already exists", path)
	}
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(buf.Bytes())
	return path, err
}

// AddUpdate appends an update to the incident with the given id and returns
// the file it changed. The file is edited in place, so comments survive. A
// status or resolved_at the file sets by hand is kept in step.
func AddUpdate(dir, id string, u IncidentUpdate) (string, error) {
	if err := validateStatus("update", u.Status); err != nil {
		return "", err
	}
	path, err := findFile(dir, id)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return "", err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("%s: not a yaml mapping", path)
	}
	root := doc.Content[0]

	var update yaml.Node
	if err := update.Encode(u); err != nil {
		return "", err
	}
	updates := mappingValue(root, "updates")
	if updates == nil {
		updates = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "updates"}, updates)
	}
	updates.Content = append(updates.Content, &update)

	if status := mappingValue(root, "status"); status != nil {
		status.Value = u.Status
	}
	if resolved := mappingValue(root, "resolved_at"); resolved != nil && u.Status == "resolved" {
		resolved.Value = u.CreatedAt.UTC().Format(time.RFC3339)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return "", err
	}
	if _, err := parseBytes(buf.Bytes(), path); err != nil {
		return "", fmt.Errorf("%s would become invalid: %w", path, err)
	}
	return path, os.WriteFile(path, buf.Bytes(), 0o600)
}

// Lint returns every problem in the incident directory. Unknown monitor
// names are only checked when monitors is not nil.
func Lint(dir string, monitors []string) ([]error, error) {
	incidents, errs, err := parseDir(dir)
	if err != nil {
		return nil, err
	}
	if monitors == nil {
		return errs, nil
	}
	for _, inc := range incidents {
		for _, name := range UnknownMonitors(inc, monitors) {
			errs = append(errs, fmt.Errorf("%s: unknown monitor %q", inc.file, name))
		}
	}
	return errs, nil
}

// findFile looks for <id>.yaml or <id>.yml first, then for a file that sets
// the id itself.
func findFile(dir, id string) (string, error) {
	for _, ext := range []string{".yaml", ".yml"} {
		path := filepath.Join(dir, id+ext)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	incidents, _, err := parseDir(dir)
	if err != nil {
		return "", err
	}
	for _, inc := range incidents {
		if inc.ID == id {
			return filepath.Join(dir, inc.file), nil
		}
	}
	return "", fmt.Errorf("no incident with id %q in %s", id, dir)
}

func mappingValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func slug(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := []rune(strings.TrimSuffix(b.String(), "-"))
	s = s[:min(len(s), 50)]
	return cmp.Or(strings.TrimRight(string(s), "-"), "incident")
}
