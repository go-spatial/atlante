package insetmap

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/winding"
)

var order = winding.Order{YPositiveDown: false}

const SVGMime = "image/svg+xml"

type SvgPath struct {
	fn    func(...float64) ([]float64, error)
	scale float64
	buff  int64
	total *geom.Extent
}

func NewSVGPath(e *geom.Extent, scale float64, buff int64) *SvgPath {
	// mid point of bounds.
	deltaX := e[0]
	deltaY := e[3] * -1
	fn := func(pts ...float64) ([]float64, error) {
		pts[0] -= deltaX
		pts[1] *= -1
		pts[1] -= deltaY
		pts[0] *= scale
		pts[1] *= scale
		return pts, nil
	}
	if debug {
		log.Printf("[DEBUG-svg] setting buff to: %v", buff)
	}
	return &SvgPath{
		fn:    fn,
		scale: scale,
		buff:  buff,
		total: e,
	}

}

func (svg *SvgPath) SetFn(fn func(pts ...float64) ([]float64, error)) {
	if debug {
		log.Println("[DEBUG] Using new adjust point.")
	}
	svg.fn = fn
}

func (svg *SvgPath) ViewBox() string {
	// mid point of bounds.
	top, _ := svg.fn(svg.total[0], svg.total[3])
	bot, _ := svg.fn(svg.total[2], svg.total[1])
	return fmt.Sprintf("%d %d %d %d",
		int64(top[0])-svg.buff, int64(top[1])-svg.buff,
		int64(bot[0])+svg.buff, int64(bot[1])+svg.buff,
	)
}

func (svg *SvgPath) Point(x, y float64) (float64, float64) {
	xy, _ := svg.fn(x, y)
	return xy[0], xy[1]
}

func (svg SvgPath) Path(g geom.Geometry) (string, error) {
	var path strings.Builder
	g, _ = geom.ApplyToPoints(g, svg.fn)
	switch geo := g.(type) {
	case geom.Polygon:
		//gpts := [][][2]float64(geo)
		gpts := order.RectifyPolygon([][][2]float64(geo))
		for _, l := range gpts {
			path.WriteString("M")
			var pts []string
			for _, pt := range l {
				pts = append(pts, fmt.Sprintf("%g %g", pt[0], pt[1]))
			}
			path.WriteString(strings.Join(pts, ","))
			path.WriteString("Z ")
		}
		return path.String(), nil
	case geom.MultiPolygon:
		for _, p := range geo {
			str, err := svg.Path(geom.Polygon(p))
			if err != nil {
				return "", err
			}
			path.WriteString(str)
		}
		return path.String(), nil
	case geom.LineString:
		path.WriteString("M")
		var pts []string
		for _, pt := range geo {
			pts = append(pts, fmt.Sprintf("%g %g", pt[0], pt[1]))
		}
		path.WriteString(strings.Join(pts, ","))
		return path.String(), nil
	case geom.MultiLineString:
		for _, l := range geo {
			str, err := svg.Path(geom.LineString(l))
			if err != nil {
				return "", err
			}
			path.WriteString(str)
		}
		return path.String(), nil

	default:
		if debug {
			log.Printf("[DEBUG] Got type: %T", g)
		}
	}
	return "", nil
}
