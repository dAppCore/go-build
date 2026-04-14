package apple

import (
	"context"
	"regexp"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	build "dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/build/pkg/release"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// AppleOptions aliases the core Apple pipeline options.
type AppleOptions = build.AppleOptions

// WailsBuildConfig aliases the Wails build configuration used by the Apple pipeline.
type WailsBuildConfig = build.WailsBuildConfig

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

// Option configures an AppleBuilder.
type Option func(*AppleBuilder)

var (
	loadConfigFn      = build.LoadConfig
	buildAppleFn      = build.BuildApple
	determineVersion  = release.DetermineVersionWithContext
	getwdFn           = ax.Getwd
	runDirFn          = ax.RunDir
	buildWailsAppFn   = build.BuildWailsApp
	createUniversalFn = build.CreateUniversal
	signFn            = build.Sign
	notariseFn        = build.Notarise
	createDMGFn       = build.CreateDMG
	uploadTFn         = build.UploadTestFlight
	submitASFn        = build.SubmitAppStore
)

// Register wires AppleBuilder into the Core service registry.
func Register(c *core.Core) core.Result {
	if c == nil {
		return core.Result{Value: coreerr.E("apple.Register", "core is nil", nil), OK: false}
	}

	builder := New()
	builder.ServiceRuntime = core.NewServiceRuntime[AppleOptions](c, builder.options)
	if r := c.RegisterService("apple", builder); !r.OK {
		return r
	}

	return core.Result{Value: builder, OK: true}
}

// New constructs an AppleBuilder with functional options.
func New(opts ...Option) *AppleBuilder {
	builder := &AppleBuilder{
		options: build.DefaultAppleOptions(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(builder)
		}
	}
	builder.ServiceRuntime = core.NewServiceRuntime[AppleOptions](nil, builder.options)
	return builder
}

// WithArch sets the target architecture.
func WithArch(arch string) Option {
	return func(builder *AppleBuilder) {
		if builder == nil {
			return
		}
		builder.options.Arch = arch
		builder.explicit.arch = true
	}
}

// WithSign enables or disables code signing.
func WithSign(sign bool) Option {
	return func(builder *AppleBuilder) {
		if builder == nil {
			return
		}
		builder.options.Sign = sign
		builder.explicit.sign = true
	}
}

// WithNotarise enables or disables notarisation.
func WithNotarise(notarise bool) Option {
	return func(builder *AppleBuilder) {
		if builder == nil {
			return
		}
		builder.options.Notarise = notarise
		builder.explicit.notarise = true
	}
}

// WithDMG enables or disables DMG creation.
func WithDMG(dmg bool) Option {
	return func(builder *AppleBuilder) {
		if builder == nil {
			return
		}
		builder.options.DMG = dmg
		builder.explicit.dmg = true
	}
}

// WithTestFlight enables or disables TestFlight upload.
func WithTestFlight(tf bool) Option {
	return func(builder *AppleBuilder) {
		if builder == nil {
			return
		}
		builder.options.TestFlight = tf
		builder.explicit.testFlight = true
	}
}

// WithAppStore enables or disables App Store submission.
func WithAppStore(appStore bool) Option {
	return func(builder *AppleBuilder) {
		if builder == nil {
			return
		}
		builder.options.AppStore = appStore
		builder.explicit.appStore = true
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
	return core.Result{Value: build.IsWailsProject(fs, dir), OK: true}
}

// Build runs the Apple pipeline for the current working directory and returns the .app bundle path.
func (b *AppleBuilder) Build(ctx context.Context, cfg *AppleOptions) core.Result {
	if ctx == nil {
		ctx = context.Background()
	}

	projectDir, err := getwdFn()
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	buildConfig, err := loadConfigFn(coreio.Local, projectDir)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	if err := build.SetupBuildCache(coreio.Local, projectDir, buildConfig); err != nil {
		return core.Result{Value: err, OK: false}
	}

	version, err := determineVersion(ctx, projectDir)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	buildNumber, err := resolveBuildNumber(ctx, projectDir)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	options := b.resolveOptions(buildConfig, cfg)
	name := resolveBundleName(buildConfig, projectDir)
	outputDir := ax.Join(projectDir, "dist", "apple")
	runtimeCfg := runtimeConfig(coreio.Local, projectDir, outputDir, name, buildConfig, version)

	result, err := buildAppleFn(ctx, runtimeCfg, options, buildNumber)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	return core.Result{Value: result.BundlePath, OK: true}
}

// BuildWailsApp compiles the Wails application for a single Apple architecture.
func BuildWailsApp(ctx context.Context, cfg WailsBuildConfig) core.Result {
	return core.Result{}.New(buildWailsAppFn(ctx, cfg))
}

// CreateUniversal merges arm64 and amd64 bundles into a universal bundle.
func CreateUniversal(arm64Path, amd64Path, outputPath string) core.Result {
	return core.Result{}.New(outputPath, createUniversalFn(arm64Path, amd64Path, outputPath))
}

// Sign code-signs the given Apple artefact.
func Sign(ctx context.Context, cfg SignConfig) core.Result {
	return core.Result{}.New(cfg.AppPath, signFn(ctx, cfg))
}

// Notarise submits the artefact for Apple notarisation.
func Notarise(ctx context.Context, cfg NotariseConfig) core.Result {
	return core.Result{}.New(cfg.AppPath, notariseFn(ctx, cfg))
}

// CreateDMG packages the app bundle into a DMG and returns the DMG path.
func CreateDMG(ctx context.Context, cfg DMGConfig) core.Result {
	return core.Result{}.New(cfg.OutputPath, createDMGFn(ctx, cfg))
}

// UploadTestFlight uploads the packaged build to TestFlight.
func UploadTestFlight(ctx context.Context, cfg TestFlightConfig) core.Result {
	return core.Result{}.New(cfg.AppPath, uploadTFn(ctx, cfg))
}

// SubmitAppStore uploads the packaged build to App Store Connect.
func SubmitAppStore(ctx context.Context, cfg AppStoreConfig) core.Result {
	return core.Result{}.New(cfg.AppPath, submitASFn(ctx, cfg))
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

		options.Sign = override.Sign
		options.Notarise = override.Notarise
		options.DMG = override.DMG
		options.TestFlight = override.TestFlight
		options.AppStore = override.AppStore
	}

	return options
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

func resolveBuildNumber(ctx context.Context, projectDir string) (string, error) {
	if value := core.Trim(core.Env("GITHUB_RUN_NUMBER")); value != "" {
		if err := validateBuildNumber(value); err == nil {
			return value, nil
		}
	}

	output, err := runDirFn(ctx, projectDir, "git", "rev-list", "--count", "HEAD")
	if err != nil {
		return "1", nil
	}

	value := core.Trim(output)
	if value == "" {
		return "1", nil
	}
	if err := validateBuildNumber(value); err != nil {
		return "", err
	}
	return value, nil
}

func validateBuildNumber(value string) error {
	if !buildNumberPattern.MatchString(value) {
		return coreerr.E("apple.validateBuildNumber", "build number must be a positive integer", nil)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if core.Trim(value) != "" {
			return value
		}
	}
	return ""
}
