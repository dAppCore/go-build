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
	"strings"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/builders"
	"dappco.re/go/build/pkg/build/signing"
	"dappco.re/go/build/pkg/release"
	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
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
func runProjectBuild(req ProjectBuildRequest) error {
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}
	// Use local filesystem as the default medium.
	filesystem := io.Local

	// Get current working directory as project root
	projectDir, err := getProjectBuildWorkingDir()
	if err != nil {
		return coreerr.E("build.Run", "failed to get working directory", err)
	}

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
			return coreerr.E("build.Run", "build config not found: "+configPath, nil)
		}
		buildConfig, err = build.LoadConfigAtPath(filesystem, configPath)
	} else {
		buildConfig, err = build.LoadConfig(filesystem, projectDir)
	}
	if err != nil {
		return coreerr.E("build.Run", "failed to load config", err)
	}

	if buildConfig.Build.Type == "pwa" {
		return runLocalPwaBuild(ctx, projectDir)
	}

	applyProjectBuildOverrides(buildConfig, req)

	// Determine targets
	var buildTargets []build.Target
	if req.TargetsFlag != "" {
		// Parse from command line
		buildTargets, err = parseTargets(req.TargetsFlag)
		if err != nil {
			return err
		}
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
	plan, err := pipeline.Plan(ctx, build.PipelineRequest{
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
	if err != nil {
		return err
	}

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
	pipelineResult, err := pipeline.Run(ctx, plan)
	if err != nil {
		if !req.CIMode {
			cli.Print("%s %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), err)
		}
		return err
	}
	artifacts := pipelineResult.Artifacts

	if req.Verbose && !req.CIMode {
		cli.Print("%s %s\n", buildSuccessStyle.Render(i18n.T("common.label.success")), i18n.T("cmd.build.built_artifacts", map[string]any{"Count": len(artifacts)}))
		cli.Blank()
		for _, artifact := range artifacts {
			relPath, err := ax.Rel(projectDir, artifact.Path)
			if err != nil {
				relPath = artifact.Path
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

		if err := signing.SignBinaries(ctx, filesystem, signCfg, signingArtifacts); err != nil {
			if !req.CIMode {
				cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.signing_failed"), err)
			}
			return err
		}

		if runtime.GOOS == "darwin" && signCfg.MacOS.Notarize {
			if err := signing.NotarizeBinaries(ctx, filesystem, signCfg, signingArtifacts); err != nil {
				if !req.CIMode {
					cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.notarization_failed"), err)
				}
				return err
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

		archiveFormatValue, err := resolveArchiveFormat(buildConfig.Build.ArchiveFormat, req.ArchiveFormat)
		if err != nil {
			return err
		}

		archivedArtifacts, err = build.ArchiveAllWithFormat(filesystem, artifacts, archiveFormatValue)
		if err != nil {
			if !req.CIMode {
				cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.archive_failed"), err)
			}
			return err
		}

		if req.Verbose && !req.CIMode {
			for _, artifact := range archivedArtifacts {
				relPath, err := ax.Rel(projectDir, artifact.Path)
				if err != nil {
					relPath = artifact.Path
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
		checksummedArtifacts, err = computeAndWriteChecksums(ctx, filesystem, projectDir, plan.OutputDir, archivedArtifacts, signCfg, req.CIMode, req.Verbose)
		if err != nil {
			return err
		}
	} else if req.ChecksumOutput && len(artifacts) > 0 && !req.ArchiveOutput {
		// Checksum raw binaries if archiving is disabled
		checksummedArtifacts, err = computeAndWriteChecksums(ctx, filesystem, projectDir, plan.OutputDir, artifacts, signCfg, req.CIMode, req.Verbose)
		if err != nil {
			return err
		}
	}

	// Output results
	if req.CIMode {
		// Determine which artifacts to output (prefer checksummed > archived > raw).
		outputArtifacts := selectOutputArtifacts(artifacts, archivedArtifacts, checksummedArtifacts)
		if err := writeArtifactMetadata(filesystem, plan.BuildName, outputArtifacts); err != nil {
			return err
		}

		// JSON output for CI
		output, err := ax.JSONMarshal(outputArtifacts)
		if err != nil {
			return coreerr.E("build.Run", "failed to marshal artifacts", err)
		}
		cli.Print("%s\n", output)
	} else if !req.Verbose {
		// Minimal output: just success with artifact count
		cli.Print("%s %s %s\n",
			buildSuccessStyle.Render(i18n.T("common.label.success")),
			i18n.T("cmd.build.built_artifacts", map[string]any{"Count": len(artifacts)}),
			buildDimStyle.Render(core.Sprintf("(%s)", plan.OutputDir)),
		)
	}

	return nil
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

	projectTypes, err := build.Discover(filesystem, projectDir)
	if err != nil || len(projectTypes) != 1 || projectTypes[0] != build.ProjectTypeGo {
		return false
	}

	if req.ObfuscateSet || req.NSISSet || req.WebView2Set || req.DenoBuildSet || req.BuildCacheSet || req.SignSet || req.NoSign || req.Notarize {
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

	targets, err := parseTargets(req.TargetsFlag)
	if err != nil {
		return false
	}

	return len(targets) == 1
}

func runGoBuildPassthrough(ctx context.Context, projectDir string, req ProjectBuildRequest) error {
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
		targets, err := parseTargets(req.TargetsFlag)
		if err != nil {
			return err
		}
		if len(targets) != 1 {
			return coreerr.E("build.Run", "go build passthrough supports exactly one target", nil)
		}

		env = append(env,
			"GOOS="+targets[0].OS,
			"GOARCH="+targets[0].Arch,
		)
	}

	if err := ax.ExecWithEnv(ctx, projectDir, env, "go", args...); err != nil {
		return coreerr.E("build.Run", "go build passthrough failed", err)
	}

	return nil
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
	if strings.TrimSpace(value) == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var tags []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || unicodeIsSpace(r)
	}) {
		tag := strings.TrimSpace(part)
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

func enableDefaultBuildCache(cfg *build.CacheConfig) {
	if cfg == nil {
		return
	}

	cfg.Enabled = true
	if cfg.Directory == "" {
		cfg.Directory = ax.Join(build.ConfigDir, "cache")
	}
	if len(cfg.Paths) == 0 {
		cfg.Paths = []string{
			ax.Join("cache", "go-build"),
			ax.Join("cache", "go-mod"),
		}
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
func writeArtifactMetadata(filesystem io.Medium, buildName string, artifacts []build.Artifact) error {
	ci := build.DetectCI()
	if ci == nil {
		ci = build.DetectGitHubMetadata()
	}
	if ci == nil {
		return nil
	}

	for _, artifact := range artifacts {
		if artifact.OS == "" || artifact.Arch == "" {
			continue
		}
		metaPath := ax.Join(ax.Dir(artifact.Path), "artifact_meta.json")
		if err := build.WriteArtifactMeta(filesystem, metaPath, buildName, build.Target{OS: artifact.OS, Arch: artifact.Arch}, ci); err != nil {
			return err
		}
	}

	return nil
}

// buildRuntimeConfig maps persisted build configuration onto the runtime builder config.
func buildRuntimeConfig(filesystem io.Medium, projectDir, outputDir, binaryName string, buildConfig *build.BuildConfig, push bool, imageName string, version string) *build.Config {
	return build.RuntimeConfigFromBuildConfig(filesystem, projectDir, outputDir, binaryName, buildConfig, push, imageName, version)
}

// resolveArchiveFormat selects the archive format from CLI overrides or config defaults.
func resolveArchiveFormat(configFormat, cliFormat string) (build.ArchiveFormat, error) {
	if cliFormat != "" {
		return build.ParseArchiveFormat(cliFormat)
	}
	return build.ParseArchiveFormat(configFormat)
}

// resolveBuildVersion determines the version string embedded into build artifacts.
//
// version, err := resolveBuildVersion(ctx, ".")
func resolveBuildVersion(ctx context.Context, projectDir string) (string, error) {
	return release.DetermineVersionWithContext(ctx, projectDir)
}

// computeAndWriteChecksums computes checksums for artifacts and writes CHECKSUMS.txt.
func computeAndWriteChecksums(ctx context.Context, filesystem io.Medium, projectDir, outputDir string, artifacts []build.Artifact, signCfg signing.SignConfig, ciMode bool, verbose bool) ([]build.Artifact, error) {
	if verbose && !ciMode {
		cli.Blank()
		cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.label.checksum")), i18n.T("cmd.build.computing_checksums"))
	}

	checksummedArtifacts, err := build.ChecksumAll(filesystem, artifacts)
	if err != nil {
		if !ciMode {
			cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.checksum_failed"), err)
		}
		return nil, err
	}

	// Write CHECKSUMS.txt
	checksumPath := ax.Join(outputDir, "CHECKSUMS.txt")
	if err := build.WriteChecksumFile(filesystem, checksummedArtifacts, checksumPath); err != nil {
		if !ciMode {
			cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("common.error.failed", map[string]any{"Action": "write CHECKSUMS.txt"}), err)
		}
		return nil, err
	}

	// Sign checksums with GPG
	if signCfg.Enabled {
		if err := signing.SignChecksums(ctx, filesystem, signCfg, checksumPath); err != nil {
			if !ciMode {
				cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.gpg_signing_failed"), err)
			}
			return nil, err
		}
	}

	if verbose && !ciMode {
		for _, artifact := range checksummedArtifacts {
			relPath, err := ax.Rel(projectDir, artifact.Path)
			if err != nil {
				relPath = artifact.Path
			}
			cli.Print("  %s %s\n",
				buildSuccessStyle.Render("*"),
				buildTargetStyle.Render(relPath),
			)
			cli.Print("    %s\n", buildDimStyle.Render(artifact.Checksum))
		}

		relChecksumPath, err := ax.Rel(projectDir, checksumPath)
		if err != nil {
			relChecksumPath = checksumPath
		}
		cli.Print("  %s %s\n",
			buildSuccessStyle.Render("*"),
			buildTargetStyle.Render(relChecksumPath),
		)

		signaturePath := checksumPath + ".asc"
		if filesystem.Exists(signaturePath) {
			relSignaturePath, err := ax.Rel(projectDir, signaturePath)
			if err != nil {
				relSignaturePath = signaturePath
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

	return outputArtifacts, nil
}

// parseTargets parses a comma-separated list of OS/arch pairs.
func parseTargets(targetsFlag string) ([]build.Target, error) {
	parts := core.Split(targetsFlag, ",")
	var targets []build.Target

	for _, part := range parts {
		part = core.Trim(part)
		if part == "" {
			continue
		}

		osArch := core.Split(part, "/")
		if len(osArch) != 2 {
			return nil, coreerr.E("build.parseTargets", "invalid target format (expected os/arch): "+part, nil)
		}

		targets = append(targets, build.Target{
			OS:   core.Trim(osArch[0]),
			Arch: core.Trim(osArch[1]),
		})
	}

	if len(targets) == 0 {
		return nil, coreerr.E("build.parseTargets", "no valid targets specified", nil)
	}

	return targets, nil
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
func getBuilder(projectType build.ProjectType) (build.Builder, error) {
	builder, err := builders.ResolveBuilder(projectType)
	if err != nil {
		return nil, coreerr.E("build.getBuilder", "unsupported project type: "+string(projectType), err)
	}
	return builder, nil
}
