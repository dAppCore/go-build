// Package builders provides build implementations for different project types.
package builders

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"text/template"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// LinuxKitImageBuilder renders and builds immutable LinuxKit base images.
type LinuxKitImageBuilder struct{}

// LinuxKitImageTemplateData is the template input for embedded immutable image definitions.
type LinuxKitImageTemplateData struct {
	Name             string
	Description      string
	Version          string
	GPU              bool
	Mounts           []string
	BootstrapCommand string
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

	renderedTemplate, err := b.renderTemplate(baseImage, imageCfg, cfg.Version)
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
			Arch: runtime.GOARCH,
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
	return cfg
}

func (b *LinuxKitImageBuilder) renderTemplate(baseImage build.LinuxKitBaseImage, cfg build.LinuxKitConfig, version string) (string, error) {
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
		Name:             baseImage.Name,
		Description:      baseImage.Description,
		Version:          version,
		GPU:              cfg.GPU,
		Mounts:           uniqueStrings(cfg.Mounts),
		BootstrapCommand: buildBootstrapCommand(baseImage.DefaultPackages, cfg.Packages, cfg.GPU),
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", coreerr.E("LinuxKitImageBuilder.renderTemplate", "failed to render LinuxKit template", err)
	}

	return rendered.String(), nil
}

func buildBootstrapCommand(defaultPackages, extraPackages []string, gpu bool) string {
	packages := uniqueStrings(append(append([]string{}, defaultPackages...), extraPackages...))
	commands := make([]string, 0, 4)
	if len(packages) > 0 {
		commands = append(commands, "apk add --no-cache "+core.Join(" ", packages...))
	}
	if gpu {
		commands = append(commands, "mkdir -p /etc/profile.d")
		commands = append(commands, `printf 'export CORE_GPU=1\n' > /etc/profile.d/core-gpu.sh`)
	}
	commands = append(commands, "mkdir -p /workspace")
	commands = append(commands, "tail -f /dev/null")
	return core.Join(" && ", commands...)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
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
