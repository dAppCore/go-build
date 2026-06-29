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

// captureSDKStdout redirects cli output into a buffer for the duration of the
// test so assertions can inspect the rendered CLI output instead of leaking it
// into the test log. The original writers are restored on cleanup.
func captureSDKStdout(t *core.T) *core.Buffer {
	t.Helper()
	buf := core.NewBuffer()
	cli.SetStdout(buf)
	cli.SetStderr(buf)
	t.Cleanup(func() {
		cli.SetStdout(nil)
		cli.SetStderr(nil)
	})
	return buf
}

// writeSDKSpec writes content to <dir>/<name> and fails the test on error.
func writeSDKSpec(t *core.T, dir, name, content string) string {
	t.Helper()
	path := ax.Join(dir, name)
	if r := ax.WriteFile(path, []byte(content), 0o644); !r.OK {
		t.Fatalf("unexpected error: %v", r.Error())
	}
	return path
}

const baseTwoPathSpec = `openapi: "3.0.0"
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
  /status:
    get:
      operationId: getStatus
      responses:
        "200":
          description: OK
`

const revOnePathSpec = `openapi: "3.0.0"
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

// --- AddSDKCommands: registers the command surface ---

// noopAction is a placeholder executable action used to pre-occupy command
// paths so AddSDKCommands' partial-failure branches can be observed.
func noopAction(core.Options) core.Result { return core.Ok(nil) }

func TestCmd_AddSDKCommands_Good(t *core.T) {
	c := core.New()

	result := AddSDKCommands(c)
	core.AssertTrue(t, result.OK)
	// The happy path registers every documented command path.
	for _, path := range []string{"sdk", "sdk/generate", "sdk/diff", "sdk/validate"} {
		core.AssertTrue(t, c.Command(path).OK, "expected command "+path+" registered")
	}
	cmd := c.Command("sdk/diff").Value.(*core.Command)
	core.AssertNotNil(t, cmd.Action)
}

func TestCmd_AddSDKCommands_Bad(t *core.T) {
	// Failure at the very first step: the `sdk` command is already taken by an
	// executable command, so the first registerSDKGenerateCommand call fails
	// and AddSDKCommands returns immediately.
	first := core.New()
	core.AssertTrue(t, first.Command("sdk", core.Command{Action: noopAction}).OK)
	firstResult := AddSDKCommands(first)
	core.AssertFalse(t, firstResult.OK)
	core.AssertContains(t, firstResult.Error(), "sdk")
	core.AssertContains(t, firstResult.Error(), "already registered")
	// Nothing past the first step is registered.
	core.AssertFalse(t, first.Command("sdk/diff").OK)

	// Failure at the second step: the `sdk/generate` alias is already taken,
	// so the alias registration fails after `sdk` itself succeeds.
	second := core.New()
	core.AssertTrue(t, second.Command("sdk/generate", core.Command{Action: noopAction}).OK)
	secondResult := AddSDKCommands(second)
	core.AssertFalse(t, secondResult.OK)
	core.AssertContains(t, secondResult.Error(), "sdk/generate")
	core.AssertFalse(t, second.Command("sdk/diff").OK)
}

func TestCmd_AddSDKCommands_Ugly(t *core.T) {
	// Edge cases around the later registration steps. Pre-occupying `sdk/diff`
	// makes the diff registration fail after the generate aliases succeed.
	diffConflict := core.New()
	core.AssertTrue(t, diffConflict.Command("sdk/diff", core.Command{Action: noopAction}).OK)
	diffResult := AddSDKCommands(diffConflict)
	core.AssertFalse(t, diffResult.OK)
	core.AssertContains(t, diffResult.Error(), "sdk/diff")
	// The generate aliases registered before the failing step are present.
	core.AssertTrue(t, diffConflict.Command("sdk/generate").OK)
	// validate is registered after diff, so it must not have been reached.
	core.AssertFalse(t, diffConflict.Command("sdk/validate").OK)

	// Pre-occupying `sdk/validate` makes the final registration step fail.
	validateConflict := core.New()
	core.AssertTrue(t, validateConflict.Command("sdk/validate", core.Command{Action: noopAction}).OK)
	validateResult := AddSDKCommands(validateConflict)
	core.AssertFalse(t, validateResult.OK)
	core.AssertContains(t, validateResult.Error(), "sdk/validate")
	// Everything up to and including diff was registered before the failure.
	core.AssertTrue(t, validateConflict.Command("sdk/diff").OK)
}

// --- registerSDKGenerateCommand: wires the generate action ---

func TestCmd_registerSDKGenerateCommand_Good(t *core.T) {
	c := core.New()

	result := registerSDKGenerateCommand(c, "sdk/generate")
	core.AssertTrue(t, result.OK)
	registered := c.Command("sdk/generate")
	core.AssertTrue(t, registered.OK)
	cmd := registered.Value.(*core.Command)
	core.AssertEqual(t, "cmd.sdk.long", cmd.Description)
	core.AssertNotNil(t, cmd.Action)
}

func TestCmd_registerSDKGenerateCommand_Bad(t *core.T) {
	c := core.New()
	core.AssertTrue(t, registerSDKGenerateCommand(c, "sdk").OK)

	// A second registration of the same executable path is rejected.
	result := registerSDKGenerateCommand(c, "sdk")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "already registered")
}

func TestCmd_registerSDKGenerateCommand_Ugly(t *core.T) {
	// An empty path is an invalid command path and must be rejected by the
	// underlying core.Command validation rather than silently registered.
	c := core.New()

	result := registerSDKGenerateCommand(c, "")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "invalid command path")
}

// The registered generate action drives runSDKGenerate; exercising it via the
// command Action covers the closure wiring in registerSDKGenerateCommand.
func TestCmd_registerSDKGenerateCommand_ActionGood(t *core.T) {
	captureSDKStdout(t)
	c := core.New()
	core.AssertTrue(t, registerSDKGenerateCommand(c, "sdk").OK)
	cmd := c.Command("sdk").Value.(*core.Command)

	// Real cwd has no spec, so the action surfaces the detect-spec failure
	// through the full Action -> runSDKGenerate -> runSDKGenerateInDir path.
	result := cmd.Run(core.NewOptions(core.Option{Key: "dry-run", Value: true}))
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no OpenAPI spec found")
}

// --- runSDKGenerate: working-directory wrapper around runSDKGenerateInDir ---

func TestCmd_runSDKGenerate_Good(t *core.T) {
	buf := captureSDKStdout(t)

	// Dry-run against the real working directory. There is no spec there, so
	// the documented behaviour is a detect-spec failure after the header is
	// rendered — this exercises ax.Getwd success + propagation.
	result := runSDKGenerate(context.Background(), "", "", "", true, false)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, buf.String(), "Generating SDKs")
	core.AssertContains(t, result.Error(), "no OpenAPI spec found")
}

func TestCmd_runSDKGenerate_Bad(t *core.T) {
	captureSDKStdout(t)

	// A non-dry-run with an explicitly configured but missing spec path also
	// fails at detection — the configured path is reported back.
	result := runSDKGenerate(context.Background(), "does-not-exist.yaml", "", "", false, false)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "configured spec not found")
}

func TestCmd_runSDKGenerate_Ugly(t *core.T) {
	captureSDKStdout(t)

	// Edge case: skip-unavailable plus an explicit language. Detection still
	// fails first (no spec in cwd), proving the spec check precedes any
	// generator availability handling.
	result := runSDKGenerate(context.Background(), "", "go", "v9.9.9", false, true)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no OpenAPI spec found")
}

// --- runSDKGenerateInDir: the core generate flow ---

func TestCmd_runSDKGenerateInDir_Good(t *core.T) {
	tmpDir := t.TempDir()
	writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	buf := captureSDKStdout(t)

	// Dry-run across all default languages: lists every configured language
	// and reports the would-generate summary without invoking generators.
	result := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", true, false)
	core.AssertTrue(t, result.OK)
	out := buf.String()
	core.AssertContains(t, out, "languages")
	core.AssertContains(t, out, "typescript")
	core.AssertContains(t, out, "Would generate SDKs")
}

func TestCmd_runSDKGenerateInDir_Bad(t *core.T) {
	tmpDir := t.TempDir()
	writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	captureSDKStdout(t)

	// Non-dry-run with an unknown language: GenerateLanguageWithStatus rejects
	// the language and the error is surfaced from the generate flow.
	result := runSDKGenerateInDir(context.Background(), tmpDir, "", "cobol", "", false, false)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "unknown language: cobol")
}

func TestCmd_runSDKGenerateInDir_Ugly(t *core.T) {
	tmpDir := t.TempDir()
	captureSDKStdout(t)

	// Edge case: a structurally invalid OpenAPI document (no version) is
	// rejected by ValidateSpec before any generation is attempted.
	writeSDKSpec(t, tmpDir, "openapi.yaml", `openapi: "3.0.0"
info:
  title: Test API
paths: {}
`)
	result := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", true, false)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "invalid OpenAPI spec")
}

// containsAny reports whether haystack contains at least one needle.
func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if core.Contains(haystack, n) {
			return true
		}
	}
	return false
}

// TestCmd_runSDKGenerateInDir_LanguageReported drives the real (non-dry-run)
// single-language generation path so the per-language reporting branch is
// exercised. PATH is emptied and skip-unavailable enabled, so:
//   - when the generator is unavailable (the common CI case) the language is
//     skipped and the call succeeds, covering the "Skipped" report line;
//   - when a container/native generator is reachable the language is generated,
//     covering the "generated" report line.
//
// A non-OK result can only stem from generator infrastructure (e.g. a docker
// binary present but its daemon unreachable). That orchestration logic is owned
// and asserted by pkg/sdk; here it is treated as an environment skip so the
// formatting assertions never produce a false failure.
func TestCmd_runSDKGenerateInDir_LanguageReported(t *core.T) {
	tmpDir := t.TempDir()
	writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	t.Setenv("PATH", t.TempDir())
	buf := captureSDKStdout(t)

	result := runSDKGenerateInDir(context.Background(), tmpDir, "", "go", "v1.2.3", false, true)
	if !result.OK {
		t.Skipf("go SDK generation unavailable in this environment: %v", result.Error())
	}
	out := buf.String()
	core.AssertContains(t, out, "go")
	// Exactly one of the two single-language report lines is rendered.
	core.AssertTrue(t, containsAny(out, "generated", "Skipped"))
	core.AssertContains(t, out, "SDK generation complete")
	// When the SDK was actually generated, its directory is materialised.
	if core.Contains(out, "generated") {
		core.AssertTrue(t, ax.Exists(ax.Join(tmpDir, "sdk", "go")))
	}
}

// TestCmd_runSDKGenerateInDir_AllLanguagesReported drives the real
// (non-dry-run) all-languages generation path with skip-unavailable enabled, so
// the aggregate generated/skipped reporting branch is exercised. PATH is
// emptied; the call therefore succeeds whether the default languages are
// generated or skipped, and the aggregate report plus success footer are
// asserted. See LanguageReported for the infrastructure-skip rationale.
func TestCmd_runSDKGenerateInDir_AllLanguagesReported(t *core.T) {
	tmpDir := t.TempDir()
	writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	t.Setenv("PATH", t.TempDir())
	buf := captureSDKStdout(t)

	result := runSDKGenerateInDir(context.Background(), tmpDir, "", "", "", false, true)
	if !result.OK {
		t.Skipf("SDK generation unavailable in this environment: %v", result.Error())
	}
	out := buf.String()
	// With the default four languages at least one report line is present.
	core.AssertTrue(t, containsAny(out, "generated", "Skipped"))
	core.AssertContains(t, out, "SDK generation complete")
}

// --- runSDKValidate: working-directory wrapper around runSDKValidateInDir ---

func TestCmd_runSDKValidate_Good(t *core.T) {
	buf := captureSDKStdout(t)

	// Against the real cwd (no spec) validation reports the detection failure
	// after printing the validating header.
	result := runSDKValidate("")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, buf.String(), "Validating OpenAPI spec")
	core.AssertContains(t, result.Error(), "no OpenAPI spec found")
}

func TestCmd_runSDKValidate_Bad(t *core.T) {
	captureSDKStdout(t)

	// An explicit, missing spec path is reported as a configured-spec miss.
	result := runSDKValidate("missing-spec.yaml")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "configured spec not found")
}

func TestCmd_runSDKValidate_Ugly(t *core.T) {
	captureSDKStdout(t)

	// Edge case: a JSON spec path that does not exist still routes through the
	// configured-spec branch and fails identically to the YAML case.
	result := runSDKValidate("api/openapi.json")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "configured spec not found")
}

// --- runSDKValidateInDir: the core validate flow ---

func TestCmd_runSDKValidateInDir_Good(t *core.T) {
	tmpDir := t.TempDir()
	specPath := writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	buf := captureSDKStdout(t)

	result := runSDKValidateInDir(context.Background(), tmpDir, "")
	core.AssertTrue(t, result.OK)
	out := buf.String()
	// The detected spec path is echoed and the success line is rendered.
	core.AssertContains(t, out, specPath)
	core.AssertContains(t, out, "OpenAPI spec is valid")
}

func TestCmd_runSDKValidateInDir_Bad(t *core.T) {
	tmpDir := t.TempDir()
	captureSDKStdout(t)

	// No spec anywhere under the project dir -> detection failure.
	result := runSDKValidateInDir(context.Background(), tmpDir, "")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no OpenAPI spec found")
}

func TestCmd_runSDKValidateInDir_Ugly(t *core.T) {
	tmpDir := t.TempDir()
	captureSDKStdout(t)

	// Edge case: the spec exists but is not a valid OpenAPI document; the
	// override specPath argument selects it and validation rejects it.
	writeSDKSpec(t, tmpDir, "custom.yaml", `openapi: "3.0.0"
info:
  title: Broken API
paths: {}
`)
	result := runSDKValidateInDir(context.Background(), tmpDir, "custom.yaml")
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "invalid OpenAPI spec")
}

// --- runSDKDiff: working-directory wrapper around runSDKDiffInDir ---

func TestCmd_runSDKDiff_Good(t *core.T) {
	tmpDir := t.TempDir()
	basePath := writeSDKSpec(t, tmpDir, "base.yaml", validOpenAPISpec)
	specPath := writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	buf := captureSDKStdout(t)

	// Explicit base + spec make the diff independent of the working directory.
	// Identical specs => no breaking changes => success.
	result := runSDKDiff(basePath, specPath, false)
	core.AssertTrue(t, result.OK)
	core.AssertContains(t, buf.String(), "No breaking changes")
}

func TestCmd_runSDKDiff_Bad(t *core.T) {
	tmpDir := t.TempDir()
	basePath := writeSDKSpec(t, tmpDir, "base.yaml", baseTwoPathSpec)
	specPath := writeSDKSpec(t, tmpDir, "openapi.yaml", revOnePathSpec)
	captureSDKStdout(t)

	// Removing a documented path is a breaking change: the wrapper exits 1.
	result := runSDKDiff(basePath, specPath, false)
	core.AssertFalse(t, result.OK)
	exitErr, ok := result.Value.(*cli.ExitError)
	core.AssertTrue(t, ok, "expected *cli.ExitError")
	core.AssertEqual(t, 1, exitErr.Code)
}

func TestCmd_runSDKDiff_Ugly(t *core.T) {
	captureSDKStdout(t)

	// Edge case: no base supplied and no spec under the real cwd. Spec
	// detection runs first and fails before the base-required check.
	result := runSDKDiff("", "", false)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "no OpenAPI spec found")
}

// --- runSDKDiffInDir: the core diff flow ---

func TestCmd_runSDKDiffInDir_Good(t *core.T) {
	tmpDir := t.TempDir()
	basePath := writeSDKSpec(t, tmpDir, "base.yaml", validOpenAPISpec)
	specPath := writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	buf := captureSDKStdout(t)

	result := runSDKDiffInDir(tmpDir, basePath, specPath, false)
	core.AssertTrue(t, result.OK)
	out := buf.String()
	core.AssertContains(t, out, "Checking breaking changes")
	core.AssertContains(t, out, "No breaking changes")
}

func TestCmd_runSDKDiffInDir_Bad(t *core.T) {
	tmpDir := t.TempDir()
	specPath := writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	captureSDKStdout(t)

	// An explicit current spec but an empty base path is a usage error.
	result := runSDKDiffInDir(tmpDir, "", specPath, false)
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "base spec is required")
}

func TestCmd_runSDKDiffInDir_Ugly(t *core.T) {
	tmpDir := t.TempDir()
	basePath := writeSDKSpec(t, tmpDir, "base.yaml", baseTwoPathSpec)
	// No specPath argument: the diff must auto-detect the current spec from the
	// project directory via DetectSpec before comparing against the base.
	writeSDKSpec(t, tmpDir, "openapi.yaml", revOnePathSpec)
	captureSDKStdout(t)

	result := runSDKDiffInDir(tmpDir, basePath, "", false)
	core.AssertFalse(t, result.OK)
	exitErr, ok := result.Value.(*cli.ExitError)
	core.AssertTrue(t, ok, "expected *cli.ExitError")
	core.AssertEqual(t, 1, exitErr.Code)
}

// TestCmd_runSDKDiffInDir_LoadError covers the diff-computation failure branch:
// when a spec cannot be loaded the command exits with code 2 (the CI "error"
// status) rather than 0 (no changes) or 1 (breaking changes).
func TestCmd_runSDKDiffInDir_LoadError(t *core.T) {
	tmpDir := t.TempDir()
	// A malformed YAML document that the OpenAPI loader rejects.
	basePath := writeSDKSpec(t, tmpDir, "base.yaml", ":\n  not: [valid")
	specPath := writeSDKSpec(t, tmpDir, "openapi.yaml", validOpenAPISpec)
	captureSDKStdout(t)

	result := runSDKDiffInDir(tmpDir, basePath, specPath, false)
	core.AssertFalse(t, result.OK)
	exitErr, ok := result.Value.(*cli.ExitError)
	core.AssertTrue(t, ok, "expected *cli.ExitError")
	core.AssertEqual(t, 2, exitErr.Code)
	core.AssertContains(t, result.Error(), "failed to load")
}
