package provider

import (
	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

type registryTestProvider struct {
	name     string
	basePath string
}

func (p registryTestProvider) Name() string { return p.name }

func (p registryTestProvider) BasePath() string { return p.basePath }

func (p registryTestProvider) RegisterRoutes(*gin.RouterGroup) {}

type registryRenderableProvider struct {
	registryTestProvider
}

func (p registryRenderableProvider) Element() ElementSpec {
	return ElementSpec{Tag: "x-test", Source: "test.js"}
}

func TestProvider_NewRegistry_Good(t *core.T) {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
	core.AssertFalse(t, registry.Get("build") == nil)
}

func TestProvider_NewRegistry_Bad(t *core.T) {
	registry := NewRegistry()
	provider := registry.Get("missing")
	core.AssertEqual(t, nil, provider)
}

func TestProvider_NewRegistry_Ugly(t *core.T) {
	first := NewRegistry()
	second := NewRegistry()
	first.Add(registryTestProvider{name: "build", basePath: "/build"})
	core.AssertEqual(t, nil, second.Get("build"))
}

func TestProvider_Registry_Add_Good(t *core.T) {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
	provider := registry.Get("build")
	core.AssertEqual(t, "/build", provider.BasePath())
}

func TestProvider_Registry_Add_Bad(t *core.T) {
	var registry *Registry
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
	core.AssertTrue(t, registry == nil)
}

func TestProvider_Registry_Add_Ugly(t *core.T) {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/old"})
	registry.Add(registryTestProvider{name: "build", basePath: "/new"})
	core.AssertEqual(t, "/new", registry.Get("build").BasePath())
}

func TestProvider_Registry_Get_Good(t *core.T) {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
	provider := registry.Get("build")
	core.AssertEqual(t, "build", provider.Name())
}

func TestProvider_Registry_Get_Bad(t *core.T) {
	var registry *Registry
	provider := registry.Get("build")
	core.AssertEqual(t, nil, provider)
}

func TestProvider_Registry_Get_Ugly(t *core.T) {
	registry := NewRegistry()
	provider := registry.Get("")
	core.AssertEqual(t, nil, provider)
}

func TestProvider_Registry_Info_Good(t *core.T) {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
	info := registry.Info()
	core.AssertEqual(t, "build", info[0]["name"].(string))
}

func TestProvider_Registry_Info_Bad(t *core.T) {
	var registry *Registry
	info := registry.Info()
	core.AssertEqual(t, 0, len(info))
}

func TestProvider_Registry_Info_Ugly(t *core.T) {
	registry := NewRegistry()
	registry.Add(registryRenderableProvider{registryTestProvider{name: "ui", basePath: "/ui"}})
	info := registry.Info()
	core.AssertEqual(t, "x-test", info[0]["element"].(ElementSpec).Tag)
}
