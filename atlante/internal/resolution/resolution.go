package resolution

import (
	"math"
)

const (
	MercatorEarthRadius        = 6378137
	MercatorEarthCircumference = 2 * math.Pi * MercatorEarthRadius
	Rad                        = math.Pi / 180
	TileSize                   = 256
	MeterPerInch               = 0.0254
	Scale50k                   = 50000
)

// Zoom returns the zoom value for the given scale, dpi, and latitude.
func Zoom(earthCircumference float64, scale uint, dpi uint, lat float64) float64 {
	//
	width := math.Cos(lat * Rad)
	ground := float64(scale) * MeterPerInch / float64(dpi)
	mapWidth := (width * earthCircumference) / ground
	return math.Log2(mapWidth / TileSize)

}

// Ground returns the ground resolution (meter/pixel)
// Formula from https://docs.microsoft.com/en-us/bingmaps/articles/bing-maps-tile-system
func Ground(earthCircumfrence float64, zoom float64, lat float64) float64 {
	mapWidth := TileSize * math.Pow(2, zoom)
	width := math.Cos(lat * Rad)
	return width * earthCircumfrence / mapWidth
}

// Scale returns the map scale for the given ground resolution and dpi
// Formula from https://docs.microsoft.com/en-us/bingmaps/articles/bing-maps-tile-system
func Scale(dpi uint, ground float64) float64 {
	return ground * (float64(dpi) / MeterPerInch)
}

// ZoomForGround returns the zoom for the given ground resolution
func ZoomForGround(earthCircumfrence float64, ground float64, lat float64) float64 {
	width := math.Cos(lat * Rad)
	return math.Log2((width * earthCircumfrence) / (ground * TileSize))
}
