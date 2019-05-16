package atlante

import (
	"context"
	"fmt"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/go-spatial/maptoolkit/atlante/filestore"
	fsfile "github.com/go-spatial/maptoolkit/atlante/filestore/file"
	fsmulti "github.com/go-spatial/maptoolkit/atlante/filestore/multi"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/atlante/internal/resolution"
	"github.com/go-spatial/maptoolkit/mbgl/bounds"
	"github.com/go-spatial/maptoolkit/mbgl/image"
	"github.com/go-spatial/maptoolkit/svg2pdf"
)

// ImgStruct is a wrapper around an image that makes the image available to the
// the template, and allows for the image to be encoded only if it's requested.
type ImgStruct struct {
	filename     string
	filestore    filestore.FileWriter
	intermediate bool
	// Have we generated the file?
	// This will allow us to generate the file only when requested
	generated bool
	lck       sync.Mutex
	image     *image.Image
}

// Height returns the height of the image
func (img ImgStruct) Height() int { return img.image.Bounds().Dy() }

// Width returns the width of the image
func (img ImgStruct) Width() int { return img.image.Bounds().Dx() }

// Filename returns the file name of the image
func (img ImgStruct) Filename() (string, error) {
	if img.generated {
		return img.filename, nil
	}
	// We need to generate the file and then return the filename
	return img.generateImage()
}

// generateIamge is use to create the image into the filestore
func (img *ImgStruct) generateImage() (fn string, err error) {
	if img.generated {
		return img.filename, nil
	}

	if img.image == nil {
		img.generated = true
		return img.filename, nil
	}

	if img.filestore == nil {
		img.generated = true
		return img.filename, nil
	}

	// generateIamge is use to create the image into the filestore
	img.lck.Lock()
	defer img.lck.Unlock()
	if img.generated {
		return img.filename, nil
	}

	file, err := img.filestore.Writer(img.filename, img.intermediate)
	if err != nil {
		return "", err
	}

	if file == nil {
		img.generated = true
		return img.filename, nil
	}
	defer file.Close()

	if err = img.image.GenerateImage(); err != nil {
		return "", err
	}
	if err := png.Encode(file, img.image); err != nil {
		return "", err
	}
	// Clean up the backing store.
	img.generated = true
	img.image.Close()
	return img.filename, nil
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
	// TODO(gdey): use MdgID once we move to partial templates system
	// grp := grid.MdgID.String(), an empty group is current directory
	grp := ""

	assetsWriter := fsfile.Writer{Base: grp, Intermediate: true}

	multiWriter := fsmulti.FileWriter{
		Writers: []filestore.FileWriter{assetsWriter},
	}

	if sheet.Filestore != nil {
		shWriter, err := sheet.Filestore.FileWriter(grp)
		if err != nil {
			return fmt.Errorf("failed to create sheed filestore writer: %v", err)
		}
		if shWriter != nil {
			multiWriter.Writers = append(multiWriter.Writers, shWriter)
		}
	}

	log.Println("filenames: ", filenames.IMG, filenames.SVG, filenames.PDF)

	const tilesize = 4096 / 2

	/*
		TODO(gdey): Keeping this for now. Not sure if we need this, or if the
					zoom calculation taking dpi into considertion is all that's
					needed
		// To calculate ppi_ratio we use 96 as the default ppi.
		ppiRatio := float64(sheet.DPI) / 96.0
	*/

	zoom := grid.ZoomForScaleDPI(sheet.Scale, sheet.DPI)

	nground := resolution.Ground(
		resolution.MercatorEarthCircumference,
		zoom,
		grid.SWLat,
	)

	// Generate the PNG
	prj := bounds.ESPG3857
	width, height := grid.WidthHeightForZoom(zoom)
	latLngCenterPt := grid.CenterPtForZoom(zoom)
	log.Println("width", width, "height", height)
	log.Println("zoom", zoom, "Scale", sheet.Scale, "dpi", sheet.DPI, "ground measure", nground)

	centerPt := bounds.LatLngToPoint(prj, latLngCenterPt[0], latLngCenterPt[1], zoom, tilesize)
	dstimg, err := image.New(
		prj,
		int(width), int(height),
		centerPt,
		zoom,
		// TODO(gdey): Need to remove this hack and figure out how to used the
		// ppi value as well as set the correct scale on the svg/pdf document
		// that is produced later on. (https://github.com/go-spatial/maptoolkit/issues/13)
		1.0, // ppiRatio, (we adjust the zoom)
		0.0, // Bearing
		0.0, // Pitch
		sheet.Style,
		"", "",
	)
	if err != nil {
		return err
	}

	file, err := multiWriter.Writer(filenames.SVG, true)
	defer file.Close()

	img := ImgStruct{
		filename:     filenames.IMG,
		image:        dstimg,
		filestore:    multiWriter,
		intermediate: true,
	}

	// Fill out template
	err = sheet.Execute(file, GridTemplateContext{
		Image:         img,
		DPI:           sheet.DPI,
		Scale:         sheet.Scale,
		Zoom:          zoom,
		GroundMeasure: nground,
		Grid:          *grid,
	})
	if err != nil {
		log.Printf("error trying to fillout sheet template")
		return err
	}

	//TODO(gdey): here we should change directories to the working directory.
	// This is needed to generate the PDF. It might make sense to do this
	// as the first thing we do when entering this function. Atlante object
	// does have a working directory parameter for this.

	//TODO(gdey): 2028 and 2607 are file sizes for the size of the
	// page we are generating. This should really be configuration
	// options in the sheet. Using standard name like A0.
	// see: https://www.belightsoft.com/products/resources/paper-sizes-and-formats-explained
	// Though the 2028 and 2606 don't quite match up.

	pdffn := assetsWriter.Path(filenames.PDF)
	svgfn := assetsWriter.Path(filenames.SVG)

	if err = svg2pdf.GeneratePDF(svgfn, pdffn, 2028, 2607); err != nil {
		return err
	}

	if len(multiWriter.Writers) > 1 {
		// Don't want the assets writer
		wrts, err := multiWriter.Writers[1].Writer(filenames.PDF, false)
		if err != nil {
			return err
		}
		// nil writer move on.
		if wrts == nil {
			return nil
		}
		defer wrts.Close()
		// Copy the pdf over
		pdffile, err := os.Open(pdffn)
		if err != nil {
			return err
		}
		io.Copy(wrts, pdffile)
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
