package atlante

import (
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/atlante/internal/resolution"
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
	Image         ImgStruct
	GroundMeasure float64
	Grid          grids.Grid
	DPI           uint
	Scale         uint
	Zoom          float64
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
		return ErrNilGrid
	}

	log.Println("filenames: ", filenames.IMG, filenames.SVG, filenames.PDF)

	const tilesize = 4096 / 2

	// To calculate ppi_ratio we use 96 as the default ppi.
	ppiRatio := float64(sheet.DPI) / 96.0
	_ = ppiRatio
	log.Println("PPI:", ppiRatio)

	zoom := grid.ZoomForScaleDPI(sheet.Scale, sheet.DPI)
	// zoom := sheet.Zoom

	log.Println("zoom", zoom, "Scale", sheet.Scale, "dpi", sheet.DPI)
	nground := resolution.Ground(
		resolution.MercatorEarthCircumference,
		zoom,
		grid.SWLat,
	)

	//nground = math.RoundToEven(nground)
	log.Println("zoom", zoom, "ground measure", nground)

	width, height := grid.WidthHeightForZoom(zoom)
	log.Println("z, width", width, "z, height", height)
	zoom = resolution.ZoomForGround(
		resolution.MercatorEarthCircumference,
		nground,
		grid.SWLat,
	)
	log.Println("zoom", zoom)

	// Generate the PNG
	prj := bounds.ESPG3857
	latLngCenterPt := grid.CenterPtForZoom(zoom)
	width, height = grid.WidthHeightForZoom(zoom)
	log.Println("width", width, "height", height)
	centerPt := bounds.LatLngToPoint(prj, latLngCenterPt[0], latLngCenterPt[1], zoom, tilesize)
	dstimg, err := image.New(
		prj,
		int(width), int(height),
		centerPt,
		zoom,
		ppiRatio, //1.0, //ppiRatio,
		0.0,      // Bearing
		0.0,      // Pitch
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
		DPI:           sheet.DPI,
		Scale:         sheet.Scale,
		Zoom:          zoom,
		GroundMeasure: nground,
		Grid:          *grid,
	})
	if err != nil {
		log.Printf("Got an error trying to fillout sheet template")
		file.Close()
		return err
	}
	_ = file.Close()

	//TODO(gdey): here we should change directories to the working directory.
	// This is needed to generate the PDF. It might make sense to do this
	// as the first thing we do when entering this function. Atlante object
	// does have a working directory parameter for this.

	// svg2pdf
	//err = svg2pdf.GeneratePDF(filenames.SVG, filenames.PDF, 2500, 3000)
	//	err = svg2pdf.GeneratePDF(filenames.SVG, filenames.PDF, 10419, 12501)
	err = svg2pdf.GeneratePDF(filenames.SVG, filenames.PDF, 2028, 2607)
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
		return nil, ErrNilAtlanteObject
	}

	if len(a.sheets) == 0 {
		return nil, ErrNoSheets
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

func (a *Atlante) generatePDF(ctx context.Context, sheet *Sheet, grid *grids.Grid, filenameTemplate string) (*GeneratedFiles, error) {
	filenames, err := a.filenamesForGrid(sheet.Name, grid, filenameTemplate)
	if err != nil {
		return nil, err
	}
	err = GeneratePDF(ctx, sheet, grid, filenames)
	return filenames, err
}

func (a *Atlante) GeneratePDFJob(ctx context.Context, job Job, filenameTemplate string) (*GeneratedFiles, error) {
	grid := job.GridsGrid()
	sheet, err := a.SheetFor(job.SheetName)
	if err != nil {
		return nil, err
	}
	return a.generatePDF(ctx, sheet, grid, filenameTemplate)
}

func (a *Atlante) GeneratePDFMDGID(ctx context.Context, sheetName string, mdgID grids.MDGID, filenameTemplate string) (*GeneratedFiles, error) {

	sheet, err := a.SheetFor(sheetName)
	if err != nil {
		return nil, err
	}

	grid, err := sheet.GridForMDGID(mdgID)
	if err != nil {
		return nil, err
	}
	return a.generatePDF(ctx, sheet, grid, filenameTemplate)
}
