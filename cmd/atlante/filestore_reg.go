package main

import (
	// Import various filestores
	_ "github.com/go-spatial/maptoolkit/atlante/filestore/file"
	_ "github.com/go-spatial/maptoolkit/atlante/filestore/multi"
	_ "github.com/go-spatial/maptoolkit/atlante/filestore/null"
	_ "github.com/go-spatial/maptoolkit/atlante/filestore/s3"
)