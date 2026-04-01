// SPDX-Licence-Identifier: EUPL-1.2

package api

import (
	"io/fs"
	"testing"

	"dappco.re/go/core/build/internal/ax"
	"dappco.re/go/core/build/pkg/build"
	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_BuildProviderIdentity_Good(t *testing.T) {
	p := NewProvider(".", nil)

	assert.Equal(t, "build", p.Name())
	assert.Equal(t, "/api/v1/build", p.BasePath())
}

func TestProvider_BuildProviderElement_Good(t *testing.T) {
	p := NewProvider(".", nil)
	el := p.Element()

	assert.Equal(t, "core-build-panel", el.Tag)
	assert.Equal(t, "/assets/core-build.js", el.Source)
}

func TestProvider_BuildProviderChannels_Good(t *testing.T) {
	p := NewProvider(".", nil)
	channels := p.Channels()

	assert.Contains(t, channels, "build.started")
	assert.Contains(t, channels, "build.complete")
	assert.Contains(t, channels, "build.failed")
	assert.Contains(t, channels, "release.started")
	assert.Contains(t, channels, "release.complete")
	assert.Contains(t, channels, "sdk.generated")
	assert.Len(t, channels, 6)
}

func TestProvider_BuildProviderDescribe_Good(t *testing.T) {
	p := NewProvider(".", nil)
	routes := p.Describe()

	// Should have 9 endpoint descriptions
	assert.Len(t, routes, 9)

	// Verify key routes exist
	paths := make(map[string]string)
	for _, r := range routes {
		paths[r.Path] = r.Method
	}

	assert.Equal(t, "GET", paths["/config"])
	assert.Equal(t, "GET", paths["/discover"])
	assert.Equal(t, "POST", paths["/build"])
	assert.Equal(t, "GET", paths["/artifacts"])
	assert.Equal(t, "GET", paths["/release/version"])
	assert.Equal(t, "GET", paths["/release/changelog"])
	assert.Equal(t, "POST", paths["/release"])
	assert.Equal(t, "GET", paths["/sdk/diff"])
	assert.Equal(t, "POST", paths["/sdk/generate"])
}

func TestProvider_BuildProviderDefaultProjectDir_Good(t *testing.T) {
	p := NewProvider("", nil)
	assert.Equal(t, ".", p.projectDir)
}

func TestProvider_BuildProviderCustomProjectDir_Good(t *testing.T) {
	p := NewProvider("/tmp/myproject", nil)
	assert.Equal(t, "/tmp/myproject", p.projectDir)
}

func TestProvider_BuildProviderNilHub_Good(t *testing.T) {
	p := NewProvider(".", nil)
	// emitEvent should not panic with nil hub
	p.emitEvent("build.started", map[string]any{"test": true})
}

func TestProvider_GetBuilderSupportedTypes_Good(t *testing.T) {
	cases := []struct {
		projectType build.ProjectType
		name        string
	}{
		{build.ProjectTypeGo, "go"},
		{build.ProjectTypeWails, "wails"},
		{build.ProjectTypeNode, "node"},
		{build.ProjectTypePHP, "php"},
		{build.ProjectTypePython, "python"},
		{build.ProjectTypeRust, "rust"},
		{build.ProjectTypeDocs, "docs"},
		{build.ProjectTypeCPP, "cpp"},
		{build.ProjectTypeDocker, "docker"},
		{build.ProjectTypeLinuxKit, "linuxkit"},
		{build.ProjectTypeTaskfile, "taskfile"},
	}

	for _, tc := range cases {
		t.Run(string(tc.projectType), func(t *testing.T) {
			b, err := getBuilder(tc.projectType)
			require.NoError(t, err)
			assert.Equal(t, tc.name, b.Name())
		})
	}
}

func TestProvider_GetBuilderUnsupportedType_Bad(t *testing.T) {
	_, err := getBuilder(build.ProjectType("unknown"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestProvider_BuildProviderResolveDir_Good(t *testing.T) {
	p := NewProvider("/tmp", nil)
	dir, err := p.resolveDir()
	require.NoError(t, err)
	assert.Equal(t, "/tmp", dir)
}

func TestProvider_BuildProviderResolveDirRelative_Good(t *testing.T) {
	p := NewProvider(".", nil)
	dir, err := p.resolveDir()
	require.NoError(t, err)
	// Should return an absolute path
	assert.True(t, len(dir) > 1 && dir[0] == '/')
}

func TestProvider_BuildProviderMediumSet_Good(t *testing.T) {
	p := NewProvider(".", nil)
	assert.NotNil(t, p.medium, "medium should be set to io.Local")
}

func TestProvider_ResolveProjectType_Good(t *testing.T) {
	t.Run("honours explicit build type override", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644))

		projectType, err := resolveProjectType(io.Local, dir, "docker")
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeDocker, projectType)
	})

	t.Run("falls back to detection when build type is empty", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ax.WriteFile(ax.Join(dir, "go.mod"), []byte("module example"), 0o644))

		projectType, err := resolveProjectType(io.Local, dir, "")
		require.NoError(t, err)
		assert.Equal(t, build.ProjectTypeGo, projectType)
	})
}
