package trellis

import (
	"log"
	"math"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/planar"

	"github.com/go-spatial/geom/planar/coord"
	"github.com/go-spatial/geom/planar/coord/utm"
)

var WGS84Ellip = coord.Ellipsoid{
	Name:           "WGS_84",
	Radius:         6378137,
	Eccentricity:   0.00669438,
	NATOCompatible: true,
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
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

type Vector struct {
	Theta float64
	M     float64 // Slope
	B     float64 // Y-Intercept
}

// Travel distance from at 0,0
func (v Vector) Travel(dist float64) (x, y float64) {
	if v.Theta == 0 {
		return dist, dist
	}
	x = dist * math.Cos(v.Theta)
	y = dist * math.Sin(v.Theta)
	return x, y
}

func NewVector(line [2][2]float64) Vector {
	m, b, defined := planar.Slope(line)
	// vertical || horizontal
	if !defined || m == 0 {
		return Vector{
			M: m,
			B: b,
		}
	}
	adj := line[1][0] - line[0][0]
	opp := line[1][1] - line[0][1]
	hyp := math.Sqrt(adj*adj + opp*opp)
	log.Printf("hyp(%v) == adj2(%v) * opp2(%v)", hyp, adj, opp)
	theta := math.Acos(adj / hyp)
	return Vector{
		M:     m,
		B:     b,
		Theta: theta,
	}

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
	Ellips         coord.Ellipsoid
	BottomLeftUTM  utm.Coord
	BottomRightUTM utm.Coord
	TopLeftUTM     utm.Coord

	Grid Grid

	LeftVector Vector
	LeftOffset int

	BottomVector Vector
	BottomOffset int
}

func (structure Structure) At(col, row int) [2]float64 {
	leftVector := structure.LeftVector
	leftOffset := structure.LeftOffset
	bottomVector := structure.BottomVector
	bottomOffset := structure.BottomOffset
	size := int(structure.Grid.Size())

	distY := float64(bottomOffset + (row * size))
	lx, ly := leftVector.Travel(distY)
	if leftVector.Theta == 0 {
		lx = 0
	}

	distX := float64(leftOffset + (col * size))
	bx, by := bottomVector.Travel(distX)
	if bottomVector.Theta == 0 {
		by = 0
	}
	// subtracting as we are going up
	// adding as we are going left to right
	return [2]float64{bx - lx, by - ly}
}

func (structure Structure) NorthingBar(idx int) geom.Line {

	leftVector := structure.LeftVector
	bottomVector := structure.BottomVector
	bottomOffset := structure.BottomOffset
	size := int(structure.Grid.Size())

	var (
		xstart = 0.0
		xend   = 0.0
		ystart = 0.0
		yend   = 0.0
	)

	barLength := float64(structure.BottomRightUTM.Easting - structure.BottomLeftUTM.Easting)
	//xend = structure.BottomLeftUTM.Easting + (barLength * math.Cos(bottomVector.Theta))

	dist := float64(bottomOffset + (idx * size))
	y := dist * math.Sin(leftVector.Theta)
	b := y

	ystart -= b

	xend, yend = bottomVector.Travel(barLength)
	xend += structure.BottomLeftUTM.Easting
	yend -= b

	log.Printf("%v: offset: %v -- %v", idx, bottomOffset, y)

	return geom.Line{{xstart, ystart}, {xend, yend}}
}
func (structure Structure) EastingBar(idx int) geom.Line {

	leftVector := structure.LeftVector
	leftOffset := structure.LeftOffset
	bottomVector := structure.BottomVector
	size := int(structure.Grid.Size())

	var (
		xstart = 0.0
		xend   = 0.0
		ystart = 0.0
		yend   = 0.0
	)

	barLength := float64(structure.TopLeftUTM.Northing - structure.BottomLeftUTM.Northing)
	yend = barLength * math.Cos(leftVector.Theta)

	dist := float64(leftOffset + (idx * size))
	x := (dist * math.Cos(bottomVector.Theta))

	xstart += x

	xend, yend = leftVector.Travel(barLength)
	//yend -= structure.BottomLeftUTM.Northing
	xend += x

	log.Printf("%v: offset: %v -- %v", idx, leftOffset, x)

	return geom.Line{{xstart, ystart}, {xend, yend}}
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
func NewLngLat(bottomLeft, topRight coord.LngLat, ellips coord.Ellipsoid, grid Grid) (Structure, error) {

	topLeft := coord.LngLat{
		Lng: bottomLeft.Lng,
		Lat: topRight.Lat,
	}
	bottomRight := coord.LngLat{
		Lng: topRight.Lng,
		Lat: bottomLeft.Lat,
	}
	log.Printf("LatLng for bottomLeft:%v", bottomLeft)
	log.Printf("LatLng for topLeft:%v", topLeft)
	log.Printf("LatLng for bottomRight:%v", bottomRight)
	log.Printf("LatLng for topRight:%v", topRight)

	tlUTM, err := utm.FromLngLat(topLeft, ellips)
	if err != nil {
		return Structure{}, err
	}
	/*
		trUTM, err := utm.FromLngLat(topRight, ellips)k
		if err != nil {
			return Structure{}, err
		}
	*/

	blUTM, err := utm.FromLngLat(bottomLeft, ellips)
	if err != nil {
		return Structure{}, err
	}
	brUTM, err := utm.FromLngLat(bottomRight, ellips)
	if err != nil {
		return Structure{}, err
	}

	leftVector := NewVector([2][2]float64{
		{blUTM.Easting, blUTM.Northing},
		{tlUTM.Easting, tlUTM.Northing},
	})
	log.Printf("leftVector: b %v m %v theta: %v", leftVector.B, leftVector.M, leftVector.Theta)

	adj := harvesinDistance(bottomLeft, topLeft, ellips)
	log.Printf("tl - bl")
	opp := tlUTM.Easting - blUTM.Easting

	leftVector = NewVector([2][2]float64{
		{0, 0},
		{opp, adj},
	})
	log.Printf("leftVector: b %v m %v theta: %v", leftVector.B, leftVector.M, leftVector.Theta)

	bottomVector := NewVector([2][2]float64{
		{blUTM.Easting, blUTM.Northing},
		{brUTM.Easting, brUTM.Northing},
	})
	log.Printf("BottomVector: b %v m %v theta: %v", bottomVector.B, bottomVector.M, bottomVector.Theta)

	adj = harvesinDistance(bottomLeft, bottomRight, ellips)
	opp = brUTM.Northing - blUTM.Northing

	bottomVector = NewVector([2][2]float64{
		{0, 0},
		{adj, opp},
	})
	log.Printf("BottomVector: b %v m %v theta: %v", bottomVector.B, bottomVector.M, bottomVector.Theta)

	log.Printf("BottomLTUTM: %v", blUTM)
	log.Printf("BottomRTUTM: %v", brUTM)

	_, _, bottomOffset := grid.PartsFor(int64(blUTM.Northing))
	log.Printf("BottomLeft Northing: %g --- offset: %v", blUTM.Northing, bottomOffset)
	_, _, leftOffset := grid.PartsFor(int64(blUTM.Easting))
	log.Printf("BottomLeft Easting: %g --- offset: %v", blUTM.Easting, leftOffset)

	size := int(grid.Size())

	bottomOffset = size - bottomOffset
	leftOffset = size - leftOffset
	return Structure{
		Ellips:         ellips,
		BottomLeftUTM:  blUTM,
		BottomRightUTM: brUTM,
		TopLeftUTM:     tlUTM,

		Grid: grid,

		LeftVector: leftVector,
		LeftOffset: leftOffset,

		BottomVector: bottomVector,
		BottomOffset: bottomOffset,
	}, nil
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

// Size is the grid size in meters
func (g Grid) Size() int64 { return int64(g) }

var widths = map[Grid]int{
	Grid(1):     0,
	Grid(10):    1,
	Grid(100):   2,
	Grid(1000):  3,
	Grid(10000): 4,
}

// Width is the number width
func (g Grid) Width() int {
	if w, ok := widths[g]; ok {
		return w
	}
	return int(math.Log10(float64(g)))
}
