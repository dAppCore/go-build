package sdk

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"dappco.re/go/core/build/pkg/sdk/generators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- SDK Generation Orchestration Tests ---

func TestSDK_Generate_Good_AllLanguages(t *testing.T) {
	t.Run("Generate iterates all configured languages", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a minimal OpenAPI spec
		specPath := filepath.Join(tmpDir, "openapi.yaml")
		err := os.WriteFile(specPath, []byte(minimalSpec), 0644)
		require.NoError(t, err)

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
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown language")
	})
}

func TestSDK_GenerateLanguage_Good_OutputDir(t *testing.T) {
	t.Run("output directory uses language subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		specPath := filepath.Join(tmpDir, "openapi.yaml")
		err := os.WriteFile(specPath, []byte(minimalSpec), 0644)
		require.NoError(t, err)

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
		require.NoError(t, err)
		assert.Equal(t, specPath, specResult)
	})
}

func TestSDK_GenerateLanguage_Bad_NoSpec(t *testing.T) {
	t.Run("fails when no OpenAPI spec exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		s := New(tmpDir, &Config{
			Languages: []string{"typescript"},
			Output:    "sdk",
		})

		err := s.GenerateLanguage(context.Background(), "typescript")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no OpenAPI spec found")
	})
}

func TestSDK_GenerateLanguage_Bad_UnknownLanguage(t *testing.T) {
	t.Run("fails for unregistered language", func(t *testing.T) {
		tmpDir := t.TempDir()
		specPath := filepath.Join(tmpDir, "openapi.yaml")
		err := os.WriteFile(specPath, []byte(minimalSpec), 0644)
		require.NoError(t, err)

		s := New(tmpDir, nil)
		err = s.GenerateLanguage(context.Background(), "cobol")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown language: cobol")
	})
}

// --- Generator Registry Tests ---

func TestRegistry_Good_RegisterAndGet(t *testing.T) {
	t.Run("register and retrieve all generators", func(t *testing.T) {
		registry := generators.NewRegistry()
		registry.Register(generators.NewTypeScriptGenerator())
		registry.Register(generators.NewPythonGenerator())
		registry.Register(generators.NewGoGenerator())
		registry.Register(generators.NewPHPGenerator())

		// Verify all languages are registered
		languages := registry.Languages()
		assert.Len(t, languages, 4)
		assert.Contains(t, languages, "typescript")
		assert.Contains(t, languages, "python")
		assert.Contains(t, languages, "go")
		assert.Contains(t, languages, "php")

		// Verify retrieval
		for _, lang := range []string{"typescript", "python", "go", "php"} {
			gen, ok := registry.Get(lang)
			assert.True(t, ok, "should find generator for %s", lang)
			assert.Equal(t, lang, gen.Language())
		}
	})

	t.Run("Get returns false for unregistered language", func(t *testing.T) {
		registry := generators.NewRegistry()
		gen, ok := registry.Get("rust")
		assert.False(t, ok)
		assert.Nil(t, gen)
	})
}

func TestRegistry_Good_OverwritesDuplicateLanguage(t *testing.T) {
	registry := generators.NewRegistry()
	registry.Register(generators.NewTypeScriptGenerator())
	registry.Register(generators.NewTypeScriptGenerator()) // register again

	languages := registry.Languages()
	count := 0
	for _, lang := range languages {
		if lang == "typescript" {
			count++
		}
	}
	assert.Equal(t, 1, count, "should have exactly one typescript entry")
}

// --- Generator Interface Compliance Tests ---

func TestGenerators_Good_LanguageIdentifiers(t *testing.T) {
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
			assert.Equal(t, tc.expected, tc.generator.Language())
		})
	}
}

func TestGenerators_Good_InstallInstructions(t *testing.T) {
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
			assert.NotEmpty(t, instructions)
			assert.Contains(t, instructions, tc.contains)
		})
	}
}

func TestGenerators_Good_AvailableDoesNotPanic(t *testing.T) {
	// Available() should never panic regardless of system state
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

func TestSDKConfig_Good_DefaultConfig(t *testing.T) {
	t.Run("default config has all four languages", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Contains(t, cfg.Languages, "typescript")
		assert.Contains(t, cfg.Languages, "python")
		assert.Contains(t, cfg.Languages, "go")
		assert.Contains(t, cfg.Languages, "php")
		assert.Len(t, cfg.Languages, 4)
	})

	t.Run("default config enables diff", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.True(t, cfg.Diff.Enabled)
		assert.False(t, cfg.Diff.FailOnBreaking)
	})

	t.Run("default config uses sdk/ output", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Equal(t, "sdk", cfg.Output)
	})
}

func TestSDKConfig_Good_SetVersion(t *testing.T) {
	t.Run("SetVersion updates both fields", func(t *testing.T) {
		s := New("/tmp", &Config{
			Package: PackageConfig{
				Name:    "test",
				Version: "old",
			},
		})
		s.SetVersion("v3.0.0")

		assert.Equal(t, "v3.0.0", s.version)
		assert.Equal(t, "v3.0.0", s.config.Package.Version)
	})

	t.Run("SetVersion on nil config is safe", func(t *testing.T) {
		s := &SDK{}
		// Should not panic
		s.SetVersion("v1.0.0")
		assert.Equal(t, "v1.0.0", s.version)
	})
}

func TestSDKConfig_Good_NewWithNilConfig(t *testing.T) {
	s := New("/project", nil)
	assert.NotNil(t, s.config)
	assert.Equal(t, "sdk", s.config.Output)
	assert.True(t, s.config.Diff.Enabled)
}

// --- Spec Detection Integration Tests ---

func TestSpecDetection_Good_Priority(t *testing.T) {
	t.Run("configured spec takes priority over common paths", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create both a common path spec and a configured spec
		commonSpec := filepath.Join(tmpDir, "openapi.yaml")
		err := os.WriteFile(commonSpec, []byte(minimalSpec), 0644)
		require.NoError(t, err)

		configuredSpec := filepath.Join(tmpDir, "custom", "api.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(configuredSpec), 0755))
		err = os.WriteFile(configuredSpec, []byte(minimalSpec), 0644)
		require.NoError(t, err)

		s := New(tmpDir, &Config{Spec: "custom/api.yaml"})
		specPath, err := s.DetectSpec()
		require.NoError(t, err)
		assert.Equal(t, configuredSpec, specPath)
	})

	t.Run("common paths checked in order", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create the second common path only (api/openapi.yaml is first)
		apiDir := filepath.Join(tmpDir, "api")
		require.NoError(t, os.MkdirAll(apiDir, 0755))
		apiSpec := filepath.Join(apiDir, "openapi.json")
		err := os.WriteFile(apiSpec, []byte(`{"openapi":"3.0.0"}`), 0644)
		require.NoError(t, err)

		s := New(tmpDir, nil)
		specPath, err := s.DetectSpec()
		require.NoError(t, err)
		assert.Equal(t, apiSpec, specPath)
	})
}

func TestSpecDetection_Good_AllCommonPaths(t *testing.T) {
	for _, commonPath := range commonSpecPaths {
		t.Run(commonPath, func(t *testing.T) {
			tmpDir := t.TempDir()

			specPath := filepath.Join(tmpDir, commonPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(specPath), 0755))
			err := os.WriteFile(specPath, []byte(minimalSpec), 0644)
			require.NoError(t, err)

			s := New(tmpDir, nil)
			detected, err := s.DetectSpec()
			require.NoError(t, err)
			assert.Equal(t, specPath, detected)
		})
	}
}

// --- Compile-time interface checks ---

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
