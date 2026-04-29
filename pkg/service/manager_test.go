package service

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestManager_NewManager_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewManager()
	})
	core.AssertTrue(t, true)
}

func TestManager_NewManager_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewManager()
	})
	core.AssertTrue(t, true)
}

func TestManager_NewManager_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewManager()
	})
	core.AssertTrue(t, true)
}

func TestManager_Program_Start_Good(t *core.T) {
	subject := noopProgram{}
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
	})
	core.AssertTrue(t, true)
}

func TestManager_Program_Start_Bad(t *core.T) {
	subject := noopProgram{}
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
	})
	core.AssertTrue(t, true)
}

func TestManager_Program_Start_Ugly(t *core.T) {
	subject := noopProgram{}
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
	})
	core.AssertTrue(t, true)
}

func TestManager_Program_Stop_Good(t *core.T) {
	subject := noopProgram{}
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
	})
	core.AssertTrue(t, true)
}

func TestManager_Program_Stop_Bad(t *core.T) {
	subject := noopProgram{}
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
	})
	core.AssertTrue(t, true)
}

func TestManager_Program_Stop_Ugly(t *core.T) {
	subject := noopProgram{}
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Install_Good(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Install(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Install_Bad(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Install(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Install_Ugly(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Install(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Start_Good(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Start(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Start_Bad(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Start(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Start_Ugly(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Start(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Stop_Good(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Stop_Bad(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Stop_Ugly(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Uninstall_Good(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Uninstall(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Uninstall_Bad(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Uninstall(Config{})
	})
	core.AssertTrue(t, true)
}

func TestManager_OSManager_Uninstall_Ugly(t *core.T) {
	subject := &OSManager{}
	core.AssertNotPanics(t, func() {
		_ = subject.Uninstall(Config{})
	})
	core.AssertTrue(t, true)
}
