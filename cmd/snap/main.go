package main

import (
	"fmt"
	"os"

	"github.com/go-spatial/maptoolkit/cmd/snap/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
