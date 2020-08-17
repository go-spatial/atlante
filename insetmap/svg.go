package insetmap

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/winding"
)

var order = winding.Order{YPositiveDown: false}

const SVGMime = "image/svg+xml"

func Attr(attrs map[string]string, extra string) string {
	// make sure attrs are always written in sorted order
	var pairs = make([]string, len(attrs))
	for k, v := range attrs {
		pairs = append(pairs, fmt.Sprintf(`%v="%v"`, strings.ToLower(k), v))
	}
	sort.Strings(pairs)
	extra = strings.TrimSpace(extra)
	if extra != "" {
		pairs = append(pairs, extra)
	}
	return strings.Join(pairs, " ")
}

func SVGTag(tag string, attrs string, body string) string {
	var svg strings.Builder
	svgTagBuilder(&svg, tag, attrs, body)
	return svg.String()
}
func svgTagBuilder(svg *strings.Builder, tag string, attrs string, body string) {
	svg.WriteRune('<')
	svg.WriteString(tag)
	svg.WriteRune(' ')
	svg.WriteString(attrs)
	svg.WriteString(">\n")

	svg.WriteString(body)

	svg.WriteString("\n</")
	svg.WriteString(tag)
	svg.WriteString(">\n")
}

func SVGTagFn(tag string, attrs string, body func() (string, error)) (string, error) {
	var svg strings.Builder
	bodyStr, err := body()
	if err != nil {
		return "", err
	}
	svgTagBuilder(&svg, tag, attrs, bodyStr)
	return svg.String(), nil
}

type SVGStringBuilder struct {
	strings.Builder
}

func (s *SVGStringBuilder) WriteTag(tag string, attr string, fn func(*SVGStringBuilder) error) error {

	s.WriteRune('<')
	s.WriteString(tag)

	s.WriteRune(' ')
	s.WriteString(attr)

	s.WriteString(">\n")

	err := fn(s)

	// We always close out the tag
	s.WriteString("\n</")
	s.WriteString(tag)
	s.WriteString(">\n")

	return err
}

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

func (svg SvgPath) encodePath(g geom.Geometry) (string, error) {
	var path strings.Builder
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
			str, err := svg.encodePath(geom.Polygon(p))
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
			str, err := svg.encodePath(geom.LineString(l))
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
func (svg SvgPath) Path(g geom.Geometry) (string, error) {
	g, _ = geom.ApplyToPoints(g, svg.fn)
	return svg.encodePath(g)
}
