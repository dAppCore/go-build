package builders

import (
	"sort"

	"dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	coreio "dappco.re/go/build/pkg/storage"
)

// AppleInfoPlist contains the generated macOS app bundle metadata.
type AppleInfoPlist struct {
	BundleID          string
	BundleName        string
	BundleDisplayName string
	BundleVersion     string
	BuildNumber       string
	Executable        string
	MinSystemVersion  string
	Category          string
	Copyright         string
}

// AppleEntitlements contains the default macOS sandbox entitlements.
type AppleEntitlements struct {
	HardenedRuntime bool
	AppSandbox      bool
	NetworkClient   bool
}

// GenerateAppleInfoPlist creates Info.plist metadata from the build Config.
func GenerateAppleInfoPlist(cfg *build.Config, options AppleOptions, buildNumber string) AppleInfoPlist {
	name := "App"
	version := "0.0.0"
	if cfg != nil {
		name = resolveAppleBuilderName(cfg)
		version = normalizeAppleBuilderVersion(cfg.Version)
	}
	if buildNumber == "" {
		buildNumber = "1"
	}

	options = options.withDefaults()
	return AppleInfoPlist{
		BundleID:          options.BundleID,
		BundleName:        name,
		BundleDisplayName: firstNonEmptyApple(options.BundleDisplayName, name),
		BundleVersion:     version,
		BuildNumber:       buildNumber,
		Executable:        name,
		MinSystemVersion:  options.MinSystemVersion,
		Category:          options.Category,
		Copyright:         options.Copyright,
	}
}

// WriteAppleInfoPlist writes Contents/Info.plist for appPath.
func WriteAppleInfoPlist(filesystem coreio.Medium, appPath string, cfg *build.Config, options AppleOptions, buildNumber string) core.Result {
	if filesystem == nil {
		filesystem = coreio.Local
	}
	if appPath == "" {
		return core.Fail(core.E("AppleBuilder.WriteInfoPlist", "app path is required", nil))
	}

	plist := GenerateAppleInfoPlist(cfg, options, buildNumber)
	path := ax.Join(appPath, "Contents", "Info.plist")
	created := filesystem.EnsureDir(ax.Dir(path))
	if !created.OK {
		return core.Fail(core.E("AppleBuilder.WriteInfoPlist", "failed to create Info.plist directory", core.NewError(created.Error())))
	}
	written := filesystem.WriteMode(path, encodeApplePlist(plist.Values()), 0o644)
	if !written.OK {
		return core.Fail(core.E("AppleBuilder.WriteInfoPlist", "failed to write Info.plist", core.NewError(written.Error())))
	}
	return core.Ok(path)
}

// Values converts the plist metadata to Apple Info.plist keys.
func (plist AppleInfoPlist) Values() map[string]any {
	return map[string]any{
		"CFBundleDevelopmentRegion":       "en",
		"CFBundleDisplayName":             plist.BundleDisplayName,
		"CFBundleExecutable":              plist.Executable,
		"CFBundleIdentifier":              plist.BundleID,
		"CFBundleInfoDictionaryVersion":   "6.0",
		"CFBundleName":                    plist.BundleName,
		"CFBundlePackageType":             "APPL",
		"CFBundleShortVersionString":      plist.BundleVersion,
		"CFBundleVersion":                 plist.BuildNumber,
		"LSApplicationCategoryType":       plist.Category,
		"LSMinimumSystemVersion":          plist.MinSystemVersion,
		"NSHighResolutionCapable":         true,
		"NSHumanReadableCopyright":        plist.Copyright,
		"NSSupportsSecureRestorableState": true,
	}
}

// DefaultAppleEntitlements returns the skeleton hardened runtime, sandbox, and network-client entitlements.
func DefaultAppleEntitlements() AppleEntitlements {
	return AppleEntitlements{
		HardenedRuntime: true,
		AppSandbox:      true,
		NetworkClient:   true,
	}
}

// WriteAppleEntitlements writes a macOS entitlements plist.
func WriteAppleEntitlements(filesystem coreio.Medium, path string, entitlements AppleEntitlements) core.Result {
	if filesystem == nil {
		filesystem = coreio.Local
	}
	if path == "" {
		return core.Fail(core.E("AppleBuilder.WriteEntitlements", "entitlements path is required", nil))
	}
	created := filesystem.EnsureDir(ax.Dir(path))
	if !created.OK {
		return core.Fail(core.E("AppleBuilder.WriteEntitlements", "failed to create entitlements directory", core.NewError(created.Error())))
	}
	written := filesystem.WriteMode(path, encodeApplePlist(entitlements.Values()), 0o644)
	if !written.OK {
		return core.Fail(core.E("AppleBuilder.WriteEntitlements", "failed to write entitlements", core.NewError(written.Error())))
	}
	return core.Ok(nil)
}

// Values converts entitlements to Apple entitlement keys.
func (entitlements AppleEntitlements) Values() map[string]any {
	return map[string]any{
		"com.apple.security.app-sandbox":                         entitlements.AppSandbox,
		"com.apple.security.cs.allow-unsigned-executable-memory": entitlements.HardenedRuntime,
		"com.apple.security.network.client":                      entitlements.NetworkClient,
	}
}

// WriteXcodeCloudConfig writes the AppleBuilder Xcode Cloud script templates.
func (b *AppleBuilder) WriteXcodeCloudConfig(filesystem coreio.Medium, projectDir string, cfg *build.Config, options AppleOptions) core.Result {
	if filesystem == nil {
		filesystem = coreio.Local
	}
	baseDir := ax.Join(projectDir, ".xcode-cloud", "ci_scripts")
	created := filesystem.EnsureDir(baseDir)
	if !created.OK {
		return core.Fail(core.E("AppleBuilder.WriteXcodeCloudConfig", "failed to create Xcode Cloud scripts directory", core.NewError(created.Error())))
	}

	name := "App"
	if cfg != nil {
		name = resolveAppleBuilderName(cfg)
	}
	buildCommand := "core build apple --config .core/build.yaml --arch " + shellQuoteApple(options.withDefaults().Arch)

	scripts := map[string]string{
		"ci_post_clone.sh":      xcodeCloudPostCloneScript(),
		"ci_pre_xcodebuild.sh":  xcodeCloudPreXcodebuildScript(buildCommand),
		"ci_post_xcodebuild.sh": xcodeCloudPostXcodebuildScript(name),
	}

	ordered := []string{"ci_post_clone.sh", "ci_pre_xcodebuild.sh", "ci_post_xcodebuild.sh"}
	paths := make([]string, 0, len(ordered))
	for _, name := range ordered {
		path := ax.Join(baseDir, name)
		written := filesystem.WriteMode(path, scripts[name], 0o755)
		if !written.OK {
			return core.Fail(core.E("AppleBuilder.WriteXcodeCloudConfig", "failed to write "+name, core.NewError(written.Error())))
		}
		paths = append(paths, path)
	}
	return core.Ok(paths)
}

func encodeApplePlist(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	b := core.NewBuilder()
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n")
	b.WriteString("<dict>\n")
	for _, key := range keys {
		b.WriteString("\t<key>")
		b.WriteString(escapeAppleXML(key))
		b.WriteString("</key>\n")
		b.WriteString(applePlistValue(values[key]))
	}
	b.WriteString("</dict>\n")
	b.WriteString("</plist>\n")
	return b.String()
}

func applePlistValue(value any) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "\t<true/>\n"
		}
		return "\t<false/>\n"
	case string:
		return "\t<string>" + escapeAppleXML(v) + "</string>\n"
	default:
		return "\t<string>" + escapeAppleXML(core.Sprintf("%v", value)) + "</string>\n"
	}
}

func escapeAppleXML(value string) string {
	b := core.NewBuilder()
	for _, r := range value {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeAppleBuilderVersion(version string) string {
	version = core.Trim(version)
	version = core.TrimPrefix(version, "v")
	if version == "" {
		return "0.0.0"
	}
	return version
}

func xcodeCloudPostCloneScript() string {
	return core.Trim(`#!/usr/bin/env bash
set -euo pipefail

export PATH="${HOME}/go/bin:${HOME}/.deno/bin:${HOME}/.bun/bin:${PATH}"

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required for AppleBuilder Xcode Cloud builds." >&2
  exit 1
fi

if ! command -v wails3 >/dev/null 2>&1 && ! command -v wails >/dev/null 2>&1; then
  echo "Wails is required for AppleBuilder Xcode Cloud builds." >&2
  exit 1
fi
`) + "\n"
}

func xcodeCloudPreXcodebuildScript(buildCommand string) string {
	return core.Trim(`#!/usr/bin/env bash
set -euo pipefail

export PATH="${HOME}/go/bin:${HOME}/.deno/bin:${HOME}/.bun/bin:${PATH}"

`+buildCommand) + "\n"
}

func xcodeCloudPostXcodebuildScript(name string) string {
	bundlePath := ax.Join("dist", "apple", name+".app")
	executablePath := ax.Join(bundlePath, "Contents", "MacOS", name)
	return core.Trim(`#!/usr/bin/env bash
set -euo pipefail

BUNDLE_PATH=`+shellQuoteApple(bundlePath)+`
EXECUTABLE_PATH=`+shellQuoteApple(executablePath)+`

test -d "$BUNDLE_PATH"
test -x "$EXECUTABLE_PATH"
`) + "\n"
}

func shellQuoteApple(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + core.Replace(value, "'", `'"'"'`) + "'"
}
