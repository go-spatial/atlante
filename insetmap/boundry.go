package insetmap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/encoding/wkb"
	"github.com/jackc/pgx/v4/pgxpool"
)

type BoundarySQL struct {
	Main     string // The sql to get the region
	Boundary string // the sql to get the boundary lines
}

type Boundary struct {
	*Inset
	SQLS []BoundarySQL
}

func NewBoundary(db *pgxpool.Pool, config Config, gCSSDir string, gCSSMap CSSMap, gCSSDefault string) (*Boundary, error) {
	inset, err := New(db, config, gCSSDir, gCSSMap, gCSSDefault)
	if err != nil {
		return nil, err
	}
	var sqls []BoundarySQL
	for i := range config.Boundaries {
		sqls = append(sqls, BoundarySQL{
			Main:     string(config.Boundaries[i].Main),
			Boundary: string(config.Boundaries[i].Boundary),
		})
	}
	return &Boundary{
		Inset: inset,
		SQLS:  sqls,
	}, nil
}

func (b *Boundary) For(ctx context.Context, mdgid string, cssKey string) (*BoundaryMap, error) {
	var (
		name, namel, namer, order,
		class, id, parent string
		geobytes   []byte
		countries  []regionLayer
		boundaries []boundaryLine

		imap = new(Map)
		err  error
	)

	if err = b.Inset.initBaseVars(ctx, imap, mdgid, cssKey); err != nil {
		return nil, err
	}
	if err = b.Inset.initLayers(ctx, imap, imap.totalExtent); err != nil {
		return nil, err
	}

	{
		// boundaryMapper maps the id name of a boundary to the boundary entry.
		var (
			boundaryMapper = make(map[string][]int)
			lcountries     []*regionLayer  // this should be countries in the boundary map
			lboundaries    []*boundaryLine // this is the boundary lines between countries
		)

		// first we will grab all the countries and boundaries... then we will restructure them
		for i, bdry := range b.SQLS {
			sql := replaceTokens(bdry.Main, imap.totalExtent)
			rows, err := b.Query(ctx, sql)
			if err != nil {
				return nil, fmt.Errorf("SQL:\n%v\nerr: %w", sql, err)
			}
			defer rows.Close()
			for rows.Next() {
				err = rows.Scan(&name, &order, &parent, &id, &class, &geobytes)

				g, err := wkb.DecodeBytes(geobytes)
				if err != nil {
					var unknownGeo wkb.ErrUnknownGeometryType
					if errors.As(err, &unknownGeo) {
						panic(fmt.Sprintf("Unknown geo %v: %v", name, unknownGeo))
					}
					log.Printf("[entry %v] Name [%v], order [%v], class [%v], id [%v], parent [%v]:\nSQL:%v", i, name, order, class, id, parent, sql)
					return nil, fmt.Errorf("DecodeBytes: %w", err)
				}
				var mp geom.MultiPolygon
				// Need to make sure the g is of MultiPolygon
				switch gg := g.(type) {
				case geom.Polygon:
					mp = append(mp, gg)
				case geom.MultiPolygon:
					mp = gg
				default:
					return nil, fmt.Errorf("boundary Geom needs to be a polygon or multipolygon for %v", name)
				}
				lcountries = append(lcountries, &regionLayer{
					parent: parent,
					id:     id,
					name:   name,
					order:  strings.ToLower(order),
					class:  class,
					geo:    mp,
				})
			}

			sql = replaceTokens(bdry.Boundary, imap.totalExtent)
			rows, err = b.Query(ctx, sql)
			if err != nil {
				return nil, fmt.Errorf("SQL:\n%v\nerr: %w", sql, err)
			}
			defer rows.Close()
			for rows.Next() {
				err = rows.Scan(&namel, &namer, &order, &parent, &class, &geobytes)

				g, err := wkb.DecodeBytes(geobytes)
				if err != nil {
					var unknownGeo wkb.ErrUnknownGeometryType
					if errors.As(err, &unknownGeo) {
						panic(fmt.Sprintf("boundary lines Unknown geo %v: %v", name, unknownGeo))
					}
					return nil, fmt.Errorf("DecodeBytes: %w", err)
				}
				var ml geom.MultiLineString
				// Need to make sure the g is of MultiPolygon
				switch gg := g.(type) {
				case geom.LineString:
					ml = append(ml, gg)
				case geom.MultiLineString:
					ml = gg
				default:
					return nil, fmt.Errorf("boundary Geom needs to be a polygon or multipolygon for %v", name)
				}
				lboundaries = append(lboundaries, &boundaryLine{
					parent:   parent,
					nameL:    namel,
					nameR:    namer,
					order:    strings.ToLower(order),
					class:    class,
					boundary: ml,
				})
			}

		}
		// Now that we have all the countries and boundaries we need to build out the three structure
		// let's sort the regions first
		sort.Sort(byParentOrderRegionLayer(lcountries))
		for i := range lcountries {
			if _, ok := boundaryMapper[lcountries[i].id]; ok {
				panic("regin name dup")
			}
			if lcountries[i].parent == "" {
				// this is a top level region.
				boundaryMapper[lcountries[i].id] = []int{len(countries)}
				countries = append(countries, *lcountries[i])
				continue
			}
			parLoc, ok := boundaryMapper[lcountries[i].parent]
			if !ok {
				panic("parent should have been found by now")
			}
			// Only supporting country and first for now.
			if len(parLoc) == 1 {
				countries[parLoc[0]].subregions = append(countries[parLoc[0]].subregions, *lcountries[i])
			}
		}
		for i := range lboundaries {
			if lboundaries[i].parent == "" {
				// this is a country border
				boundaries = append(boundaries, *lboundaries[i])
				continue
			}
			parLoc, ok := boundaryMapper[lboundaries[i].parent]
			if !ok {
				log.Printf("mapper: %#v", boundaryMapper)
				log.Printf("lbound: %#v", lboundaries[i])
				panic("parent should have been found by now")
			}
			// Only supporting country and first for now.
			if len(parLoc) == 1 {
				countries[parLoc[0]].boundaries = append(countries[parLoc[0]].boundaries, *lboundaries[i])
			}
		}
	}
	return &BoundaryMap{
		countries:  countries,
		boundaries: boundaries,
		Map:        imap,
	}, nil
}

type BoundaryMap struct {
	countries  []regionLayer  // this should be countries in the boundary map
	boundaries []boundaryLine // this is the boundary lines between countries
	*Map                      // the inset map used to draw the inset map layers
}

type boundaryLine struct {
	parent   string
	nameL    string // NameL is the name on the left face
	nameR    string // NameR is the name on the right face
	class    string // svg class to assign to the path
	order    string
	boundary geom.MultiLineString // the boundary line
}

var ordermaptonum = map[string]uint8{
	// the default is 0, so we want zero to "highest" value
	"country": 254 - 0,
	"first":   254 - 1,
	"second":  254 - 2,
	"third":   254 - 3,
}

type byParentOrderRegionLayer []*regionLayer

func (by byParentOrderRegionLayer) Len() int      { return len(by) }
func (by byParentOrderRegionLayer) Swap(i, j int) { by[i], by[j] = by[j], by[i] }
func (by byParentOrderRegionLayer) Less(i, j int) bool {
	if by[i].parent == "" {
		return by[j].parent != ""
	}
	if by[j].parent == "" {
		return false
	}
	return (254 - ordermaptonum[by[i].order]) < (254 - ordermaptonum[by[j].order])
}

type regionLayer struct {
	parent string
	id     string
	// name of the region
	name string
	// the order of the region, "country" should be country, "first" is first order, "second" second order, etc...
	// "expunged" for expunged regions
	order string
	// geo describes the country geometry clipped to the map's extent
	geo geom.MultiPolygon // this should be the multipolygon that describes the country geometry clipped to the map's extent
	// class is the svg class to assign to the group
	class string
	// the lower order regions
	subregions []regionLayer
	boundaries []boundaryLine
}

func (rl regionLayer) DrawSubRegion(svgpath *SvgPath, svg *SVGStringBuilder, deltay string, label bool) error {
	if label {
		ext, err := geom.NewExtentFromGeometry(rl.geo)
		if err != nil {
			return err
		}
		x, y := svgpath.Point(
			((ext.MaxX()-ext.MinX())/2)+ext.MinX(),
			((ext.MaxY()-ext.MinY())/2)+ext.MinY(),
		)
		attributes := map[string]string{
			"x":           fmt.Sprintf("%vpx", x),
			"y":           fmt.Sprintf("%vpx", y),
			"text-anchor": "middle",
			"class":       rl.class,
		}
		err = svg.WriteTag("text", Attr(attributes, ""),
			func(svg *SVGStringBuilder) error {
				svg.WriteString(strings.ToUpper(rl.name))
				return nil
			},
		)
		if err != nil {
			return err
		}
	}

	for _, region := range rl.subregions {
		ext, err := geom.NewExtentFromGeometry(region.geo)
		if err != nil {
			return err
		}
		x, y := svgpath.Point(
			((ext.MaxX()-ext.MinX())/2)+ext.MinX(),
			((ext.MaxY()-ext.MinY())/2)+ext.MinY(),
		)
		attributes := map[string]string{
			"x":           fmt.Sprintf("%vpx", x),
			"y":           fmt.Sprintf("%vpx", y),
			"text-anchor": "middle",
			"class":       region.class,
		}
		if deltay != "" {
			attributes["dy"] = deltay
		}
		err = svg.WriteTag("text", Attr(attributes, ""),
			func(svg *SVGStringBuilder) error {
				svg.WriteString(region.name)
				return nil
			},
		)
		if err != nil {
			return err
		}
	}
	// draw all the boundary lines.
	for _, border := range rl.boundaries {
		path, err := svgpath.Path(border.boundary)
		if err != nil {
			return err
		}
		fmt.Fprintf(svg, `<path d="%s" class="%s" />`, path, border.class)
	}
	return nil
}

func (m *BoundaryMap) writeCountry(svgpath *SvgPath, svg *SVGStringBuilder) error {
	if len(m.countries) == 0 {
		// no countries weird, international waters?
		// draw nothing
		return nil
	}
	// if there is only one country we only need to output the subregion
	if len(m.countries) == 1 {
		// If there is only one sub region.
		// We need to only output it's name in the center of a white box.
		country := m.countries[0]
		switch len(country.subregions) {
		case 0:
			// no subregions draw nothing
			return nil

		case 1:
			return svg.WriteTag(
				"g",
				Attr(map[string]string{"id": "subregion1"}, ""),
				func(svg *SVGStringBuilder) error {
					// get the mid point of the the whole box.
					x, y := svgpath.Point(
						((m.totalExtent.MaxX()-m.totalExtent.MinX())/2)+m.totalExtent.MinX(),
						((m.totalExtent.MaxY()-m.totalExtent.MinY())/2)+m.totalExtent.MinY(),
					)
					return svg.WriteTag("text", Attr(map[string]string{
						"x":           fmt.Sprintf("%vpx", x),
						"y":           fmt.Sprintf("%vpx", y),
						"text-anchor": "middle",
						"class":       country.subregions[0].class,
					}, ""),
						func(svg *SVGStringBuilder) error {
							svg.WriteString(country.subregions[0].name)
							return nil
						},
					)
				},
			)

		default: // two or more regions
			return svg.WriteTag(
				"g",
				Attr(map[string]string{"id": "subregion1"}, ""),
				func(svg *SVGStringBuilder) error { return country.DrawSubRegion(svgpath, svg, "", false) },
			)

		}

	}

	// multiple countries we need to draw each countries label.
	return svg.WriteTag(
		"g",
		Attr(map[string]string{"id": "countries"}, ""),
		func(svg *SVGStringBuilder) error {
			for i, country := range m.countries {
				err := svg.WriteTag(
					"g",
					Attr(map[string]string{"id": fmt.Sprintf("country_%v", i)}, ""),
					func(svg *SVGStringBuilder) error { return country.DrawSubRegion(svgpath, svg, "2em", true) },
				)
				if err != nil {
					return err
				}
			}

			// draw all the boundary lines.
			for _, border := range m.boundaries {
				path, err := svgpath.Path(border.boundary)
				if err != nil {
					return err
				}
				fmt.Fprintf(svg, `<path d="%s" class="%s" />`, path, border.class)
			}
			return nil
		},
	)
}

// AsSVG will render the map as an SVG image
// If there is no countries then an empty square will be returned
func (m *BoundaryMap) AsSVG(attr string) (string, error) {
	var (
		svg     = new(SVGStringBuilder)
		svgpath = m.newSVGPath()
		err     error
	)

	svg.WriteTag("defs", "", func(svg *SVGStringBuilder) error {
		svg.WriteString(m.CSSTag())
		return nil
	})

	err = svg.WriteTag(
		"g", Attr(map[string]string{"id": "diagram"}, ""),
		func(svg *SVGStringBuilder) error {
			if err := m.buildLayers(svgpath, svg); err != nil {
				return err
			}

			if err = m.writeCountry(svgpath, svg); err != nil {
				return err
			}

			if err = m.writeCutLine(svgpath, svg); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return "", err
	}

	return SVGTag(
		"svg", Attr(map[string]string{
			"viewBox":     svgpath.ViewBox(),
			"xMidyMid":    "meet",
			"version":     "1.2",
			"baseProfile": "tiny",
			"xmlns":       "http://www.w3.org/2000/svg",
		}, attr),
		svg.String(),
	), nil

}
