package grid5k

import (
	"fmt"
	"math"

	"github.com/prometheus/common/log"

	"github.com/gdey/errors"
	"github.com/go-spatial/geom"
	"github.com/go-spatial/maptoolkit/atlante/grids"
)

// Type of the grid this provider is providing
const Type = "grid5k"

// Provider is the 5k grid provider based on a another provider
type Provider struct {
	Provider grids.Provider
}

const (
	// ConfigKeyProvider is the config key for the base provider
	ConfigKeyProvider = "provider"

	// ErrBlankSubprovider is returned when the base provider name is blank
	ErrBlankSubprovider = errors.String("error, base provider name is blank")

	//ErrInvalidSheetNumber is returned when the sheet number is above 100
	ErrInvalidSheetNumber = errors.String("error, invalid sheet number")
)

// ErrUnsupportedCellSize is returned when the cell size of the base provided is not 50k
type ErrUnsupportedCellSize grids.CellSize

// Error implements the Error interface
func (err ErrUnsupportedCellSize) Error() string {
	return fmt.Sprintf("error, unsupported cell size (%v), only support 50K", grids.CellSize(err))
}

func init() {
	grids.Register(Type, NewGridProvider, nil)
}

// NewGridProvider returns a grid provider based off the Original provider creating
// subdivision of those grids.
func NewGridProvider(config grids.ProviderConfig) (grids.Provider, error) {
	subp, err := config.String(ConfigKeyProvider, nil)
	if err != nil {
		return nil, err
	}
	if subp == "" {
		return nil, ErrBlankSubprovider
	}

	log.Infof("getting provider(%v) from config.", subp)
	prv, err := config.NameGridProvider(subp)
	if err != nil {
		log.Warnf("got error getting provider: %v", err)
		return nil, err
	}

	if prv.CellSize() != grids.CellSize50K {
		return nil, ErrUnsupportedCellSize(prv.CellSize())
	}
	return &Provider{Provider: prv}, nil
}

// CellSize returns the grid cell size
func (*Provider) CellSize() grids.CellSize { return grids.CellSize5K }

// CellForBounds returns a grid cell for the given bounds
func (p *Provider) CellForBounds(bounds geom.Extent, srid uint) (*grids.Cell, error) {
	return p.Provider.CellForBounds(bounds, srid)
}

// CellForLatLng returns a grid cell for the given Lat Lng
func (p *Provider) CellForLatLng(lat, lng float64, srid uint) (*grids.Cell, error) {
	grd, err := p.Provider.CellForLatLng(lat, lng, srid)
	if err != nil {
		return nil, err
	}

	part := mdgidPart(grd, lat, lng)
	grd.Mdgid.Part = uint32(part)

	return adjustGrid(grd)
}

// CellForMDGID returns a grid cell for the given mdgid
func (p *Provider) CellForMDGID(mdgid *grids.MDGID) (*grids.Cell, error) {
	log.Infof("Getting mdgid %v", mdgid)
	part := mdgid.Part
	mdgid.Part = 0
	grd, err := p.Provider.CellForMDGID(mdgid)
	if err != nil {
		return nil, err
	}
	grd.Mdgid.Part = part
	return adjustGrid(grd)
}

func adjustGrid(grid *grids.Cell) (*grids.Cell, error) {
	part := grid.Mdgid.Part
	if part > 100 {
		return nil, ErrInvalidSheetNumber
	}

	if part <= 1 {
		grid.Mdgid.Part = 1
	}
	n, s, w, e := coords5kSheet(grid)
	grid.Ne.Lat = float32(n)
	grid.Sw.Lat = float32(s)
	grid.Sw.Lng = float32(w)
	grid.Ne.Lng = float32(e)
	return grid, nil
}

func coords5kSheet(grid *grids.Cell) (n, s, w, e float64) {

	const (
		sqrSide = 0.025
	)

	sw := grid.GetSw()
	ne := grid.GetNe()

	swNeDiff := math.Abs(float64(sw.GetLng() - ne.GetLng()))
	width := sqrSide
	if swNeDiff > width {
		width = swNeDiff / 10
	}
	part := int(grid.Mdgid.Part)
	if part < 1 {
		part = 1
	}
	if part > 100 {
		return 0.0, 0.0, 0.0, 0.0
	}

	var a, b float64

	// TODO(gdey): There is something weird about this math. We need to take a look
	// at what is going on there and if the gridding is correct.
	// parts are numbered from 1 - 100
	//
	// I think the code should be:
	//
	// 	a, b := float64(int(k/10)), float64(int(k%10))
	// 	if b == 0.0 {
	//		a--
	//		b = 10.0
	// 	}
	//
	// This means b will never be 0. Which breaks the below,
	// Unless that need to be changed to 10.0, so that blocks
	// at the end of the row have special treatment.
	//

	switch {
	case part == 100:
		a = 10.0
		b = 0.0
	case part < 11:
		a = 0.0
		b = float64(part)
	default:
		a, b = float64(int(part/10)), float64(int(part%10))
	}

	if b == 0.0 {
		n = float64(ne.GetLat()) - (a * sqrSide) + sqrSide
		s = n - sqrSide
		w = float64(ne.GetLng()) - width
		e = float64(ne.GetLng())
	} else {
		n = float64(ne.GetLat()) - (a * sqrSide)
		s = n - sqrSide
		w = float64(sw.GetLng()) + ((b - 1) * width)
		e = w + width
	}
	return n, s, w, e

}

func mdgidPart(grid *grids.Cell, lat, lng float64) uint {
	const (
		sqrSide = 0.025
	)
	sw := grid.GetSw()
	ne := grid.GetNe()

	swNeDiff := math.Abs(float64(sw.GetLng() - ne.GetLng()))
	width := sqrSide
	if swNeDiff > width {
		width = swNeDiff / 10
	}

	eastDiff := int(math.Abs(float64(sw.GetLng())-lng)/width) + 1
	northDiff := int(math.Abs(float64(ne.GetLat())-lat) / sqrSide)

	if eastDiff == 10 {
		northDiff++
		eastDiff = 0
	}
	return uint((northDiff * 10) + eastDiff)
}
