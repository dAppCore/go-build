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

	t.Run("detects multiple project types", func(t *testing.T) {
		dir := setupTestDir(t, "go.mod", "package.json")
		types, err := Discover(fs, dir)
		assert.NoError(t, err)
		assert.Equal(t, []ProjectType{ProjectTypeGo, ProjectTypeNode}, types)
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
