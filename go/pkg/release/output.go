package release

import (
	"reflect"

	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/build/pkg/storage"
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
