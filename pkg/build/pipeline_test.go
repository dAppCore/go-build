package build

import (
	"context"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPipelineBuilder struct {
	artifacts []Artifact
	lastCfg   *Config
	lastTgts  []Target
}

func (b *stubPipelineBuilder) Name() string { return "stub" }

func (b *stubPipelineBuilder) Detect(fs io.Medium, dir string) (bool, error) {
	return true, nil
}

func (b *stubPipelineBuilder) Build(ctx context.Context, cfg *Config, targets []Target) ([]Artifact, error) {
	b.lastCfg = cfg
	b.lastTgts = append([]Target(nil), targets...)
	return append([]Artifact(nil), b.artifacts...), nil
}

func TestPipeline_Plan_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))

	cfg := DefaultConfig()
	cfg.Project.Binary = "core-demo"
	cfg.Build.Obfuscate = true
	cfg.Build.NSIS = true
	cfg.Build.WebView2 = "embed"
	cfg.Build.BuildTags = []string{"integration"}
	cfg.Targets = []TargetConfig{{OS: "linux", Arch: "amd64"}}

	builder := &stubPipelineBuilder{}
	var resolvedTypes []ProjectType
	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			resolvedTypes = append(resolvedTypes, projectType)
			return builder, nil
		},
		ResolveVersion: func(ctx context.Context, projectDir string) (string, error) {
			assert.Equal(t, dir, projectDir)
			return "v1.2.3", nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: cfg,
		OutputDir:   "artifacts",
	})
	require.NoError(t, err)

	assert.Equal(t, dir, plan.ProjectDir)
	assert.Equal(t, ProjectTypeWails, plan.ProjectType)
	assert.Equal(t, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, plan.ProjectTypes)
	assert.Equal(t, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, resolvedTypes)
	assert.Equal(t, "core-demo", plan.BuildName)
	assert.Equal(t, ax.Join(dir, "artifacts"), plan.OutputDir)
	assert.Equal(t, "v1.2.3", plan.Version)
	assert.NotNil(t, plan.Discovery)
	assert.Equal(t, "wails2", plan.SetupPlan.PrimaryStackSuggestion)
	assert.Equal(t, []Target{{OS: "linux", Arch: "amd64"}}, plan.Targets)
	assert.True(t, plan.Options.Obfuscate)
	assert.True(t, plan.Options.NSIS)
	assert.Equal(t, "embed", plan.Options.WebView2)
	assert.Contains(t, plan.Options.Tags, "integration")
	assert.Equal(t, "core-demo", plan.RuntimeConfig.Name)
	assert.Equal(t, plan.OutputDir, plan.RuntimeConfig.OutputDir)
	assert.Equal(t, "v1.2.3", plan.RuntimeConfig.Version)
	assert.True(t, plan.RuntimeConfig.Obfuscate)
	assert.True(t, plan.RuntimeConfig.NSIS)
	assert.Equal(t, "embed", plan.RuntimeConfig.WebView2)
	assert.Contains(t, plan.RuntimeConfig.BuildTags, "integration")
}

func TestPipeline_Plan_UsesExplicitBuildTypeOverride_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644))

	cfg := DefaultConfig()
	cfg.Build.Type = "go"

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			assert.Equal(t, ProjectTypeNode, projectType)
			return &stubPipelineBuilder{}, nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: cfg,
		BuildType:   "NoDe",
		Targets:     []Target{{OS: "darwin", Arch: "arm64"}},
	})
	require.NoError(t, err)

	assert.Equal(t, ProjectTypeNode, plan.ProjectType)
	assert.Equal(t, "node", plan.BuildConfig.Build.Type)
	assert.Equal(t, "node", plan.SetupPlan.PrimaryStack)
	assert.Equal(t, "node", plan.SetupPlan.PrimaryStackSuggestion)
	assert.Contains(t, setupTools(plan.SetupPlan), SetupToolNode)
	assert.Equal(t, []Target{{OS: "darwin", Arch: "arm64"}}, plan.Targets)
}

func TestPipeline_Plan_NormalisesConfiguredBuildType_Good(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Build.Type = "WaIlS"

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			assert.Equal(t, ProjectTypeWails, projectType)
			return &stubPipelineBuilder{}, nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  t.TempDir(),
		BuildConfig: cfg,
		Targets:     []Target{{OS: "darwin", Arch: "arm64"}},
	})
	require.NoError(t, err)

	assert.Equal(t, ProjectTypeWails, plan.ProjectType)
	assert.Equal(t, "wails", plan.BuildConfig.Build.Type)
	assert.Equal(t, "wails2", plan.SetupPlan.PrimaryStackSuggestion)
	assert.Contains(t, setupTools(plan.SetupPlan), SetupToolWails)
	assert.Contains(t, setupTools(plan.SetupPlan), SetupToolNode)
}

func TestPipeline_Plan_AppliesActionStyleOverrides_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))

	cfg := DefaultConfig()
	cfg.Build.BuildTags = []string{"integration"}

	var resolvedTypes []ProjectType
	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			resolvedTypes = append(resolvedTypes, projectType)
			return &stubPipelineBuilder{}, nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:    dir,
		BuildConfig:   cfg,
		BuildTags:     []string{"mlx", "release", "mlx"},
		Obfuscate:     true,
		ObfuscateSet:  true,
		NSIS:          true,
		NSISSet:       true,
		WebView2:      "download",
		WebView2Set:   true,
		DenoBuild:     "deno task bundle",
		DenoBuildSet:  true,
		BuildCache:    true,
		BuildCacheSet: true,
	})
	require.NoError(t, err)

	assert.Contains(t, plan.Options.Tags, "mlx")
	assert.Contains(t, plan.Options.Tags, "release")
	assert.NotContains(t, plan.Options.Tags, "integration")
	assert.True(t, plan.Options.Obfuscate)
	assert.True(t, plan.Options.NSIS)
	assert.Equal(t, "download", plan.Options.WebView2)
	assert.Equal(t, "deno task bundle", plan.BuildConfig.Build.DenoBuild)
	assert.True(t, plan.BuildConfig.Build.Cache.Enabled)
	assert.Equal(t, ax.Join(dir, ".core", "cache"), plan.BuildConfig.Build.Cache.Directory)
	assert.Equal(t, []string{
		ax.Join(dir, "cache", "go-build"),
		ax.Join(dir, "cache", "go-mod"),
	}, plan.BuildConfig.Build.Cache.Paths)
	assert.True(t, plan.RuntimeConfig.Cache.Enabled)
	assert.Equal(t, plan.BuildConfig.Build.Cache.Directory, plan.RuntimeConfig.Cache.Directory)
	assert.Equal(t, plan.BuildConfig.Build.Cache.Paths, plan.RuntimeConfig.Cache.Paths)
	assert.Contains(t, setupTools(plan.SetupPlan), SetupToolDeno)
	assert.Equal(t, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, plan.ProjectTypes)
	assert.Equal(t, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, resolvedTypes)
}

func TestPipeline_Plan_UsesExplicitVersionOverride_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644))

	versionResolverCalled := false
	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			assert.Equal(t, ProjectTypeGo, projectType)
			return &stubPipelineBuilder{}, nil
		},
		ResolveVersion: func(ctx context.Context, projectDir string) (string, error) {
			versionResolverCalled = true
			return "v0.0.1", nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: DefaultConfig(),
		Version:     "v9.9.9",
		Targets:     []Target{{OS: "linux", Arch: "amd64"}},
	})
	require.NoError(t, err)

	assert.Equal(t, "v9.9.9", plan.Version)
	assert.Equal(t, "v9.9.9", plan.RuntimeConfig.Version)
	assert.False(t, versionResolverCalled)
}

func TestPipeline_Plan_DoesNotMutateCallerBuildConfig_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))

	cfg := DefaultConfig()
	cfg.Build.BuildTags = []string{"integration"}

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			return &stubPipelineBuilder{}, nil
		},
	}

	_, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:    dir,
		BuildConfig:   cfg,
		BuildTags:     []string{"mlx"},
		Obfuscate:     true,
		ObfuscateSet:  true,
		DenoBuild:     "deno task bundle",
		DenoBuildSet:  true,
		BuildCache:    true,
		BuildCacheSet: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"integration"}, cfg.Build.BuildTags)
	assert.False(t, cfg.Build.Obfuscate)
	assert.Empty(t, cfg.Build.DenoBuild)
	assert.False(t, cfg.Build.Cache.Enabled)
	assert.Empty(t, cfg.Build.Cache.Directory)
	assert.Empty(t, cfg.Build.Cache.Paths)
}

func TestPipeline_Run_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644))

	builder := &stubPipelineBuilder{
		artifacts: []Artifact{{Path: ax.Join(dir, "dist", "demo"), OS: "linux", Arch: "amd64"}},
	}

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			return builder, nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: DefaultConfig(),
		Targets:     []Target{{OS: "linux", Arch: "amd64"}},
	})
	require.NoError(t, err)

	result, err := pipeline.Run(context.Background(), plan)
	require.NoError(t, err)

	assert.Equal(t, plan, result.Plan)
	assert.Equal(t, []Artifact{{Path: ax.Join(dir, "dist", "demo"), OS: "linux", Arch: "amd64"}}, result.Artifacts)
	require.NotNil(t, builder.lastCfg)
	assert.Equal(t, plan.RuntimeConfig, builder.lastCfg)
	assert.Equal(t, plan.Targets, builder.lastTgts)
}

func TestPipeline_Run_MultiType_Good(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo\n"), 0o644))

	nodeBuilder := &stubPipelineBuilder{
		artifacts: []Artifact{{Path: ax.Join(dir, "dist", "node", "linux_amd64", "node-artifact"), OS: "linux", Arch: "amd64"}},
	}
	docsBuilder := &stubPipelineBuilder{
		artifacts: []Artifact{{Path: ax.Join(dir, "dist", "docs", "linux_amd64", "docs-artifact"), OS: "linux", Arch: "amd64"}},
	}

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			switch projectType {
			case ProjectTypeNode:
				return nodeBuilder, nil
			case ProjectTypeDocs:
				return docsBuilder, nil
			default:
				return nil, assert.AnError
			}
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: DefaultConfig(),
		Targets:     []Target{{OS: "linux", Arch: "amd64"}},
	})
	require.NoError(t, err)
	assert.Equal(t, []ProjectType{ProjectTypeNode, ProjectTypeDocs}, plan.ProjectTypes)

	result, err := pipeline.Run(context.Background(), plan)
	require.NoError(t, err)

	assert.Len(t, result.Artifacts, 2)
	require.NotNil(t, nodeBuilder.lastCfg)
	require.NotNil(t, docsBuilder.lastCfg)
	assert.Equal(t, ax.Join(plan.OutputDir, "node"), nodeBuilder.lastCfg.OutputDir)
	assert.Equal(t, ax.Join(plan.OutputDir, "docs"), docsBuilder.lastCfg.OutputDir)
	assert.Equal(t, plan.Targets, nodeBuilder.lastTgts)
	assert.Equal(t, plan.Targets, docsBuilder.lastTgts)
	assert.NotSame(t, plan.RuntimeConfig, nodeBuilder.lastCfg)
	assert.NotSame(t, plan.RuntimeConfig, docsBuilder.lastCfg)
}

func TestPipeline_Plan_Bad(t *testing.T) {
	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			return &stubPipelineBuilder{}, nil
		},
	}

	_, err := pipeline.Plan(context.Background(), PipelineRequest{ProjectDir: t.TempDir()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no buildable project type found")
}

func TestPipeline_Run_Bad(t *testing.T) {
	pipeline := &Pipeline{}

	_, err := pipeline.Run(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline plan is nil")
}
