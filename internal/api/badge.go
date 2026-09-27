package api

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mizuchilabs/beacon/internal/db"
)

var badgeColors = map[string]string{
	statusOperational: "#3fb950",
	statusDegraded:    "#d29922",
	statusDown:        "#f85149",
	statusUnknown:     "#8b949e",
}

type BadgeInput struct {
	Name    string `path:"name" doc:"Monitor name"`
	Kind    string `            doc:"Show the current status or the uptime over the window" query:"kind"    enum:"status,uptime" default:"status"`
	Seconds int64  `            doc:"Uptime window in seconds"                              query:"seconds"                      default:"2592000" minimum:"3600" maximum:"31536000"`
}

type BadgeOutput struct {
	ContentType  string `header:"Content-Type"`
	CacheControl string `header:"Cache-Control"`
	Body         []byte
}

type badgeHandlers struct {
	q *db.Queries
}

func registerBadges(api huma.API, q *db.Queries) {
	h := &badgeHandlers{q: q}
	huma.Register(api, huma.Operation{
		OperationID: "get-badge",
		Method:      http.MethodGet,
		Path:        "/api/badge/{name}",
		Summary:     "Get a status or uptime badge",
		Description: "An SVG badge for READMEs and dashboards.",
		Tags:        []string{"Badges"},
		Responses: map[string]*huma.Response{
			"200": {Content: map[string]*huma.MediaType{"image/svg+xml": {}}},
		},
	}, h.getBadge)
}

func (h *badgeHandlers) getBadge(ctx context.Context, in *BadgeInput) (*BadgeOutput, error) {
	monitors, err := h.q.GetMonitors(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get monitors")
	}
	var m *db.Monitor
	for _, candidate := range monitors {
		if candidate.Name == in.Name {
			m = candidate
		}
	}
	if m == nil {
		return nil, huma.Error404NotFound("monitor not found")
	}

	var value, color string
	if in.Kind == "uptime" {
		uptime, err := h.q.GetUptime(ctx, &db.GetUptimeParams{MonitorID: m.ID, FromTs: time.Now().Unix() - in.Seconds})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get uptime")
		}
		value, color = uptimeBadge(uptime.Alive, uptime.Total)
	} else {
		latest, err := h.q.GetLatestChecks(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get latest checks")
		}
		var last *db.GetLatestChecksRow
		for _, l := range latest {
			if l.MonitorID == m.ID {
				last = l
			}
		}
		status := statusOf(m, last, time.Now().Unix())
		value, color = status, badgeColors[status]
	}

	return &BadgeOutput{
		ContentType:  "image/svg+xml",
		CacheControl: "public, max-age=60",
		Body:         badgeSVG(m.Name, value, color),
	}, nil
}

func uptimeBadge(alive, total int64) (string, string) {
	if total == 0 {
		return "no data", badgeColors[statusUnknown]
	}
	pct := float64(alive) / float64(total) * 100
	value := fmt.Sprintf("%.2f%%", pct)
	switch {
	case pct >= 99:
		return value, badgeColors[statusOperational]
	case pct >= 95:
		return value, badgeColors[statusDegraded]
	default:
		return value, badgeColors[statusDown]
	}
}

// badgeSVG draws a flat two part badge. Widths are estimated from the rune
// count, close enough for an 11px sans serif.
func badgeSVG(label, value, color string) []byte {
	labelWidth := utf8.RuneCountInString(label)*7 + 12
	valueWidth := utf8.RuneCountInString(value)*7 + 12
	width := labelWidth + valueWidth
	label, value = html.EscapeString(label), html.EscapeString(value)

	return fmt.Appendf(
		nil,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%[1]d" height="20" role="img" aria-label="%[2]s: %[3]s">`+
			`<title>%[2]s: %[3]s</title>`+
			`<clipPath id="r"><rect width="%[1]d" height="20" rx="4" fill="#fff"/></clipPath>`+
			`<g clip-path="url(#r)"><rect width="%[4]d" height="20" fill="#30363d"/><rect x="%[4]d" width="%[5]d" height="20" fill="%[6]s"/></g>`+
			`<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">`+
			`<text x="%[7]d" y="14">%[2]s</text><text x="%[8]d" y="14">%[3]s</text></g></svg>`,
		width,
		label,
		value,
		labelWidth,
		valueWidth,
		color,
		labelWidth/2,
		labelWidth+valueWidth/2,
	)
}
