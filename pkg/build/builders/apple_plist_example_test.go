package builders

import core "dappco.re/go"

// ExampleGenerateAppleInfoPlist references GenerateAppleInfoPlist on this package API surface.
func ExampleGenerateAppleInfoPlist() {
	_ = GenerateAppleInfoPlist
	core.Println("GenerateAppleInfoPlist")
	// Output: GenerateAppleInfoPlist
}

// ExampleWriteAppleInfoPlist references WriteAppleInfoPlist on this package API surface.
func ExampleWriteAppleInfoPlist() {
	_ = WriteAppleInfoPlist
	core.Println("WriteAppleInfoPlist")
	// Output: WriteAppleInfoPlist
}

// ExampleAppleInfoPlist_Values references AppleInfoPlist.Values on this package API surface.
func ExampleAppleInfoPlist_Values() {
	_ = (*AppleInfoPlist).Values
	core.Println("AppleInfoPlist.Values")
	// Output: AppleInfoPlist.Values
}

// ExampleDefaultAppleEntitlements references DefaultAppleEntitlements on this package API surface.
func ExampleDefaultAppleEntitlements() {
	_ = DefaultAppleEntitlements
	core.Println("DefaultAppleEntitlements")
	// Output: DefaultAppleEntitlements
}

// ExampleWriteAppleEntitlements references WriteAppleEntitlements on this package API surface.
func ExampleWriteAppleEntitlements() {
	_ = WriteAppleEntitlements
	core.Println("WriteAppleEntitlements")
	// Output: WriteAppleEntitlements
}

// ExampleAppleEntitlements_Values references AppleEntitlements.Values on this package API surface.
func ExampleAppleEntitlements_Values() {
	_ = (*AppleEntitlements).Values
	core.Println("AppleEntitlements.Values")
	// Output: AppleEntitlements.Values
}

// ExampleAppleBuilder_WriteXcodeCloudConfig references AppleBuilder.WriteXcodeCloudConfig on this package API surface.
func ExampleAppleBuilder_WriteXcodeCloudConfig() {
	_ = (*AppleBuilder).WriteXcodeCloudConfig
	core.Println("AppleBuilder.WriteXcodeCloudConfig")
	// Output: AppleBuilder.WriteXcodeCloudConfig
}
