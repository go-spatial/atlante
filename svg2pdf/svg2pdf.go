package svg2pdf

// #cgo pkg-config: librsvg-2.0 cairo-pdf cairo-ft libxml-2.0
// #cgo pkg-config: gio-2.0
// #cgo pkg-config: libcroco-0.6 libpcre libpng
// #cgo pkg-config: pango pangocairo pangoft2 fontconfig freetype2
// #include "svg2pdf.h"
import "C"
import (
	"fmt"
)

func Svg2pdf(fileIn, fileOut string, height, width float64) error {
	e := C.svg2pdf_file(C.CString(fileIn), C.CString(fileOut),
		C.double(height), C.double(width))

	if e != 0 {
		return fmt.Errorf("error %d", e)
	} else {
		return nil
	}
}
