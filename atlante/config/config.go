package config

import (
	"errors"
	"io"
	"net/url"

	"github.com/BurntSushi/toml"
	"github.com/go-spatial/maptoolkit/atlante/internal/env"
	"github.com/go-spatial/maptoolkit/atlante/internal/urlutil"
)

// Config models the config file that can be passed into the application
type Config struct {
	// FileLocation is the location that the config file was
	// read from. If this value is nil, then the Parse() function
	// was used directly
	FileLocation *url.URL `toml:"-"`

	// Webserver is the configuration for the webserver
	Webserver Webserver `toml:"webserver"`

	Notifier env.Dict `toml:"notifier"`

	Providers []env.Dict `toml:"providers"`
	Sheets    []Sheet    `toml:"sheets"`

	// Workdirectory is the directory where the system should do it's work.
	Workdirectory string `toml:"work_directory"`

	// FileStores are used to move the generated files to locations
	// that the user wants
	FileStores []env.Dict `toml:"file_stores"`

	// metadata holds the metadata from parsing the toml
	// file
	metadata toml.MetaData `toml:"-"`
}

// Webserver represents the config values for the webserver potion
// of the application.
type Webserver struct {
	HostName              env.String        `toml:"hostname"`
	Port                  env.String        `toml:"port"`
	Scheme                env.String        `toml:"scheme"`
	Headers               map[string]string `toml:"headers"`
	Queue                 env.Dict          `toml:"queue"`
	DisableNotificationEP bool              `toml:"disable_notification_endpoint"`
	Coordinator           env.Dict          `toml:"coordinator"`
}

// Sheet models a sheet in the config file
type Sheet struct {
	Name         env.String   `toml:"name"`
	ProviderGrid env.String   `toml:"provider_grid"`
	Filestores   []env.String `toml:"file_stores"`
	DPI          env.Int      `toml:"dpi"`
	Template     env.String   `toml:"template"`
	Style        env.String   `toml:"style"`
	Description  env.String   `toml:"description"`
}

// Validate will validate the config and make sure the is valid
func (c *Config) Validate() error {
	// TODO(gdey): Actually do the validation
	if c == nil {
		return errors.New("error config not initialized")
	}
	return nil
}

// Parse will parse a config file in the io.Reader
func Parse(reader io.Reader, fileLocation *url.URL) (conf Config, err error) {
	// decode conf file, don't care about the meta data.
	_, err = toml.DecodeReader(reader, &conf)
	conf.FileLocation = fileLocation

	return conf, err
}

// Load will load and parse the config file from the given location.
func Load(location *url.URL) (conf Config, err error) {
	err = urlutil.VisitReader(location, func(r io.Reader) error {
		var e error
		conf, e = Parse(r, location)
		return e
	})
	return conf, err
}

// LoadAndValidate is helper function that just calls load and then validate
func LoadAndValidate(location *url.URL) (cfg Config, err error) {
	cfg, err = Load(location)
	if err != nil {
		return cfg, err
	}
	return cfg, cfg.Validate()
}
