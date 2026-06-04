package build

import (
	"context"
	"encoding/xml"
	"io/fs"
	"net/url"
	"sort"
	"strconv"
	"syscall"
	"time"
	"unicode"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	storage "dappco.re/go/build/pkg/storage"
)

const (
	defaultAppleArch             = "universal"
	defaultAppleMinSystemVersion = "13.0"
	defaultAppleCategory         = "public.app-category.developer-tools"
	defaultDMGIconSize           = 128
	defaultDMGWindowWidth        = 640
	defaultDMGWindowHeight       = 480
	notaryToolLogCommand         = "lo" + "g"
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

func validateAppleBuildOptions(options AppleOptions) core.Result {
	if options.Sign && core.Trim(options.CertIdentity) == "" {
		return core.Fail(core.E("build.validateAppleBuildOptions", "signing identity is required when sign is enabled", nil))
	}

	if options.Notarise {
		authArgs := notariseAuthArgs(NotariseConfig{
			AppPath:        "",
			APIKeyID:       options.APIKeyID,
			APIKeyIssuerID: options.APIKeyIssuerID,
			APIKeyPath:     options.APIKeyPath,
			TeamID:         options.TeamID,
			AppleID:        options.AppleID,
			Password:       options.Password,
		})
		if !authArgs.OK {
			return core.Fail(core.E("build.validateAppleBuildOptions", "invalid notarisation credentials", core.NewError(authArgs.Error())))
		}
	}

	if options.TestFlight || options.AppStore {
		valid := validateAppStoreConnectAPIKey(options.APIKeyID, options.APIKeyIssuerID, options.APIKeyPath, "build.validateAppleBuildOptions")
		if !valid.OK {
			return valid
		}
		if core.Trim(options.ProfilePath) == "" {
			return core.Fail(core.E("build.validateAppleBuildOptions", "profile_path is required for App Store Connect uploads", nil))
		}
		if isDeveloperIDIdentity(options.CertIdentity) {
			return core.Fail(core.E("build.validateAppleBuildOptions", "TestFlight and App Store uploads require an Apple distribution certificate, not Developer ID", nil))
		}
	}

	if options.AppStore {
		minSystemVersion := firstNonEmpty(options.MinSystemVersion, defaultAppleMinSystemVersion)
		if compareAppleVersion(minSystemVersion, defaultAppleMinSystemVersion) < 0 {
			return core.Fail(core.E("build.validateAppleBuildOptions", "App Store submissions require min_system_version 13.0 or newer", nil))
		}

		if core.Trim(firstNonEmpty(options.Category, defaultAppleCategory)) == "" {
			return core.Fail(core.E("build.validateAppleBuildOptions", "App Store submissions require an application category", nil))
		}

		if !core.Contains(core.Lower(options.Copyright), "eupl-1.2") {
			return core.Fail(core.E("build.validateAppleBuildOptions", "App Store submissions must declare EUPL-1.2 in copyright metadata", nil))
		}

		valid := validatePrivacyPolicyURL(options.PrivacyPolicyURL)
		if !valid.OK {
			return valid
		}
	}

	return core.Ok(nil)
}

// BuildApple runs the end-to-end macOS Apple pipeline for a Wails app.
func BuildApple(ctx context.Context, cfg *Config, options AppleOptions, buildNumber string) core.Result {
	if cfg == nil {
		return core.Fail(core.E("build.BuildApple", "config is nil", nil))
	}
	if cfg.FS == nil {
		cfg.FS = storage.Local
	}

	if options.BundleID == "" {
		return core.Fail(core.E("build.BuildApple", "bundle_id is required for Apple builds", nil))
	}
	if options.Notarise && !options.Sign {
		return core.Fail(core.E("build.BuildApple", "notarisation requires code signing", nil))
	}
	if (options.TestFlight || options.AppStore) && !options.Sign {
		return core.Fail(core.E("build.BuildApple", "TestFlight and App Store uploads require code signing", nil))
	}
	valid := validateAppleBuildOptions(options)
	if !valid.OK {
		return valid
	}

	name := resolveAppleBundleName(cfg)
	outputDir := resolveAppleOutputDir(cfg)
	created := cfg.FS.EnsureDir(outputDir)
	if !created.OK {
		return core.Fail(core.E("build.BuildApple", "failed to create Apple output directory", core.NewError(created.Error())))
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
		arm64Temp := ax.TempDir("core-build-apple-arm64-*")
		if !arm64Temp.OK {
			return core.Fail(core.E("build.BuildApple", "failed to create arm64 temp directory", core.NewError(arm64Temp.Error())))
		}
		arm64Dir := arm64Temp.Value.(string)
		defer ax.RemoveAll(arm64Dir)

		amd64Temp := ax.TempDir("core-build-apple-amd64-*")
		if !amd64Temp.OK {
			return core.Fail(core.E("build.BuildApple", "failed to create amd64 temp directory", core.NewError(amd64Temp.Error())))
		}
		amd64Dir := amd64Temp.Value.(string)
		defer ax.RemoveAll(amd64Dir)

		arm64BundleResult := appleBuildWailsAppFn(ctx, WailsBuildConfig{
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
		if !arm64BundleResult.OK {
			return core.Fail(core.E("build.BuildApple", "failed to build arm64 bundle", core.NewError(arm64BundleResult.Error())))
		}
		arm64Bundle := arm64BundleResult.Value.(string)

		amd64BundleResult := appleBuildWailsAppFn(ctx, WailsBuildConfig{
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
		if !amd64BundleResult.OK {
			return core.Fail(core.E("build.BuildApple", "failed to build amd64 bundle", core.NewError(amd64BundleResult.Error())))
		}
		amd64Bundle := amd64BundleResult.Value.(string)

		bundlePath = ax.Join(outputDir, name+".app")
		createdUniversal := appleCreateUniversalFn(arm64Bundle, amd64Bundle, bundlePath)
		if !createdUniversal.OK {
			return core.Fail(core.E("build.BuildApple", "failed to create universal app bundle", core.NewError(createdUniversal.Error())))
		}
	case "arm64", "amd64":
		bundleResult := appleBuildWailsAppFn(ctx, WailsBuildConfig{
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
		if !bundleResult.OK {
			return core.Fail(core.E("build.BuildApple", "failed to build app bundle", core.NewError(bundleResult.Error())))
		}
		bundlePath = bundleResult.Value.(string)
	default:
		return core.Fail(core.E("build.BuildApple", "unsupported Apple arch: "+options.Arch, nil))
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

	infoPlistResult := WriteInfoPlist(cfg.FS, bundlePath, infoPlist)
	if !infoPlistResult.OK {
		return core.Fail(core.E("build.BuildApple", "failed to write Info.plist", core.NewError(infoPlistResult.Error())))
	}
	infoPlistPath := infoPlistResult.Value.(string)

	if options.ProfilePath != "" {
		copied := copyPath(cfg.FS, options.ProfilePath, ax.Join(bundlePath, "Contents", "embedded.provisionprofile"))
		if !copied.OK {
			return core.Fail(core.E("build.BuildApple", "failed to copy provisioning profile", core.NewError(copied.Error())))
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
	entitlementsResult := WriteEntitlements(cfg.FS, entitlementsPath, entitlements)
	if !entitlementsResult.OK {
		return core.Fail(core.E("build.BuildApple", "failed to write entitlements", core.NewError(entitlementsResult.Error())))
	}

	if options.Sign {
		signed := appleSignFn(ctx, SignConfig{
			AppPath:      bundlePath,
			Identity:     options.CertIdentity,
			Entitlements: entitlementsPath,
			Hardened:     true,
			Deep:         false,
			KeychainPath: options.KeychainPath,
		})
		if !signed.OK {
			return core.Fail(core.E("build.BuildApple", "failed to sign app bundle", core.NewError(signed.Error())))
		}
	}

	distributionPath := bundlePath
	dmgPath := ""
	if options.DMG {
		dmgPath = ax.Join(outputDir, core.Sprintf("%s-%s.dmg", name, normalizeAppleVersion(version)))
		createdDMG := appleCreateDMGFn(ctx, DMGConfig{
			AppPath:    bundlePath,
			OutputPath: dmgPath,
			VolumeName: firstNonEmpty(options.DMGVolumeName, name),
			Background: options.DMGBackground,
			IconSize:   128,
			WindowSize: [2]int{640, 480},
		})
		if !createdDMG.OK {
			return core.Fail(core.E("build.BuildApple", "failed to create DMG", core.NewError(createdDMG.Error())))
		}
		if options.Sign {
			signed := appleSignFn(ctx, SignConfig{
				AppPath:      dmgPath,
				Identity:     options.CertIdentity,
				Hardened:     false,
				Deep:         false,
				KeychainPath: options.KeychainPath,
			})
			if !signed.OK {
				return core.Fail(core.E("build.BuildApple", "failed to sign DMG", core.NewError(signed.Error())))
			}
		}
		distributionPath = dmgPath
	}

	if options.Notarise {
		notarised := appleNotariseFn(ctx, NotariseConfig{
			AppPath:        distributionPath,
			APIKeyID:       options.APIKeyID,
			APIKeyIssuerID: options.APIKeyIssuerID,
			APIKeyPath:     options.APIKeyPath,
			TeamID:         options.TeamID,
			AppleID:        options.AppleID,
			Password:       options.Password,
		})
		if !notarised.OK {
			return core.Fail(core.E("build.BuildApple", "failed to notarise distribution", core.NewError(notarised.Error())))
		}
	}

	if options.TestFlight {
		uploaded := appleUploadTestFlightFn(ctx, TestFlightConfig{
			AppPath:        bundlePath,
			APIKeyID:       options.APIKeyID,
			APIKeyIssuerID: options.APIKeyIssuerID,
			APIKeyPath:     options.APIKeyPath,
			CertIdentity:   options.CertIdentity,
		})
		if !uploaded.OK {
			return core.Fail(core.E("build.BuildApple", "failed to upload TestFlight build", core.NewError(uploaded.Error())))
		}
	}

	if options.AppStore {
		preflight := validateAppStorePreflight(cfg.FS, cfg.ProjectDir, bundlePath, options)
		if !preflight.OK {
			return preflight
		}

		submitted := appleSubmitAppStoreFn(ctx, AppStoreConfig{
			AppPath:        bundlePath,
			APIKeyID:       options.APIKeyID,
			APIKeyIssuerID: options.APIKeyIssuerID,
			APIKeyPath:     options.APIKeyPath,
			CertIdentity:   options.CertIdentity,
			Version:        normalizeAppleVersion(version),
			ReleaseType:    "manual",
		})
		if !submitted.OK {
			return core.Fail(core.E("build.BuildApple", "failed to submit App Store build", core.NewError(submitted.Error())))
		}
	}

	return core.Ok(&AppleBuildResult{
		BundlePath:       bundlePath,
		DMGPath:          dmgPath,
		DistributionPath: distributionPath,
		InfoPlistPath:    infoPlistPath,
		EntitlementsPath: entitlementsPath,
		BuildNumber:      buildNumber,
		Version:          normalizeAppleVersion(version),
	})
}

// BuildWailsApp builds a single-architecture Wails app bundle for macOS.
func BuildWailsApp(ctx context.Context, cfg WailsBuildConfig) core.Result {
	if cfg.ProjectDir == "" {
		return core.Fail(core.E("build.BuildWailsApp", "project directory is required", nil))
	}

	name := cfg.Name
	if name == "" {
		name = ax.Base(cfg.ProjectDir)
	}
	if cfg.Arch == "" {
		return core.Fail(core.E("build.BuildWailsApp", "arch is required", nil))
	}

	prepared := prepareWailsFrontend(ctx, cfg)
	if !prepared.OK {
		return prepared
	}

	wailsCommandResult := resolveWails3Cli()
	if !wailsCommandResult.OK {
		return wailsCommandResult
	}
	wailsCommand := wailsCommandResult.Value.(string)

	args := []string{"build", "-platform", "darwin/" + cfg.Arch}

	buildTags := deduplicateStrings(append(append([]string{}, cfg.BuildTags...), "mlx"))
	if len(buildTags) > 0 {
		args = append(args, "-tags", core.Join(",", buildTags...))
	}

	ldflags := append([]string{}, cfg.LDFlags...)
	if cfg.Version != "" && !appleHasVersionLDFlag(ldflags) {
		versionFlag := VersionLinkerFlag(cfg.Version)
		if !versionFlag.OK {
			return versionFlag
		}
		ldflags = append(ldflags, versionFlag.Value.(string))
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags", core.Join(" ", ldflags...))
	}

	env := append([]string{}, cfg.Env...)
	env = appendEnvIfMissing(env, "CGO_ENABLED", "1")

	output := appleCombinedOutput(ctx, cfg.ProjectDir, env, wailsCommand, args...)
	if !output.OK {
		return core.Fail(core.E("build.BuildWailsApp", "wails build failed: "+output.Error(), core.NewError(output.Error())))
	}

	sourcePathResult := findBuiltAppBundle(cfg.ProjectDir, name)
	if !sourcePathResult.OK {
		return sourcePathResult
	}
	sourcePath := sourcePathResult.Value.(string)

	if cfg.OutputDir == "" {
		return core.Ok(sourcePath)
	}

	created := storage.Local.EnsureDir(cfg.OutputDir)
	if !created.OK {
		return core.Fail(core.E("build.BuildWailsApp", "failed to create Wails output directory", core.NewError(created.Error())))
	}

	destPath := ax.Join(cfg.OutputDir, name+".app")
	if storage.Local.Exists(destPath) {
		deleted := storage.Local.DeleteAll(destPath)
		if !deleted.OK {
			return core.Fail(core.E("build.BuildWailsApp", "failed to replace existing app bundle", core.NewError(deleted.Error())))
		}
	}
	copied := copyPath(storage.Local, sourcePath, destPath)
	if !copied.OK {
		return core.Fail(core.E("build.BuildWailsApp", "failed to copy built app bundle", core.NewError(copied.Error())))
	}

	return core.Ok(destPath)
}

func prepareWailsFrontend(ctx context.Context, cfg WailsBuildConfig) core.Result {
	buildResult := resolveWailsFrontendBuild(cfg)
	if !buildResult.OK {
		return buildResult
	}
	frontendBuild := buildResult.Value.(wailsFrontendBuild)
	frontendDir := frontendBuild.dir
	command := frontendBuild.command
	args := frontendBuild.args
	if command == "" {
		return core.Ok(nil)
	}

	output := appleCombinedOutput(ctx, frontendDir, cfg.Env, command, args...)
	if !output.OK {
		return core.Fail(core.E("build.prepareWailsFrontend", command+" build failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

type wailsFrontendBuild struct {
	dir     string
	command string
	args    []string
}

func resolveWailsFrontendBuild(cfg WailsBuildConfig) core.Result {
	frontendDir := resolveFrontendDir(storage.Local, cfg.ProjectDir)
	if frontendDir == "" {
		if DenoRequested(cfg.DenoBuild) {
			frontendDir = cfg.ProjectDir
			if storage.Local.IsDir(ax.Join(cfg.ProjectDir, "frontend")) {
				frontendDir = ax.Join(cfg.ProjectDir, "frontend")
			}
		} else {
			return core.Ok(wailsFrontendBuild{})
		}
	}

	if hasDenoConfig(storage.Local, frontendDir) || DenoRequested(cfg.DenoBuild) {
		denoBuild := resolveDenoBuildCommand(cfg)
		if !denoBuild.OK {
			return denoBuild
		}
		resolved := denoBuild.Value.(commandArgs)
		return core.Ok(wailsFrontendBuild{dir: frontendDir, command: resolved.command, args: resolved.args})
	}

	if storage.Local.IsFile(ax.Join(frontendDir, "package.json")) {
		return resolvePackageManagerBuild(frontendDir, detectPackageManager(storage.Local, frontendDir))
	}

	return core.Ok(wailsFrontendBuild{})
}

func resolveFrontendDir(filesystem storage.Medium, projectDir string) string {
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

func hasDenoConfig(filesystem storage.Medium, dir string) bool {
	return filesystem.IsFile(ax.Join(dir, "deno.json")) || filesystem.IsFile(ax.Join(dir, "deno.jsonc"))
}

func resolveSubtreeFrontendDir(filesystem storage.Medium, projectDir string) string {
	return findFrontendDir(filesystem, projectDir, 0)
}

func findFrontendDir(filesystem storage.Medium, dir string, depth int) string {
	if depth >= 2 {
		return ""
	}

	entriesResult := filesystem.List(dir)
	if !entriesResult.OK {
		return ""
	}
	entries := entriesResult.Value.([]fs.DirEntry)

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

func resolvePackageManagerBuild(frontendDir, packageManager string) core.Result {
	switch packageManager {
	case "bun":
		command := resolveBunCli()
		if !command.OK {
			return command
		}
		return core.Ok(wailsFrontendBuild{dir: frontendDir, command: command.Value.(string), args: []string{"run", "build"}})
	case "pnpm":
		command := resolvePnpmCli()
		if !command.OK {
			return command
		}
		return core.Ok(wailsFrontendBuild{dir: frontendDir, command: command.Value.(string), args: []string{"run", "build"}})
	case "yarn":
		command := resolveYarnCli()
		if !command.OK {
			return command
		}
		return core.Ok(wailsFrontendBuild{dir: frontendDir, command: command.Value.(string), args: []string{"build"}})
	default:
		command := resolveNpmCli()
		if !command.OK {
			return command
		}
		return core.Ok(wailsFrontendBuild{dir: frontendDir, command: command.Value.(string), args: []string{"run", "build"}})
	}
}

func detectPackageManager(filesystem storage.Medium, dir string) string {
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

func detectDeclaredPackageManager(filesystem storage.Medium, dir string) string {
	content := filesystem.Read(ax.Join(dir, "package.json"))
	if !content.OK {
		return ""
	}

	var manifest packageJSONManifest
	decoded := ax.JSONUnmarshal([]byte(content.Value.(string)), &manifest)
	if !decoded.OK {
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

type commandArgs struct {
	command string
	args    []string
}

func resolveDenoBuildCommand(cfg WailsBuildConfig) core.Result {
	override := core.Trim(core.Env("DENO_BUILD"))
	if override == "" {
		override = core.Trim(cfg.DenoBuild)
	}
	if override != "" {
		argsResult := splitCommandLine(override)
		if !argsResult.OK {
			return core.Fail(core.E("build.resolveDenoBuildCommand", "invalid DENO_BUILD command", core.NewError(argsResult.Error())))
		}
		args := argsResult.Value.([]string)
		if len(args) == 0 {
			return core.Fail(core.E("build.resolveDenoBuildCommand", "DENO_BUILD command is empty", nil))
		}
		return core.Ok(commandArgs{command: args[0], args: args[1:]})
	}

	command := resolveDenoCli()
	if !command.OK {
		return command
	}
	return core.Ok(commandArgs{command: command.Value.(string), args: []string{"task", "build"}})
}

func splitCommandLine(command string) core.Result {
	command = core.Trim(command)
	if command == "" {
		return core.Ok([]string(nil))
	}

	var (
		args   []string
		quote  rune
		escape bool
	)
	current := core.NewBuilder()

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
		return core.Fail(core.E("build.splitCommandLine", "unterminated quote in command", nil))
	}

	flush()
	return core.Ok(args)
}

// CreateUniversal merges two architecture-specific app bundles into a universal app.
func CreateUniversal(arm64Path, amd64Path, outputPath string) core.Result {
	if arm64Path == "" || amd64Path == "" || outputPath == "" {
		return core.Fail(core.E("build.CreateUniversal", "arm64, amd64, and output paths are required", nil))
	}

	if storage.Local.Exists(outputPath) {
		deleted := storage.Local.DeleteAll(outputPath)
		if !deleted.OK {
			return core.Fail(core.E("build.CreateUniversal", "failed to replace existing output bundle", core.NewError(deleted.Error())))
		}
	}

	created := storage.Local.EnsureDir(ax.Dir(outputPath))
	if !created.OK {
		return core.Fail(core.E("build.CreateUniversal", "failed to create universal output directory", core.NewError(created.Error())))
	}
	copied := copyPath(storage.Local, arm64Path, outputPath)
	if !copied.OK {
		return core.Fail(core.E("build.CreateUniversal", "failed to copy arm64 bundle", core.NewError(copied.Error())))
	}

	lipoCommandResult := resolveLipoCli()
	if !lipoCommandResult.OK {
		return lipoCommandResult
	}
	lipoCommand := lipoCommandResult.Value.(string)

	for _, candidate := range universalMergeCandidates(storage.Local, arm64Path, amd64Path) {
		armCandidate := ax.Join(arm64Path, candidate)
		amdCandidate := ax.Join(amd64Path, candidate)
		outputCandidate := ax.Join(outputPath, candidate)
		output := appleCombinedOutput(context.Background(), "", nil, lipoCommand, "-create", "-output", outputCandidate, armCandidate, amdCandidate)
		if !output.OK {
			return core.Fail(core.E("build.CreateUniversal", "lipo failed for "+candidate+": "+output.Error(), core.NewError(output.Error())))
		}
	}

	return core.Ok(nil)
}

// Sign code-signs an app bundle or Apple artefact.
func Sign(ctx context.Context, cfg SignConfig) core.Result {
	if cfg.AppPath == "" {
		return core.Fail(core.E("build.Sign", "app_path is required", nil))
	}
	if cfg.Identity == "" {
		return core.Fail(core.E("build.Sign", "signing identity is required", nil))
	}

	codesignCommandResult := resolveCodesignCli()
	if !codesignCommandResult.OK {
		return codesignCommandResult
	}
	codesignCommand := codesignCommandResult.Value.(string)

	if !storage.Local.IsDir(cfg.AppPath) || !core.HasSuffix(cfg.AppPath, ".app") {
		output := appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, cfg.AppPath, cfg.Entitlements)...)
		if !output.OK {
			return core.Fail(core.E("build.Sign", "codesign failed for "+cfg.AppPath, core.NewError(output.Error())))
		}
		return core.Ok(nil)
	}

	for _, path := range signFrameworkPaths(cfg.AppPath) {
		output := appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, path, "")...)
		if !output.OK {
			return core.Fail(core.E("build.Sign", "codesign failed for framework "+path+": "+output.Error(), core.NewError(output.Error())))
		}
	}

	mainBinary := bundleExecutablePath(cfg.AppPath)
	for _, path := range signHelperBinaryPaths(cfg.AppPath, mainBinary) {
		output := appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, path, "")...)
		if !output.OK {
			return core.Fail(core.E("build.Sign", "codesign failed for helper binary "+path+": "+output.Error(), core.NewError(output.Error())))
		}
	}

	output := appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, mainBinary, cfg.Entitlements)...)
	if !output.OK {
		return core.Fail(core.E("build.Sign", "codesign failed for main binary "+mainBinary+": "+output.Error(), core.NewError(output.Error())))
	}

	output = appleCombinedOutput(ctx, "", nil, codesignCommand, codesignArgs(cfg, cfg.AppPath, cfg.Entitlements)...)
	if !output.OK {
		return core.Fail(core.E("build.Sign", "codesign failed for app bundle "+cfg.AppPath+": "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

// Notarise submits a signed app bundle or DMG to Apple and staples the ticket.
func Notarise(ctx context.Context, cfg NotariseConfig) core.Result {
	if cfg.AppPath == "" {
		return core.Fail(core.E("build.Notarise", "app_path is required", nil))
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

	authArgsResult := notariseAuthArgs(cfg)
	if !authArgsResult.OK {
		return authArgsResult
	}
	authArgs := authArgsResult.Value.([]string)

	dittoCommandResult := resolveDittocli()
	if !dittoCommandResult.OK {
		return dittoCommandResult
	}
	dittoCommand := dittoCommandResult.Value.(string)
	xcrunCommandResult := resolveXcrunCli()
	if !xcrunCommandResult.OK {
		return xcrunCommandResult
	}
	xcrunCommand := xcrunCommandResult.Value.(string)

	tempDirResult := ax.TempDir("core-build-notary-*")
	if !tempDirResult.OK {
		return core.Fail(core.E("build.Notarise", "failed to create notarisation temp directory", core.NewError(tempDirResult.Error())))
	}
	tempDir := tempDirResult.Value.(string)
	defer ax.RemoveAll(tempDir)

	zipPath := ax.Join(tempDir, ax.Base(cfg.AppPath)+".zip")
	output := appleCombinedOutput(notariseCtx, "", nil, dittoCommand, "-c", "-k", "--keepParent", cfg.AppPath, zipPath)
	if !output.OK {
		return core.Fail(core.E("build.Notarise", "failed to create notarisation archive: "+output.Error(), core.NewError(output.Error())))
	}

	submitArgs := []string{"notarytool", "submit", zipPath, "--wait", "--output-format", "json"}
	submitArgs = append(submitArgs, authArgs...)
	output = appleCombinedOutput(notariseCtx, "", nil, xcrunCommand, submitArgs...)
	outputText := ""
	if output.OK {
		outputText = output.Value.(string)
	}
	if !output.OK {
		outputText = appendNotaryLog(notariseCtx, xcrunCommand, authArgs, output.Error())
		return core.Fail(core.E("build.Notarise", "notarisation failed: "+outputText, core.NewError(output.Error())))
	}

	status := parseNotaryStatus(outputText)
	if status != "" && core.Lower(status) != "accepted" {
		outputText = appendNotaryLog(notariseCtx, xcrunCommand, authArgs, outputText)
		return core.Fail(core.E("build.Notarise", "Apple rejected notarisation request with status "+status+": "+outputText, nil))
	}

	output = appleCombinedOutput(notariseCtx, "", nil, xcrunCommand, "stapler", "staple", cfg.AppPath)
	if !output.OK {
		return core.Fail(core.E("build.Notarise", "failed to staple notarisation ticket: "+output.Error(), core.NewError(output.Error())))
	}

	if core.HasSuffix(cfg.AppPath, ".app") {
		spctlCommandResult := resolveSPCTLCli()
		if !spctlCommandResult.OK {
			return spctlCommandResult
		}
		spctlCommand := spctlCommandResult.Value.(string)
		output = appleCombinedOutput(notariseCtx, "", nil, spctlCommand, "--assess", "--type", "execute", cfg.AppPath)
		if !output.OK {
			return core.Fail(core.E("build.Notarise", "Gatekeeper assessment failed: "+output.Error(), core.NewError(output.Error())))
		}
	}

	return core.Ok(nil)
}

// CreateDMG packages an app bundle into a distributable DMG.
func CreateDMG(ctx context.Context, cfg DMGConfig) core.Result {
	if cfg.AppPath == "" || cfg.OutputPath == "" {
		return core.Fail(core.E("build.CreateDMG", "app_path and output_path are required", nil))
	}
	if ctx == nil {
		ctx = context.Background()
	}

	cfg = normaliseDMGConfig(cfg)

	tempDirResult := ax.TempDir("core-build-dmg-*")
	if !tempDirResult.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to create DMG staging directory", core.NewError(tempDirResult.Error())))
	}
	tempDir := tempDirResult.Value.(string)
	defer ax.RemoveAll(tempDir)

	stageDir := ax.Join(tempDir, "stage")
	mountDir := ax.Join(tempDir, "mount")
	rwDMGPath := ax.Join(tempDir, "staging.dmg")
	created := storage.Local.EnsureDir(stageDir)
	if !created.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to create DMG stage directory", core.NewError(created.Error())))
	}

	appName := ax.Base(cfg.AppPath)
	stageAppPath := ax.Join(stageDir, appName)
	copied := copyPath(storage.Local, cfg.AppPath, stageAppPath)
	if !copied.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to stage app bundle", core.NewError(copied.Error())))
	}

	if err := syscall.Symlink("/Applications", ax.Join(stageDir, "Applications")); err != nil {
		return core.Fail(core.E("build.CreateDMG", "failed to create Applications symlink", err))
	}

	if cfg.Background != "" {
		backgroundDir := ax.Join(stageDir, ".background")
		backgroundCreated := storage.Local.EnsureDir(backgroundDir)
		if !backgroundCreated.OK {
			return core.Fail(core.E("build.CreateDMG", "failed to create DMG background directory", core.NewError(backgroundCreated.Error())))
		}
		backgroundCopied := copyPath(storage.Local, cfg.Background, ax.Join(backgroundDir, ax.Base(cfg.Background)))
		if !backgroundCopied.OK {
			return core.Fail(core.E("build.CreateDMG", "failed to stage DMG background", core.NewError(backgroundCopied.Error())))
		}
	}

	outputCreated := storage.Local.EnsureDir(ax.Dir(cfg.OutputPath))
	if !outputCreated.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to create DMG output directory", core.NewError(outputCreated.Error())))
	}

	hdiutilCommandResult := resolveHdiutilCli()
	if !hdiutilCommandResult.OK {
		return hdiutilCommandResult
	}
	hdiutilCommand := hdiutilCommandResult.Value.(string)
	osascriptCommandResult := resolveOsaScriptCli()
	if !osascriptCommandResult.OK {
		return osascriptCommandResult
	}
	osascriptCommand := osascriptCommandResult.Value.(string)

	volumeName := firstNonEmpty(cfg.VolumeName, core.TrimSuffix(appName, ".app"))
	createArgs := []string{
		"create",
		"-volname", volumeName,
		"-srcfolder", stageDir,
		"-ov",
		"-format", "UDRW",
		rwDMGPath,
	}
	output := appleCombinedOutput(ctx, "", nil, hdiutilCommand, createArgs...)
	if !output.OK {
		return core.Fail(core.E("build.CreateDMG", "hdiutil failed: "+output.Error(), core.NewError(output.Error())))
	}

	mountCreated := storage.Local.EnsureDir(mountDir)
	if !mountCreated.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to create DMG mount directory", core.NewError(mountCreated.Error())))
	}

	attached := false
	defer func() {
		if attached {
			detachDMG(context.Background(), hdiutilCommand, mountDir)
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
	output = appleCombinedOutput(ctx, "", nil, hdiutilCommand, attachArgs...)
	if !output.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to mount staging DMG: "+output.Error(), core.NewError(output.Error())))
	}
	attached = true

	scriptPath := ax.Join(tempDir, "layout.applescript")
	script := buildDMGAppleScript(volumeName, appName, cfg)
	written := storage.Local.WriteMode(scriptPath, script, 0o644)
	if !written.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to write DMG layout script", core.NewError(written.Error())))
	}

	output = appleCombinedOutput(ctx, "", nil, osascriptCommand, scriptPath)
	if !output.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to configure Finder layout: "+output.Error(), core.NewError(output.Error())))
	}

	detached := detachDMG(ctx, hdiutilCommand, mountDir)
	if !detached.OK {
		return detached
	}
	attached = false

	convertArgs := []string{
		"convert",
		rwDMGPath,
		"-format", "UDZO",
		"-ov",
		"-o", cfg.OutputPath,
	}
	output = appleCombinedOutput(ctx, "", nil, hdiutilCommand, convertArgs...)
	if !output.OK {
		return core.Fail(core.E("build.CreateDMG", "failed to convert DMG: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
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
	return core.Replace(core.Replace(value, `\`, `\\`), `"`, `\"`)
}

func detachDMG(ctx context.Context, hdiutilCommand, mountDir string) core.Result {
	output := appleCombinedOutput(ctx, "", nil, hdiutilCommand, "detach", mountDir)
	if output.OK {
		return core.Ok(nil)
	}

	forceOutput := appleCombinedOutput(ctx, "", nil, hdiutilCommand, "detach", mountDir, "-force")
	if !forceOutput.OK {
		message := output.Error()
		if forceOutput.Error() != "" {
			message = core.Join("\n", output.Error(), forceOutput.Error())
		}
		return core.Fail(core.E("build.CreateDMG", "failed to detach staging DMG: "+message, core.NewError(forceOutput.Error())))
	}

	return core.Ok(nil)
}

// UploadTestFlight uploads a packaged macOS artefact to TestFlight.
func UploadTestFlight(ctx context.Context, cfg TestFlightConfig) core.Result {
	if cfg.AppPath == "" {
		return core.Fail(core.E("build.UploadTestFlight", "app_path is required", nil))
	}
	valid := validateAppStoreConnectAPIKey(cfg.APIKeyID, cfg.APIKeyIssuerID, cfg.APIKeyPath, "build.UploadTestFlight")
	if !valid.OK {
		return valid
	}

	uploadPackage := packageForASCUpload(ctx, cfg.AppPath, cfg.CertIdentity, cfg.APIKeyID, cfg.APIKeyPath)
	if !uploadPackage.OK {
		return uploadPackage
	}
	upload := uploadPackage.Value.(ascUploadPackage)
	uploadPath := upload.path
	env := upload.env
	cleanup := upload.cleanup
	defer cleanup()

	xcrunCommandResult := resolveXcrunCli()
	if !xcrunCommandResult.OK {
		return xcrunCommandResult
	}
	xcrunCommand := xcrunCommandResult.Value.(string)

	output := appleCombinedOutput(ctx, "", env, xcrunCommand,
		"altool", "--upload-app", "--type", "macos",
		"--file", uploadPath,
		"--apiKey", cfg.APIKeyID,
		"--apiIssuer", cfg.APIKeyIssuerID,
	)
	if !output.OK {
		return core.Fail(core.E("build.UploadTestFlight", "altool upload failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

// SubmitAppStore uploads a packaged macOS artefact for App Store Connect review.
func SubmitAppStore(ctx context.Context, cfg AppStoreConfig) core.Result {
	if cfg.ReleaseType != "" && cfg.ReleaseType != "manual" && cfg.ReleaseType != "automatic" {
		return core.Fail(core.E("build.SubmitAppStore", "release_type must be manual or automatic", nil))
	}
	if cfg.AppPath == "" {
		return core.Fail(core.E("build.SubmitAppStore", "app_path is required", nil))
	}
	valid := validateAppStoreConnectAPIKey(cfg.APIKeyID, cfg.APIKeyIssuerID, cfg.APIKeyPath, "build.SubmitAppStore")
	if !valid.OK {
		return valid
	}

	uploadPackage := packageForASCUpload(ctx, cfg.AppPath, cfg.CertIdentity, cfg.APIKeyID, cfg.APIKeyPath)
	if !uploadPackage.OK {
		return uploadPackage
	}
	upload := uploadPackage.Value.(ascUploadPackage)
	uploadPath := upload.path
	env := upload.env
	cleanup := upload.cleanup
	defer cleanup()

	xcrunCommandResult := resolveXcrunCli()
	if !xcrunCommandResult.OK {
		return xcrunCommandResult
	}
	xcrunCommand := xcrunCommandResult.Value.(string)

	output := appleCombinedOutput(ctx, "", env, xcrunCommand,
		"altool", "--upload-app", "--type", "macos",
		"--file", uploadPath,
		"--apiKey", cfg.APIKeyID,
		"--apiIssuer", cfg.APIKeyIssuerID,
	)
	if !output.OK {
		return core.Fail(core.E("build.SubmitAppStore", "altool upload failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

// WriteInfoPlist writes the app bundle Info.plist and returns its path.
func WriteInfoPlist(filesystem storage.Medium, appPath string, plist InfoPlist) core.Result {
	if filesystem == nil {
		filesystem = storage.Local
	}

	plistPath := ax.Join(appPath, "Contents", "Info.plist")
	created := filesystem.EnsureDir(ax.Dir(plistPath))
	if !created.OK {
		return core.Fail(core.E("build.WriteInfoPlist", "failed to create Info.plist directory", core.NewError(created.Error())))
	}

	content := encodePlist(plist.Values())
	if !content.OK {
		return content
	}
	written := filesystem.WriteMode(plistPath, content.Value.(string), 0o644)
	if !written.OK {
		return core.Fail(core.E("build.WriteInfoPlist", "failed to write Info.plist", core.NewError(written.Error())))
	}

	return core.Ok(plistPath)
}

// WriteEntitlements writes an entitlements plist file.
func WriteEntitlements(filesystem storage.Medium, path string, entitlements Entitlements) core.Result {
	if filesystem == nil {
		filesystem = storage.Local
	}
	if path == "" {
		return core.Fail(core.E("build.WriteEntitlements", "entitlements path is required", nil))
	}

	created := filesystem.EnsureDir(ax.Dir(path))
	if !created.OK {
		return core.Fail(core.E("build.WriteEntitlements", "failed to create entitlements directory", core.NewError(created.Error())))
	}

	content := encodePlist(entitlements.Values())
	if !content.OK {
		return content
	}
	written := filesystem.WriteMode(path, content.Value.(string), 0o644)
	if !written.OK {
		return core.Fail(core.E("build.WriteEntitlements", "failed to write entitlements", core.NewError(written.Error())))
	}

	return core.Ok(nil)
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

func findBuiltAppBundle(projectDir, name string) core.Result {
	for _, candidate := range []string{
		ax.Join(projectDir, "build", "bin", name+".app"),
		ax.Join(projectDir, "dist", name+".app"),
		ax.Join(projectDir, name+".app"),
	} {
		if storage.Local.Exists(candidate) {
			return core.Ok(candidate)
		}
	}
	return core.Fail(core.E("build.findBuiltAppBundle", "Wails build completed but no .app bundle was found for "+name, nil))
}

func bundleExecutablePath(appPath string) string {
	executableName := core.TrimSuffix(ax.Base(appPath), ".app")
	infoPlistPath := ax.Join(appPath, "Contents", "Info.plist")
	if content := storage.Local.Read(infoPlistPath); content.OK {
		if name := plistStringValue(content.Value.(string), "CFBundleExecutable"); name != "" {
			executableName = name
		}
	}
	return ax.Join(appPath, "Contents", "MacOS", executableName)
}

func universalMergeCandidates(filesystem storage.Medium, arm64Path, amd64Path string) []string {
	candidates := map[string]struct{}{}
	seedUniversalMergeCandidates(filesystem, arm64Path, amd64Path, "", candidates)

	paths := make([]string, 0, len(candidates))
	for path := range candidates {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func seedUniversalMergeCandidates(filesystem storage.Medium, arm64Path, amd64Path, relativePath string, candidates map[string]struct{}) {
	currentPath := arm64Path
	if relativePath != "" {
		currentPath = ax.Join(arm64Path, relativePath)
	}

	entriesResult := filesystem.List(currentPath)
	if !entriesResult.OK {
		return
	}
	entries := entriesResult.Value.([]fs.DirEntry)

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

func shouldMergeUniversalPath(filesystem storage.Medium, path, relativePath string) bool {
	info := filesystem.Stat(path)
	if info.OK && info.Value.(fs.FileInfo).Mode()&0o111 != 0 {
		return true
	}

	lowerRelativePath := core.Lower(relativePath)
	if core.HasSuffix(lowerRelativePath, ".dylib") || core.HasSuffix(lowerRelativePath, ".so") {
		return true
	}

	for currentDir := ax.Dir(relativePath); currentDir != "." && currentDir != "" && currentDir != string(core.PathSeparator); currentDir = ax.Dir(currentDir) {
		base := ax.Base(currentDir)
		if core.HasSuffix(base, ".framework") {
			return ax.Base(relativePath) == core.TrimSuffix(base, ".framework")
		}
	}

	return false
}

func plistStringValue(content, key string) string {
	pattern := core.Sprintf("<key>%s</key>", key)
	parts := core.SplitN(content, pattern, 2)
	if len(parts) != 2 {
		return ""
	}

	remainder := parts[1]
	startTag := "<string>"
	endTag := "</string>"
	startParts := core.SplitN(remainder, startTag, 2)
	if len(startParts) != 2 {
		return ""
	}
	endParts := core.SplitN(startParts[1], endTag, 2)
	if len(endParts) != 2 {
		return ""
	}
	return core.Trim(endParts[0])
}

func copyPath(filesystem storage.Medium, sourcePath, destPath string) core.Result {
	if filesystem == nil {
		filesystem = storage.Local
	}

	if filesystem.IsDir(sourcePath) {
		created := filesystem.EnsureDir(destPath)
		if !created.OK {
			return created
		}
		entriesResult := filesystem.List(sourcePath)
		if !entriesResult.OK {
			return entriesResult
		}
		entries := entriesResult.Value.([]fs.DirEntry)
		for _, entry := range entries {
			copied := copyPath(filesystem, ax.Join(sourcePath, entry.Name()), ax.Join(destPath, entry.Name()))
			if !copied.OK {
				return copied
			}
		}
		return core.Ok(nil)
	}

	infoResult := filesystem.Stat(sourcePath)
	if !infoResult.OK {
		return infoResult
	}
	info := infoResult.Value.(fs.FileInfo)
	content := filesystem.Read(sourcePath)
	if !content.OK {
		return content
	}
	return filesystem.WriteMode(destPath, content.Value.(string), info.Mode().Perm())
}

func signFrameworkPaths(appPath string) []string {
	frameworksDir := ax.Join(appPath, "Contents", "Frameworks")
	if !storage.Local.IsDir(frameworksDir) {
		return nil
	}

	entriesResult := storage.Local.List(frameworksDir)
	if !entriesResult.OK {
		return nil
	}
	entries := entriesResult.Value.([]fs.DirEntry)

	var paths []string
	for _, entry := range entries {
		paths = append(paths, ax.Join(frameworksDir, entry.Name()))
	}
	sort.Strings(paths)
	return paths
}

func signHelperBinaryPaths(appPath, mainBinary string) []string {
	macOSDir := ax.Join(appPath, "Contents", "MacOS")
	if !storage.Local.IsDir(macOSDir) {
		return nil
	}

	entriesResult := storage.Local.List(macOSDir)
	if !entriesResult.OK {
		return nil
	}
	entries := entriesResult.Value.([]fs.DirEntry)

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

func notariseAuthArgs(cfg NotariseConfig) core.Result {
	if cfg.APIKeyID != "" {
		if cfg.APIKeyIssuerID == "" || cfg.APIKeyPath == "" {
			return core.Fail(core.E("build.notariseAuthArgs", "api_key_issuer_id and api_key_path are required with api_key_id", nil))
		}
		return core.Ok([]string{
			"--key", cfg.APIKeyPath,
			"--key-id", cfg.APIKeyID,
			"--issuer", cfg.APIKeyIssuerID,
		})
	}

	if cfg.AppleID == "" || cfg.Password == "" || cfg.TeamID == "" {
		return core.Fail(core.E("build.notariseAuthArgs", "team_id, apple_id, and password are required when API key auth is not configured", nil))
	}

	return core.Ok([]string{
		"--apple-id", cfg.AppleID,
		"--password", cfg.Password,
		"--team-id", cfg.TeamID,
	})
}

func validateAppStoreConnectAPIKey(apiKeyID, apiKeyIssuerID, apiKeyPath, op string) core.Result {
	switch {
	case core.Trim(apiKeyID) == "":
		return core.Fail(core.E(op, "api_key_id is required for App Store Connect uploads", nil))
	case core.Trim(apiKeyIssuerID) == "":
		return core.Fail(core.E(op, "api_key_issuer_id is required for App Store Connect uploads", nil))
	case core.Trim(apiKeyPath) == "":
		return core.Fail(core.E(op, "api_key_path is required for App Store Connect uploads", nil))
	default:
		return core.Ok(nil)
	}
}

func isDeveloperIDIdentity(identity string) bool {
	return core.Contains(core.Lower(identity), "developer id")
}

func validateAppStorePreflight(filesystem storage.Medium, projectDir, bundlePath string, options AppleOptions) core.Result {
	if filesystem == nil {
		filesystem = storage.Local
	}

	metadata := validateAppStoreMetadata(filesystem, projectDir, options.MetadataPath)
	if !metadata.OK {
		return metadata
	}
	scanned := scanBundleForPrivateAPIUsage(filesystem, bundlePath)
	if !scanned.OK {
		return scanned
	}

	return core.Ok(nil)
}

func validateAppStoreMetadata(filesystem storage.Medium, projectDir, configuredPath string) core.Result {
	metadataPath := resolveAppStoreMetadataPath(filesystem, projectDir, configuredPath)
	if metadataPath == "" {
		return core.Fail(core.E("build.validateAppStoreMetadata", "App Store submissions require metadata_path or a standard metadata directory (.core/apple/appstore, .core/appstore, or appstore)", nil))
	}

	if !hasAppStoreDescription(filesystem, metadataPath) {
		return core.Fail(core.E("build.validateAppStoreMetadata", "App Store submissions require a description file in metadata_path", nil))
	}
	if !hasAppStoreScreenshots(filesystem, metadataPath) {
		return core.Fail(core.E("build.validateAppStoreMetadata", "App Store submissions require at least one screenshot in metadata_path/screenshots", nil))
	}

	return core.Ok(nil)
}

func resolveAppStoreMetadataPath(filesystem storage.Medium, projectDir, configuredPath string) string {
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

func hasAppStoreDescription(filesystem storage.Medium, metadataPath string) bool {
	for _, name := range []string{"description.txt", "description.md", "description.markdown"} {
		if filesystem.IsFile(ax.Join(metadataPath, name)) {
			return true
		}
	}
	return false
}

func hasAppStoreScreenshots(filesystem storage.Medium, metadataPath string) bool {
	screenshotsDir := ax.Join(metadataPath, "screenshots")
	if !filesystem.IsDir(screenshotsDir) {
		return false
	}

	entriesResult := filesystem.List(screenshotsDir)
	if !entriesResult.OK {
		return false
	}
	entries := entriesResult.Value.([]fs.DirEntry)

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

func validatePrivacyPolicyURL(raw string) core.Result {
	value := core.Trim(raw)
	if value == "" {
		return core.Fail(core.E("build.validatePrivacyPolicyURL", "App Store submissions require privacy_policy_url (for example https://lthn.ai/privacy)", nil))
	}

	normalised := value
	if !core.Contains(normalised, "://") {
		normalised = "https://" + normalised
	}

	parsed, err := url.Parse(normalised)
	if err != nil {
		return core.Fail(core.E("build.validatePrivacyPolicyURL", "privacy_policy_url must be a valid URL", err))
	}
	if core.Trim(parsed.Host) == "" || parsed.Path == "" || parsed.Path == "/" {
		return core.Fail(core.E("build.validatePrivacyPolicyURL", "privacy_policy_url must include a host and non-root path", nil))
	}

	return core.Ok(nil)
}

func scanBundleForPrivateAPIUsage(filesystem storage.Medium, bundlePath string) core.Result {
	if bundlePath == "" {
		return core.Fail(core.E("build.scanBundleForPrivateAPIUsage", "bundle path is required", nil))
	}

	for _, root := range privateAPIScanRoots(bundlePath) {
		for _, path := range collectBundleFiles(filesystem, root) {
			content := filesystem.Read(path)
			if !content.OK {
				continue
			}
			if indicator := detectPrivateAPIIndicator(content.Value.(string)); indicator != "" {
				return core.Fail(core.E("build.scanBundleForPrivateAPIUsage", "private API usage detected in "+path+": "+indicator, nil))
			}
		}
	}

	return core.Ok(nil)
}

func privateAPIScanRoots(bundlePath string) []string {
	return []string{
		ax.Join(bundlePath, "Contents", "MacOS"),
		ax.Join(bundlePath, "Contents", "Frameworks"),
	}
}

func collectBundleFiles(filesystem storage.Medium, root string) []string {
	if filesystem == nil || !filesystem.Exists(root) {
		return nil
	}
	if !filesystem.IsDir(root) {
		return []string{root}
	}

	entriesResult := filesystem.List(root)
	if !entriesResult.OK {
		return nil
	}
	entries := entriesResult.Value.([]fs.DirEntry)

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
		if core.Contains(content, indicator) {
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
		part := core.Trim(rawPart)
		if part == "" {
			parts = append(parts, 0)
			continue
		}

		digits := core.NewBuilder()
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
	if decoded := core.JSONUnmarshal([]byte(output), &payload); decoded.OK {
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
	if decoded := core.JSONUnmarshal([]byte(output), &payload); decoded.OK {
		return payload.Status
	}
	return ""
}

func appendNotaryLog(ctx context.Context, xcrunCommand string, authArgs []string, output string) string {
	requestID := extractNotaryRequestID(output)
	if requestID == "" {
		return output
	}

	logArgs := []string{"notarytool", notaryToolLogCommand, requestID}
	logArgs = append(logArgs, authArgs...)
	logOutput := appleCombinedOutput(ctx, "", nil, xcrunCommand, logArgs...)
	if !logOutput.OK || logOutput.Value.(string) == "" {
		return output
	}

	return core.Join("\n", output, logOutput.Value.(string))
}

type ascUploadPackage struct {
	path    string
	env     []string
	cleanup func()
}

func packageForASCUpload(ctx context.Context, appPath, certIdentity, apiKeyID, apiKeyPath string) core.Result {
	if core.HasSuffix(appPath, ".pkg") {
		envResult := prepareASCAPIKeyEnv(apiKeyID, apiKeyPath)
		if !envResult.OK {
			return envResult
		}
		env := envResult.Value.(ascAPIKeyEnv)
		return core.Ok(ascUploadPackage{path: appPath, env: env.env, cleanup: env.cleanup})
	}

	if !core.HasSuffix(appPath, ".app") {
		return core.Fail(core.E("build.packageForASCUpload", "App Store Connect uploads require a .app or .pkg input", nil))
	}

	outputPath := ax.Join(ax.Dir(appPath), core.TrimSuffix(ax.Base(appPath), ".app")+".pkg")
	created := createDistributionPackage(ctx, appPath, certIdentity, outputPath)
	if !created.OK {
		return created
	}

	envResult := prepareASCAPIKeyEnv(apiKeyID, apiKeyPath)
	if !envResult.OK {
		return envResult
	}
	env := envResult.Value.(ascAPIKeyEnv)

	return core.Ok(ascUploadPackage{path: outputPath, env: env.env, cleanup: env.cleanup})
}

type ascAPIKeyEnv struct {
	env     []string
	cleanup func()
}

func prepareASCAPIKeyEnv(apiKeyID, apiKeyPath string) core.Result {
	if apiKeyPath == "" {
		return core.Ok(ascAPIKeyEnv{cleanup: func() {}})
	}

	expectedName := core.Sprintf("AuthKey_%s.p8", apiKeyID)
	if expectedName == "AuthKey_.p8" || ax.Base(apiKeyPath) == expectedName {
		return core.Ok(ascAPIKeyEnv{env: []string{"API_PRIVATE_KEYS_DIR=" + ax.Dir(apiKeyPath)}, cleanup: func() {}})
	}

	content := storage.Local.Read(apiKeyPath)
	if !content.OK {
		return core.Fail(core.E("build.prepareASCAPIKeyEnv", "failed to read App Store Connect API key", core.NewError(content.Error())))
	}

	tempDirResult := ax.TempDir("core-build-asc-key-*")
	if !tempDirResult.OK {
		return core.Fail(core.E("build.prepareASCAPIKeyEnv", "failed to create App Store Connect key staging directory", core.NewError(tempDirResult.Error())))
	}
	tempDir := tempDirResult.Value.(string)

	stagedPath := ax.Join(tempDir, expectedName)
	written := storage.Local.WriteMode(stagedPath, content.Value.(string), 0o600)
	if !written.OK {
		cleaned := ax.RemoveAll(tempDir)
		if !cleaned.OK {
			return core.Fail(core.E("build.prepareASCAPIKeyEnv", "failed to clean up App Store Connect key staging directory", core.NewError(cleaned.Error())))
		}
		return core.Fail(core.E("build.prepareASCAPIKeyEnv", "failed to stage App Store Connect API key", core.NewError(written.Error())))
	}

	return core.Ok(ascAPIKeyEnv{
		env: []string{"API_PRIVATE_KEYS_DIR=" + tempDir},
		cleanup: func() {
			ax.RemoveAll(tempDir)
		},
	})
}

func createDistributionPackage(ctx context.Context, appPath, certIdentity, outputPath string) core.Result {
	productbuildCommandResult := resolveProductbuildCli()
	if !productbuildCommandResult.OK {
		return productbuildCommandResult
	}
	productbuildCommand := productbuildCommandResult.Value.(string)

	args := []string{"--component", appPath, "/Applications", outputPath}
	if certIdentity != "" {
		args = append([]string{"--sign", certIdentity}, args...)
	}

	output := appleCombinedOutput(ctx, "", nil, productbuildCommand, args...)
	if !output.OK {
		return core.Fail(core.E("build.createDistributionPackage", "productbuild failed: "+output.Error(), core.NewError(output.Error())))
	}

	return core.Ok(nil)
}

func encodePlist(values map[string]any) core.Result {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	buf := core.NewBuffer()
	buf.WriteString(xml.Header)
	buf.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`)
	buf.WriteString(`<plist version="1.0"><dict>`)

	for _, key := range keys {
		buf.WriteString("<key>")
		if err := xml.EscapeText(buf, []byte(key)); err != nil {
			return core.Fail(core.E("build.encodePlist", "failed to encode plist key", err))
		}
		buf.WriteString("</key>")

		switch value := values[key].(type) {
		case string:
			buf.WriteString("<string>")
			if err := xml.EscapeText(buf, []byte(value)); err != nil {
				return core.Fail(core.E("build.encodePlist", "failed to encode plist string value", err))
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
			return core.Fail(core.E("build.encodePlist", "unsupported plist value type", nil))
		}
	}

	buf.WriteString("</dict></plist>")
	return core.Ok(buf.String())
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

func resolveWails3Cli() core.Result {
	paths := []string{
		"/usr/local/bin/wails3",
		"/opt/homebrew/bin/wails3",
	}
	if home := core.Env("HOME"); home != "" {
		paths = append(paths, ax.Join(home, "go", "bin", "wails3"))
	}
	command := appleResolveCommand("wails3", paths...)
	if command.OK {
		return command
	}

	fallbacks := []string{
		"/usr/local/bin/wails",
		"/opt/homebrew/bin/wails",
	}
	if home := core.Env("HOME"); home != "" {
		fallbacks = append(fallbacks, ax.Join(home, "go", "bin", "wails"))
	}
	fallback := appleResolveCommand("wails", fallbacks...)
	if !fallback.OK {
		return core.Fail(core.E("build.resolveWails3Cli", "wails3 CLI not found. Install Wails v3 or expose it on PATH.", core.NewError(command.Error())))
	}
	return fallback
}

func resolveDenoCli() core.Result {
	command := appleResolveCommand("deno", "/usr/local/bin/deno", "/opt/homebrew/bin/deno")
	if !command.OK {
		return core.Fail(core.E("build.resolveDenoCli", "deno CLI not found. Install it from https://deno.com/runtime", core.NewError(command.Error())))
	}
	return command
}

func resolveNpmCli() core.Result {
	command := appleResolveCommand("npm", "/usr/local/bin/npm", "/opt/homebrew/bin/npm")
	if !command.OK {
		return core.Fail(core.E("build.resolveNpmCli", "npm CLI not found. Install Node.js from https://nodejs.org/", core.NewError(command.Error())))
	}
	return command
}

func resolveBunCli() core.Result {
	command := appleResolveCommand("bun", "/usr/local/bin/bun", "/opt/homebrew/bin/bun")
	if !command.OK {
		return core.Fail(core.E("build.resolveBunCli", "bun CLI not found. Install it from https://bun.sh/", core.NewError(command.Error())))
	}
	return command
}

func resolvePnpmCli() core.Result {
	command := appleResolveCommand("pnpm", "/usr/local/bin/pnpm", "/opt/homebrew/bin/pnpm")
	if !command.OK {
		return core.Fail(core.E("build.resolvePnpmCli", "pnpm CLI not found. Install it from https://pnpm.io/installation", core.NewError(command.Error())))
	}
	return command
}

func resolveYarnCli() core.Result {
	command := appleResolveCommand("yarn", "/usr/local/bin/yarn", "/opt/homebrew/bin/yarn")
	if !command.OK {
		return core.Fail(core.E("build.resolveYarnCli", "yarn CLI not found. Install it from https://yarnpkg.com/getting-started/install", core.NewError(command.Error())))
	}
	return command
}

func resolveLipoCli() core.Result {
	command := appleResolveCommand("lipo", "/usr/bin/lipo", "/usr/local/bin/lipo", "/opt/homebrew/bin/lipo")
	if !command.OK {
		return core.Fail(core.E("build.resolveLipoCli", "lipo not found. Install Xcode Command Line Tools.", core.NewError(command.Error())))
	}
	return command
}

func resolveCodesignCli() core.Result {
	command := appleResolveCommand("codesign", "/usr/bin/codesign", "/usr/local/bin/codesign", "/opt/homebrew/bin/codesign")
	if !command.OK {
		return core.Fail(core.E("build.resolveCodesignCli", "codesign not found. Install Xcode Command Line Tools.", core.NewError(command.Error())))
	}
	return command
}

func resolveDittocli() core.Result {
	command := appleResolveCommand("ditto", "/usr/bin/ditto", "/usr/local/bin/ditto", "/opt/homebrew/bin/ditto")
	if !command.OK {
		return core.Fail(core.E("build.resolveDittocli", "ditto not found. Install Xcode Command Line Tools.", core.NewError(command.Error())))
	}
	return command
}

func resolveXcrunCli() core.Result {
	command := appleResolveCommand("xcrun", "/usr/bin/xcrun", "/usr/local/bin/xcrun", "/opt/homebrew/bin/xcrun")
	if !command.OK {
		return core.Fail(core.E("build.resolveXcrunCli", "xcrun not found. Install Xcode Command Line Tools.", core.NewError(command.Error())))
	}
	return command
}

func resolveSPCTLCli() core.Result {
	command := appleResolveCommand("spctl", "/usr/sbin/spctl", "/usr/local/bin/spctl", "/opt/homebrew/bin/spctl")
	if !command.OK {
		return core.Fail(core.E("build.resolveSPCTLCli", "spctl not found on this system.", core.NewError(command.Error())))
	}
	return command
}

func resolveHdiutilCli() core.Result {
	command := appleResolveCommand("hdiutil", "/usr/bin/hdiutil", "/usr/local/bin/hdiutil", "/opt/homebrew/bin/hdiutil")
	if !command.OK {
		return core.Fail(core.E("build.resolveHdiutilCli", "hdiutil not found. macOS disk image tools are required.", core.NewError(command.Error())))
	}
	return command
}

func resolveOsaScriptCli() core.Result {
	command := appleResolveCommand("osascript", "/usr/bin/osascript", "/usr/local/bin/osascript", "/opt/homebrew/bin/osascript")
	if !command.OK {
		return core.Fail(core.E("build.resolveOsaScriptCli", "osascript not found. Finder automation is required for DMG layout.", core.NewError(command.Error())))
	}
	return command
}

func resolveProductbuildCli() core.Result {
	command := appleResolveCommand("productbuild", "/usr/bin/productbuild", "/usr/local/bin/productbuild", "/opt/homebrew/bin/productbuild")
	if !command.OK {
		return core.Fail(core.E("build.resolveProductbuildCli", "productbuild not found. Install Xcode Command Line Tools.", core.NewError(command.Error())))
	}
	return command
}
