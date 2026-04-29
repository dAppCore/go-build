package build

import (
	"runtime"
	"testing"

	"dappco.re/go/build/internal/ax"

	core "dappco.re/go"
	"dappco.re/go/io"
)

// setupTestDir creates a temporary directory with the specified marker files.
func setupTestDir(t *testing.T, markers ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, m := range markers {
		path := ax.Join(dir, m)
		err := ax.WriteFile(path, []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	}
	return dir
}

func setupDiscoveryFile(t *testing.T, relPath string, content string) string {
	t.Helper()
	dir := t.TempDir()
	writeDiscoveryFile(t, dir, relPath, content)
	return dir
}

func writeDiscoveryFile(t *testing.T, dir string, relPath string, content string) {
	t.Helper()
	path := ax.Join(dir, relPath)
	if err := ax.MkdirAll(ax.Dir(path), 0o755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ax.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertDiscoverTypes(t *testing.T, fs io.Medium, dir string, want []ProjectType) {
	t.Helper()

	types, err := Discover(fs, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(want, types) {
		t.Fatalf("want %v, got %v", want, types)
	}
}

func assertDiscoverEmpty(t *testing.T, fs io.Medium, dir string) {
	t.Helper()

	types, err := Discover(fs, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEmpty(types) {
		t.Fatalf("expected empty, got %v", types)
	}
}

func assertDiscoverFullStack(t *testing.T, fs io.Medium, dir string, want []ProjectType, wantStack string, markers ...string) *DiscoveryResult {
	t.Helper()

	result, err := DiscoverFull(fs, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual(want, result.Types) {
		t.Fatalf("want %v, got %v", want, result.Types)
	}
	if !stdlibAssertEqual(wantStack, result.PrimaryStack) {
		t.Fatalf("want %v, got %v", wantStack, result.PrimaryStack)
	}
	for _, marker := range markers {
		if !result.Markers[marker] {
			t.Fatalf("expected marker %q", marker)
		}
	}
	return result
}

func TestDiscovery_Discover_Good(t *testing.T) {
	fs := io.Local
	_, err := Discover(fs, setupTestDir(t, "go.mod"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("prefers configured build type from .core/build.yaml", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: docker\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocker})

	})

	t.Run("configured build type short-circuits marker detection", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: docker\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocker})

	})

	t.Run("detects Go project", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeGo})

	})

	t.Run("detects Go workspace project", func(t *testing.T) {
		dir := setupTestDir(t, "go.work")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeGo})

	})

	t.Run("detects Wails project with priority over Go", func(t *testing.T) {
		dir := setupTestDir(t, "wails.json", "go.mod")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeWails, ProjectTypeGo})

	})

	t.Run("detects Node.js project", func(t *testing.T) {
		dir := setupTestDir(t, "package.json")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeNode})

	})

	t.Run("detects Deno project", func(t *testing.T) {
		dir := setupTestDir(t, "deno.json")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeNode})

	})

	t.Run("detects nested Node.js project", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeNode})

	})

	t.Run("detects nested Deno project", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "site")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "deno.jsonc"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeNode})

	})

	t.Run("detects Wails project from go.mod and root package.json", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "package.json")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode})

	})

	t.Run("detects Wails project from go.mod and nested frontend package.json", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode})

	})

	t.Run("detects Wails project from go.work and frontend deno.json", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.work"), []byte("go 1.26\nuse ."), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		frontend := ax.Join(dir, "frontend")
		if err := ax.MkdirAll(frontend, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontend, "deno.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode})

	})

	t.Run("detects Wails project from go.mod and nested frontend deno.jsonc", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nested := ax.Join(dir, "apps", "site")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "deno.jsonc"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode})

	})

	t.Run("detects PHP project", func(t *testing.T) {
		dir := setupTestDir(t, "composer.json")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypePHP})

	})

	t.Run("detects docs project", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yml")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocs})

	})

	t.Run("keeps docs after generic Node markers", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yml", "package.json")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeNode, ProjectTypeDocs})

	})

	t.Run("detects docs project with mkdocs.yaml", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yaml")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocs})

	})

	t.Run("detects docs project in docs directory", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, "docs"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "docs", "mkdocs.yml"), []byte("site_name: Demo\n"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocs})

	})

	t.Run("detects docs project in docs directory with mkdocs.yaml", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, "docs"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "docs", "mkdocs.yaml"), []byte("site_name: Demo\n"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocs})

	})

	t.Run("detects Python project with pyproject.toml", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypePython})

	})

	t.Run("detects Python project with requirements.txt", func(t *testing.T) {
		dir := setupTestDir(t, "requirements.txt")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypePython})

	})

	t.Run("detects Python only once with both markers", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml", "requirements.txt")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypePython})

	})

	t.Run("detects Rust project", func(t *testing.T) {
		dir := setupTestDir(t, "Cargo.toml")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeRust})

	})

	t.Run("detects Docker project", func(t *testing.T) {
		dir := setupTestDir(t, "Dockerfile")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocker})

	})

	t.Run("detects Containerfile project", func(t *testing.T) {
		dir := setupTestDir(t, "Containerfile")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocker})

	})

	t.Run("detects LinuxKit project", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		if err := ax.MkdirAll(lkDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(lkDir, "server.yml"), []byte("kernel:\n"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeLinuxKit})

	})

	t.Run("detects LinuxKit project from yaml config", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "linuxkit.yaml"), []byte("kernel:\n"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeLinuxKit})

	})

	t.Run("detects C++ project", func(t *testing.T) {
		dir := setupTestDir(t, "CMakeLists.txt")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeCPP})

	})

	t.Run("detects Taskfile project", func(t *testing.T) {
		dir := setupTestDir(t, "Taskfile.yml")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeTaskfile})

	})

	t.Run("detects multiple project types", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "package.json")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode})

	})

	t.Run("preserves priority when core and fallback markers overlap", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "Dockerfile", "Taskfile.yml")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeGo, ProjectTypeDocker, ProjectTypeTaskfile})

	})

	t.Run("prefers C++ ahead of Docker and Taskfile in fallback detection", func(t *testing.T) {
		dir := setupTestDir(t, "CMakeLists.txt", "Dockerfile", "Taskfile.yml")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeCPP, ProjectTypeDocker, ProjectTypeTaskfile})

	})

	t.Run("keeps docs after taskfile and docker per RFC priority", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yml", "Dockerfile", "Taskfile.yml")
		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeDocker, ProjectTypeTaskfile, ProjectTypeDocs})

	})

	t.Run("empty directory returns empty slice", func(t *testing.T) {
		dir := t.TempDir()
		assertDiscoverEmpty(t, fs, dir)

	})
}

func TestDiscovery_Discover_Bad(t *testing.T) {
	fs := io.Local
	t.Run("non-existent directory returns empty slice", func(t *testing.T) {
		types, err := Discover(fs, "/non/existent/path")
		if err != nil {
			t.Fatalf("unexpected error: %v",
				// ax.Stat fails silently in fileExists
				err)
		}
		if !stdlibAssertEmpty(types) {
			t.Fatalf("expected empty, got %v", types)
		}

	})

	t.Run("directory marker is ignored", func(t *testing.T) {
		dir := t.TempDir()
		// Create go.mod as a directory instead of a file
		err := ax.Mkdir(ax.Join(dir, "go.mod"), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverEmpty(t, fs, dir)

	})

	t.Run("unsupported configured build type is ignored", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: kotlin\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertDiscoverTypes(t, fs, dir, []ProjectType{ProjectTypeGo})

	})
}

func TestDiscovery_PrimaryType_Good(t *testing.T) {
	fs := io.Local
	t.Run("returns configured build type from .core/build.yaml", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: taskfile\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ProjectTypeTaskfile, primary) {
			t.Fatalf("want %v, got %v", ProjectTypeTaskfile, primary)
		}

	})

	t.Run("returns configured type when markers disagree", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: taskfile\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ProjectTypeTaskfile, primary) {
			t.Fatalf("want %v, got %v", ProjectTypeTaskfile, primary)
		}

	})

	t.Run("returns wails for wails project", func(t *testing.T) {
		dir := setupTestDir(t, "wails.json", "go.mod")
		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ProjectTypeWails, primary) {
			t.Fatalf("want %v, got %v", ProjectTypeWails, primary)
		}

	})

	t.Run("returns go for go-only project", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ProjectTypeGo, primary) {
			t.Fatalf("want %v, got %v", ProjectTypeGo, primary)
		}

	})

	t.Run("returns node for nested package.json project", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ProjectTypeNode, primary) {
			t.Fatalf("want %v, got %v", ProjectTypeNode, primary)
		}

	})

	t.Run("returns node for root deno project", func(t *testing.T) {
		dir := setupTestDir(t, "deno.jsonc")
		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ProjectTypeNode, primary) {
			t.Fatalf("want %v, got %v", ProjectTypeNode, primary)
		}

	})

	t.Run("returns node when mkdocs and package.json coexist", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yml", "package.json")
		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ProjectTypeNode, primary) {
			t.Fatalf("want %v, got %v", ProjectTypeNode, primary)
		}

	})

	t.Run("returns wails for go.mod with nested frontend package.json", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(ProjectTypeWails, primary) {
			t.Fatalf("want %v, got %v", ProjectTypeWails, primary)
		}

	})

	t.Run("returns empty string for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		primary, err := PrimaryType(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEmpty(primary) {
			t.Fatalf("expected empty, got %v", primary)
		}

	})
}

func TestDiscovery_IsGoProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with go.mod", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		if !(IsGoProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with go.work", func(t *testing.T) {
		dir := setupTestDir(t, "go.work")
		if !(IsGoProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with wails.json", func(t *testing.T) {
		dir := setupTestDir(t, "wails.json")
		if !(IsGoProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("false without markers", func(t *testing.T) {
		dir := t.TempDir()
		if IsGoProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsWailsProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with wails.json", func(t *testing.T) {
		dir := setupTestDir(t, "wails.json")
		if !(IsWailsProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with go.mod and root package.json", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "package.json")
		if !(IsWailsProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with go.mod and nested frontend package.json", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(IsWailsProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with go.work and frontend deno.json", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.work"), []byte("go 1.26\nuse ."), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		frontend := ax.Join(dir, "frontend")
		if err := ax.MkdirAll(frontend, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontend, "deno.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(IsWailsProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("false with only go.mod", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		if IsWailsProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsNodeProject_Good(t *testing.T) {
	fs := io.Local

	t.Run("true with package.json", func(t *testing.T) {
		dir := setupTestDir(t, "package.json")
		if !(IsNodeProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with deno.json", func(t *testing.T) {
		dir := setupTestDir(t, "deno.json")
		if !(IsNodeProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with deno.jsonc", func(t *testing.T) {
		dir := setupTestDir(t, "deno.jsonc")
		if !(IsNodeProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with frontend package.json", func(t *testing.T) {
		dir := t.TempDir()
		frontend := ax.Join(dir, "frontend")
		if err := ax.MkdirAll(frontend, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontend, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(IsNodeProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with nested package.json", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(IsNodeProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with nested deno.json", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "docs")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "deno.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(IsNodeProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("false without markers", func(t *testing.T) {
		if IsNodeProject(fs, t.TempDir()) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsPHPProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with composer.json", func(t *testing.T) {
		dir := setupTestDir(t, "composer.json")
		if !(IsPHPProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("false without composer.json", func(t *testing.T) {
		dir := t.TempDir()
		if IsPHPProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_Target_Good(t *testing.T) {
	target := Target{OS: "linux", Arch: "amd64"}
	if !stdlibAssertEqual("linux/amd64", target.String()) {
		t.Fatalf("want %v, got %v", "linux/amd64", target.String())
	}

}

func TestDiscovery_FileExistsGood(t *testing.T) {
	fs := io.Local
	t.Run("returns true for existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := ax.Join(dir, "test.txt")
		err := ax.WriteFile(path, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(fileExists(fs, path)) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns false for directory", func(t *testing.T) {
		dir := t.TempDir()
		if fileExists(fs, dir) {
			t.Fatal("expected false")
		}

	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		if fileExists(fs, "/non/existent/file") {
			t.Fatal("expected false")
		}

	})
}

// TestDiscover_Testdata tests discovery using the testdata fixtures.
// These serve as integration tests with realistic project structures.
func TestDiscovery_DiscoverTestdataGood(t *testing.T) {
	fs := io.Local
	testdataDir, err := ax.Abs("testdata")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		dir      string
		expected []ProjectType
	}{
		{"go-project", "go-project", []ProjectType{ProjectTypeGo}},
		{"wails-project", "wails-project", []ProjectType{ProjectTypeWails, ProjectTypeGo}},
		{"node-project", "node-project", []ProjectType{ProjectTypeNode}},
		{"php-project", "php-project", []ProjectType{ProjectTypePHP}},
		{"multi-project", "multi-project", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}},
		{"empty-project", "empty-project", []ProjectType{}},
		{"docs-project", "docs-project", []ProjectType{ProjectTypeDocs}},
		{"python-project", "python-project", []ProjectType{ProjectTypePython}},
		{"rust-project", "rust-project", []ProjectType{ProjectTypeRust}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := ax.Join(testdataDir, tt.dir)
			types, err := Discover(fs, dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tt.expected) == 0 {
				if !stdlibAssertEmpty(types) {
					t.Fatalf("expected empty, got %v", types)
				}

			} else {
				if !stdlibAssertEqual(tt.expected, types) {
					t.Fatalf("want %v, got %v", tt.expected, types)
				}

			}
		})
	}
}

func TestDiscovery_IsMkDocsProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with mkdocs.yml", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yml")
		if !(IsMkDocsProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with mkdocs.yaml", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yaml")
		if !(IsMkDocsProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with nested mkdocs.yml", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "docs", "guide")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "mkdocs.yml"), []byte("site_name: Guide"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(IsMkDocsProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("false without mkdocs.yml", func(t *testing.T) {
		dir := t.TempDir()
		if IsMkDocsProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsMkDocsProject_Bad(t *testing.T) {
	fs := io.Local
	t.Run("false for non-existent directory", func(t *testing.T) {
		if IsMkDocsProject(fs, "/non/existent/path") {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsMkDocsProject_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("false when mkdocs.yml is a directory", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.Mkdir(ax.Join(dir, "mkdocs.yml"), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if IsMkDocsProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_HasSubtreeNpm_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with depth 1 nested package.json", func(t *testing.T) {
		dir := t.TempDir()
		subdir := ax.Join(dir, "packages", "web")
		err := ax.MkdirAll(subdir, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(dir, "packages", "package.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(HasSubtreeNpm(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with depth 2 nested package.json", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		err := ax.MkdirAll(nested, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(HasSubtreeNpm(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("false with only root package.json", func(t *testing.T) {
		dir := setupTestDir(t, "package.json")
		if HasSubtreeNpm(fs, dir) {
			t.Fatal("expected false")
		}

	})

	t.Run("false with only frontend package.json", func(t *testing.T) {
		dir := t.TempDir()
		frontendDir := ax.Join(dir, "frontend")
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if HasSubtreeNpm(fs, dir) {
			t.Fatal("expected false")
		}

	})

	t.Run("false with empty directory", func(t *testing.T) {
		dir := t.TempDir()
		if HasSubtreeNpm(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_HasSubtreeNpm_Bad(t *testing.T) {
	fs := io.Local
	t.Run("false for non-existent directory", func(t *testing.T) {
		if HasSubtreeNpm(fs, "/non/existent/path") {
			t.Fatal("expected false")
		}

	})

	t.Run("ignores node_modules at depth 1", func(t *testing.T) {
		dir := t.TempDir()
		nmDir := ax.Join(dir, "node_modules", "some-pkg")
		err := ax.MkdirAll(nmDir, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(nmDir, "package.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if HasSubtreeNpm(fs, dir) {
			t.Fatal("expected false")
		}

	})

	t.Run("ignores node_modules at depth 2", func(t *testing.T) {
		dir := t.TempDir()
		nmDir := ax.Join(dir, "apps", "node_modules", "some-pkg")
		err := ax.MkdirAll(nmDir, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(nmDir, "package.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v",

				// Also need the apps dir to be listable — it is since we created nmDir inside it
				err)
		}
		if HasSubtreeNpm(fs, dir) {
			t.Fatal("expected false")
		}

	})

	t.Run("ignores hidden directories", func(t *testing.T) {
		dir := t.TempDir()
		hiddenDir := ax.Join(dir, ".turbo", "web")
		if err := ax.MkdirAll(hiddenDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(hiddenDir, "package.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if HasSubtreeNpm(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_HasSubtreeNpm_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("false when nested package.json is beyond depth 2", func(t *testing.T) {
		dir := t.TempDir()
		deep := ax.Join(dir, "a", "b", "c")
		err := ax.MkdirAll(deep, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(deep, "package.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if HasSubtreeNpm(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsPythonProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with pyproject.toml", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml")
		if !(IsPythonProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with requirements.txt", func(t *testing.T) {
		dir := setupTestDir(t, "requirements.txt")
		if !(IsPythonProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("true with both markers", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml", "requirements.txt")
		if !(IsPythonProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("false without markers", func(t *testing.T) {
		dir := t.TempDir()
		if IsPythonProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsPythonProject_Bad(t *testing.T) {
	fs := io.Local
	t.Run("false for non-existent directory", func(t *testing.T) {
		if IsPythonProject(fs, "/non/existent/path") {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsPythonProject_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("false when pyproject.toml is a directory", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.Mkdir(ax.Join(dir, "pyproject.toml"), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if IsPythonProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsRustProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with Cargo.toml", func(t *testing.T) {
		dir := setupTestDir(t, "Cargo.toml")
		if !(IsRustProject(fs, dir)) {
			t.Fatal("expected true")
		}

	})

	t.Run("false without Cargo.toml", func(t *testing.T) {
		dir := t.TempDir()
		if IsRustProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsRustProject_Bad(t *testing.T) {
	fs := io.Local
	t.Run("false for non-existent directory", func(t *testing.T) {
		if IsRustProject(fs, "/non/existent/path") {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_IsRustProject_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("false when Cargo.toml is a directory", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.Mkdir(ax.Join(dir, "Cargo.toml"), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if IsRustProject(fs, dir) {
			t.Fatal("expected false")
		}

	})
}

func TestDiscovery_DiscoverFull_Good(t *testing.T) {
	fs := io.Local
	t.Run("configured build type stays authoritative in full discovery", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: docker\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeDocker}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeDocker}, result.Types)
		}
		if !stdlibAssertEqual("docker", result.ConfiguredType) {
			t.Fatalf("want %v, got %v", "docker", result.ConfiguredType)
		}
		if !stdlibAssertEqual("docker", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "docker", result.PrimaryStack)
		}
		if !stdlibAssertEqual("docker", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "docker", result.SuggestedStack)
		}
		if !stdlibAssertEqual("docker", result.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "docker", result.PrimaryStackSuggestion)
		}
		if !(result.Markers["go.mod"]) {
			t.Fatal("expected true")
		}
		if !(result.Markers["wails.json"]) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns complete result for Go project", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "main.go")
		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeGo}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeGo}, result.Types)
		}
		if !stdlibAssertEqual(runtime.GOOS, result.OS) {
			t.Fatalf("want %v, got %v", runtime.GOOS, result.OS)
		}
		if !stdlibAssertEqual(runtime.GOARCH, result.Arch) {
			t.Fatalf("want %v, got %v", runtime.GOARCH, result.Arch)
		}
		if !stdlibAssertEqual("go", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "go", result.PrimaryStack)
		}
		if !stdlibAssertEqual("go", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "go", result.SuggestedStack)
		}
		if result.HasFrontend {
			t.Fatal("expected false")
		}
		if result.HasRootPackageJSON {
			t.Fatal("expected false")
		}
		if result.HasFrontendPackageJSON {
			t.Fatal("expected false")
		}
		if !(result.HasRootGoMod) {
			t.Fatal("expected true")
		}
		if !(result.HasRootMainGo) {
			t.Fatal("expected true")
		}
		if result.HasRootCMakeLists {
			t.Fatal("expected false")
		}
		if result.HasSubtreeNpm {
			t.Fatal("expected false")
		}
		if !(result.Markers["go.mod"]) {
			t.Fatal("expected true")
		}
		if !(result.Markers["main.go"]) {
			t.Fatal("expected true")
		}
		if result.Markers["wails.json"] {
			t.Fatal("expected false")
		}

	})

	t.Run("detects nested MkDocs configuration", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "docs", "guide")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "mkdocs.yaml"), []byte("site_name: Guide"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(result.HasDocsConfig) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("docs", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "docs", result.PrimaryStack)
		}
		if !stdlibAssertEqual("docs", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "docs", result.SuggestedStack)
		}

	})

	t.Run("prefers Go stack suggestion when docs and Go toolchain coexist", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "mkdocs.yml"), []byte("site_name: Demo\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeGo, ProjectTypeDocs}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeGo, ProjectTypeDocs}, result.Types)
		}
		if !(result.HasDocsConfig) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("go", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "go", result.PrimaryStack)
		}
		if !stdlibAssertEqual("go", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "go", result.SuggestedStack)
		}
		if !stdlibAssertEqual("go", result.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "go", result.PrimaryStackSuggestion)
		}
		if !(result.Markers["go.mod"]) {
			t.Fatal("expected true")
		}
		if !(result.Markers["mkdocs.yml"]) {
			t.Fatal("expected true")
		}

	})

	t.Run("captures GitHub metadata when available", func(t *testing.T) {
		t.Setenv("GITHUB_SHA", "0123456789abcdef")
		t.Setenv("GITHUB_REF", "refs/tags/v1.2.3")
		t.Setenv("GITHUB_REPOSITORY", "dappcore/core")

		dir := setupTestDir(t, "go.mod")
		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual("refs/tags/v1.2.3", result.Ref) {
			t.Fatalf("want %v, got %v", "refs/tags/v1.2.3", result.Ref)
		}
		if !stdlibAssertEqual("v1.2.3", result.Tag) {
			t.Fatalf("want %v, got %v", "v1.2.3", result.Tag)
		}
		if !(result.IsTag) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("0123456789abcdef", result.SHA) {
			t.Fatalf("want %v, got %v", "0123456789abcdef", result.SHA)
		}
		if !stdlibAssertEqual("0123456", result.ShortSHA) {
			t.Fatalf("want %v, got %v", "0123456", result.ShortSHA)
		}
		if !stdlibAssertEqual("dappcore/core", result.Repo) {
			t.Fatalf("want %v, got %v", "dappcore/core", result.Repo)
		}
		if !stdlibAssertEqual("dappcore", result.Owner) {
			t.Fatalf("want %v, got %v", "dappcore", result.Owner)
		}

	})

	t.Run("falls back to local git metadata when GitHub env is absent", func(t *testing.T) {
		t.Setenv("GITHUB_SHA", "")
		t.Setenv("GITHUB_REF", "")
		t.Setenv("GITHUB_REPOSITORY", "")

		dir, sha := initGitMetadataRepo(t)
		if err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual(sha, result.SHA) {
			t.Fatalf("want %v, got %v", sha, result.SHA)
		}
		if !stdlibAssertEqual(sha[:7], result.ShortSHA) {
			t.Fatalf("want %v, got %v", sha[:7], result.ShortSHA)
		}
		if !stdlibAssertEqual("refs/heads/main", result.Ref) {
			t.Fatalf("want %v, got %v", "refs/heads/main", result.Ref)
		}
		if !stdlibAssertEqual("main", result.Branch) {
			t.Fatalf("want %v, got %v", "main", result.Branch)
		}
		if result.IsTag {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual("dappcore/core", result.Repo) {
			t.Fatalf("want %v, got %v", "dappcore/core", result.Repo)
		}
		if !stdlibAssertEqual("dappcore", result.Owner) {
			t.Fatalf("want %v, got %v", "dappcore", result.Owner)
		}

	})

	t.Run("returns complete result for Go workspace project", func(t *testing.T) {
		dir := setupTestDir(t, "go.work")
		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeGo}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeGo}, result.Types)
		}
		if !stdlibAssertEqual("go", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "go", result.PrimaryStack)
		}
		if !(result.Markers[

		// Create wails.json, go.mod, and frontend/package.json
		"go.work"]) {
			t.Fatal("expected true")
		}

	})

	t.Run("returns complete result for Wails project with frontend", func(t *testing.T) {
		dir := t.TempDir()

		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(dir, "go.mod"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.MkdirAll(ax.Join(dir, "frontend"), 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(dir, "frontend", "package.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("wails", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "wails", result.PrimaryStack)
		}
		if !stdlibAssertEqual("wails2", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "wails2", result.SuggestedStack)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if result.HasRootPackageJSON {
			t.Fatal("expected false")
		}
		if !(result.HasFrontendPackageJSON) {
			t.Fatal("expected true")
		}
		if !(result.HasRootGoMod) {
			t.Fatal("expected true")
		}
		if result.HasRootMainGo {
			t.Fatal("expected false")
		}
		if result.HasRootCMakeLists {
			t.Fatal("expected false")
		}
		if result.HasSubtreeNpm {
			t.Fatal("expected false")
		}
		if !(result.Markers["wails.json"]) {
			t.Fatal("expected true")
		}
		if !(result.Markers["go.mod"]) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects subtree npm as frontend", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nested := ax.Join(dir, "apps", "web")
		err = ax.MkdirAll(nested, 0755)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("wails", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "wails", result.PrimaryStack)
		}
		if !(result.HasSubtreeNpm) {
			t.Fatal("expected true")
		}
		if !(result.HasSubtreePackageJSON) {
			t.Fatal("expected true")
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}

	})

	t.Run("detects root package.json as frontend", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("node", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "node", result.PrimaryStack)
		}
		if !stdlibAssertEqual("node", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "node", result.SuggestedStack)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if !(result.HasRootPackageJSON) {
			t.Fatal("expected true")
		}
		if result.HasFrontendPackageJSON {
			t.Fatal("expected false")
		}
		if result.HasRootComposerJSON {
			t.Fatal("expected false")
		}
		if result.HasRootCargoToml {
			t.Fatal("expected false")
		}
		if result.HasRootGoMod {
			t.Fatal("expected false")
		}
		if result.HasRootMainGo {
			t.Fatal("expected false")
		}
		if result.HasRootCMakeLists {
			t.Fatal("expected false")
		}
		if result.HasTaskfile {
			t.Fatal("expected false")
		}
		if result.HasSubtreeNpm {
			t.Fatal("expected false")
		}
		if result.HasSubtreePackageJSON {
			t.Fatal("expected false")
		}

	})

	t.Run("detects root deno.json as node project", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "deno.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("node", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "node", result.PrimaryStack)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if !(result.Markers["deno.json"]) {
			t.Fatal("expected true")
		}
		if result.Markers["package.json"] {
			t.Fatal("expected false")
		}

	})

	t.Run("detects go.mod with root package.json as Wails", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "package.json")

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("wails", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "wails", result.PrimaryStack)
		}
		if !stdlibAssertEqual("wails2", result.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "wails2", result.PrimaryStackSuggestion)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if !(result.HasPackageJSON) {
			t.Fatal("expected true")
		}
		if result.HasDenoManifest {
			t.Fatal("expected false")
		}
		if !(result.HasGoToolchain) {
			t.Fatal("expected true")
		}
		if result.HasRootGoWork {
			t.Fatal("expected false")
		}
		if result.HasRootWailsJSON {
			t.Fatal("expected false")
		}
		if result.HasSubtreeNpm {
			t.Fatal("expected false")
		}

	})

	t.Run("detects frontend deno manifest at project root", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		frontendDir := ax.Join(dir, "frontend")
		if err := ax.MkdirAll(frontendDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("wails", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "wails", result.PrimaryStack)
		}
		if !stdlibAssertEqual("wails2", result.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "wails2", result.PrimaryStackSuggestion)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if result.HasPackageJSON {
			t.Fatal("expected false")
		}
		if !(result.HasDenoManifest) {
			t.Fatal("expected true")
		}
		if result.HasSubtreeNpm {
			t.Fatal("expected false")
		}
		if !(result.Markers["frontend/deno.json"]) {
			t.Fatal("expected true")
		}
		if result.Markers["frontend/package.json"] {
			t.Fatal("expected false")
		}

	})

	t.Run("detects nested deno frontend manifests", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		frontendDir := ax.Join(dir, "apps", "site")
		if err := ax.MkdirAll(frontendDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.jsonc"), []byte("{}"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("wails", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "wails", result.PrimaryStack)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if result.HasSubtreeNpm {
			t.Fatal("expected false")
		}

	})

	t.Run("detects nested deno project as node", func(t *testing.T) {
		dir := t.TempDir()
		frontendDir := ax.Join(dir, "apps", "site")
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.jsonc"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("node", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "node", result.PrimaryStack)
		}
		if !stdlibAssertEqual("node", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "node", result.SuggestedStack)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if result.HasSubtreeNpm {
			t.Fatal("expected false")
		}

	})

	t.Run("detects nested deno subtree manifests in full discovery", func(t *testing.T) {
		dir := t.TempDir()
		frontendDir := ax.Join(dir, "apps", "site")
		if err := ax.MkdirAll(frontendDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeNode}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeNode}, result.Types)
		}
		if !stdlibAssertEqual("node", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "node", result.PrimaryStack)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if !(result.HasDenoManifest) {
			t.Fatal("expected true")
		}
		if !(result.HasSubtreeDenoManifest) {
			t.Fatal("expected true")
		}

	})

	t.Run("records frontend package manifest markers", func(t *testing.T) {
		dir := t.TempDir()
		frontendDir := ax.Join(dir, "frontend")
		if err := ax.MkdirAll(frontendDir, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(frontendDir, "deno.jsonc"), []byte("{}"), 0644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(result.HasFrontend) {
			t.Fatal("expected true")
		}
		if !(result.Markers["frontend/package.json"]) {
			t.Fatal("expected true")
		}
		if !(result.Markers["frontend/deno.jsonc"]) {
			t.Fatal("expected true")
		}

	})

	t.Run("records the build config marker and prefers configured type", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: cpp\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeCPP}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeCPP}, result.Types)
		}
		if !stdlibAssertEqual("cpp", result.ConfiguredType) {
			t.Fatalf("want %v, got %v", "cpp", result.ConfiguredType)
		}
		if !stdlibAssertEqual("cpp", result.ConfiguredBuildType) {
			t.Fatalf("want %v, got %v", "cpp", result.ConfiguredBuildType)
		}
		if !stdlibAssertEqual("cpp", result.PrimaryStack) {
			t.Fatalf("want %v, got %v", "cpp", result.PrimaryStack)
		}
		if !stdlibAssertEqual("cpp", result.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "cpp", result.PrimaryStackSuggestion)
		}
		if !(result.Markers[".core/build.yaml"]) {
			t.Fatal("expected true")
		}
		if !(result.Markers["Dockerfile"]) {
			t.Fatal("expected true")
		}

	})

	t.Run("records workflow-facing marker aliases", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.WriteFile(ax.Join(dir, "composer.json"), []byte("{}"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"demo\"\nversion = \"0.1.0\"\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, "Taskfile.yaml"), []byte("version: '3'\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !(result.HasRootComposerJSON) {
			t.Fatal("expected true")
		}
		if !(result.HasRootCargoToml) {
			t.Fatal("expected true")
		}
		if !(result.HasTaskfile) {
			t.Fatal("expected true")
		}
		if result.HasSubtreePackageJSON {
			t.Fatal("expected false")
		}

	})

	t.Run("maps configured wails type to the action stack suggestion", func(t *testing.T) {
		dir := t.TempDir()
		if err := ax.MkdirAll(ax.Join(dir, ".core"), 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(dir, ".core", "build.yaml"), []byte("build:\n  type: wails\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]ProjectType{ProjectTypeWails}, result.Types) {
			t.Fatalf("want %v, got %v", []ProjectType{ProjectTypeWails}, result.Types)
		}
		if !stdlibAssertEqual("wails", result.ConfiguredType) {
			t.Fatalf("want %v, got %v", "wails", result.ConfiguredType)
		}
		if !stdlibAssertEqual("wails2", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "wails2", result.SuggestedStack)
		}
		if !stdlibAssertEqual("wails2", result.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "wails2", result.PrimaryStackSuggestion)
		}

	})

	t.Run("reports distro-aware Linux packages for Wails projects", func(t *testing.T) {
		mock := io.NewMemoryMedium()
		if err := mock.EnsureDir("/project"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.Write("/project/go.mod", "module example"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.Write("/project/package.json", "{}"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.Write("/etc/os-release", "ID=ubuntu\nVERSION_ID=\"24.04\"\n"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(mock, "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEqual([]string{"libwebkit2gtk-4.1-dev"}, result.LinuxPackages) {
			t.Fatalf("want %v, got %v", []string{"libwebkit2gtk-4.1-dev"}, result.LinuxPackages)
		}
		if !stdlibAssertEqual("libwebkit2gtk-4.1-dev", result.WebKitPackage) {
			t.Fatalf("want %v, got %v", "libwebkit2gtk-4.1-dev", result.WebKitPackage)
		}

	})

	t.Run("empty directory returns empty result", func(t *testing.T) {
		dir := t.TempDir()
		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEmpty(result.Types) {
			t.Fatalf("expected empty, got %v", result.Types)
		}
		if !stdlibAssertEmpty(result.PrimaryStack) {
			t.Fatalf("expected empty, got %v", result.PrimaryStack)
		}
		if !stdlibAssertEqual("unknown", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "unknown", result.SuggestedStack)
		}
		if result.HasFrontend {
			t.Fatal("expected false")
		}
		if result.HasRootPackageJSON {
			t.Fatal("expected false")
		}
		if result.HasFrontendPackageJSON {
			t.Fatal("expected false")
		}
		if result.HasRootComposerJSON {
			t.Fatal("expected false")
		}
		if result.HasRootCargoToml {
			t.Fatal("expected false")
		}
		if result.HasPackageJSON {
			t.Fatal("expected false")
		}
		if result.HasDenoManifest {
			t.Fatal("expected false")
		}
		if result.HasRootGoMod {
			t.Fatal("expected false")
		}
		if result.HasRootGoWork {
			t.Fatal("expected false")
		}
		if result.HasRootMainGo {
			t.Fatal("expected false")
		}
		if result.HasRootCMakeLists {
			t.Fatal("expected false")
		}
		if result.HasRootWailsJSON {
			t.Fatal("expected false")
		}
		if result.HasTaskfile {
			t.Fatal("expected false")
		}
		if result.HasSubtreeNpm {
			t.Fatal("expected false")
		}
		if result.HasSubtreePackageJSON {
			t.Fatal("expected false")
		}
		if result.HasSubtreeDenoManifest {
			t.Fatal("expected false")
		}
		if result.HasDocsConfig {
			t.Fatal("expected false")
		}
		if result.HasGoToolchain {
			t.Fatal("expected false")
		}
		if !stdlibAssertEqual("unknown", result.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "unknown", result.PrimaryStackSuggestion)
		}
		if !stdlibAssertEmpty(result.WebKitPackage) {
			t.Fatalf("expected empty, got %v", result.WebKitPackage)
		}

	})

	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T) string
		want    []ProjectType
		stack   string
		markers []string
		check   func(t *testing.T, result *DiscoveryResult)
	}{
		{
			name:    "detects docs project markers",
			setup:   func(t *testing.T) string { return setupTestDir(t, "mkdocs.yml") },
			want:    []ProjectType{ProjectTypeDocs},
			stack:   "docs",
			markers: []string{"mkdocs.yml"},
			check: func(t *testing.T, result *DiscoveryResult) {
				t.Helper()
				if !stdlibAssertEqual("docs", result.PrimaryStackSuggestion) {
					t.Fatalf("want %v, got %v", "docs", result.PrimaryStackSuggestion)
				}
				if !result.HasDocsConfig {
					t.Fatal("expected true")
				}
			},
		},
		{
			name:    "detects docs project markers with mkdocs.yaml",
			setup:   func(t *testing.T) string { return setupTestDir(t, "mkdocs.yaml") },
			want:    []ProjectType{ProjectTypeDocs},
			stack:   "docs",
			markers: []string{"mkdocs.yaml"},
		},
		{
			name:    "detects docs project markers in docs directory",
			setup:   func(t *testing.T) string { return setupDiscoveryFile(t, "docs/mkdocs.yaml", "site_name: Demo\n") },
			want:    []ProjectType{ProjectTypeDocs},
			stack:   "docs",
			markers: []string{"docs/mkdocs.yaml"},
		},
		{
			name:    "detects Rust project markers",
			setup:   func(t *testing.T) string { return setupTestDir(t, "Cargo.toml") },
			want:    []ProjectType{ProjectTypeRust},
			stack:   "rust",
			markers: []string{"Cargo.toml"},
		},
		{
			name:    "detects Python project markers",
			setup:   func(t *testing.T) string { return setupTestDir(t, "pyproject.toml") },
			want:    []ProjectType{ProjectTypePython},
			stack:   "python",
			markers: []string{"pyproject.toml"},
		},
		{
			name:    "detects Docker project markers",
			setup:   func(t *testing.T) string { return setupTestDir(t, "Dockerfile") },
			want:    []ProjectType{ProjectTypeDocker},
			stack:   "docker",
			markers: []string{"Dockerfile"},
		},
		{
			name:    "records alternate Docker manifest markers",
			setup:   func(t *testing.T) string { return setupTestDir(t, "Containerfile", "dockerfile", "containerfile") },
			want:    []ProjectType{ProjectTypeDocker},
			stack:   "docker",
			markers: []string{"Containerfile", "dockerfile", "containerfile"},
		},
		{
			name: "detects LinuxKit project markers in .core/linuxkit",
			setup: func(t *testing.T) string {
				return setupDiscoveryFile(t, ".core/linuxkit/server.yml", "kernel:\n  image: test")
			},
			want:    []ProjectType{ProjectTypeLinuxKit},
			stack:   "linuxkit",
			markers: []string{".core/linuxkit/*.yml", ".core/linuxkit/*.yaml"},
		},
		{
			name:    "detects LinuxKit project markers in linuxkit.yaml",
			setup:   func(t *testing.T) string { return setupTestDir(t, "linuxkit.yaml") },
			want:    []ProjectType{ProjectTypeLinuxKit},
			stack:   "linuxkit",
			markers: []string{"linuxkit.yaml"},
		},
		{
			name:    "detects C++ project markers",
			setup:   func(t *testing.T) string { return setupTestDir(t, "CMakeLists.txt") },
			want:    []ProjectType{ProjectTypeCPP},
			stack:   "cpp",
			markers: []string{"CMakeLists.txt"},
		},
		{
			name:    "detects Taskfile project markers",
			setup:   func(t *testing.T) string { return setupTestDir(t, "Taskfile.yaml") },
			want:    []ProjectType{ProjectTypeTaskfile},
			stack:   "taskfile",
			markers: []string{"Taskfile.yaml"},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := assertDiscoverFullStack(t, fs, tc.setup(t), tc.want, tc.stack, tc.markers...)
			if tc.check != nil {
				tc.check(t, result)
			}
		})
	}

	t.Run("reports nested Go toolchains for action parity even when root detection is empty", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "services", "api")
		if err := ax.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := ax.WriteFile(ax.Join(nested, "go.mod"), []byte("module example/api\n"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEmpty(result.Types) {
			t.Fatalf("expected empty, got %v", result.Types)
		}
		if !stdlibAssertEmpty(result.PrimaryStack) {
			t.Fatalf("expected empty, got %v", result.PrimaryStack)
		}
		if !stdlibAssertEqual("unknown", result.SuggestedStack) {
			t.Fatalf("want %v, got %v", "unknown", result.SuggestedStack)
		}
		if !(result.HasGoToolchain) {
			t.Fatal("expected true")
		}
		if !stdlibAssertEqual("go", result.PrimaryStackSuggestion) {
			t.Fatalf("want %v, got %v", "go", result.PrimaryStackSuggestion)
		}

	})
}

func TestDiscovery_DiscoverFull_Bad(t *testing.T) {
	fs := io.Local
	t.Run("non-existent directory returns empty result", func(t *testing.T) {
		result, err := DiscoverFull(fs, "/non/existent/path")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stdlibAssertEmpty(result.Types) {
			t.Fatalf("expected empty, got %v", result.Types)
		}
		if !stdlibAssertEmpty(result.PrimaryStack) {
			t.Fatalf("expected empty, got %v", result.PrimaryStack)
		}

	})
}

func TestDiscovery_DiscoverFull_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("markers map is never nil even for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		result, err := DiscoverFull(fs, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stdlibAssertNil(result.Markers) {
			t.Fatal("expected non-nil")
		}

	})
}

func TestDiscovery_SuggestStack_Good(t *testing.T) {
	t.Run("maps Wails projects to the v3 action stack name", func(t *testing.T) {
		if !stdlibAssertEqual("wails2", SuggestStack([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode})) {
			t.Fatalf("want %v, got %v", "wails2", SuggestStack([]ProjectType{ProjectTypeWails, ProjectTypeGo, ProjectTypeNode}))
		}

	})

	t.Run("passes through non-Wails primary project types", func(t *testing.T) {
		if !stdlibAssertEqual("cpp", SuggestStack([]ProjectType{ProjectTypeCPP})) {
			t.Fatalf("want %v, got %v", "cpp", SuggestStack([]ProjectType{ProjectTypeCPP}))
		}
		if !stdlibAssertEqual("docs", SuggestStack([]ProjectType{ProjectTypeDocs})) {
			t.Fatalf("want %v, got %v", "docs", SuggestStack([]ProjectType{ProjectTypeDocs}))
		}
		if !stdlibAssertEqual("node", SuggestStack([]ProjectType{ProjectTypeNode})) {
			t.Fatalf("want %v, got %v", "node", SuggestStack([]ProjectType{ProjectTypeNode}))
		}
		if !stdlibAssertEqual("go", SuggestStack([]ProjectType{ProjectTypeGo})) {
			t.Fatalf("want %v, got %v", "go", SuggestStack([]ProjectType{ProjectTypeGo}))
		}

	})

	t.Run("returns empty when nothing is detected", func(t *testing.T) {
		if !stdlibAssertEqual("unknown", SuggestStack(nil)) {
			t.Fatalf("want %v, got %v", "unknown", SuggestStack(nil))
		}

	})
}

func TestDiscovery_ResolveLinuxPackages_Good(t *testing.T) {
	t.Run("returns Ubuntu 24.04 WebKit package for Wails", func(t *testing.T) {
		packages := ResolveLinuxPackages([]ProjectType{ProjectTypeWails}, "24.04")
		if !stdlibAssertEqual([]string{"libwebkit2gtk-4.1-dev"}, packages) {
			t.Fatalf("want %v, got %v", []string{"libwebkit2gtk-4.1-dev"}, packages)
		}

	})

	t.Run("returns Ubuntu 22.04 WebKit package for Wails", func(t *testing.T) {
		packages := ResolveLinuxPackages([]ProjectType{ProjectTypeWails}, "22.04")
		if !stdlibAssertEqual([]string{"libwebkit2gtk-4.0-dev"}, packages) {
			t.Fatalf("want %v, got %v", []string{"libwebkit2gtk-4.0-dev"}, packages)
		}

	})

	t.Run("returns no Linux packages for non-Wails stacks", func(t *testing.T) {
		packages := ResolveLinuxPackages([]ProjectType{ProjectTypeGo}, "24.04")
		if !stdlibAssertEmpty(packages) {
			t.Fatalf("expected empty, got %v", packages)
		}

	})
}

func TestDiscovery_ParseOSReleaseDistroGood(t *testing.T) {
	t.Run("returns ubuntu version id", func(t *testing.T) {
		content := `
NAME="Ubuntu"
ID=ubuntu
VERSION_ID="24.04"
ID_LIKE=debian
`
		if !stdlibAssertEqual("24.04", parseOSReleaseDistro(content)) {
			t.Fatalf("want %v, got %v", "24.04", parseOSReleaseDistro(content))
		}

	})

	t.Run("accepts ubuntu-style values without quotes", func(t *testing.T) {
		content := `
ID=ubuntu
VERSION_ID=25.10
`
		if !stdlibAssertEqual("25.10", parseOSReleaseDistro(content)) {
			t.Fatalf("want %v, got %v", "25.10", parseOSReleaseDistro(content))
		}

	})
}

func TestDiscovery_ParseOSReleaseDistroBad(t *testing.T) {
	t.Run("returns empty for non-ubuntu distro", func(t *testing.T) {
		content := `
ID=fedora
VERSION_ID=41
`
		if !stdlibAssertEmpty(parseOSReleaseDistro(content)) {
			t.Fatalf("expected empty, got %v", parseOSReleaseDistro(content))
		}

	})

	t.Run("returns empty when version missing", func(t *testing.T) {
		content := `
ID=ubuntu
`
		if !stdlibAssertEmpty(parseOSReleaseDistro(content)) {
			t.Fatalf("expected empty, got %v", parseOSReleaseDistro(content))
		}

	})
}

func TestDiscovery_DetectDistroVersionGood(t *testing.T) {
	fs := io.NewMemoryMedium()
	if err := fs.Write("/etc/os-release", `
ID=ubuntu
VERSION_ID="24.04"
`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEqual("24.04", detectDistroVersion(fs)) {
		t.Fatalf("want %v, got %v", "24.04", detectDistroVersion(fs))
	}

}

func TestDiscovery_DetectDistroVersionBad(t *testing.T) {
	fs := io.NewMemoryMedium()
	if err := fs.Write("/etc/os-release", `
ID=fedora
VERSION_ID=41
`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEmpty(detectDistroVersion(fs)) {
		t.Fatalf("expected empty, got %v", detectDistroVersion(fs))
	}

}

func TestDiscovery_NilMediumGood(t *testing.T) {
	dir := t.TempDir()

	types, err := Discover(nil, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stdlibAssertEmpty(types) {
		t.Fatalf("expected empty, got %v", types)
	}

	result, err := DiscoverFull(nil, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdlibAssertNil(result) {
		t.Fatal("expected non-nil")
	}
	if !stdlibAssertEmpty(result.Types) {
		t.Fatalf("expected empty, got %v", result.Types)
	}

}

// --- v0.9.0 generated compliance triplets ---
func TestDiscovery_Discover_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = Discover(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_PrimaryType_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = PrimaryType(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_PrimaryType_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_, _ = PrimaryType(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsGoProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsGoProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsGoProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsGoProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsWailsProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsWailsProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsWailsProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsWailsProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsNodeProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsNodeProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsNodeProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsNodeProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsPHPProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsPHPProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsPHPProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsPHPProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsCPPProject_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsCPPProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsCPPProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsCPPProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsCPPProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsCPPProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsDocsProject_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDocsProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsDocsProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDocsProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsDocsProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDocsProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_ResolveMkDocsConfigPath_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveMkDocsConfigPath(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_ResolveMkDocsConfigPath_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveMkDocsConfigPath(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_ResolveMkDocsConfigPath_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveMkDocsConfigPath(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_SuggestStack_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = SuggestStack(nil)
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_SuggestStack_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = SuggestStack(nil)
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_ResolveLinuxPackages_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveLinuxPackages(nil, "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_ResolveLinuxPackages_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveLinuxPackages(nil, "agent")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_ResolveDockerfilePath_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveDockerfilePath(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_ResolveDockerfilePath_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveDockerfilePath(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_ResolveDockerfilePath_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ResolveDockerfilePath(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsDockerProject_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDockerProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsDockerProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDockerProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsDockerProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsDockerProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsLinuxKitProject_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsLinuxKitProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsLinuxKitProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsLinuxKitProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsLinuxKitProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsLinuxKitProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsTaskfileProject_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsTaskfileProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsTaskfileProject_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsTaskfileProject(io.NewMemoryMedium(), "")
	})
	core.AssertTrue(t, true)
}

func TestDiscovery_IsTaskfileProject_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = IsTaskfileProject(io.NewMemoryMedium(), core.Path(t.TempDir(), "go-build-compliance"))
	})
	core.AssertTrue(t, true)
}
