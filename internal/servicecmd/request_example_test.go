package servicecmd

import (
	core "dappco.re/go"
	buildservice "dappco.re/go/build/pkg/service"
)

// --- v0.9.0 generated usage examples ---
func ExampleFromOptions() {
	_ = FromOptions(core.NewOptions())
	core.Println("FromOptions")
	// Output: FromOptions
}

func ExampleLoadConfig() {
	_, _ = LoadConfig(Request{}, func() (string, error) {
		return "", nil
	}, func(string) (buildservice.Config, error) {
		return buildservice.Config{}, nil
	})
	core.Println("LoadConfig")
	// Output: LoadConfig
}

func ExampleApplyOverrides() {
	_ = ApplyOverrides(nil, Request{})
	core.Println("ApplyOverrides")
	// Output: ApplyOverrides
}

func ExampleParseCSV() {
	_ = ParseCSV("agent")
	core.Println("ParseCSV")
	// Output: ParseCSV
}
