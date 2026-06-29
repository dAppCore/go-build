package testassert

import (
	core "dappco.re/go"
)

// Behaviour tests exercise the real branches of each predicate rather than the
// no-panic compliance triplets. They drive every reflect.Kind path so coverage
// reflects the actual decision tree.

type behaviourStruct struct {
	Name string
	Age  int
}

func TestTestassert_Equal_Behaviour_Good(t *core.T) {
	core.AssertTrue(t, Equal("go", "go"))
	core.AssertTrue(t, Equal(42, 42))
	core.AssertTrue(t, Equal([]string{"a", "b"}, []string{"a", "b"}))
	core.AssertTrue(t, Equal(behaviourStruct{Name: "x", Age: 1}, behaviourStruct{Name: "x", Age: 1}))
}

func TestTestassert_Equal_Behaviour_Bad(t *core.T) {
	core.AssertFalse(t, Equal("go", "node"))
	core.AssertFalse(t, Equal(1, 2))
	core.AssertFalse(t, Equal([]string{"a"}, []string{"a", "b"}))
	core.AssertFalse(t, Equal(behaviourStruct{Age: 1}, behaviourStruct{Age: 2}))
}

func TestTestassert_Nil_Behaviour_Good(t *core.T) {
	core.AssertTrue(t, Nil(nil))
	var ptr *behaviourStruct
	core.AssertTrue(t, Nil(ptr))
	var m map[string]int
	core.AssertTrue(t, Nil(m))
	var sl []string
	core.AssertTrue(t, Nil(sl))
	var fn func()
	core.AssertTrue(t, Nil(fn))
	var ch chan int
	core.AssertTrue(t, Nil(ch))
}

func TestTestassert_Nil_Behaviour_Bad(t *core.T) {
	core.AssertFalse(t, Nil("agent"))
	core.AssertFalse(t, Nil(0))
	core.AssertFalse(t, Nil(behaviourStruct{}))
	now := &behaviourStruct{}
	core.AssertFalse(t, Nil(now))
	core.AssertFalse(t, Nil([]string{"a"}))
}

func TestTestassert_Empty_Behaviour_Good(t *core.T) {
	core.AssertTrue(t, Empty(nil))
	core.AssertTrue(t, Empty(""))
	core.AssertTrue(t, Empty([]string{}))
	core.AssertTrue(t, Empty(map[string]int{}))
	core.AssertTrue(t, Empty([0]int{}))
	core.AssertTrue(t, Empty(0))
	core.AssertTrue(t, Empty(behaviourStruct{}))
	var ch chan int
	core.AssertTrue(t, Empty(ch))
}

func TestTestassert_Empty_Behaviour_Bad(t *core.T) {
	core.AssertFalse(t, Empty("agent"))
	core.AssertFalse(t, Empty([]string{"a"}))
	core.AssertFalse(t, Empty(map[string]int{"a": 1}))
	core.AssertFalse(t, Empty([1]int{7}))
	core.AssertFalse(t, Empty(5))
	core.AssertFalse(t, Empty(behaviourStruct{Name: "x"}))
}

func TestTestassert_Zero_Behaviour_Good(t *core.T) {
	core.AssertTrue(t, Zero(nil))
	core.AssertTrue(t, Zero(0))
	core.AssertTrue(t, Zero(""))
	core.AssertTrue(t, Zero(behaviourStruct{}))
}

func TestTestassert_Zero_Behaviour_Bad(t *core.T) {
	core.AssertFalse(t, Zero(1))
	core.AssertFalse(t, Zero("agent"))
	core.AssertFalse(t, Zero(behaviourStruct{Age: 1}))
}

func TestTestassert_Contains_Behaviour_Good(t *core.T) {
	core.AssertTrue(t, Contains("workflow_call:", "workflow_call:"))
	core.AssertTrue(t, Contains("the agent runs", "agent"))
	core.AssertTrue(t, Contains([]string{"linux", "darwin"}, "darwin"))
	core.AssertTrue(t, Contains(map[string]int{"a": 1, "b": 2}, "a"))
}

func TestTestassert_Contains_Behaviour_Bad(t *core.T) {
	core.AssertFalse(t, Contains("the agent runs", "node"))
	core.AssertFalse(t, Contains("string", 42))
	core.AssertFalse(t, Contains([]string{"linux"}, "darwin"))
	core.AssertFalse(t, Contains(map[string]int{"a": 1}, "z"))
}

func TestTestassert_Contains_Behaviour_Ugly(t *core.T) {
	// Convertible-but-not-assignable map key: a named string type indexes a
	// map keyed by plain string via the ConvertibleTo branch.
	type label string
	m := map[string]int{"release": 1}
	core.AssertTrue(t, Contains(m, label("release")))
	// Non-string elem against a string container falls through to false.
	core.AssertFalse(t, Contains("release", 1))
	// Invalid container (untyped nil) is never a match.
	core.AssertFalse(t, Contains(nil, "x"))
	// Map key with an incompatible type (struct key) returns false.
	core.AssertFalse(t, Contains(map[string]int{"a": 1}, behaviourStruct{}))
}

func TestTestassert_ElementsMatch_Behaviour_Good(t *core.T) {
	core.AssertTrue(t, ElementsMatch([]string{"linux", "darwin"}, []string{"darwin", "linux"}))
	core.AssertTrue(t, ElementsMatch([]int{1, 2, 2}, []int{2, 1, 2}))
	core.AssertTrue(t, ElementsMatch([]string{}, []string{}))
}

func TestTestassert_ElementsMatch_Behaviour_Bad(t *core.T) {
	core.AssertFalse(t, ElementsMatch([]string{"linux"}, []string{"linux", "darwin"}))
	core.AssertFalse(t, ElementsMatch([]int{1, 2, 2}, []int{1, 1, 2}))
	core.AssertFalse(t, ElementsMatch([]string{"a"}, []string{"b"}))
}

func TestTestassert_ElementsMatch_Behaviour_Ugly(t *core.T) {
	// Non-list arguments fall back to deep equality.
	core.AssertTrue(t, ElementsMatch("agent", "agent"))
	core.AssertFalse(t, ElementsMatch("agent", "node"))
	// One valid, one invalid value (untyped nil) cannot match.
	core.AssertFalse(t, ElementsMatch([]string{"a"}, nil))
	// Two untyped nils are both invalid and therefore equal.
	core.AssertTrue(t, ElementsMatch(nil, nil))
	// Mixed list and scalar falls through to deep-equal false.
	core.AssertFalse(t, ElementsMatch([]string{"a"}, "a"))
}
