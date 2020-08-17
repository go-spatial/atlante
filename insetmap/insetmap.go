package insetmap

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/encoding/wkb"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	bboxToken = "!BBOX!"
)

var (
	tokenRe = regexp.MustCompile("![a-zA-Z0-9_-]+!")
)

// Inset retrives data to populate a Map that can be used to
// generate the SVG
type Inset struct {
	Main       string
	Adjoining  string
	Scale      float64
	Buff       int64
	Layers     []Layer
	CSSMap     CSSMap
	CSSDir     string
	CSSDefault string

	*pgxpool.Pool
}

func New(db *pgxpool.Pool, config Config, gCSSDir string, gCSSMap CSSMap, gCSSDefault string) (*Inset, error) {
	scale := float64(config.Scale)
	if scale <= 0 {
		scale = 1
	}

	var eCSSMap = gCSSMap
	// Get all the css style sheets

	eCSSDir := string(config.CSSDir)
	if debug {
		log.Printf("[DEBUG] eCSSDir: '%v' -- %v\n", eCSSDir, gCSSDir)
	}
	if eCSSDir != "" && eCSSDir != gCSSDir {
		eCSSMap = make(CSSMap)
		if err := eCSSMap.GetStyleSheets(eCSSDir); err != nil {
			return nil, err
		}
	} else {
		eCSSDir = gCSSDir
	}
	eCSSDefault := string(config.CSSDefault)
	if eCSSDefault == "" {
		eCSSDefault = gCSSDefault
	}

	var layers []Layer
	for i := range config.Layers {
		layers = append(layers, Layer{
			SQL:  string(config.Layers[i].SQL),
			Name: string(config.Layers[i].Name),
		})

	}
	if debug {
		log.Printf("[DEBUG-insetmap] using buff to: %v", int64(config.ViewBuffer))
	}
	return &Inset{
		Main:       string(config.Sheet),
		Adjoining:  string(config.Adjoining),
		Scale:      float64(config.Scale),
		Buff:       int64(config.ViewBuffer),
		Layers:     layers,
		CSSMap:     eCSSMap,
		CSSDir:     eCSSDir,
		CSSDefault: eCSSDefault,

		Pool: db,
	}, nil

}

func (inset *Inset) initLayers(ctx context.Context, imap *Map, ext *geom.Extent) error {
	for _, lyr := range inset.Layers {
		var (
			geobytes []byte
			class    string
		)

		sql := replaceTokens(lyr.SQL, ext)
		var mlayer mapLayer
		if debug {
			log.Println("[DEBUG] running sql: ", sql)
		}
		rows, err := inset.Query(ctx, sql)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&class, &geobytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERR] geom failed %v: %v\n", lyr.Name, err)
				return err
			}
			g, err := wkb.DecodeBytes(geobytes)
			if err != nil {
				var unknownGeo wkb.ErrUnknownGeometryType
				if errors.As(err, &unknownGeo) {
					panic(fmt.Sprintf("Unknown geo %v: %v", lyr.Name, unknownGeo))
				}
				return err
			}
			mlayer.Geometries = append(mlayer.Geometries, g)
		}
		mlayer.Class = class
		mlayer.Name = lyr.Name
		imap.layers = append(imap.layers, mlayer)
	}
	return nil
}

func (inset *Inset) initBaseVars(ctx context.Context, imap *Map, mdgid string, cssKey string) error {
	var (
		err error
	)
	if debug {
		log.Printf("[DEBUG-inset] using buff to: %v", inset.Buff)
	}
	imap.scale = inset.Scale
	imap.buff = inset.Buff

	if cssKey == "" {
		cssKey = inset.CSSDefault
	}
	if cssKey != "" {
		cssFile := inset.CSSMap[cssKey].Path
		if cssFile != "" {
			// don't care about the error
			contents, _ := ioutil.ReadFile(cssFile)
			imap.css = string(contents)
		}
	}

	// Get main sheet
	if debug {
		log.Printf("[DEBUG] working on mdgid: %#v", mdgid)
	}
	row := inset.QueryRow(ctx, inset.Main, mdgid)
	imap.main, err = sheetForRow(row)
	if err != nil {
		return err
	}
	imap.totalExtent = imap.main.Extent.Clone()
	return nil
}

// For retrieves the data for the given mdgid to generate a Map, that can
// be rendered as an SVG
func (inset *Inset) For(ctx context.Context, mdgid string, cssKey string) (*Map, error) {
	var (
		insetmap Map
		err      error
	)
	if inset == nil {
		return nil, nil
	}

	if err = inset.initBaseVars(ctx, &insetmap, mdgid, cssKey); err != nil {
		return nil, err
	}

	{
		deltax := (insetmap.main.Extent.MaxX() - insetmap.main.Extent.MinX()) / 2
		deltay := (insetmap.main.Extent.MaxY() - insetmap.main.Extent.MinY()) / 2
		// Make sure our total bounds is at minimum 9x9
		insetmap.totalExtent[0] -= deltax
		insetmap.totalExtent[1] -= deltay
		insetmap.totalExtent[2] += deltax
		insetmap.totalExtent[3] += deltay
	}
	{
		sql := replaceTokens(inset.Adjoining, insetmap.totalExtent)
		rows, err := inset.Query(ctx, sql, mdgid)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			sheet, err := sheetForRow(rows)
			if err != nil {
				return nil, err
			}
			insetmap.adjoining = append(insetmap.adjoining, sheet)
			insetmap.totalExtent.Add(sheet.Extent)
		}
	}
	if err = inset.initLayers(ctx, &insetmap, insetmap.totalExtent); err != nil {
		return nil, err
	}
	return &insetmap, nil

}

//	uppercaseTokens converts all !tokens! to uppercase !TOKENS!. Tokens can
//	contain alphanumerics, dash and underline chars.
func uppercaseTokens(str string) string {
	return tokenRe.ReplaceAllStringFunc(str, strings.ToUpper)
}

func replaceTokens(sql string, bbox *geom.Extent) string {
	envel := envelope(bbox)

	tokenReplacer := strings.NewReplacer(
		bboxToken, envel,
	)
	uppercaseTokenSQL := uppercaseTokens(sql)
	return tokenReplacer.Replace(uppercaseTokenSQL)
}

type Layer struct {
	SQL  string
	Name string
}

// Map has contains all the information to render a inset map
type Map struct {
	main        mapSheet
	adjoining   []mapSheet
	totalExtent *geom.Extent
	scale       float64
	buff        int64
	layers      []mapLayer
	css         string // css to embed into the svg
}

func (m *Map) CSSTag() string {
	if m.css == "" {
		return ""
	}
	return fmt.Sprintf("<style>\n%s\n</style>", m.css)
}

func (m *Map) newSVGPath() *SvgPath {
	if debug {
		log.Printf("[DEBUG-map] using buff to: %v", m.buff)
	}
	return NewSVGPath(m.totalExtent, m.scale, m.buff)
}

func (m *Map) buildLayers(svgpath *SvgPath, svg *SVGStringBuilder) error {
	for i := range m.layers {
		str, err := m.layers[i].AsSVG(svgpath)
		if err != nil {
			return err
		}
		svg.WriteString(str)
	}
	return nil
}

func (m *Map) writeCutLine(svgpath *SvgPath, svg *SVGStringBuilder) error {
	str, err := mapSheet{Extent: m.totalExtent, Class: "cutline"}.AsSVG("cutline-border", svgpath)
	if err != nil {
		return err
	}
	svg.WriteString(str)
	return nil
}

// AsSVG will render the map as an SVG image
// the layers are rendered first,
// then the adjoining sheets' boxes and texts
// then the main sheet's box and text
// then the cutline surrounding the image
func (m *Map) AsSVG(partial bool, attr string) (string, error) {
	var svg = new(SVGStringBuilder)

	svgpath := m.newSVGPath()

	if m.css != "" {

		svg.WriteTag("defs", "", func(s *SVGStringBuilder) error {
			s.WriteString(m.CSSTag())
			return nil
		})
	}

	err := svg.WriteTag(
		"g", Attr(map[string]string{"id": "diagram"}, ""),
		func(svg *SVGStringBuilder) error {
			if err := m.buildLayers(svgpath, svg); err != nil {
				return err
			}

			for i := range m.adjoining {
				str, err := m.adjoining[i].AsSVG(fmt.Sprintf("adjoining_%d", i), svgpath)
				if err != nil {
					return err
				}
				svg.WriteString(str)
			}

			str, err := m.main.AsSVG("main", svgpath)
			if err != nil {
				return err
			}
			svg.WriteString(str)

			if err = m.writeCutLine(svgpath, svg); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return "", err
	}

	if partial {
		return svg.String(), nil
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

// mapLayer contains information to render a map layer as svg
type mapLayer struct {
	Class      string
	Name       string
	Geometries []geom.Geometry
}

func (lyr mapLayer) AsSVG(svgpath *SvgPath) (string, error) {
	var (
		svg   strings.Builder
		class = lyr.Class
	)
	if class != "" {
		class = `class="` + class + `"`
	}
	name := lyr.Name
	fmt.Fprintf(&svg, `<g id="%s" %s>`, name, class)
	for _, g := range lyr.Geometries {
		svg.WriteString("\n")
		path, err := svgpath.Path(g)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&svg, `<path d="%s" %s />`, path, class)
	}

	svg.WriteString("\n</g>\n")
	return svg.String(), nil
}

type mapSheet struct {
	Name   string
	Class  string
	Extent *geom.Extent
}

func (sheet mapSheet) AsSVG(id string, svgpath *SvgPath) (string, error) {
	var (
		svg   strings.Builder
		class = sheet.Class
	)
	if class != "" {
		class = `class="` + class + `"`
	}

	fmt.Fprintf(&svg, `<g id="%s" %s>`, id, class)
	svg.WriteString("\n")
	g := sheet.Extent.AsPolygon()

	path, err := svgpath.Path(g)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&svg, `<path d="%s" fill="none" stroke="black" %s/>`, path, class)

	if sheet.Name != "" {
		x, y := svgpath.Point(
			(sheet.Extent.MinX() + (sheet.Extent.XSpan() / 2)),
			(sheet.Extent.MinY() + (sheet.Extent.YSpan() / 2)),
		)
		svg.WriteString("\n")
		fmt.Fprintf(&svg, `<text x="%d" y="%d" text-anchor="middle" %s>%s-%s</text>`,
			int64(x), int64(y),
			class,
			sheet.Name[:len(sheet.Name)-1],
			sheet.Name[len(sheet.Name)-1:],
		)
	}
	svg.WriteString("\n</g>\n")
	return svg.String(), nil
}

// envelope creates an postgis Envelope for the given extent
func envelope(ext *geom.Extent) string {
	return fmt.Sprintf("ST_MakeEnvelope(%g,%g,%g,%g,%d)", ext.MinX(), ext.MinY(), ext.MaxX(), ext.MaxY(), 4326)
}

// sheetForRow will extrat the sheet form the given row object.
// expects the following columns in order sheet, class, wkb bytes
func sheetForRow(scanner interface{ Scan(...interface{}) error }) (sheet mapSheet, err error) {
	var (
		geobytes []byte
	)
	err = scanner.Scan(&sheet.Name, &sheet.Class, &geobytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "geom extent failed: %v\n", err)
		return sheet, err
	}
	g, err := wkb.DecodeBytes(geobytes)
	if err != nil {
		var unknownGeo wkb.ErrUnknownGeometryType
		if errors.As(err, &unknownGeo) {
			panic(fmt.Sprintf("Unknown geo: %v", unknownGeo))
		}
		return sheet, err
	}

	sheet.Extent, err = geom.NewExtentFromGeometry(g)
	return sheet, err
}
