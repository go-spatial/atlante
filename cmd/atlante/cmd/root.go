package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/filestore"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/cmd/atlante/config"
	"github.com/go-spatial/maptoolkit/mbgl"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

const (
	// DefaultDPI is the dpi we should render the images at.
	DefaultDPI = 144
)

var (
	// Flags
	mdgid      string
	configFile string
	sheetName  string
	dpi        = DefaultDPI
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

// Root is the main cobra command
var Root = &cobra.Command{
	Use:   "atlante",
	Short: "Atlante is a flexable server to build static print maps",
	Long: `A flexable server for building static print maps from tegola servers
built with love and c8h10n4o2. Complete documentation is available at
http://github.com/go-spatial/maptoolkit`,
	Run: rootCmdRun,
}

func generatePDFForJob(ctx context.Context, a *atlante.Atlante, jobstr string) (*atlante.GeneratedFiles, error) {
	job, err := atlante.Base64UnmarshalJob(jobstr)
	if err != nil {
		return nil, fmt.Errorf("%v : jobstr '%v' ", err, jobstr)
	}
	return a.GeneratePDFJob(ctx, *job, "")
}

func rootCmdParseArgs(ctx context.Context, a *atlante.Atlante) (*atlante.GeneratedFiles, error) {
	defer filestore.Cleanup()
	switch {
	case showJob:
		switch {
		case job != "":
			// We need to print out what's in the job.
			jb, err := atlante.Base64UnmarshalJob(job)
			if err != nil {
				return nil, fmt.Errorf("%v : jobstr '%v' ", err, job)
			}
			fmt.Fprintln(os.Stdout, proto.MarshalTextString(jb))
		default:
			sname := a.NormalizeSheetName(sheetName, true)
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
		return generatePDFForJob(ctx, a, job)
	default:
		sname := a.NormalizeSheetName(sheetName, true)
		return a.GeneratePDFMDGID(ctx, sname, grids.NewMDGID(mdgid), "")
	}
}

func rootCmdRun(cmd *cobra.Command, args []string) {

	a, err := config.Load(configFile, dpi, cmd.Flag("dpi").Changed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		cmd.Usage()
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mbgl.StartSnapshotManager(ctx)

	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			fmt.Fprintf(os.Stderr, "error changing to working dir (%v), aborting", workDir)
			cmd.Usage()
			os.Exit(3)
		}
	}

	// Check to see if JOB is set, if it is decode it into a job struct.
	generatedFiles, err := rootCmdParseArgs(ctx, a)
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
		cmd.Usage()
		os.Exit(2)
	}

	if generatedFiles != nil {
		fmt.Fprintf(os.Stdout, "PDF File: %v\n", generatedFiles.PDF)
		fmt.Fprintf(os.Stdout, "PNG File: %v\n", generatedFiles.IMG)
		fmt.Fprintf(os.Stdout, "SVG File: %v\n", generatedFiles.SVG)
	}
	os.Exit(0)
}
