package postgresql

import (
	"fmt"

	"github.com/gdey/errors"
)

const (
	// ErrMissingQueryMDGID is returned when the QueryLngLat is specified but not QueryMDGID
	ErrMissingQueryMDGID = errors.String("error " + ConfigKeyQueryLngLat + " not set, when " + ConfigKeyQueryMDGID + " is set")

	// ErrMissingQueryLngLat is returned when the QueryMDGID is specified but not QueryLngLat
	ErrMissingQueryLngLat = errors.String("error " + ConfigKeyQueryMDGID + " not set, when " + ConfigKeyQueryLngLat + " is set")
)

// ErrInvalidSSLMode is returned when something is wrong with SSL configuration
type ErrInvalidSSLMode string

func (e ErrInvalidSSLMode) Error() string {
	return fmt.Sprintf("postgis: invalid ssl mode (%v)", string(e))
}
