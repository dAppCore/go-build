package build

import (
	"testing"

	"dappco.re/go/core/build/internal/ax"

	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDir creates a temporary directory with the specified marker files.
func setupTestDir(t *testing.T, markers ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, m := range markers {
		path := ax.Join(dir, m)
		err := ax.WriteFile(path, []byte("{}"), 0644)
		require.NoError(t, err)
	}
	return dir
}

func TestDiscovery_Discover_Good(t *testing.T) {
	fs := io.Local
	t.Run("detects Go project", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeGo}, types)
	})

	t.Run("detects Wails project with priority over Go", func(t *testing.T) {
		dir := setupTestDir(t, "wails.json", "go.mod")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeWails, ProjectTypeGo}, types)
	})

	t.Run("detects Node.js project", func(t *testing.T) {
		dir := setupTestDir(t, "package.json")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeNode}, types)
	})

	t.Run("detects PHP project", func(t *testing.T) {
		dir := setupTestDir(t, "composer.json")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypePHP}, types)
	})

	t.Run("detects docs project", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yml")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeDocs}, types)
	})

	t.Run("detects docs project with mkdocs.yaml", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yaml")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeDocs}, types)
	})

	t.Run("detects Python project with pyproject.toml", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypePython}, types)
	})

	t.Run("detects Python project with requirements.txt", func(t *testing.T) {
		dir := setupTestDir(t, "requirements.txt")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypePython}, types)
	})

	t.Run("detects Python only once with both markers", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml", "requirements.txt")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypePython}, types)
	})

	t.Run("detects Rust project", func(t *testing.T) {
		dir := setupTestDir(t, "Cargo.toml")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeRust}, types)
	})

	t.Run("detects Docker project", func(t *testing.T) {
		dir := setupTestDir(t, "Dockerfile")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeDocker}, types)
	})

	t.Run("detects LinuxKit project", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(lkDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(lkDir, "server.yml"), []byte("kernel:\n"), 0644))

		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeLinuxKit}, types)
	})

	t.Run("detects C++ project", func(t *testing.T) {
		dir := setupTestDir(t, "CMakeLists.txt")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeCPP}, types)
	})

	t.Run("detects Taskfile project", func(t *testing.T) {
		dir := setupTestDir(t, "Taskfile.yml")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeTaskfile}, types)
	})

	t.Run("detects multiple project types", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "package.json")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeGo, ProjectTypeNode}, types)
	})

	t.Run("preserves priority when core and fallback markers overlap", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "Dockerfile", "Taskfile.yml")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeGo, ProjectTypeDocker, ProjectTypeTaskfile}, types)
	})

	t.Run("empty directory returns empty slice", func(t *testing.T) {
		dir := t.TempDir()
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Empty(t, types)
	})
}

func TestDiscovery_Discover_Bad(t *testing.T) {
	fs := io.Local
	t.Run("non-existent directory returns empty slice", func(t *testing.T) {
		types, err := Discover(fs, "/non/existent/path")
		assert.NoError(t, err) // ax.Stat fails silently in fileExists
		assert.Empty(t, types)
	})

	t.Run("directory marker is ignored", func(t *testing.T) {
		dir := t.TempDir()
		// Create go.mod as a directory instead of a file
		err := ax.Mkdir(ax.Join(dir, "go.mod"), 0755)
		require.NoError(t, err)

		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Empty(t, types)
	})
}

func TestDiscovery_PrimaryType_Good(t *testing.T) {
	fs := io.Local
	t.Run("returns wails for wails project", func(t *testing.T) {
		dir := setupTestDir(t, "wails.json", "go.mod")
		primary, err := PrimaryType(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, ProjectTypeWails, primary)
	})

	t.Run("returns go for go-only project", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		primary, err := PrimaryType(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, ProjectTypeGo, primary)
	})

	t.Run("returns empty string for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		primary, err := PrimaryType(fs, dir)
		assert.NoError(t, err)
		assert.Empty(t, primary)
	})
}

func TestDiscovery_IsGoProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with go.mod", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		assert.True(t, IsGoProject(fs, dir))
	})

	t.Run("true with wails.json", func(t *testing.T) {
		dir := setupTestDir(t, "wails.json")
		assert.True(t, IsGoProject(fs, dir))
	})

	t.Run("false without markers", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, IsGoProject(fs, dir))
	})
}

func TestDiscovery_IsWailsProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with wails.json", func(t *testing.T) {
		dir := setupTestDir(t, "wails.json")
		assert.True(t, IsWailsProject(fs, dir))
	})

	t.Run("false with only go.mod", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		assert.False(t, IsWailsProject(fs, dir))
	})
}

func TestDiscovery_IsNodeProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with package.json", func(t *testing.T) {
		dir := setupTestDir(t, "package.json")
		assert.True(t, IsNodeProject(fs, dir))
	})

	t.Run("false without package.json", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, IsNodeProject(fs, dir))
	})
}

func TestDiscovery_IsPHPProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with composer.json", func(t *testing.T) {
		dir := setupTestDir(t, "composer.json")
		assert.True(t, IsPHPProject(fs, dir))
	})

	t.Run("false without composer.json", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, IsPHPProject(fs, dir))
	})
}

func TestDiscovery_Target_Good(t *testing.T) {
	target := Target{OS: "linux", Arch: "amd64"}
	assert.Equal(t, "linux/amd64", target.String())
}

func TestDiscovery_FileExists_Good(t *testing.T) {
	fs := io.Local
	t.Run("returns true for existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := ax.Join(dir, "test.txt")
		err := ax.WriteFile(path, []byte("content"), 0644)
		require.NoError(t, err)
		assert.True(t, fileExists(fs, path))
	})

	t.Run("returns false for directory", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, fileExists(fs, dir))
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		assert.False(t, fileExists(fs, "/non/existent/file"))
	})
}

// TestDiscover_Testdata tests discovery using the testdata fixtures.
// These serve as integration tests with realistic project structures.
func TestDiscovery_DiscoverTestdata_Good(t *testing.T) {
	fs := io.Local
	testdataDir, err := ax.Abs("testdata")
	require.NoError(t, err)

	tests := []struct {
		name     string
		dir      string
		expected []ProjectType
	}{
		{"go-project", "go-project", []ProjectType{ProjectTypeGo}},
		{"wails-project", "wails-project", []ProjectType{ProjectTypeWails, ProjectTypeGo}},
		{"node-project", "node-project", []ProjectType{ProjectTypeNode}},
		{"php-project", "php-project", []ProjectType{ProjectTypePHP}},
		{"multi-project", "multi-project", []ProjectType{ProjectTypeGo, ProjectTypeNode}},
		{"empty-project", "empty-project", []ProjectType{}},
		{"docs-project", "docs-project", []ProjectType{ProjectTypeDocs}},
		{"python-project", "python-project", []ProjectType{ProjectTypePython}},
		{"rust-project", "rust-project", []ProjectType{ProjectTypeRust}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := ax.Join(testdataDir, tt.dir)
			types, err := Discover(fs, dir)
			assert.NoError(t, err)
			if len(tt.expected) == 0 {
				assert.Empty(t, types)
			} else {
				assert.Equal(t, tt.expected, types)
			}
		})
	}
}

func TestDiscovery_IsMkDocsProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with mkdocs.yml", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yml")
		assert.True(t, IsMkDocsProject(fs, dir))
	})

	t.Run("true with mkdocs.yaml", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yaml")
		assert.True(t, IsMkDocsProject(fs, dir))
	})

	t.Run("false without mkdocs.yml", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, IsMkDocsProject(fs, dir))
	})
}

func TestDiscovery_IsMkDocsProject_Bad(t *testing.T) {
	fs := io.Local
	t.Run("false for non-existent directory", func(t *testing.T) {
		assert.False(t, IsMkDocsProject(fs, "/non/existent/path"))
	})
}

func TestDiscovery_IsMkDocsProject_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("false when mkdocs.yml is a directory", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.Mkdir(ax.Join(dir, "mkdocs.yml"), 0755)
		require.NoError(t, err)
		assert.False(t, IsMkDocsProject(fs, dir))
	})
}

func TestDiscovery_HasSubtreeNpm_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with depth 1 nested package.json", func(t *testing.T) {
		dir := t.TempDir()
		subdir := ax.Join(dir, "packages", "web")
		err := ax.MkdirAll(subdir, 0755)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(dir, "packages", "package.json"), []byte("{}"), 0644)
		require.NoError(t, err)
		assert.True(t, HasSubtreeNpm(fs, dir))
	})

	t.Run("true with depth 2 nested package.json", func(t *testing.T) {
		dir := t.TempDir()
		nested := ax.Join(dir, "apps", "web")
		err := ax.MkdirAll(nested, 0755)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0644)
		require.NoError(t, err)
		assert.True(t, HasSubtreeNpm(fs, dir))
	})

	t.Run("false with only root package.json", func(t *testing.T) {
		dir := setupTestDir(t, "package.json")
		assert.False(t, HasSubtreeNpm(fs, dir))
	})

	t.Run("false with empty directory", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, HasSubtreeNpm(fs, dir))
	})
}

func TestDiscovery_HasSubtreeNpm_Bad(t *testing.T) {
	fs := io.Local
	t.Run("false for non-existent directory", func(t *testing.T) {
		assert.False(t, HasSubtreeNpm(fs, "/non/existent/path"))
	})

	t.Run("ignores node_modules at depth 1", func(t *testing.T) {
		dir := t.TempDir()
		nmDir := ax.Join(dir, "node_modules", "some-pkg")
		err := ax.MkdirAll(nmDir, 0755)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(nmDir, "package.json"), []byte("{}"), 0644)
		require.NoError(t, err)
		assert.False(t, HasSubtreeNpm(fs, dir))
	})

	t.Run("ignores node_modules at depth 2", func(t *testing.T) {
		dir := t.TempDir()
		nmDir := ax.Join(dir, "apps", "node_modules", "some-pkg")
		err := ax.MkdirAll(nmDir, 0755)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(nmDir, "package.json"), []byte("{}"), 0644)
		require.NoError(t, err)
		// Also need the apps dir to be listable — it is since we created nmDir inside it
		assert.False(t, HasSubtreeNpm(fs, dir))
	})
}

func TestDiscovery_HasSubtreeNpm_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("false when nested package.json is beyond depth 2", func(t *testing.T) {
		dir := t.TempDir()
		deep := ax.Join(dir, "a", "b", "c")
		err := ax.MkdirAll(deep, 0755)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(deep, "package.json"), []byte("{}"), 0644)
		require.NoError(t, err)
		assert.False(t, HasSubtreeNpm(fs, dir))
	})
}

func TestDiscovery_IsPythonProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with pyproject.toml", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml")
		assert.True(t, IsPythonProject(fs, dir))
	})

	t.Run("true with requirements.txt", func(t *testing.T) {
		dir := setupTestDir(t, "requirements.txt")
		assert.True(t, IsPythonProject(fs, dir))
	})

	t.Run("true with both markers", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml", "requirements.txt")
		assert.True(t, IsPythonProject(fs, dir))
	})

	t.Run("false without markers", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, IsPythonProject(fs, dir))
	})
}

func TestDiscovery_IsPythonProject_Bad(t *testing.T) {
	fs := io.Local
	t.Run("false for non-existent directory", func(t *testing.T) {
		assert.False(t, IsPythonProject(fs, "/non/existent/path"))
	})
}

func TestDiscovery_IsPythonProject_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("false when pyproject.toml is a directory", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.Mkdir(ax.Join(dir, "pyproject.toml"), 0755)
		require.NoError(t, err)
		assert.False(t, IsPythonProject(fs, dir))
	})
}

func TestDiscovery_IsRustProject_Good(t *testing.T) {
	fs := io.Local
	t.Run("true with Cargo.toml", func(t *testing.T) {
		dir := setupTestDir(t, "Cargo.toml")
		assert.True(t, IsRustProject(fs, dir))
	})

	t.Run("false without Cargo.toml", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, IsRustProject(fs, dir))
	})
}

func TestDiscovery_IsRustProject_Bad(t *testing.T) {
	fs := io.Local
	t.Run("false for non-existent directory", func(t *testing.T) {
		assert.False(t, IsRustProject(fs, "/non/existent/path"))
	})
}

func TestDiscovery_IsRustProject_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("false when Cargo.toml is a directory", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.Mkdir(ax.Join(dir, "Cargo.toml"), 0755)
		require.NoError(t, err)
		assert.False(t, IsRustProject(fs, dir))
	})
}

func TestDiscovery_DiscoverFull_Good(t *testing.T) {
	fs := io.Local
	t.Run("returns complete result for Go project", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod")
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeGo}, result.Types)
		assert.Equal(t, "go", result.PrimaryStack)
		assert.False(t, result.HasFrontend)
		assert.False(t, result.HasSubtreeNpm)
		assert.True(t, result.Markers["go.mod"])
		assert.False(t, result.Markers["wails.json"])
	})

	t.Run("returns complete result for Wails project with frontend", func(t *testing.T) {
		dir := t.TempDir()
		// Create wails.json, go.mod, and frontend/package.json
		err := ax.WriteFile(ax.Join(dir, "wails.json"), []byte("{}"), 0644)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(dir, "go.mod"), []byte("{}"), 0644)
		require.NoError(t, err)
		err = ax.MkdirAll(ax.Join(dir, "frontend"), 0755)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(dir, "frontend", "package.json"), []byte("{}"), 0644)
		require.NoError(t, err)

		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeWails, ProjectTypeGo}, result.Types)
		assert.Equal(t, "wails", result.PrimaryStack)
		assert.True(t, result.HasFrontend)
		assert.True(t, result.Markers["wails.json"])
		assert.True(t, result.Markers["go.mod"])
	})

	t.Run("detects subtree npm as frontend", func(t *testing.T) {
		dir := t.TempDir()
		err := ax.WriteFile(ax.Join(dir, "go.mod"), []byte("{}"), 0644)
		require.NoError(t, err)
		nested := ax.Join(dir, "apps", "web")
		err = ax.MkdirAll(nested, 0755)
		require.NoError(t, err)
		err = ax.WriteFile(ax.Join(nested, "package.json"), []byte("{}"), 0644)
		require.NoError(t, err)

		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeGo}, result.Types)
		assert.True(t, result.HasSubtreeNpm)
		assert.True(t, result.HasFrontend)
	})

	t.Run("empty directory returns empty result", func(t *testing.T) {
		dir := t.TempDir()
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Empty(t, result.Types)
		assert.Empty(t, result.PrimaryStack)
		assert.False(t, result.HasFrontend)
		assert.False(t, result.HasSubtreeNpm)
	})

	t.Run("detects docs project markers", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yml")
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeDocs}, result.Types)
		assert.Equal(t, "docs", result.PrimaryStack)
		assert.True(t, result.Markers["mkdocs.yml"])
	})

	t.Run("detects docs project markers with mkdocs.yaml", func(t *testing.T) {
		dir := setupTestDir(t, "mkdocs.yaml")
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeDocs}, result.Types)
		assert.Equal(t, "docs", result.PrimaryStack)
		assert.True(t, result.Markers["mkdocs.yaml"])
	})

	t.Run("detects Rust project markers", func(t *testing.T) {
		dir := setupTestDir(t, "Cargo.toml")
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeRust}, result.Types)
		assert.Equal(t, "rust", result.PrimaryStack)
		assert.True(t, result.Markers["Cargo.toml"])
	})

	t.Run("detects Python project markers", func(t *testing.T) {
		dir := setupTestDir(t, "pyproject.toml")
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypePython}, result.Types)
		assert.Equal(t, "python", result.PrimaryStack)
		assert.True(t, result.Markers["pyproject.toml"])
	})

	t.Run("detects Docker project markers", func(t *testing.T) {
		dir := setupTestDir(t, "Dockerfile")
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeDocker}, result.Types)
		assert.Equal(t, "docker", result.PrimaryStack)
		assert.True(t, result.Markers["Dockerfile"])
	})

	t.Run("detects LinuxKit project markers in .core/linuxkit", func(t *testing.T) {
		dir := t.TempDir()
		lkDir := ax.Join(dir, ".core", "linuxkit")
		require.NoError(t, ax.MkdirAll(lkDir, 0755))
		require.NoError(t, ax.WriteFile(ax.Join(lkDir, "server.yml"), []byte("kernel:\n  image: test"), 0644))

		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeLinuxKit}, result.Types)
		assert.Equal(t, "linuxkit", result.PrimaryStack)
		assert.True(t, result.Markers[".core/linuxkit/*.yml"])
	})

	t.Run("detects C++ project markers", func(t *testing.T) {
		dir := setupTestDir(t, "CMakeLists.txt")
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeCPP}, result.Types)
		assert.Equal(t, "cpp", result.PrimaryStack)
		assert.True(t, result.Markers["CMakeLists.txt"])
	})

	t.Run("detects Taskfile project markers", func(t *testing.T) {
		dir := setupTestDir(t, "Taskfile.yaml")
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeTaskfile}, result.Types)
		assert.Equal(t, "taskfile", result.PrimaryStack)
		assert.True(t, result.Markers["Taskfile.yaml"])
	})
}

func TestDiscovery_DiscoverFull_Bad(t *testing.T) {
	fs := io.Local
	t.Run("non-existent directory returns empty result", func(t *testing.T) {
		result, err := DiscoverFull(fs, "/non/existent/path")
		require.NoError(t, err)
		assert.Empty(t, result.Types)
		assert.Empty(t, result.PrimaryStack)
	})
}

func TestDiscovery_DiscoverFull_Ugly(t *testing.T) {
	fs := io.Local
	t.Run("markers map is never nil even for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		result, err := DiscoverFull(fs, dir)
		require.NoError(t, err)
		assert.NotNil(t, result.Markers)
	})
}

func TestDiscovery_ParseOSReleaseDistro_Good(t *testing.T) {
	t.Run("returns ubuntu version id", func(t *testing.T) {
		content := `
NAME="Ubuntu"
ID=ubuntu
VERSION_ID="24.04"
ID_LIKE=debian
`
		assert.Equal(t, "24.04", parseOSReleaseDistro(content))
	})

	t.Run("accepts ubuntu-style values without quotes", func(t *testing.T) {
		content := `
ID=ubuntu
VERSION_ID=25.10
`
		assert.Equal(t, "25.10", parseOSReleaseDistro(content))
	})
}

func TestDiscovery_ParseOSReleaseDistro_Bad(t *testing.T) {
	t.Run("returns empty for non-ubuntu distro", func(t *testing.T) {
		content := `
ID=fedora
VERSION_ID=41
`
		assert.Empty(t, parseOSReleaseDistro(content))
	})

	t.Run("returns empty when version missing", func(t *testing.T) {
		content := `
ID=ubuntu
`
		assert.Empty(t, parseOSReleaseDistro(content))
	})
}

func TestDiscovery_DetectDistroVersion_Good(t *testing.T) {
	fs := io.NewMockMedium()
	require.NoError(t, fs.Write("/etc/os-release", `
ID=ubuntu
VERSION_ID="24.04"
`))

	assert.Equal(t, "24.04", detectDistroVersion(fs))
}

func TestDiscovery_DetectDistroVersion_Bad(t *testing.T) {
	fs := io.NewMockMedium()
	require.NoError(t, fs.Write("/etc/os-release", `
ID=fedora
VERSION_ID=41
`))

	assert.Empty(t, detectDistroVersion(fs))
}
