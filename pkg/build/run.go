package build

import (
	"context"
	"io/fs"
	"reflect"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	coreio "dappco.re/go/io"
)

var defaultBuilderResolver BuilderResolver

// RunConfig captures the option-style inputs for the RFC-documented build API.
type RunConfig struct {
	Context        context.Context
	ProjectDir     string
	ConfigPath     string
	BuildConfig    *BuildConfig
	BuildType      string
	BuildTags      []string
	Obfuscate      bool
	ObfuscateSet   bool
	NSIS           bool
	NSISSet        bool
	WebView2       string
	WebView2Set    bool
	DenoBuild      string
	DenoBuildSet   bool
	NpmBuild       string
	NpmBuildSet    bool
	BuildCache     bool
	BuildCacheSet  bool
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

// WithBuildTags overrides the Go build tags passed through the pipeline.
func WithBuildTags(tags ...string) RunOption {
	return func(cfg *RunConfig) {
		cfg.BuildTags = append([]string(nil), tags...)
	}
}

// WithObfuscate enables or disables garble-backed obfuscation for the build.
func WithObfuscate(enabled bool) RunOption {
	return func(cfg *RunConfig) {
		cfg.Obfuscate = enabled
		cfg.ObfuscateSet = true
	}
}

// WithNSIS enables or disables Windows NSIS installer generation for Wails builds.
func WithNSIS(enabled bool) RunOption {
	return func(cfg *RunConfig) {
		cfg.NSIS = enabled
		cfg.NSISSet = true
	}
}

// WithWebView2 sets the Wails WebView2 delivery mode: download, embed, browser, or error.
func WithWebView2(mode string) RunOption {
	return func(cfg *RunConfig) {
		cfg.WebView2 = mode
		cfg.WebView2Set = true
	}
}

// WithDenoBuild overrides the default Deno frontend build command.
func WithDenoBuild(command string) RunOption {
	return func(cfg *RunConfig) {
		cfg.DenoBuild = command
		cfg.DenoBuildSet = true
	}
}

// WithNpmBuild overrides the default npm frontend build command.
func WithNpmBuild(command string) RunOption {
	return func(cfg *RunConfig) {
		cfg.NpmBuild = command
		cfg.NpmBuildSet = true
	}
}

// WithBuildCache enables or disables build cache setup before the pipeline runs.
func WithBuildCache(enabled bool) RunOption {
	return func(cfg *RunConfig) {
		cfg.BuildCache = enabled
		cfg.BuildCacheSet = true
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
//	result := build.Run(build.WithOutput(io.Local))
func Run(opts ...RunOption) core.Result {
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
		wd := ax.Getwd()
		if !wd.OK {
			return core.Fail(core.E("build.Run", "failed to get working directory", core.NewError(wd.Error())))
		}
		projectDir = wd.Value.(string)
	}
	projectDir = ax.Clean(projectDir)

	output := cfg.Output
	if output == nil {
		output = coreio.Local
	}

	destinationRoot := resolveRunOutputRoot(projectDir, cfg.OutputDir, output)

	stage := ax.MkdirTemp("core-build-*")
	if !stage.OK {
		return core.Fail(core.E("build.Run", "failed to create build staging directory", core.NewError(stage.Error())))
	}
	stageRoot := stage.Value.(string)
	defer ax.RemoveAll(stageRoot)

	stageOutputDir := ax.Join(stageRoot, "dist")

	resolver := cfg.ResolveBuilder
	if resolver == nil {
		resolver = DefaultBuilderResolver()
	}
	if resolver == nil {
		resolver = resolveBuiltinBuilder
	}

	pipeline := &Pipeline{
		FS:             coreio.Local,
		ResolveBuilder: resolver,
		ResolveVersion: cfg.ResolveVersion,
	}

	planResult := pipeline.Plan(ctx, PipelineRequest{
		ProjectDir:    projectDir,
		ConfigPath:    cfg.ConfigPath,
		BuildConfig:   cfg.BuildConfig,
		BuildType:     cfg.BuildType,
		BuildTags:     append([]string(nil), cfg.BuildTags...),
		Obfuscate:     cfg.Obfuscate,
		ObfuscateSet:  cfg.ObfuscateSet,
		NSIS:          cfg.NSIS,
		NSISSet:       cfg.NSISSet,
		WebView2:      cfg.WebView2,
		WebView2Set:   cfg.WebView2Set,
		DenoBuild:     cfg.DenoBuild,
		DenoBuildSet:  cfg.DenoBuildSet,
		NpmBuild:      cfg.NpmBuild,
		NpmBuildSet:   cfg.NpmBuildSet,
		BuildCache:    cfg.BuildCache,
		BuildCacheSet: cfg.BuildCacheSet,
		OutputDir:     stageOutputDir,
		BuildName:     cfg.BuildName,
		Targets:       append([]Target(nil), cfg.Targets...),
		Version:       cfg.Version,
	})
	if !planResult.OK {
		return planResult
	}
	plan := planResult.Value.(*PipelinePlan)

	result := pipeline.Run(ctx, plan)
	if !result.OK {
		return result
	}
	pipelineResult := result.Value.(*PipelineResult)

	return mirrorArtifacts(coreio.Local, output, stageOutputDir, destinationRoot, pipelineResult.Artifacts)
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

func mirrorArtifacts(source, destination coreio.Medium, sourceRoot, destinationRoot string, artifacts []Artifact) core.Result {
	if source == nil {
		source = coreio.Local
	}
	if destination == nil {
		destination = coreio.Local
	}

	mirrored := make([]Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		relativePathResult := ax.Rel(sourceRoot, artifact.Path)
		relativePath := ""
		if relativePathResult.OK {
			relativePath = relativePathResult.Value.(string)
		}
		if !relativePathResult.OK || relativePath == "" || core.HasPrefix(relativePath, "..") {
			relativePath = ax.Base(artifact.Path)
		}

		destinationPath := joinOutputPath(destinationRoot, relativePath)
		copied := copyMediumPath(source, artifact.Path, destination, destinationPath)
		if !copied.OK {
			return core.Fail(core.E("build.Run", "failed to mirror artifact "+artifact.Path, core.NewError(copied.Error())))
		}

		mirroredArtifact := artifact
		mirroredArtifact.Path = destinationPath
		mirrored = append(mirrored, mirroredArtifact)
	}

	return core.Ok(mirrored)
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

func copyMediumPath(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) core.Result {
	infoResult := source.Stat(sourcePath)
	if !infoResult.OK {
		return infoResult
	}
	info := infoResult.Value.(fs.FileInfo)

	if info.IsDir() {
		return copyMediumDir(source, sourcePath, destination, destinationPath)
	}

	return copyMediumFile(source, sourcePath, destination, destinationPath, info.Mode())
}

func copyMediumDir(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string) core.Result {
	created := destination.EnsureDir(destinationPath)
	if !created.OK {
		return created
	}

	entriesResult := source.List(sourcePath)
	if !entriesResult.OK {
		return entriesResult
	}
	entries := entriesResult.Value.([]fs.DirEntry)

	for _, entry := range entries {
		childSourcePath := ax.Join(sourcePath, entry.Name())
		childDestinationPath := ax.Join(destinationPath, entry.Name())
		copied := copyMediumPath(source, childSourcePath, destination, childDestinationPath)
		if !copied.OK {
			return copied
		}
	}

	return core.Ok(nil)
}

func copyMediumFile(source coreio.Medium, sourcePath string, destination coreio.Medium, destinationPath string, mode fs.FileMode) core.Result {
	created := destination.EnsureDir(ax.Dir(destinationPath))
	if !created.OK {
		return created
	}

	content := source.Read(sourcePath)
	if !content.OK {
		return content
	}

	return destination.WriteMode(destinationPath, content.Value.(string), mode)
}
