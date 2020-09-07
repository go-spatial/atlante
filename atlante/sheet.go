package atlante

import (
	"io"
	"io/ioutil"
	math "math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/go-spatial/atlante/atlante/filestore"
	"github.com/go-spatial/atlante/atlante/grids"
	"github.com/go-spatial/atlante/atlante/internal/urlutil"
	"github.com/go-spatial/atlante/atlante/notifiers"
	"github.com/go-spatial/atlante/atlante/server/coordinator/field"
	"github.com/go-spatial/atlante/atlante/style"
	"github.com/prometheus/common/log"
)

const (
	// DefaultWidthMM is the default mm width if a width is not given
	DefaultWidthMM = 841 // A0 width
	// DefaultHeightMM is the default mm height if a height is not given
	DefaultHeightMM = 1189 // A0 height

	// inchPerMM is the number of inches in a mm
	inchPerMM = 1 / 25.4
)

// Sheet describes a map sheet
type Sheet struct {
	Name string
	// Data provider for this sheet
	grids.Provider
	// DPI at which the sheet should be rendered. Defaults to 144
	DPI uint
	// Scale value (50000, 5000, etc...)
	Scale uint

	// Styles is the name of the style to use from the global style set.
	Styles style.Provider

	// Template file to use
	SvgTemplateFilename string

	// Where to write the file created for the seet.
	Filestore filestore.Provider

	// Parsed and ready template
	svgTemplate *template.Template

	// Description of the sheet
	Desc string

	Emitter notifiers.Emitter
	// Width of the canvas in mm
	Width float64
	// Height of the canvas in mm
	Height float64

	// This will be used by the template functions
	FuncFilestoreWriter filestore.FileWriter

	// UseCached tells remote file providers to use cached versions
	UseCached bool
}

// loadTemplateDir will load additional tempalates if the location is local and there is
// a directory called `templates` in the base of location. It will load all file with
// the extention `.tpl` from the `templates` directory
func loadTemplateDir(t *template.Template, location *url.URL) (*template.Template, error) {
	if urlutil.IsRemote(location) {
		return t, nil
	}

	// For now we only support template directories
	// if the svg file is local
	filePath := location.EscapedPath()
	baseDir := filepath.Dir(filePath)
	templatesDir := filepath.Join(baseDir, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return t, nil
	}
	var subtemplates []string

	fileInfos, err := ioutil.ReadDir(templatesDir)
	if err != nil {
		return t, err
	}
	for _, info := range fileInfos {
		if info.Size() == 0 || info.IsDir() {
			continue
		}
		fname := info.Name()
		if filepath.Ext(fname) != ".tpl" {
			continue
		}
		subtemplates = append(subtemplates,
			filepath.Join(templatesDir, info.Name()),
		)
	}

	if len(subtemplates) == 0 {
		return t, nil
	}
	sort.Strings(subtemplates)
	return t.ParseFiles(subtemplates...)
}

// NewSheet returns a new sheet
func NewSheet(name string, provider grids.Provider, dpi uint, desc string, stylelist style.Provider, svgTemplateFilename *url.URL, fs filestore.Provider) (*Sheet, error) {
	var (
		err error
		t   *template.Template
	)

	name = strings.TrimSpace(strings.ToLower(name))

	scale := provider.CellSize()
	sheet := &Sheet{
		Name:                name,
		Provider:            provider,
		DPI:                 dpi,
		Scale:               uint(scale),
		Styles:              stylelist,
		SvgTemplateFilename: svgTemplateFilename.String(),
		Filestore:           fs,
		Desc:                desc,
		Height:              DefaultHeightMM,
		Width:               DefaultWidthMM,
	}

	log.Infof("Sheet %v processing template: %v", name, svgTemplateFilename)
	tpl, err := urlutil.ReadAll(svgTemplateFilename)
	if err != nil {
		return nil, err
	}

	t = template.New(svgTemplateFilename.String()).
		Funcs(sheet.AddTemplateFuncs(funcMap)).
		Option("missingkey=error")
	t, err = loadTemplateDir(t, svgTemplateFilename)
	if err != nil {
		return nil, err
	}

	t, err = t.Parse(string(tpl))
	if err != nil {
		return nil, err
	}

	sheet.svgTemplate = t
	// let's read in our templates
	log.Infof("Sheet %v has the following templates.\n", name)
	tlps := sheet.svgTemplate.Templates()
	for i := range tlps {
		log.Infof("\tTemplate %v : %v", i, tlps[i].Name())
	}

	return sheet, nil
}

// Execute the sheets template
func (sheet *Sheet) Execute(wr io.Writer, tplContext *GridTemplateContext) error {
	return sheet.svgTemplate.Execute(wr, tplContext)
}

func mmToPoint(mm float64, dpi uint) uint64 {
	inch := mm * inchPerMM
	return uint64(math.Round(inch * float64(dpi)))
}

// HeightInPoints returns the height in points given the dpi (dots per inch)
func (sheet *Sheet) HeightInPoints(dpi uint) uint64 {
	if sheet == nil {
		// return the default points
		return mmToPoint(DefaultHeightMM, dpi)
	}
	return mmToPoint(sheet.Height, dpi)
}

// WidthInPoints returns the height in points given the dpi (dots per inch)
func (sheet *Sheet) WidthInPoints(dpi uint) uint64 {
	if sheet == nil {
		// return the default points
		return mmToPoint(DefaultWidthMM, dpi)
	}
	// mm2inch is inches/mm , dpi is points/inches
	return mmToPoint(sheet.Width, dpi)
}

// Emit will emit an notifier event if the notifier is not nil.
func (sheet *Sheet) Emit(status field.StatusEnum) error {
	if sheet == nil || sheet.Emitter == nil {
		return nil
	}
	return sheet.Emitter.Emit(status)
}

// EmitError will emit a notifier event for a failed processing job
func (sheet *Sheet) EmitError(desc string, err error) error {
	if sheet == nil || sheet.Emitter == nil || err == nil {
		return nil
	}
	return sheet.Emitter.Emit(field.Failed{
		Description: desc,
		Error:       err,
	})
}

func (sheet *Sheet) GetURL(mdgid string, filename string, intermediate bool) (filestore.URLInfo, bool) {
	var (
		pdfURL filestore.URLInfo
	)
	if sheet == nil {
		return pdfURL, false
	}
	pather, ok := sheet.Filestore.(filestore.Pather)
	if !ok {
		return pdfURL, false
	}
	pdfURL, err := pather.PathURL(mdgid, filename, intermediate)
	if err != nil {
		if err == filestore.ErrUnsupportedOperation {
			// no opt
		} else if e, ok := err.(filestore.ErrPath); ok && e.Err == filestore.ErrFileDoesNotExist {
			// no opt
		} else {
			log.Warnf("filestore error: %v", e)
		}
		return pdfURL, false
	}
	return pdfURL, true
}

// NormalizeSheetName will return a normalized version of the sheetname, or if the sheet
func (a *Atlante) NormalizeSheetName(sheetName string, getDefault bool) string {

	sheetnm := strings.TrimSpace(strings.ToLower(sheetName))
	if sheetnm != "" {
		return sheetnm
	}
	if !getDefault {
		return ""
	}
	sheets := a.SheetNames()
	if len(sheets) == 0 {
		return ""
	}
	return sheets[0]
}

// SheetFor returns the sheet for the given name, if the sheet does not exists
// sheet.ErrUnknownSheetName is returned.
func (a *Atlante) SheetFor(sheetName string) (*Sheet, error) {
	if a == nil {
		return nil, ErrNilAtlanteObject
	}
	if len(a.sheets) == 0 {
		return nil, ErrNoSheets
	}
	sheetnm := a.NormalizeSheetName(sheetName, false)
	if sheetnm == "" {
		return nil, ErrBlankSheetName
	}

	a.sLock.RLock()
	sheet := a.sheets[sheetnm]
	a.sLock.RUnlock()
	if sheet == nil {
		return nil, ErrUnknownSheetName(sheetnm)
	}
	return sheet, nil
}

// SheetNames returns the currently configured sheet names.
func (a *Atlante) SheetNames() (sheets []string) {
	if a == nil || len(a.sheets) == 0 {
		return sheets
	}
	sheets = make([]string, len(a.sheets))
	a.sLock.RLock()
	i := 0
	for k := range a.sheets {
		sheets[i] = k
		i++
	}
	a.sLock.RUnlock()
	sort.Strings(sheets)
	return sheets
}

// Sheets returns the currently configured sheets
func (a *Atlante) Sheets() (sheets []*Sheet) {
	if a == nil || len(a.sheets) == 0 {
		return sheets
	}
	sheetnames := a.SheetNames()
	sheets = make([]*Sheet, len(sheetnames))
	a.sLock.RLock()
	for i, k := range sheetnames {
		sheets[i] = a.sheets[k]
	}
	a.sLock.RUnlock()
	return sheets
}

// AddSheet will add a sheet to the instance of atlante.
// Error that can be generated are `ErrBlankSheetName` and `ErrDuplicateSheetName`
// The name of the sheet is normalize to lowercase and spaces trimmed
func (a *Atlante) AddSheet(s *Sheet) error {
	if s == nil {
		return ErrNilSheet
	}
	name := strings.TrimSpace(strings.ToLower(s.Name))
	if name == "" {
		return ErrBlankSheetName
	}
	a.sLock.Lock()
	defer a.sLock.Unlock()
	if a.sheets == nil {
		a.sheets = make(map[string]*Sheet)
		a.sheets[name] = s
		return nil
	}
	if _, ok := a.sheets[name]; ok {
		return ErrDuplicateSheetName
	}
	a.sheets[name] = s
	return nil
}
