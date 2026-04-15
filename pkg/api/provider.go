// SPDX-Licence-Identifier: EUPL-1.2

// Package api provides a service provider that wraps go-build's build,
// release, and SDK subsystems as REST endpoints with WebSocket event
// streaming.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"runtime"
	"sort"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/projectdetect"
	"dappco.re/go/build/internal/sdkcfg"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/builders"
	"dappco.re/go/build/pkg/build/signing"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/core/api"
	"dappco.re/go/core/api/pkg/provider"
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

type buildRequest struct {
	Archive  *bool `json:"archive,omitempty"`
	Checksum *bool `json:"checksum,omitempty"`
	Package  *bool `json:"package,omitempty"`
}

var (
	providerResolveProjectType = resolveProjectType
	providerGetBuilder         = getBuilder
	providerDetermineVersion   = release.DetermineVersionWithContext
	providerLoadReleaseConfig  = release.LoadConfig
	providerRunRelease         = release.Run
	providerSignBinaries       = signing.SignBinaries
	providerNotarizeBinaries   = signing.NotarizeBinaries
	providerSignChecksums      = signing.SignChecksums
)

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
	rg.POST("", p.triggerBuild)
	rg.POST("/build", p.triggerBuild)
	rg.GET("/artifacts", p.listArtifacts)
	rg.GET("/events", p.streamEvents)

	// Release
	rg.GET("/release/version", p.getVersion)
	rg.GET("/release/changelog", p.getChangelog)
	rg.POST("/release", p.triggerRelease)
	rg.POST("/release/workflow", p.generateReleaseWorkflow)

	// SDK
	rg.GET("/sdk/diff", p.getSdkDiff)
	rg.POST("/sdk", p.generateSdk)
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
			Description: "Scans the project directory for marker files and returns detected project types plus frontend, setup-plan, and distro metadata.",
			Tags:        []string{"build", "discovery"},
		},
		{
			Method:      "POST",
			Path:        "/",
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
			Path:        "/events",
			Summary:     "Subscribe to build events",
			Description: "Upgrades to a WebSocket connection and streams build, release, workflow, and SDK events emitted by this provider.",
			Tags:        []string{"build", "events"},
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
			Description: "Runs the full release pipeline: build, sign, archive, checksum, and publish.",
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
			Path:        "/sdk",
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
	options := build.ComputeOptions(cfg, discovery)
	setupPlan, err := build.ComputeSetupPlan(p.medium, dir, cfg, discovery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("setup_plan_failed", err.Error()))
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
		"types":                     typeStrings,
		"configured_type":           discovery.ConfiguredType,
		"configured_build_type":     discovery.ConfiguredBuildType,
		"os":                        discovery.OS,
		"arch":                      discovery.Arch,
		"primary":                   primary,
		"primary_stack":             discovery.PrimaryStack,
		"suggested_stack":           discovery.SuggestedStack,
		"primary_stack_suggestion":  discovery.PrimaryStackSuggestion,
		"dir":                       dir,
		"has_frontend":              discovery.HasFrontend,
		"has_root_package_json":     discovery.HasRootPackageJSON,
		"has_frontend_package_json": discovery.HasFrontendPackageJSON,
		"has_root_composer_json":    discovery.HasRootComposerJSON,
		"has_root_cargo_toml":       discovery.HasRootCargoToml,
		"has_package_json":          discovery.HasPackageJSON,
		"has_deno_manifest":         discovery.HasDenoManifest,
		"has_root_go_mod":           discovery.HasRootGoMod,
		"has_root_go_work":          discovery.HasRootGoWork,
		"has_root_main_go":          discovery.HasRootMainGo,
		"has_root_cmakelists":       discovery.HasRootCMakeLists,
		"has_root_wails_json":       discovery.HasRootWailsJSON,
		"has_taskfile":              discovery.HasTaskfile,
		"has_subtree_npm":           discovery.HasSubtreeNpm,
		"has_subtree_package_json":  discovery.HasSubtreePackageJSON,
		"has_subtree_deno_manifest": discovery.HasSubtreeDenoManifest,
		"has_docs_config":           discovery.HasDocsConfig,
		"has_go_toolchain":          discovery.HasGoToolchain,
		"deno_requested":            build.DenoRequested(cfg.Build.DenoBuild),
		"npm_requested":             build.NpmRequested(cfg.Build.NpmBuild),
		"linux_packages":            discovery.LinuxPackages,
		"webkit_package":            discovery.WebKitPackage,
		"ref":                       discovery.Ref,
		"branch":                    discovery.Branch,
		"tag":                       discovery.Tag,
		"is_tag":                    discovery.IsTag,
		"sha":                       discovery.SHA,
		"short_sha":                 discovery.ShortSHA,
		"repo":                      discovery.Repo,
		"owner":                     discovery.Owner,
		"build_options":             options.String(),
		"options": map[string]any{
			"obfuscate": options.Obfuscate,
			"tags":      options.Tags,
			"nsis":      options.NSIS,
			"webview2":  options.WebView2,
			"ldflags":   options.LDFlags,
		},
		"setup_plan": map[string]any{
			"project_dir":              setupPlan.ProjectDir,
			"primary_stack":            setupPlan.PrimaryStack,
			"primary_stack_suggestion": setupPlan.PrimaryStackSuggestion,
			"frontend_dirs":            setupPlan.FrontendDirs,
			"linux_packages":           setupPlan.LinuxPackages,
			"steps":                    setupPlan.Steps,
		},
		"markers": discovery.Markers,
		"distro":  discovery.Distro,
	}))
}

func (p *BuildProvider) triggerBuild(c *gin.Context) {
	dir, err := p.resolveDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("resolve_failed", err.Error()))
		return
	}

	request, err := decodeBuildRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail("invalid_request", err.Error()))
		return
	}
	archiveOutput, checksumOutput := resolveBuildOutputs(request)

	hasBuildConfig := build.ConfigExists(p.medium, dir)
	var cfg *build.BuildConfig
	if hasBuildConfig {
		cfg, err = build.LoadConfig(p.medium, dir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, api.Fail("config_load_failed", err.Error()))
			return
		}
	}

	projectTypes, err := build.Discover(p.medium, dir)
	if err != nil || len(projectTypes) == 0 {
		c.JSON(http.StatusBadRequest, api.Fail("no_project", "no buildable project detected"))
		return
	}
	for _, projectType := range projectTypes {
		if _, err := providerGetBuilder(projectType); err != nil {
			c.JSON(http.StatusBadRequest, api.Fail("unsupported_type", err.Error()))
			return
		}
	}

	pipeline := &build.Pipeline{
		FS: p.medium,
		ResolveBuilder: func(projectType build.ProjectType) (build.Builder, error) {
			return providerGetBuilder(projectType)
		},
		ResolveVersion: func(ctx context.Context, projectDir string) (string, error) {
			version, err := providerDetermineVersion(ctx, projectDir)
			if err != nil {
				return "dev", nil
			}
			return version, nil
		},
	}
	plan, err := pipeline.Plan(c.Request.Context(), build.PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: cfg,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("build_prepare_failed", err.Error()))
		return
	}

	projectTypeNames := make([]string, 0, len(plan.ProjectTypes))
	for _, projectType := range plan.ProjectTypes {
		projectTypeNames = append(projectTypeNames, string(projectType))
	}

	p.emitEvent("build.started", map[string]any{
		"project_type":  string(plan.ProjectType),
		"project_types": projectTypeNames,
		"targets":       plan.Targets,
		"version":       plan.Version,
	})

	result, err := pipeline.Run(c.Request.Context(), plan)
	if err != nil {
		p.emitEvent("build.failed", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, api.Fail("build_failed", err.Error()))
		return
	}
	artifacts := result.Artifacts

	signCfg := plan.BuildConfig.Sign
	if signCfg.Enabled && (runtime.GOOS == "darwin" || runtime.GOOS == "windows") {
		signingArtifacts := make([]signing.Artifact, len(artifacts))
		for i, artifact := range artifacts {
			signingArtifacts[i] = signing.Artifact{
				Path: artifact.Path,
				OS:   artifact.OS,
				Arch: artifact.Arch,
			}
		}

		if err := providerSignBinaries(c.Request.Context(), p.medium, signCfg, signingArtifacts); err != nil {
			p.emitEvent("build.failed", map[string]any{"error": err.Error()})
			c.JSON(http.StatusInternalServerError, api.Fail("sign_failed", err.Error()))
			return
		}

		if runtime.GOOS == "darwin" && signCfg.MacOS.Notarize {
			if err := providerNotarizeBinaries(c.Request.Context(), p.medium, signCfg, signingArtifacts); err != nil {
				p.emitEvent("build.failed", map[string]any{"error": err.Error()})
				c.JSON(http.StatusInternalServerError, api.Fail("notarize_failed", err.Error()))
				return
			}
		}
	}

	finalArtifacts := append([]build.Artifact(nil), artifacts...)
	response := map[string]any{
		"artifacts":     finalArtifacts,
		"project_type":  string(plan.ProjectType),
		"project_types": projectTypeNames,
		"version":       plan.Version,
	}

	if archiveOutput {
		archiveFormat, err := build.ParseArchiveFormat(plan.BuildConfig.Build.ArchiveFormat)
		if err != nil {
			p.emitEvent("build.failed", map[string]any{"error": err.Error()})
			c.JSON(http.StatusBadRequest, api.Fail("archive_format_invalid", err.Error()))
			return
		}

		finalArtifacts, err = build.ArchiveAllWithFormat(p.medium, finalArtifacts, archiveFormat)
		if err != nil {
			p.emitEvent("build.failed", map[string]any{"error": err.Error()})
			c.JSON(http.StatusInternalServerError, api.Fail("archive_failed", err.Error()))
			return
		}

		response["archive_format"] = string(archiveFormat)
	}

	if checksumOutput {
		checksummed, err := build.ChecksumAll(p.medium, finalArtifacts)
		if err != nil {
			p.emitEvent("build.failed", map[string]any{"error": err.Error()})
			c.JSON(http.StatusInternalServerError, api.Fail("checksum_failed", err.Error()))
			return
		}

		checksumPath := ax.Join(plan.OutputDir, "CHECKSUMS.txt")
		if err := build.WriteChecksumFile(p.medium, checksummed, checksumPath); err != nil {
			p.emitEvent("build.failed", map[string]any{"error": err.Error()})
			c.JSON(http.StatusInternalServerError, api.Fail("checksum_write_failed", err.Error()))
			return
		}

		if signCfg.Enabled {
			if err := providerSignChecksums(c.Request.Context(), p.medium, signCfg, checksumPath); err != nil {
				p.emitEvent("build.failed", map[string]any{"error": err.Error()})
				c.JSON(http.StatusInternalServerError, api.Fail("checksum_sign_failed", err.Error()))
				return
			}
		}

		finalArtifacts = checksummed
		response["checksum_file"] = checksumPath
	}

	p.emitEvent("build.complete", map[string]any{
		"artifact_count": len(finalArtifacts),
		"project_types":  projectTypeNames,
		"version":        plan.Version,
	})

	response["artifacts"] = finalArtifacts
	c.JSON(http.StatusOK, api.OK(response))
}

func decodeBuildRequest(c *gin.Context) (buildRequest, error) {
	var request buildRequest
	if c == nil || c.Request == nil || c.Request.Body == nil || c.Request.ContentLength == 0 {
		return request, nil
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
		return buildRequest{}, err
	}
	return request, nil
}

func resolveBuildOutputs(request buildRequest) (bool, bool) {
	archiveOutput := false
	checksumOutput := false

	archiveSet := request.Archive != nil
	if archiveSet {
		archiveOutput = *request.Archive
	}

	checksumSet := request.Checksum != nil
	if checksumSet {
		checksumOutput = *request.Checksum
	}

	if request.Package != nil {
		if !archiveSet {
			archiveOutput = *request.Package
		}
		if !checksumSet {
			checksumOutput = *request.Package
		}
	}

	return archiveOutput, checksumOutput
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

	artifacts, err := p.collectArtifacts(distDir, distDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("list_failed", err.Error()))
		return
	}

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Name < artifacts[j].Name
	})

	if artifacts == nil {
		artifacts = []artifactInfo{}
	}

	c.JSON(http.StatusOK, api.OK(map[string]any{
		"artifacts": artifacts,
		"exists":    true,
	}))
}

func (p *BuildProvider) collectArtifacts(distDir, currentDir string) ([]artifactInfo, error) {
	entries, err := p.medium.List(currentDir)
	if err != nil {
		return nil, err
	}

	var artifacts []artifactInfo
	for _, entry := range entries {
		path := ax.Join(currentDir, entry.Name())
		if entry.IsDir() {
			nested, err := p.collectArtifacts(distDir, path)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, nested...)
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		name, err := ax.Rel(distDir, path)
		if err != nil {
			name = entry.Name()
		}

		artifacts = append(artifacts, artifactInfo{
			Name: name,
			Path: path,
			Size: info.Size(),
		})
	}

	return artifacts, nil
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

	cfg, err := providerLoadReleaseConfig(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("config_load_failed", err.Error()))
		return
	}

	// Parse optional dry_run parameter
	dryRun := c.Query("dry_run") == "true"

	p.emitEvent("release.started", map[string]any{
		"dry_run": dryRun,
	})

	rel, err := providerRunRelease(c.Request.Context(), cfg, dryRun)
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
// request := ReleaseWorkflowRequest{Path: "ci/release.yml"}                 // writes ./ci/release.yml
// request := ReleaseWorkflowRequest{WorkflowOutputPath: "ops/release.yml"} // writes ./ops/release.yml
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

// resolveWorkflowTargetPath merges the workflow path and workflow output aliases into one final target path.
//
// request := ReleaseWorkflowRequest{Path: "ci/release.yml"}
// path, err := request.resolveWorkflowTargetPath("/tmp/project", io.Local)
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

// resolveWorkflowPath("ci/release.yml") and resolveWorkflowPath("workflow-path") both resolve to the same file path.
//
// request := ReleaseWorkflowRequest{WorkflowPath: "ci/release.yml"}
// workflowPath, err := request.resolveWorkflowPath("/tmp/project", io.Local)
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

// resolveOutputPath("ci/release.yml") and resolveOutputPath("workflow-output-path") both resolve to the same file path.
//
// request := ReleaseWorkflowRequest{WorkflowOutputPath: "ci/release.yml"}
// outputPath, err := request.resolveOutputPath("/tmp/project")
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

	var request ReleaseWorkflowRequest
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail("invalid_request", err.Error()))
		return
	}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &request); err != nil {
			c.JSON(http.StatusBadRequest, api.Fail("invalid_request", err.Error()))
			return
		}
	}

	workflowPath, err := request.resolveWorkflowTargetPath(dir, p.medium)
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

	sdkCfg, err := sdkcfg.LoadProjectConfig(p.medium, dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail("config_load_failed", err.Error()))
		return
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

func (p *BuildProvider) streamEvents(c *gin.Context) {
	if p.hub == nil {
		c.JSON(http.StatusServiceUnavailable, api.Fail("event_hub_unavailable", "build event stream is unavailable"))
		return
	}

	p.hub.HandleWebSocket(c.Writer, c.Request)
}

// -- Internal Helpers ---------------------------------------------------------

// getBuilder returns the appropriate builder for the project type.
func getBuilder(projectType build.ProjectType) (build.Builder, error) {
	builder, err := builders.ResolveBuilder(projectType)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	return builder, nil
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
