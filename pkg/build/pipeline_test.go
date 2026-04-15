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
	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			assert.Equal(t, ProjectTypeWails, projectType)
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
		BuildType:   string(ProjectTypeNode),
		Targets:     []Target{{OS: "darwin", Arch: "arm64"}},
	})
	require.NoError(t, err)

	assert.Equal(t, ProjectTypeNode, plan.ProjectType)
	assert.Equal(t, []Target{{OS: "darwin", Arch: "arm64"}}, plan.Targets)
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
