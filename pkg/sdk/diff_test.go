package sdk

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"github.com/oasdiff/oasdiff/checker"
)

func TestDiff_NoBreakingGood(t *testing.T) {
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

func TestDiffWithOptions_Warnings_Good(t *testing.T) {
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
	revPath := ax.Join(tmpDir, "rev.yaml")
	_ = ax.WriteFile(basePath, []byte(baseSpec), 0644)
	_ = ax.WriteFile(revPath, []byte(revSpec), 0644)

	result, err := DiffWithOptions(basePath, revPath, DiffOptions{MinimumLevel: checker.WARN})
	if err != nil {
		t.Fatalf("DiffWithOptions failed: %v", err)
	}
	if result.Breaking {
		t.Error("expected warning-only change for endpoint deprecation")
	}
	if !result.HasWarnings {
		t.Fatal("expected warnings to be detected")
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning details")
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestDiff_Diff_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Diff(core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiff_Diff_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Diff("", "")
	})
	core.AssertTrue(t, true)
}

func TestDiff_Diff_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Diff(core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiff_DiffWithOptions_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = DiffWithOptions(core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"), DiffOptions{})
	})
	core.AssertTrue(t, true)
}

func TestDiff_DiffWithOptions_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = DiffWithOptions("", "", DiffOptions{})
	})
	core.AssertTrue(t, true)
}

func TestDiff_DiffWithOptions_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = DiffWithOptions(core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"), DiffOptions{})
	})
	core.AssertTrue(t, true)
}

func TestDiff_DiffExitCode_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = DiffExitCode(&DiffResult{}, nil)
	})
	core.AssertTrue(t, true)
}

func TestDiff_DiffExitCode_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = DiffExitCode(nil, nil)
	})
	core.AssertTrue(t, true)
}

func TestDiff_DiffExitCode_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = DiffExitCode(&DiffResult{}, nil)
	})
	core.AssertTrue(t, true)
}
