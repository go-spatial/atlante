package main

import (
	"fmt"
	"os"

	"github.com/go-spatial/maptoolkit/cmd/atlante/cmd"
)

func runRoot() <-chan error {
	errch := make(chan error)
	go func(errch chan error) {
		err := cmd.Root.Execute()
		if err != nil {
			errch <- err
		}
		close(errch)
	}(errch)
	return errch
}

func main() {
	errch := runRoot()
	select {
	case err := <-errch:
		if err == nil {
			break
		}
		switch e := err.(type) {
		case cmd.ErrExitWith:
			if e.ShowUsage {
				cmd.Root.Usage()
			}
			fmt.Fprintf(os.Stderr, e.Msg)
			os.Exit(e.ExitCode)
		default:
			fmt.Fprintln(os.Stderr, "got the following error:", err)
			os.Exit(1)
		}
	}
}
