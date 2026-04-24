package builders

import (
	"context"
	"os"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"

	"dappco.re/go/build/pkg/build"
	"dappco.re/go/io"
)

func setupFakeCPPCommand(t *testing.T, binDir, name, script string) {
	t.Helper()
	if err := ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func cppCrossTarget() build.Target {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return build.Target{OS: "darwin", Arch: "amd64"}
		}
		return build.Target{OS: "darwin", Arch: "arm64"}
	case "linux":
		if runtime.GOARCH == "arm64" {
			return build.Target{OS: "linux", Arch: "amd64"}
		}
		return build.Target{OS: "linux", Arch: "arm64"}
	default:
		return build.Target{OS: "linux", Arch: "amd64"}
	}
}

func TestCPP_CPPBuilderName_Good(t *testing.T) {
	builder := NewCPPBuilder()
	if !stdlibAssertEqual("cpp", builder.Name()) {
		t.Fatalf("want %v, got %v", "cpp", builder.Name())
	}

}

func TestCPP_CPPBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects C++ project with CMakeLists.txt", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for non-C++ project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestCPP_CPPBuilderBuild_Bad(t *testing.T) {
	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewCPPBuilder()
		artifacts, err := builder.Build(nil, nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		if err == nil {
			t.Fatal("expected error")
		}
		if !stdlibAssertNil(artifacts) {
			t.Fatalf("expected nil, got %v", artifacts)
		}
		if !stdlibAssertContains(err.Error(), "config is nil") {
			t.Fatalf("expected %v to contain %v", err.Error(), "config is nil")
		}

	})
}

func TestCPP_CPPBuilderBuild_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("C++ builder command fixtures use POSIX shell scripts")
	}

	t.Run("preserves the managed Makefile pipeline when present", func(t *testing.T) {
		projectDir := t.TempDir()
		binDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "make.log")
		if err := ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(projectDir, "Makefile"), []byte("all:\n\t@true\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		setupFakeCPPCommand(t, binDir, "make", `#!/bin/sh
set -eu
printf 'make %s\n' "$*" >> "${CPP_BUILD_LOG_FILE}"
case "${1:-}" in
  configure|build)
    exit 0
    ;;
  package)
    mkdir -p build/packages
    printf 'pkg\n' > build/packages/test-1.0.tar.gz
    exit 0
    ;;
esac
exit 1
`)
		setupFakeCPPCommand(t, binDir, "conan", `#!/bin/sh
set -eu
printf 'conan %s\n' "$*" >> "${CPP_BUILD_LOG_FILE}"
exit 0
`)

		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("CPP_BUILD_LOG_FILE", logPath)

		builder := NewCPPBuilder()
		artifacts, err := builder.Build(context.Background(), &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  ax.Join(projectDir, "dist"),
			Name:       "testapp",
		}, []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(ax.Join(projectDir, "build", "packages", "test-1.0.tar.gz"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "build", "packages", "test-1.0.tar.gz"), artifacts[0].Path)
		}

		content, err := io.Local.Read(logPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "make configure") {
			t.Fatalf("expected %v to contain %v", content, "make configure")
		}
		if !stdlibAssertContains(content, "make build") {
			t.Fatalf("expected %v to contain %v", content, "make build")
		}
		if !stdlibAssertContains(content, "make package") {
			t.Fatalf("expected %v to contain %v", content, "make package")
		}
		if stdlibAssertContains(content, "cmake ") {
			t.Fatalf("expected %v not to contain %v", content, "cmake ")
		}

	})

	t.Run("falls back to plain cmake for generic CMake projects", func(t *testing.T) {
		projectDir := t.TempDir()
		binDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "cmake.log")
		statePath := ax.Join(t.TempDir(), "cmake-state")
		if err := ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\nproject(demo)\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		setupFakeCPPCommand(t, binDir, "cmake", `#!/bin/sh
set -eu
printf 'cmake %s\n' "$*" >> "${CPP_BUILD_LOG_FILE}"
if [ "${1:-}" = "-S" ]; then
  for arg in "$@"; do
    case "$arg" in
      -DCMAKE_RUNTIME_OUTPUT_DIRECTORY=*)
        printf '%s\n' "${arg#*=}" > "${CPP_CMAKE_STATE_FILE}"
        ;;
    esac
  done
  exit 0
fi
if [ "${1:-}" = "--build" ]; then
  runtime_dir="$(cat "${CPP_CMAKE_STATE_FILE}")"
  mkdir -p "${runtime_dir}"
  printf 'binary\n' > "${runtime_dir}/${NAME:-testapp}"
  chmod +x "${runtime_dir}/${NAME:-testapp}"
  exit 0
fi
exit 1
`)

		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("CPP_BUILD_LOG_FILE", logPath)
		t.Setenv("CPP_CMAKE_STATE_FILE", statePath)

		target := build.Target{OS: runtime.GOOS, Arch: runtime.GOARCH}
		builder := NewCPPBuilder()
		artifacts, err := builder.Build(context.Background(), &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  ax.Join(projectDir, "dist"),
			Name:       "testapp",
		}, []build.Target{target})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path)
		}

		content, err := io.Local.Read(logPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "cmake -S") {
			t.Fatalf("expected %v to contain %v", content, "cmake -S")
		}
		if !stdlibAssertContains(content, "cmake --build") {
			t.Fatalf("expected %v to contain %v", content, "cmake --build")
		}
		if stdlibAssertContains(content, "conan ") {
			t.Fatalf("expected %v not to contain %v", content, "conan ")
		}
		if stdlibAssertContains(content, "make configure") {
			t.Fatalf("expected %v not to contain %v", content, "make configure")
		}
		if stdlibAssertContains(content, "make build") {
			t.Fatalf("expected %v not to contain %v", content, "make build")
		}
		if stdlibAssertContains(content, "make package") {
			t.Fatalf("expected %v not to contain %v", content, "make package")
		}

	})

	t.Run("uses conan plus cmake for generic cross-builds when a conanfile exists", func(t *testing.T) {
		projectDir := t.TempDir()
		binDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "conan-cmake.log")
		statePath := ax.Join(t.TempDir(), "conan-cmake-state")
		if err := ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\nproject(demo)\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(projectDir, "conanfile.txt"), []byte("[requires]\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		setupFakeCPPCommand(t, binDir, "conan", `#!/bin/sh
set -eu
printf 'conan %s\n' "$*" >> "${CPP_BUILD_LOG_FILE}"
output_dir=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-folder" ]; then
    output_dir="$2"
    shift 2
    continue
  fi
  shift
done
mkdir -p "${output_dir}"
printf '# toolchain\n' > "${output_dir}/conan_toolchain.cmake"
`)
		setupFakeCPPCommand(t, binDir, "cmake", `#!/bin/sh
set -eu
printf 'cmake %s\n' "$*" >> "${CPP_BUILD_LOG_FILE}"
if [ "${1:-}" = "-S" ]; then
  for arg in "$@"; do
    case "$arg" in
      -DCMAKE_RUNTIME_OUTPUT_DIRECTORY=*)
        printf '%s\n' "${arg#*=}" > "${CPP_CMAKE_STATE_FILE}"
        ;;
    esac
  done
  exit 0
fi
if [ "${1:-}" = "--build" ]; then
  runtime_dir="$(cat "${CPP_CMAKE_STATE_FILE}")"
  mkdir -p "${runtime_dir}"
  printf 'binary\n' > "${runtime_dir}/${NAME:-testapp}"
  chmod +x "${runtime_dir}/${NAME:-testapp}"
  exit 0
fi
exit 1
`)

		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("CPP_BUILD_LOG_FILE", logPath)
		t.Setenv("CPP_CMAKE_STATE_FILE", statePath)

		target := cppCrossTarget()
		builder := NewCPPBuilder()
		artifacts, err := builder.Build(context.Background(), &build.Config{
			FS:         io.Local,
			ProjectDir: projectDir,
			OutputDir:  ax.Join(projectDir, "dist"),
			Name:       "testapp",
		}, []build.Target{target})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path)
		}

		content, err := io.Local.Read(logPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertContains(content, "conan install . --output-folder "+ax.Join(projectDir, "build", "cmake", target.OS+"_"+target.Arch)+" --build=missing --profile:host "+builder.targetToProfile(target)) {
			t.Fatalf("expected %v to contain %v", content, "conan install . --output-folder "+ax.Join(projectDir, "build", "cmake", target.OS+"_"+target.Arch)+" --build=missing --profile:host "+builder.targetToProfile(target))
		}
		if !stdlibAssertContains(content, "cmake -S") {
			t.Fatalf("expected %v to contain %v", content, "cmake -S")
		}
		if !stdlibAssertContains(content, "-DCMAKE_TOOLCHAIN_FILE="+ax.Join(projectDir, "build", "cmake", target.OS+"_"+target.Arch, "conan_toolchain.cmake")) {
			t.Fatalf("expected %v to contain %v", content, "-DCMAKE_TOOLCHAIN_FILE="+ax.Join(projectDir, "build", "cmake", target.OS+"_"+target.Arch, "conan_toolchain.cmake"))
		}
		if !stdlibAssertContains(content, "cmake --build") {
			t.Fatalf("expected %v to contain %v", content, "cmake --build")
		}
		if stdlibAssertContains(content, "make configure") {
			t.Fatalf("expected %v not to contain %v", content, "make configure")
		}
		if stdlibAssertContains(content, "make build") {
			t.Fatalf("expected %v not to contain %v", content, "make build")
		}
		if stdlibAssertContains(content, "make package") {
			t.Fatalf("expected %v not to contain %v", content, "make package")
		}

	})
}

func TestCPP_CPPBuilderTargetToProfile_Good(t *testing.T) {
	builder := NewCPPBuilder()

	tests := []struct {
		os, arch string
		expected string
	}{
		{"linux", "amd64", "gcc-linux-x86_64"},
		{"linux", "x86_64", "gcc-linux-x86_64"},
		{"linux", "arm64", "gcc-linux-armv8"},
		{"darwin", "arm64", "apple-clang-armv8"},
		{"darwin", "amd64", "apple-clang-x86_64"},
		{"windows", "amd64", "msvc-194-x86_64"},
	}

	for _, tt := range tests {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			profile := builder.targetToProfile(build.Target{OS: tt.os, Arch: tt.arch})
			if !stdlibAssertEqual(tt.expected, profile) {
				t.Fatalf("want %v, got %v", tt.expected, profile)
			}

		})
	}
}

func TestCPP_CPPBuilderTargetToProfile_Bad(t *testing.T) {
	builder := NewCPPBuilder()

	t.Run("returns empty for unknown target", func(t *testing.T) {
		profile := builder.targetToProfile(build.Target{OS: "plan9", Arch: "mips"})
		if !stdlibAssertEmpty(profile) {
			t.Fatalf("expected empty, got %v", profile)
		}

	})
}

func TestCPP_CPPBuilderFindArtifacts_Good(t *testing.T) {
	fs := io.Local

	t.Run("finds packages in build/packages", func(t *testing.T) {
		dir := t.TempDir()
		packagesDir := ax.Join(dir, "build", "packages")
		if err := ax.MkdirAll(packagesDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v",

				// Create mock package files
				err)
		}
		if err := ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.tar.xz"), []byte("pkg"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.tar.xz.sha256"), []byte("checksum"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.rpm"), []byte("rpm"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewCPPBuilder()
		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts, err := builder.findArtifacts(fs, dir, target)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Should find tar.xz and rpm but not sha256
				err)
		}
		if len(artifacts) != 2 {
			t.Fatalf("want len %v, got %v", 2, len(artifacts))
		}

		for _, a := range artifacts {
			if !stdlibAssertEqual("linux", a.OS) {
				t.Fatalf("want %v, got %v", "linux", a.OS)
			}
			if !stdlibAssertEqual("amd64", a.Arch) {
				t.Fatalf("want %v, got %v", "amd64", a.Arch)
			}
			if ax.Ext(a.Path) == ".sha256" {
				t.Fatal("expected false")
			}

		}
	})

	t.Run("falls back to binaries in build/release/src", func(t *testing.T) {
		dir := t.TempDir()
		binDir := ax.Join(dir, "build", "release", "src")
		if err := ax.MkdirAll(binDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v",

				// Create mock binary (executable)
				err)
		}

		binPath := ax.Join(binDir, "test-daemon")
		if err := ax.WriteFile(binPath, []byte("binary"), 0755); err != nil {
			t.Fatalf("unexpected error: %v",

				// Create a library (should be skipped)
				err)
		}
		if err := ax.WriteFile(ax.Join(binDir, "libcrypto.a"), []byte("lib"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		builder := NewCPPBuilder()
		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts, err := builder.findArtifacts(fs, dir, target)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Should find the executable but not the library
				err)
		}
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertContains(artifacts[0].Path, "test-daemon") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "test-daemon")
		}

	})
}

func TestCPP_CPPBuilderResolveMakeCli_Good(t *testing.T) {
	builder := NewCPPBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "make")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := builder.resolveMakeCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCPP_CPPBuilderResolveMakeCli_Bad(t *testing.T) {
	builder := NewCPPBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveMakeCli(ax.Join(t.TempDir(), "missing-make"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "make not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "make not found")
	}

}

func TestCPP_CPPBuilderResolveConanCli_Good(t *testing.T) {
	builder := NewCPPBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "conan")
	if err := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("PATH", "")

	command, err := builder.resolveConanCli(fallbackPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCPP_CPPBuilderResolveConanCli_Bad(t *testing.T) {
	builder := NewCPPBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveConanCli(ax.Join(t.TempDir(), "missing-conan"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "conan not found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "conan not found")
	}

}

func TestCPP_CPPBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*CPPBuilder)(nil)
	var _ build.Builder = NewCPPBuilder()
}
