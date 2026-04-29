package release

import (
	stdio "io"
	"io/fs"
	"reflect"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
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

func mirrorReleaseArtifacts(source, destination coreio.Medium, sourceRoot, destinationRoot string, artifacts []build.Artifact) core.Result {
	if source == nil {
		source = coreio.Local
	}
	if destination == nil {
		destination = coreio.Local
	}

	mirrored := make([]build.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		relativePathResult := ax.Rel(sourceRoot, artifact.Path)
		relativePath := ""
		if relativePathResult.OK {
			relativePath = relativePathResult.Value.(string)
		}
		if relativePath == "" || core.HasPrefix(relativePath, "..") {
			relativePath = ax.Base(artifact.Path)
		}

		destinationPath := joinReleasePath(destinationRoot, relativePath)
		copied := copyReleaseMediumPath(source, artifact.Path, destination, destinationPath)
		if !copied.OK {
			return core.Fail(coreerr.E("release.mirrorReleaseArtifacts", "failed to mirror artifact "+artifact.Path, core.NewError(copied.Error())))
		}

		mirrored = append(mirrored, build.Artifact{
			Path:     destinationPath,
			OS:       artifact.OS,
			Arch:     artifact.Arch,
			Checksum: artifact.Checksum,
		})
	}

	return core.Ok(mirrored)
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

func copyReleaseMediumPath(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) core.Result {
	if source.IsDir(sourcePath) {
		return copyReleaseMediumDir(source, sourcePath, destination, destinationPath)
	}

	return copyReleaseMediumFile(source, sourcePath, destination, destinationPath)
}

func copyReleaseMediumDir(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) core.Result {
	created := destination.EnsureDir(destinationPath)
	if !created.OK {
		return core.Fail(coreerr.E("release.copyReleaseMediumDir", "failed to create destination directory", core.NewError(created.Error())))
	}

	entriesResult := source.List(sourcePath)
	if !entriesResult.OK {
		return core.Fail(coreerr.E("release.copyReleaseMediumDir", "failed to list source directory", core.NewError(entriesResult.Error())))
	}
	entries := entriesResult.Value.([]fs.DirEntry)

	for _, entry := range entries {
		childSourcePath := ax.Join(sourcePath, entry.Name())
		childDestinationPath := ax.Join(destinationPath, entry.Name())
		copied := copyReleaseMediumPath(source, childSourcePath, destination, childDestinationPath)
		if !copied.OK {
			return copied
		}
	}

	return core.Ok(nil)
}

func copyReleaseMediumFile(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) core.Result {
	fileResult := source.Open(sourcePath)
	if !fileResult.OK {
		return core.Fail(coreerr.E("release.copyReleaseMediumFile", "failed to open source file", core.NewError(fileResult.Error())))
	}
	file := fileResult.Value.(core.FsFile)
	defer file.Close()

	content, readFailure := stdio.ReadAll(file)
	if readFailure != nil {
		return core.Fail(coreerr.E("release.copyReleaseMediumFile", "failed to read source file", readFailure))
	}

	mode := fs.FileMode(0o644)
	infoResult := source.Stat(sourcePath)
	if infoResult.OK {
		mode = infoResult.Value.(fs.FileInfo).Mode()
	}

	written := destination.WriteMode(destinationPath, string(content), mode)
	if !written.OK {
		return core.Fail(coreerr.E("release.copyReleaseMediumFile", "failed to write destination file", core.NewError(written.Error())))
	}

	return core.Ok(nil)
}
