package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/atlante/grids/postgresql"
	"github.com/go-spatial/maptoolkit/mbgl"
	"github.com/go-spatial/tegola/dict"
)

var a atlante.Atlante
var mdgid string

const Sheetname = "PostgistDb50k"

func init() {
	flag.StringVar(&mdgid, "mdgid", "V795G25492", "mdgid of the grid")
	provider, err := grids.For(postgresql.Name, dict.Dict{
		// special loopback host: https://docs.docker.com/docker-for-mac/networking/#there-is-no-docker0-bridge-on-macos
		postgresql.ConfigKeyHost:     "docker.for.mac.localhost",
		postgresql.ConfigKeyDB:       "50k",
		postgresql.ConfigKeyUser:     "gdey",
		postgresql.ConfigKeyPassword: "",
	})
	if err != nil {
		panic(err)
	}
	sheet, err := atlante.NewSheet(
		Sheetname,
		provider,
		13,
		"file://styles/topo.json",
		"templates/50k_template.svg",
	)
	if err != nil {
		fmt.Println(grids.Registered())
		panic(err)
	}
	err = a.AddSheet(sheet)
	if err != nil {
		fmt.Println(grids.Registered())
		panic(err)
	}
}

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cancel when we are finished consuming integers
	mbgl.StartSnapshotManager(ctx)

	// San Diego : V795G25492
	filenames, err := a.GeneratePDFMDGID(ctx, "PostgistDb50k", grids.NewMDGID(mdgid), "")
	if err != nil {
		panic(err)
	}
	log.Println("Filenames: ", filenames)
}
