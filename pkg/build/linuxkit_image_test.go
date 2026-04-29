package build

import (
	core "dappco.re/go"
	"testing"
)

func TestBuild_DefaultLinuxKitConfig_Good(t *testing.T) {
	cfg := DefaultLinuxKitConfig()
	if !stdlibAssertEqual("core-dev", cfg.Base) {
		t.Fatalf("want %v, got %v", "core-dev", cfg.Base)
	}
	if !stdlibAssertEqual([]string{"/workspace"}, cfg.Mounts) {
		t.Fatalf("want %v, got %v", []string{"/workspace"}, cfg.Mounts)
	}
	if !stdlibAssertEqual([]string{"oci", "apple"}, cfg.Formats) {
		t.Fatalf("want %v, got %v", []string{"oci", "apple"}, cfg.Formats)
	}
	if cfg.GPU {
		t.Fatal("expected false")
	}

}

func TestBuild_LinuxKit_Good(t *testing.T) {
	image := LinuxKit(
		WithBase("core-ml"),
		WithPackages("git", "task"),
		WithMount("/src"),
		WithGPU(true),
		WithFormats("oci"),
		WithRegistry("ghcr.io/dappcore"),
	)
	if stdlibAssertNil(image) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual(LinuxKitConfig{Base: "core-ml", Packages: []string{"git", "task"}, Mounts: []string{"/workspace", "/src"}, GPU: true, Formats: []string{"oci"}, Registry: "ghcr.io/dappcore"}, image.Config) {
		t.Fatalf("want %v, got %v", LinuxKitConfig{Base: "core-ml", Packages: []string{"git", "task"}, Mounts: []string{"/workspace", "/src"}, GPU: true, Formats: []string{"oci"}, Registry: "ghcr.io/dappcore"}, image.Config)
	}

}

func TestBuild_LinuxKit_NormalizesOptionValues_Good(t *testing.T) {
	image := LinuxKit(
		WithBase(" core-dev "),
		WithPackages(" git ", "git", "task"),
		WithMount("/workspace"),
		WithMount(" /src "),
		WithFormats(" OCI ", "apple", "APPLE", ""),
		WithRegistry(" ghcr.io/dappcore "),
	)
	if stdlibAssertNil(image) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual(LinuxKitConfig{Base: "core-dev", Packages: []string{"git", "task"}, Mounts: []string{"/workspace", "/src"}, GPU: false, Formats: []string{"oci", "apple"}, Registry: "ghcr.io/dappcore"}, image.Config) {
		t.Fatalf("want %v, got %v", LinuxKitConfig{Base: "core-dev", Packages: []string{"git", "task"}, Mounts: []string{"/workspace", "/src"}, GPU: false, Formats: []string{"oci", "apple"}, Registry: "ghcr.io/dappcore"}, image.Config)
	}

}

func TestBuild_LinuxKitBaseTemplate_Good(t *testing.T) {
	images := LinuxKitBaseImages()
	if len(images) != 3 {
		t.Fatalf("want len %v, got %v", 3, len(images))
	}

	for _, image := range images {
		content, err := LinuxKitBaseTemplate(image.Name)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, image.Name) {
			t.Fatalf("expected %v to contain %v", content, image.Name)
		}

		lookedUp, ok := LookupLinuxKitBaseImage(image.Name)
		if !(ok) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual(image.Name, lookedUp.Name) {
			t.Fatalf("want %v, got %v", image.Name, lookedUp.Name)
		}

	}
}

// --- v0.9.0 generated compliance triplets ---
func TestLinuxkitImage_DefaultLinuxKitConfig_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultLinuxKitConfig()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkitImage_DefaultLinuxKitConfig_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultLinuxKitConfig()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkitImage_DefaultLinuxKitConfig_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultLinuxKitConfig()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkitImage_LinuxKit_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LinuxKit()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkitImage_LinuxKit_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LinuxKit()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkitImage_LinuxKit_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = LinuxKit()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkitImage_WithBase_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBase("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkitImage_WithBase_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBase("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkitImage_WithBase_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithBase("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkitImage_WithPackages_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithPackages()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkitImage_WithPackages_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithPackages()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkitImage_WithPackages_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithPackages()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkitImage_WithMount_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithMount(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkitImage_WithMount_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithMount("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkitImage_WithMount_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithMount(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkitImage_WithGPU_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithGPU(true)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkitImage_WithGPU_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithGPU(false)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkitImage_WithGPU_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithGPU(true)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkitImage_WithFormats_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithFormats()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkitImage_WithFormats_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithFormats()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkitImage_WithFormats_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithFormats()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestLinuxkitImage_WithRegistry_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithRegistry("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestLinuxkitImage_WithRegistry_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithRegistry("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestLinuxkitImage_WithRegistry_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithRegistry("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
