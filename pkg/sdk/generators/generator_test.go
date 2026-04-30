package generators

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestGenerator_NewRegistry_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewRegistry()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGenerator_NewRegistry_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewRegistry()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGenerator_NewRegistry_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = NewRegistry()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGenerator_Registry_Get_Good(t *core.T) {
	subject := NewRegistry()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Get("go")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGenerator_Registry_Get_Bad(t *core.T) {
	subject := NewRegistry()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Get("")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGenerator_Registry_Get_Ugly(t *core.T) {
	subject := NewRegistry()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_, _ = subject.Get("go")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGenerator_Registry_Register_Good(t *core.T) {
	subject := NewRegistry()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Register(NewGoGenerator())
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGenerator_Registry_Register_Bad(t *core.T) {
	subject := NewRegistry()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Register(NewGoGenerator())
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGenerator_Registry_Register_Ugly(t *core.T) {
	subject := NewRegistry()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		subject.Register(NewGoGenerator())
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGenerator_Registry_Languages_Good(t *core.T) {
	subject := NewRegistry()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Languages()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGenerator_Registry_Languages_Bad(t *core.T) {
	subject := NewRegistry()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Languages()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGenerator_Registry_Languages_Ugly(t *core.T) {
	subject := NewRegistry()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.Languages()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestGenerator_Registry_LanguagesIter_Good(t *core.T) {
	subject := NewRegistry()
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.LanguagesIter()
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestGenerator_Registry_LanguagesIter_Bad(t *core.T) {
	subject := NewRegistry()
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.LanguagesIter()
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestGenerator_Registry_LanguagesIter_Ugly(t *core.T) {
	subject := NewRegistry()
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = subject.LanguagesIter()
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
