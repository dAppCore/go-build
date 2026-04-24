package buildcmd

import (
	"context"
	"os"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/builders"
	"dappco.re/go/core"
	"dappco.re/go/core/io"
)

func setupFakeLinuxKitImageCLI(t *testing.T, binDir string) {
	t.Helper()

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
esac

mkdir -p "$dir"
printf 'linuxkit image\n' > "$dir/$name$ext"
`
	if err := ax.WriteFile(ax.Join(binDir, "linuxkit"), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func setupFakeDockerImageCLI(t *testing.T, binDir string) {
	t.Helper()

	script := `#!/bin/sh
set -eu

log_file="${DOCKER_LOG:-}"

log() {
	if [ -n "$log_file" ]; then
		printf '%s\n' "$1" >> "$log_file"
	fi
}

case "${1:-}" in
	build)
		shift
		log "docker build $*"
		;;
	image)
		shift
		case "${1:-}" in
			load)
				shift
				log "docker image load $*"
				echo "Loaded image: imported:latest"
				;;
			tag)
				shift
				log "docker image tag $*"
				;;
			push)
				shift
				log "docker image push $*"
				;;
			*)
				log "docker image $*"
				;;
		esac
		;;
	*)
		log "docker $*"
		;;
esac
`
	if err := ax.WriteFile(ax.Join(binDir, "docker"), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestBuildCmd_AddImageCommand_Good(t *testing.T) {
	c := core.New()

	AddImageCommand(c)
	if !(c.Command("build/image").OK) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_parseImageFormats_Good(t *testing.T) {
	if !stdlibAssertEqual([]string{"oci", "apple"}, parseImageFormats(" OCI , apple,Apple, oci ")) {
		t.Fatalf("want %v, got %v", []string{"oci", "apple"}, parseImageFormats(" OCI , apple,Apple, oci "))
	}

}

func TestBuildCmd_buildPwaCommandAcceptsPath_Good(t *testing.T) {
	c := core.New()
	AddBuildCommands(c)

	command := c.Command("build/pwa").Value.(*core.Command)

	original := runLocalPwaBuild
	defer func() { runLocalPwaBuild = original }()

	calledPath := ""
	runLocalPwaBuild = func(ctx context.Context, projectDir string) error {
		calledPath = projectDir
		return nil
	}

	opts := core.NewOptions(core.Option{Key: "path", Value: "/tmp/pwa"})
	result := command.Run(opts)
	if !(result.OK) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("/tmp/pwa", calledPath) {
		t.Fatalf("want %v, got %v", "/tmp/pwa", calledPath)
	}

}

func TestBuildCmd_runBuildImage_Good(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()
	setupFakeLinuxKitImageCLI(t, binDir)
	setupFakeDockerImageCLI(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	outputDir := t.TempDir()

	err := runBuildImage(ImageBuildRequest{
		Context:   context.Background(),
		Base:      "core-minimal",
		Format:    "oci,apple",
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(ax.Join(outputDir, "core-minimal.tar")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "core-minimal.tar"))
	}
	if _, err := os.Stat(ax.Join(outputDir, "core-minimal.aci")); err != nil {
		t.Fatalf("expected file to exist: %v", ax.Join(outputDir, "core-minimal.aci"))
	}

	t.Setenv("PATH", "/definitely-missing")
	err = runBuildImage(ImageBuildRequest{
		Context:   context.Background(),
		Base:      "core-minimal",
		Format:    "oci,apple",
		OutputDir: outputDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestBuildCmd_resolveImmutableImageVersion_Good(t *testing.T) {
	t.Run("uses exact release tag on HEAD", func(t *testing.T) {
		dir := t.TempDir()

		runGit(t, dir, "init")
		runGit(t, dir, "config", "user.email", "test@example.com")
		runGit(t, dir, "config", "user.name", "Test User")
		if err := ax.WriteFile(ax.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "feat: initial commit")
		runGit(t, dir, "tag", "v1.4.2")

		version := resolveImmutableImageVersion(context.Background(), dir)
		if !stdlibAssertEqual(immutableImageVersion{BuildVersion: "v1.4.2", RetainVersion: "v1.4.2", CacheVersion: "v1.4.2"}, version) {
			t.Fatalf("want %v, got %v", immutableImageVersion{BuildVersion: "v1.4.2", RetainVersion: "v1.4.2", CacheVersion: "v1.4.2"}, version)
		}

	})

	t.Run("falls back to dev for untagged commits", func(t *testing.T) {
		dir := t.TempDir()

		runGit(t, dir, "init")
		runGit(t, dir, "config", "user.email", "test@example.com")
		runGit(t, dir, "config", "user.name", "Test User")
		if err := ax.WriteFile(ax.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "feat: initial commit")

		version := resolveImmutableImageVersion(context.Background(), dir)
		if !stdlibAssertEqual(immutableImageVersion{BuildVersion: "dev"}, version) {
			t.Fatalf("want %v, got %v", immutableImageVersion{BuildVersion: "dev"}, version)
		}

	})

	t.Run("falls back to dev after the release tag moves behind HEAD", func(t *testing.T) {
		dir := t.TempDir()

		runGit(t, dir, "init")
		runGit(t, dir, "config", "user.email", "test@example.com")
		runGit(t, dir, "config", "user.name", "Test User")
		if err := ax.WriteFile(ax.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "feat: initial commit")
		runGit(t, dir, "tag", "v1.4.2")
		if err := ax.WriteFile(ax.Join(dir, "CHANGELOG.md"), []byte("more\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "feat: follow-up work")

		version := resolveImmutableImageVersion(context.Background(), dir)
		if !stdlibAssertEqual(immutableImageVersion{BuildVersion: "dev"}, version) {
			t.Fatalf("want %v, got %v", immutableImageVersion{BuildVersion: "dev"}, version)
		}

	})
}

func TestBuildCmd_allImageArtifactsExist_RequiresMatchingCacheMetadata_Good(t *testing.T) {
	outputDir := t.TempDir()
	imageName := "core-dev"
	builder := builders.NewLinuxKitImageBuilder()
	cfg := build.LinuxKitConfig{
		Base:     "core-dev",
		Formats:  []string{"oci", "apple"},
		Packages: []string{"git", "task"},
		Mounts:   []string{"/workspace"},
	}
	if err := ax.WriteFile(ax.Join(outputDir, "core-dev.tar"), []byte("oci image"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(outputDir, "core-dev.aci"), []byte("apple image"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := writeImageBuildCacheMetadata(io.Local, outputDir, imageName, cfg, "v1.2.3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(allImageArtifactsExist(io.Local, builder, outputDir, imageName, cfg, "v1.2.3")) {
		t.Fatal("expected true")
	}
	if allImageArtifactsExist(io.Local, builder, outputDir, imageName, cfg, "v1.2.4") {
		t.Fatal("expected false")
	}

	changedCfg := cfg
	changedCfg.GPU = true
	if allImageArtifactsExist(io.Local, builder, outputDir, imageName, changedCfg, "v1.2.3") {
		t.Fatal("expected false")
	}
	if err := io.Local.Delete(imageBuildCacheMetadataPath(outputDir, imageName)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allImageArtifactsExist(io.Local, builder, outputDir, imageName, cfg, "v1.2.3") {
		t.Fatal("expected false")
	}

}

func TestBuildCmd_allImageArtifactsExist_ValidatesVersionlessCacheMetadata_Good(t *testing.T) {
	outputDir := t.TempDir()
	imageName := "core-dev"
	builder := builders.NewLinuxKitImageBuilder()
	cfg := build.LinuxKitConfig{
		Base:     "core-dev",
		Formats:  []string{"oci", "apple"},
		Packages: []string{"git", "task"},
		Mounts:   []string{"/workspace"},
	}
	if err := ax.WriteFile(ax.Join(outputDir, "core-dev.tar"), []byte("oci image"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(outputDir, "core-dev.aci"), []byte("apple image"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := writeImageBuildCacheMetadata(io.Local, outputDir, imageName, cfg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(allImageArtifactsExist(io.Local, builder, outputDir, imageName, cfg, "")) {
		t.Fatal("expected true")
	}

	changedCfg := cfg
	changedCfg.GPU = true
	if allImageArtifactsExist(io.Local, builder, outputDir, imageName, changedCfg, "") {
		t.Fatal("expected false")
	}

}

func TestBuildCmd_retainVersionedImageArtifacts_Good(t *testing.T) {
	outputDir := t.TempDir()
	tarPath := ax.Join(outputDir, "core-dev.tar")
	aciPath := ax.Join(outputDir, "core-dev.aci")
	if err := ax.WriteFile(tarPath, []byte("oci image"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(aciPath, []byte("apple image"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	versionedPaths, err := retainVersionedImageArtifacts(io.Local, []build.Artifact{
		{Path: tarPath},
		{Path: aciPath},
	}, "v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		ax.Join(outputDir, "core-dev-1.2.3.tar"),
		ax.Join(outputDir, "core-dev-1.2.3.aci"),
	}
	if !stdlibAssertElementsMatch(expected, versionedPaths) {
		t.Fatalf("expected elements %v, got %v", expected, versionedPaths)
	}

	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file to exist: %v", path)
		}

	}
}

func TestBuildCmd_publishOCIImageArchive_Good(t *testing.T) {
	binDir := t.TempDir()
	logPath := ax.Join(t.TempDir(), "docker.log")
	setupFakeDockerImageCLI(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DOCKER_LOG", logPath)

	projectDir := t.TempDir()
	artifactPath := ax.Join(projectDir, "core-dev.tar")
	if err := ax.WriteFile(artifactPath, []byte("oci image"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref, err := publishOCIImageArchive(context.Background(), projectDir, artifactPath, "ghcr.io/dappcore", "core-dev", "v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("ghcr.io/dappcore/core-dev:1.2.3", ref) {
		t.Fatalf("want %v, got %v", "ghcr.io/dappcore/core-dev:1.2.3", ref)
	}

	logContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(logContent), "docker image load --input "+artifactPath) {
		t.Fatalf("expected %v to contain %v", string(logContent), "docker image load --input "+artifactPath)
	}
	if !stdlibAssertContains(string(logContent), "docker image tag imported:latest ghcr.io/dappcore/core-dev:1.2.3") {
		t.Fatalf("expected %v to contain %v", string(logContent), "docker image tag imported:latest ghcr.io/dappcore/core-dev:1.2.3")
	}
	if !stdlibAssertContains(string(logContent), "docker image push ghcr.io/dappcore/core-dev:1.2.3") {
		t.Fatalf("expected %v to contain %v", string(logContent), "docker image push ghcr.io/dappcore/core-dev:1.2.3")
	}

}
