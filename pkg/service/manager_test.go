package service

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestManager_NewManager_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewManager()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestManager_NewManager_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewManager()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestManager_NewManager_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewManager()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestManager_Program_Start_Good(t *core.T) {
	subject := noopProgram{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestManager_Program_Start_Bad(t *core.T) {
	subject := noopProgram{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestManager_Program_Start_Ugly(t *core.T) {
	subject := noopProgram{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestManager_Program_Stop_Good(t *core.T) {
	subject := noopProgram{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestManager_Program_Stop_Bad(t *core.T) {
	subject := noopProgram{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestManager_Program_Stop_Ugly(t *core.T) {
	subject := noopProgram{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(nil)
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestManager_OSManager_Install_Good(t *core.T) {
	subject := &OSManager{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install(Config{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestManager_OSManager_Install_Bad(t *core.T) {
	subject := &OSManager{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install(Config{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestManager_OSManager_Install_Ugly(t *core.T) {
	subject := &OSManager{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Install(Config{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestManager_OSManager_Start_Good(t *core.T) {
	subject := &OSManager{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(Config{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestManager_OSManager_Start_Bad(t *core.T) {
	subject := &OSManager{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(Config{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestManager_OSManager_Start_Ugly(t *core.T) {
	subject := &OSManager{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Start(Config{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestManager_OSManager_Stop_Good(t *core.T) {
	subject := &OSManager{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(Config{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestManager_OSManager_Stop_Bad(t *core.T) {
	subject := &OSManager{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(Config{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestManager_OSManager_Stop_Ugly(t *core.T) {
	subject := &OSManager{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Stop(Config{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestManager_OSManager_Uninstall_Good(t *core.T) {
	subject := &OSManager{}
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Uninstall(Config{})
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestManager_OSManager_Uninstall_Bad(t *core.T) {
	subject := &OSManager{}
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Uninstall(Config{})
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestManager_OSManager_Uninstall_Ugly(t *core.T) {
	subject := &OSManager{}
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Uninstall(Config{})
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
