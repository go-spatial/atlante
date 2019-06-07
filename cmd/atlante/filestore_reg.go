package main

import (
	// Import various filestores
	"github.com/go-spatial/maptoolkit/atlante/filestore"
	_ "github.com/go-spatial/maptoolkit/atlante/filestore/file"
	_ "github.com/go-spatial/maptoolkit/atlante/filestore/multi"
	_ "github.com/go-spatial/maptoolkit/atlante/filestore/null"
	_ "github.com/go-spatial/maptoolkit/atlante/filestore/s3"
)

func init() {
	cleanupFns = append(cleanupFns, filestore.Cleanup)
}
