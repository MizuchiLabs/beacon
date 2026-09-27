package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpecBuilds(t *testing.T) {
	t.Parallel()

	spec := Spec()
	require.NotNil(t, spec.Paths["/api/monitors"], "registering every route must not panic")
	require.NotNil(t, spec.Paths["/api/push/{token}"])
}
