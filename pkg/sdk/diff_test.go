package sdk

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"
)

func TestDiff_NoBreaking_Good(t *testing.T) {
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
  /status:
    get:
      operationId: getStatus
      responses:
        "200":
          description: OK
`
	basePath := ax.Join(tmpDir, "base.yaml")
	revPath := ax.Join(tmpDir, "rev.yaml")
	_ = ax.WriteFile(basePath, []byte(baseSpec), 0644)
	_ = ax.WriteFile(revPath, []byte(revSpec), 0644)

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if result.Breaking {
		t.Error("expected no breaking changes for adding endpoint")
	}
}

func TestDiff_Breaking_Good(t *testing.T) {
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
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: OK
`
	revSpec := `openapi: "3.0.0"
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
	_ = ax.WriteFile(basePath, []byte(baseSpec), 0644)
	_ = ax.WriteFile(revPath, []byte(revSpec), 0644)

	result, err := Diff(basePath, revPath)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if !result.Breaking {
		t.Error("expected breaking change for removed endpoint")
	}
}
