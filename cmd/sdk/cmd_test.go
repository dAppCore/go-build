package sdkcmd

import (
	"context"
	"errors"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/cli/pkg/cli"

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

func TestRunSDKGenerateInDir_ValidSpecDryRun_Good(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validOpenAPISpec), 0o644))

	err := runSDKGenerateInDir(context.Background(), tmpDir, "", "go", "", true)
	assert.NoError(t, err)
}

func TestRunSDKGenerateInDir_UsesBuildSDKConfig_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	require.NoError(t, ax.MkdirAll(ax.Dir(specPath), 0o755))
	require.NoError(t, ax.WriteFile(specPath, []byte(validOpenAPISpec), 0o644))
	require.NoError(t, ax.MkdirAll(ax.Join(tmpDir, ".core"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(tmpDir, ".core", "build.yaml"), []byte(`version: 1
sdk:
  spec: docs/openapi.yaml
  languages:
    - go
`), 0o644))

	err := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", true)
	assert.NoError(t, err)
}

func TestRunSDKGenerateInDir_InvalidDocument_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644))

	err := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OpenAPI spec")
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

func TestRunSDKValidate_UsesBuildSDKConfig_Good(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	require.NoError(t, ax.MkdirAll(ax.Dir(specPath), 0o755))
	require.NoError(t, ax.WriteFile(specPath, []byte(validOpenAPISpec), 0o644))
	require.NoError(t, ax.MkdirAll(ax.Join(tmpDir, ".core"), 0o755))
	require.NoError(t, ax.WriteFile(ax.Join(tmpDir, ".core", "build.yaml"), []byte(`version: 1
sdk:
  spec: docs/openapi.yaml
`), 0o644))

	err := runSDKValidateInDir(context.Background(), tmpDir, "")
	assert.NoError(t, err)
}

func TestRunSDKDiffInDir_FailOnWarn_Good(t *testing.T) {
	tmpDir := t.TempDir()

	baseSpec := `openapi: "3.0.0"
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
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                  detail:
                    type: string
`
	revSpec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.1.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
`
	basePath := ax.Join(tmpDir, "base.yaml")
	specPath := ax.Join(tmpDir, "openapi.yaml")
	require.NoError(t, ax.WriteFile(basePath, []byte(baseSpec), 0o644))
	require.NoError(t, ax.WriteFile(specPath, []byte(revSpec), 0o644))

	err := runSDKDiffInDir(tmpDir, basePath, specPath, false)
	assert.NoError(t, err)

	err = runSDKDiffInDir(tmpDir, basePath, specPath, true)
	require.Error(t, err)

	var exitErr *cli.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 1, exitErr.Code)
}
