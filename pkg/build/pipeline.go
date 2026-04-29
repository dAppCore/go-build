package build

import (
	"context"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
)

// BuilderResolver resolves a project type into a concrete builder.
//
//	resolver := func(projectType build.ProjectType) core.Result { return builders.ResolveBuilder(projectType) }
type BuilderResolver func(ProjectType) core.Result

// VersionResolver determines the build version for a project directory.
//
//	resolver := func(ctx context.Context, dir string) core.Result { return release.DetermineVersionWithContext(ctx, dir) }
type VersionResolver func(context.Context, string) core.Result

// Pipeline coordinates the action-style gateway phases for a build request:
// discovery, option computation, setup planning, builder resolution, and build.
//
//	pipeline := &build.Pipeline{FS: io.Local, ResolveBuilder: resolver}
type Pipeline struct {
	FS             io.Medium
	ResolveBuilder BuilderResolver
	ResolveVersion VersionResolver
}

// PipelineRequest captures the inputs required to plan or run a build.
type PipelineRequest struct {
	ProjectDir    string
	ConfigPath    string
	BuildConfig   *BuildConfig
	BuildType     string
	BuildTags     []string
	Obfuscate     bool
	ObfuscateSet  bool
	NSIS          bool
	NSISSet       bool
	WebView2      string
	WebView2Set   bool
	DenoBuild     string
	DenoBuildSet  bool
	NpmBuild      string
	NpmBuildSet   bool
	BuildCache    bool
	BuildCacheSet bool
	OutputDir     string
	BuildName     string
	Targets       []Target
	Push          bool
	ImageName     string
	Version       string
}

// PipelinePlan is the fully resolved gateway state before the builder runs.
type PipelinePlan struct {
	ProjectDir    string
	ProjectTypes  []ProjectType
	BuildConfig   *BuildConfig
	ProjectType   ProjectType
	Builders      []Builder
	Builder       Builder
	Discovery     *DiscoveryResult
	Options       *BuildOptions
	SetupPlan     *SetupPlan
	Targets       []Target
	OutputDir     string
	BuildName     string
	Version       string
	RuntimeConfig *Config
}

// PipelineResult contains the executed plan and the produced artifacts.
type PipelineResult struct {
	Plan      *PipelinePlan
	Artifacts []Artifact
}

// Plan resolves the action-style gateway phases without executing the builder.
//
//	result := pipeline.Plan(ctx, build.PipelineRequest{ProjectDir: "."})
func (p *Pipeline) Plan(ctx context.Context, req PipelineRequest) core.Result {
	if ctx == nil {
		ctx = context.Background()
	}

	filesystem := p.FS
	if filesystem == nil {
		filesystem = io.Local
	}

	projectDir := req.ProjectDir
	if projectDir == "" {
		wd := ax.Getwd()
		if !wd.OK {
			return core.Fail(core.E("build.Pipeline.Plan", "failed to get working directory", core.NewError(wd.Error())))
		}
		projectDir = wd.Value.(string)
	}
	projectDir = ax.Clean(projectDir)

	buildConfigResult := p.loadBuildConfig(filesystem, projectDir, req)
	if !buildConfigResult.OK {
		return buildConfigResult
	}
	buildConfig := buildConfigResult.Value.(*BuildConfig)
	buildConfig = CloneBuildConfig(buildConfig)
	applyPipelineBuildOverrides(buildConfig, req)

	cacheSetup := SetupBuildCache(filesystem, projectDir, buildConfig)
	if !cacheSetup.OK {
		return core.Fail(core.E("build.Pipeline.Plan", "failed to set up build cache", core.NewError(cacheSetup.Error())))
	}

	discoveryResult := DiscoverFull(filesystem, projectDir)
	if !discoveryResult.OK {
		return core.Fail(core.E("build.Pipeline.Plan", "failed to inspect project", core.NewError(discoveryResult.Error())))
	}
	discovery := discoveryResult.Value.(*DiscoveryResult)

	options := ComputeOptions(buildConfig, discovery)
	setupPlanResult := ComputeSetupPlan(filesystem, projectDir, buildConfig, discovery)
	if !setupPlanResult.OK {
		return core.Fail(core.E("build.Pipeline.Plan", "failed to compute setup plan", core.NewError(setupPlanResult.Error())))
	}
	setupPlan := setupPlanResult.Value.(*SetupPlan)

	projectTypesResult := resolvePipelineProjectTypes(filesystem, projectDir, req.BuildType, buildConfig)
	if !projectTypesResult.OK {
		return projectTypesResult
	}
	projectTypes := projectTypesResult.Value.([]ProjectType)

	builders := make([]Builder, 0, len(projectTypes))
	for _, projectType := range projectTypes {
		builderResult := p.resolveBuilder(projectType)
		if !builderResult.OK {
			return builderResult
		}
		builder := builderResult.Value.(Builder)
		builders = append(builders, builder)
	}

	targets := req.Targets
	if len(targets) == 0 {
		if shouldUseLocalTargetByDefault(filesystem, projectDir, req) {
			targets = []Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
		} else if len(buildConfig.Targets) > 0 {
			targets = buildConfig.ToTargets()
		} else {
			targets = []Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
		}
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = "dist"
	}
	if !ax.IsAbs(outputDir) {
		outputDir = ax.Join(projectDir, outputDir)
	}
	outputDir = ax.Clean(outputDir)

	buildName := ResolveBuildName(projectDir, buildConfig, req.BuildName)

	version := req.Version
	if version == "" && p.ResolveVersion != nil {
		versionResult := p.ResolveVersion(ctx, projectDir)
		if !versionResult.OK {
			return core.Fail(core.E("build.Pipeline.Plan", "failed to determine build version", core.NewError(versionResult.Error())))
		}
		version = versionResult.Value.(string)
	}
	if version != "" {
		valid := ValidateVersionString(version)
		if !valid.OK {
			return core.Fail(core.E("build.Pipeline.Plan", "invalid build version override", core.NewError(valid.Error())))
		}
	}

	runtimeCfg := RuntimeConfigFromBuildConfig(filesystem, projectDir, outputDir, buildName, buildConfig, req.Push, req.ImageName, version)
	ApplyOptions(runtimeCfg, options)

	return core.Ok(&PipelinePlan{
		ProjectDir:    projectDir,
		ProjectTypes:  append([]ProjectType(nil), projectTypes...),
		BuildConfig:   buildConfig,
		ProjectType:   projectTypes[0],
		Builders:      builders,
		Builder:       builders[0],
		Discovery:     discovery,
		Options:       options,
		SetupPlan:     setupPlan,
		Targets:       append([]Target(nil), targets...),
		OutputDir:     outputDir,
		BuildName:     buildName,
		Version:       version,
		RuntimeConfig: runtimeCfg,
	})
}

// Run executes the builder for a precomputed plan.
//
//	result := pipeline.Run(ctx, plan)
func (p *Pipeline) Run(ctx context.Context, plan *PipelinePlan) core.Result {
	if ctx == nil {
		ctx = context.Background()
	}
	if plan == nil {
		return core.Fail(core.E("build.Pipeline.Run", "pipeline plan is nil", nil))
	}
	if plan.RuntimeConfig == nil {
		return core.Fail(core.E("build.Pipeline.Run", "pipeline plan is missing runtime config", nil))
	}

	builders := append([]Builder(nil), plan.Builders...)
	projectTypes := append([]ProjectType(nil), plan.ProjectTypes...)
	if len(builders) == 0 {
		if plan.Builder == nil {
			return core.Fail(core.E("build.Pipeline.Run", "pipeline plan is missing a builder", nil))
		}
		builders = []Builder{plan.Builder}
		if len(projectTypes) == 0 && plan.ProjectType != "" {
			projectTypes = []ProjectType{plan.ProjectType}
		}
	}
	if len(projectTypes) == 0 {
		return core.Fail(core.E("build.Pipeline.Run", "pipeline plan is missing project types", nil))
	}

	artifacts := make([]Artifact, 0, len(builders))
	multiType := len(builders) > 1
	for i, builder := range builders {
		if builder == nil {
			return core.Fail(core.E("build.Pipeline.Run", "pipeline plan contains a nil builder", nil))
		}

		runtimeCfg := plan.RuntimeConfig
		if multiType {
			runtimeCfg = cloneRuntimeConfig(plan.RuntimeConfig)
			runtimeCfg.OutputDir = multiTypeOutputDir(plan.OutputDir, projectTypes, i)
		}

		builtArtifacts := builder.Build(ctx, runtimeCfg, plan.Targets)
		if !builtArtifacts.OK {
			return builtArtifacts
		}
		artifacts = append(artifacts, builtArtifacts.Value.([]Artifact)...)
	}

	return core.Ok(&PipelineResult{
		Plan:      plan,
		Artifacts: artifacts,
	})
}

// ResolveBuildName resolves the output name from an explicit override, config,
// or the project directory name.
//
//	name := build.ResolveBuildName("/tmp/project", cfg, "")
func ResolveBuildName(projectDir string, cfg *BuildConfig, override string) string {
	if override != "" {
		return override
	}
	if cfg != nil {
		if cfg.Project.Binary != "" {
			return cfg.Project.Binary
		}
		if cfg.Project.Name != "" {
			return cfg.Project.Name
		}
	}
	return ax.Base(projectDir)
}

func (p *Pipeline) loadBuildConfig(filesystem io.Medium, projectDir string, req PipelineRequest) core.Result {
	if req.BuildConfig != nil {
		return core.Ok(req.BuildConfig)
	}

	if req.ConfigPath == "" {
		cfg := LoadConfig(filesystem, projectDir)
		if !cfg.OK {
			return core.Fail(core.E("build.Pipeline.Plan", "failed to load config", core.NewError(cfg.Error())))
		}
		return cfg
	}

	configPath := req.ConfigPath
	if !ax.IsAbs(configPath) {
		configPath = ax.Join(projectDir, configPath)
	}
	if !filesystem.Exists(configPath) {
		return core.Fail(core.E("build.Pipeline.Plan", "build config not found: "+configPath, nil))
	}

	cfg := LoadConfigAtPath(filesystem, configPath)
	if !cfg.OK {
		return core.Fail(core.E("build.Pipeline.Plan", "failed to load config", core.NewError(cfg.Error())))
	}
	return cfg
}

func (p *Pipeline) resolveBuilder(projectType ProjectType) core.Result {
	if p.ResolveBuilder == nil {
		return core.Fail(core.E("build.Pipeline.Plan", "builder resolver is required", nil))
	}

	builderResult := p.ResolveBuilder(projectType)
	if !builderResult.OK {
		return core.Fail(core.E("build.Pipeline.Plan", "failed to resolve builder for "+string(projectType), core.NewError(builderResult.Error())))
	}
	builder := builderResult.Value.(Builder)
	if builder == nil {
		return core.Fail(core.E("build.Pipeline.Plan", "builder resolver returned nil for "+string(projectType), nil))
	}

	return core.Ok(builder)
}

func resolvePipelineProjectTypes(filesystem io.Medium, projectDir, buildType string, cfg *BuildConfig) core.Result {
	if value := normalisePipelineBuildType(buildType); value != "" {
		return core.Ok([]ProjectType{ProjectType(value)})
	}
	if cfg != nil {
		if value := normalisePipelineBuildType(cfg.Build.Type); value != "" {
			return core.Ok([]ProjectType{ProjectType(value)})
		}
	}

	projectTypesResult := Discover(filesystem, projectDir)
	if !projectTypesResult.OK {
		return core.Fail(core.E("build.Pipeline.Plan", "failed to detect project type", core.NewError(projectTypesResult.Error())))
	}
	projectTypes := projectTypesResult.Value.([]ProjectType)
	if len(projectTypes) == 0 {
		return core.Fail(core.E("build.Pipeline.Plan", "no buildable project type found in "+projectDir, nil))
	}

	return projectTypesResult
}

func shouldUseLocalTargetByDefault(filesystem io.Medium, projectDir string, req PipelineRequest) bool {
	if req.BuildConfig != nil || req.ConfigPath != "" {
		return false
	}

	return !ConfigExists(filesystem, projectDir)
}

func applyPipelineBuildOverrides(cfg *BuildConfig, req PipelineRequest) {
	if cfg == nil {
		return
	}

	if cfg.Build.Type != "" {
		cfg.Build.Type = normalisePipelineBuildType(cfg.Build.Type)
	}
	if buildType := normalisePipelineBuildType(req.BuildType); buildType != "" {
		cfg.Build.Type = buildType
	}
	if len(req.BuildTags) > 0 {
		cfg.Build.BuildTags = deduplicateTags(append([]string(nil), req.BuildTags...))
	}
	if req.ObfuscateSet {
		cfg.Build.Obfuscate = req.Obfuscate
	}
	if req.NSISSet {
		cfg.Build.NSIS = req.NSIS
	}
	if req.WebView2Set {
		cfg.Build.WebView2 = req.WebView2
	}
	if req.DenoBuildSet {
		cfg.Build.DenoBuild = req.DenoBuild
	}
	if req.NpmBuildSet {
		cfg.Build.NpmBuild = req.NpmBuild
	}
	if req.BuildCacheSet {
		if req.BuildCache {
			enableDefaultPipelineBuildCache(&cfg.Build.Cache)
		} else {
			cfg.Build.Cache.Enabled = false
		}
	}
}

func cloneRuntimeConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	clone := *cfg
	clone.LDFlags = append([]string(nil), cfg.LDFlags...)
	clone.Flags = append([]string(nil), cfg.Flags...)
	clone.BuildTags = append([]string(nil), cfg.BuildTags...)
	clone.Env = append([]string(nil), cfg.Env...)
	clone.Cache = cloneCacheConfig(cfg.Cache)
	clone.Tags = append([]string(nil), cfg.Tags...)
	clone.BuildArgs = CloneStringMap(cfg.BuildArgs)
	clone.Formats = append([]string(nil), cfg.Formats...)
	clone.LinuxKit = cloneLinuxKitConfig(cfg.LinuxKit)
	return &clone
}

func multiTypeOutputDir(root string, projectTypes []ProjectType, index int) string {
	if root == "" || index < 0 || index >= len(projectTypes) || projectTypes[index] == "" {
		return root
	}
	return ax.Join(root, string(projectTypes[index]))
}

func enableDefaultPipelineBuildCache(cfg *CacheConfig) {
	if cfg == nil {
		return
	}

	cfg.Enabled = true
	if cfg.Dir == "" && cfg.Directory == "" {
		cfg.Dir = ax.Join(ConfigDir, "cache")
	}
	if cfg.Dir == "" {
		cfg.Dir = cfg.Directory
	}
	if cfg.Directory == "" {
		cfg.Directory = cfg.Dir
	}
	if len(cfg.Paths) == 0 {
		cfg.Paths = DefaultBuildCachePaths("")
	}
}

func normalisePipelineBuildType(value string) string {
	return core.Lower(core.Trim(value))
}
