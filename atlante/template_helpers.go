package atlante

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

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
