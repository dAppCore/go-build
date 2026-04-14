package build

import (
	"context"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApple_WriteInfoPlist_Good(t *testing.T) {
	appPath := ax.Join(t.TempDir(), "Core.app")

	path, err := WriteInfoPlist(io.Local, appPath, InfoPlist{
		BundleID:                      "ai.lthn.core",
		BundleName:                    "Core",
		BundleDisplayName:             "Core by Lethean",
		BundleVersion:                 "1.2.3",
		BuildNumber:                   "42",
		MinSystemVersion:              "13.0",
		Category:                      "public.app-category.developer-tools",
		Copyright:                     "Copyright 2026 Lethean CIC. EUPL-1.2.",
		Executable:                    "Core",
		HighResCapable:                true,
		SupportsSecureRestorableState: true,
	})
	require.NoError(t, err)

	content, err := io.Local.Read(path)
	require.NoError(t, err)
	assert.Contains(t, content, "<key>CFBundleIdentifier</key>")
	assert.Contains(t, content, "<string>ai.lthn.core</string>")
	assert.Contains(t, content, "<key>CFBundleExecutable</key>")
	assert.Contains(t, content, "<string>Core</string>")
}

func TestApple_CreateUniversal_Good(t *testing.T) {
	dir := t.TempDir()
	arm64Path := ax.Join(dir, "arm64", "Core.app")
	amd64Path := ax.Join(dir, "amd64", "Core.app")
	outputPath := ax.Join(dir, "universal", "Core.app")

	writeDummyAppBundle(t, arm64Path, "Core", "arm64")
	writeDummyAppBundle(t, amd64Path, "Core", "amd64")

	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		require.Equal(t, "lipo", command)
		require.Equal(t, []string{"-create", "-output", ax.Join(outputPath, "Contents", "MacOS", "Core"), ax.Join(arm64Path, "Contents", "MacOS", "Core"), ax.Join(amd64Path, "Contents", "MacOS", "Core")}, args)
		require.NoError(t, ax.WriteFile(ax.Join(outputPath, "Contents", "MacOS", "Core"), []byte("universal"), 0o755))
		return "", nil
	}

	err := CreateUniversal(arm64Path, amd64Path, outputPath)
	require.NoError(t, err)

	content, err := ax.ReadFile(ax.Join(outputPath, "Contents", "MacOS", "Core"))
	require.NoError(t, err)
	assert.Equal(t, "universal", string(content))
}

func TestApple_BuildWailsApp_AddsMLXBuildTag_Good(t *testing.T) {
	projectDir := t.TempDir()
	bundlePath := ax.Join(projectDir, "build", "bin", "Core.app")
	writeDummyAppBundle(t, bundlePath, "Core", "built")

	oldResolve := appleResolveCommand
	oldCombined := appleCombinedOutput
	t.Cleanup(func() {
		appleResolveCommand = oldResolve
		appleCombinedOutput = oldCombined
	})

	appleResolveCommand = func(name string, fallbackPaths ...string) (string, error) {
		return name, nil
	}
	appleCombinedOutput = func(ctx context.Context, dir string, env []string, command string, args ...string) (string, error) {
		require.Equal(t, "wails3", command)
		assert.Contains(t, args, "-tags")
		tagIndex := -1
		for i, arg := range args {
			if arg == "-tags" {
				tagIndex = i + 1
				break
			}
		}
		require.GreaterOrEqual(t, tagIndex, 1)
		assert.Equal(t, "integration,mlx", args[tagIndex])
		return "", nil
	}

	result, err := BuildWailsApp(context.Background(), WailsBuildConfig{
		ProjectDir: projectDir,
		Name:       "Core",
		Arch:       "arm64",
		BuildTags:  []string{"integration"},
	})
	require.NoError(t, err)
	assert.Equal(t, bundlePath, result)
}

func TestApple_BuildApple_Good(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist", "apple")

	oldBuildWails := appleBuildWailsAppFn
	oldUniversal := appleCreateUniversalFn
	oldSign := appleSignFn
	oldNotarise := appleNotariseFn
	oldDMG := appleCreateDMGFn
	t.Cleanup(func() {
		appleBuildWailsAppFn = oldBuildWails
		appleCreateUniversalFn = oldUniversal
		appleSignFn = oldSign
		appleNotariseFn = oldNotarise
		appleCreateDMGFn = oldDMG
	})

	var builtArches []string
	var buildEnvs [][]string
	appleBuildWailsAppFn = func(ctx context.Context, cfg WailsBuildConfig) (string, error) {
		builtArches = append(builtArches, cfg.Arch)
		buildEnvs = append(buildEnvs, append([]string{}, cfg.Env...))
		appPath := ax.Join(cfg.OutputDir, cfg.Name+".app")
		writeDummyAppBundle(t, appPath, cfg.Name, cfg.Arch)
		return appPath, nil
	}
	appleCreateUniversalFn = func(arm64Path, amd64Path, outputPath string) error {
		require.NoError(t, copyPath(io.Local, arm64Path, outputPath))
		return ax.WriteFile(ax.Join(outputPath, "Contents", "MacOS", "Core"), []byte("universal"), 0o755)
	}

	var signCalls []SignConfig
	appleSignFn = func(ctx context.Context, cfg SignConfig) error {
		signCalls = append(signCalls, cfg)
		return nil
	}

	var notarisedPath string
	appleNotariseFn = func(ctx context.Context, cfg NotariseConfig) error {
		notarisedPath = cfg.AppPath
		return nil
	}

	var dmgCall DMGConfig
	appleCreateDMGFn = func(ctx context.Context, cfg DMGConfig) error {
		dmgCall = cfg
		return ax.WriteFile(cfg.OutputPath, []byte("dmg"), 0o644)
	}

	result, err := BuildApple(context.Background(), &Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "Core",
		Version:    "v1.2.3",
		BuildTags:  []string{"integration"},
		LDFlags:    []string{"-s", "-w"},
		Cache: CacheConfig{
			Enabled: true,
			Paths: []string{
				ax.Join(outputDir, "cache", "go-build"),
				ax.Join(outputDir, "cache", "go-mod"),
			},
		},
	}, AppleOptions{
		BundleID:     "ai.lthn.core",
		Arch:         "universal",
		Sign:         true,
		Notarise:     true,
		DMG:          true,
		CertIdentity: "Developer ID Application: Lethean CIC (ABC123DEF4)",
		TeamID:       "ABC123DEF4",
		AppleID:      "dev@example.com",
		Password:     "app-password",
	}, "42")
	require.NoError(t, err)

	assert.Equal(t, []string{"arm64", "amd64"}, builtArches)
	require.Len(t, buildEnvs, 2)
	assert.Contains(t, buildEnvs[0], "GOCACHE="+ax.Join(outputDir, "cache", "go-build"))
	assert.Contains(t, buildEnvs[0], "GOMODCACHE="+ax.Join(outputDir, "cache", "go-mod"))
	assert.Equal(t, ax.Join(outputDir, "Core.app"), result.BundlePath)
	assert.Equal(t, ax.Join(outputDir, "Core-1.2.3.dmg"), result.DMGPath)
	assert.Equal(t, result.DMGPath, notarisedPath)
	require.Len(t, signCalls, 2)
	assert.Equal(t, result.BundlePath, signCalls[0].AppPath)
	assert.Equal(t, result.EntitlementsPath, signCalls[0].Entitlements)
	assert.Equal(t, result.DMGPath, signCalls[1].AppPath)
	assert.Empty(t, signCalls[1].Entitlements)
	assert.False(t, signCalls[1].Hardened)
	assert.Equal(t, result.DMGPath, dmgCall.OutputPath)

	plistContent, err := io.Local.Read(result.InfoPlistPath)
	require.NoError(t, err)
	assert.Contains(t, plistContent, "<string>ai.lthn.core</string>")
	assert.Contains(t, plistContent, "<string>42</string>")

	entitlementsContent, err := io.Local.Read(result.EntitlementsPath)
	require.NoError(t, err)
	assert.Contains(t, entitlementsContent, "<key>com.apple.security.app-sandbox</key>")
	assert.Contains(t, entitlementsContent, "<false/>")
}

func TestApple_NotariseAuthArgs_Good(t *testing.T) {
	args, err := notariseAuthArgs(NotariseConfig{
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
		APIKeyPath:     "/tmp/AuthKey_KEY123.p8",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"--key", "/tmp/AuthKey_KEY123.p8", "--key-id", "KEY123", "--issuer", "ISSUER456"}, args)

	args, err = notariseAuthArgs(NotariseConfig{
		TeamID:   "ABC123DEF4",
		AppleID:  "dev@example.com",
		Password: "app-password",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"--apple-id", "dev@example.com", "--password", "app-password", "--team-id", "ABC123DEF4"}, args)
}

func TestApple_BuildApple_AppStorePreflight_Bad(t *testing.T) {
	_, err := BuildApple(context.Background(), &Config{
		FS:         io.Local,
		ProjectDir: t.TempDir(),
		OutputDir:  ax.Join(t.TempDir(), "dist", "apple"),
		Name:       "Core",
		Version:    "v1.2.3",
	}, AppleOptions{
		BundleID:       "ai.lthn.core",
		Arch:           "arm64",
		Sign:           true,
		AppStore:       true,
		CertIdentity:   "Developer ID Application: Lethean CIC (ABC123DEF4)",
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
		APIKeyPath:     "/tmp/AuthKey_KEY123.p8",
		ProfilePath:    "/tmp/Core.provisionprofile",
		Category:       "public.app-category.developer-tools",
		Copyright:      "Copyright 2026 Lethean CIC. EUPL-1.2.",
	}, "42")
	require.Error(t, err)
	assert.ErrorContains(t, err, "distribution certificate")
}

func TestApple_BuildApple_AppStorePreflight_Good(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist", "apple")
	profilePath := ax.Join(projectDir, "Core.provisionprofile")
	require.NoError(t, ax.WriteFile(profilePath, []byte("profile"), 0o644))
	metadataPath := writeAppStoreMetadata(t, projectDir)

	oldBuildWails := appleBuildWailsAppFn
	oldSign := appleSignFn
	oldSubmit := appleSubmitAppStoreFn
	t.Cleanup(func() {
		appleBuildWailsAppFn = oldBuildWails
		appleSignFn = oldSign
		appleSubmitAppStoreFn = oldSubmit
	})

	appleBuildWailsAppFn = func(ctx context.Context, cfg WailsBuildConfig) (string, error) {
		appPath := ax.Join(cfg.OutputDir, cfg.Name+".app")
		writeDummyAppBundle(t, appPath, cfg.Name, "safe")
		return appPath, nil
	}
	appleSignFn = func(ctx context.Context, cfg SignConfig) error {
		return nil
	}

	var submitCfg AppStoreConfig
	var submitCalled bool
	appleSubmitAppStoreFn = func(ctx context.Context, cfg AppStoreConfig) error {
		submitCalled = true
		submitCfg = cfg
		return nil
	}

	result, err := BuildApple(context.Background(), &Config{
		FS:         io.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "Core",
		Version:    "v1.2.3",
	}, AppleOptions{
		BundleID:         "ai.lthn.core",
		Arch:             "arm64",
		Sign:             true,
		AppStore:         true,
		CertIdentity:     "Apple Distribution: Lethean CIC (ABC123DEF4)",
		APIKeyID:         "KEY123",
		APIKeyIssuerID:   "ISSUER456",
		APIKeyPath:       "/tmp/AuthKey_KEY123.p8",
		ProfilePath:      profilePath,
		MetadataPath:     metadataPath,
		PrivacyPolicyURL: "https://lthn.ai/privacy",
		Category:         "public.app-category.developer-tools",
		Copyright:        "Copyright 2026 Lethean CIC. EUPL-1.2.",
	}, "42")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, submitCalled)
	assert.Equal(t, result.BundlePath, submitCfg.AppPath)
	assert.Equal(t, "1.2.3", submitCfg.Version)
	assert.Equal(t, "manual", submitCfg.ReleaseType)
}

func TestApple_ValidatePrivacyPolicyURL_Bad(t *testing.T) {
	err := validatePrivacyPolicyURL("")
	require.Error(t, err)
	assert.ErrorContains(t, err, "privacy_policy_url")

	err = validatePrivacyPolicyURL("https://example.com")
	require.Error(t, err)
	assert.ErrorContains(t, err, "non-root path")
}

func TestApple_ValidateAppStoreMetadata_Bad(t *testing.T) {
	projectDir := t.TempDir()
	metadataPath := ax.Join(projectDir, ".core", "apple", "appstore")
	require.NoError(t, io.Local.EnsureDir(ax.Join(metadataPath, "screenshots")))
	require.NoError(t, ax.WriteFile(ax.Join(metadataPath, "screenshots", "shot.png"), []byte("png"), 0o644))

	err := validateAppStoreMetadata(io.Local, projectDir, metadataPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "description")
}

func TestApple_ScanBundleForPrivateAPIUsage_Bad(t *testing.T) {
	appPath := ax.Join(t.TempDir(), "Core.app")
	writeDummyAppBundle(t, appPath, "Core", "/System/Library/PrivateFrameworks/Example.framework")

	err := scanBundleForPrivateAPIUsage(io.Local, appPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "private API usage detected")
}

func TestApple_UploadTestFlight_Bad(t *testing.T) {
	err := UploadTestFlight(context.Background(), TestFlightConfig{
		AppPath:        "build/Core.app",
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "api_key_path")
}

func TestApple_SubmitAppStore_Bad(t *testing.T) {
	err := SubmitAppStore(context.Background(), AppStoreConfig{
		AppPath:        "build/Core.app",
		APIKeyID:       "KEY123",
		APIKeyIssuerID: "ISSUER456",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "api_key_path")
}

func writeDummyAppBundle(t *testing.T, appPath, executableName, marker string) {
	t.Helper()

	require.NoError(t, io.Local.EnsureDir(ax.Join(appPath, "Contents", "MacOS")))
	_, err := WriteInfoPlist(io.Local, appPath, InfoPlist{
		BundleID:                      "ai.lthn.core",
		BundleName:                    executableName,
		BundleDisplayName:             executableName,
		BundleVersion:                 "1.0.0",
		BuildNumber:                   "1",
		MinSystemVersion:              "13.0",
		Category:                      "public.app-category.developer-tools",
		Executable:                    executableName,
		HighResCapable:                true,
		SupportsSecureRestorableState: true,
	})
	require.NoError(t, err)
	require.NoError(t, ax.WriteFile(ax.Join(appPath, "Contents", "MacOS", executableName), []byte(marker), 0o755))
}

func writeAppStoreMetadata(t *testing.T, projectDir string) string {
	t.Helper()

	metadataPath := ax.Join(projectDir, ".core", "apple", "appstore")
	require.NoError(t, io.Local.EnsureDir(ax.Join(metadataPath, "screenshots")))
	require.NoError(t, ax.WriteFile(ax.Join(metadataPath, "description.txt"), []byte("Core App Store description"), 0o644))
	require.NoError(t, ax.WriteFile(ax.Join(metadataPath, "screenshots", "shot-1.png"), []byte("png"), 0o644))

	return metadataPath
}
