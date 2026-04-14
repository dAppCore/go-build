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

	"dappco.re/go/core"
	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

const (
	defaultAppleArch             = "universal"
	defaultAppleMinSystemVersion = "13.0"
	defaultAppleCategory         = "public.app-category.developer-tools"
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
	}

	if options.AppStore {
		if isDeveloperIDIdentity(options.CertIdentity) {
			return coreerr.E("build.validateAppleBuildOptions", "App Store submissions require an Apple distribution certificate, not Developer ID", nil)
		}

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

	output, err := appleCombinedOutput(ctx, cfg.ProjectDir, cfg.Env, wailsCommand, args...)
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

	armBinary := bundleExecutablePath(arm64Path)
	amdBinary := bundleExecutablePath(amd64Path)
	outputBinary := bundleExecutablePath(outputPath)

	lipoCommand, err := resolveLipoCli()
	if err != nil {
		return err
	}

	output, err := appleCombinedOutput(context.Background(), "", nil, lipoCommand, "-create", "-output", outputBinary, armBinary, amdBinary)
	if err != nil {
		return coreerr.E("build.CreateUniversal", "lipo failed: "+output, err)
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
	output, err := appleCombinedOutput(ctx, "", nil, dittoCommand, "-c", "-k", "--keepParent", cfg.AppPath, zipPath)
	if err != nil {
		return coreerr.E("build.Notarise", "failed to create notarisation archive: "+output, err)
	}

	submitArgs := []string{"notarytool", "submit", zipPath, "--wait", "--output-format", "json"}
	submitArgs = append(submitArgs, authArgs...)
	output, err = appleCombinedOutput(ctx, "", nil, xcrunCommand, submitArgs...)
	if err != nil {
		requestID := extractNotaryRequestID(output)
		if requestID != "" {
			logArgs := []string{"notarytool", "log", requestID}
			logArgs = append(logArgs, authArgs...)
			if logOutput, logErr := appleCombinedOutput(ctx, "", nil, xcrunCommand, logArgs...); logErr == nil && logOutput != "" {
				output = core.Join("\n", output, logOutput)
			}
		}
		return coreerr.E("build.Notarise", "notarisation failed: "+output, err)
	}

	status := parseNotaryStatus(output)
	if status != "" && core.Lower(status) != "accepted" {
		return coreerr.E("build.Notarise", "Apple rejected notarisation request with status "+status, nil)
	}

	output, err = appleCombinedOutput(ctx, "", nil, xcrunCommand, "stapler", "staple", cfg.AppPath)
	if err != nil {
		return coreerr.E("build.Notarise", "failed to staple notarisation ticket: "+output, err)
	}

	if core.HasSuffix(cfg.AppPath, ".app") {
		spctlCommand, err := resolveSPCTLCli()
		if err != nil {
			return err
		}
		output, err = appleCombinedOutput(ctx, "", nil, spctlCommand, "--assess", "--type", "execute", cfg.AppPath)
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

	stageDir, err := ax.TempDir("core-build-dmg-*")
	if err != nil {
		return coreerr.E("build.CreateDMG", "failed to create DMG staging directory", err)
	}
	defer func() { _ = ax.RemoveAll(stageDir) }()

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

	args := []string{
		"create",
		"-volname", firstNonEmpty(cfg.VolumeName, core.TrimSuffix(appName, ".app")),
		"-srcfolder", stageDir,
		"-ov",
		"-format", "UDZO",
		cfg.OutputPath,
	}
	output, err := appleCombinedOutput(ctx, "", nil, hdiutilCommand, args...)
	if err != nil {
		return coreerr.E("build.CreateDMG", "hdiutil failed: "+output, err)
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

	uploadPath, env, err := packageForASCUpload(ctx, cfg.AppPath, cfg.CertIdentity, cfg.APIKeyPath)
	if err != nil {
		return err
	}

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

	uploadPath, env, err := packageForASCUpload(ctx, cfg.AppPath, cfg.CertIdentity, cfg.APIKeyPath)
	if err != nil {
		return err
	}

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

func packageForASCUpload(ctx context.Context, appPath, certIdentity, apiKeyPath string) (string, []string, error) {
	if core.HasSuffix(appPath, ".pkg") {
		return appPath, ascAPIKeyEnv(apiKeyPath), nil
	}

	if !core.HasSuffix(appPath, ".app") {
		return "", nil, coreerr.E("build.packageForASCUpload", "App Store Connect uploads require a .app or .pkg input", nil)
	}

	outputPath := ax.Join(ax.Dir(appPath), core.TrimSuffix(ax.Base(appPath), ".app")+".pkg")
	if err := createDistributionPackage(ctx, appPath, certIdentity, outputPath); err != nil {
		return "", nil, err
	}

	return outputPath, ascAPIKeyEnv(apiKeyPath), nil
}

func ascAPIKeyEnv(apiKeyPath string) []string {
	if apiKeyPath == "" {
		return nil
	}
	return []string{"API_PRIVATE_KEYS_DIR=" + ax.Dir(apiKeyPath)}
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

func resolveProductbuildCli() (string, error) {
	command, err := appleResolveCommand("productbuild", "/usr/bin/productbuild", "/usr/local/bin/productbuild", "/opt/homebrew/bin/productbuild")
	if err != nil {
		return "", coreerr.E("build.resolveProductbuildCli", "productbuild not found. Install Xcode Command Line Tools.", err)
	}
	return command, nil
}
