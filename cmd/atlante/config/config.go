package config

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/config"
	"github.com/go-spatial/maptoolkit/atlante/filestore"
	fsmulti "github.com/go-spatial/maptoolkit/atlante/filestore/multi"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/tegola/dict"
)

var (
	// Providers provides the grid providers
	Providers = make(map[string]grids.Provider)
	// FileStores are the files store providers
	FileStores = make(map[string]filestore.Provider)
	a          atlante.Atlante
)

// Provider is a config structure for Grid Providers
type Provider struct {
	dict.Dicter
}

// NameGridProvider implements grids.Config interface
func (pcfg Provider) NameGridProvider(key string) (grids.Provider, error) {

	skey, err := pcfg.Dicter.String(key, nil)
	if err != nil {
		return nil, err
	}

	p, ok := Providers[skey]
	if !ok {
		return nil, grids.ErrProviderNotRegistered(skey)
	}
	return p, nil

}

// Filestore is a config for file stores
type Filestore struct {
	dict.Dicter
}

// FileStoreFor implements the filestore.Config interface
func (fscfg Filestore) FileStoreFor(name string) (filestore.Provider, error) {
	name = strings.ToLower(name)
	p, ok := FileStores[name]
	if !ok {
		return nil, filestore.ErrUnknownProvider(name)
	}
	return p, nil
}

// Load will attempt to load and validate a config at the given location
func Load(location string, dpi int, overrideDPI bool) (*atlante.Atlante, error) {
	var ok bool
	var a atlante.Atlante

	aURL, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	conf, err := config.LoadAndValidate(aURL)
	if err != nil {
		return nil, err
	}
	// Loop through providers creating a provider type mapping.
	for i, p := range conf.Providers {
		// type is required
		typ, err := p.String("type", nil)
		if err != nil {
			return nil, fmt.Errorf("error provider (%v) missing type : %v", i, err)
		}
		name, err := p.String("name", nil)
		if err != nil {
			return nil, fmt.Errorf("error provider( %v) missing name : %v", i, err)
		}
		name = strings.ToLower(name)
		if _, ok := Providers[name]; ok {
			return nil, fmt.Errorf("error provider with name (%v) is already registered", name)
		}
		prv, err := grids.For(typ, Provider{p})
		if err != nil {
			return nil, fmt.Errorf("error registering provider #%v: %v", i, err)
		}

		Providers[name] = prv
	}

	// filestores
	for i, fstore := range conf.FileStores {
		// type is required
		typ, err := fstore.String("type", nil)
		if err != nil {
			return nil, fmt.Errorf("error filestore (%v) missing type : %v", i, err)
		}
		name, err := fstore.String("name", nil)
		if err != nil {
			return nil, fmt.Errorf("error filestore (%v) missing name: %v", i, err)
		}
		name = strings.ToLower(name)
		if _, ok = FileStores[name]; ok {
			return nil, fmt.Errorf("error provider(%v) with name (%v) is already registered", i, name)
		}
		prv, err := filestore.For(typ, Filestore{fstore})
		if err != nil {
			return nil, fmt.Errorf("error registering filestore %v:%v", i, err)
		}
		FileStores[name] = prv
	}

	if len(conf.Sheets) == 0 {
		return nil, fmt.Errorf("no sheets configured")
	}
	// Establish sheets
	for i, sheet := range conf.Sheets {

		providerName := strings.ToLower(string(sheet.ProviderGrid))

		prv, ok := Providers[providerName]
		if providerName != "" && !ok {
			return nil, fmt.Errorf("error locating provider (%v) for sheet %v (#%v)", providerName, sheet.Name, i)
		}
		templateURL, err := url.Parse(string(sheet.Template))
		if err != nil {
			return nil, fmt.Errorf("error parsing template url (%v) for sheet %v (#%v)",
				string(sheet.Template),
				sheet.Name,
				i,
			)
		}
		name := strings.ToLower(string(sheet.Name))
		var fstores []filestore.Provider
		for _, filestoreString := range sheet.Filestores {
			filestoreName := strings.TrimSpace(strings.ToLower(string(filestoreString)))
			var fsprv filestore.Provider
			if filestoreName == "" {
				continue
			}
			fsprv, ok = FileStores[filestoreName]
			if !ok {
				log.Println("Known file stores are:")
				for k := range FileStores {
					log.Println("\t", k)
				}
				return nil, filestore.ErrUnknownProvider(filestoreName)
			}
			fstores = append(fstores, fsprv)
		}
		var fsprv filestore.Provider
		switch len(fstores) {
		case 0:
			fsprv = nil
		case 1:
			fsprv = fstores[0]
		default:
			fsprv = fsmulti.New(fstores...)
		}
		odpi := uint(sheet.DPI)
		// 0 means it's not set
		if overrideDPI || odpi == 0 {
			odpi = uint(dpi)
		}

		sht, err := atlante.NewSheet(
			name,
			prv,
			uint(odpi),
			uint(sheet.Scale),
			string(sheet.Style),
			templateURL,
			fsprv,
		)
		if err != nil {
			return nil, fmt.Errorf("error trying to create sheet %v: %v", i, err)
		}
		err = a.AddSheet(sht)
		if err != nil {
			return nil, fmt.Errorf("error trying to add sheet %v: %v", i, err)
		}
	}

	return &a, nil
}
