package build

import (
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
