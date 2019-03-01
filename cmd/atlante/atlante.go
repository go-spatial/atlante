package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/config"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	_ "github.com/go-spatial/maptoolkit/atlante/grids/postgresql"
	"github.com/go-spatial/maptoolkit/mbgl"
	"github.com/go-spatial/tegola/dict"
)

var (
	Providers = make(map[string]grids.Provider)
	a         atlante.Atlante

	// Flags
	mdgid      string
	configFile string
	sheetName  string
)

func init() {

	flag.StringVar(&mdgid, "mdgid", "V795G25492", "mdgid of the grid")
	flag.StringVar(&configFile, "config", "config.toml", "The config file to use")
	flag.StringVar(&sheetName, "sheet", "50k", "The configured sheet to use")
}

type ProviderConfig struct {
	dict.Dicter
}

func (pcfg ProviderConfig) NameGridProvider(key string) (grids.Provider, error) {

	skey, err := pcfg.Dicter.String(key, nil)
	if err != nil {
		return nil, err
	}

	p, ok := Providers[skey]
	if !ok {
		log.Println("Known Provders:")
		for k, _ := range Providers {
			log.Println("\t", k)
		}
		return nil, grids.ErrProviderNotRegistered(skey)
	}
	return p, nil

}

func LoadConfig(location string) error {

	aURL, err := url.Parse(location)
	if err != nil {
		return err
	}
	conf, err := config.LoadAndValidate(aURL)
	if err != nil {
		return err
	}
	// Loop through providers creating a provider type mapping.
	for i, p := range conf.Providers {
		// type is required
		type_, err := p.String("type", nil)
		if err != nil {
			return fmt.Errorf("type missing for provider #%v: err", i, err)
		}
		name, err := p.String("name", nil)
		if err != nil {
			return fmt.Errorf("name missing for provider #%v: err", i, err)
		}
		name = strings.ToLower(name)
		if _, ok := Providers[name]; ok {
			return fmt.Errorf("provider with name (%v) is already registered", name)
		}
		prv, err := grids.For(type_, ProviderConfig{p})
		if err != nil {
			return fmt.Errorf("registered for provider #%v: err", i, err)
		}

		Providers[name] = prv
	}

	log.Println("Known Provders:")
	for k, _ := range Providers {
		log.Printf("\t'%v'\n", k)
	}
	// Establish sheets
	for i, sheet := range conf.Sheets {

		providerName := strings.ToLower(string(sheet.ProviderGrid))

		prv, ok := Providers[providerName]
		if !ok {
			return fmt.Errorf("for sheet %v (#%v),  requested provider (%v) not registered", sheet.Name, i, providerName)
		}
		/*
		styleURL, err := url.Parse(string(sheet.Style))
		if err != nil {
			return fmt.Errorf("for sheet %v (#%v),  failed to parse style url (%v) ",
				sheet.Name,
				i,
				string(sheet.Style),
			)
		}
		*/
		templateURL, err := url.Parse(string(sheet.Template))
		if err != nil {
			return fmt.Errorf("for sheet %v (#%v),  failed to parse template url (%v) ",
				sheet.Name,
				i,
				string(sheet.Template),
			)
		}
		name := strings.ToLower(string(sheet.Name))

		sht, err := atlante.NewSheet(
			name,
			prv,
			float64(sheet.Zoom),
			string(sheet.Style),
			templateURL,
		)
		if err != nil {
			return fmt.Errorf("Failed to create sheet %v: %v", i, err)
		}
		err = a.AddSheet(sht)
		if err != nil {
			return fmt.Errorf("Failed to add sheet %v: %v", i, err)
		}
	}

	return nil

}

func main() {
	flag.Parse()
	err := LoadConfig(configFile)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cancel when we are finished consuming integers
	mbgl.StartSnapshotManager(ctx)

	// San Diego : V795G25492
	filenames, err := a.GeneratePDFMDGID(ctx, sheetName, grids.NewMDGID(mdgid), "")
	if err != nil {
		panic(err)
	}
	log.Println("Filenames: ", filenames)
}
