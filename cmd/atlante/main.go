package main

import (
	"fmt"
	"os"

	"github.com/go-spatial/maptoolkit/cmd/atlante/cmd"
)

// cleanupFns are functions that should be run before exiting.
var cleanupFns []func()

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

// Cleanup should be call just before exiting the program. This allows all the registered
// providers to run any cleanup routines they have.
func Cleanup() {
	for _, cln := range cleanupFns {
		if cln == nil {
			continue
		}
		cln()
	}
}

func main() {

	exitCode := 0
	select {
	case err := <-runRoot():
		if err == nil {
			break
		}
		switch e := err.(type) {
		case cmd.ErrExitWith:
			if e.ShowUsage {
				cmd.Root.Usage()
			}
			fmt.Fprintf(os.Stderr, e.Msg)
			exitCode = e.ExitCode
		default:
			fmt.Fprintln(os.Stderr, "got the following error:", err)
			exitCode = 1
		}
	}
	Cleanup()
	os.Exit(exitCode)
}
