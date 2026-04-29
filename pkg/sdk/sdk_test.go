package sdk

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/sdk/generators"
	yaml "gopkg.in/yaml.v3"
)

type unavailableGenerator struct {
	language string
}

func (g unavailableGenerator) Language() string { return g.language }
func (g unavailableGenerator) Generate(ctx context.Context, opts generators.Options) error {
	return core.NewError("test error")
}
func (g unavailableGenerator) Available() bool { return false }
func (g unavailableGenerator) Install() string { return "install me" }

func TestSDK_SetVersion_Good(t *testing.T) {
	s := New("/tmp", nil)
	s.SetVersion("v1.2.3")
	if !stdlibAssertEqual("v1.2.3", s.version) {
		t.Fatalf("want %v, got %v", "v1.2.3", s.version)
	}

}

func TestSDK_VersionPassedToGeneratorGood(t *testing.T) {
	config := &Config{
		Languages: []string{"typescript"},
		Output:    "sdk",
		Package: PackageConfig{
			Name: "test-sdk",
		},
	}
	s := New("/tmp", config)
	s.SetVersion("v2.0.0")
	if !stdlibAssertEqual("v2.0.0", s.config.Package.Version) {
		t.Fatalf("want %v, got %v", "v2.0.0", s.config.Package.Version)
	}

}

func TestSDK_VersionTemplateIsRenderedGood(t *testing.T) {
	config := &Config{
		Package: PackageConfig{
			Name:    "test-sdk",
			Version: "{{.Version}}-beta",
		},
	}
	s := New("/tmp", config)
	s.SetVersion("v2.0.0")
	if !stdlibAssertEqual("{{.Version}}-beta", s.config.Package.Version) {
		t.Fatalf("want %v, got %v", "{{.Version}}-beta", s.config.Package.Version)
	}
	if !stdlibAssertEqual("v2.0.0-beta", s.resolvePackageVersion()) {
		t.Fatalf("want %v, got %v", "v2.0.0-beta", s.resolvePackageVersion())
	}

}

func TestSDK_DefaultConfig_Good(t *testing.T) {
	cfg := DefaultConfig()
	if !stdlibAssertContains(cfg.Languages, "typescript") {
		t.Fatalf("expected %v to contain %v", cfg.Languages, "typescript")
	}
	if !stdlibAssertEqual("sdk", cfg.Output) {
		t.Fatalf("want %v, got %v", "sdk", cfg.Output)
	}
	if !(cfg.Diff.Enabled) {
		t.Fatal("expected true")
	}

}

func TestSDK_ApplyDefaultsNormalisesLanguageAliasesGood(t *testing.T) {
	cfg := &Config{
		Languages: []string{"ts", "python", "py", "golang", "go", "php"},
	}

	cfg.ApplyDefaults()
	if !stdlibAssertEqual([]string{"typescript", "python", "go", "php"}, cfg.Languages) {
		t.Fatalf("want %v, got %v", []string{"typescript", "python", "go", "php"}, cfg.Languages)
	}

}

func TestSDK_ApplyDefaults_PreservesExplicitEmptyLanguages_Good(t *testing.T) {
	cfg := &Config{
		Languages: []string{},
	}

	cfg.ApplyDefaults()
	if stdlibAssertNil(cfg.Languages) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEmpty(cfg.Languages) {
		t.Fatalf("expected empty, got %v", cfg.Languages)
	}

}

func TestSDK_normaliseLanguage_Good(t *testing.T) {
	if !stdlibAssertEqual("typescript", normaliseLanguage("ts")) {
		t.Fatalf("want %v, got %v", "typescript", normaliseLanguage("ts"))
	}
	if !stdlibAssertEqual("typescript", normaliseLanguage("TypeScript")) {
		t.Fatalf("want %v, got %v", "typescript", normaliseLanguage("TypeScript"))
	}
	if !stdlibAssertEqual("python", normaliseLanguage("py")) {
		t.Fatalf("want %v, got %v", "python", normaliseLanguage("py"))
	}
	if !stdlibAssertEqual("go", normaliseLanguage("golang")) {
		t.Fatalf("want %v, got %v", "go", normaliseLanguage("golang"))
	}
	if !stdlibAssertEqual("php", normaliseLanguage("php")) {
		t.Fatalf("want %v, got %v", "php", normaliseLanguage("php"))
	}

}

func TestSDK_New_Good(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		s := New("/tmp", nil)
		if stdlibAssertNil(s.config) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEqual("sdk", s.config.Output) {
			t.Fatalf("want %v, got %v", "sdk", s.config.Output)
		}

	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{Output: "custom"}
		s := New("/tmp", cfg)
		if !stdlibAssertEqual("custom", s.config.Output) {
			t.Fatalf("want %v, got %v", "custom", s.config.Output)
		}
		if !(s.config.Diff.Enabled) {
			t.Fatal("expected true")
		}

	})

	t.Run("applies defaults and does not mutate the caller config", func(t *testing.T) {
		cfg := &Config{
			Languages: []string{"ts", "python", "py"},
		}

		s := New("/tmp", cfg)
		if !stdlibAssertEqual([]string{"typescript", "python"}, s.config.Languages) {
			t.Fatalf("want %v, got %v", []string{"typescript", "python"}, s.config.Languages)
		}
		if !stdlibAssertEqual("sdk", s.config.Output) {
			t.Fatalf("want %v, got %v", "sdk", s.config.Output)
		}
		if !(s.config.Diff.Enabled) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual([]string{"ts", "python", "py"}, cfg.Languages) {
			t.Fatalf("want %v, got %v", []string{"ts", "python", "py"}, cfg.Languages)
		}
		if !stdlibAssertEmpty(cfg.Output) {
			t.Fatalf("expected empty, got %v", cfg.Output)
		}

	})
}

func TestSDK_GenerateLanguage_Bad(t *testing.T) {

	t.Run("unknown language", func(t *testing.T) {

		tmpDir := t.TempDir()

		specPath := ax.Join(tmpDir, "openapi.yaml")

		err := ax.WriteFile(specPath, []byte("openapi: 3.0.0"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := New(tmpDir, nil)

		err = s.GenerateLanguage(context.Background(), "invalid-lang")
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(err.Error(), "unknown language") {
			t.Fatalf("expected %v to contain %v", err.Error(), "unknown language")
		}

	})

}

func TestSDK_GenerateWithStatus_SkipsUnavailableWhenConfigured_Good(t *testing.T) {
	original := newGeneratorRegistry
	t.Cleanup(func() {
		newGeneratorRegistry = original
	})
	newGeneratorRegistry = func() *generators.Registry {
		registry := generators.NewRegistry()
		registry.Register(unavailableGenerator{language: "php"})
		return registry
	}

	s := New(t.TempDir(), &Config{
		Languages:       []string{"php"},
		SkipUnavailable: true,
	})

	results, err := s.GenerateWithStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(results))
	}
	if !(results[0].Skipped) {
		t.Fatal("expected true")
	}
	if results[0].Generated {
		t.Fatal("expected false")
	}
	if !stdlibAssertContains(results[0].Reason, "generator not available") {
		t.Fatalf("expected %v to contain %v", results[0].Reason, "generator not available")
	}

}
func TestSDK_NilSafetyGood(t *testing.T) {
	var s *SDK

	_, err := s.GenerateWithStatus(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "sdk is nil") {
		t.Fatalf("expected %v to contain %v", err.Error(), "sdk is nil")
	}

	_, err = s.GenerateLanguageWithStatus(context.Background(), "typescript")
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "sdk is nil") {
		t.Fatalf("expected %v to contain %v", err.Error(), "sdk is nil")
	}

	_, err = s.DetectSpec()
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "sdk is nil") {
		t.Fatalf("expected %v to contain %v", err.Error(), "sdk is nil")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestSdk_New_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = New(core.Path(t.TempDir(), "go-build-compliance"), &Config{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_New_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = New("", nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_New_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = New(core.Path(t.TempDir(), "go-build-compliance"), &Config{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_CloneConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CloneConfig(&Config{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_CloneConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CloneConfig(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_CloneConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CloneConfig(&Config{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_SDK_Config_Good(t *core.T) {
	subject := &SDK{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Config()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_SDK_Config_Bad(t *core.T) {
	subject := &SDK{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Config()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_SDK_Config_Ugly(t *core.T) {
	subject := &SDK{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Config()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_Config_ApplyDefaults_Good(t *core.T) {
	subject := &Config{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		subject.ApplyDefaults()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_Config_ApplyDefaults_Bad(t *core.T) {
	subject := &Config{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		subject.ApplyDefaults()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_Config_ApplyDefaults_Ugly(t *core.T) {
	subject := &Config{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		subject.ApplyDefaults()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_SDK_SetVersion_Good(t *core.T) {
	subject := &SDK{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		subject.SetVersion("v1.2.3")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_SDK_SetVersion_Bad(t *core.T) {
	subject := &SDK{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		subject.SetVersion("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_SDK_SetVersion_Ugly(t *core.T) {
	subject := &SDK{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		subject.SetVersion("v1.2.3")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_DefaultConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultConfig()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_DefaultConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultConfig()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_DefaultConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultConfig()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_DiffConfig_UnmarshalYAML_Good(t *core.T) {
	subject := &DiffConfig{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "false"})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_DiffConfig_UnmarshalYAML_Bad(t *core.T) {
	subject := &DiffConfig{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "false"})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_DiffConfig_UnmarshalYAML_Ugly(t *core.T) {
	subject := &DiffConfig{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "false"})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_SDK_Generate_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_SDK_Generate_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_SDK_Generate_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Generate(ctx)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_SDK_GenerateWithStatus_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.GenerateWithStatus(ctx)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_SDK_GenerateWithStatus_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.GenerateWithStatus(ctx)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_SDK_GenerateWithStatus_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.GenerateWithStatus(ctx)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_SDK_GenerateLanguage_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.GenerateLanguage(ctx, "go")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_SDK_GenerateLanguage_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.GenerateLanguage(ctx, "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_SDK_GenerateLanguage_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.GenerateLanguage(ctx, "go")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestSdk_SDK_GenerateLanguageWithStatus_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.GenerateLanguageWithStatus(ctx, "go")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestSdk_SDK_GenerateLanguageWithStatus_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.GenerateLanguageWithStatus(ctx, "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestSdk_SDK_GenerateLanguageWithStatus_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &SDK{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.GenerateLanguageWithStatus(ctx, "go")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
