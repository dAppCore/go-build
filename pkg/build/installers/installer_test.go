package installers

import (
	"reflect"
	"strings"
	"testing"
)

// validConfig is a fully populated InstallerConfig used as the happy-path baseline.
var validConfig = InstallerConfig{
	Version:    "v1.2.3",
	Repo:       "dappcore/core",
	BinaryName: "core",
}

// TestInstaller_GenerateInstaller_Good verifies that each known variant produces a non-empty
// shell script containing the expected shebang, binary name, version, and repo strings.
func TestInstaller_GenerateInstaller_Good(t *testing.T) {
	allVariants := []InstallerVariant{
		VariantFull,
		VariantCI,
		VariantPHP,
		VariantGo,
		VariantAgent,
		VariantDev,
	}

	for _, variant := range allVariants {
		v := variant // capture range variable
		t.Run(string(v), func(t *testing.T) {
			script, err := GenerateInstaller(v, validConfig)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stdlibAssertEmpty(script) {
				t.Fatal("expected non-empty")
			}
			if !stdlibAssertContains(script, "#!/usr/bin/env bash") {
				t.Fatal("script must start with bash shebang")
			}
			if !stdlibAssertContains(script, validConfig.BinaryName) {
				t.Fatal("script must reference binary name")
			}
			if !stdlibAssertContains(script, validConfig.Version) {
				t.Fatal("script must reference version")
			}
			if !stdlibAssertContains(script, validConfig.Repo) {
				t.Fatal("script must reference repo")
			}
			if !stdlibAssertContains(script, DefaultScriptBaseURL) {
				t.Fatal("script must reference the RFC installer host")
			}

		})
	}
}

func TestInstaller_GenerateInstaller_CustomScriptBaseURL_Good(t *testing.T) {
	script, err := GenerateInstaller(VariantFull, InstallerConfig{
		Version:       "v1.2.3",
		Repo:          "dappcore/core",
		BinaryName:    "core",
		ScriptBaseURL: "https://downloads.example.com/",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(script, "https://downloads.example.com/setup.sh") {
		t.Fatalf("expected %v to contain %v", script, "https://downloads.example.com/setup.sh")
	}
	if stdlibAssertContains(script, "https://downloads.example.com//setup.sh") {
		t.Fatalf("expected %v not to contain %v", script, "https://downloads.example.com//setup.sh")
	}

}

func TestInstaller_GenerateInstaller_AgenticAlias_Good(t *testing.T) {
	script, err := GenerateInstaller("agentic", validConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdlibAssertEmpty(script) {
		t.Fatal("expected non-empty")
	}
	if !stdlibAssertContains(script, DefaultScriptBaseURL) {
		t.Fatalf("expected %v to contain %v", script, DefaultScriptBaseURL)
	}

}

func TestInstaller_GenerateInstaller_DevVariantUsesVersionedImage_Good(t *testing.T) {
	script, err := GenerateInstaller(VariantDev, validConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(script, `DEV_IMAGE_VERSION="${VERSION#v}"`) {
		t.Fatalf("expected %v to contain %v", script, `DEV_IMAGE_VERSION="${VERSION#v}"`)
	}
	if !stdlibAssertContains(script, `DEV_IMAGE="ghcr.io/dappcore/core-dev:${DEV_IMAGE_VERSION}"`) {

		// TestInstaller_GenerateInstaller_Bad verifies that an unknown variant returns an error and empty output.
		t.Fatalf("expected %v to contain %v", script, `DEV_IMAGE="ghcr.io/dappcore/core-dev:${DEV_IMAGE_VERSION}"`)
	}
	if stdlibAssertContains(script, "core-dev:latest") {
		t.Fatalf("expected %v not to contain %v", script, "core-dev:latest")
	}

}

func TestInstaller_GenerateInstaller_Bad(t *testing.T) {
	t.Run("unknown variant returns error", func(t *testing.T) {
		script, err := GenerateInstaller("nonexistent", validConfig)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(script) {
			t.Fatalf("expected empty, got %v", script)
		}

	})

	t.Run("empty variant string returns error", func(t *testing.T) {
		script, err := GenerateInstaller("", validConfig)
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(script) {
			t.Fatalf("expected empty, got %v", script)
		}

	})

	t.Run("unsafe version returns error", func(t *testing.T) {
		script, err := GenerateInstaller(VariantCI, InstallerConfig{
			Version:    "v1.2.3\n--flag",
			Repo:       "dappcore/core",
			BinaryName: "core",
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertEmpty(

			// TestInstaller_GenerateInstaller_Ugly verifies that empty config fields are rendered without
			// panicking — the template may produce incomplete scripts but must not error.
			script) {
			t.Fatalf("expected empty, got %v", script)
		}

	})
}

func TestInstaller_GenerateInstaller_Ugly(t *testing.T) {
	t.Run("empty Version renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "", Repo: "dappcore/core", BinaryName: "core"}
		script, err := GenerateInstaller(VariantFull, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(script) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("empty Repo renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "v1.0.0", Repo: "", BinaryName: "core"}
		script, err := GenerateInstaller(VariantCI, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(script) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("empty BinaryName renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "v1.0.0", Repo: "dappcore/core", BinaryName: ""}
		script, err := GenerateInstaller(VariantAgent, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(script) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("fully empty config renders without error", func(t *testing.T) {
		script, err := GenerateInstaller(VariantDev, InstallerConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertEmpty(script) {
			t.Fatal("expected non-empty")
		}

	})
}

func TestInstaller_GenerateInstaller_QuotesValues(t *testing.T) {
	cfg := InstallerConfig{
		Version:    "v1.2.3-beta+1",
		Repo:       "dappcore/core",
		BinaryName: "core's tool",
	}

	script, err := GenerateInstaller(VariantCI, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(script, "BINARY_NAME='core'\"'\"'s tool'") {
		t.Fatalf("expected %v to contain %v", script, "BINARY_NAME='core'\"'\"'s tool'")
	}
	if !stdlibAssertContains(

		// TestInstaller_GenerateAll_Good verifies that GenerateAll returns one entry per variant
		// and that each script is a non-empty bash script.
		script, "VERSION='v1.2.3-beta+1'") {
		t.Fatalf("expected %v to contain %v", script, "VERSION='v1.2.3-beta+1'")
	}
	if !stdlibAssertContains(script, "REPO='dappcore/core'") {
		t.Fatalf("expected %v to contain %v", script, "REPO='dappcore/core'")
	}

}

func TestInstaller_GenerateAll_Good(t *testing.T) {
	scripts, err := GenerateAll(validConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedOutputs := []string{
		"setup.sh",
		"ci.sh",
		"php.sh",
		"go.sh",
		"agent.sh",
		"dev.sh",
	}
	if len(scripts) != len(variantTemplates) {
		t.Fatal("GenerateAll must return one entry per variant")
	}

	for _, name := range expectedOutputs {
		t.Run(name, func(t *testing.T) {
			content, ok := scripts[name]
			if !(ok) {
				t.Fatalf("GenerateAll must include %s", name)
			}
			if stdlibAssertEmpty(content) {
				t.Fatal("expected non-empty")
			}
			if !stdlibAssertContains(content, "#!/usr/bin/env bash") {
				t.Fatalf("expected %v to contain %v", content, "#!/usr/bin/env bash")
			}
			if !stdlibAssertContains(content, validConfig.BinaryName) {
				t.Fatalf("expected %v to contain %v", content, validConfig.BinaryName)
			}
			if !stdlibAssertContains(content, DefaultScriptBaseURL) {
				t.Fatalf("expected %v to contain %v", content, DefaultScriptBaseURL)
			}

		})
	}
}

func TestInstaller_Variants_Good(t *testing.T) {
	if !stdlibAssertEqual([]InstallerVariant{VariantFull, VariantCI, VariantPHP, VariantGo, VariantAgent, VariantDev}, Variants()) {
		t.Fatalf("want %v, got %v", []InstallerVariant{VariantFull, VariantCI, VariantPHP, VariantGo, VariantAgent, VariantDev}, Variants())
	}

}

func TestInstaller_GenerateAll_Bad_UnsafeVersion(t *testing.T) {
	scripts, err := GenerateAll(InstallerConfig{
		Version:    "v1.2.3 && echo unsafe",
		Repo:       "dappcore/core",
		BinaryName: "core",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertNil(scripts) {
		t.Fatalf("expected nil, got %v", scripts)
	}

}

func TestInstaller_OutputName_Good(t *testing.T) {
	if !stdlibAssertEqual("setup.sh", OutputName(VariantFull)) {
		t.Fatalf("want %v, got %v", "setup.sh", OutputName(VariantFull))
	}
	if !stdlibAssertEqual("ci.sh", OutputName(VariantCI)) {
		t.Fatalf("want %v, got %v", "ci.sh", OutputName(VariantCI))
	}
	if !stdlibAssertEqual("php.sh", OutputName(VariantPHP)) {
		t.Fatalf("want %v, got %v", "php.sh", OutputName(VariantPHP))
	}
	if !stdlibAssertEqual("go.sh", OutputName(VariantGo)) {
		t.Fatalf("want %v, got %v", "go.sh", OutputName(VariantGo))
	}
	if !stdlibAssertEqual("agent.sh", OutputName(VariantAgent)) {
		t.Fatalf("want %v, got %v", "agent.sh", OutputName(VariantAgent))
	}
	if !stdlibAssertEqual("agent.sh", OutputName("agentic")) {
		t.Fatalf("want %v, got %v", "agent.sh", OutputName("agentic"))
	}
	if !stdlibAssertEqual("dev.sh", OutputName(VariantDev)) {
		t.Fatalf("want %v, got %v", "dev.sh", OutputName(VariantDev))
	}
	if !stdlibAssertEmpty(OutputName("bogus")) {
		t.Fatalf("expected empty, got %v", OutputName("bogus"))
	}

}

func stdlibAssertEqual(want, got any) bool {
	return reflect.DeepEqual(want, got)
}

func stdlibAssertNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func stdlibAssertEmpty(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

func stdlibAssertZero(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	return !v.IsValid() || v.IsZero()
}

func stdlibAssertContains(container, elem any) bool {
	if s, ok := container.(string); ok {
		sub, ok := elem.(string)
		return ok && strings.Contains(s, sub)
	}

	v := reflect.ValueOf(container)
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(elem)
		if !key.IsValid() {
			return false
		}
		if key.Type().AssignableTo(v.Type().Key()) {
			return v.MapIndex(key).IsValid()
		}
		if key.Type().ConvertibleTo(v.Type().Key()) {
			return v.MapIndex(key.Convert(v.Type().Key())).IsValid()
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if reflect.DeepEqual(v.Index(i).Interface(), elem) {
				return true
			}
		}
	}
	return false
}

func stdlibAssertElementsMatch(want, got any) bool {
	wantValue := reflect.ValueOf(want)
	gotValue := reflect.ValueOf(got)
	if !wantValue.IsValid() || !gotValue.IsValid() {
		return !wantValue.IsValid() && !gotValue.IsValid()
	}
	if !isListValue(wantValue) || !isListValue(gotValue) {
		return reflect.DeepEqual(want, got)
	}
	if wantValue.Len() != gotValue.Len() {
		return false
	}

	used := make([]bool, gotValue.Len())
	for i := 0; i < wantValue.Len(); i++ {
		found := false
		wantElem := wantValue.Index(i).Interface()
		for j := 0; j < gotValue.Len(); j++ {
			if used[j] {
				continue
			}
			if reflect.DeepEqual(wantElem, gotValue.Index(j).Interface()) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func isListValue(value reflect.Value) bool {
	return value.Kind() == reflect.Array || value.Kind() == reflect.Slice
}
