package builders

import (
	"archive/zip"
	stdio "io"
	stdfs "io/fs"
	"slices"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// appendConfiguredEnv returns a fresh environment slice that includes the
// configured build environment, derived cache variables, and any
// builder-specific values.
func appendConfiguredEnv(cfg *build.Config, extra ...string) []string {
	return build.BuildEnvironment(cfg, extra...)
}

// ensureBuildFilesystem returns the filesystem associated with cfg, falling
// back to io.Local for zero-value configs. When cfg is non-nil, the fallback is
// also written back so downstream helpers that read cfg.FS stay safe.
func ensureBuildFilesystem(cfg *build.Config) io.Medium {
	if cfg == nil {
		return io.Local
	}
	if cfg.FS == nil {
		cfg.FS = io.Local
	}
	return cfg.FS
}

func defaultHostTargets(targets []build.Target) []build.Target {
	if len(targets) > 0 {
		return targets
	}
	return []build.Target{{OS: core.Env("GOOS"), Arch: core.Env("GOARCH")}}
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

func ensureOutputDir(fs io.Medium, outputDir, operation string) error {
	if outputDir == "" {
		return nil
	}
	if err := fs.EnsureDir(outputDir); err != nil {
		return coreerr.E(operation, "failed to create output directory", err)
	}
	return nil
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

func ensurePlatformDir(fs io.Medium, outputDir string, target build.Target, operation string) (string, error) {
	dir := platformDir(outputDir, target)
	if err := fs.EnsureDir(dir); err != nil {
		return "", coreerr.E(operation, "failed to create platform directory", err)
	}
	return dir, nil
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
	commandFS        io.Medium
	cleanup          func()
}

func prepareStagedOutput(outputDir string, artifactFS io.Medium, tempPattern, operation string) (stagedOutput, error) {
	stage := stagedOutput{
		outputDir:        outputDir,
		commandOutputDir: outputDir,
		commandFS:        artifactFS,
		cleanup:          func() {},
	}
	if build.MediumIsLocal(artifactFS) {
		return stage, nil
	}

	stageDir, err := ax.TempDir(tempPattern)
	if err != nil {
		return stagedOutput{}, coreerr.E(operation, "failed to create local artifact staging directory", err)
	}
	stage.commandOutputDir = stageDir
	stage.commandFS = io.Local
	stage.cleanup = func() { _ = ax.RemoveAll(stageDir) }
	return stage, nil
}

type zipExcludeFunc func(path string) bool

func bundleZipTree(fs io.Medium, rootDir, bundlePath, operation string, exclude zipExcludeFunc) error {
	if err := fs.EnsureDir(ax.Dir(bundlePath)); err != nil {
		return coreerr.E(operation, "failed to create bundle directory", err)
	}

	file, err := fs.Create(bundlePath)
	if err != nil {
		return coreerr.E(operation, "failed to create bundle file", err)
	}
	defer func() { _ = file.Close() }()

	writer := zip.NewWriter(file)
	defer func() { _ = writer.Close() }()

	return writeZipTree(fs, writer, rootDir, rootDir, operation, exclude)
}

func writeZipTree(fs io.Medium, writer *zip.Writer, rootDir, currentDir, operation string, exclude zipExcludeFunc) error {
	entries, err := fs.List(currentDir)
	if err != nil {
		return coreerr.E(operation, "failed to list directory", err)
	}

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
			if err := writeZipTree(fs, writer, rootDir, entryPath, operation, exclude); err != nil {
				return err
			}
			continue
		}

		if err := writeZipEntry(fs, writer, rootDir, entryPath, operation); err != nil {
			return err
		}
	}

	return nil
}

func writeZipEntry(fs io.Medium, writer *zip.Writer, rootDir, entryPath, operation string) error {
	relPath, err := ax.Rel(rootDir, entryPath)
	if err != nil {
		return coreerr.E(operation, "failed to relativise bundle path", err)
	}

	info, err := fs.Stat(entryPath)
	if err != nil {
		return coreerr.E(operation, "failed to stat bundle entry", err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return coreerr.E(operation, "failed to create zip header", err)
	}
	header.Name = core.Replace(relPath, ax.DS(), "/")
	header.Method = zip.Deflate
	header.SetModTime(deterministicZipTime)

	zipEntry, err := writer.CreateHeader(header)
	if err != nil {
		return coreerr.E(operation, "failed to create zip entry", err)
	}

	source, err := fs.Open(entryPath)
	if err != nil {
		return coreerr.E(operation, "failed to open bundle entry", err)
	}

	if _, err := stdio.Copy(zipEntry, source); err != nil {
		_ = source.Close()
		return coreerr.E(operation, "failed to write bundle entry", err)
	}
	if err := source.Close(); err != nil {
		return coreerr.E(operation, "failed to close bundle entry", err)
	}

	return nil
}
