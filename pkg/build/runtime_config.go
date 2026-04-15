package build

import "dappco.re/go/core/io"

// RuntimeConfigFromBuildConfig maps persisted build settings onto a runtime
// builder config while preserving the caller's output/name/version overrides.
func RuntimeConfigFromBuildConfig(filesystem io.Medium, projectDir, outputDir, binaryName string, buildConfig *BuildConfig, push bool, imageName string, version string) *Config {
	if buildConfig == nil {
		buildConfig = DefaultConfig()
	}

	buildDefaults := buildConfig.Build
	cfg := &Config{
		FS:             filesystem,
		Project:        buildConfig.Project,
		ProjectDir:     projectDir,
		OutputDir:      outputDir,
		Name:           binaryName,
		Version:        version,
		LDFlags:        ExpandVersionTemplates(append([]string{}, buildDefaults.LDFlags...), version),
		Flags:          ExpandVersionTemplates(append([]string{}, buildDefaults.Flags...), version),
		BuildTags:      append([]string{}, buildDefaults.BuildTags...),
		Env:            ExpandVersionTemplates(append([]string{}, buildDefaults.Env...), version),
		Cache:          buildDefaults.Cache,
		CGO:            buildDefaults.CGO,
		Obfuscate:      buildDefaults.Obfuscate,
		DenoBuild:      buildDefaults.DenoBuild,
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
