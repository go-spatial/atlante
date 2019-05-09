package atlante

import (
	"fmt"

	"github.com/gdey/errors"
)

const (
	// ErrNilGrid is returned when a nil grid is provided
	ErrNilGrid = errors.String("grid is nil")
	// ErrNilSheet is returned when a nil sheet is provided
	ErrNilSheet = errors.String("sheet is nil")
	// ErrNilAtlanteObject is returned when a nil atlante object is provided
	ErrNilAtlanteObject = errors.String("atlante object is nil")
	// ErrBlankSheetName is returned for a blank sheet name
	ErrBlankSheetName = errors.String("blank sheet name")
	// ErrDuplicateSheetName is returned for a duplicate sheet name
	ErrDuplicateSheetName = errors.String("duplicate sheet name")
	// ErrNoSheets is returned when no sheets were configured into the system
	ErrNoSheets = errors.String("no sheets configured")
)

// ErrUnknownSheetName is returned when the sheet requested is not found or known.
type ErrUnknownSheetName string

func (eusn ErrUnknownSheetName) Error() string {
	return fmt.Sprintf("unknown sheet named %v", string(eusn))
}
