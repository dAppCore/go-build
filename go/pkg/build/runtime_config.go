package build

import (
	core "dappco.re/go"
	storage "dappco.re/go/build/pkg/storage"
)

// RuntimeConfigFromBuildConfig maps persisted build settings onto a runtime
// builder config while preserving the caller's output/name/version overrides.
func RuntimeConfigFromBuildConfig(filesystem storage.Medium, projectDir, outputDir, binaryName string, buildConfig *BuildConfig, push bool, imageName string, version string) *Config {
	if buildConfig == nil {
		buildConfig = DefaultConfig()
	}

	buildDefaults := buildConfig.Build
	denoBuild := buildDefaults.DenoBuild
	if denoBuild == "" {
		denoBuild = buildConfig.PreBuild.Deno
	}
	npmBuild := buildDefaults.NpmBuild
	if npmBuild == "" {
		npmBuild = buildConfig.PreBuild.Npm
	}

	versionSafe := version == "" || versionIsSafeRelease(version)

	ldFlags := append([]string{}, buildDefaults.LDFlags...)
	if version == "" {
		// Preserve template placeholders when no version is being injected.
	} else if versionSafe {
		ldFlags = ExpandVersionTemplates(ldFlags, version)
	} else {
		ldFlags = stripVersionTemplateFlags(ldFlags)
	}

	flags := append([]string{}, buildDefaults.Flags...)
	if versionSafe {
		flags = ExpandVersionTemplates(flags, version)
	} else if version != "" {
		flags = stripVersionTemplateValues(flags)
	}

	env := append([]string{}, buildDefaults.Env...)
	if versionSafe {
		env = ExpandVersionTemplates(env, version)
	} else if version != "" {
		env = stripVersionTemplateValues(env)
	}

	cfg := &Config{
		FS:             filesystem,
		Project:        buildConfig.Project,
		ProjectDir:     projectDir,
		OutputDir:      outputDir,
		Name:           binaryName,
		Version:        version,
		LDFlags:        ldFlags,
		Flags:          flags,
		BuildTags:      append([]string{}, buildDefaults.BuildTags...),
		Env:            env,
		Cache:          buildDefaults.Cache,
		CGO:            buildDefaults.CGO,
		Obfuscate:      buildDefaults.Obfuscate,
		DenoBuild:      denoBuild,
		NpmBuild:       npmBuild,
		NSIS:           buildDefaults.NSIS,
		WebView2:       buildDefaults.WebView2,
		Dockerfile:     buildDefaults.Dockerfile,
		Registry:       buildDefaults.Registry,
		Image:          buildDefaults.Image,
		Tags:           append([]string{}, buildDefaults.Tags...),
		BuildArgs:      CloneStringMap(buildDefaults.BuildArgs),
		Push:           buildDefaults.Push || push,
		Load:           buildDefaults.Load,
		LinuxKitConfig: buildDefaults.LinuxKitConfig,
		Formats:        append([]string{}, buildDefaults.Formats...),
		LinuxKit:       cloneLinuxKitConfig(buildConfig.LinuxKit),
	}

	if imageName != "" {
		cfg.Image = imageName
	}

	return cfg
}

func versionIsSafeRelease(version string) bool {
	return ValidateVersionString(version).OK
}

func stripVersionTemplateFlags(values []string) []string {
	if len(values) == 0 {
		return values
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if containsVersionTemplate(value) {
			continue
		}
		filtered = append(filtered, value)
	}

	return filtered
}

func stripVersionTemplateValues(values []string) []string {
	if len(values) == 0 {
		return values
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if containsVersionTemplate(value) {
			continue
		}
		filtered = append(filtered, value)
	}

	return filtered
}

func containsVersionTemplate(value string) bool {
	return core.Contains(value, "v{{.Version}}") ||
		core.Contains(value, "v{{Version}}") ||
		core.Contains(value, "{{.Tag}}") ||
		core.Contains(value, "{{Tag}}") ||
		core.Contains(value, "{{.Version}}") ||
		core.Contains(value, "{{Version}}")
}
