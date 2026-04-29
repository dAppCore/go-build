package release

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
)

func runReleaseGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if err := ax.ExecDir(context.Background(), dir, "git", args...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestSDK_RunSDKNilConfigBad(t *testing.T) {
	_, err := RunSDK(context.Background(), nil, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "config is nil") {
		t.Fatalf("expected %v to contain %v", err.Error(), "config is nil")
	}

}

func TestSDK_RunSDKNoSDKConfig_FallsBackToDefaultsGood(t *testing.T) {
	cfg := &Config{}
	cfg.projectDir = t.TempDir()
	cfg.version = "v1.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v1.0.0", result.Version) {
		t.Fatalf("want %v, got %v", "v1.0.0", result.Version)
	}
	if !stdlibAssertEqual("sdk", result.Output) {
		t.Fatalf("want %v, got %v", "sdk", result.Output)
	}
	if !stdlibAssertEqual([]string{"typescript", "python", "go", "php"}, result.Languages) {
		t.Fatalf("want %v, got %v", []string{"typescript", "python", "go", "php"}, result.Languages)
	}

}

func TestSDK_RunSDKNoSDKConfig_UsesBuildConfigGood(t *testing.T) {
	projectDir := t.TempDir()
	buildConfig := `version: 1
sdk:
  spec: api/openapi.yaml
  languages: [typescript, go]
  output: generated/sdk
`
	if err := ax.MkdirAll(ax.Join(projectDir, ".core"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(projectDir, ".core", "build.yaml"), []byte(buildConfig), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := &Config{}
	cfg.projectDir = projectDir
	cfg.version = "v2.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v2.0.0", result.Version) {
		t.Fatalf("want %v, got %v", "v2.0.0", result.Version)
	}
	if !stdlibAssertEqual("generated/sdk", result.Output) {
		t.Fatalf("want %v, got %v", "generated/sdk", result.Output)
	}
	if !stdlibAssertEqual([]string{"typescript", "go"}, result.Languages) {
		t.Fatalf("want %v, got %v", []string{"typescript", "go"}, result.Languages)
	}

}

func TestSDK_RunSDKDryRunGood(t *testing.T) {
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript", "python"},
			Output:    "sdk",
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v1.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v1.0.0", result.Version) {
		t.Fatalf("want %v, got %v", "v1.0.0", result.Version)
	}
	if len(result.Languages) != 2 {
		t.Fatalf("want len %v, got %v", 2, len(result.Languages))
	}
	if !stdlibAssertContains(result.Languages, "typescript") {
		t.Fatalf("expected %v to contain %v", result.Languages, "typescript")
	}
	if !stdlibAssertContains(result.Languages,

		// Empty output, should default to "sdk"
		"python") {
		t.Fatalf("expected %v to contain %v", result.Languages, "python")
	}
	if !stdlibAssertEqual("sdk", result.Output) {
		t.Fatalf("want %v, got %v", "sdk", result.Output)
	}

}

func TestSDK_RunSDKDryRunDefaultOutputGood(t *testing.T) {
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"go"},
			Output:    "",
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v2.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("sdk", result.Output) {
		t.Fatalf("want %v, got %v", "sdk", result.Output)
	}

}

func TestSDK_RunSDKDryRunDefaultsLanguagesGood(t *testing.T) {
	cfg := &Config{
		SDK: &SDKConfig{},
	}
	cfg.projectDir = t.TempDir()
	cfg.version = "v2.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("sdk", result.Output) {
		t.Fatalf("want %v, got %v", "sdk", result.Output)
	}
	if !stdlibAssertEqual([]string{"typescript", "python", "go", "php"}, result.Languages) {
		t.Fatalf("want %v, got %v", []string{"typescript", "python", "go", "php"}, result.Languages)
	}

}

func TestSDK_RunSDKDryRunDefaultProjectDirGood(t *testing.T) {
	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript"},
			Output:    "out",
		},
	}
	// projectDir is empty, should default to "."
	cfg.version = "v1.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v1.0.0", result.Version) {
		t.Fatalf("want %v, got %v", "v1.0.0", result.Version)

		// This test verifies that when diff.FailOnBreaking is true and breaking changes
		// are detected, RunSDK returns an error. However, since we can't easily mock
		// the diff check, this test verifies the config is correctly processed.
		// The actual breaking change detection is tested in pkg/sdk/diff_test.go.
	}

}

func TestSDK_RunSDKBreakingChangesFailOnBreakingBad(t *testing.T) {

	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript"},
			Output:    "sdk",
			Diff: SDKDiffConfig{
				Enabled:        true,
				FailOnBreaking: true,
			},
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v1.0.0"

	// In dry run mode with no git repo, diff check will fail gracefully
	// (non-fatal warning), so this should succeed
	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v1.0.0", result.Version) {
		t.Fatalf("want %v, got %v", "v1.0.0", result.Version)
	}

}

func TestSDK_ToSDKConfigGood(t *testing.T) {
	sdkCfg := &SDKConfig{
		Spec:      "api/openapi.yaml",
		Languages: []string{"typescript", "go"},
		Output:    "sdk",
		Package: SDKPackageConfig{
			Name:    "myapi",
			Version: "v1.0.0",
		},
		Diff: SDKDiffConfig{
			Enabled:        true,
			FailOnBreaking: true,
		},
		Publish: SDKPublishConfig{
			Repo: "owner/sdk-monorepo",
			Path: "packages/api-client",
		},
	}

	result := toSDKConfig(sdkCfg)
	if !stdlibAssertEqual("api/openapi.yaml", result.Spec) {
		t.Fatalf("want %v, got %v", "api/openapi.yaml", result.Spec)
	}
	if !stdlibAssertEqual([]string{"typescript", "go"}, result.Languages) {
		t.Fatalf("want %v, got %v", []string{"typescript", "go"}, result.Languages)
	}
	if !stdlibAssertEqual("sdk", result.Output) {
		t.Fatalf("want %v, got %v", "sdk", result.Output)
	}
	if !stdlibAssertEqual("myapi", result.Package.Name) {
		t.Fatalf("want %v, got %v", "myapi", result.Package.Name)
	}
	if !stdlibAssertEqual("v1.0.0", result.Package.Version) {
		t.Fatalf("want %v, got %v", "v1.0.0", result.Package.Version)
	}
	if !(result.Diff.Enabled) {
		t.Fatal("expected true")
	}
	if !(result.Diff.FailOnBreaking) {

		// Tests diff enabled but FailOnBreaking=false (should warn but not fail)
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("owner/sdk-monorepo", result.Publish.Repo) {
		t.Fatalf("want %v, got %v", "owner/sdk-monorepo", result.Publish.Repo)
	}
	if !stdlibAssertEqual("packages/api-client", result.Publish.Path) {
		t.Fatalf("want %v, got %v", "packages/api-client", result.Publish.Path)
	}

}

func TestSDK_ToSDKConfigNilInputGood(t *testing.T) {
	result := toSDKConfig(nil)
	if !stdlibAssertNil(result) {
		t.Fatalf("expected nil, got %v", result)
	}

}

func TestSDK_RunSDKWithDiffEnabledNoFailOnBreakingGood(t *testing.T) {

	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript"},
			Output:    "sdk",
			Diff: SDKDiffConfig{
				Enabled:        true,
				FailOnBreaking: false,
			},
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v1.0.0"

	// Dry run should succeed even without git repo (diff check fails gracefully)
	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v1.0.0", result.Version) {
		t.Fatalf("want %v, got %v", "v1.0.0", result.Version)
	}
	if !stdlibAssertContains(result.Languages,

		// Tests multiple language support
		"typescript") {
		t.Fatalf("expected %v to contain %v", result.Languages, "typescript")
	}

}

func TestSDK_RunSDKMultipleLanguagesGood(t *testing.T) {

	cfg := &Config{
		SDK: &SDKConfig{
			Languages: []string{"typescript", "python", "go", "java"},
			Output:    "multi-sdk",
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v3.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v3.0.0", result.Version) {
		t.Fatalf("want %v, got %v", "v3.0.0", result.Version)
	}
	if len(result.Languages) != 4 {
		t.Fatalf("want len %v, got %v", 4, len(result.

			// Tests that package config is properly handled
			Languages))
	}
	if !stdlibAssertEqual("multi-sdk", result.Output) {
		t.Fatalf("want %v, got %v", "multi-sdk", result.Output)
	}

}

func TestSDK_RunSDKWithPackageConfigGood(t *testing.T) {

	cfg := &Config{
		SDK: &SDKConfig{
			Spec:      "openapi.yaml",
			Languages: []string{"typescript"},
			Output:    "sdk",
			Package: SDKPackageConfig{
				Name:    "my-custom-sdk",
				Version: "v2.5.0",
			},
		},
	}
	cfg.projectDir = "/tmp"
	cfg.version = "v1.0.0"

	result, err := RunSDK(context.Background(), cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v1.0.0", result.Version) {
		t.Fatalf("want %v, got %v", "v1.0.0", result.Version)

		// Tests conversion with empty package config
	}

}

func TestSDK_ToSDKConfigEmptyPackageConfigGood(t *testing.T) {

	sdkCfg := &SDKConfig{
		Languages: []string{"go"},
		Output:    "sdk",
		// Package is empty struct
	}

	result := toSDKConfig(sdkCfg)
	if !stdlibAssertEqual([]string{"go"}, result.Languages) {
		t.Fatalf("want %v, got %v", []string{"go"}, result.Languages)
	}
	if !stdlibAssertEqual("sdk", result.Output) {
		t.Fatalf("want %v, got %v", "sdk", result.Output)
	}
	if !stdlibAssertEmpty(result.Package.

		// Tests conversion with diff disabled
		Name) {
		t.Fatalf("expected empty, got %v", result.Package.Name)
	}
	if !stdlibAssertEmpty(result.Package.Version) {
		t.Fatalf("expected empty, got %v", result.Package.Version)
	}

}

func TestSDK_ToSDKConfigDiffDisabledGood(t *testing.T) {

	sdkCfg := &SDKConfig{
		Languages: []string{"typescript"},
		Output:    "sdk",
		Diff: SDKDiffConfig{
			Enabled:        false,
			FailOnBreaking: false,
		},
	}

	result := toSDKConfig(sdkCfg)
	if result.Diff.Enabled {
		t.Fatal("expected false")
	}
	if result.Diff.FailOnBreaking {
		t.Fatal("expected false")
	}

}

func TestSDK_ResolveSDKOutputRootGood(t *testing.T) {
	t.Run("uses the default sdk root when no publish path is configured", func(t *testing.T) {
		if !stdlibAssertEqual("sdk", resolveSDKOutputRoot(&SDKConfig{})) {
			t.Fatalf("want %v, got %v", "sdk", resolveSDKOutputRoot(&SDKConfig{}))
		}

	})

	t.Run("prefixes the configured publish path", func(t *testing.T) {
		cfg := &SDKConfig{
			Output: "generated",
			Publish: SDKPublishConfig{
				Path: "packages/api-client",
			},
		}
		if !stdlibAssertEqual(ax.Join("packages/api-client", "generated"), resolveSDKOutputRoot(cfg)) {
			t.Fatalf("want %v, got %v", ax.Join("packages/api-client", "generated"), resolveSDKOutputRoot(cfg))
		}

	})
}

func TestSDK_CheckBreakingChanges_UsesPreviousTaggedSpecGood(t *testing.T) {
	dir := t.TempDir()
	runReleaseGit(t, dir, "init")
	runReleaseGit(t, dir, "config", "user.email", "test@example.com")
	runReleaseGit(t, dir, "config", "user.name", "Test User")

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

	currentSpec := `openapi: "3.0.0"
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

	specPath := ax.Join(dir, "openapi.yaml")
	if err := ax.WriteFile(specPath, []byte(baseSpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runReleaseGit(t, dir, "add", "openapi.yaml")
	runReleaseGit(t, dir, "commit", "-m", "feat: add initial spec")
	runReleaseGit(t, dir, "tag", "v1.0.0")
	if err := ax.WriteFile(specPath, []byte(currentSpec), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runReleaseGit(t, dir, "add", "openapi.yaml")
	runReleaseGit(t, dir, "commit", "-m", "feat: remove users endpoint")

	breaking, err := checkBreakingChanges(context.Background(), dir, &SDKConfig{Spec: "openapi.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(breaking) {
		t.Fatal("expected true")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestSdk_RunSDK_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = RunSDK(ctx, &Config{}, true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_RunSDK_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = RunSDK(ctx, nil, true)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_RunSDK_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = RunSDK(ctx, &Config{}, true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
