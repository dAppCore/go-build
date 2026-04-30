package provider

func ExampleNewRegistry() {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
}

func ExampleRegistry_Add() {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
}

func ExampleRegistry_Get() {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
	_ = registry.Get("build")
}

func ExampleRegistry_Info() {
	registry := NewRegistry()
	registry.Add(registryTestProvider{name: "build", basePath: "/build"})
	_ = registry.Info()
}
