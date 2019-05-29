package main

import (
	"fmt"
	"os"

	"github.com/go-spatial/maptoolkit/cmd/atlante/cmd"
)

func main() {
	if err := cmd.Root.Execute(); err != nil {
		fmt.Println("got the following error:", err)
		os.Exit(1)
	}
}
