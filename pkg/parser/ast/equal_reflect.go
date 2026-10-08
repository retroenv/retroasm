package ast

import (
	"reflect"

	"github.com/retroenv/retrogolib/set"
)

// equalVisit records a pair of references before their contents are compared.
// This permits cyclic operands and extension values.
type equalVisit struct {
	left, right uintptr
	typ         reflect.Type
}

func equalComposite(left, right Node) bool {
	return equalValue(reflect.ValueOf(left), reflect.ValueOf(right), set.New[equalVisit]())
}

func equalValue(left, right reflect.Value, seen set.Set[equalVisit]) bool {
	if !left.IsValid() || !right.IsValid() {
		return left.IsValid() == right.IsValid()
	}
	if left.Type() != right.Type() {
		return false
	}
	if left.Type() == reflect.TypeFor[*entryHandle]() {
		return true
	}
	switch left.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		if left.IsNil() || right.IsNil() {
			return left.IsNil() == right.IsNil()
		}
		if left.Kind() != reflect.Pointer && left.Len() != right.Len() {
			return false
		}
		visit := equalVisit{
			left:  uintptr(left.UnsafePointer()),
			right: uintptr(right.UnsafePointer()),
			typ:   left.Type(),
		}
		if visit.left == visit.right || seen.Contains(visit) {
			return true
		}
		seen.Add(visit)
	}
	return equalValueContents(left, right, seen)
}

func equalValueContents(left, right reflect.Value, seen set.Set[equalVisit]) bool {
	switch left.Kind() {
	case reflect.Pointer:
		return equalValue(left.Elem(), right.Elem(), seen)

	case reflect.Interface:
		if left.IsNil() || right.IsNil() {
			return left.IsNil() == right.IsNil()
		}
		return equalValue(left.Elem(), right.Elem(), seen)

	case reflect.Struct:
		for index := range left.NumField() {
			if !equalValue(left.Field(index), right.Field(index), seen) {
				return false
			}
		}
		return true

	case reflect.Array, reflect.Slice:
		for index := range left.Len() {
			if !equalValue(left.Index(index), right.Index(index), seen) {
				return false
			}
		}
		return true

	case reflect.Map:
		iterator := left.MapRange()
		for iterator.Next() {
			if !equalValue(iterator.Value(), right.MapIndex(iterator.Key()), seen) {
				return false
			}
		}
		return true

	default:
		return equalScalar(left, right)
	}
}

func equalScalar(left, right reflect.Value) bool {
	switch left.Kind() {
	case reflect.Bool:
		return left.Bool() == right.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return left.Int() == right.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return left.Uint() == right.Uint()
	case reflect.Float32, reflect.Float64:
		return left.Float() == right.Float()
	case reflect.Complex64, reflect.Complex128:
		return left.Complex() == right.Complex()
	case reflect.String:
		return left.String() == right.String()
	case reflect.Func:
		return left.IsNil() && right.IsNil()
	case reflect.Chan, reflect.UnsafePointer:
		return left.Pointer() == right.Pointer()
	default:
		panic("unsupported reflection kind")
	}
}
