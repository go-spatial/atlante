package atlante

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/gdey/as"
	"github.com/prometheus/common/log"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/planar/coord"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"
	"github.com/go-spatial/maptoolkit/atlante/template/grating"
	"github.com/go-spatial/maptoolkit/atlante/template/remote"
	"github.com/go-spatial/maptoolkit/atlante/template/trellis"
)

const (
	ArgsKeyX                   = "X"
	ArgsKeyY                   = "Y"
	ArgsKeyWidth               = "Width"
	ArgsKeyHeight              = "Height"
	ArgsKeyNumberOfRows        = "Number-Of-Rows"
	ArgsKeyNumberOfCols        = "Number-Of-Cols"
	ArgsKeyFlipY               = "Flip-Y"
	ArgsKeyGratingNumberOfRows = "Grating-Number-Of-Rows"
	ArgsKeyGratingNumberOfCols = "Grating-Number-Of-Columns"
	ArgsKeyImageWidth          = "Image-Width"
	ArgsKeyImageHeight         = "Image-Height"
)

var funcMap = template.FuncMap{
	"to_upper":           strings.ToUpper,
	"to_lower":           strings.ToLower,
	"format":             tplFormat,
	"now":                time.Now,
	"div":                tplMathDiv,
	"add":                tplMathAdd,
	"sub":                tplMathSub,
	"mul":                tplMathMul,
	"neg":                tplMathNeg,
	"abs":                tplMathAbs,
	"seq":                tplSeq,
	"new_toggler":        tplNewToggle,
	"rounder_for":        tplRoundTo,
	"rounder3":           tplRound3,
	"first":              tplFirstNonZero,
	"DrawBars":           TplDrawBars,
	"asIntSlice":         IntSlice,
	"pixel_bounds":       PixelBounds,
	"join":               tplStringJoin,
	"split":              tplStringSplit,
	"idx":                tplIndexOf,
	"point":              tplNewPoint,
	"SimpleGridFromArgs": simpleGridFromArgs,
	"args":               NewTplArgs,
	"check_args":         checkArgs,
	"indent":             Indent,
	"log_info":           infoln,
	"squarish":           squarish,
}

func infoln(template string, vals ...interface{}) string {
	values := []interface{}{template}
	values = append(values, vals...)
	str := fmt.Sprint(values...)
	log.Infoln(str)

	return fmt.Sprintf("<!-- %v -->\n", str)
}

func squarish(args *tplArgs) (*tplArgs, error) {

	division := uint(10)
	width, _ := args.GetAsFloat64(ArgsKeyImageWidth)
	height, _ := args.GetAsFloat64(ArgsKeyImageHeight)
	switch {
	case args.Has(ArgsKeyGratingNumberOfRows):
		division, _ = args.GetAsUint(ArgsKeyGratingNumberOfRows)
	case args.Has(ArgsKeyNumberOfRows):
		division, _ = args.GetAsUint(ArgsKeyNumberOfRows)
	case args.Has(ArgsKeyGratingNumberOfCols):
		division, _ = args.GetAsUint(ArgsKeyGratingNumberOfCols)
	case args.Has(ArgsKeyNumberOfCols):
		division, _ = args.GetAsUint(ArgsKeyNumberOfCols)

	}

	_, _, rows, cols := grating.Squarish(width, height, division)
	return args.Set(
		ArgsKeyGratingNumberOfRows, rows,
		ArgsKeyGratingNumberOfCols, cols,
		ArgsKeyNumberOfRows, rows,
		ArgsKeyNumberOfCols, cols,
	)

}

func Indent(spaces interface{}, content string) (string, error) {
	spc, ok := as.Int(spaces)
	if !ok {
		return content, fmt.Errorf("Indent space needs to be a number")
	}
	content = strings.TrimSpace(content)
	var newBuff strings.Builder
	spacer := strings.Repeat("\t", spc)
	lines := strings.Split(content, "\n")
	for i := range lines {
		newBuff.WriteString(spacer)
		newBuff.WriteString(lines[i])
		newBuff.WriteString("\n")
	}
	return newBuff.String(), nil
}

func tplNewPoint(x, y interface{}) geom.Point {
	xF64, _ := as.Float64(x)
	yF64, _ := as.Float64(y)
	return geom.Point{xF64, yF64}
}

func tplStringJoin(sep string, ps ...interface{}) string {
	var parts = make([]string, len(ps))
	for i, pstr := range ps {
		parts[i] = fmt.Sprintf("%v", pstr)
	}

	return strings.Join(parts, sep)
}
func tplStringSplit(sep string, it interface{}) ([]string, error) {
	str, ok := as.String(it)
	if !ok {
		return nil, fmt.Errorf("%v is not a string", it)
	}
	return strings.Split(str, sep), nil
}

func tplIndexOf(idxArg interface{}, partsArgs interface{}) (interface{}, error) {

	idx, ok := as.Int(idxArg)
	if !ok {
		return nil, fmt.Errorf("first argument should be a number")
	}
	parts, err := as.InterfaceSlice(partsArgs)
	if err != nil {
		return nil, err
	}
	return parts[idx], nil
}

// AddTemplateFuncs will add the filestore based commands. It will panic if the command is already defined.
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

func tplMathDiv(av, bv interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := as.Float64(av)
	if !ok {
		return 0, fmt.Errorf("first value (%t) needs to be a number", av)
	}
	b, ok := as.Float64(bv)
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
	a, ok := as.Float64(av)
	if !ok {
		return 0, fmt.Errorf("first value (%v) needs to be a number", av)
	}
	b, ok := as.Float64(bv)
	if !ok {
		return 0, fmt.Errorf("second value (%v) needs to be a number", bv)
	}
	return a * b, nil
}

func tplMathSub(av, bv interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := as.Float64(av)
	if !ok {
		return 0, fmt.Errorf("first value (%v) needs to be a number", av)
	}
	b, ok := as.Float64(bv)
	if !ok {
		return 0, fmt.Errorf("second value (%v) needs to be a number", bv)
	}
	return a - b, nil
}

func tplMathAdd(av, bv interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := as.Float64(av)
	if !ok {
		return 0, fmt.Errorf("first value (%v) needs to be a number", av)
	}
	b, ok := as.Float64(bv)
	if !ok {
		return 0, fmt.Errorf("second value (%v) needs to be a number", bv)
	}
	return a + b, nil
}

func tplMathNeg(av interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := as.Float64(av)
	if !ok {
		return 0, fmt.Errorf("value (%v) needs to be a number", av)
	}
	return a * -1, nil
}

func tplMathAbs(av interface{}) (float64, error) {
	// we will convert the values to float64
	a, ok := as.Float64(av)
	if !ok {
		return 0, fmt.Errorf("value (%v) needs to be a number", av)
	}
	return math.Abs(a), nil
}

func tplSeq(startArg float64, numArg uint, incArg float64) []float64 {
	start, _ := as.Float64(startArg)
	num, _ := as.Uint(numArg)
	inc, _ := as.Float64(incArg)
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

func tplRound(xArg interface{}, unitArg interface{}) float64 {
	x, _ := as.Float64(xArg)
	unit, _ := as.Int(unitArg)
	factor := math.Pow10(unit)

	return math.Round(x*factor) / factor
}

func tplRound3(x interface{}) float64 { return tplRound(x, 3) }
func tplRoundTo(unit interface{}) func(interface{}) float64 {
	return func(x interface{}) float64 { return tplRound(x, unit) }
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
		panic(&reflect.ValueError{Method: "reflect.Value.IsZero", Kind: v.Kind()})

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
	var (
		ok bool
	)
	ints := make([]int, len(vals))
	for i := range vals {
		ints[i], ok = as.Int(vals[i])
		if !ok {
			return ints, fmt.Errorf("unknown int value at %v: '%v' ", i, vals[i])
		}
	}
	return ints, nil
}

func LngLatCoord(lngArg, latArg interface{}) (lnglat coord.LngLat, err error) {
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

//PixelBounds describes a bounds in pixels.
//	Extra Values are optional values that are defined as follows:
//		extraVals[0] is the LeftBuffer (and acts as the value for other Buffer values if not defined)
//		extraVals[1] is the BottomBuffer (and acts as the TopBuffer if the TopBuffer value is not defined)
//		extraVals[2] is the RightBuffer
//		extraVals[3] is the TopBuffer
//		extraVals[4] is the ColOffset (and acts as the value for the RowOffset if it is not defined)
//		extraVals[5] is the RowOffset
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

//TransformLine transforms a line in meters to a line in pixels
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

//TransformPoint transforms a point in meter to the a point in Pixels
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
		log.Infof("BottomRight %g BottomLeft %g easting: %g : %g", structure.BottomRightUTM.Easting, structure.BottomLeftUTM.Easting, brEblE, blEbrE)
		log.Infof("before Number of Easting steps (cols): %v", numberOfStepsEasting)
		log.Infof("Number of Northing steps (rows): %v", numberOfStepsNorthing)

		part := LabelPart{
			Grid: grid,
			Unit: "m",
		}
		log.Infof("Number of Easting steps (cols): %v", numberOfStepsEasting)
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

		log.Infof("Number of Northing steps (rows): %v", numberOfStepsNorthing)
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

type simpleGrid struct {
	grate grating.Grating
}

func (sg *simpleGrid) Path() string {
	if sg == nil {
		return ""
	}
	NumberOfCols := sg.grate.Cols
	NumberOfRows := sg.grate.Rows
	lines := make([]geom.Line, 0, NumberOfCols+NumberOfRows)
	for col := uint(0); col <= NumberOfCols; col++ {
		lines = append(lines, sg.grate.LineForCol(col))
	}
	for row := uint(0); row <= NumberOfRows; row++ {
		lines = append(lines, sg.grate.LineForRow(row))
	}

	var path strings.Builder
	// convert the lines into a path
	for _, ln := range lines {
		path.WriteString(fmt.Sprintf("M %v %v ", ln[0][0], ln[0][1]))
		path.WriteString(fmt.Sprintf("L %v %v ", ln[1][0], ln[1][1]))
	}
	return path.String()
}
func (sg *simpleGrid) X() float64 {
	if sg == nil {
		return 0.0
	}
	return sg.grate.Extent[0]
}
func (sg *simpleGrid) MaxX() float64 {
	if sg == nil {
		return 0.0
	}
	return sg.grate.Extent[2]
}
func (sg *simpleGrid) Y() float64 {
	if sg == nil {
		return 0.0
	}
	return sg.grate.Extent[1]
}
func (sg *simpleGrid) MaxY() float64 {
	if sg == nil {
		return 0.0
	}
	return sg.grate.Extent[3]
}
func (sg *simpleGrid) Rows() []int {
	if sg == nil {
		return []int{}
	}
	rows := make([]int, sg.grate.Rows)
	for i := range rows {
		rows[i] = i
	}
	return rows
}
func (sg *simpleGrid) Cols() []int {
	if sg == nil {
		return []int{}
	}
	cols := make([]int, sg.grate.Cols)
	for i := range cols {
		cols[i] = i
	}
	return cols
}
func (sg *simpleGrid) YForRow(row int) float64 {
	if sg == nil {
		return 0.0
	}
	return sg.grate.YForRow(row)
}
func (sg *simpleGrid) YForRowCenterNext(row int) float64 {
	if sg == nil {
		return 0.0
	}
	y1, y2 := sg.grate.YForRow(row), sg.grate.YForRow(row+1)
	return y1 + ((y2 - y1) / 2)
}
func (sg *simpleGrid) XForColCenterNext(col int) float64 {
	if sg == nil {
		return 0.0
	}
	x1, x2 := sg.grate.XForCol(col), sg.grate.XForCol(col+1)
	return x1 + ((x2 - x1) / 2)
}

func (sg *simpleGrid) RowLabel(row int) string {
	if sg == nil {
		return ""
	}
	return sg.grate.LabelForRow(row)
}
func (sg *simpleGrid) ColLabel(col int) string {
	if sg == nil {
		return ""
	}
	return sg.grate.LabelForCol(col)
}

func simpleGridFromArgs(args tplArgs) (*simpleGrid, error) {
	if vals := args.Required(ArgsKeyX, ArgsKeyY, ArgsKeyWidth, ArgsKeyHeight); len(vals) != 0 {
		return nil, fmt.Errorf("Missing required keys: (%v)", strings.Join(vals, ","))
	}
	var (
		NumberOfCols = uint(10)
		NumberOfRows = uint(10)
		YFlip        bool
		err          error
	)
	if args.Has(ArgsKeyNumberOfRows) {
		NumberOfRows, err = args.GetAsUint(ArgsKeyNumberOfRows)
		if err != nil {
			return nil, err
		}
		NumberOfCols = NumberOfRows
	}
	if args.Has(ArgsKeyNumberOfCols) {
		NumberOfCols, err = args.GetAsUint(ArgsKeyNumberOfCols)
		if err != nil {
			return nil, err
		}
	}
	if args.Has(ArgsKeyFlipY) {
		YFlip, err = args.GetAsBool(ArgsKeyFlipY)
		if err != nil {
			return nil, err
		}
	}
	x, err := args.GetAsFloat64(ArgsKeyX)
	if err != nil {
		return nil, err
	}
	y, err := args.GetAsFloat64(ArgsKeyY)
	if err != nil {
		return nil, err
	}
	width, err := args.GetAsFloat64(ArgsKeyWidth)
	if err != nil {
		return nil, err
	}
	height, err := args.GetAsFloat64(ArgsKeyHeight)
	if err != nil {
		return nil, err
	}

	grate, err := grating.NewGrating(
		x,
		y,
		width,
		height,
		NumberOfRows,
		NumberOfCols,
		YFlip,
	)
	if err != nil {
		return nil, err
	}
	return &simpleGrid{
		grate: *grate,
	}, nil

}
