package apple

import core "dappco.re/go"

// ExampleRegister references Register on this package API surface.
func ExampleRegister() {
	_ = Register
	core.Println("Register")
	// Output: Register
}

// ExampleNew references New on this package API surface.
func ExampleNew() {
	_ = New
	core.Println("New")
	// Output: New
}

// ExampleWithArch references WithArch on this package API surface.
func ExampleWithArch() {
	_ = WithArch
	core.Println("WithArch")
	// Output: WithArch
}

// ExampleWithSign references WithSign on this package API surface.
func ExampleWithSign() {
	_ = WithSign
	core.Println("WithSign")
	// Output: WithSign
}

// ExampleWithNotarise references WithNotarise on this package API surface.
func ExampleWithNotarise() {
	_ = WithNotarise
	core.Println("WithNotarise")
	// Output: WithNotarise
}

// ExampleWithDMG references WithDMG on this package API surface.
func ExampleWithDMG() {
	_ = WithDMG
	core.Println("WithDMG")
	// Output: WithDMG
}

// ExampleWithTestFlight references WithTestFlight on this package API surface.
func ExampleWithTestFlight() {
	_ = WithTestFlight
	core.Println("WithTestFlight")
	// Output: WithTestFlight
}

// ExampleWithAppStore references WithAppStore on this package API surface.
func ExampleWithAppStore() {
	_ = WithAppStore
	core.Println("WithAppStore")
	// Output: WithAppStore
}

// ExampleAppleBuilder_Name references AppleBuilder.Name on this package API surface.
func ExampleAppleBuilder_Name() {
	_ = (*AppleBuilder).Name
	core.Println("AppleBuilder.Name")
	// Output: AppleBuilder.Name
}

// ExampleAppleBuilder_Detect references AppleBuilder.Detect on this package API surface.
func ExampleAppleBuilder_Detect() {
	_ = (*AppleBuilder).Detect
	core.Println("AppleBuilder.Detect")
	// Output: AppleBuilder.Detect
}

// ExampleAppleBuilder_Build references AppleBuilder.Build on this package API surface.
func ExampleAppleBuilder_Build() {
	_ = (*AppleBuilder).Build
	core.Println("AppleBuilder.Build")
	// Output: AppleBuilder.Build
}

// ExampleBuildWailsApp references BuildWailsApp on this package API surface.
func ExampleBuildWailsApp() {
	_ = BuildWailsApp
	core.Println("BuildWailsApp")
	// Output: BuildWailsApp
}

// ExampleCreateUniversal references CreateUniversal on this package API surface.
func ExampleCreateUniversal() {
	_ = CreateUniversal
	core.Println("CreateUniversal")
	// Output: CreateUniversal
}

// ExampleSign references Sign on this package API surface.
func ExampleSign() {
	_ = Sign
	core.Println("Sign")
	// Output: Sign
}

// ExampleNotarise references Notarise on this package API surface.
func ExampleNotarise() {
	_ = Notarise
	core.Println("Notarise")
	// Output: Notarise
}

// ExampleCreateDMG references CreateDMG on this package API surface.
func ExampleCreateDMG() {
	_ = CreateDMG
	core.Println("CreateDMG")
	// Output: CreateDMG
}

// ExampleUploadTestFlight references UploadTestFlight on this package API surface.
func ExampleUploadTestFlight() {
	_ = UploadTestFlight
	core.Println("UploadTestFlight")
	// Output: UploadTestFlight
}

// ExampleSubmitAppStore references SubmitAppStore on this package API surface.
func ExampleSubmitAppStore() {
	_ = SubmitAppStore
	core.Println("SubmitAppStore")
	// Output: SubmitAppStore
}
