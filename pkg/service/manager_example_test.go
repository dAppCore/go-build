package service

import (
	core "dappco.re/go"
	nativeservice "github.com/kardianos/service"
)

type Program = noopProgram

type noopProgram struct{}

func (noopProgram) Start(nativeservice.Service) error {
	return nil
}

func (noopProgram) Stop(nativeservice.Service) error {
	return nil
}

// --- v0.9.0 generated usage examples ---
func ExampleNewManager() {
	_ = NewManager()
	core.Println("NewManager")
	// Output: NewManager
}

func ExampleProgram_Start() {
	subject := noopProgram{}
	_ = subject.Start(nil)
	core.Println("Program_Start")
	// Output: Program_Start
}

func ExampleProgram_Stop() {
	subject := noopProgram{}
	_ = subject.Stop(nil)
	core.Println("Program_Stop")
	// Output: Program_Stop
}

func ExampleOSManager_Install() {
	subject := &OSManager{}
	_ = subject.Install(Config{})
	core.Println("OSManager_Install")
	// Output: OSManager_Install
}

func ExampleOSManager_Start() {
	subject := &OSManager{}
	_ = subject.Start(Config{})
	core.Println("OSManager_Start")
	// Output: OSManager_Start
}

func ExampleOSManager_Stop() {
	subject := &OSManager{}
	_ = subject.Stop(Config{})
	core.Println("OSManager_Stop")
	// Output: OSManager_Stop
}

func ExampleOSManager_Uninstall() {
	subject := &OSManager{}
	_ = subject.Uninstall(Config{})
	core.Println("OSManager_Uninstall")
	// Output: OSManager_Uninstall
}
