// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"text/template" // AX-6 intrinsic: no core template primitive.

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// LinuxKitImageBuilder renders and builds immutable LinuxKit base images.
type LinuxKitImageBuilder struct{}

// LinuxKitImageTemplateData is the template input for embedded immutable image definitions.
type LinuxKitImageTemplateData struct {
	Name              string
	Description       string
	Version           string
	GPU               bool
	Mounts            []string
	ServiceImage      string
	EntrypointCommand string
}

// NewLinuxKitImageBuilder creates an immutable LinuxKit image builder.
func NewLinuxKitImageBuilder() *LinuxKitImageBuilder {
	return &LinuxKitImageBuilder{}
}

// Name returns the builder identifier.
func (b *LinuxKitImageBuilder) Name() string {
	return "linuxkit-image"
}

// ListBaseImages returns the built-in immutable LinuxKit base images.
func (b *LinuxKitImageBuilder) ListBaseImages() []build.LinuxKitBaseImage {
	return build.LinuxKitBaseImages()
}

// ArtifactPath returns the final output path for a requested immutable image format.
func (b *LinuxKitImageBuilder) ArtifactPath(outputDir, name, format string) string {
	return ax.Join(outputDir, name+b.outputExtension(format))
}

// Build renders the embedded LinuxKit template and emits one artifact per format.
func (b *LinuxKitImageBuilder) Build(ctx context.Context, cfg *build.Config) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("LinuxKitImageBuilder.Build", "build config is required", nil)
	}

	filesystem := cfg.FS
	if filesystem == nil {
		filesystem = io.Local
	}

	imageCfg := mergeLinuxKitImageConfig(build.DefaultLinuxKitConfig(), cfg.LinuxKit)
	baseImage, ok := build.LookupLinuxKitBaseImage(imageCfg.Base)
	if !ok {
		return nil, coreerr.E("LinuxKitImageBuilder.Build", "unknown LinuxKit image base: "+imageCfg.Base, nil)
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ax.Join(cfg.ProjectDir, "dist")
	}
	if !ax.IsAbs(outputDir) && cfg.ProjectDir != "" {
		outputDir = ax.Join(cfg.ProjectDir, outputDir)
	}
	if err := filesystem.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("LinuxKitImageBuilder.Build", "failed to create output directory", err)
	}

	imageName := cfg.Name
	if imageName == "" {
		imageName = imageCfg.Base
	}

	serviceImage, cleanup, err := b.prepareServiceImage(ctx, cfg.ProjectDir, imageName, cfg.Version, baseImage, imageCfg)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	renderedTemplate, err := b.renderTemplate(baseImage, imageCfg, cfg.Version, serviceImage)
	if err != nil {
		return nil, err
	}

	templatePath := ax.Join(outputDir, "."+imageName+"-linuxkit.yml")
	if err := ax.WriteFile(templatePath, []byte(renderedTemplate), 0o644); err != nil {
		return nil, coreerr.E("LinuxKitImageBuilder.Build", "failed to write LinuxKit template", err)
	}
	defer func() { _ = filesystem.Delete(templatePath) }()

	linuxkitCommand, err := (&LinuxKitBuilder{}).resolveLinuxKitCli()
	if err != nil {
		return nil, err
	}

	formats := imageCfg.Formats
	if len(formats) == 0 {
		formats = append([]string(nil), build.DefaultLinuxKitConfig().Formats...)
	}

	artifacts := make([]build.Artifact, 0, len(formats))
	for _, format := range formats {
		if format == "" {
			continue
		}

		artifactPath, err := b.buildFormat(ctx, filesystem, linuxkitCommand, cfg.ProjectDir, outputDir, imageName, templatePath, format)
		if err != nil {
			return nil, err
		}

		artifacts = append(artifacts, build.Artifact{
			Path: artifactPath,
			OS:   "linux",
			Arch: core.Env("ARCH"),
		})
	}

	return artifacts, nil
}

func mergeLinuxKitImageConfig(defaults, override build.LinuxKitConfig) build.LinuxKitConfig {
	cfg := defaults
	if override.Base != "" {
		cfg.Base = override.Base
	}
	if override.Packages != nil {
		cfg.Packages = append([]string(nil), override.Packages...)
	}
	if override.Mounts != nil {
		cfg.Mounts = append([]string(nil), override.Mounts...)
	}
	cfg.GPU = override.GPU
	if override.Formats != nil {
		cfg.Formats = append([]string(nil), override.Formats...)
	}
	if override.Registry != "" {
		cfg.Registry = override.Registry
	}
	return normalizeLinuxKitImageConfig(cfg)
}

func normalizeLinuxKitImageConfig(cfg build.LinuxKitConfig) build.LinuxKitConfig {
	defaults := build.DefaultLinuxKitConfig()

	cfg.Base = core.Trim(cfg.Base)
	if cfg.Base == "" {
		cfg.Base = defaults.Base
	}

	cfg.Registry = core.Trim(cfg.Registry)
	cfg.Packages = uniqueStrings(cfg.Packages)
	cfg.Mounts = uniqueStrings(cfg.Mounts)
	if len(cfg.Mounts) == 0 {
		cfg.Mounts = append([]string(nil), defaults.Mounts...)
	}

	cfg.Formats = normalizeLinuxKitImageFormats(cfg.Formats)
	if len(cfg.Formats) == 0 {
		cfg.Formats = append([]string(nil), defaults.Formats...)
	}

	return cfg
}

func normalizeLinuxKitImageFormats(values []string) []string {
	if len(values) == 0 {
		return values
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = core.Lower(core.Trim(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func (b *LinuxKitImageBuilder) renderTemplate(baseImage build.LinuxKitBaseImage, cfg build.LinuxKitConfig, version, serviceImage string) (string, error) {
	cfg = normalizeLinuxKitImageConfig(cfg)

	templateContent, err := build.LinuxKitBaseTemplate(baseImage.Name)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(baseImage.Name).Parse(templateContent)
	if err != nil {
		return "", coreerr.E("LinuxKitImageBuilder.renderTemplate", "failed to parse embedded LinuxKit template", err)
	}

	if version == "" {
		version = "dev"
	}

	data := LinuxKitImageTemplateData{
		Name:              baseImage.Name,
		Description:       baseImage.Description,
		Version:           version,
		GPU:               cfg.GPU,
		Mounts:            uniqueStrings(cfg.Mounts),
		ServiceImage:      serviceImage,
		EntrypointCommand: "tail -f /dev/null",
	}

	rendered := core.NewBuffer()
	if err := tmpl.Execute(rendered, data); err != nil {
		return "", coreerr.E("LinuxKitImageBuilder.renderTemplate", "failed to render LinuxKit template", err)
	}

	return rendered.String(), nil
}

func (b *LinuxKitImageBuilder) prepareServiceImage(ctx context.Context, projectDir, imageName, version string, baseImage build.LinuxKitBaseImage, cfg build.LinuxKitConfig) (string, func(), error) {
	cfg = normalizeLinuxKitImageConfig(cfg)

	dockerCommand, err := (&DockerBuilder{}).resolveDockerCli()
	if err != nil {
		return "", func() {}, coreerr.E("LinuxKitImageBuilder.prepareServiceImage", "failed to resolve docker CLI for immutable service image build", err)
	}

	tempDir, err := ax.TempDir("core-build-linuxkit-service-*")
	if err != nil {
		return "", func() {}, coreerr.E("LinuxKitImageBuilder.prepareServiceImage", "failed to create service image build context", err)
	}

	cleanup := func() {
		_ = ax.RemoveAll(tempDir)
	}

	contentHash := linuxKitServiceImageContentHash(baseImage, cfg)
	serviceImage := buildLinuxKitServiceImageReference(imageName, version)
	mounts := uniqueStrings(append([]string{"/workspace"}, cfg.Mounts...))
	dockerfile := renderLinuxKitServiceDockerfile(
		imageName,
		version,
		baseImage.Version,
		contentHash,
		append(append([]string{}, baseImage.DefaultPackages...), cfg.Packages...),
		mounts,
		cfg.GPU,
	)
	if err := ax.WriteString(ax.Join(tempDir, "Dockerfile"), dockerfile, 0o644); err != nil {
		cleanup()
		return "", func() {}, coreerr.E("LinuxKitImageBuilder.prepareServiceImage", "failed to write service image Dockerfile", err)
	}

	if err := ax.ExecDir(ctx, tempDir, dockerCommand, "build", "-t", serviceImage, "."); err != nil {
		cleanup()
		return "", func() {}, coreerr.E("LinuxKitImageBuilder.prepareServiceImage", "failed to build immutable LinuxKit service image", err)
	}

	return serviceImage, cleanup, nil
}

func renderLinuxKitServiceDockerfile(imageName, version, baseVersion, contentHash string, packages, mounts []string, gpu bool) string {
	lines := []string{
		"FROM alpine:3.19",
	}

	packages = uniqueStrings(packages)
	if len(packages) > 0 {
		lines = append(lines, "RUN apk add --no-cache "+core.Join(" ", packages...))
	}

	mounts = uniqueStrings(append([]string{"/workspace"}, mounts...))
	if len(mounts) > 0 {
		lines = append(lines, "RUN mkdir -p "+core.Join(" ", mounts...))
	}

	if gpu {
		lines = append(lines, "RUN mkdir -p /etc/profile.d && printf 'export CORE_GPU=1\\n' > /etc/profile.d/core-gpu.sh")
	}

	lines = append(lines,
		"WORKDIR /workspace",
		"LABEL org.opencontainers.image.title="+imageName,
		"LABEL org.opencontainers.image.version="+normalizeLinuxKitServiceVersionTag(version),
		"LABEL dappcore.core-build.base-version="+normalizeLinuxKitServiceTag(baseVersion),
		"LABEL dappcore.core-build.content-hash="+normalizeLinuxKitServiceTag(contentHash),
		"ENV CORE_IMAGE="+imageName,
		"ENV CORE_IMAGE_VERSION="+normalizeLinuxKitServiceVersionTag(version),
		"ENV CORE_IMAGE_BASE_VERSION="+normalizeLinuxKitServiceTag(baseVersion),
		"ENV CORE_IMAGE_CONTENT_HASH="+normalizeLinuxKitServiceTag(contentHash),
		core.Sprintf("ENV CORE_GPU=%d", boolToInt(gpu)),
		`CMD ["/bin/sh", "-lc", "tail -f /dev/null"]`,
	)

	return core.Join("\n", lines...) + "\n"
}

func buildLinuxKitServiceImageReference(imageName, version string) string {
	tag := normalizeLinuxKitServiceVersionTag(version)
	return core.Sprintf("core-build-linuxkit/%s:%s", imageName, tag)
}

func linuxKitServiceImageContentHash(baseImage build.LinuxKitBaseImage, cfg build.LinuxKitConfig) string {
	cfg = normalizeLinuxKitImageConfig(cfg)
	parts := []string{
		baseImage.Name,
		baseImage.Version,
		core.Join(",", uniqueStrings(baseImage.DefaultPackages)...),
		core.Join(",", uniqueStrings(cfg.Packages)...),
		core.Join(",", uniqueStrings(cfg.Mounts)...),
		core.Sprintf("%t", cfg.GPU),
	}
	sum := core.SHA256([]byte(core.Join("\n", parts...)))
	return core.HexEncode(sum[:6])
}

func normalizeLinuxKitServiceVersionTag(value string) string {
	value = core.Trim(value)
	value = core.TrimPrefix(value, "v")
	if value == "" {
		value = "dev"
	}
	return normalizeLinuxKitServiceTag(value)
}

func normalizeLinuxKitServiceTag(value string) string {
	value = core.Lower(core.Trim(value))
	value = core.Replace(value, "/", "-")
	value = core.Replace(value, "\\", "-")
	value = core.Replace(value, ":", "-")
	value = core.Replace(value, " ", "-")
	value = core.Replace(value, "\t", "-")
	value = core.Replace(value, "_", "-")
	value = core.Replace(value, "..", ".")
	value = trimLinuxKitServiceTagBoundary(value)
	if value == "" {
		return "latest"
	}
	return value
}

func trimLinuxKitServiceTagBoundary(value string) string {
	for value != "" {
		switch {
		case core.HasPrefix(value, "-"):
			value = core.TrimPrefix(value, "-")
		case core.HasPrefix(value, "."):
			value = core.TrimPrefix(value, ".")
		case core.HasSuffix(value, "-"):
			value = core.TrimSuffix(value, "-")
		case core.HasSuffix(value, "."):
			value = core.TrimSuffix(value, ".")
		default:
			return value
		}
	}
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
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
	return result
}

func (b *LinuxKitImageBuilder) buildFormat(ctx context.Context, filesystem io.Medium, linuxkitCommand, projectDir, outputDir, imageName, templatePath, format string) (string, error) {
	linuxKitFormat := b.linuxKitFormat(format)
	buildName := imageName
	if format == "apple" {
		buildName = imageName + "-apple"
	}

	args := []string{
		"build",
		"--format", linuxKitFormat,
		"--name", buildName,
		"--dir", outputDir,
		templatePath,
	}

	if err := ax.ExecWithEnv(ctx, projectDir, nil, linuxkitCommand, args...); err != nil {
		return "", coreerr.E("LinuxKitImageBuilder.Build", "build failed for "+format, err)
	}

	builtPath := ax.Join(outputDir, buildName+b.intermediateExtension(format))
	finalPath := b.ArtifactPath(outputDir, imageName, format)

	if format == "apple" {
		if !filesystem.Exists(builtPath) {
			return "", coreerr.E("LinuxKitImageBuilder.Build", "apple container artifact not found: "+builtPath, nil)
		}
		if err := filesystem.Rename(builtPath, finalPath); err != nil {
			return "", coreerr.E("LinuxKitImageBuilder.Build", "failed to rename Apple container artifact", err)
		}
		return finalPath, nil
	}

	if !filesystem.Exists(finalPath) {
		return "", coreerr.E("LinuxKitImageBuilder.Build", "artifact not found after build: "+finalPath, nil)
	}

	return finalPath, nil
}

func (b *LinuxKitImageBuilder) linuxKitFormat(format string) string {
	switch format {
	case "oci", "apple":
		return "tar"
	default:
		return format
	}
}

func (b *LinuxKitImageBuilder) intermediateExtension(format string) string {
	switch format {
	case "oci", "apple":
		return ".tar"
	default:
		return b.outputExtension(format)
	}
}

func (b *LinuxKitImageBuilder) outputExtension(format string) string {
	switch format {
	case "oci":
		return ".tar"
	case "apple":
		return ".aci"
	default:
		return (&LinuxKitBuilder{}).getFormatExtension(format)
	}
}
