package api

import (
	core "dappco.re/go"
	gin "github.com/gin-gonic/gin"
)

// --- v0.9.0 generated usage examples ---
func ExampleNewProvider() {
	_ = NewProvider(core.Path(core.TempDir(), "go-build-compliance"), nil)
	core.Println("NewProvider")
	// Output: NewProvider
}

func ExampleBuildProvider_Name() {
	subject := &BuildProvider{}
	_ = subject.Name()
	core.Println("BuildProvider_Name")
	// Output: BuildProvider_Name
}

func ExampleBuildProvider_BasePath() {
	subject := &BuildProvider{}
	_ = subject.BasePath()
	core.Println("BuildProvider_BasePath")
	// Output: BuildProvider_BasePath
}

func ExampleBuildProvider_Element() {
	subject := &BuildProvider{}
	_ = subject.Element()
	core.Println("BuildProvider_Element")
	// Output: BuildProvider_Element
}

func ExampleBuildProvider_Channels() {
	subject := &BuildProvider{}
	_ = subject.Channels()
	core.Println("BuildProvider_Channels")
	// Output: BuildProvider_Channels
}

func ExampleBuildProvider_RegisterRoutes() {
	subject := &BuildProvider{}
	subject.RegisterRoutes(gin.New().Group("/build"))
	core.Println("BuildProvider_RegisterRoutes")
	// Output: BuildProvider_RegisterRoutes
}

func ExampleBuildProvider_Describe() {
	subject := &BuildProvider{}
	_ = subject.Describe()
	core.Println("BuildProvider_Describe")
	// Output: BuildProvider_Describe
}

func ExampleInfo_MarshalJSON() {
	subject := Info{Name: "app.tar.gz", Path: "/dist/app.tar.gz", Size: 42}
	data, _ := subject.MarshalJSON()
	core.Println(core.Contains(string(data), "app.tar.gz"))
	// Output: true
}

func ExampleReleaseWorkflowRequest_UnmarshalJSON() {
	var subject ReleaseWorkflowRequest
	_ = subject.UnmarshalJSON([]byte(`{"` + apiPathField + `":"ci/release.yml"}`))
	core.Println(subject.Path)
	// Output: ci/release.yml
}
