package provider

import "github.com/gin-gonic/gin"

type Provider interface {
	Name() string
	BasePath() string
	RegisterRoutes(*gin.RouterGroup)
}

type Streamable interface {
	Provider
	Channels() []string
}

type Describable interface {
	Provider
}

type Renderable interface {
	Provider
	Element() ElementSpec
}

type ElementSpec struct {
	Tag    string `json:"tag"`
	Source string `json:"source"`
}

type Registry struct {
	providers map[string]Provider
	order     []string
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Add(provider Provider) {
	if r == nil || provider == nil {
		return
	}
	name := provider.Name()
	if _, exists := r.providers[name]; !exists {
		r.order = append(r.order, name)
	}
	r.providers[name] = provider
}

func (r *Registry) Get(name string) Provider {
	if r == nil {
		return nil
	}
	return r.providers[name]
}

func (r *Registry) Info() []map[string]any {
	if r == nil {
		return nil
	}
	info := make([]map[string]any, 0, len(r.order))
	for _, name := range r.order {
		p := r.providers[name]
		entry := map[string]any{
			"name":      p.Name(),
			"base_path": p.BasePath(),
		}
		if streamable, ok := p.(Streamable); ok {
			entry["channels"] = streamable.Channels()
		}
		if renderable, ok := p.(Renderable); ok {
			entry["element"] = renderable.Element()
		}
		info = append(info, entry)
	}
	return info
}
