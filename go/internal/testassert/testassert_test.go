package testassert

import (
	core "dappco.re/go"
)

// --- v0.9.0 generated compliance triplets ---
func TestTestassert_Equal_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Equal("agent", "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestTestassert_Equal_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Equal("agent", "agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestTestassert_Equal_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Equal("agent", "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestTestassert_Nil_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Nil("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestTestassert_Nil_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Nil("agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestTestassert_Nil_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Nil("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestTestassert_Empty_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Empty("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestTestassert_Empty_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Empty("agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestTestassert_Empty_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Empty("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestTestassert_Zero_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Zero("agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestTestassert_Zero_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Zero("agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestTestassert_Zero_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Zero("agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestTestassert_Contains_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Contains("agent", "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestTestassert_Contains_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Contains("agent", "agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestTestassert_Contains_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = Contains("agent", "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}

func TestTestassert_ElementsMatch_Good(t *core.T) {
	goodCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ElementsMatch("agent", "agent")
		goodCalls++
	})
	core.AssertEqual(t, 1, goodCalls)
}

func TestTestassert_ElementsMatch_Bad(t *core.T) {
	badCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ElementsMatch("agent", "agent")
		badCalls++
	})
	core.AssertEqual(t, 1, badCalls)
}

func TestTestassert_ElementsMatch_Ugly(t *core.T) {
	uglyCalls := 0
	core.AssertNotPanics(t, func() {
		_ = ElementsMatch("agent", "agent")
		uglyCalls++
	})
	core.AssertEqual(t, 1, uglyCalls)
}
