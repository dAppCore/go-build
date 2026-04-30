// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	stdio "io"
	"runtime"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	coreio "dappco.re/go/build/pkg/storage"
)

const (
	defaultAppleBuilderArch             = "universal"
	defaultAppleBuilderMinSystemVersion = "13.0"
	defaultAppleBuilderCategory         = "public.app-category.developer-tools"
)

// Builder aliases the shared build.Builder interface for callers in this package.
type Builder = build.Builder

// AppleOptions holds the Apple build pipeline settings used by AppleBuilder.
type AppleOptions struct {
	SigningIdentity  string
	CertIdentity     string
	BundleID         string
	EntitlementsPath string

	Arch              string
	BundleDisplayName string
	MinSystemVersion  string
	Category          string
	Copyright         string
	BuildNumber       string

	Sign       bool
	Notarise   bool
	Notarize   bool
	TestFlight bool
	AppStore   bool

	TeamID      string
	AppleID     string
	AppPassword string
	Password    string

	APIKeyID       string
	APIKeyIssuerID string
	APIKeyPath     string

	NotarisationProfile string
	NotarizationProfile string
	NotaryProfile       string

	TestFlightKeyID      string
	TestFlightIssuerID   string
	TestFlightKeyPath    string
	TestFlightPrivateKey string
	XcodeCloud           bool
	DMG                  AppleDMGConfig
}

// AppleDMGConfig holds DMG packaging settings for the Apple pipeline.
type AppleDMGConfig struct {
	Enabled        bool
	OutputPath     string
	VolumeName     string
	BackgroundPath string
	IconSize       int
	WindowSize     [2]int
}

// DMGConfig aliases the Apple DMG config for callers that use the shorter name.
type DMGConfig = AppleDMGConfig

// AppleCommandRunner records or executes an external command invocation.
type AppleCommandRunner interface {
	Run(ctx context.Context, opts RunOptions) core.Result
}

// AppleCommandRunnerFunc adapts a function to AppleCommandRunner.
type AppleCommandRunnerFunc func(ctx context.Context, opts RunOptions) core.Result

// Run implements AppleCommandRunner.
func (fn AppleCommandRunnerFunc) Run(ctx context.Context, opts RunOptions) core.Result {
	return fn(ctx, opts)
}

// GoProcessAppleRunner executes commands through Core's process primitive.
// It is intentionally opt-in because the skeleton defaults to non-executing
// stubs for sandbox-safe tests.
type GoProcessAppleRunner struct{}

// Run executes opts through Core's process primitive.
func (GoProcessAppleRunner) Run(ctx context.Context, opts RunOptions) core.Result {
	return runWithOptions(ctx, opts)
}

// AppleBuilder implements build.Builder for the Apple build pipeline skeleton.
type AppleBuilder struct {
	Options AppleOptions

	runner     AppleCommandRunner
	hostOS     string
	todoWriter stdio.Writer
}

// AppleBuilderOption configures an AppleBuilder.
type AppleBuilderOption func(*AppleBuilder)

// NewAppleBuilder creates an Apple build pipeline skeleton.
func NewAppleBuilder(options ...AppleBuilderOption) *AppleBuilder {
	builder := &AppleBuilder{
		Options:    DefaultAppleBuilderOptions(),
		hostOS:     runtime.GOOS,
		todoWriter: core.Stdout(),
	}
	for _, option := range options {
		if option != nil {
			option(builder)
		}
	}
	return builder
}

// WithAppleOptions replaces the default Apple options.
func WithAppleOptions(options AppleOptions) AppleBuilderOption {
	return func(builder *AppleBuilder) {
		builder.Options = options.withDefaults()
	}
}

// WithAppleCommandRunner configures the command runner used by external stubs.
func WithAppleCommandRunner(runner AppleCommandRunner) AppleBuilderOption {
	return func(builder *AppleBuilder) {
		builder.runner = runner
	}
}

// WithAppleHostOS overrides host OS detection, mainly for tests.
func WithAppleHostOS(hostOS string) AppleBuilderOption {
	return func(builder *AppleBuilder) {
		builder.hostOS = hostOS
	}
}

// WithAppleTODOWriter configures where structured TODO messages are printed.
func WithAppleTODOWriter(writer stdio.Writer) AppleBuilderOption {
	return func(builder *AppleBuilder) {
		builder.todoWriter = writer
	}
}

// DefaultAppleBuilderOptions returns sandbox-safe Apple pipeline defaults.
func DefaultAppleBuilderOptions() AppleOptions {
	return AppleOptions{
		Arch:             defaultAppleBuilderArch,
		MinSystemVersion: defaultAppleBuilderMinSystemVersion,
		Category:         defaultAppleBuilderCategory,
		DMG: AppleDMGConfig{
			IconSize:   128,
			WindowSize: [2]int{640, 480},
		},
	}
}

// Name returns the builder identifier.
func (b *AppleBuilder) Name() string {
	return "apple"
}

// Detect checks whether dir looks like a Wails macOS app project.
func (b *AppleBuilder) Detect(fs coreio.Medium, dir string) core.Result {
	if fs == nil {
		fs = coreio.Local
	}
	return core.Ok(build.IsWailsProject(fs, dir))
}

// Build runs the Apple build pipeline skeleton.
func (b *AppleBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) core.Result {
	if cfg == nil {
		return core.Fail(core.E("AppleBuilder.Build", "config is nil", nil))
	}
	if ctx == nil {
		ctx = context.Background()
	}

	filesystem := ensureBuildFilesystem(cfg)
	artifactFilesystem := build.ResolveOutputMedium(cfg)
	options := b.options()
	valid := ValidateAppleOptions(options)
	if !valid.OK {
		return valid
	}

	outputDir := resolveAppleBuilderOutputDir(cfg, artifactFilesystem)
	created := artifactFilesystem.EnsureDir(outputDir)
	if !created.OK {
		return core.Fail(core.E("AppleBuilder.Build", "failed to create Apple output directory", core.NewError(created.Error())))
	}

	name := resolveAppleBuilderName(cfg)
	buildNumber := firstNonEmptyApple(options.BuildNumber, "1")
	if options.XcodeCloud {
		written := b.WriteXcodeCloudConfig(artifactFilesystem, cfg.ProjectDir, cfg, options)
		if !written.OK {
			return written
		}
	}

	targetArch := resolveAppleBuilderArch(options, targets)
	bundleResult := b.buildBundle(ctx, filesystem, artifactFilesystem, cfg, outputDir, name, targetArch)
	if !bundleResult.OK {
		return bundleResult
	}
	bundlePath := bundleResult.Value.(string)

	plist := WriteAppleInfoPlist(artifactFilesystem, bundlePath, cfg, options, buildNumber)
	if !plist.OK {
		return plist
	}

	entitlementsPath := resolveAppleEntitlementsPath(cfg, outputDir, name, options)
	entitlements := WriteAppleEntitlements(artifactFilesystem, entitlementsPath, DefaultAppleEntitlements())
	if !entitlements.OK {
		return entitlements
	}

	if options.Sign {
		signed := b.signAppleArtifact(ctx, cfg, bundlePath, entitlementsPath, options)
		if !signed.OK {
			return signed
		}
	}

	distributionPath := bundlePath
	if options.DMG.Enabled {
		dmgPath := options.DMG.OutputPath
		if dmgPath == "" {
			dmgPath = ax.Join(outputDir, name+".dmg")
		}
		dmgConfig := options.DMG
		dmgConfig.OutputPath = dmgPath
		if dmgConfig.VolumeName == "" {
			dmgConfig.VolumeName = name
		}
		createdDMG := b.CreateDMG(ctx, artifactFilesystem, bundlePath, dmgConfig)
		if !createdDMG.OK {
			return createdDMG
		}
		distributionPath = dmgPath
	}

	if options.notariseEnabled() {
		notarised := b.Notarise(ctx, distributionPath, options)
		if !notarised.OK {
			return notarised
		}
	}

	if options.TestFlight {
		uploaded := b.uploadTestFlight(ctx, cfg, bundlePath, options)
		if !uploaded.OK {
			return uploaded
		}
	}

	return core.Ok([]build.Artifact{{
		Path: distributionPath,
		OS:   "darwin",
		Arch: targetArch,
	}})
}

func (b *AppleBuilder) buildBundle(ctx context.Context, sourceFS, artifactFS coreio.Medium, cfg *build.Config, outputDir, name, arch string) core.Result {
	switch arch {
	case "universal":
		arm64 := b.BuildWailsMacOS(ctx, artifactFS, cfg, ax.Join(outputDir, "arm64"), name, "arm64")
		if !arm64.OK {
			return arm64
		}
		arm64Path := arm64.Value.(string)
		amd64 := b.BuildWailsMacOS(ctx, artifactFS, cfg, ax.Join(outputDir, "amd64"), name, "amd64")
		if !amd64.OK {
			return amd64
		}
		amd64Path := amd64.Value.(string)
		outputPath := ax.Join(outputDir, name+".app")
		universal := b.CreateUniversal(ctx, sourceFS, artifactFS, arm64Path, amd64Path, outputPath, name)
		if !universal.OK {
			return universal
		}
		return core.Ok(outputPath)
	case "arm64", "amd64":
		return b.BuildWailsMacOS(ctx, artifactFS, cfg, outputDir, name, arch)
	default:
		return core.Fail(core.E("AppleBuilder.Build", "unsupported Apple arch: "+arch, nil))
	}
}

// BuildWailsMacOS records the Wails macOS build invocation and creates a placeholder .app bundle.
func (b *AppleBuilder) BuildWailsMacOS(ctx context.Context, filesystem coreio.Medium, cfg *build.Config, outputDir, name, arch string) core.Result {
	if filesystem == nil {
		filesystem = coreio.Local
	}
	created := filesystem.EnsureDir(outputDir)
	if !created.OK {
		return core.Fail(core.E("AppleBuilder.BuildWailsMacOS", "failed to create Wails output directory", core.NewError(created.Error())))
	}

	args := []string{"build", "-platform", "darwin/" + arch}
	if len(cfg.BuildTags) > 0 {
		args = append(args, "-tags", core.Join(",", cfg.BuildTags...))
	}
	if len(cfg.LDFlags) > 0 {
		args = append(args, "-ldflags", core.Join(" ", cfg.LDFlags...))
	}

	// TODO(#484): this requires macOS with Wails and Xcode tooling. The skeleton
	// records the command invocation instead of executing it in sandbox.
	ran := b.runExternal(ctx, "wails-build", RunOptions{
		Command: "wails3",
		Args:    args,
		Dir:     cfg.ProjectDir,
		Env:     build.BuildEnvironment(cfg, "GOOS=darwin", "GOARCH="+arch, "CGO_ENABLED=1"),
	})
	if !ran.OK {
		return ran
	}

	bundlePath := ax.Join(outputDir, name+".app")
	createdBundle := createAppleBundleSkeleton(filesystem, bundlePath, name, arch)
	if !createdBundle.OK {
		return createdBundle
	}
	return core.Ok(bundlePath)
}

// CreateUniversal records the lipo invocation and creates a placeholder universal .app bundle.
func (b *AppleBuilder) CreateUniversal(ctx context.Context, _ coreio.Medium, artifactFS coreio.Medium, arm64Path, amd64Path, outputPath, name string) core.Result {
	if artifactFS == nil {
		artifactFS = coreio.Local
	}
	if artifactFS.Exists(outputPath) {
		deleted := artifactFS.DeleteAll(outputPath)
		if !deleted.OK {
			return core.Fail(core.E("AppleBuilder.CreateUniversal", "failed to replace universal app bundle", core.NewError(deleted.Error())))
		}
	}
	copied := build.CopyMediumPath(artifactFS, arm64Path, artifactFS, outputPath)
	if !copied.OK {
		return core.Fail(core.E("AppleBuilder.CreateUniversal", "failed to copy arm64 app bundle", core.NewError(copied.Error())))
	}

	armBinary := ax.Join(arm64Path, "Contents", "MacOS", name)
	amdBinary := ax.Join(amd64Path, "Contents", "MacOS", name)
	outBinary := ax.Join(outputPath, "Contents", "MacOS", name)

	// TODO(#484): this requires macOS lipo. The skeleton records the command
	// invocation so operators can wire execution on a real macOS runner.
	return b.runExternal(ctx, "lipo-universal", RunOptions{
		Command: "lipo",
		Args:    []string{"-create", "-output", outBinary, armBinary, amdBinary},
	})
}

func (b *AppleBuilder) signAppleArtifact(ctx context.Context, cfg *build.Config, appPath, entitlementsPath string, options AppleOptions) core.Result {
	args := []string{
		"--sign", options.signingIdentity(),
		"--timestamp",
		"--force",
		"--options", "runtime",
		"--entitlements", entitlementsPath,
		appPath,
	}

	// TODO(#484): this requires macOS codesign identities and keychain access.
	return b.runExternal(ctx, "codesign", RunOptions{
		Command: "codesign",
		Args:    args,
		Dir:     cfg.ProjectDir,
	})
}

func (b *AppleBuilder) uploadTestFlight(ctx context.Context, cfg *build.Config, appPath string, options AppleOptions) core.Result {
	keyID := firstNonEmptyApple(options.TestFlightKeyID, options.APIKeyID)
	issuerID := firstNonEmptyApple(options.TestFlightIssuerID, options.APIKeyIssuerID)
	keyPath := firstNonEmptyApple(options.TestFlightKeyPath, options.APIKeyPath, options.TestFlightPrivateKey)

	// TODO(#484): this requires Apple Developer App Store Connect API credentials.
	return b.runExternal(ctx, "testflight-upload", RunOptions{
		Command: "xcrun",
		Args: []string{
			"altool", "--upload-app",
			"--type", "macos",
			"--file", appPath,
			"--apiKey", keyID,
			"--apiIssuer", issuerID,
			"--private-key", keyPath,
		},
		Dir: cfg.ProjectDir,
	})
}

func (b *AppleBuilder) runExternal(ctx context.Context, step string, opts RunOptions) core.Result {
	b.printTODO(step, opts)
	if firstNonEmptyApple(b.hostOS, runtime.GOOS) != "darwin" {
		return core.Ok(nil)
	}
	if b.runner == nil {
		return core.Ok(nil)
	}
	ran := b.runner.Run(ctx, opts)
	if !ran.OK {
		return core.Fail(core.E("AppleBuilder.runExternal", "stubbed "+step+" invocation failed", core.NewError(ran.Error())))
	}
	return core.Ok(nil)
}

func (b *AppleBuilder) printTODO(step string, opts RunOptions) {
	writer := b.todoWriter
	if writer == nil {
		return
	}

	message := appleTODOMessage{
		Level:       "todo",
		Component:   "apple-build",
		Step:        step,
		Command:     opts.Command,
		Args:        append([]string{}, opts.Args...),
		Dir:         opts.Dir,
		HostOS:      firstNonEmptyApple(b.hostOS, runtime.GOOS),
		Requirement: "this requires macOS with Apple Developer tooling and credentials",
	}
	if message.HostOS != "darwin" {
		message.Requirement = "this requires macOS; sandbox stub did not execute external CLI"
	}

	encoded := core.JSONMarshal(message)
	if !encoded.OK {
		if written := core.WriteString(writer, core.Sprintf(`{"level":"todo","component":"apple-build","step":%q}`+"\n", step)); !written.OK {
			return
		}
		return
	}
	if written := core.WriteString(writer, string(encoded.Value.([]byte))+"\n"); !written.OK {
		return
	}
}

func (b *AppleBuilder) options() AppleOptions {
	if b == nil {
		return DefaultAppleBuilderOptions()
	}
	return b.Options.withDefaults()
}

type appleTODOMessage struct {
	Level       string   `json:"level"`
	Component   string   `json:"component"`
	Step        string   `json:"step"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Dir         string   `json:"dir,omitempty"`
	HostOS      string   `json:"host_os"`
	Requirement string   `json:"requirement"`
}

func (options AppleOptions) withDefaults() AppleOptions {
	defaults := DefaultAppleBuilderOptions()
	if options.Arch == "" {
		options.Arch = defaults.Arch
	}
	if options.MinSystemVersion == "" {
		options.MinSystemVersion = defaults.MinSystemVersion
	}
	if options.Category == "" {
		options.Category = defaults.Category
	}
	if options.DMG.IconSize <= 0 {
		options.DMG.IconSize = defaults.DMG.IconSize
	}
	if options.DMG.WindowSize[0] <= 0 || options.DMG.WindowSize[1] <= 0 {
		options.DMG.WindowSize = defaults.DMG.WindowSize
	}
	return options
}

func (options AppleOptions) signingIdentity() string {
	return firstNonEmptyApple(options.SigningIdentity, options.CertIdentity)
}

func (options AppleOptions) notariseEnabled() bool {
	return options.Notarise || options.Notarize
}

func (options AppleOptions) notarisationProfile() string {
	return firstNonEmptyApple(options.NotarisationProfile, options.NotarizationProfile, options.NotaryProfile)
}

// ValidateAppleOptions checks the minimum Apple pipeline option contract.
func ValidateAppleOptions(options AppleOptions) core.Result {
	options = options.withDefaults()

	if core.Trim(options.BundleID) == "" {
		return core.Fail(core.E("AppleBuilder.ValidateOptions", "bundle ID is required", nil))
	}

	switch options.Arch {
	case "universal", "arm64", "amd64":
	default:
		return core.Fail(core.E("AppleBuilder.ValidateOptions", "arch must be universal, arm64, or amd64", nil))
	}

	if options.Sign && core.Trim(options.signingIdentity()) == "" {
		return core.Fail(core.E("AppleBuilder.ValidateOptions", "signing identity is required when signing is enabled", nil))
	}

	if options.notariseEnabled() {
		hasProfile := core.Trim(options.notarisationProfile()) != ""
		hasAPIKey := core.Trim(options.APIKeyID) != "" && core.Trim(options.APIKeyIssuerID) != "" && core.Trim(options.APIKeyPath) != ""
		hasAppleID := core.Trim(options.TeamID) != "" &&
			core.Trim(options.AppleID) != "" &&
			core.Trim(firstNonEmptyApple(options.AppPassword, options.Password)) != ""
		if !hasProfile && !hasAPIKey && !hasAppleID {
			return core.Fail(core.E("AppleBuilder.ValidateOptions", "notarisation requires a notarytool profile, API key, or Apple ID credentials", nil))
		}
	}

	if options.TestFlight {
		keyID := firstNonEmptyApple(options.TestFlightKeyID, options.APIKeyID)
		issuerID := firstNonEmptyApple(options.TestFlightIssuerID, options.APIKeyIssuerID)
		keyPath := firstNonEmptyApple(options.TestFlightKeyPath, options.APIKeyPath, options.TestFlightPrivateKey)
		if keyID == "" || issuerID == "" || keyPath == "" {
			return core.Fail(core.E("AppleBuilder.ValidateOptions", "TestFlight upload requires key id, issuer id, and key path", nil))
		}
	}

	return core.Ok(nil)
}

func resolveAppleBuilderOutputDir(cfg *build.Config, artifactFilesystem coreio.Medium) string {
	if cfg.OutputDir != "" {
		return cfg.OutputDir
	}
	if build.MediumIsLocal(artifactFilesystem) {
		return ax.Join(cfg.ProjectDir, "dist", "apple")
	}
	return "dist/apple"
}

func resolveAppleBuilderName(cfg *build.Config) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.Project.Binary != "" {
		return cfg.Project.Binary
	}
	if cfg.Project.Name != "" {
		return cfg.Project.Name
	}
	if cfg.ProjectDir != "" {
		return ax.Base(cfg.ProjectDir)
	}
	return "App"
}

func resolveAppleBuilderArch(options AppleOptions, targets []build.Target) string {
	if options.Arch != "" {
		return options.Arch
	}
	for _, target := range targets {
		if target.OS == "darwin" && target.Arch != "" {
			return target.Arch
		}
	}
	return defaultAppleBuilderArch
}

func resolveAppleEntitlementsPath(cfg *build.Config, outputDir, name string, options AppleOptions) string {
	if options.EntitlementsPath == "" {
		return ax.Join(outputDir, name+".entitlements.plist")
	}
	if ax.IsAbs(options.EntitlementsPath) || cfg == nil || cfg.ProjectDir == "" {
		return options.EntitlementsPath
	}
	return ax.Join(cfg.ProjectDir, options.EntitlementsPath)
}

func createAppleBundleSkeleton(filesystem coreio.Medium, bundlePath, name, arch string) core.Result {
	if filesystem == nil {
		filesystem = coreio.Local
	}

	macosDir := ax.Join(bundlePath, "Contents", "MacOS")
	resourcesDir := ax.Join(bundlePath, "Contents", "Resources")
	if created := filesystem.EnsureDir(macosDir); !created.OK {
		return core.Fail(core.E("AppleBuilder.createBundleSkeleton", "failed to create Contents/MacOS", core.NewError(created.Error())))
	}
	if created := filesystem.EnsureDir(resourcesDir); !created.OK {
		return core.Fail(core.E("AppleBuilder.createBundleSkeleton", "failed to create Contents/Resources", core.NewError(created.Error())))
	}

	executable := ax.Join(macosDir, name)
	content := "#!/usr/bin/env sh\n" +
		"echo \"AppleBuilder skeleton placeholder for " + name + " (" + arch + ")\"\n"
	written := filesystem.WriteMode(executable, content, 0o755)
	if !written.OK {
		return core.Fail(core.E("AppleBuilder.createBundleSkeleton", "failed to write placeholder executable", core.NewError(written.Error())))
	}
	return core.Ok(nil)
}

func firstNonEmptyApple(values ...string) string {
	for _, value := range values {
		if core.Trim(value) != "" {
			return value
		}
	}
	return ""
}

var _ build.Builder = (*AppleBuilder)(nil)
