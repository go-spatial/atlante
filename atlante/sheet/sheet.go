package sheet

import (
	"text/template"

	"github.com/go-spatial/maptoolkit/atlante/grids"
)

type Provider struct {
	p grids.Provider

	name             string
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
