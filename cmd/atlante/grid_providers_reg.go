package main

import (
	// Import various grid providers
	"github.com/go-spatial/atlante/atlante/grids"
	_ "github.com/go-spatial/atlante/atlante/grids/grid5k"
	_ "github.com/go-spatial/atlante/atlante/grids/postgresql"
)

func init() {
	cleanupFns = append(cleanupFns, grids.Cleanup)
}
