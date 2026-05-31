package provider

import (
	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// registryStreamableProvider implements the optional Streamable interface so
// the Channels branch of Registry.Info can be exercised.
type registryStreamableProvider struct {
	name     string
	basePath string
	channels []string
}

func (p registryStreamableProvider) Name() string                   { return p.name }
func (p registryStreamableProvider) BasePath() string               { return p.basePath }
func (p registryStreamableProvider) RegisterRoutes(*gin.RouterGroup) {}
func (p registryStreamableProvider) Channels() []string             { return p.channels }

func TestProvider_Registry_Info_Streamable_Ugly(t *core.T) {
	registry := NewRegistry()
	registry.Add(registryStreamableProvider{
		name:     "events",
		basePath: "/events",
		channels: []string{"build", "release"},
	})
	info := registry.Info()
	core.AssertEqual(t, "events", info[0]["name"].(string))
	core.AssertEqual(t, "/events", info[0]["base_path"].(string))
	core.AssertEqual(t, []string{"build", "release"}, info[0]["channels"].([]string))
}
