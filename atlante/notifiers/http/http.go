package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"text/template"

	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante/notifiers"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"
	"github.com/prometheus/common/log"
)

const (
	TYPE = "http"

	DefaultContentType = "application/json"
	DefaultURLTemplate = "/job/{{.job_id}}/status"

	ConfigKeyContentType = "content_type"
	ConfigKeyURLTemplate = "url_template"
)

func initFunc(cfg notifiers.Config) (notifiers.Provider, error) {
	var err error
	contentType := DefaultContentType
	contentType, err = cfg.String(ConfigKeyContentType, &contentType)
	if err != nil {
		return nil, err
	}
	urlTemplate := DefaultURLTemplate
	urlTemplate, err = cfg.String(ConfigKeyURLTemplate, &urlTemplate)
	t, err := template.New("url").Parse(urlTemplate)
	if err != nil {
		return nil, err
	}
	log.Infof("configured notifier %v", TYPE)
	return &Provider{
		contentType: contentType,
		url:         t,
	}, nil
}

func init() {
	notifiers.Register(TYPE, initFunc, nil)
}

type Provider struct {
	contentType string
	url         *template.Template
}

func (p *Provider) NewEmitter(jobid string) (notifiers.Emitter, error) {
	var str strings.Builder
	var ctx = struct {
		JobID string
	}{
		JobID: jobid,
	}
	if err := p.url.Execute(&str, ctx); err != nil {
		return nil, err
	}

	return &emitter{
		contentType: p.contentType,
		url:         str.String(),
	}, nil
}

type emitter struct {
	jobid       string
	contentType string
	url         string
}

func (e *emitter) Emit(se field.StatusEnum) error {
	if e == nil {
		return errors.String("emitter is nil")
	}
	bdy, err := json.Marshal(field.Status{se})
	if err != nil {
		return err
	}
	buff := bytes.NewBuffer(bdy)
	// Don't care about the response
	log.Infof("posting to %v:%s", e.url, string(bdy))
	_, err = http.Post(e.url, e.contentType, buff)
	if err != nil {
		log.Warnf("error posting to (%v): %v", e.url, err)
	}
	return err
}
