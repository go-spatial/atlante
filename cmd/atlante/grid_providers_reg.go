package main

import (
	// Import various grid providers
	"github.com/go-spatial/maptoolkit/atlante/grids"
	_ "github.com/go-spatial/maptoolkit/atlante/grids/postgresql"
	_ "github.com/go-spatial/maptoolkit/atlante/grids/grid5k"
)

func init() {
	cleanupFns = append(cleanupFns, grids.Cleanup)
}
