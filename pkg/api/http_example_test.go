package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ExampleOK() {
	_ = OK(map[string]any{"status": "ok"})
}

func ExampleFail() {
	_ = Fail("bad_request", "invalid request")
}

func ExampleFailWithDetails() {
	_ = FailWithDetails("bad_request", "invalid request", map[string]any{"field": "name"})
}

func ExampleWithAddr() {
	_ = New(WithAddr("127.0.0.1:0"))
}

func ExampleWithWSPath() {
	_ = New(WithWSPath("/ws"))
}

func ExampleWithWSHandler() {
	handler := func(http.ResponseWriter, *http.Request) {}
	_ = New(WithWSHandler(handler))
}

func ExampleNew() {
	_ = New(WithAddr("127.0.0.1:0"))
}

func ExampleEngine_Register() {
	engine := New().Value.(*Engine)
	engine.Register(httpTestGroup{name: "test", base: "/test"})
}

func ExampleEngine_Serve() {
	var engine *Engine
	_ = engine.Serve(context.Background())
}

func ExampleEngine_Handler() {
	engine := New().Value.(*Engine)
	_ = engine.Handler()
}

func ExampleNewToolBridge() {
	bridge := NewToolBridge("/tools")
	_ = bridge.Tools()
}

func ExampleToolBridge_Name() {
	bridge := NewToolBridge("/tools")
	_ = bridge.Name()
}

func ExampleToolBridge_BasePath() {
	bridge := NewToolBridge("/tools")
	_ = bridge.BasePath()
}

func ExampleToolBridge_Add() {
	bridge := NewToolBridge("/tools")
	bridge.Add(ToolDescriptor{Name: "build"}, func(c *gin.Context) {})
}

func ExampleToolBridge_RegisterRoutes() {
	bridge := NewToolBridge("/tools")
	router := gin.New()
	bridge.RegisterRoutes(router.Group(bridge.BasePath()))
}

func ExampleToolBridge_Tools() {
	bridge := NewToolBridge("/tools")
	bridge.Add(ToolDescriptor{Name: "build"}, func(c *gin.Context) {})
	_ = bridge.Tools()
}
