// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	stdfs "io/fs"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

// CPPBuilder implements the Builder interface for C++ projects using CMake + Conan.
// It wraps the Makefile-based build system from the .core/build submodule.
//
// b := builders.NewCPPBuilder()
type CPPBuilder struct{}

// NewCPPBuilder creates a new CPPBuilder instance.
//
// b := builders.NewCPPBuilder()
func NewCPPBuilder() *CPPBuilder {
	return &CPPBuilder{}
}

// Name returns the builder's identifier.
//
// name := b.Name() // → "cpp"
func (b *CPPBuilder) Name() string {
	return "cpp"
}

// Detect checks if this builder can handle the project (checks for CMakeLists.txt).
//
// ok, err := b.Detect(storage.Local, ".")
func (b *CPPBuilder) Detect(fs storage.Medium, dir string) core.Result {
	return core.Ok(build.IsCPPProject(fs, dir))
}

// Build compiles the C++ project using Make targets.
// The build flow is: make configure → make build → make package.
//
// artifacts, err := b.Build(ctx, cfg, []build.Target{{OS: "linux", Arch: "amd64"}})
func (b *CPPBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(core.E("CPPBuilder.Build", "config is nil", nil))
	}

	filesystem := cfg.FS
	if filesystem == nil {
		filesystem = storage.Local
		cfg.FS = filesystem
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = ax.Join(cfg.ProjectDir, "dist")
	}

	managedMake := b.hasManagedMakefile(filesystem, cfg.ProjectDir)
	if managedMake {
		// Managed C++ repos keep the Conan/CMake orchestration in the project Makefile.
		if valid := b.validateMake(); !valid.OK {
			return valid
		}
		if valid := b.validateConan(); !valid.OK {
			return valid
		}
	} else {
		if valid := b.validateCMake(); !valid.OK {
			return valid
		}
		if b.usesConan(filesystem, cfg.ProjectDir) {
			if valid := b.validateConan(); !valid.OK {
				return valid
			}
		}
	}

	// For C++ projects, the Makefile handles everything.
	// We don't iterate per-target like Go — the Makefile's configure + build
	// produces binaries for the host platform, and cross-compilation uses
	// named Conan profiles (e.g., make gcc-linux-armv8).
	if len(targets) == 0 {
		// Default to host platform
		targets = []build.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}

	var artifacts []build.Artifact

	for _, target := range targets {
		builtResult := b.buildTarget(ctx, cfg, target)
		if !builtResult.OK {
			return core.Fail(core.E("CPPBuilder.Build", "build failed", core.NewError(builtResult.Error())))
		}
		artifacts = append(artifacts, builtResult.Value.([]build.Artifact)...)
	}

	return core.Ok(artifacts)
}

// buildTarget compiles for a single target platform.
func (b *CPPBuilder) buildTarget(ctx context.Context, cfg *build.Config, target build.Target) core.Result {
	if cfg == nil {
		return core.Fail(core.E("CPPBuilder.buildTarget", "config is nil", nil))
	}
	filesystem := cfg.FS
	if filesystem == nil {
		filesystem = storage.Local
		cfg.FS = filesystem
	}
	if !b.hasManagedMakefile(filesystem, cfg.ProjectDir) {
		return b.buildWithCMake(ctx, cfg, target)
	}

	// Determine if this is a cross-compile or host build
	isHostBuild := target.OS == runtime.GOOS && target.Arch == runtime.GOARCH

	if isHostBuild {
		return b.buildHost(ctx, cfg, target)
	}

	return b.buildCross(ctx, cfg, target)
}

// buildHost runs the standard make configure → make build → make package flow.
func (b *CPPBuilder) buildHost(ctx context.Context, cfg *build.Config, target build.Target) core.Result {
	core.Print(nil, "Building C++ project for %s/%s (host)", target.OS, target.Arch)

	// Step 1: Configure (runs conan install + cmake configure)
	if ran := b.runMake(ctx, cfg, "configure"); !ran.OK {
		return core.Fail(core.E("CPPBuilder.buildHost", "configure failed", core.NewError(ran.Error())))
	}

	// Step 2: Build
	if ran := b.runMake(ctx, cfg, "build"); !ran.OK {
		return core.Fail(core.E("CPPBuilder.buildHost", "build failed", core.NewError(ran.Error())))
	}

	// Step 3: Package
	if ran := b.runMake(ctx, cfg, "package"); !ran.OK {
		return core.Fail(core.E("CPPBuilder.buildHost", "package failed", core.NewError(ran.Error())))
	}

	// Discover artifacts from build/packages/
	return b.findArtifacts(cfg.FS, cfg.ProjectDir, target)
}

// buildCross runs a cross-compilation using a Conan profile name.
// The Makefile supports profile targets like: make gcc-linux-armv8
func (b *CPPBuilder) buildCross(ctx context.Context, cfg *build.Config, target build.Target) core.Result {
	// Map target to a Conan profile name
	profile := b.targetToProfile(target)
	if profile == "" {
		return core.Fail(core.E("CPPBuilder.buildCross", "no Conan profile mapped for target "+target.OS+"/"+target.Arch, nil))
	}

	core.Print(nil, "Building C++ project for %s/%s (cross: %s)", target.OS, target.Arch, profile)

	// The Makefile exposes each profile as a top-level target
	if ran := b.runMake(ctx, cfg, profile); !ran.OK {
		return core.Fail(core.E("CPPBuilder.buildCross", "cross-compile for "+profile+" failed", core.NewError(ran.Error())))
	}

	return b.findArtifacts(cfg.FS, cfg.ProjectDir, target)
}

// buildWithCMake runs a generic CMake build for plain CMakeLists.txt projects.
// Conan is used when the project declares a conanfile; otherwise the builder
// configures CMake directly.
func (b *CPPBuilder) buildWithCMake(ctx context.Context, cfg *build.Config, target build.Target) core.Result {
	filesystem := cfg.FS
	if filesystem == nil {
		filesystem = storage.Local
		cfg.FS = filesystem
	}

	platformDir := ax.Join(cfg.OutputDir, core.Sprintf("%s_%s", target.OS, target.Arch))
	if created := filesystem.EnsureDir(platformDir); !created.OK {
		return core.Fail(core.E("CPPBuilder.buildWithCMake", "failed to create platform output directory", core.NewError(created.Error())))
	}

	buildDir := ax.Join(cfg.ProjectDir, "build", "cmake", core.Sprintf("%s_%s", target.OS, target.Arch))
	if created := filesystem.EnsureDir(buildDir); !created.OK {
		return core.Fail(core.E("CPPBuilder.buildWithCMake", "failed to create cmake build directory", core.NewError(created.Error())))
	}

	env := appendConfiguredEnv(cfg,
		core.Sprintf("GOOS=%s", target.OS),
		core.Sprintf("GOARCH=%s", target.Arch),
		core.Sprintf("TARGET_OS=%s", target.OS),
		core.Sprintf("TARGET_ARCH=%s", target.Arch),
		core.Sprintf("OUTPUT_DIR=%s", cfg.OutputDir),
		core.Sprintf("TARGET_DIR=%s", platformDir),
	)
	if cfg.CGO {
		env = append(env, "CGO_ENABLED=1")
	}

	useConan := b.usesConan(filesystem, cfg.ProjectDir)
	if useConan {
		if ran := b.runConanInstall(ctx, cfg, target, buildDir, env); !ran.OK {
			return ran
		}
	}
	if ran := b.runCMakeConfigure(ctx, cfg, target, buildDir, platformDir, useConan, env); !ran.OK {
		return ran
	}
	if ran := b.runCMakeBuild(ctx, cfg, buildDir, env); !ran.OK {
		return ran
	}

	artifacts := b.findGeneratedArtifacts(filesystem, platformDir, target)
	if len(artifacts) > 0 {
		return core.Ok(artifacts)
	}

	// Some generators ignore the explicit output directory and place binaries in
	// the build tree. Fall back to scanning the cmake build directory.
	artifacts = b.findGeneratedArtifacts(filesystem, buildDir, target)
	if len(artifacts) > 0 {
		return core.Ok(artifacts)
	}

	return core.Fail(core.E("CPPBuilder.buildWithCMake", "no build output found in "+platformDir+" or "+buildDir, nil))
}

// runMake executes a make target in the project directory.
func (b *CPPBuilder) runMake(ctx context.Context, cfg *build.Config, target string) core.Result {
	makeCommandResult := b.resolveMakeCli()
	if !makeCommandResult.OK {
		return makeCommandResult
	}
	makeCommand := makeCommandResult.Value.(string)

	ran := ax.ExecWithEnv(ctx, cfg.ProjectDir, build.BuildEnvironment(cfg), makeCommand, target)
	if !ran.OK {
		return core.Fail(core.E("CPPBuilder.runMake", "make "+target+" failed", core.NewError(ran.Error())))
	}
	return core.Ok(nil)
}

func (b *CPPBuilder) runConanInstall(ctx context.Context, cfg *build.Config, target build.Target, buildDir string, env []string) core.Result {
	conanCommandResult := b.resolveConanCli()
	if !conanCommandResult.OK {
		return conanCommandResult
	}
	conanCommand := conanCommandResult.Value.(string)

	args := []string{"install", ".", "--output-folder", buildDir, "--build=missing"}
	if target.OS != runtime.GOOS || target.Arch != runtime.GOARCH {
		profile := b.targetToProfile(target)
		if profile == "" {
			return core.Fail(core.E("CPPBuilder.runConanInstall", "no Conan profile mapped for target "+target.OS+"/"+target.Arch, nil))
		}
		args = append(args, "--profile:host", profile)
	}

	output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, conanCommand, args...)
	if !output.OK {
		return core.Fail(core.E("CPPBuilder.runConanInstall", "conan install failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

func (b *CPPBuilder) runCMakeConfigure(ctx context.Context, cfg *build.Config, target build.Target, buildDir, platformDir string, useConan bool, env []string) core.Result {
	cmakeCommandResult := b.resolveCMakeCli()
	if !cmakeCommandResult.OK {
		return cmakeCommandResult
	}
	cmakeCommand := cmakeCommandResult.Value.(string)

	args := []string{
		"-S", cfg.ProjectDir,
		"-B", buildDir,
		"-DCMAKE_BUILD_TYPE=Release",
		"-DCMAKE_RUNTIME_OUTPUT_DIRECTORY=" + platformDir,
		"-DCMAKE_LIBRARY_OUTPUT_DIRECTORY=" + platformDir,
		"-DCMAKE_ARCHIVE_OUTPUT_DIRECTORY=" + platformDir,
	}
	if useConan {
		args = append(args, "-DCMAKE_TOOLCHAIN_FILE="+ax.Join(buildDir, "conan_toolchain.cmake"))
	}
	if target.OS != runtime.GOOS || target.Arch != runtime.GOARCH {
		args = append(args, "-DCORE_TARGET="+target.OS+"/"+target.Arch)
	}

	output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, cmakeCommand, args...)
	if !output.OK {
		return core.Fail(core.E("CPPBuilder.runCMakeConfigure", "cmake configure failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

func (b *CPPBuilder) runCMakeBuild(ctx context.Context, cfg *build.Config, buildDir string, env []string) core.Result {
	cmakeCommandResult := b.resolveCMakeCli()
	if !cmakeCommandResult.OK {
		return cmakeCommandResult
	}
	cmakeCommand := cmakeCommandResult.Value.(string)

	output := ax.CombinedOutput(ctx, cfg.ProjectDir, env, cmakeCommand, "--build", buildDir, "--config", "Release")
	if !output.OK {
		return core.Fail(core.E("CPPBuilder.runCMakeBuild", "cmake build failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

// findArtifacts searches for built packages in build/packages/.
func (b *CPPBuilder) findArtifacts(fs storage.Medium, projectDir string, target build.Target) core.Result {
	packagesDir := ax.Join(projectDir, "build", "packages")

	if !fs.IsDir(packagesDir) {
		// Fall back to searching build/release/src/ for raw binaries
		return b.findBinaries(fs, projectDir, target)
	}

	entriesResult := fs.List(packagesDir)
	if !entriesResult.OK {
		return core.Fail(core.E("CPPBuilder.findArtifacts", "failed to list packages directory", core.NewError(entriesResult.Error())))
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

	var artifacts []build.Artifact
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip checksum files and hidden files
		if core.HasSuffix(name, ".sha256") || core.HasPrefix(name, ".") {
			continue
		}

		artifacts = append(artifacts, build.Artifact{
			Path: ax.Join(packagesDir, name),
			OS:   target.OS,
			Arch: target.Arch,
		})
	}

	return core.Ok(artifacts)
}

// findBinaries searches for compiled binaries in build/release/src/.
func (b *CPPBuilder) findBinaries(fs storage.Medium, projectDir string, target build.Target) core.Result {
	binDir := ax.Join(projectDir, "build", "release", "src")

	if !fs.IsDir(binDir) {
		return core.Fail(core.E("CPPBuilder.findBinaries", "no build output found in "+binDir, nil))
	}

	return core.Ok(b.findGeneratedArtifacts(fs, binDir, target))
}

func (b *CPPBuilder) findGeneratedArtifacts(fs storage.Medium, dir string, target build.Target) []build.Artifact {
	if !fs.IsDir(dir) {
		return nil
	}

	entriesResult := fs.List(dir)
	if !entriesResult.OK {
		return nil
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

	var artifacts []build.Artifact
	for _, entry := range entries {
		if entry.IsDir() {
			if target.OS == "darwin" && core.HasSuffix(entry.Name(), ".app") {
				artifacts = append(artifacts, build.Artifact{
					Path: ax.Join(dir, entry.Name()),
					OS:   target.OS,
					Arch: target.Arch,
				})
			}
			continue
		}

		name := entry.Name()
		// Skip common build metadata and non-runtime artefacts.
		if core.HasPrefix(name, ".") ||
			core.HasPrefix(name, "CMake") ||
			core.HasPrefix(name, "cmake") ||
			core.HasPrefix(name, "conan") ||
			core.HasSuffix(name, ".a") ||
			core.HasSuffix(name, ".o") ||
			core.HasSuffix(name, ".cmake") ||
			core.HasSuffix(name, ".ninja") ||
			core.HasSuffix(name, ".txt") ||
			name == "Makefile" {
			continue
		}

		fullPath := ax.Join(dir, name)

		// On Unix, check if file is executable
		if target.OS != "windows" {
			info := fs.Stat(fullPath)
			if !info.OK {
				continue
			}
			if info.Value.(stdfs.FileInfo).Mode()&0111 == 0 {
				continue
			}
		}

		artifacts = append(artifacts, build.Artifact{
			Path: fullPath,
			OS:   target.OS,
			Arch: target.Arch,
		})
	}

	return artifacts
}

// targetToProfile maps a build target to a Conan cross-compilation profile name.
// Profile names match those in .core/build/cmake/profiles/.
func (b *CPPBuilder) targetToProfile(target build.Target) string {
	key := target.OS + "/" + target.Arch
	profiles := map[string]string{
		"linux/amd64":    "gcc-linux-x86_64",
		"linux/x86_64":   "gcc-linux-x86_64",
		"linux/arm64":    "gcc-linux-armv8",
		"linux/armv8":    "gcc-linux-armv8",
		"darwin/arm64":   "apple-clang-armv8",
		"darwin/armv8":   "apple-clang-armv8",
		"darwin/amd64":   "apple-clang-x86_64",
		"darwin/x86_64":  "apple-clang-x86_64",
		"windows/amd64":  "msvc-194-x86_64",
		"windows/x86_64": "msvc-194-x86_64",
	}

	return profiles[key]
}

// validateMake checks if make is available.
func (b *CPPBuilder) validateMake() core.Result {
	return b.resolveMakeCli()
}

// validateConan checks if conan is available.
func (b *CPPBuilder) validateConan() core.Result {
	return b.resolveConanCli()
}

// validateCMake checks if cmake is available.
func (b *CPPBuilder) validateCMake() core.Result {
	return b.resolveCMakeCli()
}

// resolveMakeCli returns the executable path for make or gmake.
func (b *CPPBuilder) resolveMakeCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/make",
			"/usr/local/bin/make",
			"/opt/homebrew/bin/make",
			"/usr/local/bin/gmake",
			"/opt/homebrew/bin/gmake",
		}
	}

	command := ax.ResolveCommand("make", paths...)
	if !command.OK {
		return core.Fail(core.E("CPPBuilder.resolveMakeCli", "make not found. Install build-essential (Linux) or Xcode Command Line Tools (macOS)", core.NewError(command.Error())))
	}

	return command
}

// resolveConanCli returns the executable path for conan.
func (b *CPPBuilder) resolveConanCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/local/bin/conan",
			"/opt/homebrew/bin/conan",
		}

		if home := core.Env("HOME"); home != "" {
			paths = append(paths, ax.Join(home, ".local", "bin", "conan"))
		}
	}

	command := ax.ResolveCommand("conan", paths...)
	if !command.OK {
		return core.Fail(core.E("CPPBuilder.resolveConanCli", "conan not found. Install it with: python -m pip install conan", core.NewError(command.Error())))
	}

	return command
}

// resolveCMakeCli returns the executable path for cmake.
func (b *CPPBuilder) resolveCMakeCli(paths ...string) core.Result {
	if len(paths) == 0 {
		paths = []string{
			"/usr/bin/cmake",
			"/usr/local/bin/cmake",
			"/opt/homebrew/bin/cmake",
		}
	}

	command := ax.ResolveCommand("cmake", paths...)
	if !command.OK {
		return core.Fail(core.E("CPPBuilder.resolveCMakeCli", "cmake not found. Install it with: brew install cmake or apt-get install cmake", core.NewError(command.Error())))
	}

	return command
}

func (b *CPPBuilder) hasManagedMakefile(fs storage.Medium, dir string) bool {
	if fs == nil {
		fs = storage.Local
	}

	for _, name := range []string{"Makefile", "GNUmakefile", "makefile"} {
		if fs.IsFile(ax.Join(dir, name)) {
			return true
		}
	}

	return false
}

func (b *CPPBuilder) usesConan(fs storage.Medium, dir string) bool {
	if fs == nil {
		fs = storage.Local
	}

	return fs.IsFile(ax.Join(dir, "conanfile.py")) || fs.IsFile(ax.Join(dir, "conanfile.txt"))
}

// Ensure CPPBuilder implements the Builder interface.
var _ build.Builder = (*CPPBuilder)(nil)
