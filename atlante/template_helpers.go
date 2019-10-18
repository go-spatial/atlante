package atlante

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-spatial/geom/planar/coord"
	"github.com/go-spatial/maptoolkit/atlante/template/trellis"
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
	"DrawBars":    TplDrawBars,
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

type ShowParts int8

const (
	ShowPartPrefix ShowParts = 1 << iota
	ShowPartLabel
	ShowPartSuffix
	ShowPartUnit
	ShowPartHemi

	ShowPartMain = ShowPartPrefix | ShowPartLabel | ShowPartSuffix
	ShowPartAll  = ShowPartPrefix | ShowPartLabel | ShowPartSuffix | ShowPartUnit | ShowPartHemi
)

type LabelPart struct {
	Coord int64
	Grid  trellis.Grid
	Unit  string
	Hemi  string
}

func (lp LabelPart) Parts() (int, int, int) {
	return lp.Grid.PartsFor(lp.Coord)
}

func (lp LabelPart) IsLabelMod10() bool {
	_, lbl, _ := lp.Parts()
	return lbl%10 == 0
}

func (lp LabelPart) DrawAt(w io.Writer, x, y float64, show ShowParts) {

	// Check to see if anything is even visible
	if show&ShowPartAll == 0 {
		return
	}

	prefix, label, suffix := lp.Parts()
	var output bytes.Buffer
	fmt.Fprintf(&output, `<text x="%v" y="%v" font-size="12px" text-anchor="middle">`, x, y)
	fmt.Fprintln(&output, "")
	if show&ShowPartPrefix == ShowPartPrefix {
		fmt.Fprintf(&output, `<tspan font-size="6px" fill="black">%d</tspan>`, prefix)
	}
	if show&ShowPartLabel == ShowPartLabel {
		fmt.Fprintf(&output, `<tspan dy="%v" font-size="12px" fill="black">%0.2d</tspan>`, 6, label)
	}
	if show&ShowPartSuffix == ShowPartSuffix {
		fmt.Fprintf(&output, `<tspan dy="%v" font-size="6px" fill="black">%0.*d</tspan>`, -6, lp.Grid.Width(), suffix)
	}
	if show&ShowPartUnit == ShowPartUnit {
		fmt.Fprintf(&output, `<tspan font-size="6px" fill="black">%v</tspan>`, lp.Unit)
	}
	if show&ShowPartHemi == ShowPartHemi {
		fmt.Fprintf(&output, `<tspan dy="%v" font-size="12px" fill="black">%v</tspan>`, 6, lp.Hemi)
	}
	fmt.Fprintln(&output, "</text>")
	w.Write(output.Bytes())

}

func TplDrawBars(lng1, lat1, lng2, lat2 float64, startingX, startingY float64, groundPixel float64, grid trellis.Grid) (string, error) {

	lnglat1 := coord.LngLat{
		Lng: lng1,
		Lat: lat1,
	}
	lnglat2 := coord.LngLat{
		Lng: lng2,
		Lat: lat2,
	}

	log.Printf("lnglat1: %#v lnglat2: %#v", lnglat1, lnglat2)

	strt, err := trellis.NewLngLat2(lnglat1, lnglat2, trellis.WGS84Ellip, grid)
	if err != nil {
		return "", err
	}
	var output strings.Builder

	const lineFormat = `<line x1="%v" y1="%v" x2="%v" y2="%v" stroke="black" stroke-width="1"  />`

	output.WriteString(`<g id="bars">`)
	// Draw the horizontal lines
	output.WriteString(`<g id="horizontal_bars">`)
	err = strt.NorthingBars(func(i int, bar trellis.Bar) error {

		hemi := "N."
		if bar.Start.Northing < 0 {
			hemi = "S."
		}

		part := LabelPart{
			Grid:  grid,
			Coord: int64(bar.Start.Northing),
			Hemi:  hemi,
			Unit:  "m",
		}

		x1 := startingX
		x2 := startingX + (bar.Length / groundPixel)
		y1 := startingY - (bar.Y1 / groundPixel)
		y2 := startingY - (bar.Y2 / groundPixel)

		show := ShowPartLabel
		if part.IsLabelMod10() {
			show |= ShowPartPrefix
		}

		if i == 0 {
			show = ShowPartAll
		}

		part.DrawAt(&output, x1-40, y1, show)

		fmt.Fprintf(&output, lineFormat, x1, y1, x2, y2)

		show = ShowPartLabel
		if part.IsLabelMod10() {
			show |= ShowPartPrefix
		}

		part.DrawAt(&output, x2+40, y2, show)

		return nil

	})
	output.WriteString("</g>\n")

	// Draw the vertical lines
	output.WriteString(`<g id="vertical_bars">
	`)
	//	rowsToShowEastLabels := []uint{8, 19}
	err = strt.EastingBars(func(i int, bar trellis.Bar) error {

		x1 := startingX + (bar.X1 / groundPixel)
		x2 := startingX + (bar.X2 / groundPixel)
		y1 := startingY
		y2 := startingY - (bar.Length / groundPixel)
		/*
			fmt.Fprintf(&output, `<text x="%v" y="%v" font-size="12px" text-anchor="middle">S(%[1]v,%v)</text>`, startingX, startingY)
			fmt.Fprintf(&output, `<text x="%v" y="%v" font-size="12px" text-anchor="middle">E1(%[1]v,%v)</text>`, x1, y1)
			fmt.Fprintf(&output, `<text x="%v" y="%v" font-size="12px" text-anchor="middle">E2(%[1]v,%v)</text>`, x2, y2)
		*/

		fmt.Fprintf(&output, lineFormat, x1, y1, x2, y2)

		hemi := "E."
		if bar.Start.Easting < 0 {
			hemi = "W."
		}

		part := LabelPart{
			Grid:  grid,
			Coord: int64(bar.Start.Easting),
			Hemi:  hemi,
			Unit:  "m",
		}

		show := ShowPartLabel
		if part.IsLabelMod10() {
			show |= ShowPartPrefix
		}

		// Top
		part.DrawAt(&output, x1, y2-25, show)

		// Bottom
		if i == 0 {
			show = ShowPartAll
		}
		part.DrawAt(&output, x2, y1+25, show)

		/*
			// Draw the labels for the select rows
			for _, row := range rowsToShowEastLabels {
				y := startingY + ((float64(grid.Size()/2-10) + float64(uint(grid.Size())*(row-1)) + float64(bar.YOffsetStart)) / groundPixel)
				part.DrawAt(&output, x1, y, ShowPartLabel)

			}
		*/

		return nil

	})
	output.WriteString("</g>\n")
	output.WriteString("</g>\n")

	return output.String(), nil
}
