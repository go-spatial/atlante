//go:generate protoc "--go_out=paths=source_relative:." "cell.proto"

package grids

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/planar/coord"
	"github.com/go-spatial/geom/spherical"
	"github.com/go-spatial/maptoolkit/atlante/internal/resolution"
	"github.com/go-spatial/maptoolkit/mbgl/bounds"
)

// CellSize is the size of the cell in meters
type CellSize uint

func (cs CellSize) String() string {
	switch cs {
	case CellSize5K:
		return "5K"
	case CellSize50K:
		return "50K"
	default:
		return fmt.Sprintf("%vm", uint(cs))
	}
}

const (
	// CellSize5K returns cell sizes of 5 km
	CellSize5K = 5000

	// CellSize50K returns cell sizes of 50 km
	CellSize50K = 50000

	// CellSize250K returns cell sizes of 250 km
	CellSize250K = 250000
)

// Provider returns a grid object that can be used to generate
// a Map Grid, if the grid object is not found or does not exist
// the CellFor* methods should return a nil cell, and ErrNotFound
// error
type Provider interface {
	CellForBounds(bounds geom.Extent, srid uint) (*Cell, error)
	CellForLatLng(lat, lng float64, srid uint) (*Cell, error)
	CellForMDGID(mgdID *MDGID) (*Cell, error)
	CellSize() CellSize
}

// AsString provides a human readable version of the MDGID
func (m *MDGID) AsString() string {
	if m == nil {
		return ""
	}
	if m.Part > 0 {
		return fmt.Sprintf("%s:%d", m.Id, m.Part)
	}
	return m.Id
}

// NewMDGID take a string representing an MDGID with an optional part returning a new MDGID object
func NewMDGID(m string) *MDGID {
	trimedID := strings.TrimSpace(m)
	idx := strings.IndexByte(trimedID, '-')
	if idx == -1 {
		// Let's try ':'
		idx = strings.IndexByte(trimedID, ':')
	}
	if idx == -1 || len(m) <= idx {
		return &MDGID{Id: m}
	}
	i, err := strconv.Atoi(m[idx+1:])
	if err != nil {
		return &MDGID{Id: m}
	}
	return &MDGID{
		Id:   m[:idx],
		Part: uint32(i),
	}
}

// NewEditInfo returns a new editInfo struct
func NewEditInfo(by string, when time.Time) *EditInfo {
	// don't care for the error
	t, _ := ptypes.TimestampProto(when)
	return &EditInfo{
		By:   by,
		Date: t,
	}
}

// NewUTM returns a new utm object
func NewUTM(zone uint8, hemi HEMIType) *UTMInfo {
	if zone < 1 || zone > 60 {
		zone = 1
	}
	return &UTMInfo{
		Zone: uint32(zone),
		Hemi: hemi,
	}
}

// NewCell returns a new cell object
func NewCell(
	mdgid string,
	sw [2]float64,
	ne [2]float64,
	country string,
	city string,
	utm *UTMInfo,
	editInfo *EditInfo,
	publishedAt time.Time,
	nrn string,
	sheet string,
	series string,
	dmsSW [2]string,
	dmsNE [2]string,
	metadata map[string]string,
) *Cell {

	latlen, lnglen := CalculateSecLengths(ne[1])
	// If there is an error we will return the zero value for time.
	pubat, _ := ptypes.TimestampProto(publishedAt)

	swlatlng := &Cell_LatLng{Lat: float32(sw[0]), Lng: float32(sw[1])}
	nelatlng := &Cell_LatLng{Lat: float32(ne[0]), Lng: float32(ne[1])}

	if utm == nil {
		utm = nelatlng.ToUTMInfo()
	}

	if dmsSW[0] == "" || dmsSW[1] == "" {
		dms := swlatlng.ToDMS()
		dmsSW[0], dmsSW[1] = dms[0].String(), dms[1].String()
	}

	if dmsNE[0] == "" || dmsNE[1] == "" {
		dms := nelatlng.ToDMS()
		dmsNE[0], dmsNE[1] = dms[0].String(), dms[1].String()
	}

	return &Cell{
		Mdgid: NewMDGID(mdgid),
		Sw:    swlatlng,
		Ne:    nelatlng,
		Len:   &Cell_LatLng{Lat: float32(latlen), Lng: float32(lnglen)},
		NeDms: &Cell_LatLngDMS{Lat: dmsNE[0], Lng: dmsNE[1]},
		SwDms: &Cell_LatLngDMS{Lat: dmsSW[0], Lng: dmsSW[1]},

		Nrn:    nrn,
		Sheet:  sheet,
		Series: series,

		Country: country,
		City:    city,

		PublishedAt: pubat,
		Edited:      editInfo,
		Utm:         utm,
		MetaData:    metadata,
	}
}

// Init will fill out fields with default values based on SW and NE values.
func (c *Cell) Init() {
	ne := c.Ne

	if c.Utm == nil && ne != nil {
		c.Utm = ne.ToUTMInfo()
	}
	if c.NeDms == nil && ne != nil {
		dms := ne.ToDMS()
		c.NeDms = &Cell_LatLngDMS{Lat: dms[0].String(), Lng: dms[1].String()}
	}

	sw := c.Sw
	// Need to initialize the Len based on sw.
	if c.Len == nil && sw != nil {
		latlen, lnglen := CalculateSecLengths(float64(sw.Lat))
		c.Len = &Cell_LatLng{Lat: float32(latlen), Lng: float32(lnglen)}
	}
	if c.SwDms == nil && sw != nil {
		dms := sw.ToDMS()
		c.SwDms = &Cell_LatLngDMS{Lat: dms[0].String(), Lng: dms[1].String()}
	}
}

// PublicationDate returns the date the cell was published.
func (c *Cell) PublicationDate() (time.Time, error) {
	return ptypes.Timestamp(c.PublishedAt)
}

// ReferenceNumber is the MDG id reference number, including a
// sub-part number if there is one.
func (c *Cell) ReferenceNumber() string {
	if c.Mdgid.Part == 0 {
		return c.Mdgid.Id
	}
	return fmt.Sprintf("%s-%v", c.Mdgid.Id, c.Mdgid.Part)
}

// SheetNumber is the sheet number for the grid, including a
// sub-part number if there is one.
func (c *Cell) SheetNumber() string {
	if c.Mdgid.Part == 0 {
		return c.Sheet
	}
	return fmt.Sprintf("%s-%v", c.Sheet, c.Mdgid.Part)
}

// Zone returns the string representation of a zone i.e. 01-60, if the zone is out of that range
// we will return "01"
func (c *Cell) Zone() string {
	utm := c.GetUtm()
	if utm == nil {
		return "01"
	}
	z := utm.GetZone()
	if z == 0 || z > 60 {
		return "01"
	}
	return fmt.Sprintf("%02d", z)
}

// Hemi return the hemisphere
func (c *Cell) Hemi() string {
	if u := c.GetUtm(); u == nil || u.GetHemi() != HEMIType_SOUTH {
		return "N"
	}
	return "S"
}

// NELatDMS returns the DMS version of the NE lat
func (c *Cell) NELatDMS() (string, error) {
	dms := c.GetNeDms().GetLat()
	if dms == "" {
		dms = c.GetNe().ToDMS()[0].AsString(1)
	}
	return dms, nil
}

// NELngDMS returns the DMS version of the NE lng
func (c *Cell) NELngDMS() (string, error) {
	dms := c.GetNeDms().GetLng()
	if dms == "" {
		dms = c.GetNe().ToDMS()[1].AsString(1)
	}
	return dms, nil
}

// SWLatDMS returns the DMS version of the SW lat
func (c *Cell) SWLatDMS() (string, error) {
	dms := c.GetSwDms().GetLat()
	if dms == "" {
		dms = c.GetSw().ToDMS()[0].AsString(1)
	}
	return dms, nil
}

// SWLngDMS returns the DMS version of the NE lng
func (c *Cell) SWLngDMS() (string, error) {
	dms := c.GetSwDms().GetLng()
	if dms == "" {
		dms = c.GetSw().ToDMS()[1].AsString(1)
	}
	return dms, nil
}

// LatLen will return the lat arc length.
func (c *Cell) LatLen() float64 {
	return float64(c.GetLen().GetLat())
}

// LngLen will return the lat arc length.
func (c *Cell) LngLen() float64 {
	return float64(c.GetLen().GetLng())
}

// NE will return the North East coordinate
func (c *Cell) NE() [2]float64 {
	ne := c.GetNe()
	return [2]float64{float64(ne.GetLng()), float64(ne.GetLat())}
}

// NW will return the North West coordinate
func (c *Cell) NW() [2]float64 {
	ne := c.GetNe()
	sw := c.GetSw()
	return [2]float64{float64(ne.GetLng()), float64(sw.GetLat())}
}

// SW will return the South West coordinate
func (c *Cell) SW() [2]float64 {
	sw := c.GetSw()
	return [2]float64{float64(sw.GetLng()), float64(sw.GetLat())}
}

// SE will return the South East coordinate
func (c *Cell) SE() [2]float64 {
	ne := c.GetNe()
	sw := c.GetSw()
	return [2]float64{float64(sw.GetLng()), float64(ne.GetLat())}
}

// Hull returns the hull of the Cell
func (c *Cell) Hull() *geom.Extent { return spherical.Hull(c.NE(), c.SW()) }

// CenterPtForZoom returns the center point of the bounds for the given zoom value
func (c *Cell) CenterPtForZoom(zoom float64) [2]float64 {
	return bounds.Center(c.Hull(), zoom)
}

// WidthHeightForZoom return the width and height in pixels of the bounds for the given zoom
func (c *Cell) WidthHeightForZoom(zoom float64) (width, height float64) {
	return bounds.WidthHeightTile(c.Hull(), zoom, 4096/8)
}

// ZoomForScaleDPI returns the zoom value tto use for the given scale and dpi values.
func (c *Cell) ZoomForScaleDPI(scale uint, dpi uint) float64 {
	return resolution.Zoom(resolution.MercatorEarthCircumference, scale, dpi, c.SW()[1])
}

// ToUTMInfo will return the utm info values based on the lat.
func (latlng *Cell_LatLng) ToUTMInfo() *UTMInfo {
	lat, lng := float64(latlng.GetLat()), float64(latlng.GetLng())
	z := zoneFromLatLng(lat, lng)
	h := HEMIType_NORTH
	if lat < 0 {
		h = HEMIType_SOUTH
	}
	return &UTMInfo{
		Zone: uint32(z),
		Hemi: h,
	}
}

func (latlng *Cell_LatLng) CoordLngLat() coord.LngLat {
	return coord.LngLat{
		Lng: float64(latlng.Lng),
		Lat: float64(latlng.Lat),
	}
}

// ToDMS returns the DMS (degree minute section, hemisphere) version of the encoded lat lng
func (latlng *Cell_LatLng) ToDMS() [2]DMS {
	return ToDMS(float64(latlng.GetLat()), float64(latlng.GetLng()))
}

// CalculateSecLengths returns the arch-lengths for the latitude
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

// DMS is the degree minutes and seconds
type DMS struct {
	Degree     int64
	Minute     int64
	Second     float64
	Hemisphere rune
}

// String returns the string representation.
func (dms DMS) String() string { return dms.AsString(0) }

func (dms DMS) AsString(prec uint) string {
	if prec != 0 {
		return fmt.Sprintf("%d°%d'%.*f\"%c", dms.Degree, dms.Minute, prec, dms.Second, dms.Hemisphere)
	}
	return fmt.Sprintf("%d°%d'%f\"%c", dms.Degree, dms.Minute, dms.Second, dms.Hemisphere)
}

func toDMS(v float64) (d int64, m int64, s float64) {
	var frac float64
	df, frac := math.Modf(v)
	mf, frac := math.Modf(60 * frac)
	s = 60 * frac
	return int64(math.Abs(df)), int64(math.Abs(mf)), math.Abs(s)
}

// ToDMS will take a lat/lon value and convert it to a DMS value
func ToDMS(lat, lng float64) (todms [2]DMS) {
	defer func() { log.Printf("converted %v, %v to %v", lat, lng, todms) }()
	latD, latM, latS := toDMS(lat)
	latH := 'N'
	if lat < 0 {
		latH = 'S'
	}
	lngD, lngM, lngS := toDMS(lng)
	lngH := 'E'
	if lng < 0 {
		lngH = 'W'
	}
	return [2]DMS{
		DMS{
			Degree:     latD,
			Minute:     latM,
			Second:     latS,
			Hemisphere: latH,
		},
		DMS{
			Degree:     lngD,
			Minute:     lngM,
			Second:     lngS,
			Hemisphere: lngH,
		},
	}
}

// zoneFromLatLng get the lat zone given the two values.
// The returned value will be from 1-60, if 0 is returned
// it means that the lat,lng value was in the polar region
// and UPS should be used.
// Transcribed from: https://github.com/gdey/GDGeoCocoa/blob/master/GDGeoCoordConv.m
func zoneFromLatLng(lat, lng float64) int {
	if (lat > 84.0 && lat < 90.0) || // North Pole
		(lat > -80.0 && lat < -90.0) { // South Pole
		return 0
	}

	// Adjust for projects.
	switch {
	case lat >= 56.0 && lat < 64.0 && lng >= 3.0 && lng < 12.0:
		return 32
	case lat >= 72.0 && lat < 84.0:
		switch {
		case lng >= 0.0 && lng < 9.0:
			return 31
		case lng >= 9.0 && lng < 21.0:
			return 33
		case lng >= 21.0 && lng < 33.0:
			return 35
		case lng >= 33.0 && lng < 42.0:
			return 37
		}
	}
	// Recast from [-180,180) to [0,360).
	// the w<-> is then divided into 60 zones from 1-60.
	return int((lng+180)/6) + 1
}
