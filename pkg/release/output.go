package release

import (
	stdio "io"
	"io/fs"
	"reflect"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

func resolveReleaseOutputMedium(cfg *Config) coreio.Medium {
	if cfg == nil || cfg.output == nil {
		return coreio.Local
	}
	return cfg.output
}

func resolveReleaseOutputRoot(projectDir string, cfg *Config, output coreio.Medium) string {
	outputDir := ""
	if cfg != nil {
		outputDir = cfg.outputDir
	}

	if outputDir == "" && !mediumEquals(output, coreio.Local) {
		return ""
	}

	if outputDir == "" {
		outputDir = "dist"
	}

	if !ax.IsAbs(outputDir) && mediumEquals(output, coreio.Local) {
		return ax.Join(projectDir, outputDir)
	}

	return outputDir
}

func mirrorReleaseArtifacts(source, destination coreio.Medium, sourceRoot, destinationRoot string, artifacts []build.Artifact) ([]build.Artifact, error) {
	if source == nil {
		source = coreio.Local
	}
	if destination == nil {
		destination = coreio.Local
	}

	mirrored := make([]build.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		relativePath, err := ax.Rel(sourceRoot, artifact.Path)
		if err != nil || relativePath == "" || core.HasPrefix(relativePath, "..") {
			relativePath = ax.Base(artifact.Path)
		}

		destinationPath := joinReleasePath(destinationRoot, relativePath)
		if err := copyReleaseMediumPath(source, artifact.Path, destination, destinationPath); err != nil {
			return nil, coreerr.E("release.mirrorReleaseArtifacts", "failed to mirror artifact "+artifact.Path, err)
		}

		mirrored = append(mirrored, build.Artifact{
			Path:     destinationPath,
			OS:       artifact.OS,
			Arch:     artifact.Arch,
			Checksum: artifact.Checksum,
		})
	}

	return mirrored, nil
}

func joinReleasePath(root, path string) string {
	if root == "" || root == "." {
		return ax.Clean(path)
	}
	if path == "" || path == "." {
		return ax.Clean(root)
	}
	return ax.Join(root, path)
}

func mediumEquals(left, right coreio.Medium) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	leftType := reflect.TypeOf(left)
	rightType := reflect.TypeOf(right)
	if leftType != rightType || !leftType.Comparable() {
		return false
	}

	return reflect.ValueOf(left).Interface() == reflect.ValueOf(right).Interface()
}

func copyReleaseMediumPath(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) error {
	if source.IsDir(sourcePath) {
		return copyReleaseMediumDir(source, sourcePath, destination, destinationPath)
	}

	return copyReleaseMediumFile(source, sourcePath, destination, destinationPath)
}

func copyReleaseMediumDir(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) error {
	if err := destination.EnsureDir(destinationPath); err != nil {
		return coreerr.E("release.copyReleaseMediumDir", "failed to create destination directory", err)
	}

	entries, err := source.List(sourcePath)
	if err != nil {
		return coreerr.E("release.copyReleaseMediumDir", "failed to list source directory", err)
	}

	for _, entry := range entries {
		childSourcePath := ax.Join(sourcePath, entry.Name())
		childDestinationPath := ax.Join(destinationPath, entry.Name())
		if err := copyReleaseMediumPath(source, childSourcePath, destination, childDestinationPath); err != nil {
			return err
		}
	}

	return nil
}

func copyReleaseMediumFile(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) error {
	file, err := source.Open(sourcePath)
	if err != nil {
		return coreerr.E("release.copyReleaseMediumFile", "failed to open source file", err)
	}
	defer func() { _ = file.Close() }()

	content, err := stdio.ReadAll(file)
	if err != nil {
		return coreerr.E("release.copyReleaseMediumFile", "failed to read source file", err)
	}

	mode := fs.FileMode(0o644)
	if info, err := source.Stat(sourcePath); err == nil {
		mode = info.Mode()
	}

	if err := destination.WriteMode(destinationPath, string(content), mode); err != nil {
		return coreerr.E("release.copyReleaseMediumFile", "failed to write destination file", err)
	}

	return nil
}
