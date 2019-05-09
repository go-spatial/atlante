package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
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
	dpi        int
	outputDir  string
)

func init() {

	flag.StringVar(&mdgid, "mdgid", "V795G25492", "mdgid of the grid")
	flag.StringVar(&configFile, "config", "config.toml", "The config file to use")
	flag.StringVar(&sheetName, "sheet", "50k", "The configured sheet to use")
	flag.IntVar(&dpi, "dpi", 72, "The dpi to use")
	flag.StringVar(&outputDir, "o", "", "output location")

	log.SetFlags(log.LstdFlags | log.Lshortfile)
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

		providerName := strings.TrimSpace(strings.ToLower(string(sheet.ProviderGrid)))

		prv, ok := Providers[providerName]
		if !ok {
			return fmt.Errorf("for sheet %v (#%v),  requested provider (%v) not registered", sheet.Name, i, providerName)
		}
		templateURL, err := url.Parse(string(sheet.Template))
		if err != nil {
			return fmt.Errorf("for sheet %v (#%v),  failed to parse template url (%v) ",
				sheet.Name,
				i,
				string(sheet.Template),
			)
		}
		name := strings.ToLower(string(sheet.Name))
		log.Println("Scale", sheet.Scale, "dpi", dpi)

		sht, err := atlante.NewSheet(
			name,
			prv,
			float64(sheet.Zoom),
			uint(dpi),
			uint(sheet.Scale),
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

	// TODO(arolek): remove. this is a hack
	if outputDir != "" {
		os.Chdir(outputDir)
	}

	// San Diego : V795G25492
	filenames, err := a.GeneratePDFMDGID(ctx, sheetName, grids.NewMDGID(mdgid), "")
	if err != nil {
		panic(err)
	}
	log.Println("Filenames: ", filenames)

	/*
		if outputDir != "" {
			// filenames are relative to the binary so we don't need to pre process the path
			log.Printf("moving %v to %v\n", filenames.PDF, filepath.Join(outputDir, filenames.PDF))
			if err := copyFile(filenames.PDF, filepath.Join(outputDir, filenames.PDF)); err != nil {
				log.Fatal(err)
			}

			log.Printf("moving %v to %v\n", filenames.SVG, filepath.Join(outputDir, filenames.SVG))
			if err = copyFile(filenames.SVG, filepath.Join(outputDir, filenames.SVG)); err != nil {
				log.Fatal(err)
			}

			log.Printf("moving %v to %v\n", filenames.IMG, filepath.Join(outputDir, filenames.IMG))
			if err = copyFile(filenames.IMG, filepath.Join(outputDir, filenames.IMG)); err != nil {
				log.Fatal(err)
			}
		}
	*/
}

// copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
