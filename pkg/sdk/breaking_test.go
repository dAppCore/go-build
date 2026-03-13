package sdk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Breaking Change Detection Tests (oasdiff integration) ---

func TestDiff_Good_AddEndpoint_NonBreaking(t *testing.T) {
	tmpDir := t.TempDir()

	base := `openapi: "3.0.0"
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
	revision := `openapi: "3.0.0"
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
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
  /status:
    get:
      operationId: getStatus
      responses:
        "200":
          description: OK
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)
	assert.False(t, result.Breaking, "adding endpoints should not be breaking")
	assert.Empty(t, result.Changes)
	assert.Equal(t, "No breaking changes", result.Summary)
}

func TestDiff_Good_RemoveEndpoint_Breaking(t *testing.T) {
	tmpDir := t.TempDir()

	base := `openapi: "3.0.0"
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
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
  /orders:
    get:
      operationId: listOrders
      responses:
        "200":
          description: OK
`
	revision := `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)
	assert.True(t, result.Breaking, "removing endpoints should be breaking")
	assert.NotEmpty(t, result.Changes)
	assert.Contains(t, result.Summary, "breaking change")
}

func TestDiff_Good_AddRequiredParam_Breaking(t *testing.T) {
	tmpDir := t.TempDir()

	base := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
`
	revision := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.1.0"
paths:
  /users:
    get:
      operationId: listUsers
      parameters:
        - name: tenant_id
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)
	assert.True(t, result.Breaking, "adding required parameter should be breaking")
	assert.NotEmpty(t, result.Changes)
}

func TestDiff_Good_AddOptionalParam_NonBreaking(t *testing.T) {
	tmpDir := t.TempDir()

	base := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
`
	revision := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.1.0"
paths:
  /users:
    get:
      operationId: listUsers
      parameters:
        - name: page
          in: query
          required: false
          schema:
            type: integer
      responses:
        "200":
          description: OK
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)
	assert.False(t, result.Breaking, "adding optional parameter should not be breaking")
}

func TestDiff_Good_ChangeResponseType_Breaking(t *testing.T) {
	tmpDir := t.TempDir()

	base := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: integer
                    name:
                      type: string
`
	revision := `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      type: object
                      properties:
                        id:
                          type: integer
                        name:
                          type: string
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)
	assert.True(t, result.Breaking, "changing response schema type should be breaking")
}

func TestDiff_Good_RemoveHTTPMethod_Breaking(t *testing.T) {
	tmpDir := t.TempDir()

	base := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
    post:
      operationId: createUser
      responses:
        "201":
          description: Created
`
	revision := `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)
	assert.True(t, result.Breaking, "removing HTTP method should be breaking")
	assert.NotEmpty(t, result.Changes)
}

func TestDiff_Good_IdenticalSpecs_NonBreaking(t *testing.T) {
	tmpDir := t.TempDir()

	spec := `openapi: "3.0.0"
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
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
    post:
      operationId: createUser
      responses:
        "201":
          description: Created
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(spec), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(spec), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)
	assert.False(t, result.Breaking, "identical specs should not be breaking")
	assert.Empty(t, result.Changes)
	assert.Equal(t, "No breaking changes", result.Summary)
}

// --- Error Handling Tests ---

func TestDiff_Bad_NonExistentBase(t *testing.T) {
	tmpDir := t.TempDir()

	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(revPath, []byte(`openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`), 0644))

	_, err := Diff(filepath.Join(tmpDir, "nonexistent.yaml"), revPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load base spec")
}

func TestDiff_Bad_NonExistentRevision(t *testing.T) {
	tmpDir := t.TempDir()

	basePath := filepath.Join(tmpDir, "base.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(`openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`), 0644))

	_, err := Diff(basePath, filepath.Join(tmpDir, "nonexistent.yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load revision spec")
}

func TestDiff_Bad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte("not: valid: openapi: spec: {{{{"), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(`openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`), 0644))

	_, err := Diff(basePath, revPath)
	assert.Error(t, err)
}

// --- DiffExitCode Tests ---

func TestDiffExitCode_Good(t *testing.T) {
	tests := []struct {
		name     string
		result   *DiffResult
		err      error
		expected int
	}{
		{
			name:     "no breaking changes returns 0",
			result:   &DiffResult{Breaking: false},
			err:      nil,
			expected: 0,
		},
		{
			name:     "breaking changes returns 1",
			result:   &DiffResult{Breaking: true, Changes: []string{"removed endpoint"}},
			err:      nil,
			expected: 1,
		},
		{
			name:     "error returns 2",
			result:   nil,
			err:      assert.AnError,
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code := DiffExitCode(tc.result, tc.err)
			assert.Equal(t, tc.expected, code)
		})
	}
}

// --- DiffResult Structure Tests ---

func TestDiffResult_Good_Summary(t *testing.T) {
	t.Run("breaking result has count in summary", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create specs with 2 removed endpoints
		base := `openapi: "3.0.0"
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
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
  /orders:
    get:
      operationId: listOrders
      responses:
        "200":
          description: OK
`
		revision := `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths:
  /health:
    get:
      operationId: getHealth
      responses:
        "200":
          description: OK
`
		basePath := filepath.Join(tmpDir, "base.yaml")
		revPath := filepath.Join(tmpDir, "rev.yaml")
		require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
		require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

		result, err := Diff(basePath, revPath)
		require.NoError(t, err)

		assert.True(t, result.Breaking)
		assert.Contains(t, result.Summary, "breaking change")
		// Should have at least 2 changes (removed /users and /orders)
		assert.GreaterOrEqual(t, len(result.Changes), 2)
	})
}

func TestDiffResult_Good_ChangesAreHumanReadable(t *testing.T) {
	tmpDir := t.TempDir()

	base := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /removed-endpoint:
    get:
      operationId: removedEndpoint
      responses:
        "200":
          description: OK
`
	revision := `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths: {}
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)

	assert.True(t, result.Breaking)
	// Changes should contain human-readable descriptions from oasdiff
	for _, change := range result.Changes {
		assert.NotEmpty(t, change, "each change should have a description")
	}
}

// --- Multiple Changes Detection Tests ---

func TestDiff_Good_MultipleBreakingChanges(t *testing.T) {
	tmpDir := t.TempDir()

	base := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
    post:
      operationId: createUser
      responses:
        "201":
          description: Created
    delete:
      operationId: deleteAllUsers
      responses:
        "204":
          description: No Content
`
	revision := `openapi: "3.0.0"
info:
  title: Test API
  version: "2.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      parameters:
        - name: required_filter
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	revPath := filepath.Join(tmpDir, "rev.yaml")
	require.NoError(t, os.WriteFile(basePath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revision), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)

	assert.True(t, result.Breaking)
	// Should detect: removed POST, removed DELETE, and possibly added required param
	assert.GreaterOrEqual(t, len(result.Changes), 2,
		"should detect multiple breaking changes, got: %v", result.Changes)
}

// --- JSON Spec Support Tests ---

func TestDiff_Good_JSONSpecs(t *testing.T) {
	tmpDir := t.TempDir()

	baseJSON := `{
  "openapi": "3.0.0",
  "info": {"title": "Test API", "version": "1.0.0"},
  "paths": {
    "/health": {
      "get": {
        "operationId": "getHealth",
        "responses": {"200": {"description": "OK"}}
      }
    }
  }
}`
	revJSON := `{
  "openapi": "3.0.0",
  "info": {"title": "Test API", "version": "1.1.0"},
  "paths": {
    "/health": {
      "get": {
        "operationId": "getHealth",
        "responses": {"200": {"description": "OK"}}
      }
    },
    "/status": {
      "get": {
        "operationId": "getStatus",
        "responses": {"200": {"description": "OK"}}
      }
    }
  }
}`
	basePath := filepath.Join(tmpDir, "base.json")
	revPath := filepath.Join(tmpDir, "rev.json")
	require.NoError(t, os.WriteFile(basePath, []byte(baseJSON), 0644))
	require.NoError(t, os.WriteFile(revPath, []byte(revJSON), 0644))

	result, err := Diff(basePath, revPath)
	require.NoError(t, err)
	assert.False(t, result.Breaking, "adding endpoint in JSON format should not be breaking")
}
