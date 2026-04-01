package sdk

import (
	"context"
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validOpenAPISpec = `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
`

func TestValidateSpec_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yaml")
	require.NoError(t, ax.WriteFile(specPath, []byte(validOpenAPISpec), 0o644))

	sdk := New(tmpDir, nil)
	got, err := sdk.ValidateSpec(context.Background())
	require.NoError(t, err)
	assert.Equal(t, specPath, got)
}

func TestValidateSpec_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yaml")
	require.NoError(t, ax.WriteFile(specPath, []byte("openapi: 3.0.0\ninfo: [\n"), 0o644))

	sdk := New(tmpDir, nil)
	_, err := sdk.ValidateSpec(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load OpenAPI spec")
}

func TestValidateSpec_InvalidDocument_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "openapi.yaml")
	require.NoError(t, ax.WriteFile(specPath, []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644))

	sdk := New(tmpDir, nil)
	_, err := sdk.ValidateSpec(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OpenAPI spec")
}
