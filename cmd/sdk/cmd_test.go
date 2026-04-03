package sdkcmd

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

func TestRunSDKValidate_Good(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validOpenAPISpec), 0o644))

	err := runSDKValidateInDir(context.Background(), tmpDir, "")
	assert.NoError(t, err)
}

func TestRunSDKValidate_InvalidDocument_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644))

	err := runSDKValidateInDir(context.Background(), tmpDir, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OpenAPI spec")
}
