package server

import (
	"encoding/json"
	"fmt"
	"hash/adler32"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dimfeld/httptreemux"
	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/prometheus/common/log"
)

type URLPath string

func (u URLPath) PathComponent() string { return ":" + string(u) }

const (
	// ParamsKeyMDGID is the key used for the mdgid
	ParamsKeyMDGID = URLPath("mdgid")
	// ParamsKeyLat is the key used for the lat
	ParamsKeyLat = URLPath("lat")
	// ParamsKeyLng is the key used for the lng
	ParamsKeyLng = URLPath("lng")
	// ParamsKeySheetname is the key used for the sheetname
	ParamsKeySheetname = URLPath("sheetname")

	HTTPErrorHeader = "X-HTTP-Error-Description"
)

func GenPath(paths ...interface{}) string {
	var path strings.Builder
	for _, p := range paths {
		var str string
		switch pp := p.(type) {
		case URLPath:
			str = pp.PathComponent()
		case string:
			str = pp
		case rune:
			str = string(pp)
		default:
			if pp == nil {
				continue
			}
			str = fmt.Sprintf("%v", p)
		}
		if str == "/" || str == "" {
			continue
		}
		path.WriteString("/" + str)
	}
	gp := path.String()
	log.Warnf("Sending path: %v", gp)
	return gp
}

type (
	// Server is used to serve up grid information, and generate print jobs
	Server struct {
		// HostName is the name of the host to use for construction of URLS.
		Hostname string

		// Port is the port the server is listening on, used for construction of URLS.
		Port string

		// Scheme is the scheme that should be used for construction of URLs.
		Scheme string

		// Headers is the map of user defined response headers.
		Headers map[string]string

		// Atlante is the Atlante object that containts the providers and file_stores
		Atlante *atlante.Atlante
	}
)

var (
	// Version is the version of the software, this should be set by the main program, before starting up.
	// It is used by various Middleware to determine the version.
	Version string = "Version Not Set"

	// HostName is the name of the host to use for construction of URLS.
	HostName string

	// Port is the port the server is listening on, used for construction of URLS.
	Port string

	// DefaultHeaders is the map of user defined response headers.
	DefaultHeaders = http.Header{}

	// DefaultCORSHeaders define the default CORS response headers added to all requests
	DefaultCORSHeaders = map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, OPTIONS",
	}
)

/*
ooooo   ooooo                           .o8                            .oooooo.    .o8           o8o                         .
`888'   `888'                          "888                           d8P'  `Y8b  "888           `"'                       .o8
 888     888   .ooooo.   .oooo.    .oooo888   .ooooo.  oooo d8b      888      888  888oooo.     oooo  .ooooo.   .ooooo.  .o888oo
 888ooooo888  d88' `88b `P  )88b  d88' `888  d88' `88b `888""8P      888      888  d88' `88b    `888 d88' `88b d88' `"Y8   888
 888     888  888ooo888  .oP"888  888   888  888ooo888  888          888      888  888   888     888 888ooo888 888         888
 888     888  888    .o d8(  888  888   888  888    .o  888          `88b    d88'  888   888     888 888    .o 888   .o8   888 .
o888o   o888o `Y8bod8P' `Y888""8o `Y8bod88P" `Y8bod8P' d888b          `Y8bood8P'   `Y8bod8P'     888 `Y8bod8P' `Y8bod8P'   "888"
                                                                                                 888
                                                                                             .o. 88P
																							 `Y888P
*/

func setHeaders(h map[string]string, w http.ResponseWriter) {
	// add CORS headers
	for name, val := range DefaultCORSHeaders {
		if val == "" {
			log.Warnf("default CORS header (%v) has no value", name)
		}
		w.Header().Set(name, val)
	}

	// set user defined headers
	for name, val := range h {
		if val == "" {
			log.Warnf("header (%v) has no value", name)
		}
		w.Header().Set(name, val)
	}
}

/*
 .oooooo..o                                                          .oooooo.    .o8           o8o                         .
d8P'    `Y8                                                         d8P'  `Y8b  "888           `"'                       .o8
Y88bo.       .ooooo.  oooo d8b oooo    ooo  .ooooo.  oooo d8b      888      888  888oooo.     oooo  .ooooo.   .ooooo.  .o888oo
 `"Y8888o.  d88' `88b `888""8P  `88.  .8'  d88' `88b `888""8P      888      888  d88' `88b    `888 d88' `88b d88' `"Y8   888
     `"Y88b 888ooo888  888       `88..8'   888ooo888  888          888      888  888   888     888 888ooo888 888         888
oo     .d8P 888    .o  888        `888'    888    .o  888          `88b    d88'  888   888     888 888    .o 888   .o8   888 .
8""88888P'  `Y8bod8P' d888b        `8'     `Y8bod8P' d888b          `Y8bood8P'   `Y8bod8P'     888 `Y8bod8P' `Y8bod8P'   "888"
                                                                                               888
                                                                                           .o. 88P
																						   `Y888P
*/

func badRequest(w http.ResponseWriter, reasonFmt string, data ...interface{}) {
	w.Header().Set(HTTPErrorHeader, fmt.Sprintf(reasonFmt, data...))
	w.WriteHeader(http.StatusBadRequest)
}

func encodeCellAsJSON(w io.Writer, cell *grids.Cell, pdf string, lat, lng *float64, lastGen time.Time) {
	// Build out the geojson
	const geoJSONFmt = `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{"objectid":"%v"},"geometry":{"type":"Polygon","coordinates":[[[%v,%v],[%v, %v],[%v, %v],[%v, %v],[%v, %v]]]}}]}`

	jsonCell := struct {
		MDGID      string          `json:"mdgid"`
		part       *int            `json:"sheet_number,omitempty"`
		PDF        string          `json:"pdf_url,omitempty"`
		LastGen    string          `json:"last_generated,omitempty"` // RFC 3339 format
		LastEdited string          `json:"last_edited,omitempty"`    // RFC 3339 format
		Series     string          `json:"series"`
		Lat        *float64        `json:"lat,omitempty"`
		Lng        *float64        `json:"lng,omitempty"`
		SheetName  string          `json:"sheet_name"`
		GeoJSON    json.RawMessage `json:"geo_json"`
	}{
		MDGID:     cell.GetMdgid().Id,
		LastGen:   lastGen.Format(time.RFC3339),
		Lat:       lat,
		Lng:       lng,
		PDF:       pdf,
		Series:    cell.GetSeries(),
		SheetName: cell.GetSheet(),
	}
	pubdate, err := cell.PublicationDate()
	if err != nil && !pubdate.IsZero() {
		jsonCell.LastEdited = pubdate.Format(time.RFC3339)
	}

	mdgidStr := cell.GetMdgid().AsString()
	sw := cell.SW()
	ne := cell.NE()
	jsonCell.GeoJSON = json.RawMessage(
		fmt.Sprintf(geoJSONFmt,
			adler32.Checksum([]byte(mdgidStr)), // Just need a stable unique number, a checksum will do.
			sw[1], sw[0],
			sw[1], ne[0],
			ne[1], ne[0],
			ne[1], sw[0],
			sw[1], sw[0],
		),
	)
	// Encoding the cell into the json
	json.NewEncoder(w).Encode(jsonCell)
}

// GetHostName returns determines the hostname:port to return based on the following hierarchy
// Hostname/Port in the server object.
// the host/port in the request object.
func (s *Server) GetHostName(r *http.Request) string {
	var (
		rHostname = s.Hostname
		rPost     = s.Port
	)

	if rHostname == "" {
		substrs := strings.Split(r.Host, ":")
		switch len(substrs) {
		case 1:
			rHostname = substrs[0]
		case 2:
			rHostname = substrs[0]
			if rPost == "" || rPost == "none" {
				rPost = substrs[1]
			}
		default:
			log.Warnf("multiple colons (':') in host string: %v", r.Host)
		}
	}

	if rPost == "" || rPost == "none" {
		return rHostname
	}
	return rHostname + ":" + rPost
}

// GetScheme checks to determine if the request is http or https. The scheme is needed for the proxy
// url and pdf url.
func (s *Server) GetScheme(r *http.Request) string {
	switch {
	case r.Header.Get("X-Forwarded-Proto") != "":
		return r.Header.Get("X-Forwarded-Proto")
	case r.TLS != nil:
		return "https"
	case s.Scheme != "":
		return s.Scheme
	default:
		return "http"
	}
}

// URLRoot builds a stirng containing the scheme, host and prot based on a combination of user defined values,
// headers and request parameters. The funciton is public so it can be overridden for other implementation.
func (s *Server) URLRoot(r *http.Request) string {
	return fmt.Sprintf("%v://%v", s.GetScheme(r), s.GetHostName(r))
}

// GridInfoHandler will writer to the writer information about the requested grid. The params should contain a `sheet_name` and either an `mgdid` entry
// or a `lat` and `lng` entries.
// if an non-empty `mgdid` string is there it will be used to attempt to retrieve grid information based on that value.
// if a `lat` and `lng` entries is given the mgdid grid containing that point will be used instead.
// if both are provided, then the mdgid will be used, and the lat/lng keys will be ignored.
// if neigher are provided or if the value is bad then a 400 status code will be returned.
// Requires
func (s *Server) GridInfoHandler(w http.ResponseWriter, request *http.Request, urlParams map[string]string) {

	var (
		mdgid      *grids.MDGID
		cell       *grids.Cell
		err        error
		latp, lngp *float64

		// We will fill this out later
		pdfURL string
		// We will get this from the filestore
		lastGen time.Time
	)

	log.Warnf("Got a call to GridInfoHandler: %v", urlParams)

	sheetName, ok := urlParams[string(ParamsKeySheetname)]
	if !ok {
		// We need a sheetnumber.
		badRequest(w, "missing sheet name)")
		return
	}

	sheet, err := s.Atlante.SheetFor(sheetName)
	if err != nil {
		badRequest(w, "error getting sheet(%v):%v", sheetName, err)
		return
	}

	// check to see if the mdgid key was given.
	if mdgidStr, ok := urlParams[string(ParamsKeyMDGID)]; ok {
		mdgid = grids.NewMDGID(mdgidStr)
		cell, err = sheet.CellForMDGID(mdgid)
		if err != nil {
			badRequest(w, "error getting grid(%v):%v", mdgidStr, err)
			return
		}
	} else {
		// check to see if the lat/lng is there.
		latstr, ok := urlParams[string(ParamsKeyLat)]
		if !ok {
			badRequest(w, "missing lat or mdgid")
			return
		}
		lat, err := strconv.ParseFloat(latstr, 64)
		if err != nil {
			badRequest(w, "error converting lat(%v):%v", latstr, err)
			return
		}

		lngstr, ok := urlParams[string(ParamsKeyLng)]
		if !ok {
			badRequest(w, "missing lng or mdgid")
			return
		}
		lng, err := strconv.ParseFloat(lngstr, 64)
		if err != nil {
			badRequest(w, "error converting lng(%v):%v", lngstr, err)
			return
		}

		cell, err = sheet.CellForLatLng(lat, lng, 3857)
		if err != nil {
			badRequest(w, "error getting grid(%v,%v):%v", lat, lng, err)
			return
		}
		latp, lngp = &lat, &lng
	}
	// content type
	w.Header().Add("Content-Type", "application/json")

	// cache control headers (no-cache)
	w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Add("Pragma", "no-cache")
	w.Header().Add("Expires", "0")

	encodeCellAsJSON(w, cell, pdfURL, latp, lngp, lastGen)
}

// RegisterRoutes setup the routes
func (s *Server) RegisterRoutes(r *httptreemux.TreeMux) {

	group := r.NewGroup(GenPath(ParamsKeySheetname))
	group.GET(GenPath("info", ParamsKeyLat, ParamsKeyLng), s.GridInfoHandler)
	group.GET(GenPath("info", "mdgid", ParamsKeyMDGID), s.GridInfoHandler)
}
