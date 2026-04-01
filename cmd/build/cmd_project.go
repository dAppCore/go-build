// cmd_project.go implements the main project build logic.
//
// This handles auto-detection of project types (Go, Wails, Docker, LinuxKit, Taskfile)
// and orchestrates the build process including signing, archiving, and checksums.

package buildcmd

import (
	"context"
	"runtime"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/internal/projectdetect"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/build/builders"
	"dappco.re/go/core/build/pkg/build/signing"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"forge.lthn.ai/core/cli/pkg/cli"
)

// runProjectBuild handles the main `core build` command with auto-detection.
func runProjectBuild(ctx context.Context, buildType string, ciMode bool, targetsFlag string, outputDir string, archiveOutput bool, checksumOutput bool, configPath string, format string, push bool, imageName string, noSign bool, notarize bool, verbose bool) error {
	// Use local filesystem as the default medium.
	filesystem := io.Local

	// Get current working directory as project root
	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.Run", "failed to get working directory", err)
	}

	// Load configuration from .core/build.yaml (or defaults)
	buildConfig, err := build.LoadConfig(filesystem, projectDir)
	if err != nil {
		return coreerr.E("build.Run", "failed to load config", err)
	}

	// Detect project type if not specified
	var projectType build.ProjectType
	if buildType != "" {
		projectType = build.ProjectType(buildType)
	} else if buildConfig.Build.Type != "" {
		// Use type from .core/build.yaml
		projectType = build.ProjectType(buildConfig.Build.Type)
	} else {
		projectType, err = projectdetect.DetectProjectType(filesystem, projectDir)
		if err != nil {
			return coreerr.E("build.Run", "failed to detect project type", err)
		}
		if projectType == "" {
			return coreerr.E("build.Run", "no buildable project type found in "+projectDir, nil)
		}
	}

	// Determine targets
	var buildTargets []build.Target
	if targetsFlag != "" {
		// Parse from command line
		buildTargets, err = parseTargets(targetsFlag)
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

	// Determine output directory
	if outputDir == "" {
		outputDir = "dist"
	}
	if !ax.IsAbs(outputDir) {
		outputDir = ax.Join(projectDir, outputDir)
	}
	outputDir = ax.Clean(outputDir)

	// Ensure config path is absolute if provided
	if configPath != "" && !ax.IsAbs(configPath) {
		configPath = ax.Join(projectDir, configPath)
	}

	// Determine binary name
	binaryName := buildConfig.Project.Binary
	if binaryName == "" {
		binaryName = buildConfig.Project.Name
	}
	if binaryName == "" {
		binaryName = ax.Base(projectDir)
	}

	// Print build info (verbose mode only)
	if verbose && !ciMode {
		cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.label.build")), i18n.T("cmd.build.building_project"))
		cli.Print("  %s %s\n", i18n.T("cmd.build.label.type"), buildTargetStyle.Render(string(projectType)))
		cli.Print("  %s %s\n", i18n.T("cmd.build.label.output"), buildTargetStyle.Render(outputDir))
		cli.Print("  %s %s\n", i18n.T("cmd.build.label.binary"), buildTargetStyle.Render(binaryName))
		cli.Print("  %s %s\n", i18n.T("cmd.build.label.targets"), buildTargetStyle.Render(formatTargets(buildTargets)))
		cli.Blank()
	}

	// Get the appropriate builder
	builder, err := getBuilder(projectType)
	if err != nil {
		return err
	}

	// Create build config for the builder
	cfg := &build.Config{
		FS:         filesystem,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       binaryName,
		Version:    buildConfig.Project.Name, // Could be enhanced with git describe
		LDFlags:    buildConfig.Build.LDFlags,
		CGO:        buildConfig.Build.CGO,
		// Docker/LinuxKit specific
		Dockerfile:     configPath, // Reuse for Dockerfile path
		LinuxKitConfig: configPath,
		Push:           push,
		Image:          imageName,
	}

	// Parse formats for LinuxKit
	if format != "" {
		cfg.Formats = core.Split(format, ",")
	}

	// Execute build
	artifacts, err := builder.Build(ctx, cfg, buildTargets)
	if err != nil {
		if !ciMode {
			cli.Print("%s %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), err)
		}
		return err
	}

	if verbose && !ciMode {
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

	// Sign macOS binaries if enabled
	signCfg := buildConfig.Sign
	if notarize {
		signCfg.MacOS.Notarize = true
	}
	if noSign {
		signCfg.Enabled = false
	}

	if signCfg.Enabled && runtime.GOOS == "darwin" {
		if verbose && !ciMode {
			cli.Blank()
			cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.label.sign")), i18n.T("cmd.build.signing_binaries"))
		}

		// Convert build.Artifact to signing.Artifact
		signingArtifacts := make([]signing.Artifact, len(artifacts))
		for i, a := range artifacts {
			signingArtifacts[i] = signing.Artifact{Path: a.Path, OS: a.OS, Arch: a.Arch}
		}

		if err := signing.SignBinaries(ctx, filesystem, signCfg, signingArtifacts); err != nil {
			if !ciMode {
				cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.signing_failed"), err)
			}
			return err
		}

		if signCfg.MacOS.Notarize {
			if err := signing.NotarizeBinaries(ctx, filesystem, signCfg, signingArtifacts); err != nil {
				if !ciMode {
					cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.notarization_failed"), err)
				}
				return err
			}
		}
	}

	// Archive artifacts if enabled
	var archivedArtifacts []build.Artifact
	if archiveOutput && len(artifacts) > 0 {
		if verbose && !ciMode {
			cli.Blank()
			cli.Print("%s %s\n", buildHeaderStyle.Render(i18n.T("cmd.build.label.archive")), i18n.T("cmd.build.creating_archives"))
		}

		archivedArtifacts, err = build.ArchiveAll(filesystem, artifacts)
		if err != nil {
			if !ciMode {
				cli.Print("%s %s: %v\n", buildErrorStyle.Render(i18n.T("common.label.error")), i18n.T("cmd.build.error.archive_failed"), err)
			}
			return err
		}

		if verbose && !ciMode {
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
	if checksumOutput && len(archivedArtifacts) > 0 {
		checksummedArtifacts, err = computeAndWriteChecksums(ctx, filesystem, projectDir, outputDir, archivedArtifacts, signCfg, ciMode, verbose)
		if err != nil {
			return err
		}
	} else if checksumOutput && len(artifacts) > 0 && !archiveOutput {
		// Checksum raw binaries if archiving is disabled
		checksummedArtifacts, err = computeAndWriteChecksums(ctx, filesystem, projectDir, outputDir, artifacts, signCfg, ciMode, verbose)
		if err != nil {
			return err
		}
	}

	// Output results
	if ciMode {
		// Determine which artifacts to output (prefer checksummed > archived > raw)
		var outputArtifacts []build.Artifact
		if len(checksummedArtifacts) > 0 {
			outputArtifacts = checksummedArtifacts
		} else if len(archivedArtifacts) > 0 {
			outputArtifacts = archivedArtifacts
		} else {
			outputArtifacts = artifacts
		}

		// JSON output for CI
		output, err := ax.JSONMarshal(outputArtifacts)
		if err != nil {
			return coreerr.E("build.Run", "failed to marshal artifacts", err)
		}
		cli.Print("%s\n", output)
	} else if !verbose {
		// Minimal output: just success with artifact count
		cli.Print("%s %s %s\n",
			buildSuccessStyle.Render(i18n.T("common.label.success")),
			i18n.T("cmd.build.built_artifacts", map[string]any{"Count": len(artifacts)}),
			buildDimStyle.Render(core.Sprintf("(%s)", outputDir)),
		)
	}

	return nil
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
	}

	return checksummedArtifacts, nil
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

// getBuilder returns the appropriate builder for the project type.
func getBuilder(projectType build.ProjectType) (build.Builder, error) {
	switch projectType {
	case build.ProjectTypeWails:
		return builders.NewWailsBuilder(), nil
	case build.ProjectTypeGo:
		return builders.NewGoBuilder(), nil
	case build.ProjectTypeDocker:
		return builders.NewDockerBuilder(), nil
	case build.ProjectTypeLinuxKit:
		return builders.NewLinuxKitBuilder(), nil
	case build.ProjectTypeTaskfile:
		return builders.NewTaskfileBuilder(), nil
	case build.ProjectTypeCPP:
		return builders.NewCPPBuilder(), nil
	case build.ProjectTypeNode:
		return builders.NewNodeBuilder(), nil
	case build.ProjectTypePHP:
		return nil, coreerr.E("build.getBuilder", "PHP builder not yet implemented", nil)
	default:
		return nil, coreerr.E("build.getBuilder", "unsupported project type: "+string(projectType), nil)
	}
}
