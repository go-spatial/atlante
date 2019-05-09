package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/config"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/mbgl"
	"github.com/go-spatial/tegola/dict"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

const (
	DefaultDPI = 144
)

var (
	Providers = make(map[string]grids.Provider)
	a         atlante.Atlante

	// Flags
	mdgid      string
	configFile string
	sheetName  string
	dpi        int = DefaultDPI
	job        string
	showJob    bool
	workDir    string
)

func init() {
	Root.PersistentFlags().StringVar(&configFile, "config", "config.toml", "config file to use")
	Root.Flags().StringVar(&mdgid, "mdgid", "", "mdgid of the grid")
	Root.Flags().StringVar(&sheetName, "sheet", "", "the sheet to use")
	Root.Flags().StringVar(&job, "job", "", "base64 encoded job")
	Root.Flags().BoolVar(&showJob, "show-job", false, "print out the job string for the parameters, and exit, if job is given with a string print out what's in the job string")
	Root.Flags().IntVar(&dpi, "dpi", DefaultDPI, "dpi to use")
	Root.Flags().StringVarP(&workDir, "workdir", "o", "", "workdir to find the assets and leave the output")
}

var Root = &cobra.Command{
	Use:   "atlante",
	Short: "Atlante is a flexable server to build static print maps",
	Long: `A flexable server for building static print maps from tegola servers
built with love and c8h10n4o2. Complete documentation is available at
http://github.com/go-spatial/maptoolkit`,
	Run: rootCmdRun,
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
			return fmt.Errorf("error provider (%v) missing type : %v", i, err)
		}
		name, err := p.String("name", nil)
		if err != nil {
			return fmt.Errorf("error provider( %v) missing name : %v", i, err)
		}
		name = strings.ToLower(name)
		if _, ok := Providers[name]; ok {
			return fmt.Errorf("error provider with name (%v) is already registered", name)
		}
		prv, err := grids.For(type_, ProviderConfig{p})
		if err != nil {
			return fmt.Errorf("error registering provider #%v: err", i, err)
		}

		Providers[name] = prv
	}

	if len(conf.Sheets) == 0 {
		return fmt.Errorf("no sheets configured")
	}
	// Establish sheets
	for i, sheet := range conf.Sheets {

		providerName := strings.ToLower(string(sheet.ProviderGrid))

		prv, ok := Providers[providerName]
		if providerName != "" && !ok {
			return fmt.Errorf("error locating provider (%v) for sheet %v (#%v)", providerName, sheet.Name, i)
		}
		templateURL, err := url.Parse(string(sheet.Template))
		if err != nil {
			return fmt.Errorf("error parsing template url (%v) for sheet %v (#%v)",
				string(sheet.Template),
				sheet.Name,
				i,
			)
		}
		name := strings.ToLower(string(sheet.Name))
		//log.Println("Scale", sheet.Scale, "dpi", dpi)

		sht, err := atlante.NewSheet(
			name,
			prv,
			uint(dpi),
			uint(sheet.Scale),
			string(sheet.Style),
			templateURL,
		)
		if err != nil {
			return fmt.Errorf("error trying to create sheet %v: %v", i, err)
		}
		err = a.AddSheet(sht)
		if err != nil {
			return fmt.Errorf("error trying to add sheet %v: %v", i, err)
		}
	}

	return nil
}

func generatePDFForJob(ctx context.Context, jobstr string) (*atlante.GeneratedFiles, error) {
	job, err := atlante.Base64UnmarshalJob(jobstr)
	if err != nil {
		return nil, err
	}
	return a.GeneratePDFJob(ctx, *job, "")
}

func sheetname() string {
	sname := strings.ToLower(strings.TrimSpace(sheetName))
	if sname == "" {
		sheets := a.Sheets()
		return sheets[0]
	}
	return sname
}

func rootCmdParseArgs(ctx context.Context) (*atlante.GeneratedFiles, error) {
	switch {
	case showJob:
		switch {
		case job != "":
			// We need to print out what's in the job.
			jb, err := atlante.Base64UnmarshalJob(job)
			if err != nil {
				return nil, err
			}
			fmt.Fprintln(os.Stdout, proto.MarshalTextString(jb))
		default:
			sname := sheetname()
			mdgID := grids.NewMDGID(mdgid)
			sheet, err := a.SheetFor(sname)
			if err != nil {
				return nil, err
			}
			grid, err := sheet.GridForMDGID(mdgID)
			if err != nil {
				return nil, err
			}
			metadata := make(map[string]string)
			jb := atlante.NewJob(sname, grid, metadata)
			jbstr, err := jb.Base64Marshal()
			if err != nil {
				return nil, err
			}
			fmt.Fprintln(os.Stdout, jbstr)
		}
		return nil, nil
	case job != "":
		return generatePDFForJob(ctx, job)
	default:
		return a.GeneratePDFMDGID(ctx, sheetname(), grids.NewMDGID(mdgid), "")
	}
}

func rootCmdRun(cmd *cobra.Command, args []string) {
	err := LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mbgl.StartSnapshotManager(ctx)

	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			fmt.Fprintf(os.Stderr, "error changing to working dir (%v), aborting", workDir)
			os.Exit(3)
		}
	}

	// Check to see if JOB is set, if it is decode it into a job struct.
	generatedFiles, err := rootCmdParseArgs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating pdf\n")
		switch e := err.(type) {
		case atlante.ErrUnknownSheetName:
			fmt.Fprintf(os.Stderr, "\terror unknown sheet name `%v`\n", string(e))
			fmt.Fprintf(os.Stderr, "\tknown sheets\n")
			for _, snm := range a.Sheets() {
				fmt.Fprintf(os.Stderr, "\t\t%v\n", snm)
			}
		default:
			fmt.Fprintf(os.Stderr, "error generating pdf\n\t%v\n", err)
		}
		os.Exit(2)
	}

	if generatedFiles != nil {
		fmt.Fprintf(os.Stdout, "PDF File: %v\n", generatedFiles.PDF)
		fmt.Fprintf(os.Stdout, "PNG File: %v\n", generatedFiles.IMG)
		fmt.Fprintf(os.Stdout, "SVG File: %v\n", generatedFiles.SVG)
	}
	os.Exit(0)
}
