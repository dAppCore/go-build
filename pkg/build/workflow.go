// Package build provides project type detection and cross-compilation for the Core build system.
// This file exposes the release workflow generator and its path-resolution helpers.
package build

import (
	"embed"
	"strings"

	"dappco.re/go/core/build/internal/ax"
	io_interface "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

//go:embed templates/release.yml
var releaseWorkflowTemplate embed.FS

// DefaultReleaseWorkflowPath is the conventional output path for the release workflow.
//
// path := build.DefaultReleaseWorkflowPath // ".github/workflows/release.yml"
const DefaultReleaseWorkflowPath = ".github/workflows/release.yml"

// DefaultReleaseWorkflowFileName is the workflow filename used when a directory-style
// output path is supplied.
const DefaultReleaseWorkflowFileName = "release.yml"

// WriteReleaseWorkflow writes the embedded release workflow template to outputPath.
//
// build.WriteReleaseWorkflow(io.Local, "")                                        // writes .github/workflows/release.yml
// build.WriteReleaseWorkflow(io.Local, "ci")                                       // writes ./ci/release.yml under the project root
// build.WriteReleaseWorkflow(io.Local, "./ci")                                     // writes ./ci/release.yml under the project root
// build.WriteReleaseWorkflow(io.Local, ".github/workflows")                       // writes .github/workflows/release.yml
// build.WriteReleaseWorkflow(io.Local, "ci/release.yml")                          // writes ./ci/release.yml under the project root
// build.WriteReleaseWorkflow(io.Local, "/tmp/repo/.github/workflows/release.yml") // writes the absolute path unchanged
func WriteReleaseWorkflow(filesystem io_interface.Medium, outputPath string) error {
	if filesystem == nil {
		return coreerr.E("build.WriteReleaseWorkflow", "filesystem medium is required", nil)
	}

	outputPath = cleanWorkflowInput(outputPath)
	if outputPath == "" {
		outputPath = DefaultReleaseWorkflowPath
	}

	if isWorkflowDirectoryInput(outputPath) || filesystem.IsDir(outputPath) {
		outputPath = ax.Join(outputPath, DefaultReleaseWorkflowFileName)
	}

	content, err := releaseWorkflowTemplate.ReadFile("templates/release.yml")
	if err != nil {
		return coreerr.E("build.WriteReleaseWorkflow", "failed to read embedded workflow template", err)
	}

	if err := filesystem.EnsureDir(ax.Dir(outputPath)); err != nil {
		return coreerr.E("build.WriteReleaseWorkflow", "failed to create release workflow directory", err)
	}

	if err := filesystem.Write(outputPath, string(content)); err != nil {
		return coreerr.E("build.WriteReleaseWorkflow", "failed to write release workflow", err)
	}

	return nil
}

// ReleaseWorkflowPath joins a project directory with the conventional workflow path.
//
// build.ReleaseWorkflowPath("/home/user/project") // /home/user/project/.github/workflows/release.yml
func ReleaseWorkflowPath(projectDir string) string {
	return ax.Join(projectDir, DefaultReleaseWorkflowPath)
}

// ResolveReleaseWorkflowOutputPathWithMedium resolves the workflow output path
// relative to the project directory and treats an existing directory as a
// workflow directory even when the caller omits a trailing slash.
//
// build.ResolveReleaseWorkflowOutputPathWithMedium(io.Local, "/tmp/project", "ci")            // /tmp/project/ci/release.yml when /tmp/project/ci exists
// build.ResolveReleaseWorkflowOutputPathWithMedium(io.Local, "/tmp/project", ".github/workflows") // /tmp/project/.github/workflows/release.yml
func ResolveReleaseWorkflowOutputPathWithMedium(filesystem io_interface.Medium, projectDir, outputPath string) string {
	outputPath = cleanWorkflowInput(outputPath)
	if outputPath == "" {
		return ReleaseWorkflowPath(projectDir)
	}

	resolved := ResolveReleaseWorkflowPath(projectDir, outputPath)
	if filesystem != nil && filesystem.IsDir(resolved) {
		return ax.Join(resolved, DefaultReleaseWorkflowFileName)
	}

	return resolved
}

// ResolveReleaseWorkflowPath resolves the workflow output path relative to the
// project directory when the caller supplies a relative path.
//
// build.ResolveReleaseWorkflowPath("/tmp/project", "")                // /tmp/project/.github/workflows/release.yml
// build.ResolveReleaseWorkflowPath("/tmp/project", "./ci")            // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowPath("/tmp/project", ".github/workflows") // /tmp/project/.github/workflows/release.yml
// build.ResolveReleaseWorkflowPath("/tmp/project", "ci/release.yml")   // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowPath("/tmp/project", "ci")               // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowPath("/tmp/project", "/tmp/release.yml") // /tmp/release.yml
func ResolveReleaseWorkflowPath(projectDir, outputPath string) string {
	outputPath = cleanWorkflowInput(outputPath)
	if outputPath == "" {
		return ReleaseWorkflowPath(projectDir)
	}
	if isWorkflowDirectoryPath(outputPath) || isWorkflowDirectoryInput(outputPath) {
		if ax.IsAbs(outputPath) {
			return ax.Join(outputPath, DefaultReleaseWorkflowFileName)
		}
		return ax.Join(projectDir, outputPath, DefaultReleaseWorkflowFileName)
	}
	if !ax.IsAbs(outputPath) {
		return ax.Join(projectDir, outputPath)
	}
	return outputPath
}

// ResolveReleaseWorkflowInputPath resolves the workflow path from the CLI/API
// `path` field and its `output` alias.
//
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "", "")                      // /tmp/project/.github/workflows/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "./ci", "")                  // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "")        // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "", "ci/release.yml")        // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPath("/tmp/project", "ci/release.yml", "ci.yml")  // error
func ResolveReleaseWorkflowInputPath(projectDir, pathInput, outputPathInput string) (string, error) {
	return resolveReleaseWorkflowInputPathPair(
		pathInput,
		outputPathInput,
		func(input string) string {
			return resolveReleaseWorkflowInputPath(projectDir, input, nil)
		},
		"build.ResolveReleaseWorkflowInputPath",
	)
}

// ResolveReleaseWorkflowInputPathWithMedium resolves the workflow path and
// treats an existing directory as a directory even when the caller omits a
// trailing slash.
//
// build.ResolveReleaseWorkflowInputPathWithMedium(io.Local, "/tmp/project", "ci", "") // /tmp/project/ci/release.yml when /tmp/project/ci exists
// build.ResolveReleaseWorkflowInputPathWithMedium(io.Local, "/tmp/project", "./ci", "") // /tmp/project/ci/release.yml
func ResolveReleaseWorkflowInputPathWithMedium(filesystem io_interface.Medium, projectDir, pathInput, outputPathInput string) (string, error) {
	return resolveReleaseWorkflowInputPathPair(
		pathInput,
		outputPathInput,
		func(input string) string {
			return resolveReleaseWorkflowInputPath(projectDir, input, filesystem)
		},
		"build.ResolveReleaseWorkflowInputPathWithMedium",
	)
}

// ResolveReleaseWorkflowInputPathAliases resolves the workflow path across the
// public path aliases and treats an existing directory as a directory even
// when the caller omits a trailing slash.
//
// build.ResolveReleaseWorkflowInputPathAliases(io.Local, "/tmp/project", "ci", "", "", "") // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPathAliases(io.Local, "/tmp/project", "", "ci", "", "")  // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPathAliases(io.Local, "/tmp/project", "", "", "ci", "")  // /tmp/project/ci/release.yml
// build.ResolveReleaseWorkflowInputPathAliases(io.Local, "/tmp/project", "", "", "", "ci")  // /tmp/project/ci/release.yml
func ResolveReleaseWorkflowInputPathAliases(filesystem io_interface.Medium, projectDir, pathInput, workflowPathInput, workflowPathSnakeInput, workflowPathHyphenInput string) (string, error) {
	return resolveReleaseWorkflowInputPathAliasSet(
		filesystem,
		projectDir,
		"path",
		pathInput,
		workflowPathInput,
		workflowPathSnakeInput,
		workflowPathHyphenInput,
		"build.ResolveReleaseWorkflowInputPathAliases",
	)
}

// ResolveReleaseWorkflowOutputPath resolves the core workflow output aliases.
//
// path, err := build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "", "")
// path, err := build.ResolveReleaseWorkflowOutputPath("", "ci/release.yml", "")
// path, err := build.ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml")
//
// build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "", "")        // "ci/release.yml"
// build.ResolveReleaseWorkflowOutputPath("", "ci/release.yml", "")        // "ci/release.yml"
// build.ResolveReleaseWorkflowOutputPath("", "", "ci/release.yml")        // "ci/release.yml"
// build.ResolveReleaseWorkflowOutputPath("ci/release.yml", "ops.yml", "") // error
func ResolveReleaseWorkflowOutputPath(outputPathInput, outputPathSnakeInput, legacyOutputInput string) (string, error) {
	return ResolveReleaseWorkflowOutputPathAliases(
		outputPathInput,
		"",
		outputPathSnakeInput,
		legacyOutputInput,
		"",
		"",
		"",
		"",
		"",
	)
}

// ResolveReleaseWorkflowOutputPathAliases resolves every public workflow output
// alias across the CLI, API, and UI layers.
//
// build.ResolveReleaseWorkflowOutputPathAliases("ci/release.yml", "", "", "", "", "", "", "", "") // "ci/release.yml"
// build.ResolveReleaseWorkflowOutputPathAliases("", "ci/release.yml", "", "", "", "", "", "", "")  // "ci/release.yml"
// build.ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "ci/release.yml", "", "", "", "")  // "ci/release.yml"
// build.ResolveReleaseWorkflowOutputPathAliases("", "", "", "", "", "ci/release.yml", "", "", "")  // "ci/release.yml"
func ResolveReleaseWorkflowOutputPathAliases(
	outputPathInput,
	outputPathHyphenInput,
	outputPathSnakeInput,
	legacyOutputInput,
	workflowOutputPathInput,
	workflowOutputSnakeInput,
	workflowOutputHyphenInput,
	workflowOutputPathSnakeInput,
	workflowOutputPathHyphenInput string,
) (string, error) {
	return resolveReleaseWorkflowOutputAliasSet(
		outputPathInput,
		outputPathHyphenInput,
		outputPathSnakeInput,
		legacyOutputInput,
		workflowOutputPathInput,
		workflowOutputSnakeInput,
		workflowOutputHyphenInput,
		workflowOutputPathSnakeInput,
		workflowOutputPathHyphenInput,
		"build.ResolveReleaseWorkflowOutputPathAliases",
	)
}

// ResolveReleaseWorkflowOutputPathAliasesInProject resolves the workflow output
// aliases relative to a project directory before checking for conflicts.
//
// build.ResolveReleaseWorkflowOutputPathAliasesInProject("/tmp/project", "ci/release.yml", "", "", "", "", "", "", "")               // "/tmp/project/ci/release.yml"
// build.ResolveReleaseWorkflowOutputPathAliasesInProject("/tmp/project", "", "", "", "", "/tmp/project/ci/release.yml", "", "", "") // "/tmp/project/ci/release.yml"
func ResolveReleaseWorkflowOutputPathAliasesInProject(
	projectDir,
	outputPathInput,
	outputPathHyphenInput,
	outputPathSnakeInput,
	legacyOutputInput,
	workflowOutputPathInput,
	workflowOutputSnakeInput,
	workflowOutputHyphenInput,
	workflowOutputPathSnakeInput,
	workflowOutputPathHyphenInput string,
) (string, error) {
	return ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(
		nil,
		projectDir,
		outputPathInput,
		outputPathHyphenInput,
		outputPathSnakeInput,
		legacyOutputInput,
		workflowOutputPathInput,
		workflowOutputSnakeInput,
		workflowOutputHyphenInput,
		workflowOutputPathSnakeInput,
		workflowOutputPathHyphenInput,
	)
}

// ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium resolves the
// workflow output aliases relative to a project directory and uses the
// provided filesystem medium to treat existing directories as workflow
// directories even when callers omit a trailing separator.
//
// build.ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(io.Local, "/tmp/project", "", "", "", "", "/tmp/project/ci", "", "", "") // "/tmp/project/ci/release.yml"
func ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(
	filesystem io_interface.Medium,
	projectDir,
	outputPathInput,
	outputPathHyphenInput,
	outputPathSnakeInput,
	legacyOutputInput,
	workflowOutputPathInput,
	workflowOutputSnakeInput,
	workflowOutputHyphenInput,
	workflowOutputPathSnakeInput,
	workflowOutputPathHyphenInput string,
) (string, error) {
	return resolveReleaseWorkflowOutputAliasSetInProject(
		filesystem,
		projectDir,
		outputPathInput,
		outputPathHyphenInput,
		outputPathSnakeInput,
		legacyOutputInput,
		workflowOutputPathInput,
		workflowOutputSnakeInput,
		workflowOutputHyphenInput,
		workflowOutputPathSnakeInput,
		workflowOutputPathHyphenInput,
		"build.ResolveReleaseWorkflowOutputPathAliasesInProject",
	)
}

// resolveReleaseWorkflowInputPathPair resolves the workflow path from the path
// and output aliases, rejecting conflicting values and preferring explicit
// inputs over the default.
func resolveReleaseWorkflowInputPathPair(pathInput, outputPathInput string, resolve func(string) string, errorName string) (string, error) {
	pathInput = cleanWorkflowInput(pathInput)
	outputPathInput = cleanWorkflowInput(outputPathInput)

	if pathInput != "" && outputPathInput != "" {
		resolvedPath := resolve(pathInput)
		resolvedOutput := resolve(outputPathInput)
		if resolvedPath != resolvedOutput {
			return "", coreerr.E(errorName, "path and output specify different locations", nil)
		}
		return resolvedPath, nil
	}

	if pathInput != "" {
		return resolve(pathInput), nil
	}

	if outputPathInput != "" {
		return resolve(outputPathInput), nil
	}

	return resolve(""), nil
}

// resolveReleaseWorkflowOutputAliasSet resolves a workflow output alias set by
// trimming whitespace, rejecting conflicts, and returning the first non-empty
// value when aliases agree.
func resolveReleaseWorkflowOutputAliasSet(
	outputPathInput,
	outputPathHyphenInput,
	outputPathSnakeInput,
	legacyOutputInput,
	workflowOutputPathInput,
	workflowOutputSnakeInput,
	workflowOutputHyphenInput,
	workflowOutputPathSnakeInput,
	workflowOutputPathHyphenInput,
	errorName string,
) (string, error) {
	values := []string{
		normalizeWorkflowOutputAlias(outputPathInput),
		normalizeWorkflowOutputAlias(outputPathHyphenInput),
		normalizeWorkflowOutputAlias(outputPathSnakeInput),
		normalizeWorkflowOutputAlias(legacyOutputInput),
		normalizeWorkflowOutputAlias(workflowOutputPathInput),
		normalizeWorkflowOutputAlias(workflowOutputSnakeInput),
		normalizeWorkflowOutputAlias(workflowOutputHyphenInput),
		normalizeWorkflowOutputAlias(workflowOutputPathSnakeInput),
		normalizeWorkflowOutputAlias(workflowOutputPathHyphenInput),
	}

	var resolved string
	for _, value := range values {
		if value == "" {
			continue
		}
		if resolved == "" {
			resolved = value
			continue
		}
		if resolved != value {
			return "", coreerr.E(errorName, "output aliases specify different locations", nil)
		}
	}

	return resolved, nil
}

// resolveReleaseWorkflowOutputAliasSetInProject resolves workflow output aliases
// against a project directory so relative and absolute paths can be compared.
func resolveReleaseWorkflowOutputAliasSetInProject(
	filesystem io_interface.Medium,
	projectDir,
	outputPathInput,
	outputPathHyphenInput,
	outputPathSnakeInput,
	legacyOutputInput,
	workflowOutputPathInput,
	workflowOutputSnakeInput,
	workflowOutputHyphenInput,
	workflowOutputPathSnakeInput,
	workflowOutputPathHyphenInput,
	errorName string,
) (string, error) {
	values := []string{
		cleanWorkflowInput(outputPathInput),
		cleanWorkflowInput(outputPathHyphenInput),
		cleanWorkflowInput(outputPathSnakeInput),
		cleanWorkflowInput(legacyOutputInput),
		cleanWorkflowInput(workflowOutputPathInput),
		cleanWorkflowInput(workflowOutputSnakeInput),
		cleanWorkflowInput(workflowOutputHyphenInput),
		cleanWorkflowInput(workflowOutputPathSnakeInput),
		cleanWorkflowInput(workflowOutputPathHyphenInput),
	}

	var resolved string
	for _, value := range values {
		if value == "" {
			continue
		}

		candidate := ResolveReleaseWorkflowOutputPathWithMedium(filesystem, projectDir, value)
		if resolved == "" {
			resolved = candidate
			continue
		}

		if resolved != candidate {
			return "", coreerr.E(errorName, "output aliases specify different locations", nil)
		}
	}

	return resolved, nil
}

// normalizeWorkflowOutputAlias canonicalises a workflow output alias for comparison.
func normalizeWorkflowOutputAlias(path string) string {
	path = cleanWorkflowInput(path)
	if path == "" {
		return ""
	}

	return ax.Clean(path)
}

// resolveReleaseWorkflowInputPath resolves one workflow input into a file path.
//
// resolveReleaseWorkflowInputPath("/tmp/project", "ci", io.Local) // /tmp/project/ci/release.yml
func resolveReleaseWorkflowInputPath(projectDir, input string, medium io_interface.Medium) string {
	input = cleanWorkflowInput(input)
	if input == "" {
		return ReleaseWorkflowPath(projectDir)
	}

	if isWorkflowDirectoryInput(input) {
		if ax.IsAbs(input) {
			return ax.Join(input, DefaultReleaseWorkflowFileName)
		}
		return ax.Join(projectDir, input, DefaultReleaseWorkflowFileName)
	}

	resolved := ResolveReleaseWorkflowPath(projectDir, input)
	if medium != nil && medium.IsDir(resolved) {
		return ax.Join(resolved, DefaultReleaseWorkflowFileName)
	}
	return resolved
}

// resolveReleaseWorkflowInputPathAliasSet resolves a workflow path from a set
// of aliases and rejects conflicting values.
func resolveReleaseWorkflowInputPathAliasSet(filesystem io_interface.Medium, projectDir, primaryLabel, primaryInput, secondaryInput, tertiaryInput, quaternaryInput, errorName string) (string, error) {
	values := []string{
		cleanWorkflowInput(primaryInput),
		cleanWorkflowInput(secondaryInput),
		cleanWorkflowInput(tertiaryInput),
		cleanWorkflowInput(quaternaryInput),
	}

	var resolved string
	for _, value := range values {
		if value == "" {
			continue
		}

		candidate := resolveReleaseWorkflowInputPath(projectDir, value, filesystem)
		if resolved == "" {
			resolved = candidate
			continue
		}

		if resolved != candidate {
			return "", coreerr.E(errorName, primaryLabel+" aliases specify different locations", nil)
		}
	}

	return resolved, nil
}

// isWorkflowDirectoryPath reports whether a workflow path is explicitly marked
// as a directory with a trailing separator.
func isWorkflowDirectoryPath(path string) bool {
	path = cleanWorkflowInput(path)
	if path == "" {
		return false
	}

	if path == "." || path == "./" || path == ".\\" {
		return true
	}

	last := path[len(path)-1]
	return last == '/' || last == '\\'
}

// isWorkflowDirectoryInput reports whether a workflow input should be treated
// as a directory target. This includes explicit directory paths and bare names
// without path separators or a file extension, plus current-directory-prefixed
// directory targets like "./ci" and the conventional ".github/workflows" path.
func isWorkflowDirectoryInput(path string) bool {
	path = cleanWorkflowInput(path)
	if isWorkflowDirectoryPath(path) {
		return true
	}
	if path == "" || ax.Ext(path) != "" {
		return false
	}
	if !strings.ContainsAny(path, "/\\") {
		return true
	}

	if ax.Base(path) == "workflows" {
		return true
	}

	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, ".\\") {
		trimmed := strings.TrimPrefix(strings.TrimPrefix(path, "./"), ".\\")
		if trimmed == "" {
			return false
		}
		if ax.Base(trimmed) == "workflows" {
			return true
		}
		return !strings.ContainsAny(trimmed, "/\\")
	}

	return false
}

// cleanWorkflowInput trims surrounding whitespace from a workflow path input.
func cleanWorkflowInput(path string) string {
	return strings.TrimSpace(path)
}
