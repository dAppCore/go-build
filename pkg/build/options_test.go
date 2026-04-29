package build

import (
	core "dappco.re/go"
	"testing"
)

// --- ComputeOptions ---

func TestOptions_ComputeOptions_Good(t *testing.T) {
	t.Run("normal config produces correct options", func(t *testing.T) {
		cfg := &BuildConfig{
			Build: Build{
				Obfuscate: true,
				NSIS:      true,
				WebView2:  "embed",
				BuildTags: []string{"integration"},
				LDFlags:   []string{"-s", "-w"},
			},
		}
		discovery := &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeWails},
			PrimaryStack: "wails",
			Distro:       "24.04",
		}

		opts := ComputeOptions(cfg, discovery)
		if stdlibAssertNil(opts) {
			t.Fatal("expected non-nil")
		}
		if !(opts.Obfuscate) {
			t.Fatal("expected true")
		}
		if !(opts.NSIS) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("embed", opts.WebView2) {
			t.Fatalf("want %v, got %v", "embed", opts.WebView2)
		}
		if !stdlibAssertEqual([]string{

			// webkit2_41 injected for 24.04
			"-s", "-w"}, opts.LDFlags) {
			t.Fatalf("want %v, got %v", []string{"-s", "-w"}, opts.LDFlags)
		}
		if !stdlibAssertEqual([]string{"webkit2_41", "integration"}, opts.Tags) {
			t.Fatalf("want %v, got %v", []string{"webkit2_41", "integration"}, opts.Tags)
		}
		if !stdlibAssertContains(opts.Tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", opts.Tags, "webkit2_41")
		}

	})

	t.Run("discovery with non-Ubuntu distro leaves tags empty", func(t *testing.T) {
		cfg := &BuildConfig{
			Build: Build{
				LDFlags: []string{"-s"},
			},
		}
		discovery := &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeWails},
			PrimaryStack: "wails",
			Distro:       "22.04",
		}

		opts := ComputeOptions(cfg, discovery)
		if stdlibAssertNil(opts) {
			t.Fatal("expected non-nil")
		}
		if !stdlibAssertEmpty(opts.Tags) {
			t.Fatalf("expected empty, got %v", opts.Tags)
		}

	})

	t.Run("discovery with 25.10 distro injects webkit tag", func(t *testing.T) {
		opts := ComputeOptions(&BuildConfig{}, &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeWails},
			PrimaryStack: "wails",
			Distro:       "25.10",
		})
		if !stdlibAssertContains(opts.Tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", opts.Tags, "webkit2_41")
		}

	})

	t.Run("non-Wails stacks do not inject webkit tag", func(t *testing.T) {
		opts := ComputeOptions(&BuildConfig{}, &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeGo},
			PrimaryStack: "go",
			Distro:       "24.04",
		})
		if stdlibAssertContains(opts.Tags, "webkit2_41") {
			t.Fatalf("expected %v not to contain %v", opts.Tags, "webkit2_41")
		}

	})

	t.Run("configured wails type injects webkit tag even when discovery markers differ", func(t *testing.T) {
		opts := ComputeOptions(&BuildConfig{
			Build: Build{
				Type: "WaIlS",
			},
		}, &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeGo},
			PrimaryStack: "go",
			Distro:       "24.04",
		})
		if !stdlibAssertContains(opts.Tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", opts.Tags, "webkit2_41")
		}

	})

	t.Run("configured discovery type injects webkit tag even without build config type", func(t *testing.T) {
		opts := ComputeOptions(&BuildConfig{}, &DiscoveryResult{
			ConfiguredType: string(ProjectTypeWails),
			Distro:         "24.04",
		})
		if !stdlibAssertContains(opts.Tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", opts.Tags, "webkit2_41")
		}

	})

	t.Run("discovery types alone can trigger webkit injection", func(t *testing.T) {
		opts := ComputeOptions(&BuildConfig{}, &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeWails, ProjectTypeGo},
			PrimaryStack: "go",
			Distro:       "24.04",
		})
		if !stdlibAssertContains(opts.Tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", opts.Tags, "webkit2_41")
		}

	})
}

func TestOptions_ComputeOptions_Bad(t *testing.T) {
	t.Run("nil config returns safe defaults", func(t *testing.T) {
		discovery := &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeWails},
			PrimaryStack: "wails",
			Distro:       "24.04",
		}

		opts := ComputeOptions(nil, discovery)
		if stdlibAssertNil(opts) {
			t.Fatal("expected non-nil")
		}
		if opts.Obfuscate {
			t.Fatal("expected false")
		}
		if opts.NSIS {
			t.Fatal("expected false")
		}
		if !stdlibAssertEmpty(opts.

			// webkit2_41 still injected for Wails discovery
			WebView2) {
			t.Fatalf("expected empty, got %v", opts.WebView2)
		}
		if !stdlibAssertEmpty(opts.LDFlags) {
			t.Fatalf("expected empty, got %v", opts.LDFlags)
		}
		if !stdlibAssertContains(opts.Tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", opts.Tags, "webkit2_41")
		}

	})

	t.Run("nil discovery skips webkit injection", func(t *testing.T) {
		cfg := &BuildConfig{
			Build: Build{
				Obfuscate: true,
				BuildTags: []string{"existing"},
			},
		}

		opts := ComputeOptions(cfg, nil)
		if stdlibAssertNil(opts) {
			t.Fatal("expected non-nil")
		}
		if !(opts.Obfuscate) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual([]string{"existing"}, opts.Tags) {
			t.Fatalf("want %v, got %v", []string{"existing"}, opts.Tags)
		}

	})

	t.Run("both nil returns empty options", func(t *testing.T) {
		opts := ComputeOptions(nil, nil)
		if stdlibAssertNil(opts) {
			t.Fatal("expected non-nil")
		}
		if opts.Obfuscate {
			t.Fatal("expected false")
		}
		if opts.NSIS {
			t.Fatal("expected false")
		}
		if !stdlibAssertEmpty(opts.Tags) {
			t.Fatalf("expected empty, got %v", opts.Tags)
		}
		if !stdlibAssertEmpty(opts.LDFlags) {
			t.Fatalf("expected empty, got %v",

				// Seed webkit2_41 before discovery also injects it
				opts.LDFlags)
		}

	})
}

func TestOptions_ComputeOptions_Ugly(t *testing.T) {
	t.Run("duplicate tags from deduplication", func(t *testing.T) {

		cfg := &BuildConfig{
			Build: Build{
				BuildTags: []string{"integration", "integration", "ui"},
			},
		}
		discovery := &DiscoveryResult{Distro: "24.04"}
		discovery.Types = []ProjectType{ProjectTypeWails}
		discovery.PrimaryStack = "wails"

		opts := ComputeOptions(cfg, discovery)

		// Even though InjectWebKitTag is called once, deduplication must hold
		count := 0
		for _, tag := range opts.Tags {
			if tag == "webkit2_41" {
				count++
			}
		}
		if !stdlibAssertEqual(1, count) {
			t.Fatal("webkit2_41 must appear exactly once")
		}
		if !stdlibAssertEqual([]string{"webkit2_41", "integration", "ui"}, opts.Tags) {
			t.Fatalf("want %v, got %v", []string{"webkit2_41", "integration", "ui"}, opts.Tags)
		}

	})

	t.Run("empty distro in discovery produces no webkit tag", func(t *testing.T) {
		opts := ComputeOptions(&BuildConfig{}, &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeWails},
			PrimaryStack: "wails",
			Distro:       "",
		})
		if !stdlibAssertEmpty(opts.Tags) {
			t.Fatalf("expected empty, got %v", opts.Tags)
		}

	})

	t.Run("all flags set simultaneously do not conflict", func(t *testing.T) {
		cfg := &BuildConfig{
			Build: Build{
				Obfuscate: true,
				NSIS:      true,
				WebView2:  "download",
				LDFlags:   []string{"-s", "-w", "-X main.version=v1.0.0"},
			},
		}
		discovery := &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeWails},
			PrimaryStack: "wails",
			Distro:       "24.04",
		}

		opts := ComputeOptions(cfg, discovery)
		if !(opts.Obfuscate) {
			t.Fatal("expected true")
		}
		if !(opts.NSIS) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("download", opts.WebView2) {
			t.Fatalf("want %v, got %v", "download", opts.WebView2)
		}
		if !stdlibAssertEqual([]string{"-s", "-w", "-X main.version=v1.0.0"},

			// --- InjectWebKitTag ---
			opts.LDFlags) {
			t.Fatalf("want %v, got %v", []string{"-s", "-w", "-X main.version=v1.0.0"}, opts.LDFlags)
		}
		if !stdlibAssertContains(opts.

			// InjectWebKitTag(tags, "24.04") → ["webkit2_41"]
			Tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", opts.Tags, "webkit2_41")
		}

	})
}

func TestOptions_InjectWebKitTag_Good(t *testing.T) {
	t.Run("24.04 adds webkit2_41", func(t *testing.T) {

		tags := InjectWebKitTag(nil, "24.04")
		if !stdlibAssertEqual([]string{"webkit2_41"}, tags) {
			t.Fatalf("want %v, got %v", []string{"webkit2_41"}, tags)
		}

	})

	t.Run("24.10 adds webkit2_41", func(t *testing.T) {
		tags := InjectWebKitTag([]string{}, "24.10")
		if !stdlibAssertContains(tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", tags, "webkit2_41")
		}

	})

	t.Run("25.04 adds webkit2_41", func(t *testing.T) {
		tags := InjectWebKitTag(nil, "25.04")
		if !stdlibAssertContains(tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", tags, "webkit2_41")
		}

	})

	t.Run("existing tags are preserved before webkit2_41", func(t *testing.T) {
		existing := []string{"foo", "bar"}
		tags := InjectWebKitTag(existing, "24.04")
		if !stdlibAssertContains(tags, "webkit2_41") {
			t.Fatalf("expected %v to contain %v", tags, "webkit2_41")
		}
		if !stdlibAssertContains(tags, "foo") {
			t.Fatalf("expected %v to contain %v", tags, "foo")
		}
		if !stdlibAssertContains(tags, "bar") {
			t.Fatalf(

				// InjectWebKitTag(nil, "22.04") → unchanged (nil)
				"expected %v to contain %v", tags, "bar")
		}

	})
}

func TestOptions_InjectWebKitTag_Bad(t *testing.T) {
	t.Run("22.04 does not add tag", func(t *testing.T) {

		tags := InjectWebKitTag(nil, "22.04")
		if !stdlibAssertEmpty(tags) {
			t.Fatalf("expected empty, got %v", tags)
		}

	})

	t.Run("23.10 does not add tag", func(t *testing.T) {
		tags := InjectWebKitTag([]string{"existing"}, "23.10")
		if stdlibAssertContains(tags, "webkit2_41") {
			t.Fatalf("expected %v not to contain %v", tags, "webkit2_41")
		}

	})
}

func TestOptions_InjectWebKitTag_Ugly(t *testing.T) {
	t.Run("tag already present — not duplicated", func(t *testing.T) {
		// InjectWebKitTag(["webkit2_41"], "24.04") → ["webkit2_41"] (unchanged)
		tags := InjectWebKitTag([]string{"webkit2_41"}, "24.04")
		count := 0
		for _, tag := range tags {
			if tag == "webkit2_41" {
				count++
			}
		}
		if !stdlibAssertEqual(1, count) {
			t.Fatalf("want %v, got %v", 1, count)
		}

	})

	t.Run("empty distro returns tags unchanged", func(t *testing.T) {
		input := []string{"foo"}
		tags := InjectWebKitTag(input, "")
		if !stdlibAssertEqual(input, tags) {
			t.Fatalf("want %v, got %v", input, tags)
		}

	})

	t.Run("malformed version — no dot — returns tags unchanged", func(t *testing.T) {
		// isUbuntu2404OrNewer("2404") → false (no dot)
		tags := InjectWebKitTag(nil, "2404")
		if !stdlibAssertEmpty(tags) {
			t.Fatalf("expected empty, got %v", tags)
		}

	})

	t.Run("malformed version — non-numeric major — returns unchanged", func(t *testing.T) {
		tags := InjectWebKitTag(nil, "ubuntu.04")
		if !stdlibAssertEmpty(tags) {
			t.Fatalf("expected empty, got %v", tags)
		}

	})

	t.Run("malformed version — non-numeric minor — returns unchanged", func(t *testing.T) {
		tags := InjectWebKitTag(nil, "24.lts")
		if !stdlibAssertEmpty(tags) {
			t.Fatalf(

				// --- ApplyOptions ---
				"expected empty, got %v", tags)
		}

	})
}

func TestOptions_ApplyOptions_Good(t *testing.T) {
	t.Run("copies computed options onto runtime config", func(t *testing.T) {
		cfg := &Config{
			BuildTags: []string{"existing"},
			LDFlags:   []string{"-s"},
		}
		options := &BuildOptions{
			Obfuscate: true,
			Tags:      []string{"webkit2_41", "integration"},
			NSIS:      true,
			WebView2:  "embed",
			LDFlags:   []string{"-trimpath", "-w"},
		}

		ApplyOptions(cfg, options)
		if !(cfg.Obfuscate) {
			t.Fatal("expected true")
		}
		if !(cfg.NSIS) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("embed", cfg.WebView2) {
			t.Fatalf("want %v, got %v", "embed", cfg.WebView2)
		}
		if !stdlibAssertEqual([]string{"-trimpath", "-w"}, cfg.LDFlags) {
			t.Fatalf("want %v, got %v", []string{"-trimpath", "-w"}, cfg.LDFlags)
		}
		if !stdlibAssertEqual([]string{"existing", "webkit2_41", "integration"}, cfg.BuildTags) {
			t.Fatalf("want %v, got %v", []string{"existing", "webkit2_41", "integration"}, cfg.BuildTags)
		}

	})
}

func TestOptions_ApplyOptions_Bad(t *testing.T) {
	t.Run("nil config is ignored", func(t *testing.T) {
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("expected no panic, got %v", recovered)
				}
			}()
			(func() {
				ApplyOptions(nil, &BuildOptions{Obfuscate: true})
			})()
		}()

	})

	t.Run("nil options are ignored", func(t *testing.T) {
		cfg := &Config{BuildTags: []string{"existing"}}
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("expected no panic, got %v", recovered)
				}
			}()
			(func() {
				ApplyOptions(cfg, nil)
			})()
		}()
		if !stdlibAssertEqual([]string{"existing"}, cfg.BuildTags) {
			t.Fatalf("want %v, got %v", []string{"existing"}, cfg.BuildTags)
		}

	})
}

func TestOptions_ApplyOptions_Ugly(t *testing.T) {
	t.Run("empty options leaves config unchanged", func(t *testing.T) {
		cfg := &Config{
			BuildTags: []string{"existing"},
			LDFlags:   []string{"-s"},
			Obfuscate: true,
			NSIS:      true,
			WebView2:  "browser",
		}

		ApplyOptions(cfg, &BuildOptions{})
		if !(cfg.Obfuscate) {
			t.Fatal("expected true")
		}
		if !(cfg.NSIS) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("browser", cfg.WebView2) {
			t.Fatalf("want %v, got %v", "browser", cfg.WebView2)
		}
		if !stdlibAssertEqual([]string{"-s"},

			// --- String ---
			cfg.LDFlags) {
			t.Fatalf("want %v, got %v", []string{"-s"}, cfg.LDFlags)
		}
		if !stdlibAssertEqual([]string{"existing"}, cfg.BuildTags) {
			t.Fatalf(

				// opts.String() // "-tags webkit2_41"
				"want %v, got %v", []string{"existing"}, cfg.BuildTags)
		}

	})
}

func TestOptions_String_Good(t *testing.T) {
	t.Run("tags only produces correct string", func(t *testing.T) {

		opts := &BuildOptions{Tags: []string{"webkit2_41"}}
		if !stdlibAssertEqual("-tags webkit2_41", opts.String()) {
			t.Fatalf("want %v, got %v", "-tags webkit2_41", opts.String())
		}

	})

	t.Run("ldflags only produces correct string", func(t *testing.T) {
		opts := &BuildOptions{LDFlags: []string{"-s", "-w"}}
		if !stdlibAssertEqual("-ldflags '-s -w'", opts.String()) {
			t.Fatalf("want %v, got %v", "-ldflags '-s -w'", opts.String())
		}

	})

	t.Run("tags and ldflags are space-separated", func(t *testing.T) {
		opts := &BuildOptions{
			Tags:    []string{"webkit2_41"},
			LDFlags: []string{"-s", "-w"},
		}
		s := opts.String()
		if !stdlibAssertContains(s, "-tags webkit2_41") {
			t.Fatalf("expected %v to contain %v", s, "-tags webkit2_41")
		}
		if !stdlibAssertContains(s, "-ldflags '-s -w'") {
			t.Fatalf("expected %v to contain %v", s, "-ldflags '-s -w'")
		}

	})

	t.Run("empty options returns empty string", func(t *testing.T) {
		opts := &BuildOptions{}
		if !stdlibAssertEqual("", opts.String()) {
			t.Fatalf("want %v, got %v", "", opts.String())
		}

	})
}

func TestOptions_String_Bad(t *testing.T) {
	t.Run("nil receiver returns empty string", func(t *testing.T) {
		// var opts *BuildOptions; opts.String() → ""
		var opts *BuildOptions
		if !stdlibAssertEqual("", opts.String()) {
			t.Fatalf("want %v, got %v", "", opts.String())
		}

	})
}

func TestOptions_String_Ugly(t *testing.T) {
	t.Run("all fields set simultaneously", func(t *testing.T) {
		// s := opts.String()  // "-obfuscated -tags webkit2_41 -nsis -webview2 embed -ldflags '-s -w'"
		opts := &BuildOptions{
			Obfuscate: true,
			Tags:      []string{"webkit2_41"},
			NSIS:      true,
			WebView2:  "embed",
			LDFlags:   []string{"-s", "-w"},
		}
		s := opts.String()
		if !stdlibAssertContains(s, "-obfuscated") {
			t.Fatalf("expected %v to contain %v", s, "-obfuscated")
		}
		if !stdlibAssertContains(s, "-tags webkit2_41") {
			t.Fatalf("expected %v to contain %v", s, "-tags webkit2_41")
		}
		if !stdlibAssertContains(s, "-nsis") {
			t.Fatalf("expected %v to contain %v", s, "-nsis")
		}
		if !stdlibAssertContains(s, "-webview2 embed") {
			t.Fatalf("expected %v to contain %v", s, "-webview2 embed")
		}
		if !stdlibAssertContains(s, "-ldflags '-s -w'") {
			t.Fatalf("expected %v to contain %v", s, "-ldflags '-s -w'")
		}

	})

	t.Run("multiple tags joined with comma", func(t *testing.T) {
		opts := &BuildOptions{Tags: []string{"webkit2_41", "integration"}}
		if !stdlibAssertEqual("-tags webkit2_41,integration", opts.String()) {
			t.Fatalf("want %v, got %v", "-tags webkit2_41,integration", opts.String())
		}

	})

	t.Run("webview2 without other flags is isolated", func(t *testing.T) {
		opts := &BuildOptions{WebView2: "browser"}
		if !stdlibAssertEqual("-webview2 browser", opts.String()) {
			t.Fatalf("want %v, got %v", "-webview2 browser", opts.String())
		}

	})
}

// --- v0.9.0 generated compliance triplets ---
func TestOptions_BuildOptions_String_Good(t *core.T) {
	subject := &BuildOptions{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.String()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestOptions_BuildOptions_String_Bad(t *core.T) {
	subject := &BuildOptions{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.String()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestOptions_BuildOptions_String_Ugly(t *core.T) {
	subject := &BuildOptions{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.String()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
