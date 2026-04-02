// SPDX-Licence-Identifier: EUPL-1.2

// Package api provides a service provider that wraps go-build's build,
// release, and SDK subsystems as REST endpoints with WebSocket event
// streaming.
package api

import (
	"errors"
	stdio "io"
	"io/fs"
	"net/http"

	"dappco.re/go/core/api"
	"dappco.re/go/core/api/pkg/provider"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/internal/projectdetect"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/build/builders"
	"dappco.re/go/core/build/pkg/release"
	"dappco.re/go/core/build/pkg/sdk"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"dappco.re/go/core/ws"
	"github.com/gin-gonic/gin"
)

// BuildProvider wraps go-build's build, release, and SDK operations as a
// service provider. It implements Provider, Streamable, Describable, and
// Renderable.
//
// p := api.NewProvider(".", hub)
type BuildProvider struct {
	hub        *ws.Hub
	projectDir string
	medium     io.Medium
}

// compile-time interface checks
var (
	_ provider.Provider    = (*BuildProvider)(nil)
	_ provider.Streamable  = (*BuildProvider)(nil)
	_ provider.Describable = (*BuildProvider)(nil)
	_ provider.Renderable  = (*BuildProvider)(nil)
)

// NewProvider creates a BuildProvider for the given project directory.
// If projectDir is empty, the current working directory is used.
// The WS hub is used to emit real-time build events; pass nil if not available.
//
// p := api.NewProvider(".", hub)
func NewProvider(projectDir string, hub *ws.Hub) *BuildProvider {
	if projectDir == "" {
		projectDir = "."
	}
	return &BuildProvider{
		hub:        hub,
		projectDir: projectDir,
		medium:     io.Local,
	}
}

// Name implements api.RouteGroup.
//
// name := p.Name() // → "build"
func (p *BuildProvider) Name() string { return "build" }

// BasePath implements api.RouteGroup.
//
// path := p.BasePath() // → "/api/v1/build"
func (p *BuildProvider) BasePath() string { return "/api/v1/build" }

// Element implements provider.Renderable.
//
// spec := p.Element() // → {Tag: "core-build-panel", Source: "/assets/core-build.js"}
func (p *BuildProvider) Element() provider.ElementSpec {
	return provider.ElementSpec{
		Tag:    "core-build-panel",
		Source: "/assets/core-build.js",
	}
}

// Channels implements provider.Streamable.
//
// channels := p.Channels() // → ["build.started", "build.complete", ...]
func (p *BuildProvider) Channels() []string {
	return []string{
		"build.started",
		"build.complete",
		"build.failed",
		"release.started",
		"release.complete",
		"workflow.generated",
		"sdk.generated",
	}
}

// RegisterRoutes implements api.RouteGroup.
//
// p.RegisterRoutes(rg)
func (p *BuildProvider) RegisterRoutes(rg *gin.RouterGroup) {
	// Build
	rg.GET("/config", p.getConfig)
	rg.GET("/discover", p.discoverProject)
	rg.POST("/build", p.triggerBuild)
	rg.GET("/artifacts", p.listArtifacts)

	// Release
	rg.GET("/release/version", p.getVersion)
	rg.GET("/release/changelog", p.getChangelog)
	rg.POST("/release", p.triggerRelease)
	rg.POST("/release/workflow", p.generateReleaseWorkflow)

	// SDK
	rg.GET("/sdk/diff", p.getSdkDiff)
	rg.POST("/sdk/generate", p.generateSdk)
}

// Describe implements api.DescribableGroup.
//
// routes := p.Describe() // → [{Method: "GET", Path: "/config", ...}, ...]
func (p *BuildProvider) Describe() []api.RouteDescription {
	return []api.RouteDescription{
		{
			Method:      "GET",
			Path:        "/config",
			Summary:     "Read build configuration",
			Description: "Loads and returns the .core/build.yaml from the project directory.",
			Tags:        []string{"build", "config"},
		},
		{
			Method:      "GET",
			Path:        "/discover",
			Summary:     "Detect project type",
			Description: "Scans the project directory for marker files and returns detected project types plus frontend and distro metadata.",
			Tags:        []string{"build", "discovery"},
		},
		{
			Method:      "POST",
			Path:        "/build",
			Summary:     "Trigger a build",
			Description: "Runs the full build pipeline for the project, producing artifacts in dist/.",
			Tags:        []string{"build"},
		},
		{
			Method:      "GET",
			Path:        "/artifacts",
			Summary:     "List build artifacts",
			Description: "Returns the contents of the dist/ directory with file sizes.",
			Tags:        []string{"build", "artifacts"},
		},
		{
			Method:      "GET",
			Path:        "/release/version",
			Summary:     "Get current version",
			Description: "Determines the current version from git tags.",
			Tags:        []string{"release", "version"},
		},
		{
			Method:      "GET",
			Path:        "/release/changelog",
			Summary:     "Generate changelog",
			Description: "Generates a conventional-commit changelog from git history.",
			Tags:        []string{"release", "changelog"},
		},
		{
			Method:      "POST",
			Path:        "/release",
			Summary:     "Trigger release pipeline",
			Description: "Publishes pre-built artifacts from dist/ to configured targets.",
			Tags:        []string{"release"},
		},
		{
			Method:      "POST",
			Path:        "/release/workflow",
			Summary:     "Generate release workflow",
			Description: "Writes the embedded GitHub Actions release workflow into .github/workflows/release.yml or a custom path.",
			Tags:        []string{"release", "workflow"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Preferred workflow path input, relative to the project directory or absolute.",
					},
					"workflowPath": map[string]any{
						"type":        "string",
						"description": "Predictable alias for path, relative to the project directory or absolute.",
					},
					"workflow_path": map[string]any{
						"type":        "string",
						"description": "Snake_case alias for workflowPath.",
					},
					"workflow-path": map[string]any{
						"type":        "string",
						"description": "Hyphenated alias for workflowPath.",
					},
					"workflowOutputPath": map[string]any{
						"type":        "string",
						"description": "Predictable alias for outputPath, relative to the project directory or absolute.",
					},
					"workflow_output": map[string]any{
						"type":        "string",
						"description": "Snake_case alias for workflowOutputPath.",
					},
					"workflow-output": map[string]any{
						"type":        "string",
						"description": "Hyphenated alias for workflowOutputPath.",
					},
					"workflow_output_path": map[string]any{
						"type":        "string",
						"description": "Snake_case alias for workflowOutputPath.",
					},
					"workflow-output-path": map[string]any{
						"type":        "string",
						"description": "Hyphenated alias for workflowOutputPath.",
					},
					"outputPath": map[string]any{
						"type":        "string",
						"description": "Preferred explicit workflow output path, relative to the project directory or absolute.",
					},
					"output-path": map[string]any{
						"type":        "string",
						"description": "Hyphenated alias for outputPath.",
					},
					"output_path": map[string]any{
						"type":        "string",
						"description": "Snake_case alias for outputPath.",
					},
					"output": map[string]any{
						"type":        "string",
						"description": "Legacy alias for outputPath.",
					},
				},
			},
		},
		{
			Method:      "GET",
			Path:        "/sdk/diff",
			Summary:     "OpenAPI breaking change diff",
			Description: "Compares two OpenAPI specs and reports breaking changes. Requires base and revision query parameters.",
			Tags:        []string{"sdk", "diff"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"base":     map[string]any{"type": "string", "description": "Path to the base OpenAPI spec"},
					"revision": map[string]any{"type": "string", "description": "Path to the revision OpenAPI spec"},
				},
			},
		},
		{
			Method:      "POST",
			Path:        "/sdk/generate",
			Summary:     "Generate SDK",
			Description: "Generates SDK client libraries for configured languages from the OpenAPI spec.",
			Tags:        []string{"sdk"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"language": map[string]any{"type": "string", "description": "Target language (typescript, python, go, php). Omit for all."},
				},
			},
		},
	}
}

// resolveDir returns the absolute project directory.
func (p *BuildProvider) resolveDir() (string, error) {
	return ax.Abs(p.projectDir)
}

// -- Build Handlers -----------------------------------------------------------

func (p *BuildProvider) getConfig(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	cfg, err := build.LoadConfig(p.medium, dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("config_load_failed", err.Error()))
		return
	}

	hasConfig := build.ConfigExists(p.medium, dir)

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"config":     cfg,
		"has_config": hasConfig,
		"path":       build.ConfigPath(dir),
	}))
}

func (p *BuildProvider) discoverProject(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	discovery, err := build.DiscoverFull(p.medium, dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("discover_failed", err.Error()))
		return
	}

	// Convert to string slice for JSON
	typeStrings := make([]string, len(discovery.Types))
	for i, t := range discovery.Types {
		typeStrings[i] = string(t)
	}

	primary := ""
	if len(discovery.Types) > 0 {
		primary = string(discovery.Types[0])
	}

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"types":           typeStrings,
		"primary":         primary,
		"primary_stack":   discovery.PrimaryStack,
		"dir":             dir,
		"has_frontend":    discovery.HasFrontend,
		"has_subtree_npm": discovery.HasSubtreeNpm,
		"markers":         discovery.Markers,
		"distro":          discovery.Distro,
	}))
}

func (p *BuildProvider) triggerBuild(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	// Load build config
	cfg, err := build.LoadConfig(p.medium, dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("config_load_failed", err.Error()))
		return
	}

	discovery, err := build.DiscoverFull(p.medium, dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("discover_failed", err.Error()))
		return
	}

	// Detect project type, honouring an explicit build.type override.
	projectType, err := resolveProjectType(p.medium, dir, cfg.Build.Type)
	if err != nil || projectType == "" {
		c.JSON(http.StatusBadRequest, api.Fail("no_project", "no buildable project detected"))
		return
	}

	// Get builder
	builder, err := getBuilder(projectType)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail("unsupported_type", err.Error()))
		return
	}

	// Determine version
	version, verr := release.DetermineVersionWithContext(c.Request.Context(), dir)
	if verr != nil {
		version = "dev"
	}

	// Build name
	binaryName := cfg.Project.Binary
	if binaryName == "" {
		binaryName = cfg.Project.Name
	}
	if binaryName == "" {
		binaryName = ax.Base(dir)
	}

	outputDir := ax.Join(dir, "dist")

	buildConfig := &build.Config{
		FS:         p.medium,
		ProjectDir: dir,
		OutputDir:  outputDir,
		Name:       binaryName,
		Version:    version,
		LDFlags:    cfg.Build.LDFlags,
		CGO:        cfg.Build.CGO,
	}
	build.ApplyOptions(buildConfig, build.ComputeOptions(cfg, discovery))

	targets := cfg.ToTargets()

	p.emitEvent("build.started", map[string]any{
		"project_type": string(projectType),
		"targets":      targets,
		"version":      version,
	})

	artifacts, err := builder.Build(c.Request.Context(), buildConfig, targets)
	if err != nil {
		p.emitEvent("build.failed", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, api.Fail("build_failed", err.Error()))
		return
	}

	// Archive and checksum
	archived, err := build.ArchiveAll(p.medium, artifacts)
	if err != nil {
		p.emitEvent("build.failed", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, api.Fail("archive_failed", err.Error()))
		return
	}

	checksummed, err := build.ChecksumAll(p.medium, archived)
	if err != nil {
		p.emitEvent("build.failed", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, api.Fail("checksum_failed", err.Error()))
		return
	}

	p.emitEvent("build.complete", map[string]any{
		"artifact_count": len(checksummed),
		"version":        version,
	})

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"artifacts": checksummed,
		"version":   version,
	}))
}

// resolveProjectType returns the configured build type when present, otherwise it falls back to detection.
func resolveProjectType(filesystem io.Medium, projectDir, buildType string) (build.ProjectType, error) {
	if buildType != "" {
		return build.ProjectType(buildType), nil
	}

	return projectdetect.DetectProjectType(filesystem, projectDir)
}

// artifactInfo holds JSON-friendly metadata about a dist/ file.
type artifactInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (p *BuildProvider) listArtifacts(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	distDir := ax.Join(dir, "dist")
	if !p.medium.IsDir(distDir) {
		c.JSON(http.StatusOK, api.OK(map[string]any{
			"artifacts": []artifactInfo{},
			"exists":    false,
		}))
		return
	}

	entries, err := p.medium.List(distDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("list_failed", err.Error()))
		return
	}

	var artifacts []artifactInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		artifacts = append(artifacts, artifactInfo{
			Name: entry.Name(),
			Path: ax.Join(distDir, entry.Name()),
			Size: info.Size(),
		})
	}

	if artifacts == nil {
		artifacts = []artifactInfo{}
	}

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"artifacts": artifacts,
		"exists":    true,
	}))
}

// -- Release Handlers ---------------------------------------------------------

func (p *BuildProvider) getVersion(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	version, err := release.DetermineVersionWithContext(c.Request.Context(), dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("version_failed", err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"version": version,
	}))
}

func (p *BuildProvider) getChangelog(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	// Optional query params for ref range
	fromRef := c.Query("from")
	toRef := c.Query("to")

	changelog, err := release.GenerateWithContext(c.Request.Context(), dir, fromRef, toRef)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("changelog_failed", err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"changelog": changelog,
	}))
}

func (p *BuildProvider) triggerRelease(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	cfg, err := release.LoadConfig(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("config_load_failed", err.Error()))
		return
	}

	// Parse optional dry_run parameter
	dryRun := c.Query("dry_run") == "true"

	p.emitEvent("release.started", map[string]any{
		"dry_run": dryRun,
	})

	rel, err := release.Publish(c.Request.Context(), cfg, dryRun)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("release_failed", err.Error()))
		return
	}

	p.emitEvent("release.complete", map[string]any{
		"version":        rel.Version,
		"artifact_count": len(rel.Artifacts),
		"dry_run":        dryRun,
	})

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"version":   rel.Version,
		"artifacts": rel.Artifacts,
		"changelog": rel.Changelog,
		"dry_run":   dryRun,
	}))
}

// ReleaseWorkflowRequest captures the workflow-generation inputs exposed by the API.
//
// req := ReleaseWorkflowRequest{Path: "ci/release.yml"}
// req := ReleaseWorkflowRequest{WorkflowOutputPath: "ops/release.yml"}
type ReleaseWorkflowRequest struct {
	Path                     string `json:"path"`
	WorkflowPath             string `json:"workflowPath"`
	WorkflowPathSnake        string `json:"workflow_path"`
	WorkflowPathHyphen       string `json:"workflow-path"`
	OutputPath               string `json:"outputPath"`
	OutputPathHyphen         string `json:"output-path"`
	OutputPathSnake          string `json:"output_path"`
	LegacyOutputPath         string `json:"output"`
	WorkflowOutputPath       string `json:"workflowOutputPath"`
	WorkflowOutputSnake      string `json:"workflow_output"`
	WorkflowOutputHyphen     string `json:"workflow-output"`
	WorkflowOutputPathSnake  string `json:"workflow_output_path"`
	WorkflowOutputPathHyphen string `json:"workflow-output-path"`
}

// resolveWorkflowTargetPath resolves both workflow path inputs and workflow
// output inputs before merging them into the final target path.
//
// req := ReleaseWorkflowRequest{Path: "ci/release.yml"}
// path, err := req.resolveWorkflowTargetPath("/tmp/project", io.Local)
func (r ReleaseWorkflowRequest) resolveWorkflowTargetPath(dir string, medium io.Medium) (string, error) {
	outputPath, err := r.resolveOutputPath(dir, medium)
	if err != nil {
		return "", err
	}

	workflowPath, err := r.resolveWorkflowPath(dir, medium)
	if err != nil {
		return "", err
	}

	return build.ResolveReleaseWorkflowInputPathWithMedium(medium, dir, workflowPath, outputPath)
}

// resolveWorkflowPath resolves the workflow path aliases with the same
// conflict rules as the CLI.
//
// req := ReleaseWorkflowRequest{WorkflowPath: "ci/release.yml"}
// workflowPath, err := req.resolveWorkflowPath("/tmp/project", io.Local)
func (r ReleaseWorkflowRequest) resolveWorkflowPath(dir string, medium io.Medium) (string, error) {
	workflowPath, err := build.ResolveReleaseWorkflowInputPathAliases(
		medium,
		dir,
		r.Path,
		r.WorkflowPath,
		r.WorkflowPathSnake,
		r.WorkflowPathHyphen,
	)
	if err != nil {
		return "", coreerr.E("api.ReleaseWorkflowRequest", "workflow path aliases specify different locations", nil)
	}

	return workflowPath, nil
}

// resolveOutputPath resolves the workflow output aliases with the same
// conflict rules as the CLI.
//
// req := ReleaseWorkflowRequest{WorkflowOutputPath: "ci/release.yml"}
// outputPath, err := req.resolveOutputPath("/tmp/project")
func (r ReleaseWorkflowRequest) resolveOutputPath(dir string, medium io.Medium) (string, error) {
	resolvedOutputPath, err := build.ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(
		medium,
		dir,
		r.OutputPath,
		r.OutputPathHyphen,
		r.OutputPathSnake,
		r.LegacyOutputPath,
		r.WorkflowOutputPath,
		r.WorkflowOutputSnake,
		r.WorkflowOutputHyphen,
		r.WorkflowOutputPathSnake,
		r.WorkflowOutputPathHyphen,
	)
	if err != nil {
		return "", coreerr.E("api.ReleaseWorkflowRequest", "workflow output aliases specify different locations", nil)
	}

	return resolvedOutputPath, nil
}

func (p *BuildProvider) generateReleaseWorkflow(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	var req ReleaseWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty bodies are valid; malformed JSON is not.
		if !errors.Is(err, stdio.EOF) {
			c.JSON(http.StatusBadRequest, api.Fail("invalid_request", err.Error()))
			return
		}
	}

	workflowPath, err := req.resolveWorkflowTargetPath(dir, p.medium)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail("invalid_request", err.Error()))
		return
	}

	if err := build.WriteReleaseWorkflow(p.medium, workflowPath); err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("workflow_write_failed", err.Error()))
		return
	}

	p.emitEvent("workflow.generated", map[string]any{
		"path":      workflowPath,
		"generated": true,
	})

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"generated": true,
		"path":      workflowPath,
	}))
}

// -- SDK Handlers -------------------------------------------------------------

func (p *BuildProvider) getSdkDiff(c *gin.Context) {
	basePath := c.Query("base")
	revisionPath := c.Query("revision")

	if basePath == "" || revisionPath == "" {
		c.JSON(http.StatusBadRequest, api.Fail("missing_params", "base and revision query parameters are required"))
		return
	}

	result, err := sdk.Diff(basePath, revisionPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("diff_failed", err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.OK(result))
}

type sdkGenerateRequest struct {
	Language string `json:"language"`
}

func (p *BuildProvider) generateSdk(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	var req sdkGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// No body is fine — generate all languages
		req.Language = ""
	}

	// Load SDK config from release config
	relCfg, err := release.LoadConfig(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("config_load_failed", err.Error()))
		return
	}

	var sdkCfg *sdk.Config
	if relCfg.SDK != nil {
		sdkCfg = &sdk.Config{
			Spec:      relCfg.SDK.Spec,
			Languages: relCfg.SDK.Languages,
			Output:    relCfg.SDK.Output,
			Package: sdk.PackageConfig{
				Name:    relCfg.SDK.Package.Name,
				Version: relCfg.SDK.Package.Version,
			},
			Diff: sdk.DiffConfig{
				Enabled:        relCfg.SDK.Diff.Enabled,
				FailOnBreaking: relCfg.SDK.Diff.FailOnBreaking,
			},
			Publish: sdk.PublishConfig{
				Repo: relCfg.SDK.Publish.Repo,
				Path: relCfg.SDK.Publish.Path,
			},
		}
	}

	s := sdk.New(dir, sdkCfg)

	ctx := c.Request.Context()
	if req.Language != "" {
		err = s.GenerateLanguage(ctx, req.Language)
	} else {
		err = s.Generate(ctx)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("sdk_generate_failed", err.Error()))
		return
	}

	p.emitEvent("sdk.generated", map[string]any{
		"language": req.Language,
	})

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"generated": true,
		"language":  req.Language,
	}))
}

// -- Internal Helpers ---------------------------------------------------------

// getBuilder returns the appropriate builder for the project type.
func getBuilder(projectType build.ProjectType) (build.Builder, error) {
	switch projectType {
	case build.ProjectTypeWails:
		return builders.NewWailsBuilder(), nil
	case build.ProjectTypeGo:
		return builders.NewGoBuilder(), nil
	case build.ProjectTypeNode:
		return builders.NewNodeBuilder(), nil
	case build.ProjectTypePHP:
		return builders.NewPHPBuilder(), nil
	case build.ProjectTypePython:
		return builders.NewPythonBuilder(), nil
	case build.ProjectTypeRust:
		return builders.NewRustBuilder(), nil
	case build.ProjectTypeDocs:
		return builders.NewDocsBuilder(), nil
	case build.ProjectTypeCPP:
		return builders.NewCPPBuilder(), nil
	case build.ProjectTypeDocker:
		return builders.NewDockerBuilder(), nil
	case build.ProjectTypeLinuxKit:
		return builders.NewLinuxKitBuilder(), nil
	case build.ProjectTypeTaskfile:
		return builders.NewTaskfileBuilder(), nil
	default:
		return nil, fs.ErrNotExist
	}
}

// emitEvent sends a WS event if the hub is available.
func (p *BuildProvider) emitEvent(channel string, data any) {
	if p.hub == nil {
		return
	}
	_ = p.hub.SendToChannel(channel, ws.Message{
		Type: ws.TypeEvent,
		Data: data,
	})
}
