package release

import (
	"reflect"
	"strings"
)

func stdlibAssertEqual(want, got any) bool {
	return reflect.DeepEqual(want, got)
}

func stdlibAssertNil(value any) bool {
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

func stdlibAssertEmpty(value any) bool {
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

func stdlibAssertZero(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	return !v.IsValid() || v.IsZero()
}

func stdlibAssertContains(container, elem any) bool {
	if s, ok := container.(string); ok {
		sub, ok := elem.(string)
		return ok && strings.Contains(s, sub)
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

func stdlibAssertElementsMatch(want, got any) bool {
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
