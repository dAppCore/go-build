package build

import core "dappco.re/go"

// ExampleDefaultAppleOptions references DefaultAppleOptions on this package API surface.
func ExampleDefaultAppleOptions() {
	_ = DefaultAppleOptions
	core.Println("DefaultAppleOptions")
	// Output: DefaultAppleOptions
}

// ExampleAppleConfig_Resolve references AppleConfig.Resolve on this package API surface.
func ExampleAppleConfig_Resolve() {
	_ = (*AppleConfig).Resolve
	core.Println("AppleConfig.Resolve")
	// Output: AppleConfig.Resolve
}

// ExampleBuildApple references BuildApple on this package API surface.
func ExampleBuildApple() {
	_ = BuildApple
	core.Println("BuildApple")
	// Output: BuildApple
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

// ExampleWriteInfoPlist references WriteInfoPlist on this package API surface.
func ExampleWriteInfoPlist() {
	_ = WriteInfoPlist
	core.Println("WriteInfoPlist")
	// Output: WriteInfoPlist
}

// ExampleWriteEntitlements references WriteEntitlements on this package API surface.
func ExampleWriteEntitlements() {
	_ = WriteEntitlements
	core.Println("WriteEntitlements")
	// Output: WriteEntitlements
}

// ExampleInfoPlist_Values references InfoPlist.Values on this package API surface.
func ExampleInfoPlist_Values() {
	_ = (*InfoPlist).Values
	core.Println("InfoPlist.Values")
	// Output: InfoPlist.Values
}

// ExampleEntitlements_Values references Entitlements.Values on this package API surface.
func ExampleEntitlements_Values() {
	_ = (*Entitlements).Values
	core.Println("Entitlements.Values")
	// Output: Entitlements.Values
}
