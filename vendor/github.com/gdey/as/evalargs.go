package as

// stolen from the Go standard library, text/template package.
import (
	"fmt"
	"reflect"
)

var (
	errorType       = reflect.TypeOf((*error)(nil)).Elem()
	fmtStringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
)

// printableValue returns the, possibly indirect, interface value inside v that
// is best for a call to formatted printer.
func printableValue(v reflect.Value) (interface{}, bool) {
	var isNil bool
	if v.Kind() == reflect.Ptr {
		v, isNil = indirect(v)
		if isNil {
			return "", false
		}
	}
	if !v.IsValid() {
		return "", false
	}

	if !v.Type().Implements(errorType) && !v.Type().Implements(fmtStringerType) {
		if v.CanAddr() && (reflect.PtrTo(v.Type()).Implements(errorType) || reflect.PtrTo(v.Type()).Implements(fmtStringerType)) {
			v = v.Addr()
		} else {
			switch v.Kind() {
			case reflect.Chan, reflect.Func:
				return "", false
			}
		}
	}
	return v.Interface(), true
}

// indirect returns the item at the end of indirection, and a bool to indicate if it's nil.
func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
	}
	return v, false
}
