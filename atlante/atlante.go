package atlante

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/planar/coord"
	"github.com/go-spatial/maptoolkit/atlante/template/trellis"

	"github.com/go-spatial/maptoolkit/atlante/filestore"
	fsfile "github.com/go-spatial/maptoolkit/atlante/filestore/file"
	fsmulti "github.com/go-spatial/maptoolkit/atlante/filestore/multi"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/atlante/notifiers"
	_ "github.com/go-spatial/maptoolkit/atlante/notifiers/http"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"
	"github.com/go-spatial/maptoolkit/mbgl/bounds"
	"github.com/go-spatial/maptoolkit/svg2pdf"
	"github.com/prometheus/common/log"
)

type GridTemplateContext struct {
	Image  *Img
	Grid   *grids.Cell
	Width  float64
	Height float64
}

func (grctx *GridTemplateContext) SetWidthHeight(w float64, h float64) string {
	grctx.Width = w
	grctx.Height = h
	log.Infof("Setting page size to %v x %v",w,h)
	return ""
}

// SetImageDimension will set the image's desired Dimensions
// If Bounds is not nil, this will force the scale and GroundMeasure to be recalculated for the image
func (grctx GridTemplateContext) SetImageDimension(width, height float64) string {
	grctx.Image.SetWidthHeight(width, height)
	return ""
}

func (grctx GridTemplateContext) GroundMeasure() float64 {
	gm := grctx.Image.GroundMeasure()
	log.Infof("ground measure is: %v", gm)
	return gm
}
func (grctx GridTemplateContext) Zoom() float64 { return grctx.Image.Zoom() }
func (grctx GridTemplateContext) Scale() uint   { return grctx.Image.Scale }
func (grctx GridTemplateContext) DPI() uint {
	dpi := grctx.Image.DPI
	log.Infof("dpi is: %v", dpi)
	return dpi
}

func (grctx GridTemplateContext) DrawBars(gridSize int, pxlBox PixelBox, lblRows, lblCols []int, lblMeterOffset int) (string, error) {

	log.Infof("Draw Bars called: ground measure: %v", pxlBox.GroundPixel)
	sw := grctx.Grid.SW()
	ne := grctx.Grid.NE()

	return TplDrawBars(
		coord.LngLat{
			Lng: sw[0],
			Lat: sw[1],
		},
		coord.LngLat{
			Lng: ne[0],
			Lat: ne[1],
		},
		pxlBox,
		trellis.Grid(gridSize),
		lblRows,
		lblCols,
		lblMeterOffset,
		true,
	)
}

func (grctx GridTemplateContext) DrawOnlyLabels(gridSize int, pxlBox PixelBox, lblRows, lblCols []int, lblMeterOffset int) (string, error) {

	log.Infof("Draw only labels called: ground measure: %v", pxlBox.GroundPixel)
	sw := grctx.Grid.SW()
	ne := grctx.Grid.NE()

	return TplDrawBars(
		coord.LngLat{
			Lng: sw[0],
			Lat: sw[1],
		},
		coord.LngLat{
			Lng: ne[0],
			Lat: ne[1],
		},
		pxlBox,
		trellis.Grid(gridSize),
		lblRows,
		lblCols,
		lblMeterOffset,
		false,
	)

}

type GeneratedFiles struct {
	IMG string
	SVG string
	PDF string
}

// NewGeneratedFilesFromTpl will generate the three filesnames we need based on a filename template
func NewGeneratedFilesFromTpl(fnTemplate *filenameTemplate, sheetName string, grid *grids.Cell, wd string) *GeneratedFiles {

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
	Grid          *grids.Cell
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

// Filenames geneate the various filename for the different types we need
func (ft filenameTemplate) Filename(sheetName string, grid *grids.Cell, wd string, ext string) string {
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

// GeneratePDF will generate the PDF based on the sheet, and grid
func GeneratePDF(ctx context.Context, sheet *Sheet, grid *grids.Cell, filenames *GeneratedFiles) error {
	if grid == nil {
		return ErrNilGrid
	}

	useCached := false
	if val, ok := os.LookupEnv("ATLANTE_USED_CACHED_IMAGES"); ok {
		useCached, _ = strconv.ParseBool(val)
		log.Infof("ATLANTE_USED_CACHED_IMAGES=%t", useCached)
	}

	sheet.Emit(field.Started{})

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
			err = fmt.Errorf("failed to create sheet filestore writer: %v", err)
			sheet.EmitError("internal error", err)
			return err
		}
		if shWriter != nil {
			multiWriter.Writers = append(multiWriter.Writers, shWriter)
		}
	}

	sheet.FuncFilestoreWriter = multiWriter
	sheet.UseCached = useCached

	log.Infoln("filenames: ", filenames.IMG, filenames.SVG, filenames.PDF)

	/*
		TODO(gdey): Keeping this for now. Not sure if we need this, or if the
					zoom calculation taking dpi into considertion is all that's
					needed
		// To calculate ppi_ratio we use 96 as the default ppi.
		ppiRatio := float64(sheet.DPI) / 96.0
	*/

	img := Img{
		File: &filestore.File{
			Store:          multiWriter,
			Name:           filenames.IMG,
			IsIntermediate: true,
			UseCached:      useCached,
		},
		StartGenerationCallback: func() {
			sheet.Emit(field.Processing{
				Description: fmt.Sprintf("intermediate file: %v", filenames.IMG),
			})
		},
		FailGenerationCallback: func(err error) {
			sheet.EmitError(
				fmt.Sprintf("failed to generate intermediate file: %v", filenames.IMG),
				err,
			)
		},
		DPI:        sheet.DPI,
		Grid:       grid,
		Projection: bounds.ESPG3857,
		Scale:      sheet.Scale,
		Style:      sheet.Style,
	}

	defer func() {
		img.Close()
	}()

	sheet.Emit(field.Processing{
		Description: fmt.Sprintf("intermediate file: %v ", filenames.SVG),
	})
	file, err := multiWriter.Writer(filenames.SVG, true)
	defer file.Close()
	if err != nil {
		sheet.EmitError("failed to copy files", err)
		return err
	}
	if ctx.Err() != nil {
		sheet.EmitError("generate pdf canceled", ctx.Err())
		return ctx.Err()
	}

	widthpts, heightpts := float64(sheet.WidthInPoints(72)), float64(sheet.HeightInPoints(72))
	gtc := &GridTemplateContext{
		Image:  &img,
		Grid:   grid,
		Width:  widthpts,
		Height: heightpts,
	}
	// Fill out template
	err = sheet.Execute(file,gtc)
	if err != nil {
		sheet.EmitError("template processing failure", err)
		log.Warnf("error trying to fillout sheet template")
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

	sheet.Emit(field.Processing{
		Description: fmt.Sprintf("generate file: %v ", filenames.PDF),
	})

	log.Infof("pdf %v,%v", gtc.Width, gtc.Height)
	if err = svg2pdf.GeneratePDF(svgfn, pdffn, gtc.Width, gtc.Height); err != nil {
		log.Warnf("error generating pdf: %v", err)
		sheet.EmitError("generate pdf failed", err)
		return err
	}
	if ctx.Err() != nil {
		sheet.EmitError("generate pdf canceled", ctx.Err())
		return ctx.Err()
	}

	if len(multiWriter.Writers) > 1 {
		// Don't want the assets writer
		wrts, err := multiWriter.Writers[1].Writer(filenames.PDF, false)
		if err != nil {
			sheet.EmitError("generate pdf failed", err)
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
			sheet.EmitError("generate pdf failed", err)
			return err
		}
		io.Copy(wrts, pdffile)
	}
	sheet.Emit(field.Completed{})
	return nil
}

type Atlante struct {
	workDirectory string
	sLock         sync.RWMutex
	sheets        map[string]*Sheet
	Notifier      notifiers.Provider
	JobID         string
}

func (a *Atlante) Shutdown() {}

func (a *Atlante) filenamesForCell(sheetName string, cell *grids.Cell, fname string) (*GeneratedFiles, error) {
	if fname == "" {
		fname = DefaultFilenameTemplate
	}

	filenameGenerator, err := NewFilenameTemplate(fname)
	if err != nil {
		return nil, err
	}
	return NewGeneratedFilesFromTpl(filenameGenerator, sheetName, cell, a.workDirectory), nil
}

func (a *Atlante) FilenamesForCell(sheetName string, cell *grids.Cell) *GeneratedFiles {
	// This will not generate an error
	filenameGenerator, _ := NewFilenameTemplate(DefaultFilenameTemplate)
	return NewGeneratedFilesFromTpl(filenameGenerator, sheetName, cell, a.workDirectory)
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

	cell, err := provider.CellForLatLng(lat, lng, uint(srid))
	if err != nil {
		return nil, err
	}

	filenames, err := a.filenamesForCell(sheetName, cell, filenameTemplate)
	if err != nil {
		return nil, err
	}
	err = GeneratePDF(ctx, provider, cell, filenames)
	return filenames, err

}

func (a *Atlante) generatePDF(ctx context.Context, sheet *Sheet, grid *grids.Cell, filenameTemplate string) (*GeneratedFiles, error) {
	filenames, err := a.filenamesForCell(sheet.Name, grid, filenameTemplate)
	if err != nil {
		return nil, err
	}
	if a.Notifier != nil && a.JobID != "" {
		sheet.Emitter, err = a.Notifier.NewEmitter(a.JobID)
		if err != nil {
			sheet.Emitter = nil
			log.Warnf("Failed to init emitter: %v", err)
		}
	}
	err = GeneratePDF(ctx, sheet, grid, filenames)
	if sheet.Emitter != nil {
		if err != nil {
			sheet.Emitter.Emit(field.Failed{Error: err})
		} else {
			sheet.Emitter.Emit(field.Completed{})
		}
	}
	return filenames, err
}

func (a *Atlante) GeneratePDFJob(ctx context.Context, job Job, filenameTemplate string) (*GeneratedFiles, error) {
	cell := job.Cell
	sheet, err := a.SheetFor(job.SheetName)
	if err != nil {
		return nil, err
	}
	a.JobID = job.MetaData["job_id"]
	return a.generatePDF(ctx, sheet, cell, filenameTemplate)
}

func (a *Atlante) GeneratePDFMDGID(ctx context.Context, sheetName string, mdgID *grids.MDGID, filenameTemplate string) (*GeneratedFiles, error) {

	sheet, err := a.SheetFor(sheetName)
	if err != nil {
		return nil, err
	}

	cell, err := sheet.CellForMDGID(mdgID)
	if err != nil {
		return nil, err
	}
	return a.generatePDF(ctx, sheet, cell, filenameTemplate)
}

func (a *Atlante) GeneatePDFBounds(ctx context.Context, sheetName string, bounds geom.Extent, srid uint, filenameTemplate string) (*GeneratedFiles, error) {
	sheet, err := a.SheetFor(sheetName)
	if err != nil {
		return nil, err
	}

	cell, err := sheet.CellForBounds(bounds, srid)
	if err != nil {
		return nil, err
	}
	return a.generatePDF(ctx, sheet, cell, filenameTemplate)
}
