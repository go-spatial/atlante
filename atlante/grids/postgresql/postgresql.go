package postgresql

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/tegola"
	"github.com/jackc/pgx"
)

// Name is the name of the provider type
const Name = "postgresql"

// AppName is shown by the pqclient
var AppName = "atlante"

// Provider implements the Grid.Provider interface
type Provider struct {
	config           pgx.ConnPoolConfig
	pool             *pgx.ConnPool
	srid             uint
	editedBy         string
	editedDateFormat string
	cellSize         grids.CellSize
	queryLngLat      string
	queryMDGID       string
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
	// ConfigKeyEditDateFormat is the format to use for dates
	ConfigKeyEditDateFormat = "edit_date_format"
	// ConfigKeyEditBy who the default user for edit_by should be
	ConfigKeyEditBy = "edit_by"

	// ConfigKeyScale is the scale of this provider
	ConfigKeyScale = "scale"

	// ConfigKeyQueryMDGID is the sql for getting grid values from a MDGID.
	ConfigKeyQueryMDGID = "query_mdgid"
	// ConfigKeyQueryLngLat is the sql for getting grid values from a lng lat value.
	ConfigKeyQueryLngLat = "query_lnglat"
)

// ErrInvalidSSLMode is returned when something is wrong with SSL configuration
type ErrInvalidSSLMode string

func (e ErrInvalidSSLMode) Error() string {
	return fmt.Sprintf("postgis: invalid ssl mode (%v)", string(e))
}

func init() {
	grids.Register(Name, NewGridProvider, Cleanup)
}

// NewGridProvider returns a grid provider based on the postgis database
func NewGridProvider(config grids.ProviderConfig) (grids.Provider, error) {
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

	var scale uint
	if scale, err = config.Uint(ConfigKeyScale, nil); err != nil {
		return nil, err
	}

	editedBy := DefaultEditBy
	if editedBy, err = config.String(ConfigKeyEditBy, &editedBy); err != nil {
		return nil, err
	}

	editedDateFormat := DefaultEditDateFormat
	if editedDateFormat, err = config.String(ConfigKeyEditBy, &editedDateFormat); err != nil {
		return nil, err
	}

	var queryLngLat string
	queryLngLat, _ = config.String(ConfigKeyQueryLngLat, &queryLngLat)
	var queryMDGID string
	queryMDGID, _ = config.String(ConfigKeyQueryMDGID, &queryMDGID)
	if (queryLngLat != "" || queryMDGID != "") && (queryLngLat == "" || queryMDGID == "") {
		if queryLngLat == "" {
			return nil, errors.String("error " + ConfigKeyQueryLngLat + " not set, when " + ConfigKeyQueryMDGID + " is set")
		}
		return nil, errors.String("error " + ConfigKeyQueryMDGID + " not set, when " + ConfigKeyQueryLngLat + " is set")
	}

	connConfig := pgx.ConnConfig{
		Host:     host,
		Port:     uint16(port),
		Database: db,
		User:     user,
		Password: password,
		LogLevel: pgx.LogLevelWarn,
		RuntimeParams: map[string]string{
			"default_transaction_read_only": "TRUE",
			"application_name":              AppName,
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
		srid:             uint(srid),
		editedBy:         editedBy,
		editedDateFormat: editedDateFormat,
		cellSize:         grids.CellSize(scale),
		queryLngLat:      queryLngLat,
		queryMDGID:       queryMDGID,
	}
	if p.pool, err = pgx.NewConnPool(p.config); err != nil {
		return nil, fmt.Errorf("failed while creating connection pool: %v", err)
	}

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

// CellSize returns the grid cell size
func (p *Provider) CellSize() grids.CellSize {
	if p == nil {
		return grids.CellSize50K
	}
	return p.cellSize
}

// CellForLatLng returns a grid cell object that matches the cloest grid cell.
func (p *Provider) CellForLatLng(lat, lng float64, srid uint) (*grids.Cell, error) {
	const selectQuery = `
SELECT
  mdg_id,
  sheet,
  series,
  nrn,

  swlat_dms,
  swlon_dms AS swlng_dms,
  nelat_dms,
  nelon_dms AS nelng_dms,

  swlat,
  swlon AS swlng,
  nelat,
  nelon AS nelng,

  country,
  last_edite AS edited_by,
  last_edi_1 AS edited_at
FROM
  grids.grid50K
WHERE
  ST_Intersects(
    wkb_geometry,
    ST_Transform(
      ST_SetSRID(
        ST_MakePoint($1,$2),
        $3
      ),
      4326
    )
  )
LIMIT 1;
`
	query := selectQuery
	if p.queryLngLat != "" {
		query = p.queryLngLat
	}

	row := p.pool.QueryRow(query, lng, lat, srid)

	return p.cellFromRow(row)
}

// CellForMDGID returns an grid cell object for the given mdgid
func (p *Provider) CellForMDGID(mdgid *grids.MDGID) (*grids.Cell, error) {
	const selectQuery = `
SELECT
  mdg_id,
  sheet,
  series,
  nrn,

  swlat_dms,
  swlon_dms AS swlng_dms,
  nelat_dms,
  nelon_dms AS nelng_dms,

  swlat,
  swlon AS swlng,
  nelat,
  nelon AS nelng,

  country,
  last_edite AS edited_by,
  last_edi_1 AS edited_at
FROM
  grids.grid50K
WHERE
	mdg_id = $1
LIMIT 1;
`

	query := selectQuery
	if p.queryMDGID != "" {
		query = p.queryMDGID
	}

	row := p.pool.QueryRow(query, mdgid.Id)

	return p.cellFromRow(row)
}

// cellFromRow parses grid attributes into a girds.Cell struct
func (p *Provider) cellFromRow(row *pgx.Row) (*grids.Cell, error) {
	var (
		mdgid  sql.NullString
		sheet  sql.NullString
		series sql.NullString
		nrn    sql.NullString

		swlatdms sql.NullString
		swlngdms sql.NullString
		nelatdms sql.NullString
		nelngdms sql.NullString

		swlat sql.NullFloat64
		swlng sql.NullFloat64
		nelat sql.NullFloat64
		nelng sql.NullFloat64

		country  sql.NullString
		editedBy sql.NullString
		editedAt sql.NullString
	)

	err := row.Scan(
		&mdgid,
		&sheet,
		&series,
		&nrn,
		&swlatdms,
		&swlngdms,
		&nelatdms,
		&nelngdms,
		&swlat,
		&swlng,
		&nelat,
		&nelng,
		&country,
		&editedBy,
		&editedAt,
	)
	if err != nil {
		return nil, err
	}

	byStr, edAt, err := p.newEditInfo(editedBy, editedAt)
	if err != nil {
		return nil, err
	}

	city := ""

	return grids.NewCell(
		mdgid.String,                             // mdgid
		[2]float64{swlat.Float64, swlng.Float64}, // sw
		[2]float64{nelat.Float64, nelng.Float64}, // ne
		country.String,                           // country
		city,                                     // city
		nil,                                      // utminfo
		grids.NewEditInfo(byStr, edAt),           // edited info
		time.Now(),                               // publishedAt
		nrn.String,                               // nrn
		sheet.String,                             // sheet
		series.String,                            // series
		[2]string{swlatdms.String, swlngdms.String}, // sw dms
		[2]string{nelatdms.String, nelngdms.String}, // ne dms
		nil, // metadata
	), nil
}

func (p *Provider) newEditInfo(by, date sql.NullString) (strBy string, edtAt time.Time, err error) {

	strBy = p.editedBy
	if by.Valid {
		strBy = by.String
	}

	// Try and parse the date
	if date.Valid {
		edtAt, err = time.Parse(p.editedDateFormat, date.String)
		if err != nil {
			return strBy, edtAt, err
		}
	}
	return strBy, edtAt, nil
}

// Close will close the provider's database connection
func (p *Provider) Close() { p.pool.Close() }

var pLock sync.RWMutex

// reference to all instantiated providers
var providers []Provider

// Cleanup will close all database connections and destroy all prviously instantiated Provider instatnces
func Cleanup() {
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
