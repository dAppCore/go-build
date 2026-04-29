package apple

import (
	"context"
	"regexp"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	build "dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/release"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// AppleOptions aliases the core Apple pipeline options.
type AppleOptions = build.AppleOptions

// WailsBuildConfig mirrors the RFC-facing Apple wrapper input shape.
// The wrapper keeps LDFlags as a single string while the lower-level build
// package accepts a slice for direct CLI assembly.
type WailsBuildConfig struct {
	ProjectDir string   `json:"project_dir" yaml:"project_dir"`
	Name       string   `json:"name" yaml:"name"`
	Arch       string   `json:"arch" yaml:"arch"`
	BuildTags  []string `json:"build_tags" yaml:"build_tags"`
	LDFlags    string   `json:"ldflags" yaml:"ldflags"`
	OutputDir  string   `json:"output_dir" yaml:"output_dir"`
	Version    string   `json:"version" yaml:"version"`
	Env        []string `json:"env" yaml:"env"`
	DenoBuild  string   `json:"deno_build" yaml:"deno_build"`
}

// SignConfig aliases the codesign configuration.
type SignConfig = build.SignConfig

// NotariseConfig aliases the notarisation configuration.
type NotariseConfig = build.NotariseConfig

// DMGConfig aliases the DMG packaging configuration.
type DMGConfig = build.DMGConfig

// TestFlightConfig aliases the TestFlight upload configuration.
type TestFlightConfig = build.TestFlightConfig

// AppStoreConfig aliases the App Store Connect submission configuration.
type AppStoreConfig = build.AppStoreConfig

// InfoPlist aliases the generated Info.plist model.
type InfoPlist = build.InfoPlist

// Entitlements aliases the generated entitlements model.
type Entitlements = build.Entitlements

// XcodeCloudConfig aliases the Xcode Cloud workflow metadata stored in build config.
type XcodeCloudConfig = build.XcodeCloudConfig

// XcodeCloudTrigger aliases a single Xcode Cloud trigger rule.
type XcodeCloudTrigger = build.XcodeCloudTrigger

// Builder defines the RFC-facing Apple builder contract.
type Builder interface {
	Name() string
	Detect(fs coreio.Medium, dir string) core.Result
	Build(ctx context.Context, cfg *AppleOptions) core.Result
}

// AppleBuilder wraps the existing Apple pipeline with functional options.
type AppleBuilder struct {
	*core.ServiceRuntime[AppleOptions]
	options  AppleOptions
	explicit explicitOptions
}

type explicitOptions struct {
	arch       bool
	sign       bool
	notarise   bool
	dmg        bool
	testFlight bool
	appStore   bool
}

// Option configures Apple pipeline defaults for a new AppleBuilder.
type Option func(*AppleOptions)

var (
	loadConfigFn             = build.LoadConfig
	buildAppleFn             = build.BuildApple
	determineVersion         = release.DetermineVersionWithContext
	getwdFn                  = ax.Getwd
	runDirFn                 = ax.RunDir
	buildWailsAppFn          = build.BuildWailsApp
	createUniversalFn        = build.CreateUniversal
	signFn                   = build.Sign
	notariseFn               = build.Notarise
	createDMGFn              = build.CreateDMG
	uploadTFn                = build.UploadTestFlight
	submitASFn               = build.SubmitAppStore
	writeXcodeCloudScriptsFn = build.WriteXcodeCloudScripts
)

// Register wires AppleBuilder into the Core service container and seeds the
// builders registry when the host Core exposes one.
func Register(c *core.Core) core.Result {
	if c == nil {
		return core.Fail(coreerr.E("apple.Register", "core is nil", nil))
	}

	builder := New()
	builder.ServiceRuntime = core.NewServiceRuntime[AppleOptions](c, builder.options)
	if r := c.RegistryOf("builders").Set("apple", builder); !r.OK {
		return r
	}
	if r := c.RegisterService("apple", builder); !r.OK {
		return r
	}

	return core.Ok(builder)
}

// New constructs an AppleBuilder with functional options.
func New(opts ...Option) *AppleBuilder {
	builder := &AppleBuilder{
		options: build.DefaultAppleOptions(),
	}
	for _, opt := range opts {
		builder.applyOption(opt)
	}
	builder.ServiceRuntime = core.NewServiceRuntime[AppleOptions](nil, builder.options)
	return builder
}

// WithArch sets the target architecture.
func WithArch(arch string) Option {
	return func(options *AppleOptions) {
		if options == nil {
			return
		}
		options.Arch = arch
	}
}

// WithSign enables or disables code signing.
func WithSign(sign bool) Option {
	return func(options *AppleOptions) {
		if options == nil {
			return
		}
		options.Sign = sign
	}
}

// WithNotarise enables or disables notarisation.
func WithNotarise(notarise bool) Option {
	return func(options *AppleOptions) {
		if options == nil {
			return
		}
		options.Notarise = notarise
	}
}

// WithDMG enables or disables DMG creation.
func WithDMG(dmg bool) Option {
	return func(options *AppleOptions) {
		if options == nil {
			return
		}
		options.DMG = dmg
	}
}

// WithTestFlight enables or disables TestFlight upload.
func WithTestFlight(tf bool) Option {
	return func(options *AppleOptions) {
		if options == nil {
			return
		}
		options.TestFlight = tf
	}
}

// WithAppStore enables or disables App Store submission.
func WithAppStore(appStore bool) Option {
	return func(options *AppleOptions) {
		if options == nil {
			return
		}
		options.AppStore = appStore
	}
}

// Name returns the builder identifier.
func (b *AppleBuilder) Name() string {
	return "apple"
}

// Detect reports whether the current directory looks like a Wails-backed Apple target.
func (b *AppleBuilder) Detect(fs coreio.Medium, dir string) core.Result {
	if fs == nil {
		fs = coreio.Local
	}
	return core.Ok(build.IsWailsProject(fs, dir))
}

// Build runs the Apple pipeline for the current working directory and returns the .app bundle path.
func (b *AppleBuilder) Build(ctx context.Context, cfg *AppleOptions) core.Result {
	if ctx == nil {
		ctx = context.Background()
	}

	projectDirResult := getwdFn()
	if !projectDirResult.OK {
		return projectDirResult
	}
	projectDir := projectDirResult.Value.(string)

	buildConfigResult := loadConfigFn(coreio.Local, projectDir)
	if !buildConfigResult.OK {
		return buildConfigResult
	}
	buildConfig := buildConfigResult.Value.(*build.BuildConfig)
	cacheSetup := build.SetupBuildCache(coreio.Local, projectDir, buildConfig)
	if !cacheSetup.OK {
		return cacheSetup
	}
	if build.HasXcodeCloudConfig(buildConfig) {
		written := writeXcodeCloudScriptsFn(coreio.Local, projectDir, buildConfig)
		if !written.OK {
			return written
		}
	}

	versionResult := determineVersion(ctx, projectDir)
	if !versionResult.OK {
		return versionResult
	}
	version := versionResult.Value.(string)

	buildNumberResult := resolveBuildNumber(ctx, projectDir)
	if !buildNumberResult.OK {
		return buildNumberResult
	}
	buildNumber := buildNumberResult.Value.(string)

	options := b.resolveOptions(buildConfig, cfg)
	name := resolveBundleName(buildConfig, projectDir)
	outputDir := ax.Join(projectDir, "dist", "apple")
	runtimeCfg := runtimeConfig(coreio.Local, projectDir, outputDir, name, buildConfig, version)

	result := buildAppleFn(ctx, runtimeCfg, options, buildNumber)
	if !result.OK {
		return result
	}
	buildResult := result.Value.(*build.AppleBuildResult)

	return core.Ok(buildResult.BundlePath)
}

// BuildWailsApp compiles the Wails application for a single Apple architecture.
func BuildWailsApp(ctx context.Context, cfg WailsBuildConfig) core.Result {
	projectDir := cfg.ProjectDir
	if projectDir == "" {
		projectDirResult := getwdFn()
		if !projectDirResult.OK {
			return projectDirResult
		}
		projectDir = projectDirResult.Value.(string)
	}

	buildCfg := build.WailsBuildConfig{
		ProjectDir: projectDir,
		Name:       cfg.Name,
		Arch:       cfg.Arch,
		BuildTags:  append([]string{}, cfg.BuildTags...),
		OutputDir:  cfg.OutputDir,
		Version:    cfg.Version,
		Env:        append([]string{}, cfg.Env...),
		DenoBuild:  cfg.DenoBuild,
	}
	if core.Trim(cfg.LDFlags) != "" {
		buildCfg.LDFlags = []string{cfg.LDFlags}
	}

	return buildWailsAppFn(ctx, buildCfg)
}

// CreateUniversal merges arm64 and amd64 bundles into a universal bundle.
func CreateUniversal(arm64Path, amd64Path, outputPath string) core.Result {
	result := createUniversalFn(arm64Path, amd64Path, outputPath)
	if !result.OK {
		return result
	}
	return core.Ok(outputPath)
}

// Sign code-signs the given Apple artefact.
func Sign(ctx context.Context, cfg SignConfig) core.Result {
	result := signFn(ctx, cfg)
	if !result.OK {
		return result
	}
	return core.Ok(cfg.AppPath)
}

// Notarise submits the artefact for Apple notarisation.
func Notarise(ctx context.Context, cfg NotariseConfig) core.Result {
	result := notariseFn(ctx, cfg)
	if !result.OK {
		return result
	}
	return core.Ok(cfg.AppPath)
}

// CreateDMG packages the app bundle into a DMG and returns the DMG path.
func CreateDMG(ctx context.Context, cfg DMGConfig) core.Result {
	result := createDMGFn(ctx, cfg)
	if !result.OK {
		return result
	}
	return core.Ok(cfg.OutputPath)
}

// UploadTestFlight uploads the packaged build to TestFlight.
func UploadTestFlight(ctx context.Context, cfg TestFlightConfig) core.Result {
	result := uploadTFn(ctx, cfg)
	if !result.OK {
		return result
	}
	return core.Ok(cfg.AppPath)
}

// SubmitAppStore uploads the packaged build to App Store Connect.
func SubmitAppStore(ctx context.Context, cfg AppStoreConfig) core.Result {
	result := submitASFn(ctx, cfg)
	if !result.OK {
		return result
	}
	return core.Ok(cfg.AppPath)
}

func (b *AppleBuilder) applyOption(opt Option) {
	if b == nil || opt == nil {
		return
	}

	var zeroBefore AppleOptions
	zeroAfter := zeroBefore
	opt(&zeroAfter)

	defaultBefore := build.DefaultAppleOptions()
	defaultAfter := defaultBefore
	opt(&defaultAfter)

	if zeroAfter.Arch != zeroBefore.Arch || defaultAfter.Arch != defaultBefore.Arch {
		b.explicit.arch = true
	}
	if zeroAfter.Sign != zeroBefore.Sign || defaultAfter.Sign != defaultBefore.Sign {
		b.explicit.sign = true
	}
	if zeroAfter.Notarise != zeroBefore.Notarise || defaultAfter.Notarise != defaultBefore.Notarise {
		b.explicit.notarise = true
	}
	if zeroAfter.DMG != zeroBefore.DMG || defaultAfter.DMG != defaultBefore.DMG {
		b.explicit.dmg = true
	}
	if zeroAfter.TestFlight != zeroBefore.TestFlight || defaultAfter.TestFlight != defaultBefore.TestFlight {
		b.explicit.testFlight = true
	}
	if zeroAfter.AppStore != zeroBefore.AppStore || defaultAfter.AppStore != defaultBefore.AppStore {
		b.explicit.appStore = true
	}

	opt(&b.options)
}

func (b *AppleBuilder) resolveOptions(buildConfig *build.BuildConfig, runtime *AppleOptions) AppleOptions {
	options := build.DefaultAppleOptions()
	if buildConfig != nil {
		options = buildConfig.Apple.Resolve()
		options.CertIdentity = firstNonEmpty(options.CertIdentity, buildConfig.Sign.MacOS.Identity)
		options.TeamID = firstNonEmpty(options.TeamID, buildConfig.Sign.MacOS.TeamID)
		options.AppleID = firstNonEmpty(options.AppleID, buildConfig.Sign.MacOS.AppleID)
		options.Password = firstNonEmpty(options.Password, buildConfig.Sign.MacOS.AppPassword)
	}

	if b != nil {
		if b.explicit.arch {
			options.Arch = b.options.Arch
		}
		if b.explicit.sign {
			options.Sign = b.options.Sign
		}
		if b.explicit.notarise {
			options.Notarise = b.options.Notarise
		}
		if b.explicit.dmg {
			options.DMG = b.options.DMG
		}
		if b.explicit.testFlight {
			options.TestFlight = b.options.TestFlight
		}
		if b.explicit.appStore {
			options.AppStore = b.options.AppStore
		}
	}

	if runtime != nil {
		override := *runtime
		if override.TeamID != "" {
			options.TeamID = override.TeamID
		}
		if override.BundleID != "" {
			options.BundleID = override.BundleID
		}
		if override.Arch != "" {
			options.Arch = override.Arch
		}
		if override.CertIdentity != "" {
			options.CertIdentity = override.CertIdentity
		}
		if override.ProfilePath != "" {
			options.ProfilePath = override.ProfilePath
		}
		if override.KeychainPath != "" {
			options.KeychainPath = override.KeychainPath
		}
		if override.MetadataPath != "" {
			options.MetadataPath = override.MetadataPath
		}
		if override.APIKeyID != "" {
			options.APIKeyID = override.APIKeyID
		}
		if override.APIKeyIssuerID != "" {
			options.APIKeyIssuerID = override.APIKeyIssuerID
		}
		if override.APIKeyPath != "" {
			options.APIKeyPath = override.APIKeyPath
		}
		if override.AppleID != "" {
			options.AppleID = override.AppleID
		}
		if override.Password != "" {
			options.Password = override.Password
		}
		if override.BundleDisplayName != "" {
			options.BundleDisplayName = override.BundleDisplayName
		}
		if override.MinSystemVersion != "" {
			options.MinSystemVersion = override.MinSystemVersion
		}
		if override.Category != "" {
			options.Category = override.Category
		}
		if override.Copyright != "" {
			options.Copyright = override.Copyright
		}
		if override.PrivacyPolicyURL != "" {
			options.PrivacyPolicyURL = override.PrivacyPolicyURL
		}
		if override.DMGBackground != "" {
			options.DMGBackground = override.DMGBackground
		}
		if override.DMGVolumeName != "" {
			options.DMGVolumeName = override.DMGVolumeName
		}
		if override.EntitlementsPath != "" {
			options.EntitlementsPath = override.EntitlementsPath
		}
		applyRuntimePipelineOverrides(&options, override)
	}

	return options
}

func applyRuntimePipelineOverrides(options *AppleOptions, override AppleOptions) {
	if options == nil {
		return
	}

	// Partial runtime overrides often only provide identity/metadata fields.
	// Treat all-zero booleans in that case as "unspecified" so the builder keeps
	// config/default pipeline behavior instead of disabling sign/notarise by
	// accident. Bool-only runtime structs still override everything explicitly.
	hasNonBooleanOverrides := hasNonBooleanRuntimeOverrides(override)

	if override.Sign || !hasNonBooleanOverrides {
		options.Sign = override.Sign
	}
	if override.Notarise || !hasNonBooleanOverrides {
		options.Notarise = override.Notarise
	}
	if override.DMG || !hasNonBooleanOverrides {
		options.DMG = override.DMG
	}
	if override.TestFlight || !hasNonBooleanOverrides {
		options.TestFlight = override.TestFlight
	}
	if override.AppStore || !hasNonBooleanOverrides {
		options.AppStore = override.AppStore
	}
}

func hasNonBooleanRuntimeOverrides(options AppleOptions) bool {
	for _, value := range []string{
		options.TeamID,
		options.BundleID,
		options.Arch,
		options.CertIdentity,
		options.ProfilePath,
		options.KeychainPath,
		options.MetadataPath,
		options.APIKeyID,
		options.APIKeyIssuerID,
		options.APIKeyPath,
		options.AppleID,
		options.Password,
		options.BundleDisplayName,
		options.MinSystemVersion,
		options.Category,
		options.Copyright,
		options.PrivacyPolicyURL,
		options.DMGBackground,
		options.DMGVolumeName,
		options.EntitlementsPath,
	} {
		if core.Trim(value) != "" {
			return true
		}
	}

	return false
}

func resolveBundleName(cfg *build.BuildConfig, projectDir string) string {
	if cfg != nil {
		if cfg.Project.Binary != "" {
			return cfg.Project.Binary
		}
		if cfg.Project.Name != "" {
			return cfg.Project.Name
		}
	}
	return ax.Base(projectDir)
}

func runtimeConfig(filesystem coreio.Medium, projectDir, outputDir, name string, buildConfig *build.BuildConfig, version string) *build.Config {
	return build.RuntimeConfigFromBuildConfig(filesystem, projectDir, outputDir, name, buildConfig, false, "", version)
}

var buildNumberPattern = regexp.MustCompile(`^[0-9]+$`)

func resolveBuildNumber(ctx context.Context, projectDir string) core.Result {
	if value := core.Trim(core.Env("GITHUB_RUN_NUMBER")); value != "" {
		if validated := validateBuildNumber(value); validated.OK {
			return core.Ok(value)
		}
	}

	outputResult := runDirFn(ctx, projectDir, "git", "rev-list", "--count", "HEAD")
	if !outputResult.OK {
		return core.Ok("1")
	}

	value := core.Trim(outputResult.Value.(string))
	if value == "" {
		return core.Ok("1")
	}
	validated := validateBuildNumber(value)
	if !validated.OK {
		return validated
	}
	return core.Ok(value)
}

func validateBuildNumber(value string) core.Result {
	if !buildNumberPattern.MatchString(value) {
		return core.Fail(coreerr.E("apple.validateBuildNumber", "build number must be a positive integer", nil))
	}
	return core.Ok(nil)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if core.Trim(value) != "" {
			return value
		}
	}
	return ""
}
