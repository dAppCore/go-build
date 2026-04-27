// Package builders provides build implementations for different project types.
package builders

import (
	"context"
	"encoding/json"
	"fmt"
	stdio "io"
	"os"
	"runtime"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/core"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
	"dappco.re/go/process"
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

// AppleCommandRunner records or executes a go-process command invocation.
type AppleCommandRunner interface {
	Run(ctx context.Context, opts process.RunOptions) (string, error)
}

// AppleCommandRunnerFunc adapts a function to AppleCommandRunner.
type AppleCommandRunnerFunc func(ctx context.Context, opts process.RunOptions) (string, error)

// Run implements AppleCommandRunner.
func (fn AppleCommandRunnerFunc) Run(ctx context.Context, opts process.RunOptions) (string, error) {
	return fn(ctx, opts)
}

// GoProcessAppleRunner executes commands through the go-process package.
// It is intentionally opt-in because the skeleton defaults to non-executing
// stubs for sandbox-safe tests.
type GoProcessAppleRunner struct{}

// Run executes opts through go-process.
func (GoProcessAppleRunner) Run(ctx context.Context, opts process.RunOptions) (string, error) {
	return process.RunWithOptions(ctx, opts)
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
		todoWriter: os.Stdout,
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
func (b *AppleBuilder) Detect(fs coreio.Medium, dir string) (bool, error) {
	if fs == nil {
		fs = coreio.Local
	}
	return build.IsWailsProject(fs, dir), nil
}

// Build runs the Apple build pipeline skeleton.
func (b *AppleBuilder) Build(ctx context.Context, cfg *build.Config, targets []build.Target) ([]build.Artifact, error) {
	if cfg == nil {
		return nil, coreerr.E("AppleBuilder.Build", "config is nil", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	filesystem := ensureBuildFilesystem(cfg)
	artifactFilesystem := build.ResolveOutputMedium(cfg)
	options := b.options()
	if err := ValidateAppleOptions(options); err != nil {
		return nil, err
	}

	outputDir := resolveAppleBuilderOutputDir(cfg, artifactFilesystem)
	if err := artifactFilesystem.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("AppleBuilder.Build", "failed to create Apple output directory", err)
	}

	name := resolveAppleBuilderName(cfg)
	buildNumber := firstNonEmptyApple(options.BuildNumber, "1")
	if options.XcodeCloud {
		if _, err := b.WriteXcodeCloudConfig(artifactFilesystem, cfg.ProjectDir, cfg, options); err != nil {
			return nil, err
		}
	}

	targetArch := resolveAppleBuilderArch(options, targets)
	bundlePath, err := b.buildBundle(ctx, filesystem, artifactFilesystem, cfg, outputDir, name, targetArch)
	if err != nil {
		return nil, err
	}

	if _, err := WriteAppleInfoPlist(artifactFilesystem, bundlePath, cfg, options, buildNumber); err != nil {
		return nil, err
	}

	entitlementsPath := resolveAppleEntitlementsPath(cfg, outputDir, name, options)
	if err := WriteAppleEntitlements(artifactFilesystem, entitlementsPath, DefaultAppleEntitlements()); err != nil {
		return nil, err
	}

	if options.Sign {
		if err := b.signAppleArtifact(ctx, cfg, bundlePath, entitlementsPath, options); err != nil {
			return nil, err
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
		if err := b.CreateDMG(ctx, artifactFilesystem, bundlePath, dmgConfig); err != nil {
			return nil, err
		}
		distributionPath = dmgPath
	}

	if options.notariseEnabled() {
		if err := b.Notarise(ctx, distributionPath, options); err != nil {
			return nil, err
		}
	}

	if options.TestFlight {
		if err := b.uploadTestFlight(ctx, cfg, bundlePath, options); err != nil {
			return nil, err
		}
	}

	return []build.Artifact{{
		Path: distributionPath,
		OS:   "darwin",
		Arch: targetArch,
	}}, nil
}

func (b *AppleBuilder) buildBundle(ctx context.Context, sourceFS, artifactFS coreio.Medium, cfg *build.Config, outputDir, name, arch string) (string, error) {
	switch arch {
	case "universal":
		arm64Path, err := b.BuildWailsMacOS(ctx, artifactFS, cfg, ax.Join(outputDir, "arm64"), name, "arm64")
		if err != nil {
			return "", err
		}
		amd64Path, err := b.BuildWailsMacOS(ctx, artifactFS, cfg, ax.Join(outputDir, "amd64"), name, "amd64")
		if err != nil {
			return "", err
		}
		outputPath := ax.Join(outputDir, name+".app")
		if err := b.CreateUniversal(ctx, sourceFS, artifactFS, arm64Path, amd64Path, outputPath, name); err != nil {
			return "", err
		}
		return outputPath, nil
	case "arm64", "amd64":
		return b.BuildWailsMacOS(ctx, artifactFS, cfg, outputDir, name, arch)
	default:
		return "", coreerr.E("AppleBuilder.Build", "unsupported Apple arch: "+arch, nil)
	}
}

// BuildWailsMacOS records the Wails macOS build invocation and creates a placeholder .app bundle.
func (b *AppleBuilder) BuildWailsMacOS(ctx context.Context, filesystem coreio.Medium, cfg *build.Config, outputDir, name, arch string) (string, error) {
	if filesystem == nil {
		filesystem = coreio.Local
	}
	if err := filesystem.EnsureDir(outputDir); err != nil {
		return "", coreerr.E("AppleBuilder.BuildWailsMacOS", "failed to create Wails output directory", err)
	}

	args := []string{"build", "-platform", "darwin/" + arch}
	if len(cfg.BuildTags) > 0 {
		args = append(args, "-tags", core.Join(",", cfg.BuildTags...))
	}
	if len(cfg.LDFlags) > 0 {
		args = append(args, "-ldflags", core.Join(" ", cfg.LDFlags...))
	}

	// TODO(#484): this requires macOS with Wails and Xcode tooling. The skeleton
	// records the go-process invocation instead of executing it in sandbox.
	if err := b.runExternal(ctx, "wails-build", process.RunOptions{
		Command: "wails3",
		Args:    args,
		Dir:     cfg.ProjectDir,
		Env:     build.BuildEnvironment(cfg, "GOOS=darwin", "GOARCH="+arch, "CGO_ENABLED=1"),
	}); err != nil {
		return "", err
	}

	bundlePath := ax.Join(outputDir, name+".app")
	if err := createAppleBundleSkeleton(filesystem, bundlePath, name, arch); err != nil {
		return "", err
	}
	return bundlePath, nil
}

// CreateUniversal records the lipo invocation and creates a placeholder universal .app bundle.
func (b *AppleBuilder) CreateUniversal(ctx context.Context, _ coreio.Medium, artifactFS coreio.Medium, arm64Path, amd64Path, outputPath, name string) error {
	if artifactFS == nil {
		artifactFS = coreio.Local
	}
	if artifactFS.Exists(outputPath) {
		if err := artifactFS.DeleteAll(outputPath); err != nil {
			return coreerr.E("AppleBuilder.CreateUniversal", "failed to replace universal app bundle", err)
		}
	}
	if err := build.CopyMediumPath(artifactFS, arm64Path, artifactFS, outputPath); err != nil {
		return coreerr.E("AppleBuilder.CreateUniversal", "failed to copy arm64 app bundle", err)
	}

	armBinary := ax.Join(arm64Path, "Contents", "MacOS", name)
	amdBinary := ax.Join(amd64Path, "Contents", "MacOS", name)
	outBinary := ax.Join(outputPath, "Contents", "MacOS", name)

	// TODO(#484): this requires macOS lipo. The skeleton records the go-process
	// invocation so operators can wire execution on a real macOS runner.
	return b.runExternal(ctx, "lipo-universal", process.RunOptions{
		Command: "lipo",
		Args:    []string{"-create", "-output", outBinary, armBinary, amdBinary},
	})
}

func (b *AppleBuilder) signAppleArtifact(ctx context.Context, cfg *build.Config, appPath, entitlementsPath string, options AppleOptions) error {
	args := []string{
		"--sign", options.signingIdentity(),
		"--timestamp",
		"--force",
		"--options", "runtime",
		"--entitlements", entitlementsPath,
		appPath,
	}

	// TODO(#484): this requires macOS codesign identities and keychain access.
	return b.runExternal(ctx, "codesign", process.RunOptions{
		Command: "codesign",
		Args:    args,
		Dir:     cfg.ProjectDir,
	})
}

func (b *AppleBuilder) uploadTestFlight(ctx context.Context, cfg *build.Config, appPath string, options AppleOptions) error {
	keyID := firstNonEmptyApple(options.TestFlightKeyID, options.APIKeyID)
	issuerID := firstNonEmptyApple(options.TestFlightIssuerID, options.APIKeyIssuerID)
	keyPath := firstNonEmptyApple(options.TestFlightKeyPath, options.APIKeyPath, options.TestFlightPrivateKey)

	// TODO(#484): this requires Apple Developer App Store Connect API credentials.
	return b.runExternal(ctx, "testflight-upload", process.RunOptions{
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

func (b *AppleBuilder) runExternal(ctx context.Context, step string, opts process.RunOptions) error {
	b.printTODO(step, opts)
	if firstNonEmptyApple(b.hostOS, runtime.GOOS) != "darwin" {
		return nil
	}
	if b.runner == nil {
		return nil
	}
	_, err := b.runner.Run(ctx, opts)
	if err != nil {
		return coreerr.E("AppleBuilder.runExternal", "stubbed "+step+" invocation failed", err)
	}
	return nil
}

func (b *AppleBuilder) printTODO(step string, opts process.RunOptions) {
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

	encoded, err := json.Marshal(message)
	if err != nil {
		fmt.Fprintf(writer, `{"level":"todo","component":"apple-build","step":%q}`+"\n", step)
		return
	}
	fmt.Fprintln(writer, string(encoded))
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
func ValidateAppleOptions(options AppleOptions) error {
	options = options.withDefaults()

	if core.Trim(options.BundleID) == "" {
		return coreerr.E("AppleBuilder.ValidateOptions", "bundle ID is required", nil)
	}

	switch options.Arch {
	case "universal", "arm64", "amd64":
	default:
		return coreerr.E("AppleBuilder.ValidateOptions", "arch must be universal, arm64, or amd64", nil)
	}

	if options.Sign && core.Trim(options.signingIdentity()) == "" {
		return coreerr.E("AppleBuilder.ValidateOptions", "signing identity is required when signing is enabled", nil)
	}

	if options.notariseEnabled() {
		hasProfile := core.Trim(options.notarisationProfile()) != ""
		hasAPIKey := core.Trim(options.APIKeyID) != "" && core.Trim(options.APIKeyIssuerID) != "" && core.Trim(options.APIKeyPath) != ""
		hasAppleID := core.Trim(options.TeamID) != "" &&
			core.Trim(options.AppleID) != "" &&
			core.Trim(firstNonEmptyApple(options.AppPassword, options.Password)) != ""
		if !hasProfile && !hasAPIKey && !hasAppleID {
			return coreerr.E("AppleBuilder.ValidateOptions", "notarisation requires a notarytool profile, API key, or Apple ID credentials", nil)
		}
	}

	if options.TestFlight {
		keyID := firstNonEmptyApple(options.TestFlightKeyID, options.APIKeyID)
		issuerID := firstNonEmptyApple(options.TestFlightIssuerID, options.APIKeyIssuerID)
		keyPath := firstNonEmptyApple(options.TestFlightKeyPath, options.APIKeyPath, options.TestFlightPrivateKey)
		if keyID == "" || issuerID == "" || keyPath == "" {
			return coreerr.E("AppleBuilder.ValidateOptions", "TestFlight upload requires key id, issuer id, and key path", nil)
		}
	}

	return nil
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

func createAppleBundleSkeleton(filesystem coreio.Medium, bundlePath, name, arch string) error {
	if filesystem == nil {
		filesystem = coreio.Local
	}

	macosDir := ax.Join(bundlePath, "Contents", "MacOS")
	resourcesDir := ax.Join(bundlePath, "Contents", "Resources")
	if err := filesystem.EnsureDir(macosDir); err != nil {
		return coreerr.E("AppleBuilder.createBundleSkeleton", "failed to create Contents/MacOS", err)
	}
	if err := filesystem.EnsureDir(resourcesDir); err != nil {
		return coreerr.E("AppleBuilder.createBundleSkeleton", "failed to create Contents/Resources", err)
	}

	executable := ax.Join(macosDir, name)
	content := "#!/usr/bin/env sh\n" +
		"echo \"AppleBuilder skeleton placeholder for " + name + " (" + arch + ")\"\n"
	if err := filesystem.WriteMode(executable, content, 0o755); err != nil {
		return coreerr.E("AppleBuilder.createBundleSkeleton", "failed to write placeholder executable", err)
	}
	return nil
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
