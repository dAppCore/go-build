// SPDX-Licence-Identifier: EUPL-1.2

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildProvider_Good_Identity(t *testing.T) {
	p := NewProvider(".", nil)

	assert.Equal(t, "build", p.Name())
	assert.Equal(t, "/api/v1/build", p.BasePath())
}

func TestBuildProvider_Good_Element(t *testing.T) {
	p := NewProvider(".", nil)
	el := p.Element()

	assert.Equal(t, "core-build-panel", el.Tag)
	assert.Equal(t, "/assets/core-build.js", el.Source)
}

func TestBuildProvider_Good_Channels(t *testing.T) {
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

func TestBuildProvider_Good_Describe(t *testing.T) {
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

func TestBuildProvider_Good_DefaultProjectDir(t *testing.T) {
	p := NewProvider("", nil)
	assert.Equal(t, ".", p.projectDir)
}

func TestBuildProvider_Good_CustomProjectDir(t *testing.T) {
	p := NewProvider("/tmp/myproject", nil)
	assert.Equal(t, "/tmp/myproject", p.projectDir)
}

func TestBuildProvider_Good_NilHub(t *testing.T) {
	p := NewProvider(".", nil)
	// emitEvent should not panic with nil hub
	p.emitEvent("build.started", map[string]any{"test": true})
}
