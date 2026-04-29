package buildcmd

import core "dappco.re/go"

type Program = serviceProgram
type ServiceProgram = controlServiceProgram

// ExampleProgram_Start references Program.Start on this package API surface.
func ExampleProgram_Start() {
	_ = (*Program).Start
	core.Println("Program.Start")
	// Output: Program.Start
}

// ExampleProgram_Stop references Program.Stop on this package API surface.
func ExampleProgram_Stop() {
	_ = (*Program).Stop
	core.Println("Program.Stop")
	// Output: Program.Stop
}

// ExampleServiceProgram_Start references ServiceProgram.Start on this package API surface.
func ExampleServiceProgram_Start() {
	_ = (*ServiceProgram).Start
	core.Println("ServiceProgram.Start")
	// Output: ServiceProgram.Start
}

// ExampleServiceProgram_Stop references ServiceProgram.Stop on this package API surface.
func ExampleServiceProgram_Stop() {
	_ = (*ServiceProgram).Stop
	core.Println("ServiceProgram.Stop")
	// Output: ServiceProgram.Stop
}

// ExampleAddServiceCommands references AddServiceCommands on this package API surface.
func ExampleAddServiceCommands() {
	_ = AddServiceCommands
	core.Println("AddServiceCommands")
	// Output: AddServiceCommands
}
