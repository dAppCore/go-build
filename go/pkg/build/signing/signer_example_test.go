package signing

import core "dappco.re/go"

// ExampleDefaultSignConfig references DefaultSignConfig on this package API surface.
func ExampleDefaultSignConfig() {
	_ = DefaultSignConfig
	core.Println("DefaultSignConfig")
	// Output: DefaultSignConfig
}

// ExampleSignConfig_ExpandEnv references SignConfig.ExpandEnv on this package API surface.
func ExampleSignConfig_ExpandEnv() {
	_ = (*SignConfig).ExpandEnv
	core.Println("SignConfig.ExpandEnv")
	// Output: SignConfig.ExpandEnv
}

// ExampleWindowsConfig_SetSigntool references WindowsConfig.SetSigntool on this package API surface.
func ExampleWindowsConfig_SetSigntool() {
	_ = (*WindowsConfig).SetSigntool
	core.Println("WindowsConfig.SetSigntool")
	// Output: WindowsConfig.SetSigntool
}
