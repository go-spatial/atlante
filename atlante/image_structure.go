package atlante

import (
	"context"
	"image/png"
	"sync"

	"github.com/go-spatial/maptoolkit/atlante/filestore"
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/go-spatial/maptoolkit/atlante/internal/resolution"
	"github.com/go-spatial/maptoolkit/mbgl/bounds"
	"github.com/go-spatial/maptoolkit/mbgl/image"
	mbgl "github.com/go-spatial/maptoolkit/mbgl/image"
	"github.com/prometheus/common/log"
)

// Img is a wrapper around an mbgl image that make the image available to the
// template, and allows for the image to be encode only if it's requested
// It also will allow the Desired With and Height of the Image to be set.
// If this is set and a bounds is provided for the map region, then the
// scale will be recalculated
type Img struct {
	File *filestore.File

	DPI        uint
	Grid       *grids.Cell
	Projection bounds.AProjection
	Scale      uint
	Style      string

	StartGenerationCallback func()
	EndGenerationCallback   func()
	FailGenerationCallback  func(error)

	// Did we already generate the base image
	generated           bool
	lck                 sync.Mutex
	image               *mbgl.Image
	width, height       float64
	imgWidth, imgHeight float64
	groundMeasure       float64
	zoom                float64

	// staticWidthHeight determines if the Width and Height for this was statically defined
	// if so, then we need to dynamically figure out the scale from the bounds, if bounds is
	// not provided we may still dynamically determine a width and height
	staticWidthHeight bool
}

func (img *Img) initDynamicWidthHeight() {
	log.Infof("Using dynamic width and height")
	grid := img.Grid

	zoom := grid.ZoomForScaleDPI(img.Scale, img.DPI)
	img.groundMeasure = resolution.Ground(
		resolution.MercatorEarthCircumference,
		zoom,
		float64(grid.GetSw().GetLat()),
	)
	img.width, img.height = grid.WidthHeightForZoom(zoom)
	img.imgWidth, img.imgHeight = img.width, img.height
	img.zoom = zoom

}

func (img *Img) initStaticWidthHeight(tilesize float64) {
	var err error
	log.Infof("Using static width and height: tilesize: %v", tilesize)
	log.Infof("img.Grid %v", img.Grid)
	log.Infof("img.Grid Ne %v", img.Grid.GetNe())
	if img.width <= img.height {
		img.groundMeasure, err = resolution.GroundFromMapWidth(
			img.Grid.Sw.CoordLngLat(),
			img.Grid.Ne.CoordLngLat(),
			img.width,
		)
	} else {
		img.groundMeasure, err = resolution.GroundFromMapHeight(
			img.Grid.Sw.CoordLngLat(),
			img.Grid.Ne.CoordLngLat(),
			img.height,
		)

	}
	if err != nil {
		panic(err)
	}
	img.imgWidth, img.imgHeight, err = resolution.BoundsPixelWidthHeight(
		img.Grid.Sw.CoordLngLat(),
		img.Grid.Ne.CoordLngLat(),
		img.groundMeasure,
	)
	if err != nil {
		panic(err)
	}
	img.Scale = resolution.Scale(img.DPI, img.groundMeasure)
	// adjust the zoom as we are 256 tiles based
	img.zoom = img.Grid.ZoomForScaleDPI(img.Scale, img.DPI) - 1
}

func (img *Img) initImage(ctx context.Context) (*mbgl.Image, error) {

	if img == nil {
		return nil, nil
	}
	if img.image != nil {
		return img.image, nil
	}

	const tilesize = 4096 / 2
	var (
		err  error
		grid = img.Grid
	)

	if img.staticWidthHeight {
		img.initStaticWidthHeight(4096 / 4)
	} else {
		img.initDynamicWidthHeight()
	}

	latLngCenterPt := grid.CenterPtForZoom(img.zoom)
	log.Infoln("width", img.width, "height", img.height)
	log.Infoln("zoom", img.zoom, "Scale", img.Scale, "dpi", img.DPI, "ground measure", img.groundMeasure)

	centerPt := bounds.LatLngToPoint(img.Projection, latLngCenterPt[0], latLngCenterPt[1], img.zoom, tilesize)
	// Generate the PNG
	img.image, err = image.New(
		ctx,

		img.Projection,
		int(img.imgWidth), int(img.imgHeight),
		centerPt,
		img.zoom,
		// TODO(gdey): Need to remove this hack and figure out how to used the
		// ppi value as well as set the correct scale on the svg/pdf document
		// that is produced later on. (https://github.com/go-spatial/maptoolkit/issues/13)
		1.0, // ppiRatio, (we adjust the zoom)
		0.0, // Bearing
		0.0, // Pitch
		img.Style,
		"", "",
	)
	return img.image, err
}

func (img *Img) Image() *mbgl.Image {
	image, err := img.initImage(context.Background())
	if err != nil {
		log.Infof("failed to init image: %v", err)
	}
	return image
}

func (img *Img) SetWidthHeight(w, h float64) {

	img.lck.Lock()
	img.width = w
	img.height = h
	log.Infof("Setting image dim to: %v, %v", w, h)
	img.image = nil
	img.staticWidthHeight = true
	img.lck.Unlock()
}
func (img *Img) SetWidth(w float64) float64 {
	img.lck.Lock()
	img.width = w
	log.Infof("Setting image width to: %v", w)
	img.image = nil
	img.staticWidthHeight = true
	img.lck.Unlock()
	return w
}
func (img *Img) SetHeight(h float64) float64 {
	img.lck.Lock()
	img.height = h
	log.Infof("Setting image height to: %v", h)
	img.image = nil
	img.staticWidthHeight = true
	img.lck.Unlock()
	return h
}

// Height returns the height of the image
func (img *Img) Height() int { return img.Image().Bounds().Dy() }

// Width returns the width of the image
func (img *Img) Width() int { return img.Image().Bounds().Dx() }

// Filename returns the file name of the image, it may start up the generation of
// the file
func (img *Img) Filename() (string, error) {
	if img == nil {
		return "", nil
	}
	// User only cares about filename if img.image is nil
	if img.generated {
		return img.File.Name, nil
	}
	// We need to generate the file and then return the filename
	if err := img.generateImage(); err != nil {
		log.Infof("Go error generating image: %v", err)
		return img.File.Name, err
	}
	return img.File.Name, nil
}

// Close closes out any open resources
func (img Img) Close() error {
	if !img.generated {
		return nil
	}
	return img.File.Close()
}

func (img *Img) GroundMeasure() float64 {
	img.initImage(context.Background())
	return img.groundMeasure
}
func (img *Img) Zoom() float64 {
	img.initImage(context.Background())
	return img.zoom
}

// generateImage is use to generate the image we want to store in the filestore
func (img *Img) generateImage() (err error) {

	// No file store to write out the image.
	if img.File.Store == nil || img.File.Cached() {
		return nil
	}
	img.lck.Lock()
	defer img.lck.Unlock()

	// generateImage is use to create the image into the filestore

	if img.StartGenerationCallback != nil {
		img.StartGenerationCallback()
	}
	if img.FailGenerationCallback != nil {
		defer func() {
			if err != nil {
				img.FailGenerationCallback(err)
			}
		}()
	}

	if err := img.File.Open(); err != nil {
		return err
	}

	defer img.File.Close()

	image, err := img.initImage(context.Background())
	if err != nil {
		log.Infof("got err %v generating image", err)
		return err
	}

	if err = image.GenerateImage(); err != nil {
		log.Infof("got err %v generating image", err)
		return err
	}

	if err = png.Encode(img.File, image); err != nil {
		return err
	}

	if img.EndGenerationCallback != nil {
		img.EndGenerationCallback()
	}

	img.generated = true
	return nil
}
