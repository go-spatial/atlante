package grating

import (
	"fmt"
	"math"
	"strings"

	"github.com/go-spatial/geom"
)

const (
	MinRowCol = 2
	MaxRowCol = 40
)

// LabelLetters is the set of letter to use for labeling
var LabelLetters = [...]string{
	// Skip I and L (looks like 1 i l), O and Q (looks like 0, 6, o , q), S (looks like 5, s)
	// 0 1 2 3 4 5 6 7 8 9
	"A", "B", "C", "D", "E", "F", "G", "H", "J", "K",
	"M", "N", "P", "R", "T", "U", "V", "W", "X", "Y", "Z",
}

type Grating struct {
	// Rows the number of rows to draw, should only be between 0 and (MaxRowCol - MinRowCol)
	Rows uint
	// cols the number of cols to draw, should only be between 0 and (MaxRowCol - MinRowCol)
	Cols uint

	Extent geom.Extent

	// width and height are the amount of width and Height we
	// have to modify the value by to get the row or col line
	Width, Height float64

	FlipY      bool
	FlipYLabel bool
}

func Squarish(width, height float64, division uint) (widthDivision, heightDivision float64, rows, cols uint) {
	height, width = math.Abs(height), math.Abs(width)
	rows, cols = division, division
	widthDivision = width / float64(cols)
	heightDivision = height / float64(rows)

	if widthDivision >= heightDivision {
		rows = uint(height / widthDivision)
		heightDivision = height / float64(rows)
		return
	}
	cols = uint(width / heightDivision)
	widthDivision = width / float64(cols)
	return

}

func NewGrating(x, y, width, height float64, NumberOfRows, NumberOfCols uint, flipY bool) (*Grating, error) {

	if NumberOfRows < MinRowCol || NumberOfRows > MaxRowCol {
		return nil, fmt.Errorf("Invalid number of rows: %v; rows must be between %v and %v", NumberOfRows, MinRowCol, MaxRowCol)
	}
	if NumberOfCols < MinRowCol || NumberOfCols > MaxRowCol {
		return nil, fmt.Errorf("Invalid number of cols: %v; cols must be between %v and %v", NumberOfCols, MinRowCol, MaxRowCol)
	}

	colOffset := width / float64(NumberOfCols)
	rowOffset := height / float64(NumberOfRows)
	extent := geom.Extent{
		x, y, x + width, y + height,
	}

	if flipY {
		extent[1], extent[3] = extent[3], extent[1]
	}
	return &Grating{
		Rows: NumberOfRows,
		Cols: NumberOfCols,

		Extent: extent,

		FlipY:  flipY,
		Width:  colOffset,
		Height: rowOffset,
	}, nil

}

func (grate *Grating) labelForRow(row int) string {
	var (
		buff   strings.Builder
		lblLen = len(LabelLetters)
	)

	if row < lblLen {
		return LabelLetters[row]
	}
	prelbl := grate.labelForRow((row / lblLen) - 1)
	buff.WriteString(prelbl)
	buff.WriteString(LabelLetters[row%lblLen])
	return buff.String()
}
func (grate *Grating) LabelForRow(row int) string {
	if grate == nil || row < 0 || row >= int(grate.Rows) {
		return ""
	}
	if !grate.FlipYLabel {
		row = int(grate.Rows) - row - 1
	}
	return grate.labelForRow(row)
}

func (grate *Grating) LabelForCol(col int) string {
	if col < 0 || col >= int(grate.Cols) {
		return ""
	}
	return fmt.Sprintf("%d", col+1)
}

// LineForRow returns the row from 0 — Rows-1
func (grate *Grating) LineForRow(row uint) geom.Line {
	rowOffset := (float64(row) * grate.Height)
	if grate.FlipY {
		rowOffset *= -1
	}
	return geom.Line{
		[2]float64{
			grate.Extent[0],
			grate.Extent[1] + rowOffset,
		},
		[2]float64{
			grate.Extent[2],
			grate.Extent[1] + rowOffset,
		},
	}

}

// LineForCol returns the column from 0 — Cols-1
func (grate *Grating) LineForCol(col uint) geom.Line {
	colOffset := (float64(col) * grate.Width)
	return geom.Line{
		[2]float64{
			grate.Extent[0] + colOffset,
			grate.Extent[1],
		},
		[2]float64{
			grate.Extent[0] + colOffset,
			grate.Extent[3],
		},
	}
}

func (grate *Grating) YForRow(row int) float64 {
	if grate == nil {
		return 0.0
	}
	rowOffset := (float64(row) * grate.Height)
	if grate.FlipY {
		rowOffset *= -1
	}
	return grate.Extent[1] + rowOffset
}
func (grate *Grating) XForCol(col int) float64 {
	if grate == nil {
		return 0.0
	}
	colOffset := (float64(col) * grate.Width)
	return grate.Extent[0] + colOffset
}

// PositionFor row and col,
func (grate *Grating) PositionFor(row, col int) geom.Point {
	return geom.Point{
		grate.XForCol(col),
		grate.YForRow(row),
	}
}
