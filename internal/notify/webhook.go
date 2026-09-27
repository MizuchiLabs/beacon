package notify

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"text/template"

	"github.com/mizuchilabs/beacon/internal/config"
)

type webhook struct {
	url     string
	headers map[string]string
	body    *template.Template // nil sends the event as JSON
}

// templateFuncs lets a body template embed values in JSON safely, e.g.
// {"content": {{json .Message}}}.
var templateFuncs = template.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
}

// SetWebhooks replaces the webhooks. Nothing changes when a body template
// does not parse. Works on a zero Service, which is all test-notify needs.
func (n *Service) SetWebhooks(hooks []config.Webhook) error {
	parsed := make([]webhook, 0, len(hooks))
	for i, h := range hooks {
		w := webhook{url: h.URL, headers: h.Headers}
		if h.Body != "" {
			tmpl, err := template.New("body").Funcs(templateFuncs).Parse(h.Body)
			if err != nil {
				return fmt.Errorf("webhook #%d: invalid body template: %w", i+1, err)
			}
			w.body = tmpl
		}
		parsed = append(parsed, w)
	}

	n.mu.Lock()
	n.webhooks = parsed
	n.mu.Unlock()
	return nil
}

func (n *Service) hooks() []webhook {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.webhooks
}

func (n *Service) sendWebhook(ctx context.Context, w webhook, event Event) error {
	body, err := renderBody(w, event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Beacon/1.0")
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func renderBody(w webhook, event Event) ([]byte, error) {
	if w.body == nil {
		return json.Marshal(event)
	}
	var buf bytes.Buffer
	if err := w.body.Execute(&buf, event); err != nil {
		return nil, fmt.Errorf("rendering body: %w", err)
	}
	return buf.Bytes(), nil
}
