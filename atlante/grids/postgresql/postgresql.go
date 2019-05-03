package postgresql

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/tegola"
	"github.com/jackc/pgx"
)

const Name = "postgresql"

var AppName = "atlante"

type Provider struct {
	config           pgx.ConnPoolConfig
	pool             *pgx.ConnPool
	srid             uint
	editedBy         string
	editedDateFormat string
}

const (
	DefaultSRID           = tegola.WebMercator
	DefaultPort           = 5432
	DefaultMaxConn        = 100
	DefaultSSLMode        = "disable"
	DefaultSSLKey         = ""
	DefaultSSLCert        = ""
	DefaultEditDateFromat = time.RFC3339
	DefaultEditBy         = ""
)

const (
	ConfigKeyHost           = "host"
	ConfigKeyPort           = "port"
	ConfigKeyDB             = "database"
	ConfigKeyUser           = "user"
	ConfigKeyPassword       = "password"
	ConfigKeySSLMode        = "ssl_mode"
	ConfigKeySSLKey         = "ssl_key"
	ConfigKeySSLCert        = "ssl_cert"
	ConfigKeySSLRootCert    = "ssl_root_cert"
	ConfigKeyMaxConn        = "max_connections"
	ConfigKeySRID           = "srid"
	ConfigKeyEditDateFormat = "edit_date_format"
	ConfigKeyEditBy         = "edit_by"
)

type ErrInvalidSSLMode string

func (e ErrInvalidSSLMode) Error() string {
	return fmt.Sprintf("postgis: invalid ssl mode (%v)", string(e))
}

func init() {
	grids.Register(Name, NewGridProvider, Cleanup)
}

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

	editedBy := DefaultEditBy
	if editedBy, err = config.String(ConfigKeyEditBy, &editedBy); err != nil {
		return nil, err
	}

	editedDateFormat := DefaultEditDateFromat
	if editedDateFormat, err = config.String(ConfigKeyEditBy, &editedDateFormat); err != nil {
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
	}
	if p.pool, err = pgx.NewConnPool(p.config); err != nil {
		return nil, fmt.Errorf("Failed while creating connection pool: %v", err)
	}

	// track the provider so we can clean it up later
	pLock.Lock()
	providers = append(providers, p)
	pLock.Unlock()
	return &p, nil
}

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

func (p *Provider) newEditInfo(by, date sql.NullString) (*grids.EditInfo, error) {
	var err error
	strBy := p.editedBy
	if by.Valid {
		strBy = by.String
	}
	ei := grids.EditInfo{
		By: strBy,
	}
	// Try and parse the date
	if date.Valid {
		ei.Date, err = time.Parse(p.editedDateFormat, date.String)
		if err != nil {
			log.Printf("Got an error trying to parse %v -- %v", p.editedDateFormat, date)
			return nil, err
		}
	}
	return &ei, nil
}

func (p *Provider) GridForLatLng(lat, lng float64, srid uint) (*grids.Grid, error) {
	const selectQuery = `
SELECT
  mdg_id,
  sheet,
  series,
  nrn,
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
    geom,
    ST_Transform(
      ST_SetSRID(
        ST_MakePoint($1,$2),
        $3
      ),
      3857
    )
  )
LIMIT 1;
`

	var (
		mdgID  sql.NullString
		sheet  sql.NullString
		series sql.NullString
		nrn    sql.NullString

		swlat     sql.NullFloat64
		swlng     sql.NullFloat64
		nelat     sql.NullFloat64
		nelng     sql.NullFloat64
		country   sql.NullString
		edited_by sql.NullString
		edited_at sql.NullString
	)

	err := p.pool.QueryRow(selectQuery, lat, lng, srid).Scan(
		&mdgID,
		&sheet,
		&series,
		&nrn,
		&swlat,
		&swlng,
		&nelat,
		&nelng,
		&country,
		&edited_by,
		&edited_at,
	)
	if err != nil {
		return nil, err
	}
	latlen, lnglen := grids.CalculateSecLengths(nelng.Float64)
	mdgid := grids.NewMDGID(mdgID.String)
	ei, err := p.newEditInfo(edited_by, edited_at)
	if err != nil {
		return nil, err
	}
	return &grids.Grid{
		MdgID:  mdgid,
		SRID:   3875,
		Sheet:  sheet.String,
		Series: series.String,
		NRN:    nrn.String,
		SWLat:  swlat.Float64,
		SWLng:  swlng.Float64,
		NELat:  nelat.Float64,
		NELng:  nelng.Float64,

		LatLen: latlen,
		LngLen: lnglen,

		PublicationDate: time.Now(),
		Country:         country.String,
		Edited:          ei,
	}, nil
}
func (p *Provider) GridForMDGID(mdgid grids.MDGID) (*grids.Grid, error) {
	const selectQuery = `
SELECT
  sheet,
  series,
  nrn,
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

	var (
		sheet  sql.NullString
		series sql.NullString
		nrn    sql.NullString

		swlat     sql.NullFloat64
		swlng     sql.NullFloat64
		nelat     sql.NullFloat64
		nelng     sql.NullFloat64
		country   sql.NullString
		edited_by sql.NullString
		edited_at sql.NullString
	)
	err := p.pool.QueryRow(selectQuery, mdgid.ID).Scan(
		&sheet,
		&series,
		&nrn,
		&swlat,
		&swlng,
		&nelat,
		&nelng,
		&country,
		&edited_by,
		&edited_at,
	)
	if err != nil {
		return nil, err
	}
	latlen, lnglen := grids.CalculateSecLengths(nelng.Float64)
	ei, err := p.newEditInfo(edited_by, edited_at)
	if err != nil {
		return nil, err
	}
	return &grids.Grid{
		MdgID:  mdgid,
		SRID:   3875,
		Sheet:  sheet.String,
		Series: series.String,
		NRN:    nrn.String,
		SWLat:  swlat.Float64,
		SWLng:  swlng.Float64,
		NELat:  nelat.Float64,
		NELng:  nelng.Float64,

		LatLen: latlen,
		LngLen: lnglen,

		PublicationDate: time.Now(),
		Country:         country.String,
		Edited:          ei,
	}, nil
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
