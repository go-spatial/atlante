package config

import (
	"errors"
	"io"
	"net/url"

	"github.com/BurntSushi/toml"
	"github.com/go-spatial/maptoolkit/atlante/internal/env"
	"github.com/go-spatial/maptoolkit/atlante/internal/urlutil"
)

type Config struct {
	// FileLocation is the location that the config file was
	// read from. If this value is nil, then the Parse() function
	// was used directly
	FileLocation *url.URL   `toml:"-"`
	Providers    []env.Dict `toml:"providers"`
	Sheets       []Sheet    `toml:"sheets"`

	// metadata holds the metadata from parsing the toml
	// file
	metadata toml.MetaData `toml:"-"`
}

type Sheet struct {
	Name         env.String `toml:"name"`
	ProviderGrid env.String `toml:"provider_grid"`
	Scale        env.Int    `toml:"scale"`
	Template     env.String `toml:"template"`
	Style        env.String `toml:"style"`
	Notifier     env.String `toml:"notifier"`
}

func (c *Config) Validate() error {
	// TODO(gdey): Actually do the validation
	if c == nil {
		return errors.New("Config not initialized.")
	}
	return nil
}

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

func LoadAndValidate(location *url.URL) (cfg Config, err error) {
	cfg, err = Load(location)
	if err != nil {
		return cfg, err
	}
	return cfg, cfg.Validate()
}
