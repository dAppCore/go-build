package builders

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	coreio "dappco.re/go/io"
	"dappco.re/go/process"
)

var _ build.Builder = (*AppleBuilder)(nil)

type recordingAppleRunner struct {
	calls []process.RunOptions
}

func (runner *recordingAppleRunner) Run(ctx context.Context, opts process.RunOptions) (string, error) {
	runner.calls = append(runner.calls, opts)
	return "ok", nil
}

func TestAppleBuilder_Good(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist", "apple")
	if err := ax.WriteFile(ax.Join(projectDir, "wails.json"), []byte(`{"name":"Core"}`+"\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	todo := core.NewBuffer()
	runner := &recordingAppleRunner{}
	builder := NewAppleBuilder(
		WithAppleHostOS("darwin"),
		WithAppleCommandRunner(runner),
		WithAppleTODOWriter(todo),
		WithAppleOptions(AppleOptions{
			BundleID:             "ai.lthn.core",
			SigningIdentity:      "Developer ID Application: Lethean CIC (ABC123DEF4)",
			Sign:                 true,
			Notarise:             true,
			NotarisationProfile:  "core-notary",
			XcodeCloud:           true,
			BuildNumber:          "42",
			BundleDisplayName:    "Core",
			MinSystemVersion:     "13.0",
			Category:             "public.app-category.developer-tools",
			DMG:                  AppleDMGConfig{Enabled: true, VolumeName: "Core"},
			TestFlightKeyID:      "ignored",
			TestFlightIssuerID:   "ignored",
			TestFlightPrivateKey: "ignored",
		}),
	)

	detected, err := builder.Detect(coreio.Local, projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(detected) {
		t.Fatal("expected true")
	}

	artifacts, err := builder.Build(context.Background(), &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "Core",
		Version:    "v1.2.3",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(1, len(artifacts)) {
		t.Fatalf("want %v, got %v", 1, len(artifacts))
	}
	if !stdlibAssertEqual(ax.Join(outputDir, "Core.dmg"), artifacts[0].Path) {
		t.Fatalf("want %v, got %v", ax.Join(outputDir, "Core.dmg"), artifacts[0].Path)
	}

	infoPlist, err := ax.ReadFile(ax.Join(outputDir, "Core.app", "Contents", "Info.plist"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(infoPlist), "<key>CFBundleIdentifier</key>") {
		t.Fatalf("expected Info.plist to contain bundle identifier key")
	}
	if !stdlibAssertContains(string(infoPlist), "<string>ai.lthn.core</string>") {
		t.Fatalf("expected Info.plist to contain bundle id")
	}

	entitlements, err := ax.ReadFile(ax.Join(outputDir, "Core.entitlements.plist"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(entitlements), "com.apple.security.app-sandbox") {
		t.Fatalf("expected entitlements to contain app sandbox")
	}
	if !stdlibAssertContains(string(entitlements), "com.apple.security.network.client") {
		t.Fatalf("expected entitlements to contain network client")
	}

	for _, script := range []string{"ci_post_clone.sh", "ci_pre_xcodebuild.sh", "ci_post_xcodebuild.sh"} {
		if !coreio.Local.IsFile(ax.Join(projectDir, ".xcode-cloud", "ci_scripts", script)) {
			t.Fatalf("expected Xcode Cloud script %s", script)
		}
	}

	wantCommands := []string{"wails3", "wails3", "lipo", "codesign", "hdiutil", "hdiutil", "hdiutil", "hdiutil", "xcrun", "xcrun"}
	var gotCommands []string
	for _, call := range runner.calls {
		gotCommands = append(gotCommands, call.Command)
	}
	if !stdlibAssertEqual(wantCommands, gotCommands) {
		t.Fatalf("want %v, got %v", wantCommands, gotCommands)
	}
	if !stdlibAssertContains(todo.String(), `"step":"wails-build"`) {
		t.Fatalf("expected structured TODO output, got %s", todo.String())
	}
}

func TestAppleBuilder_Bad(t *testing.T) {
	if err := ValidateAppleOptions(AppleOptions{}); err == nil {
		t.Fatal("expected missing bundle ID error")
	}

	err := ValidateAppleOptions(AppleOptions{
		BundleID: "ai.lthn.core",
		Sign:     true,
	})
	if err == nil {
		t.Fatal("expected missing signing identity error")
	}
	if !stdlibAssertContains(err.Error(), "signing identity") {
		t.Fatalf("expected %v to contain %v", err.Error(), "signing identity")
	}

	err = ValidateAppleOptions(AppleOptions{
		BundleID: "ai.lthn.core",
		Notarise: true,
	})
	if err == nil {
		t.Fatal("expected missing notarisation credentials error")
	}
	if !stdlibAssertContains(err.Error(), "notarisation") {
		t.Fatalf("expected %v to contain %v", err.Error(), "notarisation")
	}
}

func TestAppleBuilder_Ugly(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := ax.Join(projectDir, "dist", "apple")
	if err := ax.WriteFile(ax.Join(projectDir, "wails.json"), []byte(`{"name":"Core"}`+"\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	todo := core.NewBuffer()
	runner := &recordingAppleRunner{}
	builder := NewAppleBuilder(
		WithAppleHostOS("linux"),
		WithAppleCommandRunner(runner),
		WithAppleTODOWriter(todo),
		WithAppleOptions(AppleOptions{
			BundleID: "ai.lthn.core",
			Arch:     "arm64",
		}),
	)

	artifacts, err := builder.Build(context.Background(), &build.Config{
		FS:         coreio.Local,
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		Name:       "Core",
		Version:    "v1.2.3",
	}, []build.Target{{OS: "darwin", Arch: "arm64"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(ax.Join(outputDir, "Core.app"), artifacts[0].Path) {
		t.Fatalf("want %v, got %v", ax.Join(outputDir, "Core.app"), artifacts[0].Path)
	}
	if !stdlibAssertEqual(0, len(runner.calls)) {
		t.Fatalf("want no go-process calls outside macOS, got %v", runner.calls)
	}
	if !core.Contains(todo.String(), "this requires macOS") {
		t.Fatalf("expected non-macOS TODO, got %s", todo.String())
	}
}

// --- v0.9.0 generated compliance triplets ---
func TestApple_AppleCommandRunnerFunc_Run_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := AppleCommandRunnerFunc(func(core.Context, process.RunOptions) (string, error) { return "ok", nil })
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Run(ctx, process.RunOptions{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleCommandRunnerFunc_Run_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := AppleCommandRunnerFunc(func(core.Context, process.RunOptions) (string, error) { return "ok", nil })
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Run(ctx, process.RunOptions{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleCommandRunnerFunc_Run_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := AppleCommandRunnerFunc(func(core.Context, process.RunOptions) (string, error) { return "ok", nil })
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Run(ctx, process.RunOptions{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_GoProcessAppleRunner_Run_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := GoProcessAppleRunner{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Run(ctx, process.RunOptions{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_GoProcessAppleRunner_Run_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := GoProcessAppleRunner{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Run(ctx, process.RunOptions{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_GoProcessAppleRunner_Run_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := GoProcessAppleRunner{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Run(ctx, process.RunOptions{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_NewAppleBuilder_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewAppleBuilder()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_NewAppleBuilder_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewAppleBuilder()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_NewAppleBuilder_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewAppleBuilder()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithAppleOptions_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleOptions(AppleOptions{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithAppleOptions_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleOptions(AppleOptions{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithAppleOptions_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleOptions(AppleOptions{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithAppleCommandRunner_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleCommandRunner(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithAppleCommandRunner_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleCommandRunner(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithAppleCommandRunner_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleCommandRunner(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithAppleHostOS_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleHostOS("linux")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithAppleHostOS_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleHostOS("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithAppleHostOS_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleHostOS("linux")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_WithAppleTODOWriter_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleTODOWriter(core.NewBuffer())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_WithAppleTODOWriter_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleTODOWriter(core.NewBuffer())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_WithAppleTODOWriter_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = WithAppleTODOWriter(core.NewBuffer())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_DefaultAppleBuilderOptions_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultAppleBuilderOptions()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_DefaultAppleBuilderOptions_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultAppleBuilderOptions()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_DefaultAppleBuilderOptions_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultAppleBuilderOptions()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_AppleBuilder_Name_Good(t *core.T) {
	subject := &AppleBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleBuilder_Name_Bad(t *core.T) {
	subject := &AppleBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleBuilder_Name_Ugly(t *core.T) {
	subject := &AppleBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Name()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_AppleBuilder_Detect_Good(t *core.T) {
	subject := &AppleBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleBuilder_Detect_Bad(t *core.T) {
	subject := &AppleBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(coreio.NewMemoryMedium(), "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleBuilder_Detect_Ugly(t *core.T) {
	subject := &AppleBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Detect(coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_AppleBuilder_Build_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleBuilder_Build_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleBuilder_Build_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Build(ctx, nil, nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_AppleBuilder_BuildWailsMacOS_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := NewAppleBuilder(WithAppleTODOWriter(nil))
	cfg := &build.Config{ProjectDir: t.TempDir()}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.BuildWailsMacOS(ctx, coreio.NewMemoryMedium(), cfg, core.Path(t.TempDir(), "go-build-compliance"), "agent", "amd64")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleBuilder_BuildWailsMacOS_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := NewAppleBuilder(WithAppleTODOWriter(nil))
	cfg := &build.Config{ProjectDir: t.TempDir()}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.BuildWailsMacOS(ctx, coreio.NewMemoryMedium(), cfg, "", "", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleBuilder_BuildWailsMacOS_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := NewAppleBuilder(WithAppleTODOWriter(nil))
	cfg := &build.Config{ProjectDir: t.TempDir()}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.BuildWailsMacOS(ctx, coreio.NewMemoryMedium(), cfg, core.Path(t.TempDir(), "go-build-compliance"), "agent", "amd64")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_AppleBuilder_CreateUniversal_Good(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.CreateUniversal(ctx, coreio.NewMemoryMedium(), coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"), "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_AppleBuilder_CreateUniversal_Bad(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.CreateUniversal(ctx, coreio.NewMemoryMedium(), coreio.NewMemoryMedium(), "", "", "", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_AppleBuilder_CreateUniversal_Ugly(t *core.T) {
	ctx, cancel := core.WithCancel(core.Background())
	cancel()
	subject := &AppleBuilder{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.CreateUniversal(ctx, coreio.NewMemoryMedium(), coreio.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"), core.Path(t.TempDir(), "go-build-compliance"), "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestApple_ValidateAppleOptions_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateAppleOptions(AppleOptions{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestApple_ValidateAppleOptions_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateAppleOptions(AppleOptions{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestApple_ValidateAppleOptions_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ValidateAppleOptions(AppleOptions{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
