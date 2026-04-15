package build

import (
	"context"
	"os"
	"reflect"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

var defaultBuilderResolver BuilderResolver

// RunConfig captures the option-style inputs for the RFC-documented build API.
type RunConfig struct {
	Context        context.Context
	ProjectDir     string
	ConfigPath     string
	BuildConfig    *BuildConfig
	BuildType      string
	BuildName      string
	OutputDir      string
	Output         coreio.Medium
	Targets        []Target
	Version        string
	ResolveBuilder BuilderResolver
	ResolveVersion VersionResolver
}

// RunOption mutates a RunConfig before the pipeline executes.
type RunOption func(*RunConfig)

// RegisterDefaultBuilderResolver installs the builder resolver used by Run when
// the caller does not provide one explicitly.
func RegisterDefaultBuilderResolver(resolver BuilderResolver) {
	defaultBuilderResolver = resolver
}

// DefaultBuilderResolver returns the currently registered default builder resolver.
func DefaultBuilderResolver() BuilderResolver {
	return defaultBuilderResolver
}

// DefaultRunConfig returns the default configuration for the option-style Run API.
func DefaultRunConfig() *RunConfig {
	return &RunConfig{
		Context: context.Background(),
		Output:  coreio.Local,
	}
}

// WithContext overrides the context used for discovery, versioning, and builds.
func WithContext(ctx context.Context) RunOption {
	return func(cfg *RunConfig) {
		cfg.Context = ctx
	}
}

// WithProjectDir sets the project directory to build.
func WithProjectDir(dir string) RunOption {
	return func(cfg *RunConfig) {
		cfg.ProjectDir = dir
	}
}

// WithConfigPath points Run at an explicit build config file.
func WithConfigPath(path string) RunOption {
	return func(cfg *RunConfig) {
		cfg.ConfigPath = path
	}
}

// WithBuildConfig injects a preloaded build config instead of loading .core/build.yaml.
func WithBuildConfig(buildConfig *BuildConfig) RunOption {
	return func(cfg *RunConfig) {
		cfg.BuildConfig = buildConfig
	}
}

// WithBuildType forces a specific project type instead of auto-detection.
func WithBuildType(buildType string) RunOption {
	return func(cfg *RunConfig) {
		cfg.BuildType = buildType
	}
}

// WithBuildName overrides the resolved artifact name.
func WithBuildName(name string) RunOption {
	return func(cfg *RunConfig) {
		cfg.BuildName = name
	}
}

// WithOutputDir sets the destination directory or key prefix for mirrored artifacts.
func WithOutputDir(dir string) RunOption {
	return func(cfg *RunConfig) {
		cfg.OutputDir = dir
	}
}

// WithOutput sets the destination medium used for final build artifacts.
func WithOutput(output coreio.Medium) RunOption {
	return func(cfg *RunConfig) {
		cfg.Output = output
	}
}

// WithTargets overrides the build matrix targets.
func WithTargets(targets ...Target) RunOption {
	return func(cfg *RunConfig) {
		cfg.Targets = append([]Target(nil), targets...)
	}
}

// WithVersion overrides the resolved build version.
func WithVersion(version string) RunOption {
	return func(cfg *RunConfig) {
		cfg.Version = version
	}
}

// WithBuilderResolver provides an explicit builder resolver for Run.
func WithBuilderResolver(resolver BuilderResolver) RunOption {
	return func(cfg *RunConfig) {
		cfg.ResolveBuilder = resolver
	}
}

// WithVersionResolver provides an explicit version resolver for Run.
func WithVersionResolver(resolver VersionResolver) RunOption {
	return func(cfg *RunConfig) {
		cfg.ResolveVersion = resolver
	}
}

// Run executes the build pipeline and mirrors produced artifacts into the
// configured output medium.
//
//	artifacts, err := build.Run(build.WithOutput(io.Local))
func Run(opts ...RunOption) ([]Artifact, error) {
	cfg := DefaultRunConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}

	projectDir := cfg.ProjectDir
	if projectDir == "" {
		var err error
		projectDir, err = ax.Getwd()
		if err != nil {
			return nil, coreerr.E("build.Run", "failed to get working directory", err)
		}
	}
	projectDir = ax.Clean(projectDir)

	output := cfg.Output
	if output == nil {
		output = coreio.Local
	}

	destinationRoot := resolveRunOutputRoot(projectDir, cfg.OutputDir, output)

	stageRoot, err := os.MkdirTemp("", "core-build-*")
	if err != nil {
		return nil, coreerr.E("build.Run", "failed to create build staging directory", err)
	}
	defer func() { _ = os.RemoveAll(stageRoot) }()

	stageOutputDir := ax.Join(stageRoot, "dist")

	resolver := cfg.ResolveBuilder
	if resolver == nil {
		resolver = DefaultBuilderResolver()
	}
	if resolver == nil {
		return nil, coreerr.E("build.Run", "builder resolver is required; import pkg/build/builders or use WithBuilderResolver", nil)
	}

	pipeline := &Pipeline{
		FS:             coreio.Local,
		ResolveBuilder: resolver,
		ResolveVersion: cfg.ResolveVersion,
	}

	plan, err := pipeline.Plan(ctx, PipelineRequest{
		ProjectDir:  projectDir,
		ConfigPath:  cfg.ConfigPath,
		BuildConfig: cfg.BuildConfig,
		BuildType:   cfg.BuildType,
		OutputDir:   stageOutputDir,
		BuildName:   cfg.BuildName,
		Targets:     append([]Target(nil), cfg.Targets...),
		Version:     cfg.Version,
	})
	if err != nil {
		return nil, err
	}

	result, err := pipeline.Run(ctx, plan)
	if err != nil {
		return nil, err
	}

	return mirrorArtifacts(coreio.Local, output, stageOutputDir, destinationRoot, result.Artifacts)
}

func resolveRunOutputRoot(projectDir, outputDir string, output coreio.Medium) string {
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

func mirrorArtifacts(source, destination coreio.Medium, sourceRoot, destinationRoot string, artifacts []Artifact) ([]Artifact, error) {
	if source == nil {
		source = coreio.Local
	}
	if destination == nil {
		destination = coreio.Local
	}

	mirrored := make([]Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		relativePath, err := ax.Rel(sourceRoot, artifact.Path)
		if err != nil || relativePath == "" || core.HasPrefix(relativePath, "..") {
			relativePath = ax.Base(artifact.Path)
		}

		destinationPath := joinOutputPath(destinationRoot, relativePath)
		if err := copyMediumPath(source, artifact.Path, destination, destinationPath); err != nil {
			return nil, coreerr.E("build.Run", "failed to mirror artifact "+artifact.Path, err)
		}

		mirroredArtifact := artifact
		mirroredArtifact.Path = destinationPath
		mirrored = append(mirrored, mirroredArtifact)
	}

	return mirrored, nil
}

func joinOutputPath(root, path string) string {
	if root == "" || root == "." {
		return ax.Clean(path)
	}
	if path == "" || path == "." {
		return ax.Clean(root)
	}
	return ax.Join(root, path)
}

func copyMediumPath(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) error {
	info, err := source.Stat(sourcePath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyMediumDir(source, sourcePath, destination, destinationPath)
	}

	return copyMediumFile(source, sourcePath, destination, destinationPath, info.Mode())
}

func copyMediumDir(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) error {
	if err := destination.EnsureDir(destinationPath); err != nil {
		return err
	}

	entries, err := source.List(sourcePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childSourcePath := ax.Join(sourcePath, entry.Name())
		childDestinationPath := ax.Join(destinationPath, entry.Name())
		if err := copyMediumPath(source, childSourcePath, destination, childDestinationPath); err != nil {
			return err
		}
	}

	return nil
}

func copyMediumFile(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string, mode os.FileMode) error {
	if err := destination.EnsureDir(ax.Dir(destinationPath)); err != nil {
		return err
	}

	content, err := source.Read(sourcePath)
	if err != nil {
		return err
	}

	return destination.WriteMode(destinationPath, content, mode)
}
