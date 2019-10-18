package trellis

import (
	"log"
	"math"

	"github.com/go-spatial/geom/planar/coord"
	"github.com/go-spatial/geom/planar/coord/utm"
)

var WGS84Ellip = coord.Ellipsoid{
	Name:           "WGS_84",
	Radius:         6378137,
	Eccentricity:   0.00669438,
	NATOCompatible: true,
}

type Offset struct {
	A11Offset int // original non skewed offset
	A12Offset int // original non skewed offset

	// Offset is the initial offset before bars
	// need to start to be drawn
	StartOffset float64 // in meters
	// This is the variation of the opposite coordinate
	EndOffset float64 // in meters

	// Steps is number of bars to draw
	Steps         int
	StartStepSize float64
	EndStepSize   float64

	// Length of the bar
	Length float64
}

// Bar describes a bar to be drawn
type Bar struct {
	// The UTM coord for the start of the bar
	Start *utm.Coord
	End   *utm.Coord

	X1, Y1, X2, Y2 float64

	LabelOffset float64

	StartOffset   float64 // in meters
	EndOffset     float64 // in meters
	StepSize      float64 // in meters
	NumberOfSteps int
	Length        float64 // in meters

	// Parent is the structure that containts this bar
	Parent *Structure
}

// DrawFn is the callback function for drawing a bar
type DrawFn func(count int, bar Bar) error

// Structure describes vertical and horizontal bars in the Trellis
type Structure struct {
	TopLeft        coord.LngLat
	BottomRight    coord.LngLat
	Ellips         coord.Ellipsoid
	TopLeftUTM     utm.Coord
	TopRightUTM    utm.Coord
	BottomLeftUTM  utm.Coord
	BottomRightUTM utm.Coord

	NATO     bool
	Grid     Grid
	Northing Offset
	Easting  Offset
}

// NorthingBars are the Horizontal bars that travel south to north
func (str Structure) NorthingBars(fn DrawFn) error {

	log.Printf("Northing steps: %v -- %v", str.Northing.Steps, str.Northing.Length)
	for i := 0; i < str.Northing.Steps; i++ {
		utmA11Offset := float64(str.Northing.A11Offset + (i * int(str.Grid)))
		utmA12Offset := float64(str.Northing.A12Offset + (i * int(str.Grid)))
		utmStart := str.BottomLeftUTM
		utmEnd := str.TopLeftUTM
		utmStart.Northing += utmA11Offset
		utmEnd.Northing += utmA12Offset

		startStepOffset := (float64(i) * str.Northing.StartStepSize)
		endStepOffset := (float64(i) * str.Northing.EndStepSize)
		x1 := 0.0
		y1 := str.Northing.StartOffset + startStepOffset
		x2 := str.Easting.Length
		y2 := str.Northing.EndOffset + endStepOffset
		err := fn(i, Bar{
			Start:  &utmStart,
			End:    &utmEnd,
			X1:     x1,
			Y1:     y1,
			X2:     x2,
			Y2:     y2,
			Length: str.Easting.Length,
			Parent: &str,
		})
		if err != nil {
			return err
		}
	}
	return nil

}

// EastingBars are the vertical bars in meters that travel from east to west
func (str Structure) EastingBars(fn DrawFn) error {
	log.Printf("v3 Easting steps: %v -- %v", str.Easting.Steps, str.Easting.Length)
	for i := 0; i < str.Easting.Steps; i++ {

		utmA11Offset := float64(str.Easting.A11Offset + (i * int(str.Grid)))
		utmA12Offset := float64(str.Easting.A12Offset + (i * int(str.Grid)))
		utmStart := str.BottomLeftUTM
		utmEnd := str.BottomRightUTM
		utmStart.Easting += utmA11Offset
		utmEnd.Easting += utmA12Offset

		startStepOffset := (float64(i) * str.Easting.StartStepSize)
		endStepOffset := (float64(i) * str.Easting.EndStepSize)
		x1 := str.Easting.StartOffset + startStepOffset
		y1 := 0.0
		x2 := str.Easting.EndOffset + endStepOffset
		y2 := str.Northing.Length
		err := fn(i, Bar{
			Start:  &utmStart,
			End:    &utmEnd,
			X1:     x1,
			Y1:     y1,
			X2:     x2,
			Y2:     y2,
			Length: str.Northing.Length,
			Parent: &str,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type Grid int

const (
	Grid100 Grid = 100
	Grid1K  Grid = 1000
)

func (g Grid) PartsFor(meters int64) (prefix, label, suffix int) {

	mask := int64(g)

	suffix = int(meters % mask)
	val := meters / mask
	label = int(val % 100)
	prefix = int(val / 100)

	return prefix, label, suffix
}

func (g Grid) Size() int64 { return int64(g) }

var widths = map[Grid]int{
	Grid(1):     0,
	Grid(10):    1,
	Grid(100):   2,
	Grid(1000):  3,
	Grid(10000): 4,
}

func (g Grid) Width() int {
	if w, ok := widths[g]; ok {
		return w
	}
	return int(math.Log10(float64(g)))
}

func calculateStepOffsets(grid Grid, b, a1, a2 float64) (startOffset, endOffset, numberOfSteps, stepSize, length float64, a1Offset, a2Offset int) {

	a := a2 - a1
	// this is the overall length of the bar
	length = math.Sqrt(b*b + a*a)

	ratio := length / a

	_, _, a1OffsetP := grid.PartsFor(int64(a1))
	a1Offset = int(grid) - a1OffsetP
	_, _, a2Offset = grid.PartsFor(int64(a2))
	oLength := a - float64(a1Offset+a2Offset)

	numberOfSteps = float64(int(oLength / float64(grid)))
	stepSize = float64(grid) * ratio
	startOffset = float64(a1Offset) * ratio
	endOffset = float64(a2Offset) * ratio

	return startOffset, endOffset, numberOfSteps, stepSize, length, a1Offset, a2Offset

}

func NewLngLat2(topLeft, bottomRight coord.LngLat, ellips coord.Ellipsoid, grid Grid) (Structure, error) {

	tlUTM, err := utm.FromLngLat(topLeft, ellips)
	if err != nil {
		return Structure{}, err
	}
	trUTM, err := utm.FromLngLat(coord.LngLat{
		Lng: bottomRight.Lng,
		Lat: topLeft.Lat,
	}, ellips)
	if err != nil {
		return Structure{}, err
	}

	blUTM, err := utm.FromLngLat(coord.LngLat{
		Lng: topLeft.Lng,
		Lat: bottomRight.Lat,
	}, ellips)
	if err != nil {
		return Structure{}, err
	}
	brUTM, err := utm.FromLngLat(bottomRight, ellips)
	if err != nil {
		return Structure{}, err
	}

	log.Printf("\ntlUTM: %#v\ntrUTM: %#v", tlUTM, trUTM)
	log.Printf("\nblUTM: %#v\nbrUTM: %#v", blUTM, brUTM)

	nlStartOffset, _, nlNumberOfSteps, nlStepSize, nlLength, nlA1Offset, _ :=
		calculateStepOffsets(grid,
			tlUTM.Easting-blUTM.Easting, // b
			blUTM.Northing,              // a1
			tlUTM.Northing,              // a2
		)
	nrStartOffset, _, nrNumberOfSteps, nrStepSize, nrLength, nrA1Offset, _ :=
		calculateStepOffsets(grid,
			trUTM.Easting-brUTM.Easting, // b
			brUTM.Northing,              // a1
			trUTM.Northing,              // a2
		)

	ebStartOffset, _, ebNumberOfSteps, ebStepSize, ebLength, ebA1Offset, _ :=
		calculateStepOffsets(grid,
			blUTM.Northing-brUTM.Northing, // b
			blUTM.Easting,                 // a1
			brUTM.Easting,                 // a2
		)
	etStartOffset, _, etNumberOfSteps, etStepSize, etLength, etA1Offset, _ :=
		calculateStepOffsets(grid,
			tlUTM.Northing-trUTM.Northing, // b
			tlUTM.Easting,                 // a1
			trUTM.Easting,                 // a2
		)

	log.Printf(
		"Easting Top: StartOffset: %v A1Offsetset: %v NumberOfSteps: %v StepSize: %v Lenght: %v",
		etStartOffset,
		etA1Offset,
		etNumberOfSteps,
		etStepSize,
		etLength,
	)

	return Structure{
		TopLeft:        topLeft,
		BottomRight:    bottomRight,
		Ellips:         ellips,
		TopLeftUTM:     tlUTM,
		TopRightUTM:    trUTM,
		BottomLeftUTM:  blUTM,
		BottomRightUTM: brUTM,

		Grid: grid,
		Northing: Offset{
			Length: math.Max(nlLength, nrLength),

			Steps:         int(math.Max(nlNumberOfSteps, nrNumberOfSteps)) + 1,
			StartStepSize: nlStepSize,
			StartOffset:   nlStartOffset,
			EndStepSize:   nrStepSize,
			EndOffset:     nrStartOffset,
			A11Offset:     nlA1Offset,
			A12Offset:     nrA1Offset,
		},
		Easting: Offset{
			Length: math.Max(ebLength, etLength),

			Steps:         int(math.Max(ebNumberOfSteps, etNumberOfSteps)) + 1,
			StartStepSize: ebStepSize,
			StartOffset:   ebStartOffset,
			EndStepSize:   etStepSize,
			EndOffset:     etStartOffset,
			A11Offset:     ebA1Offset,
			A12Offset:     etA1Offset,
		},
	}, nil
}
