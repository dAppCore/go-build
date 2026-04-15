package build

import (
	"context"
	"runtime"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// BuilderResolver resolves a project type into a concrete builder.
//
//	resolver := func(projectType build.ProjectType) (build.Builder, error) { return builders.ResolveBuilder(projectType) }
type BuilderResolver func(ProjectType) (Builder, error)

// VersionResolver determines the build version for a project directory.
//
//	resolver := func(ctx context.Context, dir string) (string, error) { return release.DetermineVersionWithContext(ctx, dir) }
type VersionResolver func(context.Context, string) (string, error)

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
	BuildConfig   *BuildConfig
	ProjectType   ProjectType
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
//	plan, err := pipeline.Plan(ctx, build.PipelineRequest{ProjectDir: "."})
func (p *Pipeline) Plan(ctx context.Context, req PipelineRequest) (*PipelinePlan, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	filesystem := p.FS
	if filesystem == nil {
		filesystem = io.Local
	}

	projectDir := req.ProjectDir
	if projectDir == "" {
		var err error
		projectDir, err = ax.Getwd()
		if err != nil {
			return nil, coreerr.E("build.Pipeline.Plan", "failed to get working directory", err)
		}
	}
	projectDir = ax.Clean(projectDir)

	buildConfig, err := p.loadBuildConfig(filesystem, projectDir, req)
	if err != nil {
		return nil, err
	}
	buildConfig = CloneBuildConfig(buildConfig)
	applyPipelineBuildOverrides(buildConfig, req)

	if err := SetupBuildCache(filesystem, projectDir, buildConfig); err != nil {
		return nil, coreerr.E("build.Pipeline.Plan", "failed to set up build cache", err)
	}

	discovery, err := DiscoverFull(filesystem, projectDir)
	if err != nil {
		return nil, coreerr.E("build.Pipeline.Plan", "failed to inspect project", err)
	}

	options := ComputeOptions(buildConfig, discovery)
	setupPlan, err := ComputeSetupPlan(filesystem, projectDir, buildConfig, discovery)
	if err != nil {
		return nil, coreerr.E("build.Pipeline.Plan", "failed to compute setup plan", err)
	}

	projectType, err := resolvePipelineProjectType(filesystem, projectDir, req.BuildType, buildConfig)
	if err != nil {
		return nil, err
	}

	builder, err := p.resolveBuilder(projectType)
	if err != nil {
		return nil, err
	}

	targets := req.Targets
	if len(targets) == 0 {
		if len(buildConfig.Targets) > 0 {
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
		version, err = p.ResolveVersion(ctx, projectDir)
		if err != nil {
			return nil, coreerr.E("build.Pipeline.Plan", "failed to determine build version", err)
		}
	}

	runtimeCfg := RuntimeConfigFromBuildConfig(filesystem, projectDir, outputDir, buildName, buildConfig, req.Push, req.ImageName, version)
	ApplyOptions(runtimeCfg, options)

	return &PipelinePlan{
		ProjectDir:    projectDir,
		BuildConfig:   buildConfig,
		ProjectType:   projectType,
		Builder:       builder,
		Discovery:     discovery,
		Options:       options,
		SetupPlan:     setupPlan,
		Targets:       append([]Target(nil), targets...),
		OutputDir:     outputDir,
		BuildName:     buildName,
		Version:       version,
		RuntimeConfig: runtimeCfg,
	}, nil
}

// Run executes the builder for a precomputed plan.
//
//	result, err := pipeline.Run(ctx, plan)
func (p *Pipeline) Run(ctx context.Context, plan *PipelinePlan) (*PipelineResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if plan == nil {
		return nil, coreerr.E("build.Pipeline.Run", "pipeline plan is nil", nil)
	}
	if plan.Builder == nil {
		return nil, coreerr.E("build.Pipeline.Run", "pipeline plan is missing a builder", nil)
	}
	if plan.RuntimeConfig == nil {
		return nil, coreerr.E("build.Pipeline.Run", "pipeline plan is missing runtime config", nil)
	}

	artifacts, err := plan.Builder.Build(ctx, plan.RuntimeConfig, plan.Targets)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Plan:      plan,
		Artifacts: artifacts,
	}, nil
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

func (p *Pipeline) loadBuildConfig(filesystem io.Medium, projectDir string, req PipelineRequest) (*BuildConfig, error) {
	if req.BuildConfig != nil {
		return req.BuildConfig, nil
	}

	if req.ConfigPath == "" {
		cfg, err := LoadConfig(filesystem, projectDir)
		if err != nil {
			return nil, coreerr.E("build.Pipeline.Plan", "failed to load config", err)
		}
		return cfg, nil
	}

	configPath := req.ConfigPath
	if !ax.IsAbs(configPath) {
		configPath = ax.Join(projectDir, configPath)
	}
	if !filesystem.Exists(configPath) {
		return nil, coreerr.E("build.Pipeline.Plan", "build config not found: "+configPath, nil)
	}

	cfg, err := LoadConfigAtPath(filesystem, configPath)
	if err != nil {
		return nil, coreerr.E("build.Pipeline.Plan", "failed to load config", err)
	}
	return cfg, nil
}

func (p *Pipeline) resolveBuilder(projectType ProjectType) (Builder, error) {
	if p.ResolveBuilder == nil {
		return nil, coreerr.E("build.Pipeline.Plan", "builder resolver is required", nil)
	}

	builder, err := p.ResolveBuilder(projectType)
	if err != nil {
		return nil, coreerr.E("build.Pipeline.Plan", "failed to resolve builder for "+string(projectType), err)
	}
	if builder == nil {
		return nil, coreerr.E("build.Pipeline.Plan", "builder resolver returned nil for "+string(projectType), nil)
	}

	return builder, nil
}

func resolvePipelineProjectType(filesystem io.Medium, projectDir, buildType string, cfg *BuildConfig) (ProjectType, error) {
	if buildType != "" {
		return ProjectType(buildType), nil
	}
	if cfg != nil && cfg.Build.Type != "" {
		return ProjectType(cfg.Build.Type), nil
	}

	projectType, err := PrimaryType(filesystem, projectDir)
	if err != nil {
		return "", coreerr.E("build.Pipeline.Plan", "failed to detect project type", err)
	}
	if projectType == "" {
		switch {
		case IsDockerProject(filesystem, projectDir):
			projectType = ProjectTypeDocker
		case IsLinuxKitProject(filesystem, projectDir):
			projectType = ProjectTypeLinuxKit
		case IsCPPProject(filesystem, projectDir):
			projectType = ProjectTypeCPP
		case IsTaskfileProject(filesystem, projectDir):
			projectType = ProjectTypeTaskfile
		}
	}
	if projectType == "" {
		return "", coreerr.E("build.Pipeline.Plan", "no buildable project type found in "+projectDir, nil)
	}

	return projectType, nil
}

func applyPipelineBuildOverrides(cfg *BuildConfig, req PipelineRequest) {
	if cfg == nil {
		return
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
	if req.BuildCacheSet {
		if req.BuildCache {
			enableDefaultPipelineBuildCache(&cfg.Build.Cache)
		} else {
			cfg.Build.Cache.Enabled = false
		}
	}
}

func enableDefaultPipelineBuildCache(cfg *CacheConfig) {
	if cfg == nil {
		return
	}

	cfg.Enabled = true
	if cfg.Directory == "" {
		cfg.Directory = ax.Join(ConfigDir, "cache")
	}
	if len(cfg.Paths) == 0 {
		cfg.Paths = []string{
			ax.Join("cache", "go-build"),
			ax.Join("cache", "go-mod"),
		}
	}
}
