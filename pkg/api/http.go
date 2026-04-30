package api

import (
	"context"
	"net/http"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

type RouteDescription struct {
	Method      string         `json:"method,omitempty"`
	Path        string         `json:"path,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	RequestBody map[string]any `json:"requestBody,omitempty"`
	Responses   map[string]any `json:"responses,omitempty"`
}

type RouteGroup interface {
	Name() string
	BasePath() string
	RegisterRoutes(*gin.RouterGroup)
}

type DescribableGroup interface {
	RouteGroup
	Describe() []RouteDescription
}

type response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

func OK(data any) any {
	return response{Success: true, Data: data}
}

func Fail(code, message string) any {
	return response{Success: false, Code: code, Error: message}
}

func FailWithDetails(code, message string, details any) any {
	return response{Success: false, Code: code, Error: message, Details: details}
}

type Option func(*Engine)

func WithAddr(addr string) Option {
	return func(e *Engine) { e.addr = addr }
}

func WithWSPath(path string) Option {
	return func(e *Engine) { e.wsPath = path }
}

func WithWSHandler(handler http.HandlerFunc) Option {
	return func(e *Engine) { e.wsHandler = handler }
}

type Engine struct {
	router    *gin.Engine
	addr      string
	wsPath    string
	wsHandler http.HandlerFunc
}

func New(opts ...Option) core.Result {
	engine := &Engine{router: gin.New()}
	for _, opt := range opts {
		opt(engine)
	}
	if engine.wsPath != "" && engine.wsHandler != nil {
		engine.router.GET(engine.wsPath, gin.WrapF(engine.wsHandler))
	}
	return core.Ok(engine)
}

func (e *Engine) Register(group RouteGroup) {
	if e == nil || group == nil {
		return
	}
	group.RegisterRoutes(e.router.Group(group.BasePath()))
}

func (e *Engine) Serve(ctx context.Context) core.Result {
	if e == nil {
		return core.Ok(nil)
	}
	server := &http.Server{Addr: e.addr, Handler: e.router}
	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
		if err := server.Shutdown(context.Background()); err != nil {
			return core.Fail(err)
		}
		return core.Fail(ctx.Err())
	case err := <-errCh:
		return core.ResultOf(nil, err)
	}
}

func (e *Engine) Handler() http.Handler {
	if e == nil {
		return http.NewServeMux()
	}
	return e.router
}

type ToolDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Group       string `json:"group,omitempty"`
}

type ToolBridge struct {
	basePath string
	tools    []toolRoute
}

type toolRoute struct {
	descriptor ToolDescriptor
	handler    gin.HandlerFunc
}

func NewToolBridge(basePath string) *ToolBridge {
	return &ToolBridge{basePath: basePath}
}

func (b *ToolBridge) Name() string { return "mcp" }

func (b *ToolBridge) BasePath() string {
	if b == nil {
		return ""
	}
	return b.basePath
}

func (b *ToolBridge) Add(descriptor ToolDescriptor, handler gin.HandlerFunc) {
	if b == nil {
		return
	}
	b.tools = append(b.tools, toolRoute{descriptor: descriptor, handler: handler})
}

func (b *ToolBridge) RegisterRoutes(group *gin.RouterGroup) {
	if b == nil {
		return
	}
	for _, tool := range b.tools {
		group.POST("/"+tool.descriptor.Name, tool.handler)
	}
}

func (b *ToolBridge) Tools() []ToolDescriptor {
	if b == nil {
		return nil
	}
	out := make([]ToolDescriptor, 0, len(b.tools))
	for _, tool := range b.tools {
		out = append(out, tool.descriptor)
	}
	return out
}
