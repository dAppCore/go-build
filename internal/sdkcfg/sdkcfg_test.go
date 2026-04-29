package sdkcfg

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/io"
)

func TestLoadProjectConfig_Good(t *testing.T) {
	t.Run("falls back to release config in the provided medium", func(t *testing.T) {
		medium := io.NewMemoryMedium()
		projectDir := "project"
		if err := medium.EnsureDir(ax.Join(projectDir, release.ConfigDir)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := medium.Write(release.ConfigPath(projectDir), `
version: 1
sdk:
  spec: docs/openapi.yaml
  languages: [php]
  output: generated/sdk
`); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadProjectConfig(medium, projectDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("docs/openapi.yaml", cfg.Spec) {
			t.Fatalf("want %v, got %v", "docs/openapi.yaml", cfg.Spec)
		}
		if !stdlibAssertEqual([]string{"php"}, cfg.Languages) {
			t.Fatalf("want %v, got %v", []string{"php"}, cfg.Languages)
		}
		if !stdlibAssertEqual("generated/sdk", cfg.Output) {
			t.Fatalf("want %v, got %v", "generated/sdk", cfg.Output)
		}

	})

	t.Run("prefers build config over release config in the provided medium", func(t *testing.T) {
		medium := io.NewMemoryMedium()
		projectDir := "project"
		if err := medium.EnsureDir(ax.Join(projectDir, build.ConfigDir)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := medium.Write(build.ConfigPath(projectDir), `
version: 1
sdk:
  spec: openapi.yaml
  languages: [typescript]
`); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := medium.EnsureDir(ax.Join(projectDir, release.ConfigDir)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := medium.Write(release.ConfigPath(projectDir), `
version: 1
sdk:
  spec: docs/openapi.yaml
  languages: [python]
`); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadProjectConfig(medium, projectDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("openapi.yaml", cfg.Spec) {
			t.Fatalf("want %v, got %v", "openapi.yaml", cfg.Spec)
		}
		if !stdlibAssertEqual([]string{"typescript"}, cfg.Languages) {
			t.Fatalf("want %v, got %v", []string{"typescript"}, cfg.Languages)
		}

	})

	t.Run("applies documented defaults to partial sdk config", func(t *testing.T) {
		medium := io.NewMemoryMedium()
		projectDir := "project"
		if err := medium.EnsureDir(ax.Join(projectDir, build.ConfigDir)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := medium.Write(build.ConfigPath(projectDir), `
version: 1
sdk:
  spec: openapi.yaml
`); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadProjectConfig(medium, projectDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(cfg) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("openapi.yaml", cfg.Spec) {
			t.Fatalf("want %v, got %v", "openapi.yaml", cfg.Spec)
		}
		if !stdlibAssertEqual([]string{"typescript", "python", "go", "php"}, cfg.Languages) {
			t.Fatalf("want %v, got %v", []string{"typescript", "python", "go", "php"}, cfg.Languages)
		}
		if !stdlibAssertEqual("sdk", cfg.Output) {
			t.Fatalf("want %v, got %v", "sdk", cfg.Output)
		}
		if !(cfg.Diff.Enabled) {
			t.Fatal("expected true")
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestSdkcfg_LoadProjectConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = LoadProjectConfig(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdkcfg_LoadProjectConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = LoadProjectConfig(io.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdkcfg_LoadProjectConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = LoadProjectConfig(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
