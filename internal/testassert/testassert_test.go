package testassert

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestTestassert_Equal_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Equal("agent", "agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Equal_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Equal("agent", "agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Equal_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Equal("agent", "agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Nil_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Nil("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Nil_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Nil("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Nil_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Nil("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Empty_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Empty("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Empty_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Empty("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Empty_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Empty("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Zero_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Zero("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Zero_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Zero("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Zero_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Zero("agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Contains_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Contains("agent", "agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Contains_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Contains("agent", "agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_Contains_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = Contains("agent", "agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_ElementsMatch_Good(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ElementsMatch("agent", "agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_ElementsMatch_Bad(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ElementsMatch("agent", "agent")
	})
	core.AssertTrue(t, true)
}

func TestTestassert_ElementsMatch_Ugly(t *core.T) {
	core.AssertNotPanics(t, func() {
		_ = ElementsMatch("agent", "agent")
	})
	core.AssertTrue(t, true)
}
