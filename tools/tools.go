// +build tools

// tools is a dummy package that will be ignored for builds, but included for dependencies
package tools

import (

	// used for go generate
	_ "github.com/golang/protobuf/protoc-gen-go"

	// used for Docker files and template file generation
	_ "github.com/gdey/bastet/cmd/bastet"
)
