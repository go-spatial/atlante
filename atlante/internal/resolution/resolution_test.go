package resolution

import (
	"strconv"
	"testing"

	"github.com/go-spatial/geom/cmp"
)

func TestZoom(t *testing.T) {
	type tcase struct {
		dpi               uint
		lat               float64
		earthCircumfrence float64
		scale             uint
		zoom              float64
	}

	fn := func(tc tcase) func(*testing.T) {
		return func(t *testing.T) {
			zoom := Zoom(tc.earthCircumfrence, tc.scale, tc.dpi, tc.lat)
			if !cmp.Float(zoom, tc.zoom) {
				t.Errorf("zoom, expected %f, got %v", tc.zoom, zoom)
			}
		}
	}

	tests := map[uint]map[uint]map[string]tcase{
		Scale50k: {
			96: {
				"San Deigo, US": {
					earthCircumfrence: MercatorEarthCircumference,
					lat:               32.715736,
					zoom:              13.281348714459781,
				},
				"New York, US": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               40.785091,
					zoom:              13.129229242614507,
				},
				"Umeå, Sweden": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               63.825848,
					zoom:              12.34973052323617,
				},
				"Luleå, Norrbotten, Sweden": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               65.584816,
					zoom:              12.255970484637578,
				},
				"Port Stephens, Falkland Islands": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               -52.094273,
					zoom:              12.827715254340175,
				},
				"Nakuru, Kenya": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               -0.303099,
					zoom:              13.53052931740004,
				},
			},
		},
	}

	for scale, scaleValue := range tests {
		t.Run(strconv.FormatUint(uint64(scale), 10), func(t *testing.T) {
			for dpi, dpiValue := range scaleValue {
				t.Run(strconv.FormatUint(uint64(dpi), 10), func(t *testing.T) {
					for name, tc := range dpiValue {
						tc.dpi = dpi
						tc.scale = scale
						t.Run(name, fn(tc))
					}
				})
			}
		})
	}

}

func TestGround(t *testing.T) {

	type tcase struct {
		earthCircumfrence float64
		zoom              float64
		lat               float64
		ground            float64
	}

	fn := func(tc tcase) func(*testing.T) {
		return func(t *testing.T) {
			ground := Ground(tc.earthCircumfrence, tc.zoom, tc.lat)
			if !cmp.Float(ground, tc.ground) {
				t.Errorf("ground, expected %f, got %v", ground, tc.ground)
			}
		}
	}

	tests := map[string]tcase{}
	for name, tc := range tests {
		t.Run(name, fn(tc))
	}

}

func TestScale(t *testing.T) {

	type tcase struct {
		earthCircumfrence float64
		zoom              float64
		lat               float64
		dpi               uint
		scale             float64
	}
	fn := func(tc tcase) func(*testing.T) {
		return func(t *testing.T) {
			scale := Scale(tc.dpi, Ground(tc.earthCircumfrence, tc.zoom, tc.lat))
			if !cmp.Float(scale, tc.scale) {
				t.Errorf("scale, expected %f, got %v", tc.scale, scale)
			}
		}
	}
	tests := map[float64]map[uint]map[string]tcase{
		13.281348714459781: {
			96: {
				"San Deigo, US": {
					earthCircumfrence: MercatorEarthCircumference,
					lat:               32.715736,
					scale:             50000,
				},
			},
		},
		13.0: {
			96: {
				"San Deigo, US": {
					earthCircumfrence: MercatorEarthCircumference,
					lat:               32.715736,
				},
				"New York, US": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               40.785091,
				},
				"Umeå, Sweden": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               63.825848,
				},
				"Luleå, Norrbotten, Sweden": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               65.584816,
				},
				"Port Stephens, Falkland Islands": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               -52.094273,
				},
				"Nakuru, Kenya": tcase{
					earthCircumfrence: MercatorEarthCircumference,
					lat:               -0.303099,
				},
			},
		},
	}
	for zoom, zoomValue := range tests {
		t.Run(strconv.FormatFloat(zoom, 'E', -1, 64), func(t *testing.T) {
			for dpi, dpiValue := range zoomValue {
				t.Run(strconv.FormatUint(uint64(dpi), 10), func(t *testing.T) {
					for name, tc := range dpiValue {
						tc.dpi = dpi
						tc.zoom = zoom
						t.Run(name, fn(tc))
					}
				})
			}
		})
	}

}
