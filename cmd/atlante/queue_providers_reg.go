package main

import (
	"github.com/go-spatial/atlante/atlante/queuer"
	_ "github.com/go-spatial/atlante/atlante/queuer/awsbatch"
)

func init() {
	cleanupFns = append(cleanupFns, queuer.Cleanup)
}
