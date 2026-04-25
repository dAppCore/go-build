package sdk

import (
	"context"
	"testing"

	"dappco.re/go/build/internal/ax"

	"dappco.re/go/build/pkg/sdk/generators"
)

// --- SDK Generation Orchestration Tests ---

func TestGeneration_SDKGenerateAllLanguages_Good(t *testing.T) {
	t.Run("Generate iterates all configured languages", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a minimal OpenAPI spec
		specPath := ax.Join(tmpDir, "openapi.yaml")
		err := ax.WriteFile(specPath, []byte(minimalSpec), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg := &Config{
			Spec:      "openapi.yaml",
			Languages: []string{"nonexistent-lang"},
			Output:    "sdk",
			Package: PackageConfig{
				Name:    "testclient",
				Version: "1.0.0",
			},
		}
		s := New(tmpDir, cfg)
		s.SetVersion("v1.0.0")

		// Generate should fail on unknown language
		err = s.Generate(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "unknown language") {
			t.Fatalf("expected %v to contain %v", err.Error(), "unknown language")
		}

	})
}

func TestGeneration_SDKGenerateLanguageOutputDir_Good(t *testing.T) {
	t.Run("output directory uses language subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		specPath := ax.Join(tmpDir, "openapi.yaml")
		err := ax.WriteFile(specPath, []byte(minimalSpec), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg := &Config{
			Spec:      "openapi.yaml",
			Languages: []string{"typescript"},
			Output:    "custom-sdk",
			Package: PackageConfig{
				Name:    "my-client",
				Version: "2.0.0",
			},
		}
		s := New(tmpDir, cfg)
		s.SetVersion("v2.0.0")

		// This will fail because generators aren't installed, but we can verify
		// the spec detection works correctly
		specResult, err := s.DetectSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(specPath, specResult) {
			t.Fatalf("want %v, got %v", specPath, specResult)
		}

	})
}

func TestGeneration_SDKGenerateLanguageNoSpec_Bad(t *testing.T) {
	t.Run("fails when no OpenAPI spec exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		s := New(tmpDir, &Config{
			Languages: []string{"typescript"},
			Output:    "sdk",
		})

		err := s.GenerateLanguage(context.Background(), "typescript")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "no OpenAPI spec found") {
			t.Fatalf("expected %v to contain %v", err.Error(), "no OpenAPI spec found")
		}

	})
}

func TestGeneration_SDKGenerateLanguageUnknownLanguage_Bad(t *testing.T) {
	t.Run("fails for unregistered language", func(t *testing.T) {
		tmpDir := t.TempDir()
		specPath := ax.Join(tmpDir, "openapi.yaml")
		err := ax.WriteFile(specPath, []byte(minimalSpec), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := New(tmpDir, nil)
		err = s.GenerateLanguage(context.Background(), "cobol")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "unknown language: cobol") {

			// --- Generator Registry Tests ---
			t.Fatalf("expected %v to contain %v", err.Error(), "unknown language: cobol")
		}

	})
}

func TestGeneration_RegistryRegisterAndGet_Good(t *testing.T) {
	t.Run("register and retrieve all generators", func(t *testing.T) {
		registry := generators.NewRegistry()

		// Verify all languages are registered
		languages := registry.Languages()
		if len(languages) != 4 {
			t.Fatalf("want len %v, got %v", 4, len(languages))
		}
		if !stdlibAssertContains(languages, "typescript") {
			t.Fatalf("expected %v to contain %v", languages, "typescript")
		}
		if !stdlibAssertContains(

			// Verify retrieval
			languages, "python") {
			t.Fatalf("expected %v to contain %v", languages, "python")
		}
		if !stdlibAssertContains(languages, "go") {
			t.Fatalf("expected %v to contain %v", languages, "go")
		}
		if !stdlibAssertContains(languages, "php") {
			t.Fatalf("expected %v to contain %v", languages, "php")
		}

		for _, lang := range []string{"typescript", "python", "go", "php"} {
			gen, ok := registry.Get(lang)
			if !(ok) {
				t.Fatalf("should find generator for %s", lang)
			}
			if !stdlibAssertEqual(lang, gen.Language()) {
				t.Fatalf("want %v, got %v", lang, gen.Language())
			}

		}
	})

	t.Run("Get returns false for unregistered language", func(t *testing.T) {
		registry := generators.NewRegistry()
		gen, ok := registry.Get("rust")
		if ok {
			t.Fatal("expected false")
		}
		if !stdlibAssertNil(gen) {
			t.Fatalf("expected nil, got %v", gen)
		}

	})
}

func TestGeneration_RegistryDefaults_Good(t *testing.T) {
	registry := generators.NewRegistry()

	languages := registry.Languages()
	if len(languages) != 4 {
		t.Fatalf("want len %v, got %v", 4, len(languages))
	}
	if !stdlibAssertContains(languages, "typescript") {
		t.Fatalf("expected %v to contain %v", languages, "typescript")
	}
	if !stdlibAssertContains(languages, "python") {
		t.Fatalf("expected %v to contain %v", languages, "python")
	}
	if !stdlibAssertContains(languages, "go") {
		t.Fatalf("expected %v to contain %v", languages, "go")
	}
	if !stdlibAssertContains(languages, "php") {
		t.Fatalf(

			// register again
			"expected %v to contain %v", languages, "php")
	}

}

func TestGeneration_RegistryOverwritesDuplicateLanguage_Good(t *testing.T) {
	registry := generators.NewRegistry()
	registry.Register(generators.NewTypeScriptGenerator())
	registry.Register(generators.NewTypeScriptGenerator())

	languages := registry.Languages()
	count := 0
	for _, lang := range languages {
		if lang == "typescript" {
			count++
		}
	}
	if !stdlibAssertEqual(1, count) {
		t.Fatal("should have exactly one typescript entry")

		// --- Generator Interface Compliance Tests ---
	}

}

func TestGeneration_GeneratorsLanguageIdentifiers_Good(t *testing.T) {
	tests := []struct {
		generator generators.Generator
		expected  string
	}{
		{generators.NewTypeScriptGenerator(), "typescript"},
		{generators.NewPythonGenerator(), "python"},
		{generators.NewGoGenerator(), "go"},
		{generators.NewPHPGenerator(), "php"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if !stdlibAssertEqual(tc.expected, tc.generator.Language()) {
				t.Fatalf("want %v, got %v", tc.expected, tc.generator.Language())
			}

		})
	}
}

func TestGeneration_GeneratorsInstallInstructions_Good(t *testing.T) {
	tests := []struct {
		language string
		gen      generators.Generator
		contains string
	}{
		{"typescript", generators.NewTypeScriptGenerator(), "npm install"},
		{"python", generators.NewPythonGenerator(), "pip install"},
		{"go", generators.NewGoGenerator(), "go install"},
		{"php", generators.NewPHPGenerator(), "Docker"},
	}

	for _, tc := range tests {
		t.Run(tc.language, func(t *testing.T) {
			instructions := tc.gen.Install()
			if stdlibAssertEmpty(instructions) {
				t.Fatal("expected non-empty")
			}
			if !stdlibAssertContains(instructions, tc.contains) {
				t.Fatalf("expected %v to contain %v", instructions, tc.contains)

				// Available() should never panic regardless of system state
			}

		})
	}
}

func TestGeneration_GeneratorsAvailableDoesNotPanic_Good(t *testing.T) {

	gens := []generators.Generator{
		generators.NewTypeScriptGenerator(),
		generators.NewPythonGenerator(),
		generators.NewGoGenerator(),
		generators.NewPHPGenerator(),
	}

	for _, gen := range gens {
		t.Run(gen.Language(), func(t *testing.T) {
			// Should not panic — result depends on system
			_ = gen.Available()
		})
	}
}

// --- SDK Config Tests ---

func TestGeneration_SDKConfigDefaultConfig_Good(t *testing.T) {
	t.Run("default config has all four languages", func(t *testing.T) {
		cfg := DefaultConfig()
		if !stdlibAssertContains(cfg.Languages, "typescript") {
			t.Fatalf("expected %v to contain %v", cfg.Languages, "typescript")
		}
		if !stdlibAssertContains(cfg.Languages, "python") {
			t.Fatalf("expected %v to contain %v", cfg.Languages, "python")
		}
		if !stdlibAssertContains(cfg.Languages, "go") {
			t.Fatalf("expected %v to contain %v", cfg.Languages, "go")
		}
		if !stdlibAssertContains(cfg.Languages, "php") {
			t.Fatalf("expected %v to contain %v", cfg.Languages, "php")
		}
		if len(cfg.Languages) != 4 {
			t.Fatalf("want len %v, got %v", 4, len(cfg.Languages))
		}

	})

	t.Run("default config enables diff", func(t *testing.T) {
		cfg := DefaultConfig()
		if !(cfg.Diff.Enabled) {
			t.Fatal("expected true")
		}
		if cfg.Diff.FailOnBreaking {
			t.Fatal("expected false")
		}

	})

	t.Run("default config uses sdk/ output", func(t *testing.T) {
		cfg := DefaultConfig()
		if !stdlibAssertEqual("sdk", cfg.Output) {
			t.Fatalf("want %v, got %v", "sdk", cfg.Output)
		}

	})
}

func TestGeneration_SDKConfigSetVersion_Good(t *testing.T) {
	t.Run("SetVersion updates both fields", func(t *testing.T) {
		s := New("/tmp", &Config{
			Package: PackageConfig{
				Name:    "test",
				Version: "old",
			},
		})
		s.SetVersion("v3.0.0")
		if !stdlibAssertEqual("v3.0.0", s.version) {
			t.Fatalf("want %v, got %v", "v3.0.0", s.version)
		}
		if !stdlibAssertEqual("v3.0.0", s.config.Package.Version) {
			t.Fatalf("want %v, got %v", "v3.0.0", s.

				// Should not panic
				config.Package.Version)
		}

	})

	t.Run("SetVersion on nil config is safe", func(t *testing.T) {
		s := &SDK{}

		s.SetVersion("v1.0.0")
		if !stdlibAssertEqual("v1.0.0", s.version) {
			t.Fatalf("want %v, got %v", "v1.0.0", s.version)
		}

	})
}

func TestGeneration_SDKConfigNewWithNilConfig_Good(t *testing.T) {
	s := New("/project", nil)
	if stdlibAssertNil(s.config) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("sdk", s.config.Output) {
		t.Fatalf("want %v, got %v", "sdk", s.config.Output)
	}
	if !(s.config.Diff.Enabled) {
		t.Fatal("expected true")
	}

}

func TestGeneration_SDKOutputDirWithPublishPath_Good(t *testing.T) {
	s := New("/project", &Config{
		Output: "generated",
		Publish: PublishConfig{
			Path: "packages/api-client",
		},
	})
	if !stdlibAssertEqual(ax.Join("/project", "packages/api-client", "generated", "typescript"), s.outputDir("typescript")) {
		t.Fatalf(

			// --- Spec Detection Integration Tests ---
			"want %v, got %v", ax.Join("/project", "packages/api-client", "generated", "typescript"), s.outputDir("typescript"))
	}

}

func TestGeneration_SpecDetectionPriority_Good(t *testing.T) {
	t.Run("configured spec takes priority over common paths", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create both a common path spec and a configured spec
		commonSpec := ax.Join(tmpDir, "openapi.yaml")
		err := ax.WriteFile(commonSpec, []byte(minimalSpec), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configuredSpec := ax.Join(tmpDir, "custom", "api.yaml")
		if err := ax.MkdirAll(ax.Dir(configuredSpec), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(configuredSpec, []byte(minimalSpec), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := New(tmpDir, &Config{Spec: "custom/api.yaml"})
		specPath, err := s.DetectSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(configuredSpec, specPath) {
			t.Fatalf("want %v, got %v", configuredSpec, specPath)
		}

	})

	t.Run("common paths checked in order", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create the second common path only (api/openapi.yaml is first)
		apiDir := ax.Join(tmpDir, "api")
		if err := ax.MkdirAll(apiDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiSpec := ax.Join(apiDir, "openapi.json")
		err := ax.WriteFile(apiSpec, []byte(`{"openapi":"3.0.0"}`), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := New(tmpDir, nil)
		specPath, err := s.DetectSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(apiSpec, specPath) {
			t.Fatalf("want %v, got %v", apiSpec, specPath)
		}

	})
}

func TestGeneration_SpecDetectionAllCommonPaths_Good(t *testing.T) {
	for _, commonPath := range commonSpecPaths {
		t.Run(commonPath, func(t *testing.T) {
			tmpDir := t.TempDir()

			specPath := ax.Join(tmpDir, commonPath)
			if err := ax.MkdirAll(ax.Dir(specPath), 0755); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err := ax.WriteFile(specPath, []byte(minimalSpec), 0644)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			s := New(tmpDir, nil)
			detected, err := s.DetectSpec()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !stdlibAssertEqual(specPath, detected) {

				// --- Compile-time interface checks ---
				t.Fatalf("want %v, got %v", specPath, detected)
			}

		})
	}
}

var _ generators.Generator = (*generators.TypeScriptGenerator)(nil)
var _ generators.Generator = (*generators.PythonGenerator)(nil)
var _ generators.Generator = (*generators.GoGenerator)(nil)
var _ generators.Generator = (*generators.PHPGenerator)(nil)

// minimalSpec is a valid OpenAPI 3.0 spec used across tests.
const minimalSpec = `openapi: "3.0.0"
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
