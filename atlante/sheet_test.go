package atlante

import (
	"fmt"
	math "math"
	"testing"
)

// TOLERANCE is the epsilon value used in comparing floats.
const TOLERANCE = 0.000001

// Float64 compares two floats to see if they are within the given tolerance.
func Float64(f1, f2, tolerance float64) bool {
	if math.IsInf(f1, 1) {
		return math.IsInf(f2, 1)
	}
	if math.IsInf(f2, 1) {
		return math.IsInf(f1, 1)
	}
	if math.IsInf(f1, -1) {
		return math.IsInf(f2, -1)
	}
	if math.IsInf(f2, -1) {
		return math.IsInf(f1, -1)
	}
	return math.Abs(f1-f2) < tolerance
}

func Test_MMConversion(t *testing.T) {
	type tcase struct {
		lbl string
		// DPI to use
		dpi uint
		// mm for conversion
		mms []float64
		// expected point values
		pts []uint64
	}

	fn := func(dpi uint, mm float64, pt uint64) (string, func(*testing.T)) {
		lbl := fmt.Sprintf("mm:%.1f inch:%.1f", mm, mm*inchPerMM)
		return lbl, func(t *testing.T) {
			got := mmToPoint(mm, dpi)
			if got != pt {
				t.Errorf("value, expected %v got %v", pt, got)
			}
		}
	}
	tests := []tcase{
		{
			lbl: "original",
			dpi: 72,
			mms: []float64{
				715.518, 919.3277,
			},
			pts: []uint64{
				2028, 2606,
			},
		},
		{
			lbl: "A0",
			dpi: 72,
			mms: []float64{
				841, 1189,
			},
			pts: []uint64{
				2384, 3370,
			},
		},
		{
			lbl: "A1",
			dpi: 72,
			mms: []float64{
				594, 841,
			},
			pts: []uint64{
				1684, 2384,
			},
		},
		{
			lbl: "10inch",
			dpi: 72,
			mms: []float64{
				254, // 10 inches
			},
			pts: []uint64{
				720, // 10 inches
			},
		},
		{
			lbl: "100mm",
			dpi: 72,
			mms: []float64{
				100, // 10 inches
			},
			pts: []uint64{
				283, // 10 inches
			},
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s dpi %v", test.lbl, test.dpi), func(t *testing.T) {
			tl := len(test.mms)
			for i := 0; i < tl; i++ {
				t.Run(fn(test.dpi, test.mms[i], test.pts[i]))
			}
		})
	}
}
