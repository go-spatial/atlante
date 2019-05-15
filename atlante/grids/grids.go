package grids

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/spherical"
	"github.com/go-spatial/maptoolkit/atlante/internal/resolution"
	"github.com/go-spatial/maptoolkit/mbgl/bounds"
)

type MDGID struct {
	ID string
	// start's at 1, zero means it does not exists
	Part uint
}

func (m MDGID) String() string {
	if m.Part > 0 {
		return fmt.Sprintf("%s:%d", m.ID, m.Part)
	}
	return m.ID
}

func NewMDGID(m string) MDGID {
	idx := strings.IndexByte(strings.TrimSpace(m), ':')
	if idx == -1 || len(m) <= idx {
		return MDGID{ID: m}
	}
	i, err := strconv.Atoi(m[idx+1:])
	if err != nil {
		return MDGID{ID: m}
	}
	return MDGID{
		ID:   m[:idx],
		Part: uint(i),
	}
}

// Provider returns a grid object that can be used to generate
// a Map Grid
type Provider interface {
	GridForLatLng(lat, lng float64, srid uint) (*Grid, error)
	GridForMDGID(mgdID MDGID) (*Grid, error)
}

type UTMInfo struct {
	Zone string
	Hemi string
}

type EditInfo struct {
	By   string
	Date time.Time
}

type Grid struct {
	MdgID MDGID

	Sheet  string
	Series string
	NRN    string

	// Degree, Minute, Seconds representatino
	SWLatDMS string
	SWLngDMS string
	NELatDMS string
	NELngDMS string

	// Decimal representation
	SWLat float64
	SWLng float64
	NELat float64
	NELng float64

	LatLen float64
	LngLen float64

	Country string
	City    string

	PublicationDate time.Time
	UTM             *UTMInfo
	Metadata        map[string]string
	Edited          *EditInfo
}

// ReferenceNumber is the MDG id reference number, including a
// sub-part number if there is one.
func (g Grid) ReferenceNumber() string {
	if g.MdgID.Part == 0 {
		return g.MdgID.ID
	}
	return fmt.Sprintf("%s-%v", g.MdgID.ID, g.MdgID.Part)
}

// SheetNumber is the sheet number for the grid, including a
// sub-part number if there is one.
func (g Grid) SheetNumber() string {
	if g.MdgID.Part == 0 {
		return g.Sheet
	}
	return fmt.Sprintf("%s-%v", g.Sheet, g.MdgID.Part)
}

func (g Grid) Zone() string {
	if g.UTM == nil {
		return ""
	}
	return g.UTM.Zone
}
func (g Grid) Hemi() string {
	if g.UTM == nil {
		return "N"
	}
	return g.UTM.Hemi
}

func (g Grid) NE() [2]float64     { return [2]float64{g.NELng, g.NELat} }
func (g Grid) SW() [2]float64     { return [2]float64{g.SWLng, g.SWLat} }
func (g Grid) Hull() *geom.Extent { return spherical.Hull(g.NE(), g.SW()) }

func (g Grid) CenterPtForZoom(zoom float64) [2]float64 {
	return bounds.Center(g.Hull(), zoom)
}
func (g Grid) WidthHeightForZoom(zoom float64) (width, height float64) {
	log.Println("Hull", g.Hull())
	return bounds.WidthHeightTile(g.Hull(), zoom, 4096/8)
}

func (g Grid) ZoomForScaleDPI(scale uint, dpi uint) float64 {
	log.Println("SWLat", g.SWLat)
	return resolution.Zoom(resolution.MercatorEarthCircumference, scale, dpi, g.SWLat)
}

func CalculateSecLengths(latitude float64) (latLen, lngLen float64) {
	// from https://msi.nga.mil/msisitecontent/staticfiles/calculators/degree.html
	const (
		m1 = 111132.92 // latitude calculation term 1
		m2 = -559.82   // latitude calculation term 2
		m3 = 1.175     // latitude calculation term 3
		m4 = -0.0023   // latitude calculation term 4
		p1 = 111412.84 // longitude calculation term 1
		p2 = -93.5     // longitude calculation term 2
		p3 = 0.118     // longitude calculation term 3
	)

	// convert to radians
	lat := latitude * ((2.0 * math.Pi) / 360.0)
	latLen = (m1 + (m2 * math.Cos(2*lat)) + (m3 * math.Cos(4*lat)) + (m4 * math.Cos(6*lat))) / 3600
	lngLen = ((p1 * math.Cos(lat)) + (p2 * math.Cos(3*lat)) + (p3 * math.Cos(5*lat))) / 3600
	return latLen, lngLen
}
