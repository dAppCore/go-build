package build

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

const (
	defaultAppleArch             = "universal"
	defaultAppleMinSystemVersion = "13.0"
	defaultAppleCategory         = "public.app-category.developer-tools"
	defaultDMGIconSize           = 128
	defaultDMGWindowWidth        = 640
	defaultDMGWindowHeight       = 480
)

// AppleOptions holds the resolved runtime settings for the macOS Apple pipeline.
type AppleOptions struct {
	TeamID       string `json:"team_id" yaml:"team_id"`
	BundleID     string `json:"bundle_id" yaml:"bundle_id"`
	Arch         string `json:"arch" yaml:"arch"`
	CertIdentity string `json:"cert_identity" yaml:"cert_identity"`
	ProfilePath  string `json:"profile_path" yaml:"profile_path"`
	KeychainPath string `json:"keychain_path" yaml:"keychain_path"`
	MetadataPath string `json:"metadata_path" yaml:"metadata_path"`

	Sign       bool `json:"sign" yaml:"sign"`
	Notarise   bool `json:"notarise" yaml:"notarise"`
	DMG        bool `json:"dmg" yaml:"dmg"`
	TestFlight bool `json:"testflight" yaml:"testflight"`
	AppStore   bool `json:"appstore" yaml:"appstore"`

	APIKeyID       string `json:"api_key_id" yaml:"api_key_id"`
	APIKeyIssuerID string `json:"api_key_issuer_id" yaml:"api_key_issuer_id"`
	APIKeyPath     string `json:"api_key_path" yaml:"api_key_path"`
	AppleID        string `json:"apple_id" yaml:"apple_id"`
	Password       string `json:"password" yaml:"password"`

	BundleDisplayName string `json:"bundle_display_name" yaml:"bundle_display_name"`
	MinSystemVersion  string `json:"min_system_version" yaml:"min_system_version"`
	Category          string `json:"category" yaml:"category"`
	Copyright         string `json:"copyright" yaml:"copyright"`
	PrivacyPolicyURL  string `json:"privacy_policy_url" yaml:"privacy_policy_url"`
	DMGBackground     string `json:"dmg_background" yaml:"dmg_background"`
	DMGVolumeName     string `json:"dmg_volume_name" yaml:"dmg_volume_name"`
	EntitlementsPath  string `json:"entitlements_path" yaml:"entitlements_path"`
}

// AppleBuildResult captures the primary outputs of the Apple pipeline.
type AppleBuildResult struct {
	BundlePath       string
	DMGPath          string
	DistributionPath string
	InfoPlistPath    string
	EntitlementsPath string
	BuildNumber      string
	Version          string
}

// WailsBuildConfig defines the Wails v3 build inputs for a macOS app bundle.
type WailsBuildConfig struct {
	ProjectDir string   `json:"project_dir" yaml:"project_dir"`
	Name       string   `json:"name" yaml:"name"`
	Arch       string   `json:"arch" yaml:"arch"`
	BuildTags  []string `json:"build_tags" yaml:"build_tags"`
	LDFlags    []string `json:"ldflags" yaml:"ldflags"`
	OutputDir  string   `json:"output_dir" yaml:"output_dir"`
	Version    string   `json:"version" yaml:"version"`
	Env        []string `json:"env" yaml:"env"`
	DenoBuild  string   `json:"deno_build" yaml:"deno_build"`
}

// SignConfig defines the codesign inputs for a macOS app bundle.
type SignConfig struct {
	AppPath      string `json:"app_path" yaml:"app_path"`
	Identity     string `json:"identity" yaml:"identity"`
	Entitlements string `json:"entitlements" yaml:"entitlements"`
	Hardened     bool   `json:"hardened" yaml:"hardened"`
	Deep         bool   `json:"deep" yaml:"deep"`
	KeychainPath string `json:"keychain_path" yaml:"keychain_path"`
}

// NotariseConfig defines the Apple notarisation request.
type NotariseConfig struct {
	AppPath string `json:"app_path" yaml:"app_path"`

	APIKeyID       string `json:"api_key_id" yaml:"api_key_id"`
	APIKeyIssuerID string `json:"api_key_issuer_id" yaml:"api_key_issuer_id"`
	APIKeyPath     string `json:"api_key_path" yaml:"api_key_path"`

	TeamID   string `json:"team_id" yaml:"team_id"`
	AppleID  string `json:"apple_id" yaml:"apple_id"`
	Password string `json:"password" yaml:"password"`
}

// DMGConfig defines the DMG packaging inputs.
type DMGConfig struct {
	AppPath    string `json:"app_path" yaml:"app_path"`
	OutputPath string `json:"output_path" yaml:"output_path"`
	VolumeName string `json:"volume_name" yaml:"volume_name"`
	Background string `json:"background" yaml:"background"`
	IconSize   int    `json:"icon_size" yaml:"icon_size"`
	WindowSize [2]int `json:"window_size" yaml:"window_size"`
}

// TestFlightConfig defines the TestFlight upload inputs.
type TestFlightConfig struct {
	AppPath        string `json:"app_path" yaml:"app_path"`
	APIKeyID       string `json:"api_key_id" yaml:"api_key_id"`
	APIKeyIssuerID string `json:"api_key_issuer_id" yaml:"api_key_issuer_id"`
	APIKeyPath     string `json:"api_key_path" yaml:"api_key_path"`
	CertIdentity   string `json:"cert_identity" yaml:"cert_identity"`
}

// AppStoreConfig defines the App Store Connect submission inputs.
type AppStoreConfig struct {
	AppPath        string `json:"app_path" yaml:"app_path"`
	APIKeyID       string `json:"api_key_id" yaml:"api_key_id"`
	APIKeyIssuerID string `json:"api_key_issuer_id" yaml:"api_key_issuer_id"`
	APIKeyPath     string `json:"api_key_path" yaml:"api_key_path"`
	CertIdentity   string `json:"cert_identity" yaml:"cert_identity"`
	Version        string `json:"version" yaml:"version"`
	ReleaseType    string `json:"release_type" yaml:"release_type"`
}

// InfoPlist defines the generated macOS application metadata.
type InfoPlist struct {
	BundleID                      string `json:"bundle_id" plist:"CFBundleIdentifier"`
	BundleName                    string `json:"bundle_name" plist:"CFBundleName"`
	BundleDisplayName             string `json:"bundle_display_name" plist:"CFBundleDisplayName"`
	BundleVersion                 string `json:"bundle_version" plist:"CFBundleShortVersionString"`
	BuildNumber                   string `json:"build_number" plist:"CFBundleVersion"`
	MinSystemVersion              string `json:"min_system_version" plist:"LSMinimumSystemVersion"`
	Category                      string `json:"category" plist:"LSApplicationCategoryType"`
	Copyright                     string `json:"copyright" plist:"NSHumanReadableCopyright"`
	Executable                    string `json:"executable" plist:"CFBundleExecutable"`
	HighResCapable                bool   `json:"high_res_capable" plist:"NSHighResolutionCapable"`
	SupportsSecureRestorableState bool   `json:"supports_secure_restorable_state" plist:"NSSupportsSecureRestorableState"`
}

// Entitlements defines the generated macOS entitlements profile.
type Entitlements struct {
	Sandbox               bool `json:"sandbox" plist:"com.apple.security.app-sandbox"`
	NetworkClient         bool `json:"network_client" plist:"com.apple.security.network.client"`
	NetworkServer         bool `json:"network_server" plist:"com.apple.security.network.server"`
	MetalGPU              bool `json:"metal_gpu" plist:"com.apple.security.device.metal"`
	UserSelectedReadWrite bool `json:"user_selected_read_write" plist:"com.apple.security.files.user-selected.read-write"`
	Downloads             bool `json:"downloads" plist:"com.apple.security.files.downloads.read-write"`
	HardenedRuntime       bool `json:"hardened_runtime" plist:"com.apple.security.cs.allow-unsigned-executable-memory"`
	JIT                   bool `json:"jit" plist:"com.apple.security.cs.allow-jit"`
	DylibEnvVar           bool `json:"dylib_env_var" plist:"com.apple.security.cs.allow-dylib-environment-variables"`
}

var (
	appleBuildWailsAppFn    = BuildWailsApp
	appleCreateUniversalFn  = CreateUniversal
	appleSignFn             = Sign
	appleNotariseFn         = Notarise
	appleCreateDMGFn        = CreateDMG
	appleUploadTestFlightFn = UploadTestFlight
	appleSubmitAppStoreFn   = SubmitAppStore
	appleResolveCommand     = ax.ResolveCommand
	appleCombinedOutput     = ax.CombinedOutput
)

// DefaultAppleOptions returns the runtime defaults for the Apple build pipeline.
func DefaultAppleOptions() AppleOptions {
	return AppleOptions{
		Arch:             defaultAppleArch,
		Sign:             true,
		Notarise:         true,
		MinSystemVersion: defaultAppleMinSystemVersion,
		Category:         defaultAppleCategory,
	}
}

// Resolve materialises a config-backed Apple runtime option set.
func (cfg AppleConfig) Resolve() AppleOptions {
	options := DefaultAppleOptions()

	if cfg.TeamID != "" {
		options.TeamID = cfg.TeamID
	}
	if cfg.BundleID != "" {
		options.BundleID = cfg.BundleID
	}
	if cfg.Arch != "" {
		options.Arch = cfg.Arch
	}
	if cfg.CertIdentity != "" {
		options.CertIdentity = cfg.CertIdentity
	}
	if cfg.ProfilePath != "" {
		options.ProfilePath = cfg.ProfilePath
	}
	if cfg.KeychainPath != "" {
		options.KeychainPath = cfg.KeychainPath
	}
	if cfg.MetadataPath != "" {
		options.MetadataPath = cfg.MetadataPath
	}
	if cfg.Sign != nil {
		options.Sign = *cfg.Sign
	}
	if cfg.Notarise != nil {
		options.Notarise = *cfg.Notarise
	}
	if cfg.DMG != nil {
		options.DMG = *cfg.DMG
	}
	if cfg.TestFlight != nil {
		options.TestFlight = *cfg.TestFlight
	}
	if cfg.AppStore != nil {
		options.AppStore = *cfg.AppStore
	}
	if cfg.APIKeyID != "" {
		options.APIKeyID = cfg.APIKeyID
	}
	if cfg.APIKeyIssuerID != "" {
		options.APIKeyIssuerID = cfg.APIKeyIssuerID
	}
	if cfg.APIKeyPath != "" {
		options.APIKeyPath = cfg.APIKeyPath
	}
	if cfg.AppleID != "" {
		options.AppleID = cfg.AppleID
	}
	if cfg.Password != "" {
		options.Password = cfg.Password
	}
	if cfg.BundleDisplayName != "" {
		options.BundleDisplayName = cfg.BundleDisplayName
	}
	if cfg.MinSystemVersion != "" {
		options.MinSystemVersion = cfg.MinSystemVersion
	}
	if cfg.Category != "" {
		options.Category = cfg.Category
	}
	if cfg.Copyright != "" {
		options.Copyright = cfg.Copyright
	}
	if cfg.PrivacyPolicyURL != "" {
		options.PrivacyPolicyURL = cfg.PrivacyPolicyURL
	}
	if cfg.DMGBackground != "" {
		options.DMGBackground = cfg.DMGBackground
	}
	if cfg.DMGVolumeName != "" {
		options.DMGVolumeName = cfg.DMGVolumeName
	}
	if cfg.EntitlementsPath != "" {
		options.EntitlementsPath = cfg.EntitlementsPath
	}

	return options
}

func validateAppleBuildOptions(options AppleOptions) error {
	if options.Sign && core.Trim(options.CertIdentity) == "" {
		return coreerr.E("build.validateAppleBuildOptions", "signing identity is required when sign is enabled", nil)
	}

	if options.Notarise {
		if _, err := notariseAuthArgs(NotariseConfig{
			AppPath:        "",
			APIKeyID:       options.APIKeyID,
			APIKeyIssuerID: options.APIKeyIssuerID,
			APIKeyPath:     options.APIKeyPath,
			TeamID:         options.TeamID,
			AppleID:        options.AppleID,
			Password:       options.Password,
		}); err != nil {
			return coreerr.E("build.validateAppleBuildOptions", "invalid notarisation credentials", err)
		}
	}

	if options.TestFlight || options.AppStore {
		if err := validateAppStoreConnectAPIKey(options.APIKeyID, options.APIKeyIssuerID, options.APIKeyPath, "build.validateAppleBuildOptions"); err != nil {
			return err
		}
		if core.Trim(options.ProfilePath) == "" {
			return coreerr.E("build.validateAppleBuildOptions", "profile_path is required for App Store Connect uploads", nil)
		}
		if isDeveloperIDIdentity(options.CertIdentity) {
			return coreerr.E("build.validateAppleBuildOptions", "TestFlight and App Store uploads require an Apple distribution certificate, not Developer ID", nil)
		}
	}

	if options.AppStore {
		minSystemVersion := firstNonEmpty(options.MinSystemVersion, defaultAppleMinSystemVersion)
		if compareAppleVersion(minSystemVersion, defaultAppleMinSystemVersion) < 0 {
			return coreerr.E("build.validateAppleBuildOptions", "App Store submissions require min_system_version 13.0 or newer", nil)
		}

		if core.Trim(firstNonEmpty(options.Category, defaultAppleCategory)) == "" {
			return coreerr.E("build.validateAppleBuildOptions", "App Store submissions require an application category", nil)
		}

		if !core.Contains(core.Lower(options.Copyright), "eupl-1.2") {
			return coreerr.E("build.validateAppleBuildOptions", "App Store submissions must declare EUPL-1.2 in copyright metadata", nil)
		}

		if err := validatePrivacyPolicyURL(options.PrivacyPolicyURL); err != nil {
			return err
		}
	}

	return nil
}

// BuildApple runs the end-to-end macOS Apple pipeline for a Wails app.
func BuildApple(ctx context.Context, cfg *Config, options AppleOptions, buildNumber string) (*AppleBuildResult, error) {
	if cfg == nil {
		return nil, coreerr.E("build.BuildApple", "config is nil", nil)
	}
	if cfg.FS == nil {
		cfg.FS = io.Local
	}

	if options.BundleID == "" {
		return nil, coreerr.E("build.BuildApple", "bundle_id is required for Apple builds", nil)
	}
	if options.Notarise && !options.Sign {
		return nil, coreerr.E("build.BuildApple", "notarisation requires code signing", nil)
	}
	if (options.TestFlight || options.AppStore) && !options.Sign {
		return nil, coreerr.E("build.BuildApple", "TestFlight and App Store uploads require code signing", nil)
	}
	if err := validateAppleBuildOptions(options); err != nil {
		return nil, err
	}

	name := resolveAppleBundleName(cfg)
	outputDir := resolveAppleOutputDir(cfg)
	if err := cfg.FS.EnsureDir(outputDir); err != nil {
		return nil, coreerr.E("build.BuildApple", "failed to create Apple output directory", err)
	}

	if buildNumber == "" {
		buildNumber = "1"
	}

	buildTags := deduplicateStrings(append(append([]string{}, cfg.BuildTags...), "mlx"))
	ldflags := append([]string{}, cfg.LDFlags...)
	version := cfg.Version

	var bundlePath string
	if options.Arch == "" {
		options.Arch = defaultAppleArch
	}

	switch options.Arch {
	case "universal":
		arm64Dir, err := ax.TempDir("core-build-apple-arm64-*")
		if err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to create arm64 temp directory", err)
		}
		defer func() { _ = ax.RemoveAll(arm64Dir) }()

		amd64Dir, err := ax.TempDir("core-build-apple-amd64-*")
		if err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to create amd64 temp directory", err)
		}
		defer func() { _ = ax.RemoveAll(amd64Dir) }()

		arm64Bundle, err := appleBuildWailsAppFn(ctx, WailsBuildConfig{
			ProjectDir: cfg.ProjectDir,
			Name:       name,
			Arch:       "arm64",
			BuildTags:  buildTags,
			LDFlags:    ldflags,
			OutputDir:  arm64Dir,
			Version:    version,
			Env:        BuildEnvironment(cfg),
			DenoBuild:  cfg.DenoBuild,
		})
		if err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to build arm64 bundle", err)
		}

		amd64Bundle, err := appleBuildWailsAppFn(ctx, WailsBuildConfig{
			ProjectDir: cfg.ProjectDir,
			Name:       name,
			Arch:       "amd64",
			BuildTags:  buildTags,
			LDFlags:    ldflags,
			OutputDir:  amd64Dir,
			Version:    version,
			Env:        BuildEnvironment(cfg),
			DenoBuild:  cfg.DenoBuild,
		})
		if err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to build amd64 bundle", err)
		}

		bundlePath = ax.Join(outputDir, name+".app")
		if err := appleCreateUniversalFn(arm64Bundle, amd64Bundle, bundlePath); err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to create universal app bundle", err)
		}
	case "arm64", "amd64":
		var err error
		bundlePath, err = appleBuildWailsAppFn(ctx, WailsBuildConfig{
			ProjectDir: cfg.ProjectDir,
			Name:       name,
			Arch:       options.Arch,
			BuildTags:  buildTags,
			LDFlags:    ldflags,
			OutputDir:  outputDir,
			Version:    version,
			Env:        BuildEnvironment(cfg),
			DenoBuild:  cfg.DenoBuild,
		})
		if err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to build app bundle", err)
		}
	default:
		return nil, coreerr.E("build.BuildApple", "unsupported Apple arch: "+options.Arch, nil)
	}

	infoPlist := InfoPlist{
		BundleID:                      options.BundleID,
		BundleName:                    name,
		BundleDisplayName:             firstNonEmpty(options.BundleDisplayName, name),
		BundleVersion:                 normalizeAppleVersion(version),
		BuildNumber:                   buildNumber,
		MinSystemVersion:              firstNonEmpty(options.MinSystemVersion, defaultAppleMinSystemVersion),
		Category:                      firstNonEmpty(options.Category, defaultAppleCategory),
		Copyright:                     options.Copyright,
		Executable:                    name,
		HighResCapable:                true,
		SupportsSecureRestorableState: true,
	}

	infoPlistPath, err := WriteInfoPlist(cfg.FS, bundlePath, infoPlist)
	if err != nil {
		return nil, coreerr.E("build.BuildApple", "failed to write Info.plist", err)
	}

	if options.ProfilePath != "" {
		if err := copyPath(cfg.FS, options.ProfilePath, ax.Join(bundlePath, "Contents", "embedded.provisionprofile")); err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to copy provisioning profile", err)
		}
	}

	entitlementsPath := options.EntitlementsPath
	if entitlementsPath == "" {
		entitlementsPath = ax.Join(outputDir, name+".entitlements")
	}
	entitlements := directDistributionEntitlements()
	if options.AppStore || options.TestFlight {
		entitlements = appStoreEntitlements()
	}
	if err := WriteEntitlements(cfg.FS, entitlementsPath, entitlements); err != nil {
		return nil, coreerr.E("build.BuildApple", "failed to write entitlements", err)
	}

	if options.Sign {
		if err := appleSignFn(ctx, SignConfig{
			AppPath:      bundlePath,
			Identity:     options.CertIdentity,
			Entitlements: entitlementsPath,
			Hardened:     true,
			Deep:         false,
			KeychainPath: options.KeychainPath,
		}); err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to sign app bundle", err)
		}
	}

	distributionPath := bundlePath
	dmgPath := ""
	if options.DMG {
		dmgPath = ax.Join(outputDir, core.Sprintf("%s-%s.dmg", name, normalizeAppleVersion(version)))
		if err := appleCreateDMGFn(ctx, DMGConfig{
			AppPath:    bundlePath,
			OutputPath: dmgPath,
			VolumeName: firstNonEmpty(options.DMGVolumeName, name),
			Background: options.DMGBackground,
			IconSize:   128,
			WindowSize: [2]int{640, 480},
		}); err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to create DMG", err)
		}
		if options.Sign {
			if err := appleSignFn(ctx, SignConfig{
				AppPath:      dmgPath,
				Identity:     options.CertIdentity,
				Hardened:     false,
				Deep:         false,
				KeychainPath: options.KeychainPath,
			}); err != nil {
				return nil, coreerr.E("build.BuildApple", "failed to sign DMG", err)
			}
		}
		distributionPath = dmgPath
	}

	if options.Notarise {
		if err := appleNotariseFn(ctx, NotariseConfig{
			AppPath:        distributionPath,
			APIKeyID:       options.APIKeyID,
			APIKeyIssuerID: options.APIKeyIssuerID,
			APIKeyPath:     options.APIKeyPath,
			TeamID:         options.TeamID,
			AppleID:        options.AppleID,
			Password:       options.Password,
		}); err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to notarise distribution", err)
		}
	}

	if options.TestFlight {
		if err := appleUploadTestFlightFn(ctx, TestFlightConfig{
			AppPath:        bundlePath,
			APIKeyID:       options.APIKeyID,
			APIKeyIssuerID: options.APIKeyIssuerID,
			APIKeyPath:     options.APIKeyPath,
			CertIdentity:   options.CertIdentity,
		}); err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to upload TestFlight build", err)
		}
	}

	if options.AppStore {
		if err := validateAppStorePreflight(cfg.FS, cfg.ProjectDir, bundlePath, options); err != nil {
			return nil, err
		}

		if err := appleSubmitAppStoreFn(ctx, AppStoreConfig{
			AppPath:        bundlePath,
			APIKeyID:       options.APIKeyID,
			APIKeyIssuerID: options.APIKeyIssuerID,
			APIKeyPath:     options.APIKeyPath,
			CertIdentity:   options.CertIdentity,
			Version:        normalizeAppleVersion(version),
			ReleaseType:    "manual",
		}); err != nil {
			return nil, coreerr.E("build.BuildApple", "failed to submit App Store build", err)
		}
	}

	return &AppleBuildResult{
		BundlePath:       bundlePath,
		DMGPath:          dmgPath,
		DistributionPath: distributionPath,
		InfoPlistPath:    infoPlistPath,
		EntitlementsPath: entitlementsPath,
		BuildNumber:      buildNumber,
		Version:          normalizeAppleVersion(version),
	}, nil
}

// BuildWailsApp builds a single-architecture Wails app bundle for macOS.
func BuildWailsApp(ctx context.Context, cfg WailsBuildConfig) (string, error) {
	if cfg.ProjectDir == "" {
		return "", coreerr.E("build.BuildWailsApp", "project directory is required", nil)
	}

	name := cfg.Name
	if name == "" {
		name = ax.Base(cfg.ProjectDir)
	}
	if cfg.Arch == "" {
		return "", coreerr.E("build.BuildWailsApp", "arch is required", nil)
	}

	if err := prepareWailsFrontend(ctx, cfg); err != nil {
		return "", err
	}

	wailsCommand, err := resolveWails3Cli()
	if err != nil {
		return "", err
	}

	args := []string{"build", "-platform", "darwin/" + cfg.Arch}

	buildTags := deduplicateStrings(append(append([]string{}, cfg.BuildTags...), "mlx"))
	if len(buildTags) > 0 {
		args = append(args, "-tags", core.Join(",", buildTags...))
	}

	ldflags := append([]string{}, cfg.LDFlags...)
	if cfg.Version != "" && !appleHasVersionLDFlag(ldflags) {
		ldflags = append(ldflags, core.Sprintf("-X main.version=%s", cfg.Version))
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags", core.Join(" ", ldflags...))
	}

	env := append([]string{}, cfg.Env...)
	env = appendEnvIfMissing(env, "CGO_ENABLED", "1")

	output, err := appleCombinedOutput(ctx, cfg.ProjectDir, env, wailsCommand, args...)
	if err != nil {
		return "", coreerr.E("build.BuildWailsApp", "wails build failed: "+output, err)
	}

	sourcePath, err := findBuiltAppBundle(cfg.ProjectDir, name)
	if err != nil {
		return "", err
	}

	if cfg.OutputDir == "" {
		return sourcePath, nil
	}

	if err := io.Local.EnsureDir(cfg.OutputDir); err != nil {
		return "", coreerr.E("build.BuildWailsApp", "failed to create Wails output directory", err)
	}

	destPath := ax.Join(cfg.OutputDir, name+".app")
	if io.Local.Exists(destPath) {
		if err := io.Local.DeleteAll(destPath); err != nil {
			return "", coreerr.E("build.BuildWailsApp", "failed to replace existing app bundle", err)
		}
	}
	if err := copyPath(io.Local, sourcePath, destPath); err != nil {
		return "", coreerr.E("build.BuildWailsApp", "failed to copy built app bundle", err)
	}

	return destPath, nil
}

func prepareWailsFrontend(ctx context.Context, cfg WailsBuildConfig) error {
	frontendDir, command, args, err := resolveWailsFrontendBuild(cfg)
	if err != nil {
		return err
	}
	if command == "" {
		return nil
	}

	output, err := appleCombinedOutput(ctx, frontendDir, cfg.Env, command, args...)
	if err != nil {
		return coreerr.E("build.prepareWailsFrontend", command+" build failed: "+output, err)
	}

	return nil
}

func resolveWailsFrontendBuild(cfg WailsBuildConfig) (string, string, []string, error) {
	frontendDir := resolveFrontendDir(io.Local, cfg.ProjectDir)
	if frontendDir == "" {
		if DenoRequested(cfg.DenoBuild) {
			frontendDir = cfg.ProjectDir
			if io.Local.IsDir(ax.Join(cfg.ProjectDir, "frontend")) {
				frontendDir = ax.Join(cfg.ProjectDir, "frontend")
			}
		} else {
			return "", "", nil, nil
		}
	}

	if hasDenoConfig(io.Local, frontendDir) || DenoRequested(cfg.DenoBuild) {
		command, args, err := resolveDenoBuildCommand(cfg)
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, args, nil
	}

	if io.Local.IsFile(ax.Join(frontendDir, "package.json")) {
		return resolvePackageManagerBuild(frontendDir, detectPackageManager(io.Local, frontendDir))
	}

	return "", "", nil, nil
}

func resolveFrontendDir(filesystem io.Medium, projectDir string) string {
	frontendDir := ax.Join(projectDir, "frontend")
	if filesystem.IsDir(frontendDir) && (hasDenoConfig(filesystem, frontendDir) || filesystem.IsFile(ax.Join(frontendDir, "package.json"))) {
		return frontendDir
	}

	if hasDenoConfig(filesystem, projectDir) || filesystem.IsFile(ax.Join(projectDir, "package.json")) {
		return projectDir
	}

	if nested := resolveSubtreeFrontendDir(filesystem, projectDir); nested != "" {
		return nested
	}

	if DenoRequested("") {
		if filesystem.IsDir(frontendDir) {
			return frontendDir
		}
		return projectDir
	}

	return ""
}

func hasDenoConfig(filesystem io.Medium, dir string) bool {
	return filesystem.IsFile(ax.Join(dir, "deno.json")) || filesystem.IsFile(ax.Join(dir, "deno.jsonc"))
}

func resolveSubtreeFrontendDir(filesystem io.Medium, projectDir string) string {
	return findFrontendDir(filesystem, projectDir, 0)
}

func findFrontendDir(filesystem io.Medium, dir string, depth int) string {
	if depth >= 2 {
		return ""
	}

	entries, err := filesystem.List(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if name == "node_modules" || core.HasPrefix(name, ".") {
			continue
		}

		candidateDir := ax.Join(dir, name)
		if hasDenoConfig(filesystem, candidateDir) || filesystem.IsFile(ax.Join(candidateDir, "package.json")) {
			return candidateDir
		}

		if nested := findFrontendDir(filesystem, candidateDir, depth+1); nested != "" {
			return nested
		}
	}

	return ""
}

func resolvePackageManagerBuild(frontendDir, packageManager string) (string, string, []string, error) {
	switch packageManager {
	case "bun":
		command, err := resolveBunCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"run", "build"}, nil
	case "pnpm":
		command, err := resolvePnpmCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"run", "build"}, nil
	case "yarn":
		command, err := resolveYarnCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"build"}, nil
	default:
		command, err := resolveNpmCli()
		if err != nil {
			return "", "", nil, err
		}
		return frontendDir, command, []string{"run", "build"}, nil
	}
}

func detectPackageManager(filesystem io.Medium, dir string) string {
	if declared := detectDeclaredPackageManager(filesystem, dir); declared != "" {
		return declared
	}

	lockFiles := []struct {
		file    string
		manager string
	}{
		{"bun.lock", "bun"},
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}

	for _, lockFile := range lockFiles {
		if filesystem.IsFile(ax.Join(dir, lockFile.file)) {
			return lockFile.manager
		}
	}

	return "npm"
}

type packageJSONManifest struct {
	PackageManager string `json:"packageManager"`
}

func detectDeclaredPackageManager(filesystem io.Medium, dir string) string {
	content, err := filesystem.Read(ax.Join(dir, "package.json"))
	if err != nil {
		return ""
	}

	var manifest packageJSONManifest
	if err := ax.JSONUnmarshal([]byte(content), &manifest); err != nil {
		return ""
	}

	return normalisePackageManager(manifest.PackageManager)
}

func normalisePackageManager(value string) string {
	value = core.Trim(value)
	if value == "" {
		return ""
	}

	parts := core.SplitN(value, "@", 2)
	manager := parts[0]

	switch manager {
	case "bun", "pnpm", "yarn", "npm":
		return manager
	default:
		return ""
	}
}

func resolveDenoBuildCommand(cfg WailsBuildConfig) (string, []string, error) {
	override := core.Trim(core.Env("DENO_BUILD"))
	if override == "" {
		override = core.Trim(cfg.DenoBuild)
	}
	if override != "" {
		args, err := splitCommandLine(override)
		if err != nil {
			return "", nil, coreerr.E("build.resolveDenoBuildCommand", "invalid DENO_BUILD command", err)
		}
		if len(args) == 0 {
			return "", nil, coreerr.E("build.resolveDenoBuildCommand", "DENO_BUILD command is empty", nil)
		}
		return args[0], args[1:], nil
	}

	command, err := resolveDenoCli()
	if err != nil {
		return "", nil, err
	}
	return command, []string{"task", "build"}, nil
}

func splitCommandLine(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, nil
	}

	var (
		args    []string
		current strings.Builder
		quote   rune
		escape  bool
	)

	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}

	for _, r := range command {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
		case r == '\\' && quote != '\'':
			escape = true
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '"' || r == '\'':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if escape {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, coreerr.E("build.splitCommandLine", "unterminated quote in command", nil)
	}

	flush()
	return args, nil
}

// CreateUniversal merges two architecture-specific app bundles into a universal app.
func CreateUniversal(arm64Path, amd64Path, outputPath string) error {
	if arm64Path == "" || amd64Path == "" || outputPath == "" {
		return coreerr.E("build.CreateUniversal", "arm64, amd64, and output paths are required", nil)
	}

	if io.Local.Exists(outputPath) {
		if err := io.Local.DeleteAll(outputPath); err != nil {
			return coreerr.E("build.CreateUniversal", "failed to replace existing output bundle", err)
		}
	}

	if err := io.Local.EnsureDir(ax.Dir(outputPath)); err != nil {
		return coreerr.E("build.CreateUniversal", "failed to create universal output directory", err)
	}
	if err := copyPath(io.Local, arm64Path, outputPath); err != nil {
		return coreerr.E("build.CreateUniversal", "failed to copy arm64 bundle", err)
	}

	lipoCommand, err := resolveLipoCli()
	if err != nil {
		return err
	}

	for _, candidate := range universalMergeCandidates(io.Local, arm64Path, amd64Path) {
		armCandidate := ax.Join(arm64Path, candidate)
		amdCandidate := ax.Join(amd64Path, candidate)
		outputCandidate := ax.Join(outputPath, candidate)
		output, err := appleCombinedOutput(context.Background(), "", nil, lipoCommand, "-create", "-output", outputCandidate, armCandidate, amdCandidate)
		if err != nil {
			return coreerr.E("build.CreateUniversal", "lipo failed for "+candidate+": "+output, err)
		}
	}

	return nil
}

// Sign code-signs an app bundle or Apple artefact.
func Sign(ctx context.Context, cfg SignConfig) error {
	if cfg.AppPath == "" {
		return coreerr.E("build.Sign", "app_path is required", nil)
	}
	if cfg.Identity == "" {
		return coreerr.E("build.Sign", "signing identity is required", nil)
	}

	codesignCommand, err := resolveCodesignCli()
	if err != nil {
		return err
	}

	if !io.Local.IsDir(cfg.AppPath) || !core.HasSuffix(cfg.AppPath, ".app") {
		_, err := appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, cfg.AppPath, cfg.Entitlements)...)
		if err != nil {
			return coreerr.E("build.Sign", "codesign failed for "+cfg.AppPath, err)
		}
		return nil
	}

	for _, path := range signFrameworkPaths(cfg.AppPath) {
		output, err := appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, path, "")...)
		if err != nil {
			return coreerr.E("build.Sign", "codesign failed for framework "+path+": "+output, err)
		}
	}

	mainBinary := bundleExecutablePath(cfg.AppPath)
	for _, path := range signHelperBinaryPaths(cfg.AppPath, mainBinary) {
		output, err := appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, path, "")...)
		if err != nil {
			return coreerr.E("build.Sign", "codesign failed for helper binary "+path+": "+output, err)
		}
	}

	output, err := appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, mainBinary, cfg.Entitlements)...)
	if err != nil {
		return coreerr.E("build.Sign", "codesign failed for main binary "+mainBinary+": "+output, err)
	}

	output, err = appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, cfg.AppPath, cfg.Entitlements)...)
	if err != nil {
		return coreerr.E("build.Sign", "codesign failed for app bundle "+cfg.AppPath+": "+output, err)
	}

	return nil
}

// Notarise submits a signed app bundle or DMG to Apple and staples the ticket.
func Notarise(ctx context.Context, cfg NotariseConfig) error {
	if cfg.AppPath == "" {
		return coreerr.E("build.Notarise", "app_path is required", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	notariseCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		notariseCtx, cancel = context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
	}

	authArgs, err := notariseAuthArgs(cfg)
	if err != nil {
		return err
	}

	dittoCommand, err := resolveDittocli()
	if err != nil {
		return err
	}
	xcrunCommand, err := resolveXcrunCli()
	if err != nil {
		return err
	}

	tempDir, err := ax.TempDir("core-build-notary-*")
	if err != nil {
		return coreerr.E("build.Notarise", "failed to create notarisation temp directory", err)
	}
	defer func() { _ = ax.RemoveAll(tempDir) }()

	zipPath := ax.Join(tempDir, ax.Base(cfg.AppPath)+".zip")
	output, err := appleCombinedOutput(notariseCtx, "", nil, dittoCommand, "-c", "-k", "--keepParent", cfg.AppPath, zipPath)
	if err != nil {
		return coreerr.E("build.Notarise", "failed to create notarisation archive: "+output, err)
	}

	submitArgs := []string{"notarytool", "submit", zipPath, "--wait", "--output-format", "json"}
	submitArgs = append(submitArgs, authArgs...)
	output, err = appleCombinedOutput(notariseCtx, "", nil, xcrunCommand, submitArgs...)
	if err != nil {
		requestID := extractNotaryRequestID(output)
		if requestID != "" {
			logArgs := []string{"notarytool", "log", requestID}
			logArgs = append(logArgs, authArgs...)
			if logOutput, logErr := appleCombinedOutput(notariseCtx, "", nil, xcrunCommand, logArgs...); logErr == nil && logOutput != "" {
				output = core.Join("\n", output, logOutput)
			}
		}
		return coreerr.E("build.Notarise", "notarisation failed: "+output, err)
	}

	status := parseNotaryStatus(output)
	if status != "" && core.Lower(status) != "accepted" {
		return coreerr.E("build.Notarise", "Apple rejected notarisation request with status "+status, nil)
	}

	output, err = appleCombinedOutput(notariseCtx, "", nil, xcrunCommand, "stapler", "staple", cfg.AppPath)
	if err != nil {
		return coreerr.E("build.Notarise", "failed to staple notarisation ticket: "+output, err)
	}

	if core.HasSuffix(cfg.AppPath, ".app") {
		spctlCommand, err := resolveSPCTLCli()
		if err != nil {
			return err
		}
		output, err = appleCombinedOutput(notariseCtx, "", nil, spctlCommand, "--assess", "--type", "execute", cfg.AppPath)
		if err != nil {
			return coreerr.E("build.Notarise", "Gatekeeper assessment failed: "+output, err)
		}
	}

	return nil
}

// CreateDMG packages an app bundle into a distributable DMG.
func CreateDMG(ctx context.Context, cfg DMGConfig) error {
	if cfg.AppPath == "" || cfg.OutputPath == "" {
		return coreerr.E("build.CreateDMG", "app_path and output_path are required", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	cfg = normaliseDMGConfig(cfg)

	tempDir, err := ax.TempDir("core-build-dmg-*")
	if err != nil {
		return coreerr.E("build.CreateDMG", "failed to create DMG staging directory", err)
	}
	defer func() { _ = ax.RemoveAll(tempDir) }()

	stageDir := ax.Join(tempDir, "stage")
	mountDir := ax.Join(tempDir, "mount")
	rwDMGPath := ax.Join(tempDir, "staging.dmg")
	if err := io.Local.EnsureDir(stageDir); err != nil {
		return coreerr.E("build.CreateDMG", "failed to create DMG stage directory", err)
	}

	appName := ax.Base(cfg.AppPath)
	stageAppPath := ax.Join(stageDir, appName)
	if err := copyPath(io.Local, cfg.AppPath, stageAppPath); err != nil {
		return coreerr.E("build.CreateDMG", "failed to stage app bundle", err)
	}

	if err := os.Symlink("/Applications", ax.Join(stageDir, "Applications")); err != nil {
		return coreerr.E("build.CreateDMG", "failed to create Applications symlink", err)
	}

	if cfg.Background != "" {
		backgroundDir := ax.Join(stageDir, ".background")
		if err := io.Local.EnsureDir(backgroundDir); err != nil {
			return coreerr.E("build.CreateDMG", "failed to create DMG background directory", err)
		}
		if err := copyPath(io.Local, cfg.Background, ax.Join(backgroundDir, ax.Base(cfg.Background))); err != nil {
			return coreerr.E("build.CreateDMG", "failed to stage DMG background", err)
		}
	}

	if err := io.Local.EnsureDir(ax.Dir(cfg.OutputPath)); err != nil {
		return coreerr.E("build.CreateDMG", "failed to create DMG output directory", err)
	}

	hdiutilCommand, err := resolveHdiutilCli()
	if err != nil {
		return err
	}
	osascriptCommand, err := resolveOsaScriptCli()
	if err != nil {
		return err
	}

	volumeName := firstNonEmpty(cfg.VolumeName, core.TrimSuffix(appName, ".app"))
	createArgs := []string{
		"create",
		"-volname", volumeName,
		"-srcfolder", stageDir,
		"-ov",
		"-format", "UDRW",
		rwDMGPath,
	}
	output, err := appleCombinedOutput(ctx, "", nil, hdiutilCommand, createArgs...)
	if err != nil {
		return coreerr.E("build.CreateDMG", "hdiutil failed: "+output, err)
	}

	if err := io.Local.EnsureDir(mountDir); err != nil {
		return coreerr.E("build.CreateDMG", "failed to create DMG mount directory", err)
	}

	attached := false
	defer func() {
		if attached {
			_ = detachDMG(context.Background(), hdiutilCommand, mountDir)
		}
	}()

	attachArgs := []string{
		"attach",
		"-readwrite",
		"-noverify",
		"-noautoopen",
		"-mountpoint", mountDir,
		rwDMGPath,
	}
	output, err = appleCombinedOutput(ctx, "", nil, hdiutilCommand, attachArgs...)
	if err != nil {
		return coreerr.E("build.CreateDMG", "failed to mount staging DMG: "+output, err)
	}
	attached = true

	scriptPath := ax.Join(tempDir, "layout.applescript")
	script := buildDMGAppleScript(volumeName, appName, cfg)
	if err := io.Local.WriteMode(scriptPath, script, 0o644); err != nil {
		return coreerr.E("build.CreateDMG", "failed to write DMG layout script", err)
	}

	output, err = appleCombinedOutput(ctx, "", nil, osascriptCommand, scriptPath)
	if err != nil {
		return coreerr.E("build.CreateDMG", "failed to configure Finder layout: "+output, err)
	}

	if err := detachDMG(ctx, hdiutilCommand, mountDir); err != nil {
		return err
	}
	attached = false

	convertArgs := []string{
		"convert",
		rwDMGPath,
		"-format", "UDZO",
		"-ov",
		"-o", cfg.OutputPath,
	}
	output, err = appleCombinedOutput(ctx, "", nil, hdiutilCommand, convertArgs...)
	if err != nil {
		return coreerr.E("build.CreateDMG", "failed to convert DMG: "+output, err)
	}

	return nil
}

func normaliseDMGConfig(cfg DMGConfig) DMGConfig {
	if cfg.IconSize <= 0 {
		cfg.IconSize = defaultDMGIconSize
	}
	if cfg.WindowSize[0] <= 0 || cfg.WindowSize[1] <= 0 {
		cfg.WindowSize = [2]int{defaultDMGWindowWidth, defaultDMGWindowHeight}
	}
	if cfg.VolumeName == "" {
		cfg.VolumeName = core.TrimSuffix(ax.Base(cfg.AppPath), ".app")
	}
	return cfg
}

func buildDMGAppleScript(volumeName, appName string, cfg DMGConfig) string {
	cfg = normaliseDMGConfig(cfg)
	appX, appY, applicationsX, applicationsY := dmgLayoutPositions(cfg.WindowSize, cfg.IconSize)

	backgroundLine := ""
	if cfg.Background != "" {
		backgroundLine = core.Sprintf("\n    set background picture of opts to file \".background:%s\"", escapeAppleScriptString(ax.Base(cfg.Background)))
	}

	return core.Sprintf(
		"tell application \"Finder\"\n"+
			"  tell disk \"%s\"\n"+
			"    open\n"+
			"    set current view of container window to icon view\n"+
			"    set toolbar visible of container window to false\n"+
			"    set statusbar visible of container window to false\n"+
			"    set bounds of container window to {100, 100, %d, %d}\n"+
			"    set opts to the icon view options of container window\n"+
			"    set arrangement of opts to not arranged\n"+
			"    set icon size of opts to %d%s\n"+
			"    set position of item \"%s\" of container window to {%d, %d}\n"+
			"    set position of item \"Applications\" of container window to {%d, %d}\n"+
			"    update without registering applications\n"+
			"    delay 1\n"+
			"    close\n"+
			"    open\n"+
			"    update without registering applications\n"+
			"    delay 1\n"+
			"  end tell\n"+
			"end tell\n",
		escapeAppleScriptString(volumeName),
		100+cfg.WindowSize[0],
		100+cfg.WindowSize[1],
		cfg.IconSize,
		backgroundLine,
		escapeAppleScriptString(appName),
		appX,
		appY,
		applicationsX,
		applicationsY,
	)
}

func dmgLayoutPositions(windowSize [2]int, iconSize int) (int, int, int, int) {
	width := windowSize[0]
	height := windowSize[1]
	if width <= 0 {
		width = defaultDMGWindowWidth
	}
	if height <= 0 {
		height = defaultDMGWindowHeight
	}
	if iconSize <= 0 {
		iconSize = defaultDMGIconSize
	}

	appX := width / 4
	if appX < iconSize+32 {
		appX = iconSize + 32
	}
	applicationsX := (width * 3) / 4
	if applicationsX <= appX {
		applicationsX = appX + iconSize + 96
	}
	appY := height / 2
	if appY < iconSize+32 {
		appY = iconSize + 32
	}

	return appX, appY, applicationsX, appY
}

func escapeAppleScriptString(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
}

func detachDMG(ctx context.Context, hdiutilCommand, mountDir string) error {
	output, err := appleCombinedOutput(ctx, "", nil, hdiutilCommand, "detach", mountDir)
	if err == nil {
		return nil
	}

	forceOutput, forceErr := appleCombinedOutput(ctx, "", nil, hdiutilCommand, "detach", mountDir, "-force")
	if forceErr != nil {
		message := output
		if forceOutput != "" {
			message = core.Join("\n", output, forceOutput)
		}
		return coreerr.E("build.CreateDMG", "failed to detach staging DMG: "+message, forceErr)
	}

	return nil
}

// UploadTestFlight uploads a packaged macOS artefact to TestFlight.
func UploadTestFlight(ctx context.Context, cfg TestFlightConfig) error {
	if cfg.AppPath == "" {
		return coreerr.E("build.UploadTestFlight", "app_path is required", nil)
	}
	if err := validateAppStoreConnectAPIKey(cfg.APIKeyID, cfg.APIKeyIssuerID, cfg.APIKeyPath, "build.UploadTestFlight"); err != nil {
		return err
	}

	uploadPath, env, cleanup, err := packageForASCUpload(ctx, cfg.AppPath, cfg.CertIdentity, cfg.APIKeyID, cfg.APIKeyPath)
	if err != nil {
		return err
	}
	defer cleanup()

	xcrunCommand, err := resolveXcrunCli()
	if err != nil {
		return err
	}

	output, err := appleCombinedOutput(ctx, "", env, xcrunCommand,
		"altool", "--upload-app", "--type", "macos",
		"--file", uploadPath,
		"--apiKey", cfg.APIKeyID,
		"--apiIssuer", cfg.APIKeyIssuerID,
	)
	if err != nil {
		return coreerr.E("build.UploadTestFlight", "altool upload failed: "+output, err)
	}

	return nil
}

// SubmitAppStore uploads a packaged macOS artefact for App Store Connect review.
func SubmitAppStore(ctx context.Context, cfg AppStoreConfig) error {
	if cfg.ReleaseType != "" && cfg.ReleaseType != "manual" && cfg.ReleaseType != "automatic" {
		return coreerr.E("build.SubmitAppStore", "release_type must be manual or automatic", nil)
	}
	if cfg.AppPath == "" {
		return coreerr.E("build.SubmitAppStore", "app_path is required", nil)
	}
	if err := validateAppStoreConnectAPIKey(cfg.APIKeyID, cfg.APIKeyIssuerID, cfg.APIKeyPath, "build.SubmitAppStore"); err != nil {
		return err
	}

	uploadPath, env, cleanup, err := packageForASCUpload(ctx, cfg.AppPath, cfg.CertIdentity, cfg.APIKeyID, cfg.APIKeyPath)
	if err != nil {
		return err
	}
	defer cleanup()

	xcrunCommand, err := resolveXcrunCli()
	if err != nil {
		return err
	}

	output, err := appleCombinedOutput(ctx, "", env, xcrunCommand,
		"altool", "--upload-app", "--type", "macos",
		"--file", uploadPath,
		"--apiKey", cfg.APIKeyID,
		"--apiIssuer", cfg.APIKeyIssuerID,
	)
	if err != nil {
		return coreerr.E("build.SubmitAppStore", "altool upload failed: "+output, err)
	}

	return nil
}

// WriteInfoPlist writes the app bundle Info.plist and returns its path.
func WriteInfoPlist(filesystem io.Medium, appPath string, plist InfoPlist) (string, error) {
	if filesystem == nil {
		filesystem = io.Local
	}

	plistPath := ax.Join(appPath, "Contents", "Info.plist")
	if err := filesystem.EnsureDir(ax.Dir(plistPath)); err != nil {
		return "", coreerr.E("build.WriteInfoPlist", "failed to create Info.plist directory", err)
	}

	content, err := encodePlist(plist.Values())
	if err != nil {
		return "", err
	}
	if err := filesystem.WriteMode(plistPath, content, 0o644); err != nil {
		return "", coreerr.E("build.WriteInfoPlist", "failed to write Info.plist", err)
	}

	return plistPath, nil
}

// WriteEntitlements writes an entitlements plist file.
func WriteEntitlements(filesystem io.Medium, path string, entitlements Entitlements) error {
	if filesystem == nil {
		filesystem = io.Local
	}
	if path == "" {
		return coreerr.E("build.WriteEntitlements", "entitlements path is required", nil)
	}

	if err := filesystem.EnsureDir(ax.Dir(path)); err != nil {
		return coreerr.E("build.WriteEntitlements", "failed to create entitlements directory", err)
	}

	content, err := encodePlist(entitlements.Values())
	if err != nil {
		return err
	}
	if err := filesystem.WriteMode(path, content, 0o644); err != nil {
		return coreerr.E("build.WriteEntitlements", "failed to write entitlements", err)
	}

	return nil
}

// Values converts InfoPlist to plist key/value pairs.
func (p InfoPlist) Values() map[string]any {
	return map[string]any{
		"CFBundleDisplayName":             p.BundleDisplayName,
		"CFBundleExecutable":              p.Executable,
		"CFBundleIdentifier":              p.BundleID,
		"CFBundleName":                    p.BundleName,
		"CFBundlePackageType":             "APPL",
		"CFBundleShortVersionString":      p.BundleVersion,
		"CFBundleVersion":                 p.BuildNumber,
		"LSApplicationCategoryType":       p.Category,
		"LSMinimumSystemVersion":          p.MinSystemVersion,
		"NSHighResolutionCapable":         p.HighResCapable,
		"NSHumanReadableCopyright":        p.Copyright,
		"NSSupportsSecureRestorableState": p.SupportsSecureRestorableState,
	}
}

// Values converts Entitlements to plist key/value pairs.
func (e Entitlements) Values() map[string]any {
	return map[string]any{
		"com.apple.security.app-sandbox":                          e.Sandbox,
		"com.apple.security.cs.allow-dylib-environment-variables": e.DylibEnvVar,
		"com.apple.security.cs.allow-jit":                         e.JIT,
		"com.apple.security.cs.allow-unsigned-executable-memory":  e.HardenedRuntime,
		"com.apple.security.device.metal":                         e.MetalGPU,
		"com.apple.security.files.downloads.read-write":           e.Downloads,
		"com.apple.security.files.user-selected.read-write":       e.UserSelectedReadWrite,
		"com.apple.security.network.client":                       e.NetworkClient,
		"com.apple.security.network.server":                       e.NetworkServer,
	}
}

func directDistributionEntitlements() Entitlements {
	return Entitlements{
		Sandbox:               false,
		NetworkClient:         true,
		NetworkServer:         true,
		MetalGPU:              true,
		UserSelectedReadWrite: true,
		Downloads:             true,
		HardenedRuntime:       true,
		JIT:                   true,
		DylibEnvVar:           false,
	}
}

func appStoreEntitlements() Entitlements {
	return Entitlements{
		Sandbox:               true,
		NetworkClient:         true,
		NetworkServer:         true,
		MetalGPU:              true,
		UserSelectedReadWrite: true,
		Downloads:             true,
		HardenedRuntime:       false,
		JIT:                   false,
		DylibEnvVar:           false,
	}
}

func resolveAppleBundleName(cfg *Config) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.Project.Binary != "" {
		return cfg.Project.Binary
	}
	if cfg.Project.Name != "" {
		return cfg.Project.Name
	}
	return ax.Base(cfg.ProjectDir)
}

func resolveAppleOutputDir(cfg *Config) string {
	if cfg.OutputDir != "" {
		return cfg.OutputDir
	}
	return ax.Join(cfg.ProjectDir, "dist", "apple")
}

func normalizeAppleVersion(version string) string {
	version = core.Trim(version)
	version = core.TrimPrefix(version, "v")
	if version == "" {
		return "0.0.1"
	}
	return version
}

func appleHasVersionLDFlag(ldflags []string) bool {
	for _, flag := range ldflags {
		if core.Contains(flag, "main.version=") || core.Contains(flag, "main.Version=") {
			return true
		}
	}
	return false
}

func findBuiltAppBundle(projectDir, name string) (string, error) {
	for _, candidate := range []string{
		ax.Join(projectDir, "build", "bin", name+".app"),
		ax.Join(projectDir, "dist", name+".app"),
		ax.Join(projectDir, name+".app"),
	} {
		if io.Local.Exists(candidate) {
			return candidate, nil
		}
	}
	return "", coreerr.E("build.findBuiltAppBundle", "Wails build completed but no .app bundle was found for "+name, nil)
}

func bundleExecutablePath(appPath string) string {
	executableName := core.TrimSuffix(ax.Base(appPath), ".app")
	infoPlistPath := ax.Join(appPath, "Contents", "Info.plist")
	if content, err := io.Local.Read(infoPlistPath); err == nil {
		if name := plistStringValue(content, "CFBundleExecutable"); name != "" {
			executableName = name
		}
	}
	return ax.Join(appPath, "Contents", "MacOS", executableName)
}

func universalMergeCandidates(filesystem io.Medium, arm64Path, amd64Path string) []string {
	candidates := map[string]struct{}{}
	seedUniversalMergeCandidates(filesystem, arm64Path, amd64Path, "", candidates)

	paths := make([]string, 0, len(candidates))
	for path := range candidates {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func seedUniversalMergeCandidates(filesystem io.Medium, arm64Path, amd64Path, relativePath string, candidates map[string]struct{}) {
	currentPath := arm64Path
	if relativePath != "" {
		currentPath = ax.Join(arm64Path, relativePath)
	}

	entries, err := filesystem.List(currentPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		entryRelativePath := entry.Name()
		if relativePath != "" {
			entryRelativePath = ax.Join(relativePath, entry.Name())
		}

		armEntryPath := ax.Join(arm64Path, entryRelativePath)
		amdEntryPath := ax.Join(amd64Path, entryRelativePath)
		if entry.IsDir() {
			if filesystem.IsDir(amdEntryPath) {
				seedUniversalMergeCandidates(filesystem, arm64Path, amd64Path, entryRelativePath, candidates)
			}
			continue
		}

		if !filesystem.IsFile(amdEntryPath) || !shouldMergeUniversalPath(filesystem, armEntryPath, entryRelativePath) {
			continue
		}
		candidates[entryRelativePath] = struct{}{}
	}
}

func shouldMergeUniversalPath(filesystem io.Medium, path, relativePath string) bool {
	info, err := filesystem.Stat(path)
	if err == nil && info.Mode()&0o111 != 0 {
		return true
	}

	lowerRelativePath := core.Lower(relativePath)
	if core.HasSuffix(lowerRelativePath, ".dylib") || core.HasSuffix(lowerRelativePath, ".so") {
		return true
	}

	for currentDir := ax.Dir(relativePath); currentDir != "." && currentDir != "" && currentDir != string(os.PathSeparator); currentDir = ax.Dir(currentDir) {
		base := ax.Base(currentDir)
		if core.HasSuffix(base, ".framework") {
			return ax.Base(relativePath) == core.TrimSuffix(base, ".framework")
		}
	}

	return false
}

func plistStringValue(content, key string) string {
	pattern := core.Sprintf("<key>%s</key>", key)
	index := strings.Index(content, pattern)
	if index == -1 {
		return ""
	}

	remainder := content[index+len(pattern):]
	startTag := "<string>"
	endTag := "</string>"
	start := strings.Index(remainder, startTag)
	end := strings.Index(remainder, endTag)
	if start == -1 || end == -1 || end <= start+len(startTag) {
		return ""
	}
	return core.Trim(remainder[start+len(startTag) : end])
}

func copyPath(filesystem io.Medium, sourcePath, destPath string) error {
	if filesystem == nil {
		filesystem = io.Local
	}

	if filesystem.IsDir(sourcePath) {
		if err := filesystem.EnsureDir(destPath); err != nil {
			return err
		}
		entries, err := filesystem.List(sourcePath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyPath(filesystem, ax.Join(sourcePath, entry.Name()), ax.Join(destPath, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	info, err := filesystem.Stat(sourcePath)
	if err != nil {
		return err
	}
	content, err := filesystem.Read(sourcePath)
	if err != nil {
		return err
	}
	return filesystem.WriteMode(destPath, content, info.Mode().Perm())
}

func signFrameworkPaths(appPath string) []string {
	frameworksDir := ax.Join(appPath, "Contents", "Frameworks")
	if !io.Local.IsDir(frameworksDir) {
		return nil
	}

	entries, err := io.Local.List(frameworksDir)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		paths = append(paths, ax.Join(frameworksDir, entry.Name()))
	}
	sort.Strings(paths)
	return paths
}

func signHelperBinaryPaths(appPath, mainBinary string) []string {
	macOSDir := ax.Join(appPath, "Contents", "MacOS")
	if !io.Local.IsDir(macOSDir) {
		return nil
	}

	entries, err := io.Local.List(macOSDir)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		path := ax.Join(macOSDir, entry.Name())
		if path == mainBinary {
			continue
		}
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0111 == 0 {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func codesignArgs(cfg SignConfig, path string, entitlements string) []string {
	args := []string{
		"--sign", cfg.Identity,
		"--timestamp",
		"--force",
	}
	if cfg.KeychainPath != "" {
		args = append(args, "--keychain", cfg.KeychainPath)
	}
	if cfg.Hardened {
		args = append(args, "--options", "runtime")
	}
	if cfg.Deep {
		args = append(args, "--deep")
	}
	if entitlements != "" {
		args = append(args, "--entitlements", entitlements)
	}
	args = append(args, path)
	return args
}

func notariseAuthArgs(cfg NotariseConfig) ([]string, error) {
	if cfg.APIKeyID != "" {
		if cfg.APIKeyIssuerID == "" || cfg.APIKeyPath == "" {
			return nil, coreerr.E("build.notariseAuthArgs", "api_key_issuer_id and api_key_path are required with api_key_id", nil)
		}
		return []string{
			"--key", cfg.APIKeyPath,
			"--key-id", cfg.APIKeyID,
			"--issuer", cfg.APIKeyIssuerID,
		}, nil
	}

	if cfg.AppleID == "" || cfg.Password == "" || cfg.TeamID == "" {
		return nil, coreerr.E("build.notariseAuthArgs", "team_id, apple_id, and password are required when API key auth is not configured", nil)
	}

	return []string{
		"--apple-id", cfg.AppleID,
		"--password", cfg.Password,
		"--team-id", cfg.TeamID,
	}, nil
}

func validateAppStoreConnectAPIKey(apiKeyID, apiKeyIssuerID, apiKeyPath, op string) error {
	switch {
	case core.Trim(apiKeyID) == "":
		return coreerr.E(op, "api_key_id is required for App Store Connect uploads", nil)
	case core.Trim(apiKeyIssuerID) == "":
		return coreerr.E(op, "api_key_issuer_id is required for App Store Connect uploads", nil)
	case core.Trim(apiKeyPath) == "":
		return coreerr.E(op, "api_key_path is required for App Store Connect uploads", nil)
	default:
		return nil
	}
}

func isDeveloperIDIdentity(identity string) bool {
	return core.Contains(core.Lower(identity), "developer id")
}

func validateAppStorePreflight(filesystem io.Medium, projectDir, bundlePath string, options AppleOptions) error {
	if filesystem == nil {
		filesystem = io.Local
	}

	if err := validateAppStoreMetadata(filesystem, projectDir, options.MetadataPath); err != nil {
		return err
	}
	if err := scanBundleForPrivateAPIUsage(filesystem, bundlePath); err != nil {
		return err
	}

	return nil
}

func validateAppStoreMetadata(filesystem io.Medium, projectDir, configuredPath string) error {
	metadataPath := resolveAppStoreMetadataPath(filesystem, projectDir, configuredPath)
	if metadataPath == "" {
		return coreerr.E("build.validateAppStoreMetadata", "App Store submissions require metadata_path or a standard metadata directory (.core/apple/appstore, .core/appstore, or appstore)", nil)
	}

	if !hasAppStoreDescription(filesystem, metadataPath) {
		return coreerr.E("build.validateAppStoreMetadata", "App Store submissions require a description file in metadata_path", nil)
	}
	if !hasAppStoreScreenshots(filesystem, metadataPath) {
		return coreerr.E("build.validateAppStoreMetadata", "App Store submissions require at least one screenshot in metadata_path/screenshots", nil)
	}

	return nil
}

func resolveAppStoreMetadataPath(filesystem io.Medium, projectDir, configuredPath string) string {
	candidates := []string{}
	if configuredPath != "" {
		if ax.IsAbs(configuredPath) {
			candidates = append(candidates, configuredPath)
		} else {
			candidates = append(candidates, ax.Join(projectDir, configuredPath))
		}
	}
	candidates = append(candidates,
		ax.Join(projectDir, ".core", "apple", "appstore"),
		ax.Join(projectDir, ".core", "appstore"),
		ax.Join(projectDir, "appstore"),
	)

	for _, candidate := range candidates {
		if candidate != "" && filesystem.IsDir(candidate) {
			return candidate
		}
	}

	return ""
}

func hasAppStoreDescription(filesystem io.Medium, metadataPath string) bool {
	for _, name := range []string{"description.txt", "description.md", "description.markdown"} {
		if filesystem.IsFile(ax.Join(metadataPath, name)) {
			return true
		}
	}
	return false
}

func hasAppStoreScreenshots(filesystem io.Medium, metadataPath string) bool {
	screenshotsDir := ax.Join(metadataPath, "screenshots")
	if !filesystem.IsDir(screenshotsDir) {
		return false
	}

	entries, err := filesystem.List(screenshotsDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := core.Lower(entry.Name())
		if core.HasSuffix(name, ".png") ||
			core.HasSuffix(name, ".jpg") ||
			core.HasSuffix(name, ".jpeg") ||
			core.HasSuffix(name, ".heic") {
			return true
		}
	}

	return false
}

func validatePrivacyPolicyURL(raw string) error {
	value := core.Trim(raw)
	if value == "" {
		return coreerr.E("build.validatePrivacyPolicyURL", "App Store submissions require privacy_policy_url (for example https://lthn.ai/privacy)", nil)
	}

	normalised := value
	if !strings.Contains(normalised, "://") {
		normalised = "https://" + normalised
	}

	parsed, err := url.Parse(normalised)
	if err != nil {
		return coreerr.E("build.validatePrivacyPolicyURL", "privacy_policy_url must be a valid URL", err)
	}
	if core.Trim(parsed.Host) == "" || parsed.Path == "" || parsed.Path == "/" {
		return coreerr.E("build.validatePrivacyPolicyURL", "privacy_policy_url must include a host and non-root path", nil)
	}

	return nil
}

func scanBundleForPrivateAPIUsage(filesystem io.Medium, bundlePath string) error {
	if bundlePath == "" {
		return coreerr.E("build.scanBundleForPrivateAPIUsage", "bundle path is required", nil)
	}

	for _, root := range privateAPIScanRoots(bundlePath) {
		for _, path := range collectBundleFiles(filesystem, root) {
			content, err := filesystem.Read(path)
			if err != nil {
				continue
			}
			if indicator := detectPrivateAPIIndicator(content); indicator != "" {
				return coreerr.E("build.scanBundleForPrivateAPIUsage", "private API usage detected in "+path+": "+indicator, nil)
			}
		}
	}

	return nil
}

func privateAPIScanRoots(bundlePath string) []string {
	return []string{
		ax.Join(bundlePath, "Contents", "MacOS"),
		ax.Join(bundlePath, "Contents", "Frameworks"),
	}
}

func collectBundleFiles(filesystem io.Medium, root string) []string {
	if filesystem == nil || !filesystem.Exists(root) {
		return nil
	}
	if !filesystem.IsDir(root) {
		return []string{root}
	}

	entries, err := filesystem.List(root)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		path := ax.Join(root, entry.Name())
		if entry.IsDir() {
			paths = append(paths, collectBundleFiles(filesystem, path)...)
			continue
		}
		paths = append(paths, path)
	}

	return paths
}

func detectPrivateAPIIndicator(content string) string {
	for _, indicator := range []string{
		"/System/Library/PrivateFrameworks/",
		"PrivateFrameworks/",
		"com.apple.private.",
		"LSApplicationWorkspace",
		"MobileInstallation",
		"SpringBoardServices",
	} {
		if strings.Contains(content, indicator) {
			return indicator
		}
	}

	return ""
}

func compareAppleVersion(left, right string) int {
	leftParts := appleVersionParts(left)
	rightParts := appleVersionParts(right)

	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}

	for i := 0; i < maxLen; i++ {
		var leftValue, rightValue int
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		switch {
		case leftValue < rightValue:
			return -1
		case leftValue > rightValue:
			return 1
		}
	}

	return 0
}

func appleVersionParts(value string) []int {
	value = core.Trim(core.TrimPrefix(value, "v"))
	if value == "" {
		return nil
	}

	rawParts := core.Split(value, ".")
	parts := make([]int, 0, len(rawParts))
	for _, rawPart := range rawParts {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			parts = append(parts, 0)
			continue
		}

		digits := strings.Builder{}
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			digits.WriteRune(r)
		}

		if digits.Len() == 0 {
			parts = append(parts, 0)
			continue
		}

		number, err := strconv.Atoi(digits.String())
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, number)
	}

	return parts
}

func extractNotaryRequestID(output string) string {
	if output == "" {
		return ""
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err == nil {
		return payload.ID
	}
	return ""
}

func parseNotaryStatus(output string) string {
	if output == "" {
		return ""
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err == nil {
		return payload.Status
	}
	return ""
}

func packageForASCUpload(ctx context.Context, appPath, certIdentity, apiKeyID, apiKeyPath string) (string, []string, func(), error) {
	if core.HasSuffix(appPath, ".pkg") {
		env, cleanup, err := prepareASCAPIKeyEnv(apiKeyID, apiKeyPath)
		if err != nil {
			return "", nil, nil, err
		}
		return appPath, env, cleanup, nil
	}

	if !core.HasSuffix(appPath, ".app") {
		return "", nil, nil, coreerr.E("build.packageForASCUpload", "App Store Connect uploads require a .app or .pkg input", nil)
	}

	outputPath := ax.Join(ax.Dir(appPath), core.TrimSuffix(ax.Base(appPath), ".app")+".pkg")
	if err := createDistributionPackage(ctx, appPath, certIdentity, outputPath); err != nil {
		return "", nil, nil, err
	}

	env, cleanup, err := prepareASCAPIKeyEnv(apiKeyID, apiKeyPath)
	if err != nil {
		return "", nil, nil, err
	}

	return outputPath, env, cleanup, nil
}

func prepareASCAPIKeyEnv(apiKeyID, apiKeyPath string) ([]string, func(), error) {
	if apiKeyPath == "" {
		return nil, func() {}, nil
	}

	expectedName := core.Sprintf("AuthKey_%s.p8", apiKeyID)
	if expectedName == "AuthKey_.p8" || ax.Base(apiKeyPath) == expectedName {
		return []string{"API_PRIVATE_KEYS_DIR=" + ax.Dir(apiKeyPath)}, func() {}, nil
	}

	content, err := io.Local.Read(apiKeyPath)
	if err != nil {
		return nil, nil, coreerr.E("build.prepareASCAPIKeyEnv", "failed to read App Store Connect API key", err)
	}

	tempDir, err := ax.TempDir("core-build-asc-key-*")
	if err != nil {
		return nil, nil, coreerr.E("build.prepareASCAPIKeyEnv", "failed to create App Store Connect key staging directory", err)
	}

	stagedPath := ax.Join(tempDir, expectedName)
	if err := io.Local.WriteMode(stagedPath, content, 0o600); err != nil {
		_ = ax.RemoveAll(tempDir)
		return nil, nil, coreerr.E("build.prepareASCAPIKeyEnv", "failed to stage App Store Connect API key", err)
	}

	return []string{"API_PRIVATE_KEYS_DIR=" + tempDir}, func() {
		_ = ax.RemoveAll(tempDir)
	}, nil
}

func createDistributionPackage(ctx context.Context, appPath, certIdentity, outputPath string) error {
	productbuildCommand, err := resolveProductbuildCli()
	if err != nil {
		return err
	}

	args := []string{"--component", appPath, "/Applications", outputPath}
	if certIdentity != "" {
		args = append([]string{"--sign", certIdentity}, args...)
	}

	output, err := appleCombinedOutput(ctx, "", nil, productbuildCommand, args...)
	if err != nil {
		return coreerr.E("build.createDistributionPackage", "productbuild failed: "+output, err)
	}

	return nil
}

func encodePlist(values map[string]any) (string, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`)
	buf.WriteString(`<plist version="1.0"><dict>`)

	for _, key := range keys {
		buf.WriteString("<key>")
		if err := xml.EscapeText(&buf, []byte(key)); err != nil {
			return "", coreerr.E("build.encodePlist", "failed to encode plist key", err)
		}
		buf.WriteString("</key>")

		switch value := values[key].(type) {
		case string:
			buf.WriteString("<string>")
			if err := xml.EscapeText(&buf, []byte(value)); err != nil {
				return "", coreerr.E("build.encodePlist", "failed to encode plist string value", err)
			}
			buf.WriteString("</string>")
		case bool:
			if value {
				buf.WriteString("<true/>")
			} else {
				buf.WriteString("<false/>")
			}
		case int:
			buf.WriteString("<integer>")
			buf.WriteString(strconv.Itoa(value))
			buf.WriteString("</integer>")
		default:
			return "", coreerr.E("build.encodePlist", "unsupported plist value type", nil)
		}
	}

	buf.WriteString("</dict></plist>")
	return buf.String(), nil
}

func appendEnvIfMissing(env []string, key, value string) []string {
	prefix := key + "="
	for _, entry := range env {
		if core.HasPrefix(entry, prefix) {
			return env
		}
	}
	return append(env, prefix+value)
}

func resolveWails3Cli() (string, error) {
	paths := []string{
		"/usr/local/bin/wails3",
		"/opt/homebrew/bin/wails3",
	}
	if home := core.Env("HOME"); home != "" {
		paths = append(paths, ax.Join(home, "go", "bin", "wails3"))
	}
	command, err := appleResolveCommand("wails3", paths...)
	if err == nil {
		return command, nil
	}

	fallbacks := []string{
		"/usr/local/bin/wails",
		"/opt/homebrew/bin/wails",
	}
	if home := core.Env("HOME"); home != "" {
		fallbacks = append(fallbacks, ax.Join(home, "go", "bin", "wails"))
	}
	command, fallbackErr := appleResolveCommand("wails", fallbacks...)
	if fallbackErr != nil {
		return "", coreerr.E("build.resolveWails3Cli", "wails3 CLI not found. Install Wails v3 or expose it on PATH.", err)
	}
	return command, nil
}

func resolveDenoCli() (string, error) {
	command, err := appleResolveCommand("deno", "/usr/local/bin/deno", "/opt/homebrew/bin/deno")
	if err != nil {
		return "", coreerr.E("build.resolveDenoCli", "deno CLI not found. Install it from https://deno.com/runtime", err)
	}
	return command, nil
}

func resolveNpmCli() (string, error) {
	command, err := appleResolveCommand("npm", "/usr/local/bin/npm", "/opt/homebrew/bin/npm")
	if err != nil {
		return "", coreerr.E("build.resolveNpmCli", "npm CLI not found. Install Node.js from https://nodejs.org/", err)
	}
	return command, nil
}

func resolveBunCli() (string, error) {
	command, err := appleResolveCommand("bun", "/usr/local/bin/bun", "/opt/homebrew/bin/bun")
	if err != nil {
		return "", coreerr.E("build.resolveBunCli", "bun CLI not found. Install it from https://bun.sh/", err)
	}
	return command, nil
}

func resolvePnpmCli() (string, error) {
	command, err := appleResolveCommand("pnpm", "/usr/local/bin/pnpm", "/opt/homebrew/bin/pnpm")
	if err != nil {
		return "", coreerr.E("build.resolvePnpmCli", "pnpm CLI not found. Install it from https://pnpm.io/installation", err)
	}
	return command, nil
}

func resolveYarnCli() (string, error) {
	command, err := appleResolveCommand("yarn", "/usr/local/bin/yarn", "/opt/homebrew/bin/yarn")
	if err != nil {
		return "", coreerr.E("build.resolveYarnCli", "yarn CLI not found. Install it from https://yarnpkg.com/getting-started/install", err)
	}
	return command, nil
}

func resolveLipoCli() (string, error) {
	command, err := appleResolveCommand("lipo", "/usr/bin/lipo", "/usr/local/bin/lipo", "/opt/homebrew/bin/lipo")
	if err != nil {
		return "", coreerr.E("build.resolveLipoCli", "lipo not found. Install Xcode Command Line Tools.", err)
	}
	return command, nil
}

func resolveCodesignCli() (string, error) {
	command, err := appleResolveCommand("codesign", "/usr/bin/codesign", "/usr/local/bin/codesign", "/opt/homebrew/bin/codesign")
	if err != nil {
		return "", coreerr.E("build.resolveCodesignCli", "codesign not found. Install Xcode Command Line Tools.", err)
	}
	return command, nil
}

func resolveDittocli() (string, error) {
	command, err := appleResolveCommand("ditto", "/usr/bin/ditto", "/usr/local/bin/ditto", "/opt/homebrew/bin/ditto")
	if err != nil {
		return "", coreerr.E("build.resolveDittocli", "ditto not found. Install Xcode Command Line Tools.", err)
	}
	return command, nil
}

func resolveXcrunCli() (string, error) {
	command, err := appleResolveCommand("xcrun", "/usr/bin/xcrun", "/usr/local/bin/xcrun", "/opt/homebrew/bin/xcrun")
	if err != nil {
		return "", coreerr.E("build.resolveXcrunCli", "xcrun not found. Install Xcode Command Line Tools.", err)
	}
	return command, nil
}

func resolveSPCTLCli() (string, error) {
	command, err := appleResolveCommand("spctl", "/usr/sbin/spctl", "/usr/local/bin/spctl", "/opt/homebrew/bin/spctl")
	if err != nil {
		return "", coreerr.E("build.resolveSPCTLCli", "spctl not found on this system.", err)
	}
	return command, nil
}

func resolveHdiutilCli() (string, error) {
	command, err := appleResolveCommand("hdiutil", "/usr/bin/hdiutil", "/usr/local/bin/hdiutil", "/opt/homebrew/bin/hdiutil")
	if err != nil {
		return "", coreerr.E("build.resolveHdiutilCli", "hdiutil not found. macOS disk image tools are required.", err)
	}
	return command, nil
}

func resolveOsaScriptCli() (string, error) {
	command, err := appleResolveCommand("osascript", "/usr/bin/osascript", "/usr/local/bin/osascript", "/opt/homebrew/bin/osascript")
	if err != nil {
		return "", coreerr.E("build.resolveOsaScriptCli", "osascript not found. Finder automation is required for DMG layout.", err)
	}
	return command, nil
}

func resolveProductbuildCli() (string, error) {
	command, err := appleResolveCommand("productbuild", "/usr/bin/productbuild", "/usr/local/bin/productbuild", "/opt/homebrew/bin/productbuild")
	if err != nil {
		return "", coreerr.E("build.resolveProductbuildCli", "productbuild not found. Install Xcode Command Line Tools.", err)
	}
	return command, nil
}
