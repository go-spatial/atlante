package atlante

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/go-spatial/atlante/svg2pdf"
)

var mediabox = regexp.MustCompile(`/MediaBox\s+\[(.+)\]`)

func getWidthHeightFromPDF(t *testing.T, pdfFilename string) (width, height float64, err error) {
	// Now we are going to hack around the pdf, and just look for the
	file, err := os.Open(pdfFilename)
	if err != nil {
		return 0, 0, fmt.Errorf("error opening pdf, expected nil got %v", err)
	}
	defer file.Close()
	freader := bufio.NewReader(file)
	idxs := mediabox.FindReaderSubmatchIndex(freader)
	if len(idxs) != 4 {
		t.Logf("got media box idxs: %v", idxs)
		return 0, 0, fmt.Errorf("get media box, expect 10 idxs, got %v.", len(idxs))
	}

	// Get enough space for any of the number
	byteBuff := make([]byte, idxs[3]-idxs[2])
	if _, err = file.Seek(int64(idxs[2]), 0); err != nil {
		return 0, 0, fmt.Errorf("error seeking to start , expected nil got %v", err)
	}
	if _, err = file.Read(byteBuff); err != nil {
		return 0, 0, fmt.Errorf("error reading needed bytes for entries")
	}
	vals := bytes.Split(byteBuff, []byte{' '})
	var fvals []float64
	for _, bstr := range vals {
		bstr = bytes.TrimSpace(bstr)
		if len(bstr) == 0 {
			continue
		}
		f, _ := strconv.ParseFloat(string(bstr), 64)
		fvals = append(fvals, f)
	}
	if len(fvals) != 4 {
		return 0, 0, fmt.Errorf("expected len to be 4 got: %v : %v", len(fvals), fvals)
	}
	width = fvals[2] - fvals[0]
	height = fvals[3] - fvals[1]

	return width, height, nil
}

func TestWidthHeightSVG2PDF(t *testing.T) {

	const (
		svgFilename = `testdata/testsvg.svg`
	)

	// got this from: https://unix.stackexchange.com/questions/39464/how-to-query-pdf-page-size-from-the-command-line

	dir, err := ioutil.TempDir("", "TestWidthHeightSVG2PDF")
	if err != nil {
		t.Skipf("Failed to create temp dir: %v", err)
		return
	}
	defer os.RemoveAll(dir)

	type tcase struct {
		Width  float64
		Height float64
	}

	fn := func(tc tcase) (string, func(*testing.T)) {
		tname := fmt.Sprintf("%v_%v", tc.Width, tc.Height)
		return tname, func(t *testing.T) {
			// let's first create a tmp pdf file name to use.
			pdfFilename := filepath.Join(dir, tname+".pdf")
			t.Logf("test pdf: %v", pdfFilename)
			err := svg2pdf.GeneratePDF(svgFilename, pdfFilename, tc.Width, tc.Height)
			if err != nil {
				t.Errorf("error generating pdf, expected nil got %v", err)
				return
			}
			width, height, err := getWidthHeightFromPDF(t, pdfFilename)
			if err != nil {
				t.Errorf("getting size error, expected nil, got %v", err)
				return
			}
			if tc.Width != width {
				t.Errorf("width, expected %v got %v", tc.Width, width)
			}
			if tc.Height != height {
				t.Errorf("height, expected %v got %v", tc.Height, height)
			}
		}
	}
	tests := []tcase{
		{
			Width:  720,
			Height: 720,
		},
		{
			Width:  800,
			Height: 600,
		},
		{
			Width:  1080,
			Height: 2048,
		},
		{
			Width:  720.5,
			Height: 830.23,
		},
		{
			Width:  2337.11,
			Height: 1765.45,
		},
	}
	for _, tc := range tests {
		t.Run(fn(tc))
	}
}
