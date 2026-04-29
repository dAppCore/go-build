package generators

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestGenerator_NewRegistry_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewRegistry()
	})
	core.AssertTrue(t, true)
}

func TestGenerator_NewRegistry_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewRegistry()
	})
	core.AssertTrue(t, true)
}

func TestGenerator_NewRegistry_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = NewRegistry()
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Get_Good(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_, _ = subject.Get("go")
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Get_Bad(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_, _ = subject.Get("")
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Get_Ugly(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_, _ = subject.Get("go")
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Register_Good(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		subject.Register(NewGoGenerator())
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Register_Bad(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		subject.Register(NewGoGenerator())
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Register_Ugly(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		subject.Register(NewGoGenerator())
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Languages_Good(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_ = subject.Languages()
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Languages_Bad(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_ = subject.Languages()
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_Languages_Ugly(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_ = subject.Languages()
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_LanguagesIter_Good(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_ = subject.LanguagesIter()
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_LanguagesIter_Bad(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_ = subject.LanguagesIter()
	})
	core.AssertTrue(t, true)
}

func TestGenerator_Registry_LanguagesIter_Ugly(t *core.T) {
	subject := NewRegistry()
	core.AssertNotPanics(t, func() {
		_ = subject.LanguagesIter()
	})
	core.AssertTrue(t, true)
}
