package atlante

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var funcMap = template.FuncMap{
	"to_upper":    strings.ToUpper,
	"to_lower":    strings.ToLower,
	"format":      tplFormat,
	"now":         time.Now,
	"div":         tplMathDiv,
	"add":         tplMathAdd,
	"sub":         tplMathSub,
	"mul":         tplMathMul,
	"neg":         tplMathNeg,
	"abs":         tplMathAbs,
	"seq":         tplSeq,
	"new_toggler": tplNewToggle,
	"rounder_for": tplRoundTo,
	"rounder3":    tplRound3,
	"first":       tplFirstNonZero,
}

//tplFormat is a helper function for templates that will format the given
// value. It uses Sprintf for most value, except time.
func tplFormat(format string, data interface{}) string {
	// Allow Format to be used for time's as well.
	if d, ok := data.(time.Time); ok {
		return d.Format(format)
	}
	return fmt.Sprintf(format, data)
}

func toFloat64(a interface{}) (float64, bool) {
	switch aa := a.(type) {
	case int:
		return float64(aa), true
	case int8:
		return float64(aa), true
	case int16:
		return float64(aa), true
	case int32:
		return float64(aa), true
	case int64:
		return float64(aa), true
	case uint:
		return float64(aa), true
	case uint8:
		return float64(aa), true
	case uint16:
		return float64(aa), true
	case uint32:
		return float64(aa), true
	case uint64:
		return float64(aa), true
	case float32:
		return float64(aa), true
	case float64:
		return aa, true
	case complex64:
		return float64(real(aa)), true
	case complex128:
		return float64(real(aa)), true
	case string:
		b, err := strconv.ParseFloat(aa, 64)
		return b, err == nil
	default:
		return 0.0, false
	}
}

func tplMathDiv(av, bv interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := toFloat64(av)
	if !ok {
		return 0, fmt.Errorf("first value (%t) needs to be a number", av)
	}
	b, ok := toFloat64(bv)
	if !ok {
		return 0, fmt.Errorf("second value (%t) needs to be a number", bv)
	}
	if b == 0 {
		return 0, errors.New("divide by zero")
	}
	return a / b, nil
}

func tplMathMul(av, bv interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := toFloat64(av)
	if !ok {
		return 0, fmt.Errorf("first value (%v) needs to be a number", av)
	}
	b, ok := toFloat64(bv)
	if !ok {
		return 0, fmt.Errorf("second value (%v) needs to be a number", bv)
	}
	return a * b, nil
}

func tplMathSub(av, bv interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := toFloat64(av)
	if !ok {
		return 0, fmt.Errorf("first value (%v) needs to be a number", av)
	}
	b, ok := toFloat64(bv)
	if !ok {
		return 0, fmt.Errorf("second value (%v) needs to be a number", bv)
	}
	return a - b, nil
}

func tplMathAdd(av, bv interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := toFloat64(av)
	if !ok {
		return 0, fmt.Errorf("first value (%v) needs to be a number", av)
	}
	b, ok := toFloat64(bv)
	if !ok {
		return 0, fmt.Errorf("second value (%v) needs to be a number", bv)
	}
	return a + b, nil
}

func tplMathNeg(av interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := toFloat64(av)
	if !ok {
		return 0, fmt.Errorf("value (%v) needs to be a number", av)
	}
	return a * -1, nil
}

func tplMathAbs(av interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := toFloat64(av)
	if !ok {
		return 0, fmt.Errorf("value (%v) needs to be a number", av)
	}
	return math.Abs(a), nil
}

func tplSeq(start float64, num uint, inc float64) []float64 {
	if num == 0 {
		return []float64{}
	}
	is := make([]float64, 0, int(num))
	last := start
	for i := 0; i < int(num); i++ {
		is = append(is, last)
		last += inc
	}
	return is
}

type tplToggle struct {
	idx  int
	strs []string
}

func (t *tplToggle) Value() string {
	if t == nil || len(t.strs) == 0 {
		return ""
	}

	if t.idx >= len(t.strs) {
		t.idx = 0
	}
	s := t.strs[t.idx]
	t.idx++
	return s
}

func (t *tplToggle) Reset() {
	t.idx = 0
}

func (t *tplToggle) First() string {
	if t == nil || len(t.strs) == 0 {
		return ""
	}
	t.idx = 1
	return t.strs[0]
}

func tplNewToggle(strs ...string) *tplToggle {
	return &tplToggle{
		strs: strs,
	}
}

func tplRound(x float64, unit float64) float64 {
	return math.Round(x/unit) * unit
}

func tplRound3(x float64) float64 { return tplRound(x, 0.001) }
func tplRoundTo(unit float64) func(float64) float64 {
	return func(x float64) float64 { return tplRound(x, unit) }
}

// IsZero reports whether v is a zero value for its type.
// It panics if the argument is invalid.
// this is from go1.13 once we are on go1.13 we can remove this function
// and use reflect.IsZero
func isZero(v reflect.Value) bool {

	switch v.Kind() {

	case reflect.Bool:

		return !v.Bool()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return v.Int() == 0

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return v.Uint() == 0

	case reflect.Float32, reflect.Float64:

		return math.Float64bits(v.Float()) == 0

	case reflect.Complex64, reflect.Complex128:

		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0

	case reflect.Array:

		for i := 0; i < v.Len(); i++ {
			if !isZero(v.Index(i)) {
				return false
			}
		}
		return true

	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:

		return v.IsNil()

	case reflect.String:

		return v.Len() == 0

	case reflect.Struct:

		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true

	default:
		// This should never happens, but will act as a safeguard for
		// later, as a default value doesn't makes sense here.
		panic(&reflect.ValueError{"reflect.Value.IsZero", v.Kind()})

	}

}

func tplIsZero(v interface{}) bool { return isZero(reflect.ValueOf(v)) }
func tplFirstNonZero(vls ...interface{}) interface{} {
	for i := range vls {
		if !tplIsZero(vls[i]) {
			return vls[i]
		}
	}
	return nil
}
