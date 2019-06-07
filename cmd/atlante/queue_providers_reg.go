package main

import (
	"github.com/go-spatial/maptoolkit/atlante/queuer"
	_ "github.com/go-spatial/maptoolkit/atlante/queuer/awsbatch"
)

func init() {
	cleanupFns = append(cleanupFns, queuer.Cleanup)
}
