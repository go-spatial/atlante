package atlante

import (
	"fmt"

	"github.com/gdey/as"
	"github.com/go-spatial/geom/planar/coord"
)

type AsTypeError struct {
	Type  string
	Value interface{}
}

func (err AsTypeError) Error() string {
	return fmt.Sprintf("unknown %v value at '%v'", err.Type, err.Value)
}

func ATplAs() TplAs { return TplAs{} }

type TplAs struct{}

func (TplAs) Int64(v interface{}) (int64, error) {
	if i64, ok := as.Int64(v); ok {
		return i64, nil
	}
	return 0, AsTypeError{"int64", v}
}
func (TplAs) Float64(v interface{}) (float64, error) {
	if i64, ok := as.Float64(v); ok {
		return i64, nil
	}
	return 0, AsTypeError{"float64", v}
}

func (TplAs) LngLat(lngArg, latArg interface{}) (lnglat coord.LngLat, err error) {
	var (
		ok bool
	)
	lnglat.Lng, ok = as.Float64(lngArg)
	if !ok {
		return lnglat, fmt.Errorf("unknown lng value '%v' ", lngArg)
	}
	lnglat.Lat, ok = as.Float64(latArg)
	if !ok {
		return lnglat, fmt.Errorf("unknown lat value '%v' ", latArg)
	}
	return lnglat, nil
}
