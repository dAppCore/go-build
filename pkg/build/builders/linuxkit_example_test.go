package builders

import core "dappco.re/go"

// ExampleNewLinuxKitBuilder references NewLinuxKitBuilder on this package API surface.
func ExampleNewLinuxKitBuilder() {
	_ = NewLinuxKitBuilder
	core.Println("NewLinuxKitBuilder")
	// Output: NewLinuxKitBuilder
}

// ExampleLinuxKitBuilder_Name references LinuxKitBuilder.Name on this package API surface.
func ExampleLinuxKitBuilder_Name() {
	_ = (*LinuxKitBuilder).Name
	core.Println("LinuxKitBuilder.Name")
	// Output: LinuxKitBuilder.Name
}

// ExampleLinuxKitBuilder_Detect references LinuxKitBuilder.Detect on this package API surface.
func ExampleLinuxKitBuilder_Detect() {
	_ = (*LinuxKitBuilder).Detect
	core.Println("LinuxKitBuilder.Detect")
	// Output: LinuxKitBuilder.Detect
}

// ExampleLinuxKitBuilder_Build references LinuxKitBuilder.Build on this package API surface.
func ExampleLinuxKitBuilder_Build() {
	_ = (*LinuxKitBuilder).Build
	core.Println("LinuxKitBuilder.Build")
	// Output: LinuxKitBuilder.Build
}
