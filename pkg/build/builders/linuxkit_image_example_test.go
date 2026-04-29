package builders

import core "dappco.re/go"

// ExampleNewLinuxKitImageBuilder references NewLinuxKitImageBuilder on this package API surface.
func ExampleNewLinuxKitImageBuilder() {
	_ = NewLinuxKitImageBuilder
	core.Println("NewLinuxKitImageBuilder")
	// Output: NewLinuxKitImageBuilder
}

// ExampleLinuxKitImageBuilder_Name references LinuxKitImageBuilder.Name on this package API surface.
func ExampleLinuxKitImageBuilder_Name() {
	_ = (*LinuxKitImageBuilder).Name
	core.Println("LinuxKitImageBuilder.Name")
	// Output: LinuxKitImageBuilder.Name
}

// ExampleLinuxKitImageBuilder_ListBaseImages references LinuxKitImageBuilder.ListBaseImages on this package API surface.
func ExampleLinuxKitImageBuilder_ListBaseImages() {
	_ = (*LinuxKitImageBuilder).ListBaseImages
	core.Println("LinuxKitImageBuilder.ListBaseImages")
	// Output: LinuxKitImageBuilder.ListBaseImages
}

// ExampleLinuxKitImageBuilder_ArtifactPath references LinuxKitImageBuilder.ArtifactPath on this package API surface.
func ExampleLinuxKitImageBuilder_ArtifactPath() {
	_ = (*LinuxKitImageBuilder).ArtifactPath
	core.Println("LinuxKitImageBuilder.ArtifactPath")
	// Output: LinuxKitImageBuilder.ArtifactPath
}

// ExampleLinuxKitImageBuilder_Build references LinuxKitImageBuilder.Build on this package API surface.
func ExampleLinuxKitImageBuilder_Build() {
	_ = (*LinuxKitImageBuilder).Build
	core.Println("LinuxKitImageBuilder.Build")
	// Output: LinuxKitImageBuilder.Build
}
