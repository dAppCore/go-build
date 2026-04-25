package builders

import (
	"context"
	"os"
	"testing"

	"dappco.re/go/build/internal/ax"

	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func setupFakeLinuxKitToolchain(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

if [ "${1:-}" != "build" ]; then
	exit 1
fi

config=""
dir=""
name=""
while [ $# -gt 0 ]; do
	if [ "$1" = "--dir" ]; then
		shift
		dir="${1:-}"
	elif [ "$1" = "--name" ]; then
		shift
		name="${1:-}"
	fi
	shift
done

if [ -n "$dir" ] && [ -n "$name" ]; then
	mkdir -p "$dir"
	printf 'linuxkit image\n' > "$dir/$name.iso"
fi
`
	if err := ax.WriteFile(ax.Join(binDir, "linuxkit"), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestLinuxKit_LinuxKitBuilderName_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()
	if !stdlibAssertEqual("linuxkit", builder.Name()) {
		t.Fatalf("want %v, got %v", "linuxkit", builder.Name())
	}

}

func TestLinuxKit_LinuxKitBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects linuxkit.yml in root", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "linuxkit.yml"), []byte("kernel:\n  image: test\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects linuxkit.yaml in root", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "linuxkit.yaml"), []byte("kernel:\n  image: test\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects .core/linuxkit/*.yml", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		if err := ax.MkdirAll(lkDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := ax.WriteFile(ax.Join(lkDir, "server.yml"), []byte("kernel:\n  image: test\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects .core/linuxkit/*.yaml", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		if err := ax.MkdirAll(lkDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := ax.WriteFile(ax.Join(lkDir, "server.yaml"), []byte("kernel:\n  image: test\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects .core/linuxkit with multiple yml files", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		if err := ax.MkdirAll(lkDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := ax.WriteFile(ax.Join(lkDir, "server.yml"), []byte("kernel:\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(lkDir, "desktop.yml"), []byte("kernel:\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for non-LinuxKit project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for empty .core/linuxkit directory", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		if err := ax.MkdirAll(lkDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false when .core/linuxkit has only non-yml files", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		if err := ax.MkdirAll(lkDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := ax.WriteFile(ax.Join(lkDir, "README.md"), []byte("# LinuxKit\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false when .core/linuxkit has only non-yaml files", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		if err := ax.MkdirAll(lkDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err := ax.WriteFile(ax.Join(lkDir, "README.md"), []byte("# LinuxKit\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("ignores subdirectories in .core/linuxkit", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		subDir := ax.Join(lkDir, "subdir")
		if err := ax.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v",

				// Put yml in subdir only, not in lkDir itself
				err)
		}

		err := ax.WriteFile(ax.Join(subDir, "server.yml"), []byte("kernel:\n"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewLinuxKitBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestLinuxKit_LinuxKitBuilderGetFormatExtension_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()

	tests := []struct {
		format   string
		expected string
	}{
		{"iso", ".iso"},
		{"iso-bios", ".iso"},
		{"iso-efi", ".iso"},
		{"raw", ".raw"},
		{"raw-bios", ".raw"},
		{"raw-efi", ".raw"},
		{"qcow2", ".qcow2"},
		{"qcow2-bios", ".qcow2"},
		{"qcow2-efi", ".qcow2"},
		{"vmdk", ".vmdk"},
		{"vhd", ".vhd"},
		{"gcp", ".img.tar.gz"},
		{"aws", ".raw"},
		{"docker", ".docker.tar"},
		{"tar", ".tar"},
		{"kernel+initrd", "-initrd.img"},
		{"custom", ".custom"},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			ext := builder.getFormatExtension(tc.format)
			if !stdlibAssertEqual(tc.expected, ext) {
				t.Fatalf("want %v, got %v", tc.expected, ext)
			}

		})
	}
}

func TestLinuxKit_LinuxKitBuilderGetArtifactPath_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()

	t.Run("constructs correct path", func(t *testing.T) {
		path := builder.getArtifactPath("/dist", "server-amd64", "iso")
		if !stdlibAssertEqual("/dist/server-amd64.iso", path) {
			t.Fatalf("want %v, got %v", "/dist/server-amd64.iso", path)
		}

	})

	t.Run("constructs correct path for qcow2", func(t *testing.T) {
		path := builder.getArtifactPath("/output/linuxkit", "server-arm64", "qcow2-bios")
		if !stdlibAssertEqual("/output/linuxkit/server-arm64.qcow2", path) {
			t.Fatalf("want %v, got %v", "/output/linuxkit/server-arm64.qcow2", path)
		}

	})

	t.Run("constructs correct path for docker images", func(t *testing.T) {
		path := builder.getArtifactPath("/output/linuxkit", "server-amd64", "docker")
		if !stdlibAssertEqual("/output/linuxkit/server-amd64.docker.tar", path) {
			t.Fatalf("want %v, got %v", "/output/linuxkit/server-amd64.docker.tar", path)
		}

	})

	t.Run("constructs correct path for kernel+initrd images", func(t *testing.T) {
		path := builder.getArtifactPath("/output/linuxkit", "server-amd64", "kernel+initrd")
		if !stdlibAssertEqual("/output/linuxkit/server-amd64-initrd.img", path) {
			t.Fatalf("want %v, got %v", "/output/linuxkit/server-amd64-initrd.img", path)
		}

	})
}

func TestLinuxKit_LinuxKitBuilderBuildLinuxKitArgs_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()

	t.Run("builds args for amd64 without --arch", func(t *testing.T) {
		args := builder.buildLinuxKitArgs("/config.yml", "iso", "output", "/dist", "amd64")
		if !stdlibAssertContains(args, "build") {
			t.Fatalf("expected %v to contain %v", args, "build")
		}
		if !stdlibAssertContains(args, "--format") {
			t.Fatalf("expected %v to contain %v", args, "--format")
		}
		if !stdlibAssertContains(args, "iso") {
			t.Fatalf("expected %v to contain %v", args, "iso")
		}
		if !stdlibAssertContains(args, "--name") {
			t.Fatalf("expected %v to contain %v", args, "--name")
		}
		if !stdlibAssertContains(args, "output") {
			t.Fatalf("expected %v to contain %v", args, "output")
		}
		if !stdlibAssertContains(args, "--dir") {
			t.Fatalf("expected %v to contain %v", args, "--dir")
		}
		if !stdlibAssertContains(args, "/dist") {
			t.Fatalf("expected %v to contain %v", args, "/dist")
		}
		if !stdlibAssertContains(args, "/config.yml") {
			t.Fatalf("expected %v to contain %v", args, "/config.yml")
		}
		if stdlibAssertContains(args, "--arch") {
			t.Fatalf("expected %v not to contain %v", args, "--arch")
		}

	})

	t.Run("builds args for arm64 with --arch", func(t *testing.T) {
		args := builder.buildLinuxKitArgs("/config.yml", "qcow2", "output", "/dist", "arm64")
		if !stdlibAssertContains(args, "--arch") {
			t.Fatalf("expected %v to contain %v", args, "--arch")
		}
		if !stdlibAssertContains(args, "arm64") {
			t.Fatalf("expected %v to contain %v", args, "arm64")
		}

	})
}

func TestLinuxKit_LinuxKitBuilderBuild_ResolvesRelativeConfigPath_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeLinuxKitToolchain(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	projectDir := t.TempDir()
	configPath := ax.Join(projectDir, "deploy", "linuxkit.yml")
	if err := ax.MkdirAll(ax.Dir(configPath), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(configPath, []byte("kernel:\n  image: test\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputDir := t.TempDir()
	builder := NewLinuxKitBuilder()
	cfg := &build.Config{
		FS:             io.Local,
		ProjectDir:     projectDir,
		OutputDir:      outputDir,
		Name:           "sample",
		LinuxKitConfig: "deploy/linuxkit.yml",
		Formats:        []string{"iso"},
	}

	artifacts, err := builder.Build(context.Background(), cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("want len %v, got %v", 1, len(artifacts))
	}

	expectedPath := ax.Join(outputDir, "sample-amd64.iso")
	if !stdlibAssertEqual(expectedPath, artifacts[0].Path) {
		t.Fatalf("want %v, got %v", expectedPath, artifacts[0].Path)
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected file to exist: %v", expectedPath)
	}

}

func TestLinuxKit_LinuxKitBuilderFindArtifact_Good(t *testing.T) {
	fs := io.Local
	builder := NewLinuxKitBuilder()

	t.Run("finds artifact with exact extension", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "server-amd64.iso")
		if err := ax.WriteFile(artifactPath, []byte("fake iso"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := builder.findArtifact(fs, dir, "server-amd64", "iso")
		if !stdlibAssertEqual(artifactPath, found) {
			t.Fatalf("want %v, got %v", artifactPath, found)
		}

	})

	t.Run("returns empty for missing artifact", func(t *testing.T) {
		dir := t.TempDir()

		found := builder.findArtifact(fs, dir, "nonexistent", "iso")
		if !stdlibAssertEmpty(found) {
			t.Fatalf("expected empty, got %v", found)
		}

	})

	t.Run("finds artifact with alternate naming", func(t *testing.T) {
		dir := t.TempDir()
		// Create file matching the name prefix + known image extension
		artifactPath := ax.Join(dir, "server-amd64.qcow2")
		if err := ax.WriteFile(artifactPath, []byte("fake qcow2"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := builder.findArtifact(fs, dir, "server-amd64", "qcow2")
		if !stdlibAssertEqual(artifactPath, found) {
			t.Fatalf("want %v, got %v", artifactPath, found)
		}

	})

	t.Run("finds cloud image artifacts", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "server-amd64-gcp.img.tar.gz")
		if err := ax.WriteFile(artifactPath, []byte("fake gcp image"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := builder.findArtifact(fs, dir, "server-amd64", "gcp")
		if !stdlibAssertEqual(artifactPath, found) {
			t.Fatalf("want %v, got %v", artifactPath, found)
		}

	})

	t.Run("finds docker artifacts", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "server-amd64.docker.tar")
		if err := ax.WriteFile(artifactPath, []byte("fake docker tar"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := builder.findArtifact(fs, dir, "server-amd64", "docker")
		if !stdlibAssertEqual(artifactPath, found) {
			t.Fatalf("want %v, got %v", artifactPath, found)
		}

	})

	t.Run("finds kernel+initrd artifacts", func(t *testing.T) {
		dir := t.TempDir()
		artifactPath := ax.Join(dir, "server-amd64-initrd.img")
		if err := ax.WriteFile(artifactPath, []byte("fake initrd"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := builder.findArtifact(fs, dir, "server-amd64", "kernel+initrd")
		if !stdlibAssertEqual(artifactPath, found) {
			t.Fatalf("want %v, got %v", artifactPath, found)
		}

	})
}

func TestLinuxKit_LinuxKitBuilderInterface_Good(t *testing.T) {
	// Verify LinuxKitBuilder implements Builder interface
	var _ build.Builder = (*LinuxKitBuilder)(nil)
	var _ build.Builder = NewLinuxKitBuilder()
}

func TestLinuxKit_LinuxKitBuilderResolveLinuxKitCli_Good(t *testing.T) {
	builder := NewLinuxKitBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "linuxkit")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := builder.resolveLinuxKitCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestLinuxKit_LinuxKitBuilderResolveLinuxKitCli_Bad(t *testing.T) {
	builder := NewLinuxKitBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveLinuxKitCli(ax.Join(t.TempDir(), "missing-linuxkit"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "linuxkit CLI not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "linuxkit CLI not found")
	}

}
