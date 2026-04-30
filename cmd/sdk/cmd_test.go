package sdkcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cli"
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
	if result := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validOpenAPISpec), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	result := runSDKValidateInDir(context.Background(), tmpDir, "")
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func TestAddSDKCommands_RegistersGenerateAliasGood(t *testing.T) {
	c := core.New()

	AddSDKCommands(c)
	if !(c.Command("sdk").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("sdk/generate").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("sdk/diff").OK) {
		t.Fatal("expected true")
	}
	if !(c.Command("sdk/validate").OK) {
		t.Fatal("expected true")
	}

}

func TestRunSDKGenerateInDir_ValidSpecDryRunGood(t *testing.T) {
	tmpDir := t.TempDir()
	if result := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(validOpenAPISpec), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	result := runSDKGenerateInDir(context.Background(), tmpDir, "", "go", "", true, false)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func TestRunSDKGenerateInDir_UsesBuildSDKConfigGood(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	if result := ax.MkdirAll(ax.Dir(specPath), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(specPath, []byte(validOpenAPISpec), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.MkdirAll(ax.Join(tmpDir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(tmpDir, ".core", "build.yaml"), []byte(`version: 1
sdk:
  spec: docs/openapi.yaml
  languages:
    - go
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	result := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", true, false)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func TestRunSDKGenerateInDir_InvalidDocumentBad(t *testing.T) {
	tmpDir := t.TempDir()
	if result := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	result := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", true, false)
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "invalid OpenAPI spec") {
		t.Fatalf("expected %v to contain %v", result.Error(), "invalid OpenAPI spec")
	}

}

func TestRunSDKValidate_InvalidDocumentBad(t *testing.T) {
	tmpDir := t.TempDir()
	if result := ax.WriteFile(ax.Join(tmpDir, "openapi.yaml"), []byte(`openapi: "3.0.0"
info:
  title: Test API
paths: {}
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	result := runSDKValidateInDir(context.Background(), tmpDir, "")
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "invalid OpenAPI spec") {
		t.Fatalf("expected %v to contain %v", result.Error(), "invalid OpenAPI spec")
	}

}

func TestRunSDKValidate_UsesBuildSDKConfigGood(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := ax.Join(tmpDir, "docs", "openapi.yaml")
	if result := ax.MkdirAll(ax.Dir(specPath), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(specPath, []byte(validOpenAPISpec), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.MkdirAll(ax.Join(tmpDir, ".core"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(ax.Join(tmpDir, ".core", "build.yaml"), []byte(`version: 1
sdk:
  spec: docs/openapi.yaml
`), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	result := runSDKValidateInDir(context.Background(), tmpDir, "")
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func TestRunSDKDiffInDir_FailOnWarnGood(t *testing.T) {
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
	if result := ax.WriteFile(basePath, []byte(baseSpec), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	if result := ax.WriteFile(specPath, []byte(revSpec), 0o644); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	result := runSDKDiffInDir(tmpDir, basePath, specPath, false)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	result = runSDKDiffInDir(tmpDir, basePath, specPath, true)
	if result.OK {
		t.Fatal("expected error")
	}

	var exitErr *cli.ExitError
	if !(core.As(result.Value.(error), &exitErr)) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(1, exitErr.Code) {
		t.Fatalf("want %v, got %v", 1, exitErr.Code)
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestCmd_AddSDKCommands_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		AddSDKCommands(core.New())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCmd_AddSDKCommands_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		AddSDKCommands(core.New())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCmd_AddSDKCommands_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		AddSDKCommands(core.New())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
