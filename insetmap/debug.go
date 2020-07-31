package insetmap

import (
	"log"
	"os"
	"strconv"
)

var debug bool

func init() {
	debug, _ = strconv.ParseBool(os.Getenv("INSETMAP_DEBUG"))
	if debug {
		log.Printf("[INFO] debug is on\n")
	}
}
