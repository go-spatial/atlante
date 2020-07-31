package main

import (
	// Import various filestores
	"github.com/go-spatial/atlante/atlante/filestore"
	_ "github.com/go-spatial/atlante/atlante/filestore/file"
	_ "github.com/go-spatial/atlante/atlante/filestore/multi"
	_ "github.com/go-spatial/atlante/atlante/filestore/null"
	_ "github.com/go-spatial/atlante/atlante/filestore/s3"
)

func init() {
	cleanupFns = append(cleanupFns, filestore.Cleanup)
}
