package atlante

import (
	"context"
	"fmt"
	"image/png"
	"io"
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
	"github.com/go-spatial/maptoolkit/atlante/notifiers"
	_ "github.com/go-spatial/maptoolkit/atlante/notifiers/http"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"
	"github.com/go-spatial/maptoolkit/mbgl/bounds"
	"github.com/go-spatial/maptoolkit/mbgl/image"
	mbgl "github.com/go-spatial/maptoolkit/mbgl/image"
	"github.com/go-spatial/maptoolkit/svg2pdf"
	"github.com/prometheus/common/log"
)

// ImgStruct is a wrapper around an image that makes the image available to the
// the template, and allows for the image to be encoded only if it's requested.
type ImgStruct struct {
	filename     string
	filestore    filestore.FileWriter
	intermediate bool
	// Have we generated the file?
	// This will allow us to generate the file only when requested
	generated                 bool
	lck                       sync.Mutex
	image                     *mbgl.Image
	startGenerationCallBack   func()
	endGenerationCallBack     func()
	generationFailureCallBack func(err error)
}

// Height returns the height of the image
func (img ImgStruct) Height() int { return img.image.Bounds().Dy() }

// Width returns the width of the image
func (img ImgStruct) Width() int { return img.image.Bounds().Dx() }

// Filename returns the file name of the image
func (img *ImgStruct) Filename() (string, error) {
	if img == nil {
		return "", nil
	}
	// User only cares about filename if img.image is nil
	if img.generated || img.image == nil {
		return img.filename, nil
	}
	// We need to generate the file and then return the filename
	return img.generateImage()
}

// Close closes out any open resources
func (img ImgStruct) Close() {
	if !img.generated || img.image == nil {
		return
	}
	img.image.Close()
}

// generateIamge is use to create the image into the filestore
func (img *ImgStruct) generateImage() (fn string, err error) {
	if img.generated {
		return img.filename, nil
	}

	// If we don't have an image to generator.
	// User only cares about the filename
	if img.image == nil {
		img.generated = true
		return img.filename, nil
	}

	// No file store to write out the image.
	// User only cares about the filename
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
	if img.startGenerationCallBack != nil {
		img.startGenerationCallBack()
	}
	if img.generationFailureCallBack != nil {
		defer func() {
			if err != nil {
				img.generationFailureCallBack(err)
			}
		}()
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
	if img.endGenerationCallBack != nil {
		img.endGenerationCallBack()
	}
	img.generated = true
	return img.filename, nil
}

type GridTemplateContext struct {
	Image         *ImgStruct
	GroundMeasure float64
	Grid          *grids.Cell
	DPI           uint
	Scale         uint
	Zoom          float64
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

func GeneratePDF(ctx context.Context, sheet *Sheet, grid *grids.Cell, filenames *GeneratedFiles) error {
	if grid == nil {
		return ErrNilGrid
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

	log.Infoln("filenames: ", filenames.IMG, filenames.SVG, filenames.PDF)

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
		float64(grid.GetSw().GetLat()),
	)

	// Generate the PNG
	prj := bounds.ESPG3857
	width, height := grid.WidthHeightForZoom(zoom)
	latLngCenterPt := grid.CenterPtForZoom(zoom)
	log.Infoln("width", width, "height", height)
	log.Infoln("zoom", zoom, "Scale", sheet.Scale, "dpi", sheet.DPI, "ground measure", nground)

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

	img := ImgStruct{
		filename:     filenames.IMG,
		image:        dstimg,
		filestore:    multiWriter,
		intermediate: true,
		startGenerationCallBack: func() {
			sheet.Emit(field.Processing{
				Description: fmt.Sprintf("intermediate file: %v", filenames.IMG),
			})
		},
		generationFailureCallBack: func(err error) {
			sheet.EmitError(
				fmt.Sprintf("failed to generate intermediate file: %v", filenames.IMG),
				err,
			)
		},
	}
	defer func() {
		img.Close()
	}()

	sheet.Emit(field.Processing{
		Description: fmt.Sprintf("intermediate file: %v ", filenames.SVG),
	})
	file, err := multiWriter.Writer(filenames.SVG, true)
	defer file.Close()

	// Fill out template
	err = sheet.Execute(file, GridTemplateContext{
		Image:         &img,
		DPI:           sheet.DPI,
		Scale:         sheet.Scale,
		Zoom:          zoom,
		GroundMeasure: nground,
		Grid:          grid,
	})
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

	widthpts, heightpts := float64(sheet.WidthInPoints(72)), float64(sheet.HeightInPoints(72))
	log.Infof("pdf %v,%v", widthpts, heightpts)
	if err = svg2pdf.GeneratePDF(svgfn, pdffn, widthpts, heightpts); err != nil {
		log.Warnf("error generating pdf: %v", err)
		sheet.EmitError("generate pdf failed", err)
		return err
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
