package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-spatial/maptoolkit/atlante/queuer"

	"github.com/dimfeld/httptreemux"
	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/prometheus/common/log"
)

// URLPlaceholder allows us to define placeholder that cna be used by
// genPath to construct the final url string.
type URLPlaceholder string

// PathComponent return the placeholder with a ':' prepended
func (u URLPlaceholder) PathComponent() string { return ":" + string(u) }

const (
	// ParamsKeyMDGID is the key used for the mdgid
	ParamsKeyMDGID = URLPlaceholder("mdgid")
	// ParamsKeyLat is the key used for the lat
	ParamsKeyLat = URLPlaceholder("lat")
	// ParamsKeyLng is the key used for the lng
	ParamsKeyLng = URLPlaceholder("lng")
	// ParamsKeySheetname is the key used for the sheetname
	ParamsKeySheetname = URLPlaceholder("sheetname")

	// HTTPErrorHeader is the name of the X-header where details of the error
	// is provided.
	HTTPErrorHeader = "X-HTTP-Error-Description"

	// JobFileName is the filename  of the sqlite job tracking db
	JobFileName = "jobs.db"
)

// GenPath take a set of compontents and constructs a url
// string for treemux
func GenPath(paths ...interface{}) string {
	var path strings.Builder
	for _, p := range paths {
		var str string
		switch pp := p.(type) {
		case URLPlaceholder:
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
	return gp
}

type (
	jobItem struct {
		ID    int    `json:"-"`
		JobID string `json:"job_id"`
		// QJobID is the job id returned by the queue when
		// the item was enqueued
		QJobID    string    `json:"-"`
		MdgID     string    `json:"mdgid"`
		MdgIDPart uint32    `json:"sheet_number,omitempty"`
		Status    string    `json:"status,omitempty"`
		EnquedAt  time.Time `json:"enqued_at,omitempty"`
		UpdatedAt time.Time `json:"updated_at,omitempty"`
	}

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

		// Queue is a QueueProvider that is configured for this server
		Queue queuer.Provider

		// jobsDB is the database (sqlite) containing the jobs we have sent to be processed
		// this is for job tracking
		jobsDB *sql.DB
	}
)

var (
	// Version is the version of the software, this should be set by the main program, before starting up.
	// It is used by various Middleware to determine the version.
	Version = "Version Not Set"

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
	mdgid := cell.GetMdgid()

	jsonCell := struct {
		MDGID      string          `json:"mdgid"`
		Part       *uint32         `json:"sheet_number,omitempty"`
		PDF        string          `json:"pdf_url,omitempty"`
		LastGen    string          `json:"last_generated,omitempty"` // RFC 3339 format
		LastEdited string          `json:"last_edited,omitempty"`    // RFC 3339 format
		Series     string          `json:"series"`
		Lat        *float64        `json:"lat,omitempty"`
		Lng        *float64        `json:"lng,omitempty"`
		SheetName  string          `json:"sheet_name"`
		GeoJSON    json.RawMessage `json:"geo_json"`
	}{
		MDGID:     mdgid.Id,
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
	if mdgid.Part != 0 {
		jsonCell.Part = &mdgid.Part
	}

	mdgidStr := mdgid.AsString()
	sw := cell.GetSw()
	ne := cell.GetNe()
	jsonCell.GeoJSON = json.RawMessage(
		fmt.Sprintf(geoJSONFmt,
			adler32.Checksum([]byte(mdgidStr)), // Just need a stable unique number, a checksum will do.
			sw.GetLng(), sw.GetLat(),
			sw.GetLng(), ne.GetLat(),
			ne.GetLng(), ne.GetLat(),
			ne.GetLng(), sw.GetLat(),
			sw.GetLng(), sw.GetLat(),
		),
	)
	// Encoding the cell into the json
	json.NewEncoder(w).Encode(jsonCell)
}

// CreateJobDB will create the job tracking database
func CreateJobDB(filename string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	statement, err := database.Prepare(`
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY, 
		job_id TEXT, 
		mdgid TEXT,
		status TEXT,
		enqued_at TEXT,
		updated_at TEXT,
		qjob_id TEXT,
		jobobj TEXT
	)
`)
	if err != nil {
		return nil, err
	}
	statement.Exec()
	statement, err = database.Prepare(`CREATE UNIQUE INDEX idx_job_id ON jobs (job_id);`)
	if err != nil {
		return nil, err
	}
	statement.Exec()
	statement, err = database.Prepare(`CREATE UNIQUE INDEX idx_qjob_id ON jobs (qjob_id);`)
	if err != nil {
		return nil, err
	}
	statement.Exec()
	return database, nil
}

func (s *Server) findJobItem(mdgidstr string) (jbs []jobItem, err error) {
	database := s.jobsDB
	if database == nil {
		return nil, nil
	}
	mdgid := grids.NewMDGID(mdgidstr)
	const selectSQL = `
	SELECT  
		id,
		job_id,
		status,
		enqued_at,
		updated_at
	FROM 
		jobs
	WHERE 
		mdgid=?
	ORDER BY id DESC
	`
	rows, err := database.Query(selectSQL, mdgid.AsString())
	if err != nil {
		return jbs, err
	}

	for rows.Next() {
		var enquedat, updatedat string
		ji := jobItem{MdgID: mdgid.Id}
		if mdgid.Part != 0 {
			ji.MdgIDPart = mdgid.Part
		}
		err = rows.Scan(
			&ji.ID,
			&ji.JobID,
			&ji.Status,
			&enquedat,
			&updatedat,
		)
		if err != nil {
			return []jobItem{}, err
		}
		ji.EnquedAt, err = time.Parse(time.RFC3339, enquedat)
		if err != nil {
			log.Warnf("failed to parse enqued at from db: %v for id: %v", enquedat, ji.ID)
		}
		ji.UpdatedAt, err = time.Parse(time.RFC3339, updatedat)
		if err != nil {
			log.Warnf("failed to parse updated at from db: %v for id: %v", updatedat, ji.ID)
		}
		jbs = append(jbs, ji)
	}
	return jbs, nil
}

// TODO(gdey): job management
func (s *Server) addJobItem(ji *jobItem) (*jobItem, error) {
	database := s.jobsDB
	if database == nil {
		return nil, fmt.Errorf("jobDB not initilized")
	}
	const intertSQL = `
	INSERT INTO jobs (job_id, status, ) VALUES (?, ?)
	`
	return nil, nil
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

	const (
		srid = 4326
	)
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

	sheetName, ok := urlParams[string(ParamsKeySheetname)]
	if !ok {
		// We need a sheetnumber.
		badRequest(w, "missing sheet name)")
		return
	}

	sheetName = s.Atlante.NormalizeSheetName(sheetName, false)

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

		// TODO(gdey): make srid for server configurable? should this come through
		// the url?
		cell, err = sheet.CellForLatLng(lat, lng, srid)
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

// QueueHandler takes a job from a post and enqueues it on the configured queue
// if the job has not be submitted before
func (s *Server) QueueHandler(w http.ResponseWriter, request *http.Request, urlParams map[string]string) {
	// TODO(gdey): this initial version will not do job tracking.
	// meaning every request to the handler will get a job enqued into the
	// queueing system.

	// Get json body
	var ji jobItem
	bdy, err := ioutil.ReadAll(request.Body)
	request.Body.Close()
	if err != nil {
		badRequest(w, "error reading body")
		return
	}
	err = json.Unmarshal(bdy, &ji)
	if err != nil {
		badRequest(w, "unable to unmarshal json: %v", err)
		return
	}
	mdgid := grids.MDGID{
		Id:   ji.MdgID,
		Part: ji.MdgIDPart,
	}

	sheetName, ok := urlParams[string(ParamsKeySheetname)]
	if !ok {
		// We need a sheetnumber.
		badRequest(w, "missing sheet name)")
		return
	}

	sheetName = s.Atlante.NormalizeSheetName(sheetName, false)

	// ji is now going to be what get's returned
	ji.EnquedAt = time.Now()
	ji.JobID = fmt.Sprintf("%v:%v", sheetName, mdgid.AsString())

	sheet, err := s.Atlante.SheetFor(sheetName)
	if err != nil {
		badRequest(w, "error getting sheet(%v):%v", sheetName, err)
		return
	}
	cell, err := sheet.CellForMDGID(&mdgid)
	if err != nil {
		badRequest(w, "error getting grid(%v):%v", mdgid.AsString(), err)
		return
	}

	qjob := atlante.Job{
		Cell:      cell,
		SheetName: sheetName,
		MetaData: map[string]string{
			"job_id": ji.JobID,
		},
	}
	// TODO(gdye):Ignoring the returned job id for now. Will need it when we
	// have the job management
	_, err = s.Queue.Enqueue(ji.JobID, &qjob)
	if err != nil {
		badRequest(w, "failed to queue job: %v", err)
		return
	}

	err = json.NewEncoder(w).Encode(ji)
	if err != nil {
		badRequest(w, "failed marshal json: %v", err)
		return
	}
}

// RegisterRoutes setup the routes
func (s *Server) RegisterRoutes(r *httptreemux.TreeMux) {

	group := r.NewGroup(GenPath(ParamsKeySheetname))
	log.Infof("registering: GET  /:sheetname/info/:lat/:lng")
	group.GET(GenPath("info", ParamsKeyLat, ParamsKeyLng), s.GridInfoHandler)
	log.Infof("registering: GET  /:sheetname/info/:mdgid")
	group.GET(GenPath("info", "mdgid", ParamsKeyMDGID), s.GridInfoHandler)
	if s.Queue != nil {
		log.Infof("registering: POST /:sheetname/mdgid")
		group.POST("/mdgid", s.QueueHandler)
	}
}
