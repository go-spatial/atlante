package postgresql

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/encoding/wkt"
	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"
	"github.com/prometheus/common/log"

	"github.com/go-spatial/tegola"
	"github.com/jackc/pgx"
)

// TYPE is the name of the provider type
const TYPE = "postgresql"

// AppName is shown by the pqclient
var AppName = "atlante"

// Provider implements the Grid.Provider interface
type Provider struct {
	config                    pgx.ConnPoolConfig
	pool                      *pgx.ConnPool
	srid                      uint
	QueryNewJob               string
	QueryUpdateQueueJobID     string
	QueryUpdateJobData        string
	QueryInsertStatus         string
	QuerySelectMDGIDSheetName string
	QuerySelectJobID          string
	QuerySelectAllJobs        string
}

const (
	// DefaultSRID is the assumed srid of data unless specified
	DefaultSRID = tegola.WebMercator
	// DefaultPort is the default port for postgis
	DefaultPort = 5432
	// DefaultMaxConn is the max number of connections to attempt
	DefaultMaxConn = 100
	// DefaultSSLMode by default ssl is disabled
	DefaultSSLMode = "disable"
	// DefaultSSLKey by default is empty
	DefaultSSLKey = ""
	// DefaultSSLCert by default is empty
	DefaultSSLCert = ""
	// DefaultEditDateFormat the time format to expect
	DefaultEditDateFormat = time.RFC3339
	// DefaultEditBy who edited the content if not provided
	DefaultEditBy = ""
)

const (
	// ConfigKeyHost is the config key for the postgres host
	ConfigKeyHost = "host"
	// ConfigKeyPort is the config key for the postgres port
	ConfigKeyPort = "port"
	// ConfigKeyDB is the config key for the postgres db
	ConfigKeyDB = "database"
	// ConfigKeyUser is the config key for the postgres user
	ConfigKeyUser = "user"
	// ConfigKeyPassword is the config key for the postgres user's password
	ConfigKeyPassword = "password"
	// ConfigKeySSLMode is the config key for the postgres SSL
	ConfigKeySSLMode = "ssl_mode"
	// ConfigKeySSLKey is the config key for the postgres SSL
	ConfigKeySSLKey = "ssl_key"
	// ConfigKeySSLCert is the config key for the postgres SSL
	ConfigKeySSLCert = "ssl_cert"
	// ConfigKeySSLRootCert is the config key for the postgres SSL
	ConfigKeySSLRootCert = "ssl_root_cert"
	// ConfigKeyMaxConn is the max number of connections to keep in the pool
	ConfigKeyMaxConn = "max_connections"
	// ConfigKeySRID is the srid of the data
	ConfigKeySRID = "srid"
)

// ErrInvalidSSLMode is returned when something is wrong with SSL configuration
type ErrInvalidSSLMode string

func (e ErrInvalidSSLMode) Error() string {
	return fmt.Sprintf("postgis: invalid ssl mode (%v)", string(e))
}

// ErrInvalidStatus is returned when a job entry as not have a valid status. Either
// the status is missing or the status type is unknown
type ErrInvalidStatus struct {
	Job    int
	Status string
	Desc   string
	Err    error
}

func (e ErrInvalidStatus) Error() string {
	if e.Status == "" {
		return fmt.Sprintf("postgis: invalid status for job %d", e.Job)
	}
	return fmt.Sprintf("postgis: invalid status (%v) for job %d", e.Status, e.Job)
}

// ErrNoQueueID is returned when a job does not have a queue id associated with it
type ErrNoQueueID int

func (e ErrNoQueueID) Error() string {
	return fmt.Sprintf("postgis: no queueid for job %d", int(e))
}

func init() {
	coordinator.Register(TYPE, initFunc, cleanup)
}

// rowScanner will scan the values into the given dest parameters
type rowScanner interface {
	//	Scan reads the values from the current row into dest values positionally.
	// dest can include pointers to core types, values implementing the Scanner
	// interface, []byte, and nil. []byte will skip the decoding process and directly
	// copy the raw bytes received.
	Scan(dest ...interface{}) error
}

// initFunc returns a new provider based on the postgresql database
func initFunc(config coordinator.Config) (coordinator.Provider, error) {
	var emptystr string

	host, err := config.String(ConfigKeyHost, nil)
	if err != nil {
		return nil, err
	}

	db, err := config.String(ConfigKeyDB, nil)
	if err != nil {
		return nil, err
	}

	user, err := config.String(ConfigKeyUser, nil)
	if err != nil {
		return nil, err
	}

	password, err := config.String(ConfigKeyPassword, nil)
	if err != nil {
		return nil, err
	}

	sslmode := DefaultSSLMode
	sslmode, err = config.String(ConfigKeySSLMode, &sslmode)

	sslkey := DefaultSSLKey
	sslkey, err = config.String(ConfigKeySSLKey, &sslkey)
	if err != nil {
		return nil, err
	}

	sslcert := DefaultSSLCert
	sslcert, err = config.String(ConfigKeySSLCert, &sslcert)
	if err != nil {
		return nil, err
	}

	sslrootcert := DefaultSSLCert
	sslrootcert, err = config.String(ConfigKeySSLRootCert, &sslrootcert)
	if err != nil {
		return nil, err
	}

	port := DefaultPort
	if port, err = config.Int(ConfigKeyPort, &port); err != nil {
		return nil, err
	}

	maxcon := DefaultMaxConn
	if maxcon, err = config.Int(ConfigKeyMaxConn, &maxcon); err != nil {
		return nil, err
	}

	srid := DefaultSRID
	if srid, err = config.Int(ConfigKeySRID, &srid); err != nil {
		return nil, err
	}

	connConfig := pgx.ConnConfig{
		Host:     host,
		Port:     uint16(port),
		Database: db,
		User:     user,
		Password: password,
		LogLevel: pgx.LogLevelWarn,
		RuntimeParams: map[string]string{
			//			"default_transaction_read_only": "TRUE",
			"application_name": AppName,
		},
	}

	err = ConfigTLS(sslmode, sslkey, sslcert, sslrootcert, &connConfig)
	if err != nil {
		return nil, err
	}

	p := Provider{
		config: pgx.ConnPoolConfig{
			ConnConfig:     connConfig,
			MaxConnections: int(maxcon),
		},
		srid: uint(srid),
	}
	if p.pool, err = pgx.NewConnPool(p.config); err != nil {
		return nil, fmt.Errorf("Failed while creating connection pool: %v", err)
	}

	// Check and step up all the queries.
	p.QueryNewJob, _ = config.String("query_new_job", &emptystr)
	p.QueryUpdateQueueJobID, _ = config.String("query_update_queue_job_id", &emptystr)
	p.QueryUpdateJobData, _ = config.String("query_update_job_data", &emptystr)
	p.QueryInsertStatus, _ = config.String("query_insert_status", &emptystr)
	p.QuerySelectMDGIDSheetName, _ = config.String("query_select_mdgid_sheetname", &emptystr)
	p.QuerySelectJobID, _ = config.String("query_select_job_id", &emptystr)
	p.QuerySelectAllJobs, _ = config.String("query_select_all_jobs", &emptystr)

	// track the provider so we can clean it up later
	pLock.Lock()
	providers = append(providers, p)
	pLock.Unlock()
	return &p, nil
}

// ConfigTLS is used to configure TLS
// derived from github.com/jackc/pgx configTLS (https://github.com/jackc/pgx/blob/master/conn.go)
func ConfigTLS(sslMode string, sslKey string, sslCert string, sslRootCert string, cc *pgx.ConnConfig) error {

	switch sslMode {
	case "disable":
		cc.UseFallbackTLS = false
		cc.TLSConfig = nil
		cc.FallbackTLSConfig = nil
		return nil
	case "allow":
		cc.UseFallbackTLS = true
		cc.FallbackTLSConfig = &tls.Config{InsecureSkipVerify: true}
	case "prefer":
		cc.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		cc.UseFallbackTLS = true
		cc.FallbackTLSConfig = nil
	case "require":
		cc.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	case "verify-ca", "verify-full":
		cc.TLSConfig = &tls.Config{
			ServerName: cc.Host,
		}
	default:
		return ErrInvalidSSLMode(sslMode)
	}

	if sslRootCert != "" {
		caCertPool := x509.NewCertPool()

		caCert, err := ioutil.ReadFile(sslRootCert)
		if err != nil {
			return fmt.Errorf("unable to read CA file (%q): %v", sslRootCert, err)
		}

		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("unable to add CA to cert pool")
		}

		cc.TLSConfig.RootCAs = caCertPool
		cc.TLSConfig.ClientCAs = caCertPool
	}

	if (sslCert == "") != (sslKey == "") {
		return fmt.Errorf("both 'sslcert' and 'sslkey' are required")
	} else if sslCert != "" { // we must have both now
		cert, err := tls.LoadX509KeyPair(sslCert, sslKey)
		if err != nil {
			return fmt.Errorf("unable to read cert: %v", err)
		}

		cc.TLSConfig.Certificates = []tls.Certificate{cert}
	}

	return nil
}

// NewJob returns a new coordinator job based on the atlante job description
func (p *Provider) NewJob(job *atlante.Job) (jb *coordinator.Job, err error) {
	if job == nil {
		return nil, coordinator.ErrNilAtlanteJob
	}

	const insertQuery = `
INSERT INTO jobs(
	mdgid,
	sheet_number,
	sheet_name,
	bounds
)
VALUES($1,$2,$3,ST_GeometryFromText($4,$5))
RETURNING id;
`

	query := insertQuery
	if p.QueryNewJob != "" {
		query = p.QueryNewJob
	}
	var id int

	// TODO(gdey):Bug in wkt.Encode it should close the ring
	//	bounds, _ := wkt.Encode(job.Cell.Hull().AsPolygon())
	h := job.Cell.Hull().Vertices()
	h = append(h, h[0])
	bounds, _ := wkt.EncodeString(geom.Polygon{h})

	row := p.pool.QueryRow(
		query,
		job.Cell.Mdgid.Id,
		job.Cell.Mdgid.Part,
		job.SheetName,
		bounds,
		4326,
	)
	if err := row.Scan(&id); err != nil {
		return nil, err
	}
	return coordinator.NewJob(fmt.Sprintf("%v", id), job), nil
}

// UpdateField will update the job status for the given fields
func (p *Provider) UpdateField(job *coordinator.Job, fields ...field.Value) error {
	const updateQJobIDQuery = `
UPDATE jobs 
SET queue_id=$2
WHERE id=$1
	`
	const updateJobDataQuery = `
UPDATE jobs 
SET job_data=$2
WHERE id=$1
	`
	const insertStatusQuery = `
INSERT INTO job_statuses(
	job_id,
	status,
	description
)
VALUES($1,$2,$3);
	`

	var err error
	for _, f := range fields {
		switch fld := f.(type) {
		case field.QJobID:
			query := updateQJobIDQuery
			if p.QueryUpdateQueueJobID != "" {
				query = p.QueryUpdateQueueJobID
			}
			qjbid := string(fld)
			_, err = p.pool.Exec(query, job.JobID, qjbid)
		case field.JobData:
			query := updateJobDataQuery
			if p.QueryUpdateJobData != "" {
				query = p.QueryUpdateJobData
			}
			jbdata := string(fld)
			_, err = p.pool.Exec(query, job.JobID, jbdata)

		case field.Status:
			query := insertStatusQuery
			if p.QueryInsertStatus != "" {
				query = p.QueryInsertStatus
			}
			switch status := fld.Status.(type) {
			case field.Requested, field.Started, field.Completed:
				_, err = p.pool.Exec(
					query,
					job.JobID,
					fld.Status.String(),
					"",
				)
			case field.Processing:
				_, err = p.pool.Exec(
					query,
					job.JobID,
					"processing",
					status.Description,
				)
			case field.Failed:
				_, err = p.pool.Exec(
					query,
					job.JobID,
					"failed",
					status.Description,
				)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func logScanError(err error, query string) {
	switch e := err.(type) {
	case ErrNoQueueID:
		log.Infof("no queuid for job %v, skipping", int(e))
	case ErrInvalidStatus:
		log.Infof("invalid status entries for job %v, skipping", e.Job)
	default:
		if err != pgx.ErrNoRows {
			log.Infof("got error scanning job skipping: '%v'", err)
			log.Infof("sql: %v", query)
		}
	}
}
func scanRow(row rowScanner) (*coordinator.Job, error) {

	var (
		// an empty string we can take the pointer of
		emptyString string
		zeroTime    time.Time

		jobid       int
		mdgid       string
		jobdata     string
		sheetNumber int
		sheetName   string
		queueIDp    *string
		enqueued    time.Time
		status      *string
		desc        *string
		updated     *time.Time
		ajob        *atlante.Job
	)

	if err := row.Scan(
		&jobid,
		&mdgid,
		&sheetNumber,
		&sheetName,
		&queueIDp,
		&jobdata,
		&enqueued,
		&status,
		&desc,
		&updated,
	); err != nil {
		return nil, err
	}
	if queueIDp == nil || *queueIDp == "" {
		return nil, ErrNoQueueID(jobid)
	}

	if status == nil || *status == "" {
		return nil, ErrInvalidStatus{Job: jobid}
	}

	if desc == nil {
		desc = &emptyString
	}
	s, err := field.NewStatusFor(*status, *desc)
	if err != nil {
		return nil, ErrInvalidStatus{
			Job:    jobid,
			Status: *status,
			Desc:   *desc,
			Err:    err,
		}
	}
	if updated == nil {
		updated = &zeroTime
	}

	if jobdata != "" {
		ajob, _ = atlante.Base64UnmarshalJob(jobdata)
	}

	return &coordinator.Job{
		JobID:      fmt.Sprintf("%v", jobid),
		QJobID:     *queueIDp,
		MdgID:      mdgid,
		MdgIDPart:  uint32(sheetNumber),
		SheetName:  sheetName,
		Status:     field.Status{s},
		EnqueuedAt: enqueued,
		UpdatedAt:  *updated,
		AJob:       ajob,
	}, nil

}

// FindByJob will find the lastest 2 jobs as described by the atlante job descriptions
func (p *Provider) FindByJob(job *atlante.Job) (jobs []*coordinator.Job) {

	const selectQuery = `
SELECT 
	job.id,
	job.mdgid,
	job.sheet_number,
	job.sheet_name,
	job.queue_id,
	job.job_data,
    job.created as enqueued,
    jobstatus.status,
    jobstatus.description,
    jobstatus.created as updated
FROM jobs AS job
LEFT JOIN
( -- find the most recent job status if it exists
	SELECT DISTINCT ON (job_id)
	*
	FROM
	   job_statuses
	ORDER BY
	   job_id, id DESC
) AS jobstatus ON jobstatus.job_id = job.id
WHERE job.mdgid = $1 AND job.sheet_number = $2 AND job.sheet_name = $3
ORDER BY jobstatus.id desc 
LIMIT 2;
	`
	query := selectQuery
	if p.QuerySelectMDGIDSheetName != "" {
		query = p.QuerySelectMDGIDSheetName
	}
	if job == nil {
		return nil
	}
	mdgid := job.Cell.Mdgid.Id
	sheetNumber := job.Cell.Mdgid.Part
	sheetName := job.SheetName

	rows, err := p.pool.Query(query, mdgid, sheetNumber, sheetName)
	if err != nil {
		log.Errorf("postgresql: unable to preform query: %v", err)
		log.Errorf("postgresql: sql: %v", query)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		jb, err := scanRow(rows)
		if err != nil {
			logScanError(err, query)
			continue
		}
		jobs = append(jobs, jb)
	}
	return jobs
}

// FindByJobID will attempt to locate the job for the given job id
func (p *Provider) FindByJobID(jobid string) (jb *coordinator.Job, found bool) {

	const selectQuery = `
SELECT 
	job.id,
    job.mdgid,
    job.sheet_number,
    job.sheet_name,
	job.queue_id,
	job.job_data,
    job.created as enqueued,
    jobstatus.status,
    jobstatus.description,
    jobstatus.created as updated
FROM jobs AS job
JOIN job_statuses AS jobstatus ON job.id = jobstatus.job_id
WHERE job.id = $1
ORDER BY jobstatus.id desc limit 1;
	`
	query := selectQuery
	if p.QuerySelectJobID != "" {
		query = p.QuerySelectJobID
	}
	id, err := strconv.ParseInt(jobid, 10, 64)
	if err != nil {
		return nil, false
	}

	row := p.pool.QueryRow(query, id)
	jb, err = scanRow(row)
	if err != nil {
		logScanError(err, query)
		return nil, false
	}
	return jb, true

}

// genAllSQL retuns the all sql (primarySQL if it's not empty otherwise defaultSQL) that results from running the provided
// sql through a template processor
func genAllSQL(primarySQL, defaultSQL string, limit uint) (string, error) {
	if primarySQL == "" {
		primarySQL = defaultSQL
	}
	var lmt = func() string { return "" }
	if limit != 0 {
		lmtstr := fmt.Sprintf("limit %v", limit)
		lmt = func() string { return lmtstr }
	}
	tmpl, err := template.New("allsql").Funcs(
		map[string]interface{}{
			"limit": lmt,
		},
	).Parse(primarySQL)
	if err != nil {
		return "", err
	}
	var str strings.Builder

	err = tmpl.Execute(&str, struct{}{})
	if err != nil {
		return "", err
	}
	return str.String(), nil
}

// JobsByMDGID retuns the jobs that the provider knows about, that match the the given
// MDGID.

// Jobs returns the jobs that the provider knows about, if limit is not zero it will
// be used to limit the number of jobs returned to that limit.  Jobs will be returned
// from newest to oldest
func (p *Provider) Jobs(limit uint) (jobs []*coordinator.Job, err error) {
	const selectQuery = `
SELECT 
	job.id,
    job.mdgid,
    job.sheet_number,
    job.sheet_name,
    job.queue_id,
	job.job_data,
    job.created as enqueued,
    jobstatus.status,
    jobstatus.description,
    jobstatus.created as updated
FROM jobs AS job
LEFT JOIN
( -- find the most recent job status if it exists
	SELECT DISTINCT ON (job_id)
	*
	FROM 
	   job_statuses
	ORDER BY
	   job_id, id DESC
) AS jobstatus ON jobstatus.job_id = job.id
ORDER BY jobstatus.id desc
{{limit}}
;
`

	query, err := genAllSQL(p.QuerySelectAllJobs, selectQuery, limit)
	if err != nil {
		return nil, err
	}

	rows, err := p.pool.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		jb, err := scanRow(rows)
		if err != nil {
			logScanError(err, query)
			continue
		}
		jobs = append(jobs, jb)
	}
	return jobs, nil
}

// Close will close the provider's database connection
func (p *Provider) Close() { p.pool.Close() }

var pLock sync.RWMutex

// reference to all instantiated providers
var providers []Provider

// cleanup will close all database connections and destroy all prviously instantiated Provider instatnces
func cleanup() {
	if len(providers) == 0 {
		// Nothing to do
		return
	}
	pLock.Lock()
	for i := range providers {
		providers[i].Close()
	}
	providers = make([]Provider, 0)
	pLock.Unlock()
}
