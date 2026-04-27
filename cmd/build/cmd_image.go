package buildcmd

import (
	"context"
	"io/fs" // AX-6: fs.FileMode is structural for core/io.Medium.WriteMode.
	"slices"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/internal/cmdutil"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/builders"
	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

type immutableImageVersion struct {
	BuildVersion  string
	RetainVersion string
	CacheVersion  string
}

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

	buildConfig, err := build.LoadConfig(coreio.Local, projectDir)
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

	versionInfo := resolveImmutableImageVersion(ctx, projectDir)
	version := versionInfo.BuildVersion
	if err := build.ValidateVersionIdentifier(version); err != nil {
		return coreerr.E("build.runBuildImage", "unsafe release tag detected for immutable image", err)
	}

	imageName := buildConfig.LinuxKit.Base
	if imageName == "" {
		imageName = build.DefaultLinuxKitConfig().Base
	}

	runtimeCfg := &build.Config{
		FS:         coreio.Local,
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

	artifacts := cachedImageArtifacts(imageBuilder, outputDir, imageName, formats)
	usedCache := !req.Rebuild && allImageArtifactsExist(coreio.Local, imageBuilder, outputDir, imageName, cacheCfg, versionInfo.CacheVersion)
	if usedCache {
		cli.Print("%s %s\n", buildSuccessStyle.Render("Using"), "cached immutable image artifacts")
	} else {
		artifacts, err = imageBuilder.Build(ctx, runtimeCfg)
		if err != nil {
			return err
		}
		if err := writeImageBuildCacheMetadata(coreio.Local, outputDir, imageName, cacheCfg, versionInfo.CacheVersion); err != nil {
			return coreerr.E("build.runBuildImage", "failed to write image cache metadata", err)
		}
	}

	versionedArtifacts, err := retainVersionedImageArtifacts(coreio.Local, artifacts, versionInfo.RetainVersion)
	if err != nil {
		return coreerr.E("build.runBuildImage", "failed to retain versioned immutable image artifacts", err)
	}

	publishedRef := ""
	if containsImageFormat(formats, "oci") && core.Trim(runtimeCfg.LinuxKit.Registry) != "" {
		ociArtifactPath := imageBuilder.ArtifactPath(outputDir, imageName, "oci")
		publishedRef, err = publishOCIImageArchive(ctx, projectDir, ociArtifactPath, runtimeCfg.LinuxKit.Registry, imageName, version)
		if err != nil {
			return err
		}
	}

	if !usedCache {
		cli.Print("%s %s\n", buildSuccessStyle.Render("Built"), buildTargetStyle.Render(imageName))
	}
	for _, artifact := range artifacts {
		relPath, relErr := ax.Rel(projectDir, artifact.Path)
		if relErr != nil {
			relPath = artifact.Path
		}
		cli.Print("  %s\n", relPath)
	}
	for _, artifactPath := range versionedArtifacts {
		relPath, relErr := ax.Rel(projectDir, artifactPath)
		if relErr != nil {
			relPath = artifactPath
		}
		cli.Print("  %s\n", relPath)
	}
	if publishedRef != "" {
		cli.Print("%s %s\n", buildSuccessStyle.Render("Published"), buildTargetStyle.Render(publishedRef))
	}

	return nil
}

func resolveImmutableImageVersion(ctx context.Context, projectDir string) immutableImageVersion {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := ax.LookPath("git"); err != nil {
		return immutableImageVersion{BuildVersion: "dev"}
	}

	tag, err := ax.RunDir(ctx, projectDir, "git", "describe", "--tags", "--exact-match", "HEAD")
	if err != nil {
		return immutableImageVersion{BuildVersion: "dev"}
	}

	tag = core.Trim(tag)
	if tag == "" {
		return immutableImageVersion{BuildVersion: "dev"}
	}
	if !core.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	return immutableImageVersion{
		BuildVersion:  tag,
		RetainVersion: tag,
		CacheVersion:  tag,
	}
}

func parseImageFormats(value string) []string {
	if value == "" {
		return nil
	}

	parts := core.Split(value, ",")
	formats := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = core.Lower(core.Trim(part))
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

func cachedImageArtifacts(imageBuilder *builders.LinuxKitImageBuilder, outputDir, imageName string, formats []string) []build.Artifact {
	artifacts := make([]build.Artifact, 0, len(formats))
	for _, format := range formats {
		format = core.Trim(format)
		if format == "" {
			continue
		}
		artifacts = append(artifacts, build.Artifact{
			Path: imageBuilder.ArtifactPath(outputDir, imageName, format),
			OS:   "linux",
			Arch: core.Env("ARCH"),
		})
	}
	return artifacts
}

func containsImageFormat(formats []string, want string) bool {
	want = core.Lower(core.Trim(want))
	for _, format := range formats {
		if core.Lower(core.Trim(format)) == want {
			return true
		}
	}
	return false
}

func retainVersionedImageArtifacts(filesystem coreio.Medium, artifacts []build.Artifact, version string) ([]string, error) {
	versionTag := normalizeImageVersionTag(version)
	if versionTag == "" {
		return nil, nil
	}

	versionedPaths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Path == "" {
			continue
		}
		versionedPath := versionedImageArtifactPath(artifact.Path, versionTag)
		if versionedPath == artifact.Path {
			continue
		}
		if err := copyImageArtifact(filesystem, artifact.Path, versionedPath); err != nil {
			return nil, err
		}
		versionedPaths = append(versionedPaths, versionedPath)
	}

	return versionedPaths, nil
}

func versionedImageArtifactPath(path, versionTag string) string {
	if path == "" || versionTag == "" {
		return path
	}

	ext := ax.Ext(path)
	base := core.TrimSuffix(ax.Base(path), ext)
	return ax.Join(ax.Dir(path), base+"-"+versionTag+ext)
}

func normalizeImageVersionTag(version string) string {
	version = core.Trim(version)
	version = core.TrimPrefix(version, "v")
	if version == "" {
		return ""
	}

	version = core.Replace(version, "/", "-")
	version = core.Replace(version, "\\", "-")
	version = core.Replace(version, ":", "-")
	version = core.Replace(version, " ", "-")
	version = core.Replace(version, "\t", "-")
	return trimImageVersionTagEdges(version)
}

func trimImageVersionTagEdges(version string) string {
	start := 0
	for start < len(version) && isImageVersionTagEdge(version[start]) {
		start++
	}

	end := len(version)
	for end > start && isImageVersionTagEdge(version[end-1]) {
		end--
	}

	return version[start:end]
}

func isImageVersionTagEdge(ch byte) bool {
	return ch == '-' || ch == '.'
}

func copyImageArtifact(filesystem coreio.Medium, sourcePath, destinationPath string) error {
	content, err := filesystem.Read(sourcePath)
	if err != nil {
		return err
	}

	mode := fs.FileMode(0o644)
	if info, err := filesystem.Stat(sourcePath); err == nil {
		mode = info.Mode()
	}

	return filesystem.WriteMode(destinationPath, content, mode)
}

func publishOCIImageArchive(ctx context.Context, projectDir, artifactPath, registry, imageName, version string) (string, error) {
	if core.Trim(registry) == "" || core.Trim(artifactPath) == "" {
		return "", nil
	}

	dockerCommand, err := resolveImageDockerCli()
	if err != nil {
		return "", coreerr.E("build.runBuildImage", "failed to resolve docker CLI for OCI publish", err)
	}

	destinationRef := resolveOCIImageReference(registry, imageName, version)
	sourceRef, err := loadOCIImageArchive(ctx, projectDir, dockerCommand, artifactPath)
	if err != nil {
		return "", err
	}

	if sourceRef != destinationRef {
		if err := ax.ExecWithEnv(ctx, projectDir, nil, dockerCommand, "image", "tag", sourceRef, destinationRef); err != nil {
			return "", coreerr.E("build.runBuildImage", "failed to tag OCI image for registry publish", err)
		}
	}

	if err := ax.ExecWithEnv(ctx, projectDir, nil, dockerCommand, "image", "push", destinationRef); err != nil {
		return "", coreerr.E("build.runBuildImage", "failed to push OCI image to registry", err)
	}

	return destinationRef, nil
}

func resolveImageDockerCli() (string, error) {
	return ax.ResolveCommand("docker",
		"/usr/local/bin/docker",
		"/opt/homebrew/bin/docker",
		"/Applications/Docker.app/Contents/Resources/bin/docker",
	)
}

func resolveOCIImageReference(registry, imageName, version string) string {
	tag := normalizeImageVersionTag(version)
	if tag == "" {
		tag = "dev"
	}

	registry = trimTrailingImageRegistrySlashes(core.Trim(registry))
	if registry == "" {
		return imageName + ":" + tag
	}

	return registry + "/" + imageName + ":" + tag
}

func trimTrailingImageRegistrySlashes(registry string) string {
	for core.HasSuffix(registry, "/") {
		registry = core.TrimSuffix(registry, "/")
	}
	return registry
}

func loadOCIImageArchive(ctx context.Context, projectDir, dockerCommand, artifactPath string) (string, error) {
	output, err := ax.CombinedOutput(ctx, projectDir, nil, dockerCommand, "image", "load", "--input", artifactPath)
	if err != nil {
		return "", coreerr.E("build.runBuildImage", "failed to load OCI image archive", err)
	}

	reference := parseLoadedDockerImageReference(output)
	if reference == "" {
		return "", coreerr.E("build.runBuildImage", "docker image load did not report a loaded image reference", nil)
	}

	return reference, nil
}

func parseLoadedDockerImageReference(output string) string {
	for _, line := range core.Split(output, "\n") {
		line = core.Trim(line)
		switch {
		case core.HasPrefix(line, "Loaded image:"):
			return core.Trim(core.TrimPrefix(line, "Loaded image:"))
		case core.HasPrefix(line, "Loaded image ID:"):
			return core.Trim(core.TrimPrefix(line, "Loaded image ID:"))
		}
	}
	return ""
}

func allImageArtifactsExist(filesystem coreio.Medium, imageBuilder *builders.LinuxKitImageBuilder, outputDir, imageName string, cfg build.LinuxKitConfig, version string) bool {
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

	expectedVersion := core.Trim(expected.BuildVersion)
	if expectedVersion == "" {
		return true
	}

	return core.Trim(metadata.BuildVersion) == expectedVersion
}

func writeImageBuildCacheMetadata(filesystem coreio.Medium, outputDir, imageName string, cfg build.LinuxKitConfig, version string) error {
	metadata := buildImageCacheMetadata(imageName, cfg, version)
	encoded, err := ax.JSONMarshal(metadata)
	if err != nil {
		return err
	}
	return filesystem.Write(imageBuildCacheMetadataPath(outputDir, imageName), encoded)
}

func loadImageBuildCacheMetadata(filesystem coreio.Medium, outputDir, imageName string) (*imageBuildCacheMetadata, error) {
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
		BuildVersion: core.Trim(version),
		Formats:      normalizeImageCacheValues(cfg.Formats),
		Packages:     normalizeImageCacheValues(cfg.Packages),
		Mounts:       normalizeImageCacheValues(cfg.Mounts),
		GPU:          cfg.GPU,
		Registry:     core.Trim(cfg.Registry),
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

	return core.SHA256Hex([]byte(core.Join("\n", parts...)))
}

func normalizeImageCacheValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = core.Trim(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	slices.SortFunc(result, func(a, b string) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	})
	return result
}
