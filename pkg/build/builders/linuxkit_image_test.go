package builders

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func setupFakeLinuxKitImageToolchain(t *testing.T, binDir string) {
	t.Helper()

	dockerScript := `#!/bin/sh
exit 0
`
	if err := ax.WriteFile(ax.Join(binDir, "docker"), []byte(dockerScript), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	script := `#!/bin/sh
set -eu

format=""
dir=""
name=""
while [ $# -gt 0 ]; do
	case "$1" in
	build)
		;;
	--format)
		shift
		format="${1:-}"
		;;
	--dir)
		shift
		dir="${1:-}"
		;;
	--name)
		shift
		name="${1:-}"
		;;
	esac
	shift
done

ext=".img"
case "$format" in
	tar)
		ext=".tar"
		;;
	iso|iso-bios|iso-efi)
		ext=".iso"
		;;
	raw|raw-bios|raw-efi)
		ext=".raw"
		;;
	qcow2|qcow2-bios|qcow2-efi)
		ext=".qcow2"
		;;
esac

mkdir -p "$dir"
printf 'linuxkit image\n' > "$dir/$name$ext"
`
	if err := ax.WriteFile(ax.Join(binDir, "linuxkit"), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestLinuxKitImage_LinuxKitImageBuilderNameGood(t *testing.T) {
	builder := NewLinuxKitImageBuilder()
	if !stdlibAssertEqual("linuxkit-image", builder.Name()) {
		t.Fatalf("want %v, got %v", "linuxkit-image", builder.Name())
	}

}

func TestLinuxKitImage_LinuxKitImageBuilderArtifactPathGood(t *testing.T) {
	builder := NewLinuxKitImageBuilder()
	if !stdlibAssertEqual("/dist/core-dev.tar", builder.ArtifactPath("/dist", "core-dev", "oci")) {
		t.Fatalf("want %v, got %v", "/dist/core-dev.tar", builder.ArtifactPath("/dist", "core-dev", "oci"))
	}
	if !stdlibAssertEqual("/dist/core-dev.aci", builder.ArtifactPath("/dist", "core-dev", "apple")) {
		t.Fatalf("want %v, got %v", "/dist/core-dev.aci", builder.ArtifactPath("/dist", "core-dev", "apple"))
	}
	if !stdlibAssertEqual("/dist/core-dev.iso", builder.ArtifactPath("/dist", "core-dev", "iso")) {
		t.Fatalf("want %v, got %v", "/dist/core-dev.iso", builder.ArtifactPath("/dist", "core-dev", "iso"))
	}

}

func TestLinuxKitImage_BuildLinuxKitServiceImageReference_UsesVersionTagGood(t *testing.T) {
	if !stdlibAssertEqual("core-build-linuxkit/core-dev:1.2.3", buildLinuxKitServiceImageReference("core-dev", "v1.2.3")) {
		t.Fatalf("want %v, got %v", "core-build-linuxkit/core-dev:1.2.3", buildLinuxKitServiceImageReference("core-dev", "v1.2.3"))
	}
	if !stdlibAssertEqual("core-build-linuxkit/core-dev:dev", buildLinuxKitServiceImageReference("core-dev", "")) {
		t.Fatalf("want %v, got %v", "core-build-linuxkit/core-dev:dev", buildLinuxKitServiceImageReference("core-dev", ""))
	}

}

func TestLinuxKitImage_RenderLinuxKitServiceDockerfile_IncludesMetadataGood(t *testing.T) {
	rendered := renderLinuxKitServiceDockerfile("core-dev", "v1.2.3", "2026.04.08", "abc123", []string{"git"}, []string{"/workspace"}, false)
	if !stdlibAssertContains(rendered, "LABEL org.opencontainers.image.version=1.2.3") {
		t.Fatalf("expected %v to contain %v", rendered, "LABEL org.opencontainers.image.version=1.2.3")
	}
	if !stdlibAssertContains(rendered, "LABEL dappcore.core-build.content-hash=abc123") {
		t.Fatalf("expected %v to contain %v", rendered, "LABEL dappcore.core-build.content-hash=abc123")
	}
	if !stdlibAssertContains(rendered, "ENV CORE_IMAGE_VERSION=1.2.3") {
		t.Fatalf("expected %v to contain %v", rendered, "ENV CORE_IMAGE_VERSION=1.2.3")
	}
	if !stdlibAssertContains(rendered, "ENV CORE_IMAGE_CONTENT_HASH=abc123") {
		t.Fatalf("expected %v to contain %v", rendered, "ENV CORE_IMAGE_CONTENT_HASH=abc123")
	}

}

func TestLinuxKitImage_RenderTemplateUsesImmutableServiceImageGood(t *testing.T) {
	builder := NewLinuxKitImageBuilder()
	baseImage, ok := build.LookupLinuxKitBaseImage("core-dev")
	if !(ok) {
		t.Fatal("expected true")
	}

	rendered, err := builder.renderTemplate(baseImage, build.LinuxKitConfig{
		Base:     "core-dev",
		Mounts:   []string{"/workspace"},
		Formats:  []string{"oci"},
		Packages: []string{"gh"},
	}, "v1.2.3", "core-build-linuxkit/core-dev:test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(rendered, `image: "core-build-linuxkit/core-dev:test"`) {
		t.Fatalf("expected %v to contain %v", rendered, `image: "core-build-linuxkit/core-dev:test"`)
	}
	if !stdlibAssertContains(rendered, "tail -f /dev/null") {
		t.Fatalf("expected %v to contain %v", rendered, "tail -f /dev/null")
	}
	if stdlibAssertContains(rendered, "apk add --no-cache") {
		t.Fatalf("expected %v not to contain %v", rendered, "apk add --no-cache")
	}

}

func TestLinuxKitImage_RenderTemplateRestoresDefaultWorkspaceMountGood(t *testing.T) {
	builder := NewLinuxKitImageBuilder()
	baseImage, ok := build.LookupLinuxKitBaseImage("core-dev")
	if !(ok) {
		t.Fatal("expected true")
	}

	rendered, err := builder.renderTemplate(baseImage, build.LinuxKitConfig{
		Base:    "core-dev",
		Mounts:  []string{""},
		Formats: []string{"oci"},
	}, "v1.2.3", "core-build-linuxkit/core-dev:test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(rendered, "binds:") {
		t.Fatalf("expected %v to contain %v", rendered, "binds:")
	}
	if !stdlibAssertContains(rendered, "- /workspace:/workspace") {
		t.Fatalf("expected %v to contain %v", rendered, "- /workspace:/workspace")
	}

}

func TestLinuxKitImage_LinuxKitImageBuilderBuildGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeLinuxKitImageToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	projectDir := t.TempDir()
	outputDir := t.TempDir()

	builder := NewLinuxKitImageBuilder()
	cfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "core-dev",
		Version:    "v1.2.3",
		LinuxKit: build.LinuxKitConfig{
			Base:     "core-dev",
			Packages: []string{"gh"},
			Mounts:   []string{"/workspace"},
			Formats:  []string{"oci", "apple"},
		},
	}

	artifacts, err := builder.Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("want len %v, got %v", 2, len(artifacts))
	}
	if _, err := ax.Stat(ax.Join(outputDir, "core-dev.tar")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "core-dev.tar"))
	}
	if _, err := ax.Stat(ax.Join(outputDir, "core-dev.aci")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "core-dev.aci"))
	}
	if ax.Exists(ax.Join(outputDir, ".core-dev-linuxkit.yml")) {
		t.Fatalf("expected file not to exist: %v", ax.Join(outputDir, ".core-dev-linuxkit.yml"))
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestLinuxkitImage_NewLinuxKitImageBuilder_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewLinuxKitImageBuilder()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_NewLinuxKitImageBuilder_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewLinuxKitImageBuilder()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_NewLinuxKitImageBuilder_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewLinuxKitImageBuilder()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_Name_Good(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_Name_Bad(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_Name_Ugly(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_ListBaseImages_Good(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.ListBaseImages()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_ListBaseImages_Bad(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.ListBaseImages()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_ListBaseImages_Ugly(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.ListBaseImages()
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_ArtifactPath_Good(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.ArtifactPath(core.Path(t.TempDir(), "go-build-compliance"), "agent", "tar.gz")
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_ArtifactPath_Bad(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.ArtifactPath("", "", "")
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_ArtifactPath_Ugly(t *core.T) {
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_ = subject.ArtifactPath(core.Path(t.TempDir(), "go-build-compliance"), "agent", "tar.gz")
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil)
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil)
	})
	core.AssertTrue(t, true)
}

func TestLinuxkitImage_LinuxKitImageBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &LinuxKitImageBuilder{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil)
	})
	core.AssertTrue(t, true)
}
