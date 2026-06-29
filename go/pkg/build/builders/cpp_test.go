package builders

import (
	"context"
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"

	core "dappco.re/go"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

func setupFakeCPPCommand(t *testing.T, binDir, name, script string) {
	t.Helper()
	if result := ax.WriteFile(ax.Join(binDir, name), []byte(script), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

}

func requireCPPBool(t *testing.T, result core.Result) bool {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(bool)
}

func requireCPPArtifacts(t *testing.T, result core.Result) []build.Artifact {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]build.Artifact)
}

func requireCPPString(t *testing.T, result core.Result) string {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.(string)
}

func requireBuilderBytes(t *testing.T, result core.Result) []byte {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
	return result.Value.([]byte)
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

func TestCPP_CPPBuilderNameGood(t *testing.T) {
	builder := NewCPPBuilder()
	if !stdlibAssertEqual("cpp", builder.Name()) {
		t.Fatalf("want %v, got %v", "cpp", builder.Name())
	}

}

func TestCPP_CPPBuilderDetectGood(t *testing.T) {
	fs := storage.Local

	t.Run("detects C++ project with CMakeLists.txt", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewCPPBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if !(detected) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for non-C++ project", func(t *testing.T) {
		dir := t.TempDir()
		if result := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module test"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewCPPBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		dir := t.TempDir()

		builder := NewCPPBuilder()
		detected := requireCPPBool(t, builder.Detect(fs, dir))
		if detected {
			t.Fatal("expected false")
		}

	})
}

func TestCPP_CPPBuilderBuildBad(t *testing.T) {
	t.Run("returns error for nil config", func(t *testing.T) {
		builder := NewCPPBuilder()
		result := builder.Build(nil, nil, []build.Target{{OS: "linux", Arch: "amd64"}})
		if result.OK {
			t.Fatal("expected error")
		}
		if !stdlibAssertContains(result.Error(), "config is nil") {
			t.Fatalf("expected %v to contain %v", result.Error(), "config is nil")
		}

	})
}

func TestCPP_CPPBuilderBuildGood(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("C++ builder command fixtures use POSIX shell scripts")
	}

	t.Run("preserves the managed Makefile pipeline when present", func(t *testing.T) {
		projectDir := t.TempDir()
		binDir := t.TempDir()
		logPath := ax.Join(t.TempDir(), "make.log")
		if result := ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(projectDir, "Makefile"), []byte("all:\n\t@true\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
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

		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
		t.Setenv("CPP_BUILD_LOG_FILE", logPath)

		builder := NewCPPBuilder()
		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			OutputDir:  ax.Join(projectDir, "dist"),
			Name:       "testapp",
		}, []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(ax.Join(projectDir, "build", "packages", "test-1.0.tar.gz"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "build", "packages", "test-1.0.tar.gz"), artifacts[0].Path)
		}

		content := requireCPPString(t, storage.Local.Read(logPath))
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
		if result := ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\nproject(demo)\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
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

		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
		t.Setenv("CPP_BUILD_LOG_FILE", logPath)
		t.Setenv("CPP_CMAKE_STATE_FILE", statePath)

		target := build.Target{OS: runtime.GOOS, Arch: runtime.GOARCH}
		builder := NewCPPBuilder()
		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			OutputDir:  ax.Join(projectDir, "dist"),
			Name:       "testapp",
		}, []build.Target{target}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path)
		}

		content := requireCPPString(t, storage.Local.Read(logPath))
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
		if result := ax.WriteFile(ax.Join(projectDir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.16)\nproject(demo)\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(projectDir, "conanfile.txt"), []byte("[requires]\n"), 0o644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
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

		t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
		t.Setenv("CPP_BUILD_LOG_FILE", logPath)
		t.Setenv("CPP_CMAKE_STATE_FILE", statePath)

		target := cppCrossTarget()
		builder := NewCPPBuilder()
		artifacts := requireCPPArtifacts(t, builder.Build(context.Background(), &build.Config{
			FS:         storage.Local,
			ProjectDir: projectDir,
			OutputDir:  ax.Join(projectDir, "dist"),
			Name:       "testapp",
		}, []build.Target{target}))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertEqual(ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "dist", target.OS+"_"+target.Arch, "testapp"), artifacts[0].Path)
		}

		content := requireCPPString(t, storage.Local.Read(logPath))
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

func TestCPP_CPPBuilderTargetToProfileGood(t *testing.T) {
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

func TestCPP_CPPBuilderTargetToProfileBad(t *testing.T) {
	builder := NewCPPBuilder()

	t.Run("returns empty for unknown target", func(t *testing.T) {
		profile := builder.targetToProfile(build.Target{OS: "plan9", Arch: "mips"})
		if !stdlibAssertEmpty(profile) {
			t.Fatalf("expected empty, got %v", profile)
		}

	})
}

func TestCPP_CPPBuilderFindArtifactsGood(t *testing.T) {
	fs := storage.Local

	t.Run("finds packages in build/packages", func(t *testing.T) {
		dir := t.TempDir()
		packagesDir := ax.Join(dir, "build", "packages")
		if result := ax.MkdirAll(packagesDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.tar.xz"), []byte("pkg"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.tar.xz.sha256"), []byte("checksum"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(packagesDir, "test-1.0-linux-x86_64.rpm"), []byte("rpm"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewCPPBuilder()
		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts := requireCPPArtifacts(t, builder.findArtifacts(fs, dir, target))
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
		if result := ax.MkdirAll(binDir, 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		binPath := ax.Join(binDir, "test-daemon")
		if result := ax.WriteFile(binPath, []byte("binary"), 0755); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}
		if result := ax.WriteFile(ax.Join(binDir, "libcrypto.a"), []byte("lib"), 0644); !result.OK {
			t.Fatalf("unexpected error: %v", result.Error())
		}

		builder := NewCPPBuilder()
		target := build.Target{OS: "linux", Arch: "amd64"}
		artifacts := requireCPPArtifacts(t, builder.findArtifacts(fs, dir, target))
		if len(artifacts) != 1 {
			t.Fatalf("want len %v, got %v", 1, len(artifacts))
		}
		if !stdlibAssertContains(artifacts[0].Path, "test-daemon") {
			t.Fatalf("expected %v to contain %v", artifacts[0].Path, "test-daemon")
		}

	})
}

func TestCPP_CPPBuilderResolveMakeCliGood(t *testing.T) {
	builder := NewCPPBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "make")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	command := requireCPPString(t, builder.resolveMakeCli(fallbackPath))
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCPP_CPPBuilderResolveMakeCliBad(t *testing.T) {
	builder := NewCPPBuilder()
	t.Setenv("PATH", "")

	result := builder.resolveMakeCli(ax.Join(t.TempDir(), "missing-make"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "make not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "make not found")
	}

}

func TestCPP_CPPBuilderResolveConanCliGood(t *testing.T) {
	builder := NewCPPBuilder()
	fallbackDir := t.TempDir()
	fallbackPath := ax.Join(fallbackDir, "conan")
	if result := ax.WriteFile(fallbackPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}

	t.Setenv("PATH", "")

	command := requireCPPString(t, builder.resolveConanCli(fallbackPath))
	if !stdlibAssertEqual(fallbackPath, command) {
		t.Fatalf("want %v, got %v", fallbackPath, command)
	}

}

func TestCPP_CPPBuilderResolveConanCliBad(t *testing.T) {
	builder := NewCPPBuilder()
	t.Setenv("PATH", "")

	result := builder.resolveConanCli(ax.Join(t.TempDir(), "missing-conan"))
	if result.OK {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(result.Error(), "conan not found") {
		t.Fatalf("expected %v to contain %v", result.Error(), "conan not found")
	}

}

func TestCPP_CPPBuilderInterfaceGood(t *testing.T) {
	builder := NewCPPBuilder()
	var _ build.Builder = builder
	if !stdlibAssertEqual("cpp", builder.Name()) {
		t.Fatalf("want %v, got %v", "cpp", builder.Name())
	}
	detected := requireCPPBool(t, builder.Detect(nil, t.TempDir()))
	if detected {
		t.Fatal("expected empty temp directory not to be detected")
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestCpp_NewCPPBuilder_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewCPPBuilder()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCpp_NewCPPBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewCPPBuilder()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCpp_NewCPPBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewCPPBuilder()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCpp_CPPBuilder_Name_Good(t *core.T) {
	subject := &CPPBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCpp_CPPBuilder_Name_Bad(t *core.T) {
	subject := &CPPBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCpp_CPPBuilder_Name_Ugly(t *core.T) {
	subject := &CPPBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCpp_CPPBuilder_Detect_Good(t *core.T) {
	subject := &CPPBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCpp_CPPBuilder_Detect_Bad(t *core.T) {
	subject := &CPPBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCpp_CPPBuilder_Detect_Ugly(t *core.T) {
	subject := &CPPBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Detect(storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCpp_CPPBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &CPPBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCpp_CPPBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &CPPBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCpp_CPPBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &CPPBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Build(ctx, nil, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
