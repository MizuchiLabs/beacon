package notify

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mizuchilabs/beacon/internal/config"
	"github.com/mizuchilabs/beacon/internal/db"
)

type captured struct {
	body    string
	headers http.Header
}

func captureServer(t *testing.T) (string, <-chan captured) {
	t.Helper()

	got := make(chan captured, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got <- captured{body: string(body), headers: r.Header}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	return server.URL, got
}

func TestWebhookDefaultBodyIsEventJSON(t *testing.T) {
	t.Parallel()

	url, got := captureServer(t)
	n := &Service{}
	require.NoError(t, n.SetWebhooks([]config.Webhook{{URL: url}}))

	event := Event{Event: EventDown, Monitor: "API", Title: "down", Message: "API is down: HTTP 502"}
	require.NoError(t, n.sendWebhook(t.Context(), n.hooks()[0], event))

	req := <-got
	assert.Equal(t, "application/json", req.headers.Get("Content-Type"))
	assert.Contains(t, req.body, `"event":"down"`)
	assert.Contains(t, req.body, `"message":"API is down: HTTP 502"`)
	assert.NotContains(t, req.body, `"url"`, "an empty url is left out")
}

func TestWebhookTemplateEscapesJSON(t *testing.T) {
	t.Parallel()

	url, got := captureServer(t)
	n := &Service{}
	require.NoError(t, n.SetWebhooks([]config.Webhook{{
		URL:     url,
		Headers: map[string]string{"Authorization": "Bearer secret"},
		Body:    `{"content": {{json .Message}}}`,
	}}))

	event := Event{Message: `keyword "ok" not found`}
	require.NoError(t, n.sendWebhook(t.Context(), n.hooks()[0], event))

	req := <-got
	assert.JSONEq(t, `{"content": "keyword \"ok\" not found"}`, req.body)
	assert.Equal(t, "Bearer secret", req.headers.Get("Authorization"))
}

func TestSetWebhooksRejectsBrokenTemplate(t *testing.T) {
	t.Parallel()

	n := &Service{}
	require.NoError(t, n.SetWebhooks([]config.Webhook{{URL: "https://a.test"}}))

	err := n.SetWebhooks([]config.Webhook{{URL: "https://b.test", Body: "{{.Message"}})
	require.ErrorContains(t, err, "webhook #1")
	require.Len(t, n.hooks(), 1, "a broken template keeps the previous webhooks")
	assert.Equal(t, "https://a.test", n.hooks()[0].url)
}

func TestPublicURLHidesPushToken(t *testing.T) {
	t.Parallel()

	assert.Empty(t, PublicURL(&db.Monitor{Type: "push", Url: "push://s3cret"}))
	assert.Equal(t, "https://a.test", PublicURL(&db.Monitor{Type: "http", Url: "https://a.test"}))
}
