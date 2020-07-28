package as

import (
	"fmt"
	"reflect"
	"strconv"
)

// InvalidTypeErr describes a type error
type InvalidTypeErr struct {
	Expected string
	Have     reflect.Type
}

// Error implements the error interface
func (err InvalidTypeErr) Error() string {
	return fmt.Sprintf("expected %s, but got %v", err.Expected, err.Have)
}

// InterfaceSlice ties to coerce slice into a slice of interfaces.
func InterfaceSlice(slice interface{}) ([]interface{}, error) {
	if slice == nil {
		return []interface{}{}, nil
	}
	if s, ok := slice.([]interface{}); ok {
		return s, nil
	}
	s := reflect.ValueOf(slice)
	switch kind := s.Kind(); kind {
	case reflect.Array, reflect.Slice:
		l := s.Len()
		ret := make([]interface{}, l)
		for i := 0; i < l; i++ {
			ret[i] = s.Index(i).Interface()
		}
		return ret, nil
	default:
		return nil, InvalidTypeErr{Expected: "array or slice", Have: reflect.TypeOf(slice)}
	}
}

// Bool converts an arbitrary input into a boolean. If it could not coerce the
// value properly ok will be false
func Bool(it interface{}) (v bool, ok bool) {

	if it == nil {
		return false, false
	}

	if b, ok := it.(bool); ok {
		return b, true
	}

	if b, ok := it.(*bool); ok {
		if b == nil {
			return false, true
		}
		return *b, true
	}

	if str, ok := it.(string); ok {
		b, err := strconv.ParseBool(str)
		return b, err == nil
	}

	val := reflect.Indirect(reflect.ValueOf(it))
	switch val.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return val.Int() == 1, true
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return val.Uint() == 1, true
	case reflect.Float32, reflect.Float64:
		return val.Float() == 1.0, true

	default:
		return false, false
	}

}

// Int64 tries to coerce it into a int64, returning the coerced int64
// and weather it was able to do it.
func Int64(it interface{}) (int64, bool) {
	if it == nil {
		return 0, false
	}
	if str, ok := it.(string); ok {
		i, err := strconv.ParseInt(str, 10, 64)
		return i, err == nil
	}

	val := reflect.Indirect(reflect.ValueOf(it))
	switch val.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return val.Int(), true
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return int64(val.Uint()), true
	case reflect.Uint, reflect.Uint64:
		tv := val.Uint()
		// this can overflow and give -1, but IMO this is better than
		// returning maxint64
		return int64(tv), true
	case reflect.Float32, reflect.Float64:
		return int64(val.Float()), true
	case reflect.Bool:
		if val.Bool() {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// Int tries to coerce it into a int, returning the coerced int
// and weather it was able to do it.
func Int(it interface{}) (int, bool) {
	i, ok := Int64(it)
	return int(i), ok
}

// Uint64 tries to coerce it into a int64, returning the coerced int64
// and weather it was able to do it.
func Uint64(it interface{}) (uint64, bool) {
	if it == nil {
		return 0, false
	}
	if str, ok := it.(string); ok {
		i, err := strconv.ParseUint(str, 10, 64)
		return i, err == nil
	}

	val := reflect.Indirect(reflect.ValueOf(it))
	switch val.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return uint64(val.Int()), true
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return val.Uint(), true
	case reflect.Float32, reflect.Float64:
		return uint64(val.Float()), true
	case reflect.Bool:
		if val.Bool() {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// Uint tries to coerce it into a uint, returning the coerced uint
// and weather it was able to do it.
func Uint(it interface{}) (uint, bool) {
	i, ok := Uint64(it)
	return uint(i), ok
}

// Float64 tries to coerce it into a float64, returning the coerced float64
// and weather it was able to do it.
func Float64(it interface{}) (float64, bool) {
	if str, ok := it.(string); ok {
		i, err := strconv.ParseFloat(str, 64)
		return i, err == nil
	}
	val := reflect.Indirect(reflect.ValueOf(it))
	switch val.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return float64(val.Int()), true
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return float64(val.Uint()), true
	case reflect.Uint, reflect.Uint64:
		return float64(val.Uint()), true
	case reflect.Float32, reflect.Float64:
		return val.Float(), true
	case reflect.Bool:
		if val.Bool() {
			return 1.0, true
		}
		return 0.0, true
	default:
		return 0.0, false
	}
}

// String tries to coerce it into a string returning the coerced string
// and weather it was able to do it.
func String(it interface{}) (string, bool) {
	var (
		ok bool
	)
	if it == nil {
		return "", false
	}
	if s, ok := it.(string); ok {
		return s, true
	}
	if s, ok := it.(fmt.Stringer); ok {
		return s.String(), true
	}

	if it, ok = printableValue(reflect.ValueOf(it)); !ok {
		return "", false
	}
	return fmt.Sprint(it), true
}
