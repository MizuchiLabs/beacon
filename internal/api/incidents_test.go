package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mizuchilabs/beacon/internal/incidents"
)

func TestSyncIncidentsAuth(t *testing.T) {
	t.Parallel()

	_, api := humatest.New(t)
	registerIncidents(api, &incidents.Service{SyncToken: "secret"})

	sync := func(args ...any) int {
		return api.Post("/api/incidents/sync", args...).Code
	}
	body := map[string]any{"ref": "refs/heads/main"}
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(raw)
	signature := "X-Hub-Signature-256: sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, http.StatusAccepted, sync("Authorization: Bearer secret"))
	assert.Equal(t, http.StatusAccepted, sync("X-Gitlab-Token: secret"))
	assert.Equal(t, http.StatusAccepted, sync(signature, body), "github signs the body with the secret")
	assert.Equal(t, http.StatusUnauthorized, sync("X-Hub-Signature-256: sha256=00", body))
	assert.Equal(t, http.StatusUnauthorized, sync("Authorization: Bearer nope"))
	assert.Equal(t, http.StatusUnauthorized, sync(), "no token")
	assert.Equal(t, http.StatusUnauthorized, api.Post("/api/incidents/sync?token=secret").Code, "the query is ignored")

	_, off := humatest.New(t)
	registerIncidents(off, &incidents.Service{})
	assert.Equal(
		t,
		http.StatusNotFound,
		off.Post("/api/incidents/sync", "Authorization: Bearer ").Code,
		"no token configured turns the endpoint off",
	)
}
