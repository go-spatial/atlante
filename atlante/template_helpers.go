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

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/planar/coord"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"
	"github.com/go-spatial/maptoolkit/atlante/template/remote"
	"github.com/go-spatial/maptoolkit/atlante/template/trellis"
)

var funcMap = template.FuncMap{
	"to_upper":     strings.ToUpper,
	"to_lower":     strings.ToLower,
	"format":       tplFormat,
	"now":          time.Now,
	"div":          tplMathDiv,
	"add":          tplMathAdd,
	"sub":          tplMathSub,
	"mul":          tplMathMul,
	"neg":          tplMathNeg,
	"abs":          tplMathAbs,
	"seq":          tplSeq,
	"new_toggler":  tplNewToggle,
	"rounder_for":  tplRoundTo,
	"rounder3":     tplRound3,
	"first":        tplFirstNonZero,
	"DrawBars":     TplDrawBars,
	"asIntSlice":   IntSlice,
	"pixel_bounds": PixelBounds,
	"join":         tplStringJoin,
	"idx":          tplIndexOf,
}

func tplStringJoin(sep string, ps ...interface{}) string {
	var parts = make([]string, len(ps))
	for i, pstr := range ps {
		parts[i] = fmt.Sprintf("%v", pstr)
	}

	return strings.Join(parts, sep)
}

func tplIndexOf(idx int, parts interface{}) (interface{}, error) {

	v := reflect.ValueOf(parts)

	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array && v.Kind() != reflect.String {
		return nil, fmt.Errorf("not slice/array/string type: %v", v.Kind())
	}

	vlen := v.Len()
	if vlen == 0 {
		return nil, nil
	}

	var i int
	if idx < 0 {
		i = vlen + idx
	} else {
		i = idx
	}
	fmt.Println("returning before idx", i)
	i = i % vlen
	fmt.Println("returning idx", i)

	return v.Index(i).Interface(), nil
}

// AddTemplateFunc will add the filestore based commands. It will panic if the command is already defined.
func (sheet *Sheet) AddTemplateFuncs(funcMap template.FuncMap) template.FuncMap {
	funcMap["remote"] = sheet.templateFuncRemote
	return funcMap
}

func (sheet *Sheet) templateFuncRemote(loc string) (string, error) {
	sheet.Emit(field.Processing{Description: fmt.Sprintf("remote file: %v", loc)})
	return remote.Remote(loc, sheet.FuncFilestoreWriter, sheet.UseCached)
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

func tplRound(x float64, unit int) float64 {
	factor := math.Pow10(unit)

	return math.Round(x*factor) / factor
}

func tplRound3(x float64) float64 { return tplRound(x, 3) }
func tplRoundTo(unit int) func(float64) float64 {
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
	fmt.Fprintf(&output, `<text x="%v" y="%v" font-size="12px" text-anchor="middle" >`, x, y)
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

func IntSlice(vals ...interface{}) ([]int, error) {
	ints := make([]int, 0, len(vals))
	for i := range vals {
		switch v := vals[i].(type) {
		case int8:
			ints = append(ints, int(v))
		case int16:
			ints = append(ints, int(v))
		case int32:
			ints = append(ints, int(v))
		case int:
			ints = append(ints, v)
		case int64:
			ints = append(ints, int(v))
		case uint8:
			ints = append(ints, int(v))
		case uint16:
			ints = append(ints, int(v))
		case uint32:
			ints = append(ints, int(v))
		case uint:
			ints = append(ints, int(v))
		case uint64:
			ints = append(ints, int(v))
		case float64:
			ints = append(ints, int(v))
		case float32:
			ints = append(ints, int(v))
		case string:
			a, err := strconv.Atoi(v)
			if err != nil {
				return ints, err
			}
			ints = append(ints, a)
		default:
			return ints, fmt.Errorf("unknown int value at %v: '%v' ", i, v)
		}
	}
	return ints, nil
}

func LngLatCoord(lng, lat float64) coord.LngLat {
	return coord.LngLat{
		Lng: lng,
		Lat: lat,
	}
}

type PixelBox struct {
	Starting     [2]float64
	Ending       [2]float64
	GroundPixel  float64
	LeftBuffer   float64
	BottomBuffer float64
	RightBuffer  float64
	TopBuffer    float64
	RowOffset    float64
	ColOffset    float64
}

func PixelBounds(x1, y1, x2, y2 float64, groundPixel float64, extraVals ...float64) PixelBox {
	pbx := PixelBox{
		Starting:     [2]float64{x1, y1},
		Ending:       [2]float64{x2, y2},
		GroundPixel:  groundPixel,
		LeftBuffer:   10.0,
		BottomBuffer: 10.0,
		RightBuffer:  10.0,
		TopBuffer:    10.0,
		RowOffset:    0.0,
		ColOffset:    0.0,
	}
	ln := len(extraVals)
	if ln >= 1 {
		pbx.LeftBuffer = extraVals[0]
		pbx.BottomBuffer = extraVals[0]
		pbx.RightBuffer = extraVals[0]
		pbx.TopBuffer = extraVals[0]
	}
	if ln >= 2 {
		pbx.BottomBuffer = extraVals[1]
		pbx.TopBuffer = extraVals[1]
	}
	if ln >= 3 {
		pbx.RightBuffer = extraVals[2]
	}
	if ln >= 4 {
		pbx.TopBuffer = extraVals[3]
	}
	if ln >= 5 {
		pbx.ColOffset = extraVals[4]
		pbx.RowOffset = extraVals[4]
	}
	if ln >= 6 {
		pbx.RowOffset = extraVals[5]
	}
	return pbx
}

func (pbx PixelBox) TransformLine(l geom.Line) geom.Line {
	return geom.Line{
		{
			pbx.Starting[0] + (l[0][0] / pbx.GroundPixel),
			pbx.Starting[1] + (l[0][1] / pbx.GroundPixel),
		}, {
			pbx.Starting[0] + (l[1][0] / pbx.GroundPixel),
			pbx.Starting[1] + (l[1][1] / pbx.GroundPixel),
		},
	}

}
func (pbx PixelBox) TransformPoint(pt geom.Point) geom.Point {
	return geom.Point{
		pbx.Starting[0] + (pt[0] / pbx.GroundPixel),
		pbx.Starting[1] + (pt[1] / pbx.GroundPixel),
	}
}

func (pbx PixelBox) TransformPoints(pts ...geom.Point) []geom.Point {
	rpts := make([]geom.Point, len(pts))
	for i := range pts {
		rpts[i][0] = pbx.Starting[0] + (pts[i][0] / pbx.GroundPixel)
		rpts[i][1] = pbx.Starting[1] + (pts[i][1] / pbx.GroundPixel)
	}
	return rpts
}

func harvesinDistance(pt1, pt2 coord.LngLat, earth coord.Ellipsoid) float64 {

	// Got from : https://www.movable-type.co.uk/scripts/latlong.html
	R := earth.Radius
	phi1 := pt1.LatInRadians()
	phi2 := pt2.LatInRadians()

	deltaCoord := coord.LngLat{
		Lat: pt2.Lat - pt1.Lat,
		Lng: pt2.Lng - pt1.Lng,
	}

	deltaPhi := deltaCoord.LatInRadians()
	deltaLambda := deltaCoord.LngInRadians()

	a := (math.Sin(deltaPhi/2) * math.Sin(deltaPhi/2)) +
		(math.Cos(phi1)*math.Cos(phi2))*
			(math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2))

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c
	return d

}

type lblEntry struct {
	lbl  LabelPart
	show ShowParts
	x, y float64
}

func (lbl lblEntry) DrawTo(w io.Writer) {
	lbl.lbl.DrawAt(w, lbl.x, lbl.y, lbl.show)
}

func TplDrawBars(bottomLeft, topRight coord.LngLat, pxlBox PixelBox, grid trellis.Grid, lblRows, lblCols []int, lblMeterOffset int, drawLines bool) (string, error) {

	const debug = true

	topLeft := coord.LngLat{
		Lng: bottomLeft.Lng,
		Lat: topRight.Lat,
	}
	bottomRight := coord.LngLat{
		Lng: topRight.Lng,
		Lat: bottomLeft.Lat,
	}
	structure, err := trellis.NewLngLat(bottomLeft, topRight, trellis.WGS84Ellip, grid)
	if err != nil {
		return "", err
	}

	var (
		// Lines to be drawn
		lines       []geom.Line
		internalLbL []lblEntry
		externalLbl []lblEntry

		output               strings.Builder
		size                 = int(grid.Size())
		numberOfStepsEasting = int(
			math.Abs(
				math.Ceil(
					harvesinDistance(bottomLeft, bottomRight, trellis.WGS84Ellip) / float64(size),
				),
			),
		)
		numberOfStepsNorthing = int(
			math.Abs(
				math.Ceil(
					harvesinDistance(bottomLeft, topLeft, trellis.WGS84Ellip) / float64(size),
				),
			),
		)

		xValueInMeters = math.Abs(
			math.Ceil(
				harvesinDistance(bottomLeft, bottomRight, trellis.WGS84Ellip) + float64(size),
			),
		)
		yValueInMeters = math.Abs(
			math.Ceil(
				harvesinDistance(bottomLeft, topLeft, trellis.WGS84Ellip)+float64(size),
			),
		) * -1
	)

	output.WriteString(`<g id="bars">`)

	// draw the the northing lines (horizontal lines) -- the labels for these will be on the lblCol
	{

		// Calculate the number of steps
		brEblE := structure.BottomRightUTM.Easting - structure.BottomLeftUTM.Easting
		blEbrE := structure.BottomLeftUTM.Easting - structure.BottomRightUTM.Easting
		log.Printf("BottomRight %g BottomLeft %g easting: %g : %g", structure.BottomRightUTM.Easting, structure.BottomLeftUTM.Easting, brEblE, blEbrE)
		log.Printf("before Number of Easting steps (cols): %v", numberOfStepsEasting)
		log.Printf("Number of Northing steps (rows): %v", numberOfStepsNorthing)

		part := LabelPart{
			Grid: grid,
			Unit: "m",
		}
		log.Printf("Number of Easting steps (cols): %v", numberOfStepsEasting)
		for col := -1; col < numberOfStepsEasting+2; col++ {
			colMeter := float64(structure.LeftOffset) + float64(col*size)
			tx, ty := structure.BottomVector.Travel(colMeter)
			pVector := structure.BottomVector.PerpendicularVector(tx, ty)
			pt2x := pVector.XFor(yValueInMeters)

			if drawLines {
				pt1y := ty + float64(size)
				pt1x := pVector.XFor(pt1y)
				ln := pxlBox.TransformLine(
					geom.Line{
						[2]float64{pt1x, pt1y},
						[2]float64{pt2x, yValueInMeters},
					},
				)
				lines = append(lines, ln)
			}

			part.Coord = int64(structure.BottomLeftUTM.Easting) + int64(structure.LeftOffset) + (int64(col) * grid.Size())
			part.Hemi = "E."
			if part.Coord < 0 {
				part.Hemi = "W."
			}

			/*
				if len(lblCols) <= 0 {
					continue
				}
			*/
			pt := pxlBox.TransformPoint(geom.Point{tx, ty})
			endPt := pxlBox.TransformPoint(geom.Point{pt2x, yValueInMeters})
			if col >= 0 && col < numberOfStepsEasting {
				// outter label
				show := ShowPartLabel
				if part.IsLabelMod10() {
					show |= ShowPartPrefix
				}
				if col == 0 {
					show = ShowPartAll
				}
				externalLbl = append(externalLbl, lblEntry{
					lbl:  part,
					show: show,
					x:    pt[0],
					y:    pxlBox.Starting[1] + pxlBox.BottomBuffer,
				})

				externalLbl = append(externalLbl, lblEntry{
					lbl:  part,
					show: show,
					x:    endPt[0],
					y:    pxlBox.Ending[1] - pxlBox.TopBuffer,
				})

				for _, row := range lblRows {
					startRowMeter := float64((row - 1) * size)
					endRowMeter := float64(row * size)

					ltx, lty := structure.LeftVector.Travel(startRowMeter)
					lty -= float64(structure.BottomOffset)
					bottomVector := structure.LeftVector.PerpendicularVector(ltx, lty)
					startX := pVector.XFor(lty)
					startY := bottomVector.YFor(startX)

					ltx, lty = structure.LeftVector.Travel(endRowMeter)
					lty -= float64(structure.BottomOffset)
					bottomVector = structure.LeftVector.PerpendicularVector(ltx, lty)
					endX := pVector.XFor(lty)
					endY := bottomVector.YFor(endX)

					pt = pxlBox.TransformPoint(geom.Point{
						startX + ((endX - startX) / 2),
						startY + ((endY - startY) / 2),
					})
					internalLbL = append(internalLbL, lblEntry{
						lbl:  part,
						show: ShowPartLabel,
						x:    pt[0],
						y:    pt[1],
					})
				}

			}
		}

		log.Printf("Number of Northing steps (rows): %v", numberOfStepsNorthing)
		for row := -1; row < numberOfStepsNorthing+2; row++ {
			rowMeter := float64(row * size)
			tx, ty := structure.LeftVector.Travel(rowMeter)
			ty -= float64(structure.BottomOffset)
			pVector := structure.LeftVector.PerpendicularVector(tx, ty)
			pt2y := pVector.YFor(xValueInMeters)

			if drawLines {
				pt1x := tx - float64(size)
				pt1y := pVector.YFor(pt1x)
				ln := pxlBox.TransformLine(
					geom.Line{
						[2]float64{pt1x, pt1y},
						[2]float64{xValueInMeters, pt2y},
					},
				)
				lines = append(lines, ln)
			}
			part.Coord = int64(structure.BottomLeftUTM.Northing) + int64(structure.BottomOffset) + (int64(row) * grid.Size())
			part.Hemi = "N."
			if part.Coord < 0 {
				part.Hemi = "S."
			}
			pt := pxlBox.TransformPoint(geom.Point{tx, ty})
			endPt := pxlBox.TransformPoint(geom.Point{xValueInMeters, pt2y})
			if row >= 0 && row < numberOfStepsNorthing {
				// outter label
				show := ShowPartLabel
				if part.IsLabelMod10() {
					show |= ShowPartPrefix
				}
				if row == 0 {
					show = ShowPartAll
				}
				externalLbl = append(externalLbl, lblEntry{
					lbl:  part,
					show: show,
					x:    pxlBox.Starting[0] - pxlBox.LeftBuffer,
					y:    pt[1],
				})
				externalLbl = append(externalLbl, lblEntry{
					lbl:  part,
					show: show,
					x:    pxlBox.Ending[0] + pxlBox.RightBuffer,
					y:    endPt[1],
				})
				for _, col := range lblCols {
					startColMeter := float64((col-1)*size) + float64(structure.LeftOffset)
					endColMeter := float64(col*size) + float64(structure.LeftOffset)

					btx, bty := structure.BottomVector.Travel(startColMeter)
					leftVector := structure.BottomVector.PerpendicularVector(btx, bty)

					startY := pVector.YFor(btx)
					startX := leftVector.XFor(startY)

					btx, bty = structure.BottomVector.Travel(endColMeter)
					leftVector = structure.BottomVector.PerpendicularVector(btx, bty)

					endY := pVector.YFor(btx)
					endX := leftVector.XFor(startY)

					pt = pxlBox.TransformPoint(geom.Point{
						startX + ((endX - startX) / 2),
						startY + ((endY - startY) / 2),
					})

					internalLbL = append(internalLbL, lblEntry{
						lbl:  part,
						show: ShowPartLabel,
						x:    pt[0],
						y:    pt[1],
					})

				}
			}

		}

	}

	const lineFormat = `<line x1="%g" y1="%g" x2="%g" y2="%g" stroke="black" stroke-width="1"  />`
	output.WriteString(`<g id="internal_lines" clip-path="url(#imageClip)">`)
	fmt.Fprintf(&output, "<!--  number of lines: %v -->", len(lines))
	for _, line := range lines {
		fmt.Fprintf(&output, lineFormat, line[0][0], line[0][1], line[1][0], line[1][1])
	}
	for _, part := range internalLbL {
		part.DrawTo(&output)
	}

	output.WriteString("</g>\n")
	output.WriteString(`<g id="external_labels" clip-path="url(#externalLabelClip)">`)
	for _, part := range externalLbl {
		part.DrawTo(&output)
	}
	output.WriteString("</g>\n")
	output.WriteString("</g>\n")
	return output.String(), nil
}
