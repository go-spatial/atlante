package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-spatial/maptoolkit/atlante"
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

// ErrExitWith returns the an error that should be displayed to the user.
// with the error code
type ErrExitWith struct {
	// Msg to show to the user.
	Msg string
	// Error is the wrapped error
	Err error
	// ExitCode to return to the system
	ExitCode int
	// ShowUsage will tell the main application
	ShowUsage bool
}

// Error implements the Error interface
func (e ErrExitWith) Error() string {
	return fmt.Sprintf("%v: %v", e.Msg, e.Err)
}

var (
	// Flags
	mdgid      string
	configFile string
	sheetName  string
	dpi        = DefaultDPI
	job        string
	jobid      string
	showJob    bool
	workDir    string
	timeout    uint8
)

func init() {

	Root.PersistentFlags().StringVar(&configFile, "config", "config.toml", "config file to use")
	Root.PersistentFlags().IntVar(&dpi, "dpi", DefaultDPI, "dpi to use")
	Root.Flags().StringVar(&mdgid, "mdgid", "", "mdgid of the grid")
	Root.Flags().StringVar(&sheetName, "sheet", "", "the sheet to use")
	Root.Flags().StringVar(&job, "job", "", "base64 encoded job")
	Root.Flags().StringVar(&jobid, "job-id", "", "job-id to use, if a job is defined in the job, it will override this value")
	Root.Flags().BoolVar(&showJob, "show-job", false, "print out the job string for the parameters, and exit. If job is given, print out string representation of the job")
	Root.Flags().StringVarP(&workDir, "workdir", "o", "", "workdir to find the assets and leave the output")
	Root.Flags().Uint8Var(&timeout, "timeout", 0, "timeout in minutes, 0 means no timeout.")

	// Add server command
	Root.AddCommand(Server)

}

// Root is the main cobra command
var Root = &cobra.Command{
	Use:   "atlante",
	Short: "Atlante is a flexable server to build static print maps",
	Long: `A flexable server for building static print maps from tegola servers
built with love and c8h10n4o2. Complete documentation is available at
http://github.com/go-spatial/maptoolkit`,
	RunE: rootCmdRun,
}

func generatePDFForJob(ctx context.Context, a *atlante.Atlante, jobstr string) (*atlante.GeneratedFiles, error) {
	job, err := atlante.Base64UnmarshalJob(jobstr)
	if err != nil {
		return nil, fmt.Errorf("%v : jobstr '%v' ", err, jobstr)
	}
	return a.GeneratePDFJob(ctx, *job, "")
}

func rootCmdParseArgs(ctx context.Context, a *atlante.Atlante) (*atlante.GeneratedFiles, error) {
	a.JobID = jobid
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
			grid, err := sheet.CellForMDGID(mdgID)
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

func rootCmdRun(cmd *cobra.Command, args []string) error {

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	a, err := config.Load(configFile, dpi, cmd.Flag("dpi").Changed)
	if err != nil {
		return ErrExitWith{
			ShowUsage: true,
			Msg:       fmt.Sprintf("error loading config: %v\n", err),
			Err:       err,
			ExitCode:  1,
		}
	}
	if timeout != 0 {
		to := time.Duration(timeout) * time.Minute
		fmt.Fprintf(cmd.OutOrStderr(), "setting timeout to: %v\n", to)
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(to))
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	defer cancel()
	mbgl.StartSnapshotManager(ctx)

	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return ErrExitWith{
				ShowUsage: true,
				Msg:       fmt.Sprintf("error changing to working dir (%v), aborting", workDir),
				Err:       err,
				ExitCode:  2,
			}
		}
	}

	// Check to see if JOB is set, if it is decode it into a job struct.
	generatedFiles, err := rootCmdParseArgs(ctx, a)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "error generating pdf\n")
		eerr := ErrExitWith{
			Err:       err,
			ShowUsage: true,
			ExitCode:  3,
		}
		var strwriter strings.Builder
		switch e := err.(type) {
		case atlante.ErrUnknownSheetName:
			fmt.Fprintf(&strwriter, "\terror unknown sheet name `%v`\n", string(e))
			fmt.Fprintf(&strwriter, "\tknown sheets\n")
			for _, snm := range a.SheetNames() {
				fmt.Fprintf(&strwriter, "\t\t%v\n", snm)
			}
		default:
			fmt.Fprintf(&strwriter, "error generating pdf\n\t%v\n", err)
		}
		eerr.Msg = strwriter.String()
		return eerr
	}

	if generatedFiles != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "PDF File: %v\n", generatedFiles.PDF)
		fmt.Fprintf(cmd.OutOrStderr(), "PNG File: %v\n", generatedFiles.IMG)
		fmt.Fprintf(cmd.OutOrStderr(), "SVG File: %v\n", generatedFiles.SVG)
	}
	return nil
}
