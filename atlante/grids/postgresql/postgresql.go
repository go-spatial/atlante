package postgresql

import (
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/go-spatial/geom/encoding/wkb"

	"github.com/go-spatial/geom"
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
	queryBounds      string
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
	// ConfigKeyQueryBounds is the sql for getting grid values from a bounds value
	ConfigKeyQueryBounds = "query_bounds"

	// SQLGeometryField is the expected sql field name for the geometry
	SQLGeometryField = "geometry"
	// SQLMDGIDField is the expected sql field name for the mdgid
	SQLMDGIDField = "mdg_id"
	// SQLSheetField is the expected sql field name for the sheet
	SQLSheetField = "sheet"
	// SQLSeriesField is the expected sql field name for the series
	SQLSeriesField = "series"
	// SQLNRNField is the expected sql field name for the nrn
	SQLNRNField = "nrn"
	// SQLSWLATField is the expected sql field name for the southwest lat
	SQLSWLATField = "swlat"
	// SQLSWLNGField is the expected sql field name for the soutwest lng
	SQLSWLNGField = "swlng"
	// SQLNELATField is the expected sql field name for the northeast lat
	SQLNELATField = "nelat"
	// SQLNELNGField is the expected sql field name for the northeast lng
	SQLNELNGField = "nelng"
	// SQLSWLATDMSField is the expected sql field name for the soutwest lat in degree minute seconds
	SQLSWLATDMSField = "swlat_dms"
	// SQLSWLNGDMSField is the expected sql field name for the soutwest lng in degree minute seconds
	SQLSWLNGDMSField = "swlng_dms"
	// SQLNELATDMSField is the expected sql field name for the northeast lat in degree minute seconds
	SQLNELATDMSField = "nelat_dms"
	// SQLNELNGDMSField is the expected sql field name for the northeast lng in degree minute seconds
	SQLNELNGDMSField = "nelng_dms"
	// SQLCountryField is field for the country
	SQLCountryField = "country"
	// SQLCityField is field for the city
	SQLCityField = "city"
	// SQLEditedByField is field for the edited by
	SQLEditedByField = "edited_by"
	// SQLEditedAtField is field for the edited at
	SQLEditedAtField = "edited_at"
)

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
	var queryBounds string
	queryBounds, _ = config.String(ConfigKeyQueryBounds, &queryBounds)

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
		queryBounds:      queryBounds,
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

// CellForBounds get the cell for the given bounds
func (p *Provider) CellForBounds(bounds geom.Extent, srid uint) (*grids.Cell, error) {
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
		    wkb_geometry,
		    ST_Transform(
		      ST_SetSRID(
		        ST_MakeEnvelope($1,$2,$3,$4),
		        $5
		      ),
		      4326
		    )
		  )
		LIMIT 1;
		`
	query := selectQuery
	if p.queryBounds != "" {
		query = p.queryBounds
	}

	//log.Infof("Running SQL: %v\n%v", selectQuery, bounds)

	// bounds is sw,ne
	row := p.pool.QueryRow(query, bounds[0], bounds[1], bounds[2], bounds[3], srid)
	//row := p.pool.QueryRow(query, bounds[0], bounds[1], srid)

	cell, err := p.cellFromRow(row)
	if err != nil {
		return nil, err
	}

	cell.Sw = &grids.Cell_LatLng{
		Lng: float32(bounds[0]),
		Lat: float32(bounds[1]),
	}
	cell.Ne = &grids.Cell_LatLng{
		Lng: float32(bounds[2]),
		Lat: float32(bounds[3]),
	}
	latlen, lnglen := grids.CalculateSecLengths(float64(cell.Ne.Lat))
	cell.SwDms = nil
	cell.NeDms = nil
	/*
		swDMS := grids.ToDMS(float64(cell.Sw.Lat), float64(cell.Sw.Lng))
		neDMS := grids.ToDMS(float64(cell.Ne.Lat), float64(cell.Ne.Lng))
		cell.SwDms = &grids.Cell_LatLngDMS{Lat: swDMS[0].AsString(1), Lng: swDMS[1].AsString(1)}
		cell.NeDms = &grids.Cell_LatLngDMS{Lat: neDMS[0].AsString(1), Lng: neDMS[1].AsString(1)}
	*/
	cell.Len = &grids.Cell_LatLng{Lat: float32(latlen), Lng: float32(lnglen)}

	h := sha1.New()
	fmt.Fprintf(h, "%#V %v %v", bounds, srid, time.Now())
	cell.MetaData["filename"] = fmt.Sprintf("%x", h.Sum(nil))

	return cell, nil
}

// CellForLatLng returns a grid cell object that matches the closest grid cell.
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

	//log.Infof("lat/lng sql: %v",query)

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

type assignTo interface {
	AssignTo(interface{}) error
}

func float4val(val interface{}) (*float64, error) {
	var (
		vv  float64
		err error
	)
	switch v := val.(type) {
	case float32:
		vv = float64(v)
		return &vv, nil

	case float64:
		return &v, nil

	case string:
		vv, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		return &vv, nil

	case assignTo:
		if err := v.AssignTo(&vv); err != nil {
			return nil, err
		}
		return &vv, nil

	default:
		return nil, ErrInvalidType
	}
}

// cellFromRow parses grid attributes into a girds.Cell struct
func (p *Provider) cellFromRow(r *pgx.Row) (*grids.Cell, error) {
	rows := (*pgx.Rows)(r)
	defer rows.Close()

	return p.cellFromRows(rows)
}

// cellFromRows parse the grid attributes into a grids.Cell struct, only get's the first row.
func (p *Provider) cellFromRows(rows *pgx.Rows) (*grids.Cell, error) {
	fdescs := rows.FieldDescriptions()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if !rows.Next() {
		switch rows.Err() {
		case nil, pgx.ErrNoRows:
			return nil, grids.ErrNotFound
		default:
			return nil, rows.Err()
		}
	}
	vals, err := rows.Values()
	if err != nil {
		return nil, err
	}
	var (
		ok bool

		geomExtent *geom.Extent
		mdgid      string
		sheet      string
		series     string
		nrn        string
		swlat      *float64
		swlng      *float64
		nelat      *float64
		nelng      *float64
		swlatDMS   string
		swlngDMS   string
		nelatDMS   string
		nelngDMS   string
		country    string
		city       string
		editedBy   = p.editedBy
		editedAt   time.Time
	)

	for i := range vals {
		if vals[i] == nil {
			continue
		}
		switch fdescs[i].Name {

		case SQLGeometryField:
			geomBytes, ok := vals[i].([]byte)
			if !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into bytes to decode into geometry", fdescs[i].Name, i)
			}
			geometry, err := wkb.DecodeBytes(geomBytes)
			if err != nil {
				return nil, err
			}
			geomExtent, err = geom.NewExtentFromGeometry(geometry)
			if err != nil {
				return nil, err
			}

		case SQLMDGIDField:
			if mdgid, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for mdgid", fdescs[i].Name, i)
			}

		case SQLSheetField:
			if sheet, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for sheet", fdescs[i].Name, i)
			}

		case SQLSeriesField:
			if series, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for sheet", fdescs[i].Name, i)
			}

		case SQLNRNField:
			if nrn, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for nrn", fdescs[i].Name, i)
			}

		case SQLSWLATField:
			swlat, err = float4val(vals[i])
			if err != nil {
				if err == ErrInvalidType {
					return nil, fmt.Errorf("error unabled to convert field %v (%v) [%v] into float64 for swlat", fdescs[i].Name, i, vals[i])
				}
				return nil, fmt.Errorf("error for %v(%v) failed to parse %v as float64 for swlat: %v", fdescs[i].Name, i, vals[i], err)
			}

		case SQLSWLNGField:
			swlng, err = float4val(vals[i])
			if err != nil {
				if err == ErrInvalidType {
					return nil, fmt.Errorf("error unabled to convert field %v (%v) [%v] into float64 for swlng", fdescs[i].Name, i, vals[i])
				}
				return nil, fmt.Errorf("error for %v(%v) failed to parse %v as float64 for swlng: %v", fdescs[i].Name, i, vals[i], err)
			}

		case SQLNELATField:
			nelat, err = float4val(vals[i])
			if err != nil {
				if err == ErrInvalidType {
					return nil, fmt.Errorf("error unabled to convert field %v (%v) into float64 for nelat", fdescs[i].Name, i)
				}
				return nil, fmt.Errorf("error for %v(%v) failed to parse %v as float64 for nelat: %v", fdescs[i].Name, i, vals[i], err)
			}

		case SQLNELNGField:
			nelng, err = float4val(vals[i])
			if err != nil {
				if err == ErrInvalidType {
					return nil, fmt.Errorf("error unabled to convert field %v (%v) into float64 for nelng", fdescs[i].Name, i)
				}
				return nil, fmt.Errorf("error for %v(%v) failed to parse %v as float64 for nelng: %v", fdescs[i].Name, i, vals[i], err)
			}

		case SQLSWLATDMSField:
			if swlatDMS, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for swlatdms", fdescs[i].Name, i)
			}

		case SQLSWLNGDMSField:
			if swlngDMS, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for swlngdms", fdescs[i].Name, i)
			}

		case SQLNELATDMSField:
			if nelatDMS, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for nelatdms", fdescs[i].Name, i)
			}

		case SQLNELNGDMSField:
			if nelngDMS, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for nelngdms", fdescs[i].Name, i)
			}

		case SQLCountryField:
			if country, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for country", fdescs[i].Name, i)
			}

		case SQLCityField:
			if city, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for city", fdescs[i].Name, i)
			}

		case SQLEditedByField:
			if editedBy, ok = vals[i].(string); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for edited_by", fdescs[i].Name, i)
			}

		case SQLEditedAtField:
			if editedAt, ok = vals[i].(time.Time); !ok {
				return nil, fmt.Errorf("error unabled to convert field %v (%v) into string for edited_at", fdescs[i].Name, i)
			}
		}
	} // for range vals
	if mdgid == "" {
		return nil, fmt.Errorf("error required field mdgid not provided")
	}
	if sheet == "" {
		return nil, fmt.Errorf("error required field sheet not provided")

	}
	if series == "" {
		return nil, fmt.Errorf("error required field series not provided")
	}

	calculateCoords := (swlat == nil || nelat == nil || swlng == nil || nelng == nil)
	if calculateCoords {
		// if anyone is nil we will just calculate all of them
		if geomExtent == nil {
			return nil, fmt.Errorf("error required field geometry not provided")
		}
		swlngv, swlatv, nelngv, nelatv := geomExtent.MinX(), geomExtent.MinY(), geomExtent.MaxX(), geomExtent.MaxY()
		swlng, swlat, nelng, nelat = &swlngv, &swlatv, &nelngv, &nelatv
	}

	if calculateCoords || swlatDMS == "" || swlngDMS == "" || nelatDMS == "" || nelngDMS == "" {
		swdms := grids.ToDMS(*swlat, *swlng)
		swlatDMS, swlngDMS = swdms[0].String(), swdms[1].String()
		nedms := grids.ToDMS(*nelat, *nelng)
		nelatDMS, nelngDMS = nedms[0].String(), nedms[1].String()
	}

	metadata := make(map[string]string)

	return grids.NewCell(
		mdgid,                                 // mdgid
		[2]float64{*swlat, *swlng},            // sw
		[2]float64{*nelat, *nelng},            // ne
		country,                               // country
		city,                                  // city
		nil,                                   // utminfo
		grids.NewEditInfo(editedBy, editedAt), // edited info
		time.Now(),                            // publishedAt
		nrn,                                   // nrn
		sheet,                                 // sheet
		series,                                // series
		[2]string{swlatDMS, swlngDMS},         // sw dms
		[2]string{nelatDMS, nelngDMS},         // ne dms
		metadata,                              // metadata
	), nil

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
