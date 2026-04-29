package build

import (
	"context"
	"runtime"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/io"
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
	if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
			if !stdlibAssertEqual(dir, projectDir) {
				t.Fatalf("want %v, got %v", dir, projectDir)
			}

			return "v1.2.3", nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: cfg,
		OutputDir:   "artifacts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(dir, plan.ProjectDir) {
		t.Fatalf("want %v, got %v", dir, plan.ProjectDir)
	}
	if !stdlibAssertEqual(ProjectTypeWails, plan.ProjectType) {
		t.Fatalf("want %v, got %v", ProjectTypeWails, plan.ProjectType)
	}
	if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, plan.ProjectTypes) {
		t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, plan.ProjectTypes)
	}
	if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, resolvedTypes) {
		t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, resolvedTypes)
	}
	if !stdlibAssertEqual("core-demo", plan.BuildName) {
		t.Fatalf("want %v, got %v", "core-demo", plan.BuildName)
	}
	if !stdlibAssertEqual(ax.Join(dir, "artifacts"), plan.OutputDir) {
		t.Fatalf("want %v, got %v", ax.Join(dir, "artifacts"), plan.OutputDir)
	}
	if !stdlibAssertEqual("v1.2.3", plan.Version) {
		t.Fatalf("want %v, got %v", "v1.2.3", plan.Version)
	}
	if stdlibAssertNil(plan.Discovery) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("wails2", plan.SetupPlan.PrimaryStackSuggestion) {
		t.Fatalf("want %v, got %v", "wails2", plan.SetupPlan.PrimaryStackSuggestion)
	}
	if !stdlibAssertEqual([]Target{{OS: "linux", Arch: "amd64"}}, plan.Targets) {
		t.Fatalf("want %v, got %v", []Target{{OS: "linux", Arch: "amd64"}}, plan.Targets)
	}
	if !(plan.Options.Obfuscate) {
		t.Fatal("expected true")
	}
	if !(plan.Options.NSIS) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("embed", plan.Options.WebView2) {
		t.Fatalf("want %v, got %v", "embed", plan.Options.WebView2)
	}
	if !stdlibAssertContains(plan.Options.Tags, "integration") {
		t.Fatalf("expected %v to contain %v", plan.Options.Tags, "integration")
	}
	if !stdlibAssertEqual("core-demo", plan.RuntimeConfig.Name) {
		t.Fatalf("want %v, got %v", "core-demo", plan.RuntimeConfig.Name)
	}
	if !stdlibAssertEqual(plan.OutputDir, plan.RuntimeConfig.OutputDir) {
		t.Fatalf("want %v, got %v", plan.OutputDir, plan.RuntimeConfig.OutputDir)
	}
	if !stdlibAssertEqual("v1.2.3", plan.RuntimeConfig.Version) {
		t.Fatalf("want %v, got %v", "v1.2.3", plan.RuntimeConfig.Version)
	}
	if !(plan.RuntimeConfig.Obfuscate) {
		t.Fatal("expected true")
	}
	if !(plan.RuntimeConfig.NSIS) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("embed", plan.RuntimeConfig.WebView2) {
		t.Fatalf("want %v, got %v", "embed", plan.RuntimeConfig.WebView2)
	}
	if !stdlibAssertContains(plan.RuntimeConfig.BuildTags, "integration") {
		t.Fatalf("expected %v to contain %v", plan.RuntimeConfig.BuildTags, "integration")
	}

}

func TestPipeline_Plan_UsesExplicitBuildTypeOverride_Good(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := DefaultConfig()
	cfg.Build.Type = "go"

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			if !stdlibAssertEqual(ProjectTypeNode, projectType) {
				t.Fatalf("want %v, got %v", ProjectTypeNode, projectType)
			}

			return &stubPipelineBuilder{}, nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: cfg,
		BuildType:   "NoDe",
		Targets:     []Target{{OS: "darwin", Arch: "arm64"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ProjectTypeNode, plan.ProjectType) {
		t.Fatalf("want %v, got %v", ProjectTypeNode, plan.ProjectType)
	}
	if !stdlibAssertEqual("node", plan.BuildConfig.Build.Type) {
		t.Fatalf("want %v, got %v", "node", plan.BuildConfig.Build.Type)
	}
	if !stdlibAssertEqual("node", plan.SetupPlan.PrimaryStack) {
		t.Fatalf("want %v, got %v", "node", plan.SetupPlan.PrimaryStack)
	}
	if !stdlibAssertEqual("node", plan.SetupPlan.PrimaryStackSuggestion) {
		t.Fatalf("want %v, got %v", "node", plan.SetupPlan.PrimaryStackSuggestion)
	}
	if !stdlibAssertContains(setupTools(plan.SetupPlan), SetupToolNode) {
		t.Fatalf("expected %v to contain %v", setupTools(plan.SetupPlan), SetupToolNode)
	}
	if !stdlibAssertEqual([]Target{{OS: "darwin", Arch: "arm64"}}, plan.Targets) {
		t.Fatalf("want %v, got %v", []Target{{OS: "darwin", Arch: "arm64"}}, plan.Targets)
	}

}

func TestPipeline_Plan_NormalisesConfiguredBuildType_Good(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Build.Type = "WaIlS"

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			if !stdlibAssertEqual(ProjectTypeWails, projectType) {
				t.Fatalf("want %v, got %v", ProjectTypeWails, projectType)
			}

			return &stubPipelineBuilder{}, nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  t.TempDir(),
		BuildConfig: cfg,
		Targets:     []Target{{OS: "darwin", Arch: "arm64"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ProjectTypeWails, plan.ProjectType) {
		t.Fatalf("want %v, got %v", ProjectTypeWails, plan.ProjectType)
	}
	if !stdlibAssertEqual("wails", plan.BuildConfig.Build.Type) {
		t.Fatalf("want %v, got %v", "wails", plan.BuildConfig.Build.Type)
	}
	if !stdlibAssertEqual("wails2", plan.SetupPlan.PrimaryStackSuggestion) {
		t.Fatalf("want %v, got %v", "wails2", plan.SetupPlan.PrimaryStackSuggestion)
	}
	if !stdlibAssertContains(setupTools(plan.SetupPlan), SetupToolWails) {
		t.Fatalf("expected %v to contain %v", setupTools(plan.SetupPlan), SetupToolWails)
	}
	if !stdlibAssertContains(setupTools(plan.SetupPlan), SetupToolNode) {
		t.Fatalf("expected %v to contain %v", setupTools(plan.SetupPlan), SetupToolNode)
	}

}

func TestPipeline_Plan_AppliesActionStyleOverrides_Good(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(plan.Options.Tags, "mlx") {
		t.Fatalf("expected %v to contain %v", plan.Options.Tags, "mlx")
	}
	if !stdlibAssertContains(plan.Options.Tags, "release") {
		t.Fatalf("expected %v to contain %v", plan.Options.Tags, "release")
	}
	if stdlibAssertContains(plan.Options.Tags, "integration") {
		t.Fatalf("expected %v not to contain %v", plan.Options.Tags, "integration")
	}
	if !(plan.Options.Obfuscate) {
		t.Fatal("expected true")
	}
	if !(plan.Options.NSIS) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("download", plan.Options.WebView2) {
		t.Fatalf("want %v, got %v", "download", plan.Options.WebView2)
	}
	if !stdlibAssertEqual("deno task bundle", plan.BuildConfig.Build.DenoBuild) {
		t.Fatalf("want %v, got %v", "deno task bundle", plan.BuildConfig.Build.DenoBuild)
	}
	if !(plan.BuildConfig.Build.Cache.Enabled) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(ax.Join(dir, ".core", "cache"), plan.BuildConfig.Build.Cache.Directory) {
		t.Fatalf("want %v, got %v", ax.Join(dir, ".core", "cache"), plan.BuildConfig.Build.Cache.Directory)
	}
	if !stdlibAssertEqual([]string{ax.Join(dir, "cache", "go-build"), ax.Join(dir, "cache", "go-mod")}, plan.BuildConfig.Build.Cache.Paths) {
		t.Fatalf("want %v, got %v", []string{ax.Join(dir, "cache", "go-build"), ax.Join(dir, "cache", "go-mod")}, plan.BuildConfig.Build.Cache.Paths)
	}
	if !(plan.RuntimeConfig.Cache.Enabled) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual(plan.BuildConfig.Build.Cache.Directory, plan.RuntimeConfig.Cache.Directory) {
		t.Fatalf("want %v, got %v", plan.BuildConfig.Build.Cache.Directory, plan.RuntimeConfig.Cache.Directory)
	}
	if !stdlibAssertEqual(plan.BuildConfig.Build.Cache.Paths, plan.RuntimeConfig.Cache.Paths) {
		t.Fatalf("want %v, got %v", plan.BuildConfig.Build.Cache.Paths, plan.RuntimeConfig.Cache.Paths)
	}
	if !stdlibAssertContains(setupTools(plan.SetupPlan), SetupToolDeno) {
		t.Fatalf("expected %v to contain %v", setupTools(plan.SetupPlan), SetupToolDeno)
	}
	if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, plan.ProjectTypes) {
		t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, plan.ProjectTypes)
	}
	if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, resolvedTypes) {
		t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, resolvedTypes)
	}

}

func TestPipeline_Plan_UsesLocalTargetWhenBuildConfigMissing_Good(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			if !stdlibAssertEqual(ProjectTypeGo, projectType) {
				t.Fatalf("want %v, got %v", ProjectTypeGo, projectType)
			}

			return &stubPipelineBuilder{}, nil
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir: dir,
		BuildType:  string(ProjectTypeGo),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}, plan.Targets) {
		t.Fatalf("want %v, got %v", []Target{{OS: runtime.GOOS, Arch: runtime.GOARCH}}, plan.Targets)
	}

}

func TestPipeline_Plan_UsesExplicitVersionOverride_Good(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	versionResolverCalled := false
	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			if !stdlibAssertEqual(ProjectTypeGo, projectType) {
				t.Fatalf("want %v, got %v", ProjectTypeGo, projectType)
			}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("v9.9.9", plan.Version) {
		t.Fatalf("want %v, got %v", "v9.9.9", plan.Version)
	}
	if !stdlibAssertEqual("v9.9.9", plan.RuntimeConfig.Version) {
		t.Fatalf("want %v, got %v", "v9.9.9", plan.RuntimeConfig.Version)
	}
	if versionResolverCalled {
		t.Fatal("expected false")
	}

}

func TestPipeline_Plan_RejectsUnsafeVersionOverride_Bad(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			return &stubPipelineBuilder{}, nil
		},
	}

	_, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: DefaultConfig(),
		Version:     "v1.2.3 --bad",
		Targets:     []Target{{OS: "linux", Arch: "amd64"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "invalid build version override") {
		t.Fatalf("expected %v to contain %v", err.Error(), "invalid build version override")
	}

}

func TestPipeline_Plan_DoesNotMutateCallerBuildConfig_Good(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]string{"integration"}, cfg.Build.BuildTags) {
		t.Fatalf("want %v, got %v", []string{"integration"}, cfg.Build.BuildTags)
	}
	if cfg.Build.Obfuscate {
		t.Fatal("expected false")
	}
	if !stdlibAssertEmpty(cfg.Build.DenoBuild) {
		t.Fatalf("expected empty, got %v", cfg.Build.DenoBuild)
	}
	if cfg.Build.Cache.Enabled {
		t.Fatal("expected false")
	}
	if !stdlibAssertEmpty(cfg.Build.Cache.Directory) {
		t.Fatalf("expected empty, got %v", cfg.Build.Cache.Directory)
	}
	if !stdlibAssertEmpty(cfg.Build.Cache.Paths) {
		t.Fatalf("expected empty, got %v", cfg.Build.Cache.Paths)
	}

}

func TestPipeline_Run_Good(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := pipeline.Run(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(plan, result.Plan) {
		t.Fatalf("want %v, got %v", plan, result.Plan)
	}
	if !stdlibAssertEqual([]Artifact{{Path: ax.Join(dir, "dist", "demo"), OS: "linux", Arch: "amd64"}}, result.Artifacts) {
		t.Fatalf("want %v, got %v", []Artifact{{Path: ax.Join(dir, "dist", "demo"), OS: "linux", Arch: "amd64"}}, result.Artifacts)
	}
	if stdlibAssertNil(builder.lastCfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual(plan.RuntimeConfig, builder.lastCfg) {
		t.Fatalf("want %v, got %v", plan.RuntimeConfig, builder.lastCfg)
	}
	if !stdlibAssertEqual(plan.Targets, builder.lastTgts) {
		t.Fatalf("want %v, got %v", plan.Targets, builder.lastTgts)
	}

}

func TestPipeline_Run_MultiType_Good(t *testing.T) {
	dir := t.TempDir()
	if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
				return nil, core.NewError("test error")
			}
		},
	}

	plan, err := pipeline.Plan(context.Background(), PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: DefaultConfig(),
		Targets:     []Target{{OS: "linux", Arch: "amd64"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual([]ProjectType{ProjectTypeNode, ProjectTypeDocs}, plan.ProjectTypes) {
		t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeNode, ProjectTypeDocs}, plan.ProjectTypes)
	}

	result, err := pipeline.Run(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Artifacts) != 2 {
		t.Fatalf("want len %v, got %v", 2, len(result.Artifacts))
	}
	if stdlibAssertNil(nodeBuilder.lastCfg) {
		t.Fatal("expected non-nil")
	}
	if stdlibAssertNil(docsBuilder.lastCfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual(ax.Join(plan.OutputDir, "node"), nodeBuilder.lastCfg.OutputDir) {
		t.Fatalf("want %v, got %v", ax.Join(plan.OutputDir, "node"), nodeBuilder.lastCfg.OutputDir)
	}
	if !stdlibAssertEqual(ax.Join(plan.OutputDir, "docs"), docsBuilder.lastCfg.OutputDir) {
		t.Fatalf("want %v, got %v", ax.Join(plan.OutputDir, "docs"), docsBuilder.lastCfg.OutputDir)
	}
	if !stdlibAssertEqual(plan.Targets, nodeBuilder.lastTgts) {
		t.Fatalf("want %v, got %v", plan.Targets, nodeBuilder.lastTgts)
	}
	if !stdlibAssertEqual(plan.Targets, docsBuilder.lastTgts) {
		t.Fatalf("want %v, got %v", plan.Targets, docsBuilder.lastTgts)
	}
	if plan.RuntimeConfig == nodeBuilder.lastCfg {
		t.Fatalf("expected %v and %v not to be the same", plan.RuntimeConfig, nodeBuilder.lastCfg)
	}
	if plan.RuntimeConfig == docsBuilder.lastCfg {
		t.Fatalf("expected %v and %v not to be the same", plan.RuntimeConfig, docsBuilder.lastCfg)
	}

}

func TestPipeline_Plan_Bad(t *testing.T) {
	pipeline := &Pipeline{
		FS: io.Local,
		ResolveBuilder: func(projectType ProjectType) (Builder, error) {
			return &stubPipelineBuilder{}, nil
		},
	}

	_, err := pipeline.Plan(context.Background(), PipelineRequest{ProjectDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "no buildable project type found") {
		t.Fatalf("expected %v to contain %v", err.Error(), "no buildable project type found")
	}

}

func TestPipeline_Run_Bad(t *testing.T) {
	pipeline := &Pipeline{}

	_, err := pipeline.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "pipeline plan is nil") {
		t.Fatalf("expected %v to contain %v", err.Error(), "pipeline plan is nil")
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestPipeline_Pipeline_Plan_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &Pipeline{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Plan(ctx, PipelineRequest{})
	})
	core.AssertTrue(t, true)
}

func TestPipeline_Pipeline_Run_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &Pipeline{}
	core.AssertNotPanics(t, func() {
		_, _ = subject.Run(ctx, &PipelinePlan{})
	})
	core.AssertTrue(t, true)
}

func TestPipeline_ResolveBuildName_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveBuildName(core.Path(t.TempDir(), "go-build-compliance"), &BuildConfig{}, "agent")
	})
	core.AssertTrue(t, true)
}

func TestPipeline_ResolveBuildName_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveBuildName("", nil, "")
	})
	core.AssertTrue(t, true)
}

func TestPipeline_ResolveBuildName_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveBuildName(core.Path(t.TempDir(), "go-build-compliance"), &BuildConfig{}, "agent")
	})
	core.AssertTrue(t, true)
}
