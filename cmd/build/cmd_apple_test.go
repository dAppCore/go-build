package buildcmd

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"dappco.re/go/build/internal/ax"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/build/pkg/build/signing"
	"dappco.re/go/core"
)

func TestBuildCmd_resolveAppleCommandOptions_Good(t *testing.T) {
	cfg := &build.BuildConfig{
		Apple: build.AppleConfig{
			BundleID: "ai.lthn.core",
			Arch:     "arm64",
			Sign:     boolPtr(false),
		},
		Sign: signing.SignConfig{
			MacOS: signing.MacOSConfig{
				Identity:    "Developer ID Application: Lethean CIC (ABC123DEF4)",
				TeamID:      "ABC123DEF4",
				AppleID:     "dev@example.com",
				AppPassword: "secret",
			},
		},
	}

	options := resolveAppleCommandOptions(cfg, appleCLIOptions{})
	if !stdlibAssertEqual("ai.lthn.core", options.BundleID) {
		t.Fatalf("want %v, got %v", "ai.lthn.core", options.BundleID)
	}
	if !stdlibAssertEqual("arm64", options.Arch) {
		t.Fatalf("want %v, got %v", "arm64", options.Arch)
	}
	if options.Sign {
		t.Fatal("expected false")
	}
	if !stdlibAssertEqual("Developer ID Application: Lethean CIC (ABC123DEF4)", options.CertIdentity) {
		t.Fatalf("want %v, got %v", "Developer ID Application: Lethean CIC (ABC123DEF4)", options.CertIdentity)
	}
	if !stdlibAssertEqual("ABC123DEF4", options.TeamID) {
		t.Fatalf("want %v, got %v", "ABC123DEF4", options.TeamID)
	}
	if !stdlibAssertEqual("dev@example.com", options.AppleID) {
		t.Fatalf("want %v, got %v", "dev@example.com", options.AppleID)
	}
	if !stdlibAssertEqual("secret", options.Password) {
		t.Fatalf("want %v, got %v", "secret", options.Password)
	}

	options = resolveAppleCommandOptions(cfg, appleCLIOptions{
		Arch:              "universal",
		ArchChanged:       true,
		Sign:              true,
		SignChanged:       true,
		BundleID:          "ai.lthn.core.preview",
		BundleIDChanged:   true,
		TeamID:            "ZZZ9876543",
		TeamIDChanged:     true,
		TestFlight:        true,
		TestFlightChanged: true,
	})
	if !stdlibAssertEqual("universal", options.Arch) {
		t.Fatalf("want %v, got %v", "universal", options.Arch)
	}
	if !(options.Sign) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("ai.lthn.core.preview", options.BundleID) {
		t.Fatalf("want %v, got %v", "ai.lthn.core.preview", options.BundleID)
	}
	if !stdlibAssertEqual("ZZZ9876543", options.TeamID) {
		t.Fatalf("want %v, got %v", "ZZZ9876543", options.TeamID)
	}
	if !(options.TestFlight) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_resolveAppleBuildNumber_Good(t *testing.T) {
	t.Run("prefers github run number when valid", func(t *testing.T) {
		t.Setenv("GITHUB_RUN_NUMBER", "77")
		value, err := resolveAppleBuildNumber(context.Background(), t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("77", value) {
			t.Fatalf("want %v, got %v", "77", value)
		}

	})

	t.Run("falls back to git commit count", func(t *testing.T) {
		dir := t.TempDir()
		runGit(t, dir, "init")
		runGit(t, dir, "config", "user.email", "test@example.com")
		runGit(t, dir, "config", "user.name", "Test User")
		if err := ax.WriteFile(ax.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "feat: initial commit")

		t.Setenv("GITHUB_RUN_NUMBER", "")
		value, err := resolveAppleBuildNumber(context.Background(), dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("1", value) {
			t.Fatalf("want %v, got %v", "1", value)
		}

	})
}

func TestBuildCmd_AddAppleCommand_Good(t *testing.T) {
	c := core.New()
	AddAppleCommand(c)

	result := c.Command("build/apple")
	if !(result.OK) {
		t.Fatal("expected true")
	}

	command, ok := result.Value.(*core.Command)
	if !(ok) {
		t.Fatal("expected true")
	}
	if !stdlibAssertEqual("build/apple", command.Path) {
		t.Fatalf("want %v, got %v", "build/apple", command.Path)
	}
	if !stdlibAssertEqual("cmd.build.apple.long", command.Description) {
		t.Fatalf("want %v, got %v", "cmd.build.apple.long", command.Description)
	}

}

func TestBuildCmd_runAppleBuildInDir_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	if err := ax.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
sign:
  macos:
    identity: "Developer ID Application: Lethean CIC (ABC123DEF4)"
    team_id: ABC123DEF4
    apple_id: dev@example.com
    app_password: secret
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	var called bool
	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		called = true
		if !stdlibAssertEqual(ax.Join(projectDir, "out"), cfg.OutputDir) {
			t.Fatalf("want %v, got %v", ax.Join(projectDir, "out"), cfg.OutputDir)
		}
		if !stdlibAssertEqual("Core", cfg.Name) {
			t.Fatalf("want %v, got %v", "Core", cfg.Name)
		}
		if !stdlibAssertEqual("v1.2.3", cfg.Version) {
			t.Fatalf("want %v, got %v", "v1.2.3", cfg.Version)
		}
		if !stdlibAssertEqual("42", buildNumber) {
			t.Fatalf("want %v, got %v", "42", buildNumber)
		}
		if !stdlibAssertEqual("ai.lthn.core", options.BundleID) {
			t.Fatalf("want %v, got %v", "ai.lthn.core", options.BundleID)
		}
		if !(options.Sign) {
			t.Fatal("expected true")
		}

		return &build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		}, nil
	}

	err := runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Sign:        true,
		SignChanged: true,
		Version:     "v1.2.3",
		BuildNumber: "42",
		OutputDir:   "out",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !(called) {
		t.Fatal("expected true")
	}

}

func TestBuildCmd_runAppleBuildInDir_RejectsUnsafeVersion_Bad(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	if err := ax.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		t.Fatal("buildAppleFn must not be called for unsafe versions")
		return nil, nil
	}

	err := runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3 --bad",
		BuildNumber: "42",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !stdlibAssertContains(err.Error(), "invalid build version") {
		t.Fatalf("expected %v to contain %v", err.Error(), "invalid build version")
	}

}

func TestBuildCmd_runAppleBuildInDir_SetsUpBuildCache_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	if err := ax.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
build:
  cache:
    enabled: true
    paths:
      - cache/go-build
      - cache/go-mod
apple:
  bundle_id: ai.lthn.core
  sign: false
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		if !stdlibAssertEqual([]string{ax.Join(projectDir, "cache", "go-build"), ax.Join(projectDir, "cache", "go-mod")}, cfg.Cache.Paths) {
			t.Fatalf("want %v, got %v", []string{ax.Join(projectDir, "cache", "go-build"), ax.Join(projectDir, "cache", "go-mod")}, cfg.Cache.Paths)
		}
		if !(cfg.Cache.Enabled) {
			t.Fatal("expected true")
		}
		if !(cfg.FS.Exists(ax.Join(projectDir, ".core", "cache"))) {
			t.Fatal("expected true")
		}
		if !(cfg.FS.Exists(ax.Join(projectDir, "cache", "go-build"))) {
			t.Fatal("expected true")
		}
		if !(cfg.FS.Exists(ax.Join(projectDir, "cache", "go-mod"))) {
			t.Fatal("expected true")
		}

		return &build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		}, nil
	}

	err := runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3",
		BuildNumber: "42",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestBuildCmd_runAppleBuildInDir_WritesXcodeCloudScripts_Good(t *testing.T) {
	projectDir := t.TempDir()
	coreDir := ax.Join(projectDir, ".core")
	if err := ax.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(ax.Join(coreDir, "build.yaml"), []byte(`
project:
  name: Core
  binary: Core
apple:
  bundle_id: ai.lthn.core
  sign: false
  xcode_cloud:
    workflow: CoreGUI Release
`), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldBuildApple := buildAppleFn
	t.Cleanup(func() {
		buildAppleFn = oldBuildApple
	})

	buildAppleFn = func(ctx context.Context, cfg *build.Config, options build.AppleOptions, buildNumber string) (*build.AppleBuildResult, error) {
		return &build.AppleBuildResult{
			BundlePath:  ax.Join(cfg.OutputDir, "Core.app"),
			Version:     "1.2.3",
			BuildNumber: buildNumber,
		}, nil
	}

	err := runAppleBuildInDir(context.Background(), projectDir, appleCLIOptions{
		Version:     "v1.2.3",
		BuildNumber: "42",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	preScriptPath := ax.Join(projectDir, build.XcodeCloudScriptsDir, build.XcodeCloudPreXcodebuildScriptName)
	preScript, err := ax.ReadFile(preScriptPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertContains(string(preScript), `core build apple --arch 'universal' --config '.core/build.yaml'`) {
		t.Fatalf("expected %v to contain %v", string(preScript), `core build apple --arch 'universal' --config '.core/build.yaml'`)
	}

}

func boolPtr(value bool) *bool {
	return &value
}

func stdlibAssertEqual(want, got any) bool {
	return reflect.DeepEqual(want, got)
}

func stdlibAssertNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func stdlibAssertEmpty(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

func stdlibAssertZero(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	return !v.IsValid() || v.IsZero()
}

func stdlibAssertContains(container, elem any) bool {
	if s, ok := container.(string); ok {
		sub, ok := elem.(string)
		return ok && strings.Contains(s, sub)
	}

	v := reflect.ValueOf(container)
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(elem)
		if !key.IsValid() {
			return false
		}
		if key.Type().AssignableTo(v.Type().Key()) {
			return v.MapIndex(key).IsValid()
		}
		if key.Type().ConvertibleTo(v.Type().Key()) {
			return v.MapIndex(key.Convert(v.Type().Key())).IsValid()
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if reflect.DeepEqual(v.Index(i).Interface(), elem) {
				return true
			}
		}
	}
	return false
}

func stdlibAssertElementsMatch(want, got any) bool {
	wantValue := reflect.ValueOf(want)
	gotValue := reflect.ValueOf(got)
	if !wantValue.IsValid() || !gotValue.IsValid() {
		return !wantValue.IsValid() && !gotValue.IsValid()
	}
	if !isListValue(wantValue) || !isListValue(gotValue) {
		return reflect.DeepEqual(want, got)
	}
	if wantValue.Len() != gotValue.Len() {
		return false
	}

	used := make([]bool, gotValue.Len())
	for i := 0; i < wantValue.Len(); i++ {
		found := false
		wantElem := wantValue.Index(i).Interface()
		for j := 0; j < gotValue.Len(); j++ {
			if used[j] {
				continue
			}
			if reflect.DeepEqual(wantElem, gotValue.Index(j).Interface()) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func isListValue(value reflect.Value) bool {
	return value.Kind() == reflect.Array || value.Kind() == reflect.Slice
}
