package builders

import (
	"archive/zip"
	stdio "io"
	stdfs "io/fs"
	"runtime"
	"slices"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	storage "dappco.re/go/build/pkg/storage"
)

// appendConfiguredEnv returns a fresh environment slice that includes the
// configured build environment, derived cache variables, and any
// builder-specific values.
func appendConfiguredEnv(cfg *build.Config, extra ...string) []string {
	return build.BuildEnvironment(cfg, extra...)
}

// ensureBuildFilesystem returns the filesystem associated with cfg, falling
// back to storage.Local for zero-value configs. When cfg is non-nil, the fallback is
// also written back so downstream helpers that read cfg.FS stay safe.
func ensureBuildFilesystem(cfg *build.Config) storage.Medium {
	if cfg == nil {
		return storage.Local
	}
	if cfg.FS == nil {
		cfg.FS = storage.Local
	}
	return cfg.FS
}

func defaultHostTargets(targets []build.Target) []build.Target {
	if len(targets) > 0 {
		return targets
	}
	goos := core.Env("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := core.Env("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	return []build.Target{{OS: goos, Arch: goarch}}
}

func defaultRuntimeTargets(targets []build.Target, osName, archName string) []build.Target {
	if len(targets) > 0 {
		return targets
	}
	return []build.Target{{OS: osName, Arch: archName}}
}

func defaultLinuxTargets(targets []build.Target) []build.Target {
	if len(targets) > 0 {
		return targets
	}
	return []build.Target{{OS: "linux", Arch: "amd64"}}
}

func defaultOutputDir(cfg *build.Config) string {
	if cfg == nil || cfg.OutputDir != "" {
		return ""
	}
	return ax.Join(cfg.ProjectDir, "dist")
}

func ensureOutputDir(fs storage.Medium, outputDir, operation string) core.Result {
	if outputDir == "" {
		return core.Ok(nil)
	}
	created := fs.EnsureDir(outputDir)
	if !created.OK {
		return core.Fail(core.E(operation, "failed to create output directory", core.NewError(created.Error())))
	}
	return core.Ok(nil)
}

func platformName(target build.Target) string {
	return core.Sprintf("%s_%s", target.OS, target.Arch)
}

func platformDir(outputDir string, target build.Target) string {
	name := platformName(target)
	if outputDir == "" {
		return name
	}
	return ax.Join(outputDir, name)
}

func ensurePlatformDir(fs storage.Medium, outputDir string, target build.Target, operation string) core.Result {
	dir := platformDir(outputDir, target)
	created := fs.EnsureDir(dir)
	if !created.OK {
		return core.Fail(core.E(operation, "failed to create platform directory", core.NewError(created.Error())))
	}
	return core.Ok(dir)
}

func standardTargetValues(outputDir, targetDir string, target build.Target) []string {
	return []string{
		core.Sprintf("GOOS=%s", target.OS),
		core.Sprintf("GOARCH=%s", target.Arch),
		core.Sprintf("TARGET_OS=%s", target.OS),
		core.Sprintf("TARGET_ARCH=%s", target.Arch),
		core.Sprintf("OUTPUT_DIR=%s", outputDir),
		core.Sprintf("TARGET_DIR=%s", targetDir),
	}
}

func configuredTargetEnv(cfg *build.Config, target build.Target, values ...string) []string {
	env := appendConfiguredEnv(cfg, values...)
	return appendNameVersionEnv(env, cfg)
}

func appendNameVersionEnv(env []string, cfg *build.Config) []string {
	if cfg == nil {
		return env
	}
	if cfg.Name != "" {
		env = append(env, core.Sprintf("NAME=%s", cfg.Name))
	}
	if cfg.Version != "" {
		env = append(env, core.Sprintf("VERSION=%s", cfg.Version))
	}
	return env
}

func cgoEnvValue(enabled bool) string {
	if enabled {
		return "CGO_ENABLED=1"
	}
	return "CGO_ENABLED=0"
}

type stagedOutput struct {
	outputDir        string
	commandOutputDir string
	commandFS        storage.Medium
	cleanup          func()
}

func prepareStagedOutput(outputDir string, artifactFS storage.Medium, tempPattern, operation string) core.Result {
	stage := stagedOutput{
		outputDir:        outputDir,
		commandOutputDir: outputDir,
		commandFS:        artifactFS,
		cleanup:          func() {},
	}
	if build.MediumIsLocal(artifactFS) {
		return core.Ok(stage)
	}

	stageDirResult := ax.TempDir(tempPattern)
	if !stageDirResult.OK {
		return core.Fail(core.E(operation, "failed to create local artifact staging directory", core.NewError(stageDirResult.Error())))
	}
	stageDir := stageDirResult.Value.(string)
	stage.commandOutputDir = stageDir
	stage.commandFS = storage.Local
	stage.cleanup = func() { ax.RemoveAll(stageDir) }
	return core.Ok(stage)
}

type zipExcludeFunc func(path string) bool

func bundleZipTree(fs storage.Medium, rootDir, bundlePath, operation string, exclude zipExcludeFunc) core.Result {
	created := fs.EnsureDir(ax.Dir(bundlePath))
	if !created.OK {
		return core.Fail(core.E(operation, "failed to create bundle directory", core.NewError(created.Error())))
	}

	fileResult := fs.Create(bundlePath)
	if !fileResult.OK {
		return core.Fail(core.E(operation, "failed to create bundle file", core.NewError(fileResult.Error())))
	}
	file := fileResult.Value.(core.WriteCloser)
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	return writeZipTree(fs, writer, rootDir, rootDir, operation, exclude)
}

func writeZipTree(fs storage.Medium, writer *zip.Writer, rootDir, currentDir, operation string, exclude zipExcludeFunc) core.Result {
	entriesResult := fs.List(currentDir)
	if !entriesResult.OK {
		return core.Fail(core.E(operation, "failed to list directory", core.NewError(entriesResult.Error())))
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

	slices.SortFunc(entries, func(a, b stdfs.DirEntry) int {
		if a.Name() < b.Name() {
			return -1
		}
		if a.Name() > b.Name() {
			return 1
		}
		return 0
	})

	for _, entry := range entries {
		entryPath := ax.Join(currentDir, entry.Name())
		if exclude != nil && exclude(entryPath) {
			continue
		}

		if entry.IsDir() {
			written := writeZipTree(fs, writer, rootDir, entryPath, operation, exclude)
			if !written.OK {
				return written
			}
			continue
		}

		written := writeZipEntry(fs, writer, rootDir, entryPath, operation)
		if !written.OK {
			return written
		}
	}

	return core.Ok(nil)
}

func writeZipEntry(fs storage.Medium, writer *zip.Writer, rootDir, entryPath, operation string) core.Result {
	relPathResult := ax.Rel(rootDir, entryPath)
	if !relPathResult.OK {
		return core.Fail(core.E(operation, "failed to relativise bundle path", core.NewError(relPathResult.Error())))
	}
	relPath := relPathResult.Value.(string)

	infoResult := fs.Stat(entryPath)
	if !infoResult.OK {
		return core.Fail(core.E(operation, "failed to stat bundle entry", core.NewError(infoResult.Error())))
	}
	info := infoResult.Value.(stdfs.FileInfo)

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return core.Fail(core.E(operation, "failed to create zip header", err))
	}
	header.Name = core.Replace(relPath, ax.DS(), "/")
	header.Method = zip.Deflate
	header.SetModTime(deterministicZipTime)

	zipEntry, err := writer.CreateHeader(header)
	if err != nil {
		return core.Fail(core.E(operation, "failed to create zip entry", err))
	}

	sourceResult := fs.Open(entryPath)
	if !sourceResult.OK {
		return core.Fail(core.E(operation, "failed to open bundle entry", core.NewError(sourceResult.Error())))
	}
	source := sourceResult.Value.(core.FsFile)

	if _, err := stdio.Copy(zipEntry, source); err != nil {
		if closeErr := source.Close(); closeErr != nil {
			return core.Fail(core.E(operation, "failed to close bundle entry after write failure", closeErr))
		}
		return core.Fail(core.E(operation, "failed to write bundle entry", err))
	}
	if err := source.Close(); err != nil {
		return core.Fail(core.E(operation, "failed to close bundle entry", err))
	}

	return core.Ok(nil)
}
