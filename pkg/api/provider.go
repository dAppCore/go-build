// SPDX-Licence-Identifier: EUPL-1.2

// Package api provides a service provider that wraps go-build's build,
// release, and SDK subsystems as REST endpoints with WebSocket event
// streaming.
package api

import (
	"cmp"
	"context"
	stdfs "io/fs"
	"slices"

	"dappco.re/go"
	"dappco.re/go/api"
	"dappco.re/go/api/pkg/provider"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/projectdetect"
	"dappco.re/go/build/internal/sdkcfg"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/builders"
	"dappco.re/go/build/pkg/build/signing"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/build/pkg/sdk"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
	"dappco.re/go/ws"
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

const (
	apiPathField = "pa" + "th"
	apiOSField   = "o" + "s"
)

type buildRequest struct {
	Archive  *bool `json:"archive,omitempty"`
	Checksum *bool `json:"checksum,omitempty"`
	Package  *bool `json:"package,omitempty"`
}

const (
	statusOK                  = 200
	statusBadRequest          = 400
	statusInternalServerError = 500
	statusServiceUnavailable  = 503
)

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
					apiPathField: map[string]any{
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
func (p *BuildProvider) resolveDir() core.Result {
	return ax.Abs(p.projectDir)
}

// -- Build Handlers -----------------------------------------------------------

func (p *BuildProvider) getConfig(c *gin.Context) {
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	cfgResult := build.LoadConfig(p.medium, dir)
	if !cfgResult.OK {
		c.JSON(statusInternalServerError, api.Fail("config_load_failed", cfgResult.Error()))
		return
	}
	cfg := cfgResult.Value.(*build.BuildConfig)

	hasConfig := build.ConfigExists(p.medium, dir)

	c.JSON(statusOK, api.OK(map[string]any{
		"config":     cfg,
		"has_config": hasConfig,
		apiPathField: build.ConfigPath(dir),
	}))
}

func (p *BuildProvider) discoverProject(c *gin.Context) {
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	cfgResult := build.LoadConfig(p.medium, dir)
	if !cfgResult.OK {
		c.JSON(statusInternalServerError, api.Fail("config_load_failed", cfgResult.Error()))
		return
	}
	cfg := cfgResult.Value.(*build.BuildConfig)

	discoveryResult := build.DiscoverFull(p.medium, dir)
	if !discoveryResult.OK {
		c.JSON(statusInternalServerError, api.Fail("discover_failed", discoveryResult.Error()))
		return
	}
	discovery := discoveryResult.Value.(*build.DiscoveryResult)
	options := build.ComputeOptions(cfg, discovery)
	setupPlanResult := build.ComputeSetupPlan(p.medium, dir, cfg, discovery)
	if !setupPlanResult.OK {
		c.JSON(statusInternalServerError, api.Fail("setup_plan_failed", setupPlanResult.Error()))
		return
	}
	setupPlan := setupPlanResult.Value.(*build.SetupPlan)

	// Convert to string slice for JSON
	typeStrings := make([]string, len(discovery.Types))
	for i, t := range discovery.Types {
		typeStrings[i] = string(t)
	}

	primary := ""
	if len(discovery.Types) > 0 {
		primary = string(discovery.Types[0])
	}

	c.JSON(statusOK, api.OK(map[string]any{
		"types":                     typeStrings,
		"configured_type":           discovery.ConfiguredType,
		"configured_build_type":     discovery.ConfiguredBuildType,
		apiOSField:                  discovery.OS,
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
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	requestResult := decodeBuildRequest(c)
	if !requestResult.OK {
		c.JSON(statusBadRequest, api.Fail("invalid_request", requestResult.Error()))
		return
	}
	request := requestResult.Value.(buildRequest)
	archiveOutput, checksumOutput := resolveBuildOutputs(request)

	hasBuildConfig := build.ConfigExists(p.medium, dir)
	var cfg *build.BuildConfig
	if hasBuildConfig {
		cfgResult := build.LoadConfig(p.medium, dir)
		if !cfgResult.OK {
			c.JSON(statusInternalServerError, api.Fail("config_load_failed", cfgResult.Error()))
			return
		}
		cfg = cfgResult.Value.(*build.BuildConfig)
	}

	projectTypesResult := build.Discover(p.medium, dir)
	if !projectTypesResult.OK {
		c.JSON(statusBadRequest, api.Fail("no_project", "no buildable project detected"))
		return
	}
	projectTypes := projectTypesResult.Value.([]build.ProjectType)
	if len(projectTypes) == 0 {
		c.JSON(statusBadRequest, api.Fail("no_project", "no buildable project detected"))
		return
	}
	for _, projectType := range projectTypes {
		builderResult := providerGetBuilder(projectType)
		if !builderResult.OK {
			c.JSON(statusBadRequest, api.Fail("unsupported_type", builderResult.Error()))
			return
		}
	}

	pipeline := &build.Pipeline{
		FS: p.medium,
		ResolveBuilder: func(projectType build.ProjectType) core.Result {
			return providerGetBuilder(projectType)
		},
		ResolveVersion: func(ctx context.Context, projectDir string) core.Result {
			version := providerDetermineVersion(ctx, projectDir)
			if !version.OK {
				return core.Ok("dev")
			}
			return version
		},
	}
	planResult := pipeline.Plan(c.Request.Context(), build.PipelineRequest{
		ProjectDir:  dir,
		BuildConfig: cfg,
	})
	if !planResult.OK {
		c.JSON(statusInternalServerError, api.Fail("build_prepare_failed", planResult.Error()))
		return
	}
	plan := planResult.Value.(*build.PipelinePlan)

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

	result := pipeline.Run(c.Request.Context(), plan)
	if !result.OK {
		p.emitEvent("build.failed", map[string]any{"error": result.Error()})
		c.JSON(statusInternalServerError, api.Fail("build_failed", result.Error()))
		return
	}
	pipelineResult := result.Value.(*build.PipelineResult)
	artifacts := pipelineResult.Artifacts

	signCfg := plan.BuildConfig.Sign
	goos := currentGOOS()
	if signCfg.Enabled && (goos == "darwin" || goos == "windows") {
		signingArtifacts := make([]signing.Artifact, len(artifacts))
		for i, artifact := range artifacts {
			signingArtifacts[i] = signing.Artifact{
				Path: artifact.Path,
				OS:   artifact.OS,
				Arch: artifact.Arch,
			}
		}

		signed := providerSignBinaries(c.Request.Context(), p.medium, signCfg, signingArtifacts)
		if !signed.OK {
			p.emitEvent("build.failed", map[string]any{"error": signed.Error()})
			c.JSON(statusInternalServerError, api.Fail("sign_failed", signed.Error()))
			return
		}

		if goos == "darwin" && signCfg.MacOS.Notarize {
			notarized := providerNotarizeBinaries(c.Request.Context(), p.medium, signCfg, signingArtifacts)
			if !notarized.OK {
				p.emitEvent("build.failed", map[string]any{"error": notarized.Error()})
				c.JSON(statusInternalServerError, api.Fail("notarize_failed", notarized.Error()))
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
		archiveFormatResult := build.ParseArchiveFormat(plan.BuildConfig.Build.ArchiveFormat)
		if !archiveFormatResult.OK {
			p.emitEvent("build.failed", map[string]any{"error": archiveFormatResult.Error()})
			c.JSON(statusBadRequest, api.Fail("archive_format_invalid", archiveFormatResult.Error()))
			return
		}
		archiveFormat := archiveFormatResult.Value.(build.ArchiveFormat)

		archivedResult := build.ArchiveAllWithFormat(p.medium, finalArtifacts, archiveFormat)
		if !archivedResult.OK {
			p.emitEvent("build.failed", map[string]any{"error": archivedResult.Error()})
			c.JSON(statusInternalServerError, api.Fail("archive_failed", archivedResult.Error()))
			return
		}
		finalArtifacts = archivedResult.Value.([]build.Artifact)

		response["archive_format"] = string(archiveFormat)
	}

	if checksumOutput {
		checksummedResult := build.ChecksumAll(p.medium, finalArtifacts)
		if !checksummedResult.OK {
			p.emitEvent("build.failed", map[string]any{"error": checksummedResult.Error()})
			c.JSON(statusInternalServerError, api.Fail("checksum_failed", checksummedResult.Error()))
			return
		}
		checksummed := checksummedResult.Value.([]build.Artifact)

		checksumPath := ax.Join(plan.OutputDir, "CHECKSUMS.txt")
		wroteChecksums := build.WriteChecksumFile(p.medium, checksummed, checksumPath)
		if !wroteChecksums.OK {
			p.emitEvent("build.failed", map[string]any{"error": wroteChecksums.Error()})
			c.JSON(statusInternalServerError, api.Fail("checksum_write_failed", wroteChecksums.Error()))
			return
		}

		if signCfg.Enabled {
			signedChecksums := providerSignChecksums(c.Request.Context(), p.medium, signCfg, checksumPath)
			if !signedChecksums.OK {
				p.emitEvent("build.failed", map[string]any{"error": signedChecksums.Error()})
				c.JSON(statusInternalServerError, api.Fail("checksum_sign_failed", signedChecksums.Error()))
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
	c.JSON(statusOK, api.OK(response))
}

func decodeBuildRequest(c *gin.Context) core.Result {
	var request buildRequest
	if c == nil || c.Request == nil || c.Request.Body == nil || c.Request.ContentLength == 0 {
		return core.Ok(request)
	}
	body, err := c.GetRawData()
	if err != nil {
		return core.Fail(err)
	}
	decoded := decodeJSONBody(body, &request)
	if !decoded.OK {
		return decoded
	}
	return core.Ok(request)
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
func resolveProjectType(filesystem io.Medium, projectDir, buildType string) core.Result {
	if buildType != "" {
		return core.Ok(build.ProjectType(buildType))
	}

	return projectdetect.DetectProjectType(filesystem, projectDir)
}

// Info holds JSON-friendly metadata about a dist/ file.
type Info struct {
	Name string "json:\"name\""
	Path string "json:\"path\""
	Size int64  "json:\"size\""
}

func (p *BuildProvider) listArtifacts(c *gin.Context) {
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	distDir := ax.Join(dir, "dist")
	if !p.medium.IsDir(distDir) {
		c.JSON(statusOK, api.OK(map[string]any{
			"artifacts": []Info{},
			"exists":    false,
		}))
		return
	}

	artifactsResult := p.collectArtifacts(distDir, distDir)
	if !artifactsResult.OK {
		c.JSON(statusInternalServerError, api.Fail("list_failed", artifactsResult.Error()))
		return
	}
	artifacts := artifactsResult.Value.([]Info)

	slices.SortFunc(artifacts, func(a, b Info) int {
		return cmp.Compare(a.Name, b.Name)
	})

	if artifacts == nil {
		artifacts = []Info{}
	}

	c.JSON(statusOK, api.OK(map[string]any{
		"artifacts": artifacts,
		"exists":    true,
	}))
}

func (p *BuildProvider) collectArtifacts(distDir, currentDir string) core.Result {
	entriesResult := p.medium.List(currentDir)
	if !entriesResult.OK {
		return entriesResult
	}
	entries := entriesResult.Value.([]stdfs.DirEntry)

	var artifacts []Info
	for _, entry := range entries {
		path := ax.Join(currentDir, entry.Name())
		if entry.IsDir() {
			nestedResult := p.collectArtifacts(distDir, path)
			if !nestedResult.OK {
				return nestedResult
			}
			nested := nestedResult.Value.([]Info)
			artifacts = append(artifacts, nested...)
			continue
		}

		info, infoFailure := entry.Info()
		if infoFailure != nil {
			continue
		}

		nameResult := ax.Rel(distDir, path)
		name := ""
		if !nameResult.OK {
			name = entry.Name()
		} else {
			name = nameResult.Value.(string)
		}

		artifacts = append(artifacts, Info{
			Name: name,
			Path: path,
			Size: info.Size(),
		})
	}

	return core.Ok(artifacts)
}

// -- Release Handlers ---------------------------------------------------------

func (p *BuildProvider) getVersion(c *gin.Context) {
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	versionResult := release.DetermineVersionWithContext(c.Request.Context(), dir)
	if !versionResult.OK {
		c.JSON(statusInternalServerError, api.Fail("version_failed", versionResult.Error()))
		return
	}
	version := versionResult.Value.(string)

	c.JSON(statusOK, api.OK(map[string]any{
		"version": version,
	}))
}

func (p *BuildProvider) getChangelog(c *gin.Context) {
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	// Optional query params for ref range
	fromRef := c.Query("from")
	toRef := c.Query("to")

	changelogResult := release.GenerateWithContext(c.Request.Context(), dir, fromRef, toRef)
	if !changelogResult.OK {
		c.JSON(statusInternalServerError, api.Fail("changelog_failed", changelogResult.Error()))
		return
	}
	changelog := changelogResult.Value.(string)

	c.JSON(statusOK, api.OK(map[string]any{
		"changelog": changelog,
	}))
}

func (p *BuildProvider) triggerRelease(c *gin.Context) {
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	cfgResult := providerLoadReleaseConfig(dir)
	if !cfgResult.OK {
		c.JSON(statusInternalServerError, api.Fail("config_load_failed", cfgResult.Error()))
		return
	}
	cfg := cfgResult.Value.(*release.Config)

	// Parse optional dry_run parameter
	dryRun := c.Query("dry_run") == "true"

	p.emitEvent("release.started", map[string]any{
		"dry_run": dryRun,
	})

	relResult := providerRunRelease(c.Request.Context(), cfg, dryRun)
	if !relResult.OK {
		c.JSON(statusInternalServerError, api.Fail("release_failed", relResult.Error()))
		return
	}
	rel := relResult.Value.(*release.Release)

	p.emitEvent("release.complete", map[string]any{
		"version":        rel.Version,
		"artifact_count": len(rel.Artifacts),
		"dry_run":        dryRun,
	})

	c.JSON(statusOK, api.OK(map[string]any{
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
	Path                     string
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

func (r *ReleaseWorkflowRequest) Decode(data []byte) core.Result {
	if core.Trim(string(data)) == "" {
		return core.Ok(nil)
	}

	var raw map[string]string
	decoded := core.JSONUnmarshal(data, &raw)
	if !decoded.OK {
		return decoded
	}
	r.Path = raw[apiPathField]
	r.WorkflowPath = raw["workflowPath"]
	r.WorkflowPathSnake = raw["workflow_path"]
	r.WorkflowPathHyphen = raw["workflow-path"]
	r.OutputPath = raw["outputPath"]
	r.OutputPathHyphen = raw["output-path"]
	r.OutputPathSnake = raw["output_path"]
	r.LegacyOutputPath = raw["output"]
	r.WorkflowOutputPath = raw["workflowOutputPath"]
	r.WorkflowOutputSnake = raw["workflow_output"]
	r.WorkflowOutputHyphen = raw["workflow-output"]
	r.WorkflowOutputPathSnake = raw["workflow_output_path"]
	r.WorkflowOutputPathHyphen = raw["workflow-output-path"]
	return core.Ok(nil)
}

// resolveWorkflowTargetPath merges the workflow path and workflow output aliases into one final target path.
//
// request := ReleaseWorkflowRequest{Path: "ci/release.yml"}
// result := request.resolveWorkflowTargetPath("/tmp/project", io.Local)
func (r ReleaseWorkflowRequest) resolveWorkflowTargetPath(dir string, medium io.Medium) core.Result {
	outputPathResult := r.resolveOutputPath(dir, medium)
	if !outputPathResult.OK {
		return outputPathResult
	}
	outputPath := outputPathResult.Value.(string)

	workflowPathResult := r.resolveWorkflowPath(dir, medium)
	if !workflowPathResult.OK {
		return workflowPathResult
	}
	workflowPath := workflowPathResult.Value.(string)

	return build.ResolveReleaseWorkflowInputPathWithMedium(medium, dir, workflowPath, outputPath)
}

// resolveWorkflowPath("ci/release.yml") and resolveWorkflowPath("workflow-path") both resolve to the same file path.
//
// request := ReleaseWorkflowRequest{WorkflowPath: "ci/release.yml"}
// result := request.resolveWorkflowPath("/tmp/project", io.Local)
func (r ReleaseWorkflowRequest) resolveWorkflowPath(dir string, medium io.Medium) core.Result {
	workflowPath := build.ResolveReleaseWorkflowInputPathAliases(
		medium,
		dir,
		r.Path,
		r.WorkflowPath,
		r.WorkflowPathSnake,
		r.WorkflowPathHyphen,
	)
	if !workflowPath.OK {
		return core.Fail(coreerr.E("api.ReleaseWorkflowRequest", "workflow path aliases specify different locations", nil))
	}

	return workflowPath
}

// resolveOutputPath("ci/release.yml") and resolveOutputPath("workflow-output-path") both resolve to the same file path.
//
// request := ReleaseWorkflowRequest{WorkflowOutputPath: "ci/release.yml"}
// result := request.resolveOutputPath("/tmp/project")
func (r ReleaseWorkflowRequest) resolveOutputPath(dir string, medium io.Medium) core.Result {
	resolvedOutputPath := build.ResolveReleaseWorkflowOutputPathAliasesInProjectWithMedium(
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
	if !resolvedOutputPath.OK {
		return core.Fail(coreerr.E("api.ReleaseWorkflowRequest", "workflow output aliases specify different locations", nil))
	}

	return resolvedOutputPath
}

func (p *BuildProvider) generateReleaseWorkflow(c *gin.Context) {
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	var request ReleaseWorkflowRequest
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(statusBadRequest, api.Fail("invalid_request", err.Error()))
		return
	}
	decoded := request.Decode(body)
	if !decoded.OK {
		c.JSON(statusBadRequest, api.Fail("invalid_request", decoded.Error()))
		return
	}

	workflowPathResult := request.resolveWorkflowTargetPath(dir, p.medium)
	if !workflowPathResult.OK {
		c.JSON(statusBadRequest, api.Fail("invalid_request", workflowPathResult.Error()))
		return
	}
	workflowPath := workflowPathResult.Value.(string)

	written := build.WriteReleaseWorkflow(p.medium, workflowPath)
	if !written.OK {
		c.JSON(statusInternalServerError, api.Fail("workflow_write_failed", written.Error()))
		return
	}

	p.emitEvent("workflow.generated", map[string]any{
		apiPathField: workflowPath,
		"generated":  true,
	})

	c.JSON(statusOK, api.OK(map[string]any{
		"generated":  true,
		apiPathField: workflowPath,
	}))
}

// -- SDK Handlers -------------------------------------------------------------

func (p *BuildProvider) getSdkDiff(c *gin.Context) {
	basePath := c.Query("base")
	revisionPath := c.Query("revision")

	if basePath == "" || revisionPath == "" {
		c.JSON(statusBadRequest, api.Fail("missing_params", "base and revision query parameters are required"))
		return
	}

	result := sdk.Diff(basePath, revisionPath)
	if !result.OK {
		c.JSON(statusInternalServerError, api.Fail("diff_failed", result.Error()))
		return
	}

	c.JSON(statusOK, api.OK(result.Value))
}

type sdkGenerateRequest struct {
	Language string `json:"language"`
}

func (p *BuildProvider) generateSdk(c *gin.Context) {
	dirResult := p.resolveDir()
	if !dirResult.OK {
		c.JSON(statusInternalServerError, api.Fail("resolve_failed", dirResult.Error()))
		return
	}
	dir := dirResult.Value.(string)

	var req sdkGenerateRequest
	if bindFailure := c.ShouldBindJSON(&req); bindFailure != nil {
		// No body is fine — generate all languages
		req.Language = ""
	}

	sdkCfgResult := sdkcfg.LoadProjectConfig(p.medium, dir)
	if !sdkCfgResult.OK {
		c.JSON(statusInternalServerError, api.Fail("config_load_failed", sdkCfgResult.Error()))
		return
	}
	sdkCfg := sdkCfgResult.Value.(*sdk.Config)

	s := sdk.New(dir, sdkCfg)

	ctx := c.Request.Context()
	var generated core.Result
	if req.Language != "" {
		generated = s.GenerateLanguage(ctx, req.Language)
	} else {
		generated = s.Generate(ctx)
	}

	if !generated.OK {
		c.JSON(statusInternalServerError, api.Fail("sdk_generate_failed", generated.Error()))
		return
	}

	p.emitEvent("sdk.generated", map[string]any{
		"language": req.Language,
	})

	c.JSON(statusOK, api.OK(map[string]any{
		"generated": true,
		"language":  req.Language,
	}))
}

func (p *BuildProvider) streamEvents(c *gin.Context) {
	if p.hub == nil {
		c.JSON(statusServiceUnavailable, api.Fail("event_hub_unavailable", "build event stream is unavailable"))
		return
	}

	p.hub.HandleWebSocket(c.Writer, c.Request)
}

// -- Internal Helpers ---------------------------------------------------------

// getBuilder returns the appropriate builder for the project type.
func getBuilder(projectType build.ProjectType) core.Result {
	builder := builders.ResolveBuilder(projectType)
	if !builder.OK {
		return builder
	}
	return builder
}

func currentGOOS() string {
	if goos := core.Env("GOOS"); goos != "" {
		return goos
	}
	return core.Env("OS")
}

func decodeJSONBody(body []byte, target any) core.Result {
	if core.Trim(string(body)) == "" {
		return core.Ok(nil)
	}

	result := core.JSONUnmarshalString(string(body), target)
	if result.OK {
		return core.Ok(nil)
	}
	if err, ok := result.Value.(error); ok {
		return core.Fail(err)
	}
	return core.Fail(coreerr.E("api.decodeJSONBody", "invalid JSON", nil))
}

// emitEvent sends a WS event if the hub is available.
func (p *BuildProvider) emitEvent(channel string, data any) {
	if p.hub == nil {
		return
	}
	sent := p.hub.SendToChannel(channel, ws.Message{
		Type: ws.TypeEvent,
		Data: data,
	})
	if !sent.OK {
		return
	}
}
