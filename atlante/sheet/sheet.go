package sheet

import (
	"text/template"

	"github.com/go-spatial/maptoolkit/atlante/grids"
)

type Provider struct {
	p grids.Provider

	name             string
	zoom             float64
	style            string
	templateFilename string
	template         *template.Template
}

func (p *Provider) Name() string {
	if p == nil {
		return ""
	}
	return p.Name
}

func (p *Provider) Zoom() float64 {
	if p == nil {
		return 0
	}
	return p.zoom
}
