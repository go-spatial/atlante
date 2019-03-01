package atlante

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/atlante/internal/urlutil"
	"github.com/go-spatial/maptoolkit/mbgl/bounds"
	"github.com/go-spatial/maptoolkit/mbgl/image"
	"github.com/go-spatial/maptoolkit/svg2pdf"
)

type ImgStruct struct {
	Filename string
	Width    int
	Height   int
}

func (img ImgStruct) ImageTagBase64() string {

	return fmt.Sprintf(
		`<image width="%d" height="%d" xlink:href="data:%s;base64,%s" />`,
		img.Width,
		img.Height,
		img.MimeType(),
		img.Base64Image(),
	)
}

func (img ImgStruct) MimeType() string { return "image/png" }

func (img ImgStruct) Base64Image() string {
	filebytes, err := ioutil.ReadFile(img.Filename)
	if err != nil {
		log.Printf("got error reading file %v: %v\n", img.Filename, err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(filebytes)
}

type GridTemplateContext struct {
	Image ImgStruct
	Grid  grids.Grid
}

type Sheet struct {
	Name string
	grids.Provider
	Zoom                float64
	Style               string
	SvgTemplateFilename string

	svgTemplate *template.Template
}

var funcMap = template.FuncMap{
	"ToUpper": strings.ToUpper,
	"ToLower": strings.ToLower,
	"Format": func(format string, data interface{}) string {
		// Allow Format to be used for time's as well.
		if d, ok := data.(time.Time); ok {
			return d.Format(format)
		}
		return fmt.Sprintf(format, data)
	},
	"Now": time.Now,
}

func NewSheet(name string, provider grids.Provider, zoom float64, style string, svgTemplateFilename *url.URL) (*Sheet, error) {
	var (
		err error
		t   *template.Template
	)

	tpl, err := urlutil.ReadAll(svgTemplateFilename)
	if err != nil {
		return nil, err
	}

	t, err = template.New(svgTemplateFilename.String()).
		Funcs(funcMap).
		Option("missingkey=error").
		Parse(string(tpl))
	if err != nil {
		return nil, err
	}

	return &Sheet{
		Name:                name,
		Provider:            provider,
		Zoom:                zoom,
		Style:               style,
		SvgTemplateFilename: svgTemplateFilename.String(),
		svgTemplate:         t,
	}, nil
}

func (sheet *Sheet) Execute(wr io.Writer, tplContext GridTemplateContext) error {
	return sheet.svgTemplate.Execute(wr, tplContext)
}

type GeneratedFiles struct {
	IMG string
	SVG string
	PDF string
}

func NewGeneratedFilesFromTpl(fnTemplate *filenameTemplate, sheetName string, grid grids.Grid, wd string) *GeneratedFiles {

	fn := func(ext string) string {
		dirtyFilename := fnTemplate.Filename(sheetName, grid, wd, ext)
		return filepath.Clean(dirtyFilename)
	}

	return &GeneratedFiles{
		IMG: fn("png"),
		SVG: fn("svg"),
		PDF: fn("pdf"),
	}
}

type FilenameTemplateContext struct {
	Grid          grids.Grid
	Ext           string
	SheetName     string
	WorkDirectory string
}

type filenameTemplate struct {
	t *template.Template
}

func NewFilenameTemplate(fnTemplate string) (*filenameTemplate, error) {
	var (
		ft  filenameTemplate
		err error
	)
	ft.t, err = template.New("filename").Option("missingkey=zero").Parse(fnTemplate)
	return &ft, err
}

const DefaultFilenameTemplate = "{{.SheetName}}_{{.Grid.ReferenceNumber}}.{{.Ext}}"

func (ft filenameTemplate) Filename(sheetName string, grid grids.Grid, wd string, ext string) string {
	var str strings.Builder
	err := ft.t.Execute(&str, FilenameTemplateContext{
		Grid:          grid,
		SheetName:     sheetName,
		WorkDirectory: wd,
		Ext:           ext,
	})
	if err != nil {
		panic(err)
	}
	return str.String()
}

func GeneratePDF(ctx context.Context, sheet *Sheet, grid *grids.Grid, filenames *GeneratedFiles) error {
	if grid == nil {
		return errors.New("grid is nil")
	}

	log.Println("filenames: ", filenames.IMG, filenames.SVG, filenames.PDF)

	const tilesize = 4096 / 2

	// Generate the PNG
	prj := bounds.ESPG3857
	latLngCenterPt := grid.CenterPtForZoom(sheet.Zoom)
	width, height := grid.WidthHeightForZoom(sheet.Zoom)
	centerPt := bounds.LatLngToPoint(prj, latLngCenterPt[0], latLngCenterPt[1], sheet.Zoom, tilesize)
	dstimg, err := image.New(
		prj,
		int(width), int(height),
		centerPt,
		sheet.Zoom,
		1.0,
		0.0,
		0.0,
		sheet.Style,
		"", "",
	)
	if err != nil {
		return err
	}
	file, err := os.Create(filenames.IMG)
	if err != nil {
		return err
	}
	if err := png.Encode(file, dstimg); err != nil {
		file.Close()
		return err
	}
	_ = file.Close()
	imgBounds := dstimg.Bounds()

	file, err = os.Create(filenames.SVG)
	if err != nil {
		return err
	}

	// Fill out template
	err = sheet.Execute(file, GridTemplateContext{
		Image: ImgStruct{
			Filename: filenames.IMG,
			Width:    imgBounds.Dx(),
			Height:   imgBounds.Dy(),
		},
		Grid: *grid,
	})
	if err != nil {
		log.Printf("Got an error trying to fillout sheet template")
		file.Close()
		return err
	}
	_ = file.Close()
	// svg2pdf
	err = svg2pdf.GeneratePDF(filenames.SVG, filenames.PDF, 2500, 3000)
	if err != nil {
		panic(err)
	}
	return err
}

type Atlante struct {
	workDirectory string
	sLock         sync.RWMutex
	sheets        map[string]*Sheet
}

func (a *Atlante) Shutdown() {}
func (a *Atlante) AddSheet(s *Sheet) error {
	if s == nil {
		return errors.New("nil sheet")
	}
	name := strings.TrimSpace(s.Name)
	if name == "" {
		return errors.New("blank sheet name")
	}
	a.sLock.Lock()
	defer a.sLock.Unlock()
	if a.sheets == nil {
		a.sheets = make(map[string]*Sheet)
		a.sheets[name] = s
		return nil
	}
	if _, ok := a.sheets[name]; ok {
		return errors.New("duplicate sheet name")
	}
	a.sheets[name] = s
	return nil
}

func (a *Atlante) filenamesForGrid(sheetName string, grid *grids.Grid, fname string) (*GeneratedFiles, error) {
	if fname == "" {
		fname = DefaultFilenameTemplate
	}

	filenameGenerator, err := NewFilenameTemplate(fname)
	if err != nil {
		return nil, err
	}

	return NewGeneratedFilesFromTpl(filenameGenerator, sheetName, *grid, a.workDirectory), nil
}

func (a *Atlante) GeneratePDFLatLng(ctx context.Context, sheetName string, lat, lng float64, srid uint64, filenameTemplate string) (*GeneratedFiles, error) {
	if a == nil {
		return nil, errors.New("atlante object is nil")
	}
	if len(a.sheets) == 0 {
		return nil, errors.New("atlante no sheets configured")
	}

	a.sLock.RLock()
	provider, ok := a.sheets[sheetName]
	if !ok || provider == nil {
		a.sLock.Unlock()
		return nil, fmt.Errorf("atlante sheet (%v) not found", sheetName)
	}
	a.sLock.Unlock()

	grid, err := provider.GridForLatLng(lat, lng, uint(srid))
	if err != nil {
		return nil, err
	}

	filenames, err := a.filenamesForGrid(sheetName, grid, filenameTemplate)
	if err != nil {
		return nil, err
	}

	err = GeneratePDF(ctx, provider, grid, filenames)
	return filenames, err

}

func (a *Atlante) GeneratePDFMDGID(ctx context.Context, sheetName string, mdgID grids.MDGID, filenameTemplate string) (*GeneratedFiles, error) {
	if a == nil {
		return nil, errors.New("atlante object is nil")
	}
	if len(a.sheets) == 0 {
		return nil, errors.New("atlante no sheets configured")
	}

	a.sLock.RLock()
	provider, ok := a.sheets[sheetName]
	if !ok || provider == nil {
		a.sLock.RUnlock()
		return nil, fmt.Errorf("atlante sheet (%v) not found", sheetName)
	}
	a.sLock.RUnlock()

	grid, err := provider.GridForMDGID(mdgID)
	if err != nil {
		return nil, err
	}

	filenames, err := a.filenamesForGrid(sheetName, grid, filenameTemplate)
	if err != nil {
		return nil, err
	}

	err = GeneratePDF(ctx, provider, grid, filenames)
	return filenames, err
}
