package buildcmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
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

type imageBuildCacheMetadata struct {
	ImageName    string   `json:"image_name"`
	Base         string   `json:"base"`
	BaseVersion  string   `json:"base_version,omitempty"`
	BuildVersion string   `json:"build_version"`
	Formats      []string `json:"formats,omitempty"`
	Packages     []string `json:"packages,omitempty"`
	Mounts       []string `json:"mounts,omitempty"`
	GPU          bool     `json:"gpu,omitempty"`
	Registry     string   `json:"registry,omitempty"`
	Signature    string   `json:"signature"`
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
	cacheVersion := strings.TrimSpace(version)
	if _, err := ax.LookPath("git"); err != nil {
		cacheVersion = ""
	}
	if versionErr != nil || version == "" {
		version = "dev"
		cacheVersion = ""
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

	cacheCfg := runtimeCfg.LinuxKit
	cacheCfg.Formats = append([]string(nil), formats...)

	if !req.Rebuild && allImageArtifactsExist(io.Local, imageBuilder, outputDir, imageName, cacheCfg, cacheVersion) {
		cli.Print("%s %s\n", buildSuccessStyle.Render("Using"), "cached immutable image artifacts")
		return nil
	}

	artifacts, err := imageBuilder.Build(ctx, runtimeCfg)
	if err != nil {
		return err
	}
	if err := writeImageBuildCacheMetadata(io.Local, outputDir, imageName, cacheCfg, cacheVersion); err != nil {
		return coreerr.E("build.runBuildImage", "failed to write image cache metadata", err)
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

func allImageArtifactsExist(filesystem io.Medium, imageBuilder *builders.LinuxKitImageBuilder, outputDir, imageName string, cfg build.LinuxKitConfig, version string) bool {
	formats := normalizeImageCacheValues(cfg.Formats)
	if len(formats) == 0 {
		return false
	}

	for _, format := range formats {
		if !filesystem.Exists(imageBuilder.ArtifactPath(outputDir, imageName, format)) {
			return false
		}
	}

	metadata, err := loadImageBuildCacheMetadata(filesystem, outputDir, imageName)
	if err != nil || metadata == nil {
		return false
	}
	expected := buildImageCacheMetadata(imageName, cfg, version)
	if metadata.Signature != expected.Signature {
		return false
	}

	expectedVersion := strings.TrimSpace(expected.BuildVersion)
	if expectedVersion == "" {
		return true
	}

	return strings.TrimSpace(metadata.BuildVersion) == expectedVersion
}

func writeImageBuildCacheMetadata(filesystem io.Medium, outputDir, imageName string, cfg build.LinuxKitConfig, version string) error {
	metadata := buildImageCacheMetadata(imageName, cfg, version)
	encoded, err := ax.JSONMarshal(metadata)
	if err != nil {
		return err
	}
	return filesystem.Write(imageBuildCacheMetadataPath(outputDir, imageName), encoded)
}

func loadImageBuildCacheMetadata(filesystem io.Medium, outputDir, imageName string) (*imageBuildCacheMetadata, error) {
	path := imageBuildCacheMetadataPath(outputDir, imageName)
	if !filesystem.Exists(path) {
		return nil, nil
	}

	content, err := filesystem.Read(path)
	if err != nil {
		return nil, err
	}

	var metadata imageBuildCacheMetadata
	if err := ax.JSONUnmarshal([]byte(content), &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func imageBuildCacheMetadataPath(outputDir, imageName string) string {
	return ax.Join(outputDir, "."+imageName+"-linuxkit-image.json")
}

func buildImageCacheMetadata(imageName string, cfg build.LinuxKitConfig, version string) imageBuildCacheMetadata {
	base := cfg.Base
	baseVersion := ""
	if baseImage, ok := build.LookupLinuxKitBaseImage(base); ok {
		baseVersion = baseImage.Version
	}

	metadata := imageBuildCacheMetadata{
		ImageName:    imageName,
		Base:         base,
		BaseVersion:  baseVersion,
		BuildVersion: strings.TrimSpace(version),
		Formats:      normalizeImageCacheValues(cfg.Formats),
		Packages:     normalizeImageCacheValues(cfg.Packages),
		Mounts:       normalizeImageCacheValues(cfg.Mounts),
		GPU:          cfg.GPU,
		Registry:     strings.TrimSpace(cfg.Registry),
	}
	metadata.Signature = imageBuildCacheSignature(metadata)
	return metadata
}

func imageBuildCacheSignature(metadata imageBuildCacheMetadata) string {
	parts := []string{
		metadata.ImageName,
		metadata.Base,
		metadata.BaseVersion,
		core.Join(",", metadata.Formats...),
		core.Join(",", metadata.Packages...),
		core.Join(",", metadata.Mounts...),
		core.Sprintf("%t", metadata.GPU),
		metadata.Registry,
	}

	sum := sha256.Sum256([]byte(core.Join("\n", parts...)))
	return hex.EncodeToString(sum[:])
}

func normalizeImageCacheValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	sort.Strings(result)
	return result
}
