package atlante

import (
	"io"
	"net/url"
	"sort"
	"strings"
	"text/template"

	"github.com/go-spatial/maptoolkit/atlante/filestore"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/atlante/internal/urlutil"
	"github.com/go-spatial/maptoolkit/atlante/notifiers"
)

const (
	// DefaultHeightMM is the default mm height if a height is not given
	DefaultHeightMM = 28.16667
	// DefaultWidthMM is the default mm width if a width is not given
	DefaultWidthMM = 36.20833

	mm2inch = 1 / 25.4
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
	// URL to the style file
	Style string
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
}

// NewSheet returns a new sheet
func NewSheet(name string, provider grids.Provider, dpi uint, desc string, style string, svgTemplateFilename *url.URL, fs filestore.Provider) (*Sheet, error) {
	var (
		err error
		t   *template.Template
	)

	name = strings.TrimSpace(strings.ToLower(name))

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

	scale := provider.CellSize()

	return &Sheet{
		Name:                name,
		Provider:            provider,
		DPI:                 dpi,
		Scale:               uint(scale),
		Style:               style,
		SvgTemplateFilename: svgTemplateFilename.String(),
		svgTemplate:         t,
		Filestore:           fs,
		Desc:                desc,
		Height:              DefaultHeightMM,
		Width:               DefaultWidthMM,
	}, nil
}

// Execute the sheets template
func (sheet *Sheet) Execute(wr io.Writer, tplContext GridTemplateContext) error {
	return sheet.svgTemplate.Execute(wr, tplContext)
}

// HeightInPoints returns the height in points given the dpi (dots per inch)
func (sheet *Sheet) HeightInPoints(dpi uint) float64 {
	var mm float64
	if sheet == nil {
		// return the default points
		mm = DefaultHeightMM
	}
	mm = sheet.Height
	// mm2inch is inches/mm , dpi is points/inches
	return mm * float64(dpi) * mm2inch
}

// WidthInPoints returns the height in points given the dpi (dots per inch)
func (sheet *Sheet) WidthInPoints(dpi uint) float64 {
	var mm float64
	if sheet == nil {
		// return the default points
		mm = DefaultWidthMM
	}
	mm = sheet.Width
	// mm2inch is inches/mm , dpi is points/inches
	return mm * float64(dpi) * mm2inch
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
// sheet.ErrUnkownSheetName is returned.
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
