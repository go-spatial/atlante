package main

import (
	// Import various grid providers
	"github.com/go-spatial/maptoolkit/atlante/grids"
	_ "github.com/go-spatial/maptoolkit/atlante/grids/postgresql"
)

func init() {
	cleanupFns = append(cleanupFns, grids.Cleanup)
}
