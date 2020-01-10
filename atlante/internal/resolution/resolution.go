package resolution

import (
	"math"

	"github.com/go-spatial/geom/planar/coord"
	"github.com/go-spatial/geom/planar/coord/utm"
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

// TODO(gdey) : get a proper projection system
var WGS84Ellipsoid = coord.Ellipsoid{
	Name:           "WSG_84",
	Radius:         MercatorEarthRadius,
	Eccentricity:   0.00669438,
	NATOCompatible: true,
}

func ZoomMapWidth(earthCircumference float64, scale uint, dpi uint, mapWidth float64) float64 {
	ground := float64(scale) * MeterPerInch / float64(dpi)
	log.Infof("Zoom: ground %v m/px,  mapWidth %v mapWidth %v px size %v", ground, mapWidth, mapWidth/ground, mapWidth/TileSize)
	return math.Log2((mapWidth / ground))
}

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

/*
from: https://sentinelhub-py.readthedocs.io/en/latest/_modules/sentinelhub/geo_utils.html#bbox_to_dimensions
  """ Calculates width and height in pixels for a given bbox of a given pixel resolution (in meters). The result is
    rounded to nearest integers

    :param bbox: bounding box
    :type bbox: geometry.BBox
    :param resolution: Resolution of desired image in meters. It can be a single number or a tuple of two numbers -
        resolution in horizontal and resolution in vertical direction.
    :type resolution: float or (float, float)
    :return: width and height in pixels for given bounding box and pixel resolution
    :rtype: int, int
    """
    utm_bbox = to_utm_bbox(bbox)
    east1, north1 = utm_bbox.lower_left
    east2, north2 = utm_bbox.upper_right

    resx, resy = resolution if isinstance(resolution, tuple) else (resolution, resolution)

	return round(abs(east2 - east1) / resx), round(abs(north2 - north1) / resy)
*/

func BoundsPixelWidthHeight(sw, ne coord.LngLat, gm float64) (float64, float64, error) {
	utmSw, err := utm.FromLngLat(sw, WGS84Ellipsoid)
	if err != nil {
		return 0, 0, err
	}
	utmNe, err := utm.FromLngLat(ne, WGS84Ellipsoid)
	if err != nil {
		return 0, 0, err
	}

	width := math.Abs(utmNe.Easting-utmSw.Easting) / gm
	height := math.Abs(utmNe.Northing-utmSw.Northing) / gm
	return width, height, nil

}

func GroundFromMapWidth(sw, ne coord.LngLat, imageWidth float64) (float64, error) {

	utmSw, err := utm.FromLngLat(sw, WGS84Ellipsoid)
	if err != nil {
		return 0, err
	}
	utmNe, err := utm.FromLngLat(ne, WGS84Ellipsoid)
	if err != nil {
		return 0, err
	}

	gm := math.Abs(utmNe.Easting-utmSw.Easting) / imageWidth
	log.Infof("easting1 %v easting2 %v  gm %v", utmNe.Easting, utmSw.Easting, gm)
	return gm, nil
}

func GroundFromMapHeight(sw, ne coord.LngLat, imageHeight float64) (float64, error) {

	utmSw, err := utm.FromLngLat(sw, WGS84Ellipsoid)
	if err != nil {
		return 0, err
	}
	utmNe, err := utm.FromLngLat(ne, WGS84Ellipsoid)
	if err != nil {
		return 0, err
	}

	gm := math.Abs(utmNe.Northing-utmSw.Northing) / imageHeight
	log.Infof("northing1 %v northing2 %v  gm %v", utmNe.Northing, utmSw.Northing, gm)
	return gm, nil
}

// Scale returns the map scale for the given ground resolution and dpi
// Formula from https://docs.microsoft.com/en-us/bingmaps/articles/bing-maps-tile-system
func Scale(dpi uint, ground float64) uint {
	return uint(ground * (float64(dpi) / MeterPerInch))
}

func LatInMeters(ec float64, lat float64) float64 {
	return math.Cos(lat*Rad) * ec
}

// ZoomForGround returns the zoom for the given ground resolution
func ZoomForGround(earthCircumfrence float64, ground float64, lat float64) float64 {
	width := math.Cos(lat * Rad)
	return math.Log2((width * earthCircumfrence) / (ground * TileSize))
}
