package builders

import (
	"context"
	"os"
	"runtime"
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeCPPCommand(t *testing.T, binDir, name, script string) {
	t.Helper()
	require.NoError(t, ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755))
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
	assert.Equal(t, "cpp", builder.Name())
}

func TestCPP_CPPBuilderDetect_Good(t *testing.T) {
	fs := io.Local

	t.Run("detects C++ project with CMakeLists.txt", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)"), 0644)
		require.NoError(t, err)

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.True(t, detected)
	})

	t.Run("returns false for non-C++ project", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644)
		require.NoError(t, err)

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewCPPBuilder()
		detected, err := builder.Detect(fs, dir)
		assert.NoError(t, err)
		assert.False(t, detected)
	})
}

func TestCPP_CPPBuilderBuild_Bad(t *testing.T) {
	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewCPPBuilder()
		artifacts, err := builder.Build(nil, nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		assert.Error(t, err)
		assert.Nil(t, artifacts)
		assert.Contains(t, err.Error(), "config is nil")
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

		require.NoError(t, ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\n"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(projectDir, "Makefile"), []byte("all:\n\t@true\n"), 0o644))

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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.Equal(t, ax.Join(projectDir, "build", "packages", "test-1.0.tar.gz"), artifacts[0].Path)

		content, err := io.Local.Read(logPath)
		require.NoError(t, err)
		assert.Contains(t, content, "make configure")
		assert.Contains(t, content, "make build")
		assert.Contains(t, content, "make package")
		assert.NotContains(t, content, "cmake ")
	})

	t.Run("falls back to plain cmake for generic CMake projects", func(t *testing.T) {
		projectDir := t.TempDir()
		binDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "cmake.log")
		statePath := ax.Join(t.TempDir(), "cmake-state")

		require.NoError(t, ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\nproject(demo)\n"), 0o644))

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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.Equal(t, ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path)

		content, err := io.Local.Read(logPath)
		require.NoError(t, err)
		assert.Contains(t, content, "cmake -S")
		assert.Contains(t, content, "cmake --build")
		assert.NotContains(t, content, "conan ")
		assert.NotContains(t, content, "make configure")
		assert.NotContains(t, content, "make build")
		assert.NotContains(t, content, "make package")
	})

	t.Run("uses conan plus cmake for generic cross-builds when a conanfile exists", func(t *testing.T) {
		projectDir := t.TempDir()
		binDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "conan-cmake.log")
		statePath := ax.Join(t.TempDir(), "conan-cmake-state")

		require.NoError(t, ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\nproject(demo)\n"), 0o644))
		require.NoError(t, ax.WriteFile(ax.Join(projectDir, "conanfile.txt"), []byte("[requires]\n"), 0o644))

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
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
		assert.Equal(t, ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path)

		content, err := io.Local.Read(logPath)
		require.NoError(t, err)
		assert.Contains(t, content, "conan install . --output-folder "+ax.Join(projectDir, "build", "cmake", target.OS+"_"+target.Arch)+" --build=missing --profile:host "+builder.targetToProfile(target))
		assert.Contains(t, content, "cmake -S")
		assert.Contains(t, content, "-DCMAKE_TOOLCHAIN_FILE="+ax.Join(projectDir, "build", "cmake", target.OS+"_"+target.Arch, "conan_toolchain.cmake"))
		assert.Contains(t, content, "cmake --build")
		assert.NotContains(t, content, "make configure")
		assert.NotContains(t, content, "make build")
		assert.NotContains(t, content, "make package")
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
			assert.Equal(t, tt.expected, profile)
		})
	}
}

func TestCPP_CPPBuilderTargetToProfile_Bad(t *testing.T) {
	builder := NewCPPBuilder()

	t.Run("returns empty for unknown target", func(t *testing.T) {
		profile := builder.targetToProfile(build.Target{OS: "plan9", Arch: "mips"})
		assert.Empty(t, profile)
	})
}

func TestCPP_CPPBuilderFindArtifacts_Good(t *testing.T) {
	fs := io.Local

	t.Run("finds packages in build/packages", func(t *testing.T) {
		dir := t.TempDir()
		packagesDir := ax.Join(dir, "build", "packages")
		require.NoError(t, ax.MkdirAll(packagesDir, 0755))

		// Create mock package files
		require.NoError(t, ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.tar.xz"), []byte("pkg"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.tar.xz.sha256"), []byte("checksum"), 0644))
		require.NoError(t, ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.rpm"), []byte("rpm"), 0644))

		builder := NewCPPBuilder()
		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts, err := builder.findArtifacts(fs, dir, target)
		require.NoError(t, err)

		// Should find tar.xz and rpm but not sha256
		assert.Len(t, artifacts, 2)
		for _, a := range artifacts {
			assert.Equal(t, "linux", a.OS)
			assert.Equal(t, "amd64", a.Arch)
			assert.False(t, ax.Ext(a.Path) == ".sha256")
		}
	})

	t.Run("falls back to binaries in build/release/src", func(t *testing.T) {
		dir := t.TempDir()
		binDir := ax.Join(dir, "build", "release", "src")
		require.NoError(t, ax.MkdirAll(binDir, 0755))

		// Create mock binary (executable)
		binPath := ax.Join(binDir, "test-daemon")
		require.NoError(t, ax.WriteFile(binPath, []byte("binary"), 0755))

		// Create a library (should be skipped)
		require.NoError(t, ax.WriteFile(ax.Join(binDir, "libcrypto.a"), []byte("lib"), 0644))

		builder := NewCPPBuilder()
		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts, err := builder.findArtifacts(fs, dir, target)
		require.NoError(t, err)

		// Should find the executable but not the library
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts[0].Path, "test-daemon")
	})
}

func TestCPP_CPPBuilderResolveMakeCli_Good(t *testing.T) {
	builder := NewCPPBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "make")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := builder.resolveMakeCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestCPP_CPPBuilderResolveMakeCli_Bad(t *testing.T) {
	builder := NewCPPBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveMakeCli(ax.Join(t.TempDir(), "missing-make"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "make not found")
}

func TestCPP_CPPBuilderResolveConanCli_Good(t *testing.T) {
	builder := NewCPPBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "conan")
	require.NoError(t, ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	t.Setenv("PATH", "")

	command, err := builder.resolveConanCli(fallbackPath)
	require.NoError(t, err)
	assert.Equal(t, fallbackPath, command)
}

func TestCPP_CPPBuilderResolveConanCli_Bad(t *testing.T) {
	builder := NewCPPBuilder()
	t.Setenv("PATH", "")

	_, err := builder.resolveConanCli(ax.Join(t.TempDir(), "missing-conan"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conan not found")
}

func TestCPP_CPPBuilderInterface_Good(t *testing.T) {
	var _ build.Builder = (*CPPBuilder)(nil)
	var _ build.Builder = NewCPPBuilder()
}
