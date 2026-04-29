package sdk

import (
	core "dappco.re/go"
	"testing"

	"dappco.re/go/build/internal/ax"
)

// --- Breaking Change Detection Tests (oasdiff integration) ---

func TestBreaking_DiffAddEndpointNonBreakingGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Breaking {
		t.Fatal("adding endpoints should not be breaking")
	}
	if !stdlibAssertEmpty(result.Changes) {
		t.Fatalf("expected empty, got %v", result.Changes)
	}
	if !stdlibAssertEqual("No breaking changes", result.Summary) {
		t.Fatalf("want %v, got %v", "No breaking changes", result.Summary)
	}

}

func TestBreaking_DiffRemoveEndpointBreakingGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(result.Breaking) {
		t.Fatal("removing endpoints should be breaking")
	}
	if stdlibAssertEmpty(result.Changes) {
		t.Fatal("expected non-empty")
	}
	if !stdlibAssertContains(result.Summary, "breaking change") {
		t.Fatalf("expected %v to contain %v", result.Summary, "breaking change")
	}

}

func TestBreaking_DiffAddRequiredParamBreakingGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(result.Breaking) {
		t.Fatal("adding required parameter should be breaking")
	}
	if stdlibAssertEmpty(result.Changes) {
		t.Fatal("expected non-empty")
	}

}

func TestBreaking_DiffAddOptionalParamNonBreakingGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Breaking {
		t.Fatal("adding optional parameter should not be breaking")
	}

}

func TestBreaking_DiffChangeResponseTypeBreakingGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(result.Breaking) {
		t.Fatal("changing response schema type should be breaking")
	}

}

func TestBreaking_DiffRemoveHTTPMethodBreakingGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(result.Breaking) {
		t.Fatal("removing HTTP method should be breaking")
	}
	if stdlibAssertEmpty(result.Changes) {
		t.Fatal("expected non-empty")
	}

}

func TestBreaking_DiffIdenticalSpecsNonBreakingGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(spec), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(spec), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Breaking {
		t.Fatal("identical specs should not be breaking")
	}
	if !stdlibAssertEmpty(result.Changes) {
		t.Fatalf("expected empty, got %v", result.Changes)

		// --- Error Handling Tests ---
	}
	if !stdlibAssertEqual("No breaking changes", result.Summary) {
		t.Fatalf("want %v, got %v", "No breaking changes", result.Summary)
	}

}

func TestBreaking_DiffNonExistentBaseBad(t *testing.T) {
	tmpDir := t.TempDir()

	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(revPath, []byte(`openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := Diff(ax.Join(tmpDir, "nonexistent.yaml"), revPath)
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "failed to load base spec") {
		t.Fatalf("expected %v to contain %v", err.Error(), "failed to load base spec")
	}

}

func TestBreaking_DiffNonExistentRevisionBad(t *testing.T) {
	tmpDir := t.TempDir()

	basePath := ax.Join(tmpDir, "base.yaml")
	if err := ax.WriteFile(basePath, []byte(`openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := Diff(basePath, ax.Join(tmpDir, "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "failed to load revision spec") {
		t.Fatalf("expected %v to contain %v", err.Error(), "failed to load revision spec")
	}

}

func TestBreaking_DiffInvalidYAMLBad(t *testing.T) {
	tmpDir := t.TempDir()

	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte("not: valid: openapi: spec: {{{{"), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(`openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := Diff(basePath, revPath)
	if err == nil {
		t.Fatal("expected error")

		// --- DiffExitCode Tests ---
	}

}

func TestBreaking_DiffExitCode_Good(t *testing.T) {
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
			err:      core.NewError("test error"),
			expected: 2,
		},
		{
			name:     "nil result returns 2",
			result:   nil,
			err:      nil,
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code := DiffExitCode(tc.result, tc.err)
			if !stdlibAssertEqual(tc.expected, code) {
				t.Fatalf("want %v, got %v",

					// --- DiffResult Structure Tests ---
					tc.expected, code)
			}

		})
	}
}

func TestBreaking_DiffResultSummaryGood(t *testing.T) {
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
		basePath := ax.Join(tmpDir, "base.yaml")
		revPath := ax.Join(tmpDir, "rev.yaml")
		if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := Diff(basePath, revPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(result.Breaking) {
			t.Fatal("expected true")
		}
		if !stdlibAssertContains(result.

			// Should have at least 2 changes (removed /users and /orders)
			Summary, "breaking change") {
			t.Fatalf("expected %v to contain %v", result.Summary, "breaking change")
		}
		if len(result.Changes) < 2 {
			t.Fatalf("expected %v to be greater than or equal to %v", len(result.Changes), 2)
		}

	})
}

func TestBreaking_DiffResultChangesAreHumanReadableGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(result.Breaking) {

		// Changes should contain human-readable descriptions from oasdiff
		t.Fatal("expected true")
	}

	for _, change := range result.Changes {
		if stdlibAssertEmpty(change) {
			t.Fatal("each change should have a description")
		}

	}
}

// --- Multiple Changes Detection Tests ---

func TestBreaking_DiffMultipleBreakingChangesGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	if err := ax.WriteFile(basePath, []byte(base), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revision), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(result.Breaking) {

		// Should detect: removed POST, removed DELETE, and possibly added required param
		t.Fatal("expected true")
	}
	if len(result.Changes) < 2 {
		t.Fatalf("should detect multiple breaking changes, got: %v", result.Changes)
	}

}

// --- JSON Spec Support Tests ---

func TestBreaking_DiffJSONSpecsGood(t *testing.T) {
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
	basePath := ax.Join(tmpDir, "base.json")
	revPath := ax.Join(tmpDir, "rev.json")
	if err := ax.WriteFile(basePath, []byte(baseJSON), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(revPath, []byte(revJSON), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Breaking {
		t.Fatal("adding endpoint in JSON format should not be breaking")
	}

}
