package builders

import (
	"context"
	"os"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
	"errors"
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

func TestLinuxKitImage_LinuxKitImageBuilderName_Good(t *testing.T) {
	builder := NewLinuxKitImageBuilder()
	if !stdlibAssertEqual("linuxkit-image", builder.Name()) {
		t.Fatalf("want %v, got %v", "linuxkit-image", builder.Name())
	}

}

func TestLinuxKitImage_LinuxKitImageBuilderArtifactPath_Good(t *testing.T) {
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

func TestLinuxKitImage_BuildLinuxKitServiceImageReference_UsesVersionTag_Good(t *testing.T) {
	if !stdlibAssertEqual("core-build-linuxkit/core-dev:1.2.3", buildLinuxKitServiceImageReference("core-dev", "v1.2.3")) {
		t.Fatalf("want %v, got %v", "core-build-linuxkit/core-dev:1.2.3", buildLinuxKitServiceImageReference("core-dev", "v1.2.3"))
	}
	if !stdlibAssertEqual("core-build-linuxkit/core-dev:dev", buildLinuxKitServiceImageReference("core-dev", "")) {
		t.Fatalf("want %v, got %v", "core-build-linuxkit/core-dev:dev", buildLinuxKitServiceImageReference("core-dev", ""))
	}

}

func TestLinuxKitImage_RenderLinuxKitServiceDockerfile_IncludesMetadata_Good(t *testing.T) {
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

func TestLinuxKitImage_RenderTemplateUsesImmutableServiceImage_Good(t *testing.T) {
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

func TestLinuxKitImage_RenderTemplateRestoresDefaultWorkspaceMount_Good(t *testing.T) {
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

func TestLinuxKitImage_LinuxKitImageBuilderBuild_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeLinuxKitImageToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

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
	if _, err := os.Stat(ax.Join(outputDir, "core-dev.tar")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "core-dev.tar"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "core-dev.aci")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "core-dev.aci"))
	}
	if _, err := os.Stat(ax.Join(outputDir, ".core-dev-linuxkit.yml")); err == nil {
		t.Fatalf("expected file not to exist: %v", ax.Join(outputDir, ".core-dev-linuxkit.yml"))
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

}
