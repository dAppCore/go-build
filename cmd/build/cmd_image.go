package buildcmd

import (
	"context"
	"strings"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/internal/cmdutil"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/build/builders"
	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// ImageBuildRequest groups the inputs for `core build image`.
type ImageBuildRequest struct {
	Context   context.Context
	Base      string
	Format    string
	OutputDir string
	List      bool
	Rebuild   bool
}

// AddImageCommand registers the immutable LinuxKit image builder command.
func AddImageCommand(c *core.Core) {
	c.Command("build/image", core.Command{
		Description: "Build immutable LinuxKit base images",
		Action: func(opts core.Options) core.Result {
			return cmdutil.ResultFromError(runBuildImage(ImageBuildRequest{
				Context:   cmdutil.ContextOrBackground(),
				Base:      resolveImageBase(opts),
				Format:    cmdutil.OptionString(opts, "format"),
				OutputDir: cmdutil.OptionString(opts, "output"),
				List:      cmdutil.OptionBool(opts, "list"),
				Rebuild:   cmdutil.OptionBool(opts, "rebuild"),
			}))
		},
	})
}

func resolveImageBase(opts core.Options) string {
	if base := cmdutil.OptionString(opts, "base", "name"); base != "" {
		return base
	}
	return opts.String("_arg")
}

// runBuildImage renders the embedded immutable LinuxKit image template and builds the requested formats.
func runBuildImage(req ImageBuildRequest) error {
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}

	projectDir, err := ax.Getwd()
	if err != nil {
		return coreerr.E("build.runBuildImage", "failed to get working directory", err)
	}

	imageBuilder := builders.NewLinuxKitImageBuilder()
	if req.List {
		cli.Print("%s %s\n", buildHeaderStyle.Render("Images"), "available immutable LinuxKit bases")
		for _, baseImage := range imageBuilder.ListBaseImages() {
			cli.Print("  %s %s %s\n", buildTargetStyle.Render(baseImage.Name), buildDimStyle.Render(baseImage.Version), baseImage.Description)
		}
		return nil
	}

	buildConfig, err := build.LoadConfig(io.Local, projectDir)
	if err != nil {
		return coreerr.E("build.runBuildImage", "failed to load build config", err)
	}

	if req.Base != "" {
		buildConfig.LinuxKit.Base = req.Base
	}
	if req.Format != "" {
		buildConfig.LinuxKit.Formats = parseImageFormats(req.Format)
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = "dist"
	}
	if !ax.IsAbs(outputDir) {
		outputDir = ax.Join(projectDir, outputDir)
	}

	version, versionErr := resolveBuildVersion(ctx, projectDir)
	if versionErr != nil || version == "" {
		version = "dev"
	}

	imageName := buildConfig.LinuxKit.Base
	if imageName == "" {
		imageName = build.DefaultLinuxKitConfig().Base
	}

	runtimeCfg := &build.Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       imageName,
		Version:    version,
		LinuxKit:   buildConfig.LinuxKit,
	}

	formats := runtimeCfg.LinuxKit.Formats
	if len(formats) == 0 {
		formats = append([]string(nil), build.DefaultLinuxKitConfig().Formats...)
	}

	if !req.Rebuild && allImageArtifactsExist(io.Local, imageBuilder, outputDir, imageName, formats) {
		cli.Print("%s %s\n", buildSuccessStyle.Render("Using"), "cached immutable image artifacts")
		return nil
	}

	artifacts, err := imageBuilder.Build(ctx, runtimeCfg)
	if err != nil {
		return err
	}

	cli.Print("%s %s\n", buildSuccessStyle.Render("Built"), buildTargetStyle.Render(imageName))
	for _, artifact := range artifacts {
		relPath, relErr := ax.Rel(projectDir, artifact.Path)
		if relErr != nil {
			relPath = artifact.Path
		}
		cli.Print("  %s\n", relPath)
	}

	return nil
}

func parseImageFormats(value string) []string {
	if value == "" {
		return nil
	}

	parts := core.Split(value, ",")
	formats := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		formats = append(formats, part)
	}
	return formats
}

func allImageArtifactsExist(filesystem io.Medium, imageBuilder *builders.LinuxKitImageBuilder, outputDir, imageName string, formats []string) bool {
	if len(formats) == 0 {
		return false
	}

	for _, format := range formats {
		if !filesystem.Exists(imageBuilder.ArtifactPath(outputDir, imageName, format)) {
			return false
		}
	}
	return true
}
