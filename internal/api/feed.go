package api

import (
	"encoding/xml"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mizuchilabs/beacon/internal/incidents"
)

const feedLimit = 50

type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Links   []atomLink  `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
}

type atomEntry struct {
	Title   string   `xml:"title"`
	ID      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Link    atomLink `xml:"link"`
	Summary string   `xml:"summary"`
}

// incidentFeed serves the incidents as Atom, so people can follow the status
// page in a feed reader or a chat integration.
func incidentFeed(inc *incidents.Service, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := baseURL(r)
		feed := atomFeed{
			Title: title + " incidents",
			ID:    base + "/events",
			Links: []atomLink{{Href: base + "/incidents.atom", Rel: "self"}, {Href: base + "/events"}},
		}

		var list []incidents.Incident
		if inc != nil {
			list = inc.GetIncidents()
		}
		var newest time.Time
		for _, i := range list[:min(len(list), feedLimit)] {
			updated, summary := i.StartedAt, i.Description
			if len(i.Updates) > 0 {
				latest := i.Updates[len(i.Updates)-1]
				updated, summary = latest.CreatedAt, latest.Message
			}
			if updated.After(newest) {
				newest = updated
			}
			link := base + "/events/" + url.PathEscape(i.ID)
			feed.Entries = append(feed.Entries, atomEntry{
				Title:   "[" + i.Status + "] " + i.Title,
				ID:      link,
				Updated: updated.UTC().Format(time.RFC3339),
				Link:    atomLink{Href: link},
				Summary: summary,
			})
		}
		feed.Updated = newest.UTC().Format(time.RFC3339)

		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_, _ = w.Write([]byte(xml.Header))
		if err := xml.NewEncoder(w).Encode(feed); err != nil {
			slog.Error("Failed to write incident feed", "error", err)
		}
	}
}

// baseURL is the public origin the request came in on, honoring a TLS
// terminating proxy.
func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
