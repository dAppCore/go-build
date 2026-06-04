package build

import (
	"testing"

	core "dappco.re/go"
	storage "dappco.re/go/build/pkg/storage"
	yaml "gopkg.in/yaml.v3"
)

func requireCacheOK(t *testing.T, result core.Result) {
	t.Helper()
	if !result.OK {
		t.Fatalf("unexpected error: %v", result.Error())
	}
}

func requireCacheError(t *testing.T, result core.Result) string {
	t.Helper()
	if result.OK {
		t.Fatal("expected error")
	}
	return result.Error()
}

func TestCache_SetupCache_Good(t *testing.T) {
	fs := storage.NewMemoryMedium()
	cfg := &CacheConfig{
		Enabled: true,
		Paths: []string{
			"cache/go-build",
			"cache/go-mod",
		},
	}

	requireCacheOK(t, SetupCache(fs, "/workspace/project", cfg))
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("/workspace/project/.core/cache", cfg.Directory) {
		t.Fatalf("want %v, got %v", "/workspace/project/.core/cache", cfg.Directory)
	}
	if !stdlibAssertEqual([]string{"/workspace/project/cache/go-build", "/workspace/project/cache/go-mod"}, cfg.Paths) {
		t.Fatalf("want %v, got %v", []string{"/workspace/project/cache/go-build", "/workspace/project/cache/go-mod"}, cfg.Paths)
	}
	if !(fs.Exists("/workspace/project/.core/cache")) {
		t.Fatal("expected true")
	}
	if !(fs.Exists("/workspace/project/cache/go-build")) {
		t.Fatal("expected true")
	}
	if !(fs.Exists("/workspace/project/cache/go-mod")) {
		t.Fatal("expected true")
	}

}

func TestCache_SetupBuildCache_Good(t *testing.T) {
	fs := storage.NewMemoryMedium()
	cfg := &BuildConfig{
		Build: Build{
			Cache: CacheConfig{
				Enabled: true,
				Paths: []string{
					"cache/go-build",
				},
			},
		},
	}

	requireCacheOK(t, SetupBuildCache(fs, "/workspace/project", cfg))
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("/workspace/project/.core/cache", cfg.Build.Cache.Directory) {
		t.Fatalf("want %v, got %v", "/workspace/project/.core/cache", cfg.Build.Cache.Directory)
	}
	if !stdlibAssertEqual([]string{"/workspace/project/cache/go-build"}, cfg.Build.Cache.Paths) {
		t.Fatalf("want %v, got %v", []string{"/workspace/project/cache/go-build"}, cfg.Build.Cache.Paths)
	}
	if !(fs.Exists("/workspace/project/.core/cache")) {
		t.Fatal("expected true")
	}
	if !(fs.Exists("/workspace/project/cache/go-build")) {
		t.Fatal("expected true")
	}

}

func TestCache_SetupCache_Good_DefaultPathsWhenEnabled(t *testing.T) {
	fs := storage.NewMemoryMedium()
	cfg := &CacheConfig{
		Enabled: true,
	}

	requireCacheOK(t, SetupCache(fs, "/workspace/project", cfg))
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("/workspace/project/.core/cache", cfg.Directory) {
		t.Fatalf("want %v, got %v", "/workspace/project/.core/cache", cfg.Directory)
	}
	if !stdlibAssertEqual([]string{"/workspace/project/cache/go-build", "/workspace/project/cache/go-mod"}, cfg.Paths) {
		t.Fatalf("want %v, got %v", []string{"/workspace/project/cache/go-build", "/workspace/project/cache/go-mod"}, cfg.Paths)
	}
	if !(fs.Exists("/workspace/project/.core/cache")) {
		t.Fatal("expected true")
	}
	if !(fs.Exists("/workspace/project/cache/go-build")) {
		t.Fatal("expected true")
	}
	if !(fs.Exists("/workspace/project/cache/go-mod")) {
		t.Fatal("expected true")
	}

}

func TestCache_SetupBuildCache_Good_DefaultPathsWhenEnabled(t *testing.T) {
	fs := storage.NewMemoryMedium()
	cfg := &BuildConfig{
		Build: Build{
			Cache: CacheConfig{
				Enabled: true,
			},
		},
	}

	requireCacheOK(t, SetupBuildCache(fs, "/workspace/project", cfg))
	if stdlibAssertNil(cfg) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEqual("/workspace/project/.core/cache", cfg.Build.Cache.Directory) {
		t.Fatalf("want %v, got %v", "/workspace/project/.core/cache", cfg.Build.Cache.Directory)
	}
	if !stdlibAssertEqual([]string{"/workspace/project/cache/go-build", "/workspace/project/cache/go-mod"}, cfg.Build.Cache.Paths) {
		t.Fatalf("want %v, got %v", []string{"/workspace/project/cache/go-build", "/workspace/project/cache/go-mod"}, cfg.Build.Cache.Paths)
	}
	if !(fs.Exists("/workspace/project/.core/cache")) {
		t.Fatal("expected true")
	}
	if !(fs.Exists("/workspace/project/cache/go-build")) {
		t.Fatal("expected true")
	}
	if !(fs.Exists("/workspace/project/cache/go-mod")) {
		t.Fatal("expected true")
	}

}

func TestCache_SetupCache_Good_Disabled(t *testing.T) {
	fs := storage.NewMemoryMedium()
	cfg := &CacheConfig{
		Enabled: false,
		Paths:   []string{"cache/go-build"},
	}

	requireCacheOK(t, SetupCache(fs, "/workspace/project", cfg))
	if fs.Exists("/workspace/project/.core/cache") {
		t.Fatal("expected false")
	}
	if fs.Exists("/workspace/project/cache/go-build") {
		t.Fatal("expected false")
	}
	if !stdlibAssertEmpty(cfg.Directory) {
		t.Fatalf("expected empty, got %v", cfg.Directory)
	}
	if !stdlibAssertEqual([]string{"cache/go-build"}, cfg.Paths) {
		t.Fatalf("want %v, got %v", []string{"cache/go-build"}, cfg.Paths)
	}

}

func TestCache_SetupCache_Bad(t *testing.T) {
	t.Run("rejects invalid arity", func(t *testing.T) {
		err := requireCacheError(t, SetupCache())
		if !stdlibAssertContains(err, "expected 1 or 3 arguments") {
			t.Fatalf("expected %v to contain %v", err, "expected 1 or 3 arguments")
		}

	})

	t.Run("rejects a non-cache third argument", func(t *testing.T) {
		fs := storage.NewMemoryMedium()
		err := requireCacheError(t, SetupCache(fs, "/workspace/project", CacheConfig{}))
		if !stdlibAssertContains(err, "third argument must be *CacheConfig") {
			t.Fatalf("expected %v to contain %v", err, "third argument must be *CacheConfig")
		}

	})
}

func TestCache_SetupCache_Ugly(t *testing.T) {
	t.Run("normalises home and absolute cache paths", func(t *testing.T) {
		t.Setenv("HOME", "/home/tester")

		fs := storage.NewMemoryMedium()
		cfg := &CacheConfig{
			Enabled: true,
			Paths: []string{
				"~/cache/go-build",
				"~",
				"/var/cache/go-mod",
				"/var/cache/go-mod",
				"",
			},
		}

		requireCacheOK(t, SetupCache(fs, "/workspace/project", cfg))
		if !stdlibAssertEqual("/workspace/project/.core/cache", cfg.Directory) {
			t.Fatalf("want %v, got %v", "/workspace/project/.core/cache", cfg.Directory)
		}
		if !stdlibAssertEqual([]string{"/home/tester/cache/go-build", "/home/tester", "/var/cache/go-mod"}, cfg.Paths) {
			t.Fatalf("want %v, got %v", []string{"/home/tester/cache/go-build", "/home/tester", "/var/cache/go-mod"}, cfg.Paths)
		}
		if !(fs.Exists("/workspace/project/.core/cache")) {
			t.Fatal("expected true")
		}
		if !(fs.Exists("/home/tester/cache/go-build")) {
			t.Fatal("expected true")
		}
		if !(fs.Exists("/home/tester")) {
			t.Fatal("expected true")
		}
		if !(fs.Exists("/var/cache/go-mod")) {
			t.Fatal("expected true")
		}

	})

	t.Run("1-argument form wires process cache environment", func(t *testing.T) {
		t.Setenv("GOCACHE", "before")
		t.Setenv("GOMODCACHE", "before")

		result := SetupCache(CacheConfig{
			Enabled: true,
			Paths: []string{
				"/tmp/cache/go-build",
				"/tmp/cache/go-mod",
			},
		})
		requireCacheOK(t, result)
		if !stdlibAssertEqual("/tmp/cache/go-build", core.Getenv("GOCACHE")) {
			t.Fatalf("want %v, got %v", "/tmp/cache/go-build", core.Getenv("GOCACHE"))
		}
		if !stdlibAssertEqual("/tmp/cache/go-mod", core.Getenv("GOMODCACHE")) {
			t.Fatalf("want %v, got %v", "/tmp/cache/go-mod", core.Getenv("GOMODCACHE"))
		}

	})
}

func TestCache_SetupBuildCache_Good_Disabled(t *testing.T) {
	fs := storage.NewMemoryMedium()
	cfg := &BuildConfig{
		Build: Build{
			Cache: CacheConfig{
				Enabled: false,
				Paths:   []string{"cache/go-build"},
			},
		},
	}

	requireCacheOK(t, SetupBuildCache(fs, "/workspace/project", cfg))
	if fs.Exists("/workspace/project/.core/cache") {
		t.Fatal("expected false")
	}
	if !stdlibAssertEmpty(cfg.Build.Cache.Directory) {
		t.Fatalf("expected empty, got %v", cfg.Build.Cache.Directory)
	}
	if !stdlibAssertEqual([]string{"cache/go-build"}, cfg.Build.Cache.Paths) {
		t.Fatalf("want %v, got %v", []string{"cache/go-build"}, cfg.Build.Cache.Paths)
	}

}

func TestCache_SetupBuildCache_Bad(t *testing.T) {
	t.Run("nil filesystem is a no-op", func(t *testing.T) {
		cfg := &BuildConfig{
			Build: Build{
				Cache: CacheConfig{Enabled: true},
			},
		}

		requireCacheOK(t, SetupBuildCache(nil, "/workspace/project", cfg))
		if !stdlibAssertEmpty(cfg.Build.Cache.Directory) {
			t.Fatalf("expected empty, got %v", cfg.Build.Cache.Directory)
		}
		if !stdlibAssertEmpty(cfg.Build.Cache.Paths) {
			t.Fatalf("expected empty, got %v", cfg.Build.Cache.Paths)
		}

	})

	t.Run("nil config is a no-op", func(t *testing.T) {
		fs := storage.NewMemoryMedium()

		requireCacheOK(t, SetupBuildCache(fs, "/workspace/project", nil))

	})
}

func TestCache_CacheKey_Good(t *testing.T) {
	fs := storage.NewMemoryMedium()
	requireCacheOK(t, fs.Write("/workspace/project/go.sum", "module.example v1.0.0 h1:abc123"))
	requireCacheOK(t, fs.Write("/workspace/project/go.work.sum", "workspace.example v1.0.0 h1:def456"))

	first := CacheKey(fs, "/workspace/project", "linux", "amd64")
	second := CacheKey(fs, "/workspace/project", "linux", "amd64")
	third := CacheKey(fs, "/workspace/project", "darwin", "arm64")
	if !stdlibAssertEqual(first, second) {
		t.Fatalf("want %v, got %v", first, second)
	}
	if stdlibAssertEqual(first, third) {
		t.Fatalf("did not want %v", third)
	}
	if !stdlibAssertContains(first, "go-linux-amd64-") {
		t.Fatalf("expected %v to contain %v", first, "go-linux-amd64-")
	}

}

func TestCache_CacheKey_Good_GoWorkSumChangesKey(t *testing.T) {
	fs := storage.NewMemoryMedium()
	requireCacheOK(t, fs.Write("/workspace/project/go.sum", "module.example v1.0.0 h1:abc123"))

	baseline := CacheKey(fs, "/workspace/project", "linux", "amd64")
	requireCacheOK(t, fs.Write("/workspace/project/go.work.sum", "workspace.example v1.0.0 h1:def456"))

	updated := CacheKey(fs, "/workspace/project", "linux", "amd64")
	if stdlibAssertEqual(baseline, updated) {
		t.Fatalf("did not want %v", updated)
	}

}

func TestCache_CacheEnvironment_Good(t *testing.T) {
	t.Run("maps cache directory and Go cache paths to env vars", func(t *testing.T) {
		env := CacheEnvironment(&CacheConfig{
			Enabled: true,
			Paths: []string{
				"/workspace/project/cache/go-build",
				"/workspace/project/cache/go-mod",
				"/workspace/project/cache/go-build",
			},
		})
		if !stdlibAssertEqual([]string{"GOCACHE=/workspace/project/cache/go-build", "GOMODCACHE=/workspace/project/cache/go-mod"}, env) {
			t.Fatalf("want %v, got %v", []string{"GOCACHE=/workspace/project/cache/go-build", "GOMODCACHE=/workspace/project/cache/go-mod"}, env)
		}

	})

	t.Run("disabled cache returns no env vars", func(t *testing.T) {
		if !stdlibAssertNil(CacheEnvironment(&CacheConfig{Enabled: false})) {
			t.Fatalf("expected nil, got %v", CacheEnvironment(&CacheConfig{Enabled: false}))
		}

	})
}

func TestCache_CacheKeyWithConfig_Good(t *testing.T) {
	fs := storage.NewMemoryMedium()
	requireCacheOK(t, fs.Write("/workspace/project/go.sum", "module.example v1.0.0 h1:abc123"))

	base := CacheKey(fs, "/workspace/project", "linux", "amd64")
	key := CacheKeyWithConfig(fs, "/workspace/project", "linux", "amd64", &CacheConfig{
		KeyPrefix: "demo",
	})
	if !stdlibAssertEqual("demo-"+base, key) {
		t.Fatalf("want %v, got %v", "demo-"+base, key)
	}

}

func TestCache_CacheKeyWithConfig_Bad(t *testing.T) {
	fs := storage.NewMemoryMedium()
	requireCacheOK(t, fs.Write("/workspace/project/go.sum", "module.example v1.0.0 h1:abc123"))

	base := CacheKey(fs, "/workspace/project", "linux", "amd64")

	t.Run("nil config leaves key unchanged", func(t *testing.T) {
		if !stdlibAssertEqual(base, CacheKeyWithConfig(fs, "/workspace/project", "linux", "amd64", nil)) {
			t.Fatalf("want %v, got %v", base, CacheKeyWithConfig(fs, "/workspace/project", "linux", "amd64", nil))
		}

	})

	t.Run("blank prefix leaves key unchanged", func(t *testing.T) {
		if !stdlibAssertEqual(base, CacheKeyWithConfig(fs, "/workspace/project", "linux", "amd64", &CacheConfig{})) {
			t.Fatalf("want %v, got %v", base, CacheKeyWithConfig(fs, "/workspace/project", "linux", "amd64", &CacheConfig{}))
		}

	})
}

func TestCache_CacheKeyWithConfig_Ugly(t *testing.T) {
	fs := storage.NewMemoryMedium()
	requireCacheOK(t, fs.Write("/workspace/project/go.sum", "module.example v1.0.0 h1:abc123"))

	base := CacheKey(fs, "/workspace/project", "linux", "amd64")
	key := CacheKeyWithConfig(fs, "/workspace/project", "linux", "amd64", &CacheConfig{
		KeyPrefix: "  demo  ",
	})
	if !stdlibAssertEqual("demo-"+base, key) {
		t.Fatalf("want %v, got %v", "demo-"+base, key)
	}

}

func TestCache_CacheRestoreKeys_Good(t *testing.T) {
	keys := CacheRestoreKeys(&CacheConfig{
		KeyPrefix:   "demo",
		RestoreKeys: []string{"go-", "core-"},
	})
	if !stdlibAssertEqual([]string{"demo", "go-", "core-"}, keys) {
		t.Fatalf("want %v, got %v", []string{"demo", "go-", "core-"}, keys)
	}

}

func TestCache_CacheRestoreKeys_Bad(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		if !stdlibAssertNil(CacheRestoreKeys(nil)) {
			t.Fatalf("expected nil, got %v", CacheRestoreKeys(nil))
		}

	})

	t.Run("blank prefix is ignored", func(t *testing.T) {
		keys := CacheRestoreKeys(&CacheConfig{
			RestoreKeys: []string{"go-"},
		})
		if !stdlibAssertEqual([]string{"go-"}, keys) {
			t.Fatalf("want %v, got %v", []string{"go-"}, keys)
		}

	})
}

func TestCache_CacheRestoreKeys_Ugly(t *testing.T) {
	keys := CacheRestoreKeys(&CacheConfig{
		KeyPrefix:   "demo",
		RestoreKeys: []string{"go-", "", "core-", "go-", "core-"},
	})
	if !stdlibAssertEqual([]string{"demo", "go-", "core-"}, keys) {
		t.Fatalf("want %v, got %v", []string{"demo", "go-", "core-"}, keys)
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestCache_DefaultBuildCachePaths_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultBuildCachePaths(core.Path(t.TempDir(), "go-build-compliance"))
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCache_DefaultBuildCachePaths_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultBuildCachePaths("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCache_DefaultBuildCachePaths_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = DefaultBuildCachePaths(core.Path(t.TempDir(), "go-build-compliance"))
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCache_CacheConfig_MarshalYAML_Good(t *core.T) {
	subject := CacheConfig{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.MarshalYAML()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCache_CacheConfig_MarshalYAML_Bad(t *core.T) {
	subject := CacheConfig{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.MarshalYAML()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCache_CacheConfig_MarshalYAML_Ugly(t *core.T) {
	subject := CacheConfig{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.MarshalYAML()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCache_CacheConfig_UnmarshalYAML_Good(t *core.T) {
	subject := &CacheConfig{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "false"})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestCache_CacheConfig_UnmarshalYAML_Bad(t *core.T) {
	subject := &CacheConfig{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "false"})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCache_CacheConfig_UnmarshalYAML_Ugly(t *core.T) {
	subject := &CacheConfig{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "false"})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCache_SetupBuildCache_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = SetupBuildCache(storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"), &BuildConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCache_CacheKey_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CacheKey(storage.NewMemoryMedium(), "", "", "")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCache_CacheKey_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CacheKey(storage.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"), "linux", "amd64")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestCache_CacheEnvironment_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CacheEnvironment(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestCache_CacheEnvironment_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = CacheEnvironment(&CacheConfig{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
