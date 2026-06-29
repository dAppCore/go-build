package sdk

import (
	core "dappco.re/go"
	"dappco.re/go/build/internal/ax"
	"github.com/oasdiff/oasdiff/checker"
	yaml "gopkg.in/yaml.v3"
)

// Behaviour tests drive the real config-resolution branches the no-panic
// triplets skipped: every UnmarshalYAML node shape, version-template
// resolution, monorepo output-path composition, config cloning, and language
// normalisation.

func TestSdk_UnmarshalYAML_Scalar_Good(t *core.T) {
	c := &DiffConfig{}
	core.AssertTrue(t, c.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "true"}).OK)
	core.AssertTrue(t, c.Enabled)
	core.AssertTrue(t, c.EnabledConfigured)

	c = &DiffConfig{}
	core.AssertTrue(t, c.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "false"}).OK)
	core.AssertFalse(t, c.Enabled)
	core.AssertTrue(t, c.EnabledConfigured)
}

func TestSdk_UnmarshalYAML_Scalar_Bad(t *core.T) {
	// A non-boolean scalar cannot decode into the enabled flag.
	c := &DiffConfig{}
	result := c.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "not-a-bool"})
	core.AssertFalse(t, result.OK)
}

func TestSdk_UnmarshalYAML_Mapping_Ugly(t *core.T) {
	// The expanded mapping form sets fail_on_breaking and the explicit enabled.
	var root yaml.Node
	core.AssertEqual(t, nil, yaml.Unmarshal([]byte("enabled: true\nfail_on_breaking: true\n"), &root))
	mapping := root.Content[0]

	c := &DiffConfig{}
	core.AssertTrue(t, c.UnmarshalYAML(mapping).OK)
	core.AssertTrue(t, c.Enabled)
	core.AssertTrue(t, c.EnabledConfigured)
	core.AssertTrue(t, c.FailOnBreaking)

	// A mapping omitting enabled leaves EnabledConfigured false.
	core.AssertEqual(t, nil, yaml.Unmarshal([]byte("fail_on_breaking: false\n"), &root))
	c = &DiffConfig{}
	core.AssertTrue(t, c.UnmarshalYAML(root.Content[0]).OK)
	core.AssertFalse(t, c.EnabledConfigured)
}

func TestSdk_UnmarshalYAML_Sequence_Bad(t *core.T) {
	// A sequence node hits the default branch and fails to decode into the alias
	// struct.
	var root yaml.Node
	core.AssertEqual(t, nil, yaml.Unmarshal([]byte("- one\n- two\n"), &root))
	c := &DiffConfig{}
	core.AssertFalse(t, c.UnmarshalYAML(root.Content[0]).OK)
}

func TestSdk_ResolvePackageVersion_Good(t *core.T) {
	// An explicit, non-template version is returned verbatim.
	s := New(".", &Config{Package: PackageConfig{Version: "3.4.5"}})
	core.AssertEqual(t, "3.4.5", s.resolvePackageVersion())
}

func TestSdk_ResolvePackageVersion_Template_Ugly(t *core.T) {
	// A template placeholder is rendered against the SDK version.
	s := New(".", &Config{Package: PackageConfig{Version: "{{.Version}}-rc"}})
	s.SetVersion("9.0.0")
	core.AssertEqual(t, "9.0.0-rc", s.resolvePackageVersion())

	// With no SDK version set, the template string is left untouched.
	s = New(".", &Config{Package: PackageConfig{Version: "{{Version}}"}})
	core.AssertEqual(t, "{{Version}}", s.resolvePackageVersion())
}

func TestSdk_ResolvePackageVersion_Empty_Bad(t *core.T) {
	// An empty package version falls back to the SDK version field.
	s := New(".", &Config{})
	s.version = "fallback-1.0"
	core.AssertEqual(t, "fallback-1.0", s.resolvePackageVersion())
}

func TestSdk_SetVersion_DoesNotOverrideTemplate_Ugly(t *core.T) {
	// SetVersion records the version but leaves a templated package version in
	// place so it can be rendered later.
	s := New(".", &Config{Package: PackageConfig{Version: "{{.Version}}"}})
	s.SetVersion("2.2.2")
	core.AssertEqual(t, "{{.Version}}", s.config.Package.Version)
	core.AssertEqual(t, "2.2.2", s.resolvePackageVersion())
}

func TestSdk_OutputDir_PlainRoot_Good(t *core.T) {
	s := New("/proj", &Config{Output: "sdk"})
	core.AssertEqual(t, ax.Join("/proj", "sdk", "typescript"), s.outputDir("typescript"))
}

func TestSdk_OutputDir_MonorepoPublishPath_Ugly(t *core.T) {
	// A publish path prefixes the output root, composing the monorepo layout.
	s := New("/proj", &Config{Output: "sdk", Publish: PublishConfig{Path: "packages/api"}})
	core.AssertEqual(t, "packages/api/sdk", s.outputRoot())
	core.AssertEqual(t, ax.Join("/proj", "packages/api/sdk", "go"), s.outputDir("go"))
}

func TestSdk_Config_ReturnsClone_Good(t *core.T) {
	s := New(".", &Config{Languages: []string{"go"}})
	clone := s.Config()
	core.AssertFalse(t, clone == nil)
	// Mutating the clone must not affect the SDK's internal config.
	clone.Languages[0] = "rust"
	core.AssertEqual(t, "go", s.config.Languages[0])
}

func TestSdk_Config_NilSDK_Bad(t *core.T) {
	var s *SDK
	core.AssertTrue(t, s.Config() == nil)
}

func TestSdk_NormaliseLanguages_DedupesAndAliases_Ugly(t *core.T) {
	// Aliases collapse to canonical names, duplicates and blanks are dropped,
	// order is preserved.
	got := normaliseLanguages([]string{"ts", "TypeScript", "py", "", "golang", "go", "php"})
	core.AssertEqual(t, []string{"typescript", "python", "go", "php"}, got)

	// A nil slice stays nil; an empty slice stays empty (distinct contracts).
	core.AssertTrue(t, normaliseLanguages(nil) == nil)
	core.AssertEqual(t, 0, len(normaliseLanguages([]string{})))
}

func TestSdk_DiffSummary_ErrLevel_Good(t *core.T) {
	// At ERR level only breaking changes are summarised; warnings are ignored.
	breaking := &DiffResult{Breaking: true, Changes: []string{"a", "b"}}
	core.AssertEqual(t, "2 breaking change(s) detected", diffSummary(breaking, checker.ERR))

	clean := &DiffResult{HasWarnings: true, Warnings: []string{"w"}}
	core.AssertEqual(t, "No breaking changes", diffSummary(clean, checker.ERR))
}

func TestSdk_DiffSummary_WarnLevel_Ugly(t *core.T) {
	// At WARN level breaking + warning counts combine and degrade gracefully.
	both := &DiffResult{Breaking: true, Changes: []string{"a"}, HasWarnings: true, Warnings: []string{"w1", "w2"}}
	core.AssertEqual(t, "1 breaking change(s), 2 warning(s) detected", diffSummary(both, checker.WARN))

	breakingOnly := &DiffResult{Breaking: true, Changes: []string{"a", "b", "c"}}
	core.AssertEqual(t, "3 breaking change(s) detected", diffSummary(breakingOnly, checker.WARN))

	warnOnly := &DiffResult{HasWarnings: true, Warnings: []string{"w"}}
	core.AssertEqual(t, "1 warning(s) detected", diffSummary(warnOnly, checker.WARN))

	none := &DiffResult{}
	core.AssertEqual(t, "No warnings or breaking changes", diffSummary(none, checker.WARN))
}

func TestSdk_DiffSummary_Nil_Bad(t *core.T) {
	core.AssertEqual(t, "No breaking changes", diffSummary(nil, checker.WARN))
}

func TestSdk_DetectScramble_NoComposer_Bad(t *core.T) {
	// An empty project directory has no composer.json, so scramble detection
	// fails up front.
	s := New(t.TempDir(), &Config{})
	result := s.detectScramble()
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "no composer.json"))
}

func TestSdk_DetectScramble_ComposerWithoutScramble_Bad(t *core.T) {
	// A composer.json that does not reference scramble is rejected.
	dir := t.TempDir()
	core.AssertTrue(t, ax.WriteFile(ax.Join(dir, "composer.json"), []byte(`{"require":{"laravel/framework":"^11"}}`), 0o644).OK)
	s := New(dir, &Config{})
	result := s.detectScramble()
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "scramble not found"))
}

func TestSdk_DetectSpec_ConfiguredMissing_Bad(t *core.T) {
	s := New(t.TempDir(), &Config{Spec: "docs/openapi.yaml"})
	result := s.DetectSpec()
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "configured spec not found"))
}

func TestSdk_DetectSpec_ConfiguredFound_Good(t *core.T) {
	dir := t.TempDir()
	core.AssertTrue(t, ax.WriteFile(ax.Join(dir, "my-spec.yaml"), []byte("openapi: 3.0.0\n"), 0o644).OK)
	s := New(dir, &Config{Spec: "my-spec.yaml"})
	result := s.DetectSpec()
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, ax.Join(dir, "my-spec.yaml"), result.Value.(string))
}

func TestSdk_DetectSpec_None_Bad(t *core.T) {
	s := New(t.TempDir(), &Config{})
	result := s.DetectSpec()
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "no OpenAPI spec found"))
}
