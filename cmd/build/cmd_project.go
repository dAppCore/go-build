// cmd_project.go implements project build orchestration and auto-detection.
//
// runProjectBuild(ProjectBuildRequest{
//   BuildType: "go",
//   TargetsFlag: "linux/amd64,darwin/arm64",
//   ArchiveOutput: true,
// }) executes end-to-end build/sign/archive/checksum flow for the selected project.

package buildcmd

import (
	"context"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/builders"
	"dappco.re/go/build/pkg/build/signing"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	"dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

var getProjectBuildWorkingDir = ax.Getwd

// ProjectBuildRequest groups the inputs for the main `core build` command.
//
//	req := ProjectBuildRequest{
//		Context:     cmd.Context(),
//		BuildType:   "go",
//		TargetsFlag: "linux/amd64,linux/arm64",
//	}
type ProjectBuildRequest struct {
	Context           context.Context
	BuildType         string
	Version           string
	CIMode            bool
	TargetsFlag       string
	OutputDir         string
	BuildName         string
	BuildTagsFlag     string
	Obfuscate         bool
	ObfuscateSet      bool
	NSIS              bool
	NSISSet           bool
	WebView2          string
	WebView2Set       bool
	DenoBuild         string
	DenoBuildSet      bool
	NpmBuild          string
	NpmBuildSet       bool
	BuildCache        bool
	BuildCacheSet     bool
	ArchiveOutput     bool
	ArchiveOutputSet  bool
	ChecksumOutput    bool
	ChecksumOutputSet bool
	PackageSet        bool
	ArchiveFormat     string
	ConfigPath        string
	Format            string
	Push              bool
	ImageName         string
	Sign              bool
	SignSet           bool
	NoSign            bool
	Notarize          bool
	Verbose           bool
}

// runProjectBuild handles the main `core build` command with auto-detection.
//
//	runProjectBuild(ProjectBuildRequest{
//	  BuildType: "node",
//	  TargetsFlag: "linux/amd64",
//	  ArchiveOutput: true,
//	  ChecksumOutput: true,
//	  Format: "gz",
//	})
func runProjectBuild(req ProjectBuildRequest) (result core.Result) {
	if req.CIMode {
		defer func() {
			emitCIErrorAnnotation(result)
		}()
	}

	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}
	// Use local filesystem as the default medium.
	filesystem := io.Local

	// Get current working directory as project root
	projectDirResult := getProjectBuildWorkingDir()
	if !projectDirResult.OK {
		return core.Fail(coreerr.E("build.Run", "failed to get working directory", core.NewError(projectDirResult.Error())))
	}
	projectDir := projectDirResult.Value.(string)

	// PWA builds use the dedicated local web-app pipeline rather than the
	// project-type builder registry.
	if req.BuildType == "pwa" {
		return runLocalPwaBuild(ctx, projectDir)
	}

	if shouldUseGoBuildPassthrough(filesystem, projectDir, req) {
		return runGoBuildPassthrough(ctx, projectDir, req)
	}

	// Load configuration from .core/build.yaml (or defaults)
	var buildConfig *build.BuildConfig
	configPath := req.ConfigPath
	if configPath != "" {
		if !ax.IsAbs(configPath) {
			configPath = ax.Join(projectDir, configPath)
		}
		if !filesystem.Exists(configPath) {
			return core.Fail(coreerr.E("build.Run", "build config not found: "+configPath, nil))
		}
		configResult := build.LoadConfigAtPath(filesystem, configPath)
		if !configResult.OK {
			return core.Fail(coreerr.E("build.Run", "failed to load config", core.NewError(configResult.Error())))
		}
		buildConfig = configResult.Value.(*build.BuildConfig)
	} else {
		configResult := build.LoadConfig(filesystem, projectDir)
		if !configResult.OK {
			return core.Fail(coreerr.E("build.Run", "failed to load config", core.NewError(configResult.Error())))
		}
		buildConfig = configResult.Value.(*build.BuildConfig)
	}

	if buildConfig.Build.Type == "pwa" {
		return runLocalPwaBuild(ctx, projectDir)
	}

	applyProjectBuildOverrides(buildConfig, req)

	// Determine targets
	var buildTargets []build.Target
	if req.TargetsFlag != "" {
		// Parse from command line
		targetsResult := parseTargets(req.TargetsFlag)
		if !targetsResult.OK {
			return targetsResult
		}
		buildTargets = targetsResult.Value.([]build.Target)
	} else if len(buildConfig.Targets) > 0 {
		// Use config targets
		buildTargets = buildConfig.ToTargets()
	} else {
		// Fall back to current OS/arch
		buildTargets = []build.Target{
			{OS: runtime.GOOS, Arch: runtime.GOARCH},
		}
	}

	pipeline := &build.Pipeline{
		FS:             filesystem,
		ResolveBuilder: getBuilder,
		ResolveVersion: resolveBuildVersion,
	}
	planResult := pipeline.Plan(ctx, build.PipelineRequest{
		ProjectDir:  projectDir,
		BuildConfig: buildConfig,
		BuildType:   req.BuildType,
		Version:     req.Version,
		OutputDir:   req.OutputDir,
		BuildName:   req.BuildName,
		Targets:     buildTargets,
		Push:        req.Push,
		ImageName:   req.ImageName,
	})
	if !planResult.OK {
		return planResult
	}
	plan := planResult.Value.(*build.PipelinePlan)

	// Print build info (verbose mode only)
	if req.Verbose && !req.CIMode {
		cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.label.build")), i18n.T("cmd.build.building_project"))
		cli.Print("  %s %s\n", i18n.T("cmd.build.label.type"), buildTargetStyle.Render(formatProjectTypes(plan.ProjectTypes)))
		cli.Print("  %s %s\n", i18n.T("cmd.build.label.output"), buildTargetStyle.Render(plan.OutputDir))
		cli.Print("  %s %s\n", i18n.T("cmd.build.label.binary"), buildTargetStyle.Render(plan.BuildName))
		cli.Print("  %s %s\n", i18n.T("cmd.build.label.targets"), buildTargetStyle.Render(formatTargets(plan.Targets)))
		cli.Blank()
	}

	// Parse formats for LinuxKit
	if req.Format != "" {
		plan.RuntimeConfig.Formats = core.Split(req.Format, ",")
	}

	// Execute build
	pipelineResultValue := pipeline.Run(ctx, plan)
	if !pipelineResultValue.OK {
		if !req.CIMode {
			cli.Print("%s %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), pipelineResultValue.Error())
		}
		return pipelineResultValue
	}
	pipelineResult := pipelineResultValue.Value.(*build.PipelineResult)
	artifacts := pipelineResult.Artifacts
	if req.CIMode {
		rewritten := rewriteArtifactsForCI(filesystem, plan.BuildName, artifacts)
		if !rewritten.OK {
			return rewritten
		}
		artifacts = rewritten.Value.([]build.Artifact)
	}

	if req.Verbose && !req.CIMode {
		cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("common.label.success")), i18n.T("cmd.build.built_artifacts", map[string]any{"Count": len(artifacts)}))
		cli.Blank()
		for _, artifact := range artifacts {
			relPath := artifact.Path
			relPathResult := ax.Rel(projectDir, artifact.Path)
			if relPathResult.OK {
				relPath = relPathResult.Value.(string)
			}
			cli.Print("  %s %s %s\n",
				buildSuccessStyle.Render("*"),
				buildTargetStyle.Render(relPath),
				buildDimStyle.Render(core.Sprintf("(%s/%s)", artifact.OS, artifact.Arch)),
			)
		}
	}

	// Sign binaries if enabled.
	signCfg := resolveBuildSignConfig(plan.BuildConfig.Sign, req)

	if signCfg.Enabled && (runtime.GOOS == "darwin" || runtime.GOOS == "windows") {
		if req.Verbose && !req.CIMode {
			cli.Blank()
			cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.label.sign")), i18n.T("cmd.build.signing_binaries"))
		}

		// Convert build.Artifact to signing.Artifact
		signingArtifacts := make([]signing.Artifact, len(artifacts))
		for i, a := range artifacts {
			signingArtifacts[i] = signing.Artifact{Path: a.Path, OS: a.OS, Arch: a.Arch}
		}

		signed := signing.SignBinaries(ctx, filesystem, signCfg, signingArtifacts)
		if !signed.OK {
			if !req.CIMode {
				cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.signing_failed"), signed.Error())
			}
			return signed
		}

		if runtime.GOOS == "darwin" && signCfg.MacOS.Notarize {
			notarized := signing.NotarizeBinaries(ctx, filesystem, signCfg, signingArtifacts)
			if !notarized.OK {
				if !req.CIMode {
					cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.notarization_failed"), notarized.Error())
				}
				return notarized
			}
		}
	}

	// Archive artifacts if enabled
	var archivedArtifacts []build.Artifact
	if req.ArchiveOutput && len(artifacts) > 0 {
		if req.Verbose && !req.CIMode {
			cli.Blank()
			cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.label.archive")), i18n.T("cmd.build.creating_archives"))
		}

		archiveFormatResult := resolveArchiveFormat(buildConfig.Build.ArchiveFormat, req.ArchiveFormat)
		if !archiveFormatResult.OK {
			return archiveFormatResult
		}
		archiveFormatValue := archiveFormatResult.Value.(build.ArchiveFormat)

		archivedArtifactsResult := build.ArchiveAllWithFormat(filesystem, artifacts, archiveFormatValue)
		if !archivedArtifactsResult.OK {
			if !req.CIMode {
				cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.archive_failed"), archivedArtifactsResult.Error())
			}
			return archivedArtifactsResult
		}
		archivedArtifacts = archivedArtifactsResult.Value.([]build.Artifact)

		if req.Verbose && !req.CIMode {
			for _, artifact := range archivedArtifacts {
				relPath := artifact.Path
				relPathResult := ax.Rel(projectDir, artifact.Path)
				if relPathResult.OK {
					relPath = relPathResult.Value.(string)
				}
				cli.Print("  %s %s %s\n",
					buildSuccessStyle.Render("*"),
					buildTargetStyle.Render(relPath),
					buildDimStyle.Render(core.Sprintf("(%s/%s)", artifact.OS, artifact.Arch)),
				)
			}
		}
	}

	// Compute checksums if enabled
	var checksummedArtifacts []build.Artifact
	if req.ChecksumOutput && len(archivedArtifacts) > 0 {
		checksummed := computeAndWriteChecksums(ctx, filesystem, projectDir, plan.OutputDir, archivedArtifacts, signCfg, req.CIMode, req.Verbose)
		if !checksummed.OK {
			return checksummed
		}
		checksummedArtifacts = checksummed.Value.([]build.Artifact)
	} else if req.ChecksumOutput && len(artifacts) > 0 && !req.ArchiveOutput {
		// Checksum raw binaries if archiving is disabled
		checksummed := computeAndWriteChecksums(ctx, filesystem, projectDir, plan.OutputDir, artifacts, signCfg, req.CIMode, req.Verbose)
		if !checksummed.OK {
			return checksummed
		}
		checksummedArtifacts = checksummed.Value.([]build.Artifact)
	}

	// Output results
	if req.CIMode {
		// Determine which artifacts to output (prefer checksummed > archived > raw).
		outputArtifacts := selectOutputArtifacts(artifacts, archivedArtifacts, checksummedArtifacts)
		metadataWritten := writeArtifactMetadata(filesystem, plan.BuildName, outputArtifacts)
		if !metadataWritten.OK {
			return metadataWritten
		}

		// JSON output for CI
		output := ax.JSONMarshal(outputArtifacts)
		if !output.OK {
			return core.Fail(coreerr.E("build.Run", "failed to marshal artifacts", core.NewError(output.Error())))
		}
		cli.Print("%s\n", output.Value.(string))
	} else if !req.Verbose {
		// Minimal output: just success with artifact count
		cli.Print("%s %s %s\n",
			buildSuccessStyle.Render(i18n.T("common.label.success")),
			i18n.T("cmd.build.built_artifacts", map[string]any{"Count": len(artifacts)}),
			buildDimStyle.Render(core.Sprintf("(%s)", plan.OutputDir)),
		)
	}

	return core.Ok(nil)
}

func resolveBuildSignConfig(base signing.SignConfig, req ProjectBuildRequest) signing.SignConfig {
	signCfg := base

	if req.Notarize {
		signCfg.MacOS.Notarize = true
		if !req.NoSign {
			signCfg.Enabled = true
		}
	}
	if req.NoSign {
		signCfg.Enabled = false
	}

	return signCfg
}

func shouldUseGoBuildPassthrough(filesystem io.Medium, projectDir string, req ProjectBuildRequest) bool {
	if req.ConfigPath != "" || build.ConfigExists(filesystem, projectDir) {
		return false
	}

	if req.BuildType != "" && req.BuildType != string(build.ProjectTypeGo) {
		return false
	}

	if !build.IsGoProject(filesystem, projectDir) {
		return false
	}

	projectTypesResult := build.Discover(filesystem, projectDir)
	if !projectTypesResult.OK {
		return false
	}
	projectTypes := projectTypesResult.Value.([]build.ProjectType)
	if len(projectTypes) != 1 || projectTypes[0] != build.ProjectTypeGo {
		return false
	}

	if req.ObfuscateSet || req.NSISSet || req.WebView2Set || req.DenoBuildSet || req.NpmBuildSet || req.BuildCacheSet || req.SignSet || req.NoSign || req.Notarize {
		return false
	}

	if req.Push || req.ImageName != "" || req.Format != "" {
		return false
	}
	if req.CIMode || req.Version != "" || req.ArchiveFormat != "" {
		return false
	}
	if req.ArchiveOutputSet && req.ArchiveOutput {
		return false
	}
	if req.ChecksumOutputSet && req.ChecksumOutput {
		return false
	}
	if req.PackageSet && (req.ArchiveOutput || req.ChecksumOutput) {
		return false
	}

	if req.TargetsFlag == "" {
		return true
	}

	targetsResult := parseTargets(req.TargetsFlag)
	if !targetsResult.OK {
		return false
	}
	targets := targetsResult.Value.([]build.Target)

	return len(targets) == 1
}

func runGoBuildPassthrough(ctx context.Context, projectDir string, req ProjectBuildRequest) core.Result {
	args := []string{"build"}

	if outputPath := resolveGoPassthroughOutput(req.OutputDir, req.BuildName); outputPath != "" {
		args = append(args, "-o", outputPath)
	}

	if tags := parseBuildTagsFlag(req.BuildTagsFlag); len(tags) > 0 {
		args = append(args, "-tags", core.Join(",", tags...))
	}

	args = append(args, ".")

	env := []string{}
	if req.TargetsFlag != "" {
		targetsResult := parseTargets(req.TargetsFlag)
		if !targetsResult.OK {
			return targetsResult
		}
		targets := targetsResult.Value.([]build.Target)
		if len(targets) != 1 {
			return core.Fail(coreerr.E("build.Run", "go build passthrough supports exactly one target", nil))
		}

		env = append(env,
			"GOOS="+targets[0].OS,
			"GOARCH="+targets[0].Arch,
		)
	}

	built := ax.ExecWithEnv(ctx, projectDir, env, "go", args...)
	if !built.OK {
		return core.Fail(coreerr.E("build.Run", "go build passthrough failed", core.NewError(built.Error())))
	}

	return core.Ok(nil)
}

func resolveGoPassthroughOutput(outputDir, buildName string) string {
	switch {
	case outputDir != "" && buildName != "":
		return ax.Join(outputDir, buildName)
	case outputDir != "":
		return outputDir
	default:
		return buildName
	}
}

func applyProjectBuildOverrides(cfg *build.BuildConfig, req ProjectBuildRequest) {
	if cfg == nil {
		return
	}

	if tags := parseBuildTagsFlag(req.BuildTagsFlag); len(tags) > 0 {
		cfg.Build.BuildTags = tags
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
			enableDefaultBuildCache(&cfg.Build.Cache)
		} else {
			cfg.Build.Cache.Enabled = false
		}
	}
	if req.SignSet {
		cfg.Sign.Enabled = req.Sign
	}
}

func parseBuildTagsFlag(value string) []string {
	if core.Trim(value) == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var tags []string
	for _, part := range buildTagFields(value) {
		tag := core.Trim(part)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}

	return tags
}

func buildTagFields(value string) []string {
	var fields []string
	start := -1
	for i, r := range value {
		if r == ',' || unicodeIsSpace(r) {
			if start >= 0 {
				fields = append(fields, value[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		fields = append(fields, value[start:])
	}
	return fields
}

func enableDefaultBuildCache(cfg *build.CacheConfig) {
	if cfg == nil {
		return
	}

	cfg.Enabled = true
	if cfg.Directory == "" {
		cfg.Directory = ax.Join(build.ConfigDir, "cache")
	}
	if len(cfg.Paths) == 0 {
		cfg.Paths = build.DefaultBuildCachePaths("")
	}
}

func resolveProjectBuildName(projectDir string, buildConfig *build.BuildConfig, override string) string {
	return build.ResolveBuildName(projectDir, buildConfig, override)
}

func unicodeIsSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// selectOutputArtifacts chooses the final artifact list for CI output.
//
// output := selectOutputArtifacts(rawArtifacts, archivedArtifacts, checksummedArtifacts)
func selectOutputArtifacts(rawArtifacts, archivedArtifacts, checksummedArtifacts []build.Artifact) []build.Artifact {
	if len(checksummedArtifacts) > 0 {
		return checksummedArtifacts
	}
	if len(archivedArtifacts) > 0 {
		return archivedArtifacts
	}
	return rawArtifacts
}

// writeArtifactMetadata writes artifact_meta.json files next to built artifacts when CI metadata is available.
func writeArtifactMetadata(filesystem io.Medium, buildName string, artifacts []build.Artifact) core.Result {
	ci := resolveCIContext()
	if ci == nil {
		return core.Ok(nil)
	}

	for _, artifact := range artifacts {
		if artifact.OS == "" || artifact.Arch == "" {
			continue
		}
		metaPath := ax.Join(ax.Dir(artifact.Path), "artifact_meta.json")
		written := build.WriteArtifactMeta(filesystem, metaPath, buildName, build.Target{OS: artifact.OS, Arch: artifact.Arch}, ci)
		if !written.OK {
			return written
		}
	}

	return core.Ok(nil)
}

func rewriteArtifactsForCI(filesystem io.Medium, buildName string, artifacts []build.Artifact) core.Result {
	ci := resolveCIContext()
	if ci == nil {
		return core.Ok(artifacts)
	}

	rewritten := make([]build.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		ciPath := build.CIArtifactPath(buildName, ci, artifact)
		if ciPath == "" || ciPath == artifact.Path {
			rewritten = append(rewritten, artifact)
			continue
		}

		created := filesystem.EnsureDir(ax.Dir(ciPath))
		if !created.OK {
			return core.Fail(coreerr.E("build.rewriteArtifactsForCI", "failed to create artifact directory", core.NewError(created.Error())))
		}
		copied := io.Copy(filesystem, artifact.Path, filesystem, ciPath)
		if !copied.OK {
			return core.Fail(coreerr.E("build.rewriteArtifactsForCI", "failed to copy artifact", core.NewError(copied.Error())))
		}

		artifact.Path = ciPath
		rewritten = append(rewritten, artifact)
	}

	return core.Ok(rewritten)
}

func resolveCIContext() *build.CIContext {
	if ci := build.DetectCI(); ci != nil {
		return ci
	}

	return build.DetectGitHubMetadata()
}

// buildRuntimeConfig maps persisted build configuration onto the runtime builder config.
func buildRuntimeConfig(filesystem io.Medium, projectDir, outputDir, binaryName string, buildConfig *build.BuildConfig, push bool, imageName string, version string) *build.Config {
	return build.RuntimeConfigFromBuildConfig(filesystem, projectDir, outputDir, binaryName, buildConfig, push, imageName, version)
}

// resolveArchiveFormat selects the archive format from CLI overrides or config defaults.
func resolveArchiveFormat(configFormat, cliFormat string) core.Result {
	if cliFormat != "" {
		return build.ParseArchiveFormat(cliFormat)
	}
	return build.ParseArchiveFormat(configFormat)
}

// resolveBuildVersion determines the version string embedded into build artifacts.
//
// version, err := resolveBuildVersion(ctx, ".")
func resolveBuildVersion(ctx context.Context, projectDir string) core.Result {
	return release.DetermineVersionWithContext(ctx, projectDir)
}

// computeAndWriteChecksums computes checksums for artifacts and writes CHECKSUMS.txt.
func computeAndWriteChecksums(ctx context.Context, filesystem io.Medium, projectDir, outputDir string, artifacts []build.Artifact, signCfg signing.SignConfig, ciMode bool, verbose bool) core.Result {
	if verbose && !ciMode {
		cli.Blank()
		cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.label.checksum")), i18n.T("cmd.build.computing_checksums"))
	}

	checksummedArtifactsResult := build.ChecksumAll(filesystem, artifacts)
	if !checksummedArtifactsResult.OK {
		if !ciMode {
			cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.checksum_failed"), checksummedArtifactsResult.Error())
		}
		return checksummedArtifactsResult
	}
	checksummedArtifacts := checksummedArtifactsResult.Value.([]build.Artifact)

	// Write CHECKSUMS.txt
	checksumPath := ax.Join(outputDir, "CHECKSUMS.txt")
	written := build.WriteChecksumFile(filesystem, checksummedArtifacts, checksumPath)
	if !written.OK {
		if !ciMode {
			cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("common.error.failed", map[string]any{"Action": "write CHECKSUMS.txt"}), written.Error())
		}
		return written
	}

	// Sign checksums with GPG
	if signCfg.Enabled {
		signed := signing.SignChecksums(ctx, filesystem, signCfg, checksumPath)
		if !signed.OK {
			if !ciMode {
				cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.gpg_signing_failed"), signed.Error())
			}
			return signed
		}
	}

	if verbose && !ciMode {
		for _, artifact := range checksummedArtifacts {
			relPath := artifact.Path
			relPathResult := ax.Rel(projectDir, artifact.Path)
			if relPathResult.OK {
				relPath = relPathResult.Value.(string)
			}
			cli.Print("  %s %s\n",
				buildSuccessStyle.Render("*"),
				buildTargetStyle.Render(relPath),
			)
			cli.Print("    %s\n", buildDimStyle.Render(artifact.Checksum))
		}

		relChecksumPath := checksumPath
		relChecksumPathResult := ax.Rel(projectDir, checksumPath)
		if relChecksumPathResult.OK {
			relChecksumPath = relChecksumPathResult.Value.(string)
		}
		cli.Print("  %s %s\n",
			buildSuccessStyle.Render("*"),
			buildTargetStyle.Render(relChecksumPath),
		)

		signaturePath := checksumPath + ".asc"
		if filesystem.Exists(signaturePath) {
			relSignaturePath := signaturePath
			relSignaturePathResult := ax.Rel(projectDir, signaturePath)
			if relSignaturePathResult.OK {
				relSignaturePath = relSignaturePathResult.Value.(string)
			}
			cli.Print("  %s %s\n",
				buildSuccessStyle.Render("*"),
				buildTargetStyle.Render(relSignaturePath),
			)
		}
	}

	outputArtifacts := append([]build.Artifact(nil), checksummedArtifacts...)
	outputArtifacts = append(outputArtifacts, build.Artifact{Path: checksumPath})

	signaturePath := checksumPath + ".asc"
	if filesystem.Exists(signaturePath) {
		outputArtifacts = append(outputArtifacts, build.Artifact{Path: signaturePath})
	}

	return core.Ok(outputArtifacts)
}

// parseTargets parses a comma-separated list of OS/arch pairs.
func parseTargets(targetsFlag string) core.Result {
	parts := core.Split(targetsFlag, ",")
	var targets []build.Target

	for _, part := range parts {
		part = core.Trim(part)
		if part == "" {
			continue
		}

		osArch := core.Split(part, "/")
		if len(osArch) != 2 {
			return core.Fail(coreerr.E("build.parseTargets", "invalid target format (expected os/arch): "+part, nil))
		}

		targets = append(targets, build.Target{
			OS:   core.Trim(osArch[0]),
			Arch: core.Trim(osArch[1]),
		})
	}

	if len(targets) == 0 {
		return core.Fail(coreerr.E("build.parseTargets", "no valid targets specified", nil))
	}

	return core.Ok(targets)
}

// formatTargets returns a human-readable string of targets.
func formatTargets(targets []build.Target) string {
	var parts []string
	for _, t := range targets {
		parts = append(parts, t.String())
	}
	return core.Join(", ", parts...)
}

func formatProjectTypes(projectTypes []build.ProjectType) string {
	if len(projectTypes) == 0 {
		return ""
	}

	parts := make([]string, 0, len(projectTypes))
	for _, projectType := range projectTypes {
		parts = append(parts, string(projectType))
	}

	return core.Join(", ", parts...)
}

// getBuilder returns the appropriate builder for the project type.
func getBuilder(projectType build.ProjectType) core.Result {
	builder := builders.ResolveBuilder(projectType)
	if !builder.OK {
		return core.Fail(coreerr.E("build.getBuilder", "unsupported project type: "+string(projectType), core.NewError(builder.Error())))
	}
	return builder
}
