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

	"github.com/go-spatial/atlante/atlante/server/coordinator/field"
	"github.com/go-spatial/atlante/atlante/server/coordinator/null"
	"github.com/go-spatial/atlante/atlante/template/grating"
	"github.com/go-spatial/geom"
	"github.com/golang/protobuf/ptypes"

	"github.com/go-spatial/atlante/atlante/server/coordinator"

	"github.com/go-spatial/atlante/atlante/filestore"
	"github.com/go-spatial/atlante/atlante/queuer"

	"github.com/dimfeld/httptreemux"
	"github.com/go-spatial/atlante/atlante"
	"github.com/go-spatial/atlante/atlante/grids"
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

	// ParamsKeyJobID is the key used for the jobid
	ParamsKeyJobID = URLPlaceholder("job_id")

	// HTTPErrorHeader is the name of the X-header where details of the error
	// is provided.
	HTTPErrorHeader = "X-HTTP-Error-Description"

	// JobFileName is the filename  of the sqlite job tracking db
	JobFileName = "jobs.db"

	// MaxJobs returns the max number of jobs to return
	MaxJobs = 100

	GratingNumRowsKey  = "grating-number-of-rows"
	GratingNumColsKey  = "grating-number-of-columns"
	GratingSquarishKey = "grating-not-squarish"
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

		// Coordinator  is a Coordinator Provider for managing jobs
		Coordinator coordinator.Provider

		// DisableNotificationEP will disable the job notification end points from being registered.
		DisableNotificationEP bool
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
		"Access-Control-Allow-Headers": "*",
		"Access-Control-Allow-Methods": "DELETE, GET, POST, PUT, OPTIONS",
	}
)

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

func badRequest(w http.ResponseWriter, reasonFmt string, data ...interface{}) {
	str := fmt.Sprintf(reasonFmt, data...)
	setHeaders(map[string]string{HTTPErrorHeader: str}, w)
	w.WriteHeader(http.StatusBadRequest)
}

func serverError(w http.ResponseWriter, reasonFmt string, data ...interface{}) {
	setHeaders(map[string]string{
		HTTPErrorHeader: fmt.Sprintf(reasonFmt, data...),
	}, w)
	w.WriteHeader(http.StatusInternalServerError)
}

func encodeCellAsJSON(w io.Writer, cell *grids.Cell, pdf filestore.URLInfo, lat, lng *float64, jobs []*coordinator.Job) {
	// Build out the geojson
	const geoJSONFmt = `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{"objectid":"%v"},"geometry":{"type":"Polygon","coordinates":[[[%v,%v],[%v, %v],[%v, %v],[%v, %v],[%v, %v]]]}}]}`
	mdgid := cell.GetMdgid()
	if jobs == nil {
		jobs = []*coordinator.Job{}
	}

	jsonCell := struct {
		MDGID      string             `json:"mdgid"`
		Part       *uint32            `json:"sheet_number"`
		Jobs       []*coordinator.Job `json:"jobs"`
		PDF        string             `json:"pdf_url"`
		LastGen    string             `json:"last_generated"` // RFC 3339 format
		LastEdited string             `json:"last_edited"`    // RFC 3339 format
		EditedBy   string             `json:"edited_by"`
		Series     string             `json:"series"`
		Lat        *float64           `json:"lat"`
		Lng        *float64           `json:"lng"`
		SheetName  string             `json:"sheet_name"`
		GeoJSON    json.RawMessage    `json:"geo_json"`
	}{
		MDGID:     mdgid.Id,
		Jobs:      jobs,
		Lat:       lat,
		Lng:       lng,
		PDF:       pdf.String(),
		LastGen:   pdf.TimeString(),
		Series:    cell.GetSeries(),
		SheetName: cell.GetSheet(),
	}

	if cell.Edited != nil {
		edited := cell.Edited
		at, _ := ptypes.Timestamp(edited.Date)
		jsonCell.LastEdited = at.Format(time.RFC3339)
		jsonCell.EditedBy = edited.By
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
	err := json.NewEncoder(w).Encode(jsonCell)
	if err != nil {
		log.Warnf("failed to encode jsonCell: %v", err)
	}
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

// URLRoot builds a string containing the scheme, host and prot based on a combination of user defined values,
// headers and request parameters. The function is public so it can be overridden for other implementation.
func (s *Server) URLRoot(r *http.Request) string {
	return fmt.Sprintf("%v://%v", s.GetScheme(r), s.GetHostName(r))
}

// GridInfoHandler will write to the writer information about the requested grid. The params should contain a `sheet_name` and either an `mgdid` entry
// or a `lat` and `lng` entries.
// if an non-empty `mgdid` string is there it will be used to attempt to retrieve grid information based on that value.
// if a `lat` and `lng` entries is given the mgdid grid containing that point will be used instead.
// if both are provided, then the mdgid will be used, and the lat/lng keys will be ignored.
// if neither are provided or if the value is bad then a 400 status code will be returned.
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
		pdfURL filestore.URLInfo
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
		log.Infof("Failed to get sheet %v, %v", sheetName, err)
		badRequest(w, "error getting sheet(%v):%v", sheetName, err)
		return
	}

	// check to see if the mdgid key was given.
	if mdgidStr, ok := urlParams[string(ParamsKeyMDGID)]; ok {
		mdgid = grids.NewMDGID(mdgidStr)
		cell, err = sheet.CellForMDGID(mdgid)
		if err != nil {
			if err == grids.ErrNotFound {
				setHeaders(nil, w)
				w.WriteHeader(http.StatusNotFound)
				return
			}
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
			if err == grids.ErrNotFound {
				setHeaders(nil, w)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			badRequest(w, "error getting grid(%v,%v,srid: %v):%v", lat, lng, srid, err)
			return
		}
		latp, lngp = &lat, &lng
	}

	// Figure out the PDF URL
	{
		gf := s.Atlante.FilenamesForCell(sheetName, cell)
		pdfURL, _ = sheet.GetURL(mdgid.AsString(), gf.PDF, false)
	}

	// Ask the coordinator for the status:
	jobs := s.Coordinator.FindByJob(&atlante.Job{SheetName: sheetName, Cell: cell})

	setHeaders(map[string]string{
		"Content-Type":  "application/json",
		"Cache-Control": "no-cache, no-store, must-revalidate",
		"Pragma":        "no-cache",
		"Expires":       "0",
	},
		w)

	encodeCellAsJSON(w, cell, pdfURL, latp, lngp, jobs)
}

// QueueHandler takes a job from a post and queues it on the configured queue
// if the job has not be submitted before
func (s *Server) QueueHandler(w http.ResponseWriter, request *http.Request, urlParams map[string]string) {
	// TODO(gdey): this initial version will not do job tracking.
	// meaning every request to the handler will get a job enqueued into the
	// queueing system.
	if s.Coordinator == nil {
		s.Coordinator = &null.Provider{}
	}

	var (
		ji struct {
			MdgID     *string      `json:"mdgid,omitempty"`
			MdgIDPart uint32       `json:"sheet_number,omitempty"`
			Bounds    *geom.Extent `json:"bounds,omitempty"`
			NumRows   *uint        `json:"number_of_rows,omitempty"`
			NumCols   *uint        `json:"number_of_cols,omitempty"`
			Rectangle bool         `json:"rectangle,omitempty"`
			Srid      uint         `json:"srid,omitempty"`
		}
		err  error
		jobs []*coordinator.Job
	)

	// Get json body
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
	if ji.Bounds == nil && ji.MdgID == nil {
		badRequest(w, "mdgid or bounds must be given")
		return
	}

	if ji.NumRows != nil && (*ji.NumRows < grating.MinRowCol ||
		*ji.NumRows > grating.MaxRowCol) {
		badRequest(w, "number_of_rows need to be between %v and %v", grating.MinRowCol, grating.MaxRowCol)
		return
	}
	if ji.NumCols != nil && (*ji.NumCols < grating.MinRowCol ||
		*ji.NumCols > grating.MaxRowCol) {
		badRequest(w, "number_of_cols need to be between %v and %v", grating.MinRowCol, grating.MaxRowCol)
		return
	}

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
	if ji.Srid == 0 {
		ji.Srid = 4326
	}

	qjob := atlante.Job{
		SheetName: sheetName,
	}

	// We need to figure out what type of information we have to build
	// the cell from.
	if ji.Bounds != nil {
		// Assume bounds first
		qjob.Cell, err = sheet.CellForBounds(*ji.Bounds, ji.Srid)
		if err != nil {
			badRequest(w, "error getting grid(%v):%v", *ji.Bounds, err)
			return
		}
		// bounds based will always queue up a job

	} else {
		mdgid := grids.MDGID{
			Id:   *ji.MdgID,
			Part: ji.MdgIDPart,
		}
		qjob.Cell, err = sheet.CellForMDGID(&mdgid)
		if err != nil {
			badRequest(w, "error getting grid(%v):%v", mdgid.AsString(), err)
			return
		}
		// Check the queue to see if there is already a job with these params:
		jobs = s.Coordinator.FindByJob(&qjob)
	}

	if len(jobs) > 0 {
		// Let's just check the latest job.
		jb := jobs[0]
		switch jb.Status.Status.(type) {
		default:
			// do nothing we should enqueue a
			// new job.
		case field.Requested, field.Started:
			// Job is already there just return
			// info about the old job.
			setHeaders(nil, w)
			if err = json.NewEncoder(w).Encode(jb); err != nil {
				serverError(w, "failed marshal json: %v", err)
			}
			return
		}
	}

	jb, err := s.Coordinator.NewJob(&qjob)
	if err != nil {
		serverError(w, "failed to get new job from coordinator: %v", err)
		return
	}
	// Fill out the Metadata with JobID
	qjob.MetaData = map[string]string{
		"job_id":           jb.JobID,
		GratingSquarishKey: strconv.FormatBool(ji.Rectangle),
	}
	if ji.NumRows != nil {
		// Add row to Metadata
		qjob.MetaData[GratingNumRowsKey] = fmt.Sprintf("%d", *ji.NumRows)
		qjob.MetaData[GratingNumColsKey] = fmt.Sprintf("%d", *ji.NumRows)
	}
	if ji.NumCols != nil {
		// Add col to Metadata
		qjob.MetaData[GratingNumColsKey] = fmt.Sprintf("%d", *ji.NumCols)
		if _, ok := qjob.MetaData[GratingNumRowsKey]; !ok {
			qjob.MetaData[GratingNumRowsKey] = fmt.Sprintf("%d", *ji.NumCols)
		}
	}

	qjobid, err := s.Queue.Enqueue(jb.JobID, &qjob)
	if err != nil {
		s.Coordinator.UpdateField(jb,
			field.Status{field.Failed{
				Description: "Failed to enqueue job",
				Error:       err,
			}},
		)
		badRequest(w, "failed to queue job: %v", err)
		return
	}
	jbData, _ := qjob.Base64Marshal()
	s.Coordinator.UpdateField(jb,
		field.QJobID(qjobid),
		field.JobData(jbData),
		field.Status{field.Requested{}},
	)

	setHeaders(nil, w)
	if err = json.NewEncoder(w).Encode(jb); err != nil {
		serverError(w, "failed marshal json: %v", err)
		return
	}
}

// SheetInfoHandler takes a job from a post and enqueue it on the configured queue
// if the job has not be submitted before
func (s *Server) SheetInfoHandler(w http.ResponseWriter, request *http.Request, urlParams map[string]string) {
	type sheetInfo struct {
		Name  string `json:"name"`
		Desc  string `json:"desc"`
		Scale uint   `json:"scale"`
	}
	type sheetsDef struct {
		Sheets []sheetInfo `json:"sheets"`
	}
	var newSheets sheetsDef
	sheets := s.Atlante.Sheets()
	newSheets.Sheets = make([]sheetInfo, 0, len(sheets))
	for _, sh := range sheets {
		newSheets.Sheets = append(newSheets.Sheets, sheetInfo{
			Name:  sh.Name,
			Desc:  sh.Desc,
			Scale: sh.Scale,
		})
	}

	setHeaders(nil, w)
	err := json.NewEncoder(w).Encode(newSheets)
	if err != nil {
		serverError(w, "failed to marshal json: %v", err)
		return
	}
}

// JobInfoHandler is a http handler for information about a job.
func (s *Server) JobInfoHandler(w http.ResponseWriter, request *http.Request, urlParams map[string]string) {

	jobid, ok := urlParams[string(ParamsKeyJobID)]
	if !ok {
		// We need a sheetnumber.
		badRequest(w, "missing job_id")
		return
	}
	job, ok := s.Coordinator.FindByJobID(jobid)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if job.AJob != nil {
		sheetName := job.SheetName
		sheetName = s.Atlante.NormalizeSheetName(sheetName, false)
		sheet, err := s.Atlante.SheetFor(sheetName)
		if err == nil {
			// let's see if we can fill out the pdf and LastGen parts
			mdgid := job.MdgID
			gf := s.Atlante.FilenamesForCell(sheetName, job.AJob.Cell)
			if pdfURL, ok := sheet.GetURL(mdgid, gf.PDF, false); ok {
				job.PDF = pdfURL.String()
				job.LastGen = pdfURL.TimeString()
			}
		}
	}

	setHeaders(nil, w)
	if err := json.NewEncoder(w).Encode(job); err != nil {
		serverError(w, "failed to marshal json: %v", err)
	}

}

// JobsHandler is a http handler for the jobs end-point
func (s *Server) JobsHandler(w http.ResponseWriter, request *http.Request, urlParams map[string]string) {
	// Hardcode 100 limit for now.
	jobs, err := s.Coordinator.Jobs(MaxJobs)
	if err != nil {
		serverError(w, "failed to get jobs: %v", err)
		return
	}
	// Make sure we always encode an empty array.
	if jobs == nil {
		jobs = []*coordinator.Job{}
	}

	for i := range jobs {
		if jobs[i] == nil || jobs[i].AJob == nil {
			continue
		}
		sheetName := jobs[i].SheetName
		sheetName = s.Atlante.NormalizeSheetName(sheetName, false)
		sheet, err := s.Atlante.SheetFor(sheetName)
		if err != nil {
			continue
		}
		// let's see if we can fill out the pdf and LastGen parts
		mdgid := jobs[i].MdgID
		gf := s.Atlante.FilenamesForCell(sheetName, jobs[i].AJob.Cell)
		if pdfURL, ok := sheet.GetURL(mdgid, gf.PDF, false); ok {
			jobs[i].PDF = pdfURL.String()
			jobs[i].LastGen = pdfURL.TimeString()
		}
	}
	setHeaders(nil, w)
	err = json.NewEncoder(w).Encode(jobs)
	if err != nil {
		serverError(w, "failed to marshal json: %v", err)
	}
}

// NotificationHandler is an http handle for worker job progress notifications
func (s *Server) NotificationHandler(w http.ResponseWriter, request *http.Request, urlParams map[string]string) {

	jobid, ok := urlParams[string(ParamsKeyJobID)]
	if !ok {
		// We need a sheetnumber.
		log.Infof("Missing job_id: %v", urlParams)
		badRequest(w, "missing job_id")
		return
	}

	job, ok := s.Coordinator.FindByJobID(jobid)
	if !ok {
		setHeaders(nil, w)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Get json body
	bdy, err := ioutil.ReadAll(request.Body)
	request.Body.Close()
	if err != nil {
		badRequest(w, "error reading body")
		return
	}

	var si field.Status
	err = json.Unmarshal(bdy, &si)
	if err != nil {
		badRequest(w, "unable to unmarshal json: %v", err)
		return
	}

	if err := s.Coordinator.UpdateField(job, si); err != nil {
		serverError(w, "failed to update job %v: %v", jobid, err)
	}
	setHeaders(nil, w)
	w.WriteHeader(http.StatusNoContent)
}

// HealthCheckHandler is an http handler use to indicate the health of the server.
// TODO(gdey): make the handler more intelligent as we understand more about the environment.
func (*Server) HealthCheckHandler(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
	// We always return a 200 while we are able to serve requests.
	setHeaders(nil, w)
	w.WriteHeader(http.StatusOK)
	return
}

// corsHanlder is used to respond to all OPTIONS requests for registered routes
func corsHandler(w http.ResponseWriter, r *http.Request, params map[string]string) {
	setHeaders(map[string]string{}, w)
	return
}

// RegisterRoutes setup the routes
func (s *Server) RegisterRoutes(r *httptreemux.TreeMux) {

	r.OptionsHandler = corsHandler

	r.GET("/status", s.HealthCheckHandler)
	log.Infof("registering: GET  /status")
	r.GET("/sheets", s.SheetInfoHandler)
	log.Infof("registering: GET  /sheets")
	group := r.NewGroup(GenPath("sheets", ParamsKeySheetname))
	log.Infof("registering: GET  /sheets/:sheetname/info/:lng/:lat")
	group.GET(GenPath("info", ParamsKeyLng, ParamsKeyLat), s.GridInfoHandler)
	log.Infof("registering: GET  /sheets/:sheetname/info/mdgid/:mdgid")
	group.GET(GenPath("info", "mdgid", ParamsKeyMDGID), s.GridInfoHandler)
	if s.Queue != nil {
		log.Infof("registering: POST /sheets/:sheetname/mdgid")
		group.POST("/mdgid", s.QueueHandler)
		log.Infof("registering: POST /sheets/:sheetname/bounds")
		group.POST("/bounds", s.QueueHandler)
	}

	log.Infof("registering: GET  /jobs")
	r.GET("/jobs", s.JobsHandler)
	jobsGroup := r.NewGroup(GenPath("jobs", ParamsKeyJobID))
	log.Infof("registering: GET  /jobs/:jobid/status")
	jobsGroup.GET("/status", s.JobInfoHandler)
	if !s.DisableNotificationEP {
		log.Infof("registering: POST  /jobs/:jobid/status")
		jobsGroup.POST("/status", s.NotificationHandler)
	}
}
