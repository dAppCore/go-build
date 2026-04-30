package api

import (
	"context"
	"net/http"
	"net/http/httptest"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

type httpTestGroup struct {
	name string
	base string
}

func (g httpTestGroup) Name() string { return g.name }

func (g httpTestGroup) BasePath() string { return g.base }

func (g httpTestGroup) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
}

func TestHttp_OK_Good(t *core.T) {
	value := OK("data").(response)
	core.AssertTrue(t, value.Success)
	core.AssertEqual(t, "data", value.Data.(string))
}

func TestHttp_OK_Bad(t *core.T) {
	value := OK(nil).(response)
	core.AssertTrue(t, value.Success)
	core.AssertEqual(t, nil, value.Data)
}

func TestHttp_OK_Ugly(t *core.T) {
	payload := map[string]any{"count": 1}
	value := OK(payload).(response)
	core.AssertEqual(t, payload, value.Data)
	core.AssertEqual(t, "", value.Error)
}

func TestHttp_Fail_Good(t *core.T) {
	value := Fail("bad_request", "invalid").(response)
	core.AssertFalse(t, value.Success)
	core.AssertEqual(t, "bad_request", value.Code)
}

func TestHttp_Fail_Bad(t *core.T) {
	value := Fail("", "").(response)
	core.AssertFalse(t, value.Success)
	core.AssertEqual(t, "", value.Error)
}

func TestHttp_Fail_Ugly(t *core.T) {
	value := Fail("conflict", "already exists").(response)
	core.AssertEqual(t, "already exists", value.Error)
	core.AssertEqual(t, "conflict", value.Code)
}

func TestHttp_FailWithDetails_Good(t *core.T) {
	value := FailWithDetails("bad_request", "invalid", "field").(response)
	core.AssertFalse(t, value.Success)
	core.AssertEqual(t, "field", value.Details.(string))
}

func TestHttp_FailWithDetails_Bad(t *core.T) {
	value := FailWithDetails("", "", nil).(response)
	core.AssertFalse(t, value.Success)
	core.AssertEqual(t, nil, value.Details)
}

func TestHttp_FailWithDetails_Ugly(t *core.T) {
	details := map[string]any{"field": "name"}
	value := FailWithDetails("invalid", "bad", details).(response)
	core.AssertEqual(t, details, value.Details)
	core.AssertEqual(t, "invalid", value.Code)
}

func TestHttp_WithAddr_Good(t *core.T) {
	result := New(WithAddr("127.0.0.1:0"))
	engine := result.Value.(*Engine)
	core.AssertEqual(t, "127.0.0.1:0", engine.addr)
}

func TestHttp_WithAddr_Bad(t *core.T) {
	result := New(WithAddr(""))
	engine := result.Value.(*Engine)
	core.AssertEqual(t, "", engine.addr)
}

func TestHttp_WithAddr_Ugly(t *core.T) {
	result := New(WithAddr(":0"), WithAddr("127.0.0.1:0"))
	engine := result.Value.(*Engine)
	core.AssertEqual(t, "127.0.0.1:0", engine.addr)
}

func TestHttp_WithWSPath_Good(t *core.T) {
	result := New(WithWSPath("/ws"))
	engine := result.Value.(*Engine)
	core.AssertEqual(t, "/ws", engine.wsPath)
}

func TestHttp_WithWSPath_Bad(t *core.T) {
	result := New(WithWSPath(""))
	engine := result.Value.(*Engine)
	core.AssertEqual(t, "", engine.wsPath)
}

func TestHttp_WithWSPath_Ugly(t *core.T) {
	result := New(WithWSPath("/first"), WithWSPath("/second"))
	engine := result.Value.(*Engine)
	core.AssertEqual(t, "/second", engine.wsPath)
}

func TestHttp_WithWSHandler_Good(t *core.T) {
	handler := func(http.ResponseWriter, *http.Request) {}
	result := New(WithWSHandler(handler))
	engine := result.Value.(*Engine)
	core.AssertFalse(t, engine.wsHandler == nil)
}

func TestHttp_WithWSHandler_Bad(t *core.T) {
	result := New(WithWSHandler(nil))
	engine := result.Value.(*Engine)
	core.AssertTrue(t, engine.wsHandler == nil)
}

func TestHttp_WithWSHandler_Ugly(t *core.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }
	result := New(WithWSPath("/ws"), WithWSHandler(handler))
	engine := result.Value.(*Engine)
	core.AssertFalse(t, engine.wsHandler == nil)
}

func TestHttp_New_Good(t *core.T) {
	result := New()
	engine := result.Value.(*Engine)
	core.AssertTrue(t, result.OK)
	core.AssertFalse(t, engine.router == nil)
}

func TestHttp_New_Bad(t *core.T) {
	result := New(WithAddr(""))
	engine := result.Value.(*Engine)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "", engine.addr)
}

func TestHttp_New_Ugly(t *core.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }
	result := New(WithWSPath("/ws"), WithWSHandler(handler))
	recorder := httptest.NewRecorder()
	result.Value.(*Engine).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))
	core.AssertEqual(t, http.StatusNoContent, recorder.Code)
}

func TestHttp_Engine_Register_Good(t *core.T) {
	result := New()
	engine := result.Value.(*Engine)
	engine.Register(httpTestGroup{name: "test", base: "/test"})
	recorder := httptest.NewRecorder()
	engine.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test/ping", nil))
	core.AssertEqual(t, http.StatusOK, recorder.Code)
}

func TestHttp_Engine_Register_Bad(t *core.T) {
	var engine *Engine
	engine.Register(httpTestGroup{name: "test", base: "/test"})
	core.AssertTrue(t, engine == nil)
}

func TestHttp_Engine_Register_Ugly(t *core.T) {
	result := New()
	engine := result.Value.(*Engine)
	engine.Register(nil)
	core.AssertFalse(t, engine.router == nil)
}

func TestHttp_Engine_Serve_Good(t *core.T) {
	var engine *Engine
	result := engine.Serve(context.Background())
	core.AssertTrue(t, result.OK)
}

func TestHttp_Engine_Serve_Bad(t *core.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine := New(WithAddr("127.0.0.1:0")).Value.(*Engine)
	result := engine.Serve(ctx)
	core.AssertFalse(t, result.OK)
}

func TestHttp_Engine_Serve_Ugly(t *core.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine := &Engine{router: gin.New(), addr: "127.0.0.1:0"}
	result := engine.Serve(ctx)
	core.AssertFalse(t, result.OK)
}

func TestHttp_Engine_Handler_Good(t *core.T) {
	engine := New().Value.(*Engine)
	handler := engine.Handler()
	core.AssertFalse(t, handler == nil)
}

func TestHttp_Engine_Handler_Bad(t *core.T) {
	var engine *Engine
	handler := engine.Handler()
	core.AssertFalse(t, handler == nil)
}

func TestHttp_Engine_Handler_Ugly(t *core.T) {
	engine := New().Value.(*Engine)
	recorder := httptest.NewRecorder()
	engine.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missing", nil))
	core.AssertEqual(t, http.StatusNotFound, recorder.Code)
}

func TestHttp_NewToolBridge_Good(t *core.T) {
	bridge := NewToolBridge("/tools")
	core.AssertEqual(t, "/tools", bridge.BasePath())
	core.AssertEqual(t, "mcp", bridge.Name())
}

func TestHttp_NewToolBridge_Bad(t *core.T) {
	bridge := NewToolBridge("")
	core.AssertEqual(t, "", bridge.BasePath())
	core.AssertEqual(t, 0, len(bridge.Tools()))
}

func TestHttp_NewToolBridge_Ugly(t *core.T) {
	bridge := NewToolBridge("/tools/")
	core.AssertTrue(t, core.HasSuffix(bridge.BasePath(), "/"))
	core.AssertEqual(t, "mcp", bridge.Name())
}

func TestHttp_ToolBridge_Name_Good(t *core.T) {
	bridge := NewToolBridge("/tools")
	name := bridge.Name()
	core.AssertEqual(t, "mcp", name)
}

func TestHttp_ToolBridge_Name_Bad(t *core.T) {
	var bridge *ToolBridge
	name := bridge.Name()
	core.AssertEqual(t, "mcp", name)
}

func TestHttp_ToolBridge_Name_Ugly(t *core.T) {
	bridge := NewToolBridge("")
	name := bridge.Name()
	core.AssertTrue(t, core.Contains(name, "mcp"))
}

func TestHttp_ToolBridge_BasePath_Good(t *core.T) {
	bridge := NewToolBridge("/tools")
	basePath := bridge.BasePath()
	core.AssertEqual(t, "/tools", basePath)
}

func TestHttp_ToolBridge_BasePath_Bad(t *core.T) {
	var bridge *ToolBridge
	basePath := bridge.BasePath()
	core.AssertEqual(t, "", basePath)
}

func TestHttp_ToolBridge_BasePath_Ugly(t *core.T) {
	bridge := NewToolBridge("")
	basePath := bridge.BasePath()
	core.AssertEqual(t, "", basePath)
}

func TestHttp_ToolBridge_Add_Good(t *core.T) {
	bridge := NewToolBridge("/tools")
	bridge.Add(ToolDescriptor{Name: "build"}, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	tools := bridge.Tools()
	core.AssertEqual(t, "build", tools[0].Name)
}

func TestHttp_ToolBridge_Add_Bad(t *core.T) {
	var bridge *ToolBridge
	bridge.Add(ToolDescriptor{Name: "build"}, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	core.AssertTrue(t, bridge == nil)
}

func TestHttp_ToolBridge_Add_Ugly(t *core.T) {
	bridge := NewToolBridge("/tools")
	bridge.Add(ToolDescriptor{}, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	core.AssertEqual(t, "", bridge.Tools()[0].Name)
}

func TestHttp_ToolBridge_RegisterRoutes_Good(t *core.T) {
	bridge := NewToolBridge("/tools")
	bridge.Add(ToolDescriptor{Name: "build"}, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router := gin.New()
	bridge.RegisterRoutes(router.Group(bridge.BasePath()))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/tools/build", nil))
	core.AssertEqual(t, http.StatusNoContent, recorder.Code)
}

func TestHttp_ToolBridge_RegisterRoutes_Bad(t *core.T) {
	var bridge *ToolBridge
	router := gin.New()
	bridge.RegisterRoutes(router.Group("/tools"))
	core.AssertEqual(t, 0, len(router.Routes()))
}

func TestHttp_ToolBridge_RegisterRoutes_Ugly(t *core.T) {
	bridge := NewToolBridge("/tools")
	router := gin.New()
	bridge.RegisterRoutes(router.Group("/tools"))
	core.AssertEqual(t, 0, len(router.Routes()))
}

func TestHttp_ToolBridge_Tools_Good(t *core.T) {
	bridge := NewToolBridge("/tools")
	bridge.Add(ToolDescriptor{Name: "build"}, nil)
	tools := bridge.Tools()
	core.AssertEqual(t, 1, len(tools))
}

func TestHttp_ToolBridge_Tools_Bad(t *core.T) {
	var bridge *ToolBridge
	tools := bridge.Tools()
	core.AssertEqual(t, 0, len(tools))
}

func TestHttp_ToolBridge_Tools_Ugly(t *core.T) {
	bridge := NewToolBridge("/tools")
	bridge.Add(ToolDescriptor{Name: "one"}, nil)
	bridge.Add(ToolDescriptor{Name: "two"}, nil)
	core.AssertEqual(t, 2, len(bridge.Tools()))
}
