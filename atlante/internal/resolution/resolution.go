package resolution

import (
	"math"

	"github.com/prometheus/common/log"
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
	log.Infof("Zoom: ground %v m/px, width %v m, mapWidth %v px size %v", ground, width, mapWidth, mapWidth/TileSize)
	return math.Log2(mapWidth / TileSize)

}

// Ground returns the ground resolution (meter/pixel)
// Formula from https://docs.microsoft.com/en-us/bingmaps/articles/bing-maps-tile-system
func Ground(earthCircumfrence float64, zoom float64, lat float64) float64 {
	mapWidth := TileSize * math.Pow(2, zoom)
	width := math.Cos(lat * Rad)
	return width * earthCircumfrence / mapWidth
}

func GroundFromMapWidth(earthCircumfrence float64, mapWidth float64, lat float64) float64 {
	width := math.Cos(lat * Rad)
	log.Infof("bounds width: %v", width)
	return width * earthCircumfrence / mapWidth
}

// Scale returns the map scale for the given ground resolution and dpi
// Formula from https://docs.microsoft.com/en-us/bingmaps/articles/bing-maps-tile-system
func Scale(dpi uint, ground float64) uint {
	return uint(ground * (float64(dpi) / MeterPerInch))
}

// ZoomForGround returns the zoom for the given ground resolution
func ZoomForGround(earthCircumfrence float64, ground float64, lat float64) float64 {
	width := math.Cos(lat * Rad)
	return math.Log2((width * earthCircumfrence) / (ground * TileSize))
}
