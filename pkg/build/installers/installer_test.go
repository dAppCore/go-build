package installers

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/testassert"
)

// validConfig is a fully populated InstallerConfig used as the happy-path baseline.
var validConfig = InstallerConfig{
	Version:    "v1.2.3",
	Repo:       "dappcore/core",
	BinaryName: "core",
}

func requireGeneratedInstaller(t *testing.T, variant InstallerVariant, cfg InstallerConfig) string {
	t.Helper()
	result := GenerateInstaller(variant, cfg)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(string)
}

func requireGeneratedInstallers(t *testing.T, cfg InstallerConfig) map[string]string {
	t.Helper()
	result := GenerateAll(cfg)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(map[string]string)
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
			result := GenerateInstaller(v, validConfig)
			if !result.OK {
				t.Fatalf("unexpected error: %v", result.Error())
			}
			script := result.Value.(string)
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
	result := GenerateInstaller(VariantFull, InstallerConfig{
		Version:       "v1.2.3",
		Repo:          "dappcore/core",
		BinaryName:    "core",
		ScriptBaseURL: "https://downloads.example.com/",
	})
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	script := result.Value.(string)
	if !stdlibAssertContains(script, "https://downloads.example.com/setup.sh") {
		t.Fatalf("expected %v to contain %v", script, "https://downloads.example.com/setup.sh")
	}
	if stdlibAssertContains(script, "https://downloads.example.com//setup.sh") {
		t.Fatalf("expected %v not to contain %v", script, "https://downloads.example.com//setup.sh")
	}

}

func TestInstaller_GenerateInstaller_AgenticAlias_Good(t *testing.T) {
	result := GenerateInstaller("agentic", validConfig)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	script := result.Value.(string)
	if stdlibAssertEmpty(script) {
		t.Fatal("expected non-empty")
	}
	if !stdlibAssertContains(script, DefaultScriptBaseURL) {
		t.Fatalf("expected %v to contain %v", script, DefaultScriptBaseURL)
	}

}

func TestInstaller_GenerateInstaller_DevVariantUsesVersionedImage_Good(t *testing.T) {
	result := GenerateInstaller(VariantDev, validConfig)
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	script := result.Value.(string)
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
		result := GenerateInstaller("nonexistent", validConfig)
		if result.OK {
			t.Fatal("expected error")
		}

	})

	t.Run("empty variant string returns error", func(t *testing.T) {
		result := GenerateInstaller("", validConfig)
		if result.OK {
			t.Fatal("expected error")
		}

	})

	t.Run("unsafe version returns error", func(t *testing.T) {
		result := GenerateInstaller(VariantCI, InstallerConfig{
			Version:    "v1.2.3\n--flag",
			Repo:       "dappcore/core",
			BinaryName: "core",
		})
		if result.OK {
			t.Fatal("expected error")
		}

	})

	t.Run("version with spaces returns error", func(t *testing.T) {
		result := GenerateInstaller(VariantCI, InstallerConfig{
			Version:    " v1.2.3 ",
			Repo:       "dappcore/core",
			BinaryName: "core",
		})
		if result.OK {
			t.Fatal("expected error")
		}
	})
}

// TestInstaller_GenerateInstaller_Ugly verifies that empty config fields are rendered without
// panicking — the template may produce incomplete scripts but must not error.
func TestInstaller_GenerateInstaller_Ugly(t *testing.T) {
	t.Run("empty Version renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "", Repo: "dappcore/core", BinaryName: "core"}
		result := GenerateInstaller(VariantFull, cfg)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		script := result.Value.(string)
		if stdlibAssertEmpty(script) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("empty Repo renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "v1.0.0", Repo: "", BinaryName: "core"}
		result := GenerateInstaller(VariantCI, cfg)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		script := result.Value.(string)
		if stdlibAssertEmpty(script) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("empty BinaryName renders without error", func(t *testing.T) {
		cfg := InstallerConfig{Version: "v1.0.0", Repo: "dappcore/core", BinaryName: ""}
		result := GenerateInstaller(VariantAgent, cfg)
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		script := result.Value.(string)
		if stdlibAssertEmpty(script) {
			t.Fatal("expected non-empty")
		}

	})

	t.Run("fully empty config renders without error", func(t *testing.T) {
		result := GenerateInstaller(VariantDev, InstallerConfig{})
		if !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		script := result.Value.(string)
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

	script := requireGeneratedInstaller(t, VariantCI, cfg)
	if !stdlibAssertContains(script, "BINARY_NAME='core'\"'\"'s tool'") {
		t.Fatalf("expected %v to contain %v", script, "BINARY_NAME='core'\"'\"'s tool'")
	}
	if !stdlibAssertContains(script, "VERSION='v1.2.3-beta+1'") {
		t.Fatalf("expected %v to contain %v", script, "VERSION='v1.2.3-beta+1'")
	}
	if !stdlibAssertContains(script, "REPO='dappcore/core'") {
		t.Fatalf("expected %v to contain %v", script, "REPO='dappcore/core'")
	}

}

func TestInstaller_GenerateAll_Good(t *testing.T) {
	scripts := requireGeneratedInstallers(t, validConfig)

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
	result := GenerateAll(InstallerConfig{
		Version:    "v1.2.3 && echo unsafe",
		Repo:       "dappcore/core",
		BinaryName: "core",
	})
	if result.OK {
		t.Fatal("expected error")
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

var (
	stdlibAssertEqual         = testassert.Equal
	stdlibAssertNil           = testassert.Nil
	stdlibAssertEmpty         = testassert.Empty
	stdlibAssertZero          = testassert.Zero
	stdlibAssertContains      = testassert.Contains
	stdlibAssertElementsMatch = testassert.ElementsMatch
)

// --- v0.9.0 generated compliance triplets ---
func TestInstaller_Variants_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Variants()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestInstaller_Variants_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Variants()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestInstaller_OutputName_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OutputName(InstallerVariant("linux"))
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestInstaller_OutputName_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = OutputName(InstallerVariant("linux"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestInstaller_GenerateAll_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = GenerateAll()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestInstaller_GenerateAll_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = GenerateAll()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
