package builders

import core "dappco.re/go"

// ExampleNewDocsBuilder references NewDocsBuilder on this package API surface.
func ExampleNewDocsBuilder() {
	_ = NewDocsBuilder
	core.Println("NewDocsBuilder")
	// Output: NewDocsBuilder
}

// ExampleDocsBuilder_Name references DocsBuilder.Name on this package API surface.
func ExampleDocsBuilder_Name() {
	_ = (*DocsBuilder).Name
	core.Println("DocsBuilder.Name")
	// Output: DocsBuilder.Name
}

// ExampleDocsBuilder_Detect references DocsBuilder.Detect on this package API surface.
func ExampleDocsBuilder_Detect() {
	_ = (*DocsBuilder).Detect
	core.Println("DocsBuilder.Detect")
	// Output: DocsBuilder.Detect
}

// ExampleDocsBuilder_Build references DocsBuilder.Build on this package API surface.
func ExampleDocsBuilder_Build() {
	_ = (*DocsBuilder).Build
	core.Println("DocsBuilder.Build")
	// Output: DocsBuilder.Build
}
