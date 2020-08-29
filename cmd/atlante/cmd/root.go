package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-spatial/atlante/atlante"
	"github.com/go-spatial/atlante/atlante/grids"
	"github.com/go-spatial/atlante/cmd/atlante/config"
	"github.com/go-spatial/atlante/mbgl"
	"github.com/go-spatial/geom"
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
	listStyles bool
	workDir    string
	timeout    uint8
	srid       int
	boundsStr  string
	bounds     [4]float64
	haveBounds bool
	styleName  string
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
	Root.Flags().IntVar(&srid, "srid", 4326, "the srid for the bounds")
	Root.Flags().StringVar(&boundsStr, "bounds", "", "the bounds to use to generate the map")
	Root.Flags().StringVar(&styleName, "style", "", "The name of the style to use; will use the default for sheet if not given")
	Root.Flags().BoolVar(&listStyles, "list-styles", false, "list out the styles, if sheet is defined, then just list the styles for that sheet.")

	// Add server command
	Root.AddCommand(Server)
}

// Root is the main cobra command
var Root = &cobra.Command{
	Use:   "atlante",
	Short: "Atlante is a flexable server to build static print maps",
	Long: `A flexable server for building static print maps from tegola servers
built with love and c8h10n4o2. Complete documentation is available at
http://github.com/go-spatial/atlante`,
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

	case listStyles:
		sheets := a.Sheets()
		for _, sheet := range sheets {
			if sheetName != "" && sheet.Name != sheetName {
				continue
			}
			styleNames := sheet.Styles.Styles()
			styleword := "styles"
			if len(styleNames) == 0 {
				fmt.Fprintf(os.Stdout, "sheet '%v' has no styles; please fix.", sheet.Name)
				continue
			}
			if len(styleNames) == 1 {
				styleword = "style"
			}
			fmt.Fprintf(os.Stdout, "sheet '%v' has %v %s:\n", sheet.Name, len(styleNames), styleword)
			for i, name := range styleNames {
				style, _ := sheet.Styles.For(name)
				name = style.Name
				format := "  % 5d: %s -- %20s\n%s\n"
				if i == 0 {
					name = fmt.Sprintf("[default] %s", name)
				}
				fmt.Fprintf(os.Stdout, format, i+1, name, style.Location, style.Description)
			}
			fmt.Fprintln(os.Stdout, "\n")
		}
		return nil, nil

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
			sheet, err := a.SheetFor(sname)
			if err != nil {
				return nil, err
			}

			var grid *grids.Cell

			if haveBounds {
				ext := geom.Extent{float64(bounds[0]), float64(bounds[1]), float64(bounds[2]), float64(bounds[3])}
				grid, err = sheet.CellForBounds(ext, uint(srid))
			} else {
				mdgID := grids.NewMDGID(mdgid)
				grid, err = sheet.CellForMDGID(mdgID)
			}
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
	case haveBounds:
		// We have bounds to deal with.
		sname := a.NormalizeSheetName(sheetName, true)
		ext := geom.Extent{float64(bounds[0]), float64(bounds[1]), float64(bounds[2]), float64(bounds[3])}
		return a.GeneratePDFBounds(ctx, sname, styleName, ext, uint(srid), "")
	default:
		sname := a.NormalizeSheetName(sheetName, true)
		return a.GeneratePDFMDGID(ctx, sname, styleName, grids.NewMDGID(mdgid), "")
	}
}

func parseBounds() error {
	if boundsStr == "" {
		return nil
	}
	parts := strings.Split(boundsStr, ",")
	var floatParts []float64
	for i := range parts {
		part := strings.ReplaceAll(parts[i], " ", "")
		if part == "" {
			continue
		}
		flt, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return err
		}
		floatParts = append(floatParts, flt)
	}
	if len(floatParts) != 4 {
	}
	bounds[0] = floatParts[0]
	bounds[1] = floatParts[1]
	bounds[2] = floatParts[2]
	bounds[3] = floatParts[3]
	haveBounds = true
	return nil
}

func rootCmdRun(cmd *cobra.Command, args []string) error {

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if err := parseBounds(); err != nil {
		return ErrExitWith{
			ShowUsage: true,
			Msg:       fmt.Sprintf("[error] bounds incorrect: %\n", err),
			Err:       err,
			ExitCode:  1,
		}

	}
	{
		pid := os.Getpid()
		fmt.Fprintf(cmd.OutOrStderr(), "[config] OS PID: %v\n", pid)
		if pid != 1 {
			fmt.Fprintf(cmd.OutOrStderr(), "[config] Parent PID: %v\n", os.Getppid())
		}
	}

	a, err := config.Load(configFile, dpi, cmd.Flag("dpi").Changed)
	if err != nil {
		return ErrExitWith{
			ShowUsage: true,
			Msg:       fmt.Sprintf("[error] loading config: %v\n", err),
			Err:       err,
			ExitCode:  1,
		}
	}

	if timeout != 0 {
		to := time.Duration(timeout) * time.Minute
		fmt.Fprintf(cmd.OutOrStderr(), "[config] timeout: %v\n", to)
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(to))
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	defer cancel()
	mbgl.StartSnapshotManager(ctx)

	go func() {
		sigch := make(chan os.Signal, 1)
		signal.Notify(sigch)
		done := ctx.Done()
		//cancelled := gdcmd.Cancelled()
		for {
			select {
			case <-done:
				// no opt
				return
			case sig := <-sigch:
				log.Printf("Got signal %v", sig)
				sysSignal, ok := sig.(syscall.Signal)
				if !ok {
					break
				}
				switch sysSignal {
				default:
					break
				case syscall.SIGABRT:
				case syscall.SIGQUIT:
				case syscall.SIGTERM:
				case syscall.SIGKILL:
				}
				cancel()
				return
			}
		}
	}()

	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return ErrExitWith{
				ShowUsage: true,
				Msg:       fmt.Sprintf("[error] changing to working dir (%v), aborting", workDir),
				Err:       err,
				ExitCode:  2,
			}
		}
	}

	if len(bounds) > 4 {
		return ErrExitWith{
			ShowUsage: true,
			Msg:       fmt.Sprintf("[error] bounds should only be 4 values"),
			ExitCode:  2,
		}
	}

	// Check to see if JOB is set, if it is decode it into a job struct.
	generatedFiles, err := rootCmdParseArgs(ctx, a)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "[error] generating pdf\n")
		eerr := ErrExitWith{
			Err:       err,
			ShowUsage: true,
			ExitCode:  3,
		}
		var strwriter strings.Builder
		switch e := err.(type) {
		case atlante.ErrUnknownSheetName:
			fmt.Fprintf(&strwriter, "\t[error] unknown sheet name `%v`\n", string(e))
			fmt.Fprintf(&strwriter, "\tknown sheets\n")
			for _, snm := range a.SheetNames() {
				fmt.Fprintf(&strwriter, "\t\t%v\n", snm)
			}
		default:
			fmt.Fprintf(&strwriter, "[error] generating pdf\n\t%v\n", err)
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
