// Package testassert provides small assertion predicates shared by tests.
package testassert

import (
	"reflect"

	core "dappco.re/go"
)

// Equal reports whether want and got are deeply equal.
//
// if !testassert.Equal("go", builder.Name()) { t.Fatal("unexpected builder name") }
func Equal(want, got any) bool {
	return reflect.DeepEqual(want, got)
}

// Nil reports whether value is nil, including typed nil interfaces.
//
// if !testassert.Nil(err) { t.Fatalf("unexpected error: %v", err) }
func Nil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// Empty reports whether value is nil, the zero value, or a zero-length container.
//
// if testassert.Empty(artifacts) { t.Fatal("expected artifacts") }
func Empty(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

// Zero reports whether value is nil or the zero value for its type.
//
// if !testassert.Zero(build.Config{}) { t.Fatal("expected zero config") }
func Zero(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	return !v.IsValid() || v.IsZero()
}

// Contains reports whether container contains elem for strings, maps, arrays, or slices.
//
// if !testassert.Contains(content, "workflow_call:") { t.Fatal("missing workflow call") }
func Contains(container, elem any) bool {
	if s, ok := container.(string); ok {
		sub, ok := elem.(string)
		return ok && core.Contains(s, sub)
	}

	v := reflect.ValueOf(container)
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(elem)
		if !key.IsValid() {
			return false
		}
		if key.Type().AssignableTo(v.Type().Key()) {
			return v.MapIndex(key).IsValid()
		}
		if key.Type().ConvertibleTo(v.Type().Key()) {
			return v.MapIndex(key.Convert(v.Type().Key())).IsValid()
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if reflect.DeepEqual(v.Index(i).Interface(), elem) {
				return true
			}
		}
	}
	return false
}

// ElementsMatch reports whether want and got contain the same elements, ignoring order.
//
// if !testassert.ElementsMatch([]string{"linux", "darwin"}, targets) { t.Fatal("target mismatch") }
func ElementsMatch(want, got any) bool {
	wantValue := reflect.ValueOf(want)
	gotValue := reflect.ValueOf(got)
	if !wantValue.IsValid() || !gotValue.IsValid() {
		return !wantValue.IsValid() && !gotValue.IsValid()
	}
	if !isListValue(wantValue) || !isListValue(gotValue) {
		return reflect.DeepEqual(want, got)
	}
	if wantValue.Len() != gotValue.Len() {
		return false
	}

	used := make([]bool, gotValue.Len())
	for i := 0; i < wantValue.Len(); i++ {
		found := false
		wantElem := wantValue.Index(i).Interface()
		for j := 0; j < gotValue.Len(); j++ {
			if used[j] {
				continue
			}
			if reflect.DeepEqual(wantElem, gotValue.Index(j).Interface()) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func isListValue(value reflect.Value) bool {
	return value.Kind() == reflect.Array || value.Kind() == reflect.Slice
}
