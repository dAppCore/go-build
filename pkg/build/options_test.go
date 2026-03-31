package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- ComputeOptions ---

func TestOptions_ComputeOptions_Good(t *testing.T) {
	t.Run("normal config produces correct options", func(t *testing.T) {
		cfg := &BuildConfig{
			Build: Build{
				Obfuscate: true,
				NSIS:      true,
				WebView2:  "embed",
				LDFlags:   []string{"-s", "-w"},
			},
		}
		discovery := &DiscoveryResult{
			Types:        []ProjectType{ProjectTypeWails},
			PrimaryStack: "wails",
			Distro:       "24.04",
		}

		opts := ComputeOptions(cfg, discovery)

		assert.NotNil(t, opts)
		assert.True(t, opts.Obfuscate)
		assert.True(t, opts.NSIS)
		assert.Equal(t, "embed", opts.WebView2)
		assert.Equal(t, []string{"-s", "-w"}, opts.LDFlags)
		// webkit2_41 injected for 24.04
		assert.Contains(t, opts.Tags, "webkit2_41")
	})

	t.Run("discovery with non-Ubuntu distro leaves tags empty", func(t *testing.T) {
		cfg := &BuildConfig{
			Build: Build{
				LDFlags: []string{"-s"},
			},
		}
		discovery := &DiscoveryResult{
			Distro: "22.04",
		}

		opts := ComputeOptions(cfg, discovery)

		assert.NotNil(t, opts)
		assert.Empty(t, opts.Tags)
	})

	t.Run("discovery with 25.10 distro injects webkit tag", func(t *testing.T) {
		opts := ComputeOptions(&BuildConfig{}, &DiscoveryResult{Distro: "25.10"})
		assert.Contains(t, opts.Tags, "webkit2_41")
	})
}

func TestOptions_ComputeOptions_Bad(t *testing.T) {
	t.Run("nil config returns safe defaults", func(t *testing.T) {
		discovery := &DiscoveryResult{Distro: "24.04"}

		opts := ComputeOptions(nil, discovery)

		assert.NotNil(t, opts)
		assert.False(t, opts.Obfuscate)
		assert.False(t, opts.NSIS)
		assert.Empty(t, opts.WebView2)
		assert.Empty(t, opts.LDFlags)
		// webkit2_41 still injected from discovery
		assert.Contains(t, opts.Tags, "webkit2_41")
	})

	t.Run("nil discovery skips webkit injection", func(t *testing.T) {
		cfg := &BuildConfig{
			Build: Build{Obfuscate: true},
		}

		opts := ComputeOptions(cfg, nil)

		assert.NotNil(t, opts)
		assert.True(t, opts.Obfuscate)
		assert.Empty(t, opts.Tags)
	})

	t.Run("both nil returns empty options", func(t *testing.T) {
		opts := ComputeOptions(nil, nil)

		assert.NotNil(t, opts)
		assert.False(t, opts.Obfuscate)
		assert.False(t, opts.NSIS)
		assert.Empty(t, opts.Tags)
		assert.Empty(t, opts.LDFlags)
	})
}

func TestOptions_ComputeOptions_Ugly(t *testing.T) {
	t.Run("duplicate tags from deduplication", func(t *testing.T) {
		// Seed webkit2_41 before discovery also injects it
		cfg := &BuildConfig{}
		discovery := &DiscoveryResult{Distro: "24.04"}

		opts := ComputeOptions(cfg, discovery)

		// Even though InjectWebKitTag is called once, deduplication must hold
		count := 0
		for _, tag := range opts.Tags {
			if tag == "webkit2_41" {
				count++
			}
		}
		assert.Equal(t, 1, count, "webkit2_41 must appear exactly once")
	})

	t.Run("empty distro in discovery produces no webkit tag", func(t *testing.T) {
		opts := ComputeOptions(&BuildConfig{}, &DiscoveryResult{Distro: ""})
		assert.Empty(t, opts.Tags)
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
		discovery := &DiscoveryResult{Distro: "24.04"}

		opts := ComputeOptions(cfg, discovery)

		assert.True(t, opts.Obfuscate)
		assert.True(t, opts.NSIS)
		assert.Equal(t, "download", opts.WebView2)
		assert.Equal(t, []string{"-s", "-w", "-X main.version=v1.0.0"}, opts.LDFlags)
		assert.Contains(t, opts.Tags, "webkit2_41")
	})
}

// --- InjectWebKitTag ---

func TestOptions_InjectWebKitTag_Good(t *testing.T) {
	t.Run("24.04 adds webkit2_41", func(t *testing.T) {
		// InjectWebKitTag(tags, "24.04") → ["webkit2_41"]
		tags := InjectWebKitTag(nil, "24.04")
		assert.Equal(t, []string{"webkit2_41"}, tags)
	})

	t.Run("24.10 adds webkit2_41", func(t *testing.T) {
		tags := InjectWebKitTag([]string{}, "24.10")
		assert.Contains(t, tags, "webkit2_41")
	})

	t.Run("25.04 adds webkit2_41", func(t *testing.T) {
		tags := InjectWebKitTag(nil, "25.04")
		assert.Contains(t, tags, "webkit2_41")
	})

	t.Run("existing tags are preserved before webkit2_41", func(t *testing.T) {
		existing := []string{"foo", "bar"}
		tags := InjectWebKitTag(existing, "24.04")
		assert.Contains(t, tags, "webkit2_41")
		assert.Contains(t, tags, "foo")
		assert.Contains(t, tags, "bar")
	})
}

func TestOptions_InjectWebKitTag_Bad(t *testing.T) {
	t.Run("22.04 does not add tag", func(t *testing.T) {
		// InjectWebKitTag(nil, "22.04") → unchanged (nil)
		tags := InjectWebKitTag(nil, "22.04")
		assert.Empty(t, tags)
	})

	t.Run("23.10 does not add tag", func(t *testing.T) {
		tags := InjectWebKitTag([]string{"existing"}, "23.10")
		assert.NotContains(t, tags, "webkit2_41")
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
		assert.Equal(t, 1, count)
	})

	t.Run("empty distro returns tags unchanged", func(t *testing.T) {
		input := []string{"foo"}
		tags := InjectWebKitTag(input, "")
		assert.Equal(t, input, tags)
	})

	t.Run("malformed version — no dot — returns tags unchanged", func(t *testing.T) {
		// isUbuntu2404OrNewer("2404") → false (no dot)
		tags := InjectWebKitTag(nil, "2404")
		assert.Empty(t, tags)
	})

	t.Run("malformed version — non-numeric major — returns unchanged", func(t *testing.T) {
		tags := InjectWebKitTag(nil, "ubuntu.04")
		assert.Empty(t, tags)
	})

	t.Run("malformed version — non-numeric minor — returns unchanged", func(t *testing.T) {
		tags := InjectWebKitTag(nil, "24.lts")
		assert.Empty(t, tags)
	})
}

// --- String ---

func TestOptions_String_Good(t *testing.T) {
	t.Run("tags only produces correct string", func(t *testing.T) {
		// opts.String() // "-tags webkit2_41"
		opts := &BuildOptions{Tags: []string{"webkit2_41"}}
		assert.Equal(t, "-tags webkit2_41", opts.String())
	})

	t.Run("ldflags only produces correct string", func(t *testing.T) {
		opts := &BuildOptions{LDFlags: []string{"-s", "-w"}}
		assert.Equal(t, "-ldflags '-s -w'", opts.String())
	})

	t.Run("tags and ldflags are space-separated", func(t *testing.T) {
		opts := &BuildOptions{
			Tags:    []string{"webkit2_41"},
			LDFlags: []string{"-s", "-w"},
		}
		s := opts.String()
		assert.Contains(t, s, "-tags webkit2_41")
		assert.Contains(t, s, "-ldflags '-s -w'")
	})

	t.Run("empty options returns empty string", func(t *testing.T) {
		opts := &BuildOptions{}
		assert.Equal(t, "", opts.String())
	})
}

func TestOptions_String_Bad(t *testing.T) {
	t.Run("nil receiver returns empty string", func(t *testing.T) {
		// var opts *BuildOptions; opts.String() → ""
		var opts *BuildOptions
		assert.Equal(t, "", opts.String())
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
		assert.Contains(t, s, "-obfuscated")
		assert.Contains(t, s, "-tags webkit2_41")
		assert.Contains(t, s, "-nsis")
		assert.Contains(t, s, "-webview2 embed")
		assert.Contains(t, s, "-ldflags '-s -w'")
	})

	t.Run("multiple tags joined with comma", func(t *testing.T) {
		opts := &BuildOptions{Tags: []string{"webkit2_41", "integration"}}
		assert.Equal(t, "-tags webkit2_41,integration", opts.String())
	})

	t.Run("webview2 without other flags is isolated", func(t *testing.T) {
		opts := &BuildOptions{WebView2: "browser"}
		assert.Equal(t, "-webview2 browser", opts.String())
	})
}
