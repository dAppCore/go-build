package builders

import (
	core "dappco.re/go"
	"dappco.re/go/build/pkg/build"
	coreio "dappco.re/go/io"
)

func TestApplePlist_GenerateAppleInfoPlist_Good(t *core.T) {
	plist := GenerateAppleInfoPlist(&build.Config{Name: "Core", Version: "v1.2.3"}, AppleOptions{BundleID: "ai.lthn.core"}, "42")
	core.AssertEqual(t, "Core", plist.BundleName)
	core.AssertEqual(t, "1.2.3", plist.BundleVersion)
}

func TestApplePlist_GenerateAppleInfoPlist_Bad(t *core.T) {
	plist := GenerateAppleInfoPlist(nil, AppleOptions{}, "")
	core.AssertEqual(t, "App", plist.BundleName)
	core.AssertEqual(t, "1", plist.BuildNumber)
}

func TestApplePlist_GenerateAppleInfoPlist_Ugly(t *core.T) {
	plist := GenerateAppleInfoPlist(&build.Config{Project: build.Project{Name: "ProjectName"}}, AppleOptions{BundleDisplayName: "Display"}, "")
	core.AssertEqual(t, "ProjectName", plist.BundleName)
	core.AssertEqual(t, "Display", plist.BundleDisplayName)
}

func TestApplePlist_WriteAppleInfoPlist_Good(t *core.T) {
	fs := coreio.NewMemoryMedium()
	path, err := WriteAppleInfoPlist(fs, "Core.app", &build.Config{Name: "Core"}, AppleOptions{BundleID: "ai.lthn.core"}, "7")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Core.app/Contents/Info.plist", path)
	core.AssertTrue(t, fs.IsFile(path))
}

func TestApplePlist_WriteAppleInfoPlist_Bad(t *core.T) {
	path, err := WriteAppleInfoPlist(coreio.NewMemoryMedium(), "", nil, AppleOptions{}, "")
	core.AssertError(t, err)
	core.AssertEqual(t, "", path)
}

func TestApplePlist_WriteAppleInfoPlist_Ugly(t *core.T) {
	fs := coreio.NewMemoryMedium()
	path, err := WriteAppleInfoPlist(fs, "Edge.app", nil, AppleOptions{}, "")
	core.RequireNoError(t, err)
	content, readErr := fs.Read(path)
	core.RequireNoError(t, readErr)
	core.AssertContains(t, content, "CFBundleName")
}

func TestApplePlist_AppleInfoPlist_Values_Good(t *core.T) {
	values := (AppleInfoPlist{BundleID: "ai.lthn.core", BundleName: "Core", Executable: "Core"}).Values()
	core.AssertEqual(t, "ai.lthn.core", values["CFBundleIdentifier"])
	core.AssertEqual(t, "Core", values["CFBundleExecutable"])
}

func TestApplePlist_AppleInfoPlist_Values_Bad(t *core.T) {
	values := (AppleInfoPlist{}).Values()
	core.AssertEqual(t, "", values["CFBundleIdentifier"])
	core.AssertEqual(t, true, values["NSHighResolutionCapable"])
}

func TestApplePlist_AppleInfoPlist_Values_Ugly(t *core.T) {
	values := (AppleInfoPlist{BundleVersion: "0.0.0", BuildNumber: "1"}).Values()
	core.AssertEqual(t, "0.0.0", values["CFBundleShortVersionString"])
	core.AssertEqual(t, "1", values["CFBundleVersion"])
}

func TestApplePlist_DefaultAppleEntitlements_Good(t *core.T) {
	entitlements := DefaultAppleEntitlements()
	core.AssertTrue(t, entitlements.HardenedRuntime)
	core.AssertTrue(t, entitlements.NetworkClient)
}

func TestApplePlist_DefaultAppleEntitlements_Bad(t *core.T) {
	entitlements := DefaultAppleEntitlements()
	entitlements.AppSandbox = false
	core.AssertFalse(t, entitlements.AppSandbox)
}

func TestApplePlist_DefaultAppleEntitlements_Ugly(t *core.T) {
	values := DefaultAppleEntitlements().Values()
	core.AssertEqual(t, true, values["com.apple.security.cs.allow-unsigned-executable-memory"])
	core.AssertEqual(t, true, values["com.apple.security.network.client"])
}

func TestApplePlist_WriteAppleEntitlements_Good(t *core.T) {
	fs := coreio.NewMemoryMedium()
	err := WriteAppleEntitlements(fs, "Core.app/Contents/Core.entitlements", DefaultAppleEntitlements())
	core.RequireNoError(t, err)
	core.AssertTrue(t, fs.IsFile("Core.app/Contents/Core.entitlements"))
}

func TestApplePlist_WriteAppleEntitlements_Bad(t *core.T) {
	err := WriteAppleEntitlements(coreio.NewMemoryMedium(), "", DefaultAppleEntitlements())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "path is required")
}

func TestApplePlist_WriteAppleEntitlements_Ugly(t *core.T) {
	fs := coreio.NewMemoryMedium()
	err := WriteAppleEntitlements(fs, "Core.entitlements", AppleEntitlements{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, fs.IsFile("Core.entitlements"))
}

func TestApplePlist_AppleEntitlements_Values_Good(t *core.T) {
	values := DefaultAppleEntitlements().Values()
	core.AssertEqual(t, true, values["com.apple.security.cs.allow-unsigned-executable-memory"])
	core.AssertEqual(t, true, values["com.apple.security.app-sandbox"])
}

func TestApplePlist_AppleEntitlements_Values_Bad(t *core.T) {
	values := (AppleEntitlements{}).Values()
	core.AssertEqual(t, false, values["com.apple.security.network.client"])
	core.AssertEqual(t, false, values["com.apple.security.app-sandbox"])
}

func TestApplePlist_AppleEntitlements_Values_Ugly(t *core.T) {
	values := (AppleEntitlements{NetworkClient: true}).Values()
	core.AssertEqual(t, false, values["com.apple.security.app-sandbox"])
	core.AssertEqual(t, true, values["com.apple.security.network.client"])
}

func TestApplePlist_AppleBuilder_WriteXcodeCloudConfig_Good(t *core.T) {
	fs := coreio.NewMemoryMedium()
	paths, err := NewAppleBuilder().WriteXcodeCloudConfig(fs, "Project", &build.Config{Name: "Core"}, AppleOptions{Arch: "universal"})
	core.RequireNoError(t, err)
	core.AssertLen(t, paths, 3)
}

func TestApplePlist_AppleBuilder_WriteXcodeCloudConfig_Bad(t *core.T) {
	paths, err := NewAppleBuilder().WriteXcodeCloudConfig(nil, "", nil, AppleOptions{})
	core.AssertError(t, err)
	core.AssertNil(t, paths)
}

func TestApplePlist_AppleBuilder_WriteXcodeCloudConfig_Ugly(t *core.T) {
	fs := coreio.NewMemoryMedium()
	paths, err := NewAppleBuilder().WriteXcodeCloudConfig(fs, ".", nil, AppleOptions{})
	core.RequireNoError(t, err)
	core.AssertContains(t, paths, ".xcode-cloud/ci_scripts/ci_post_clone.sh")
}
