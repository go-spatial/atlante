package image

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"math"
	"os"
	"sync"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/maptoolkit/mbgl"
	"github.com/go-spatial/maptoolkit/mbgl/bounds"
	"github.com/prometheus/common/log"
)

const (
	tilesize = 4096 / 2
	scale    = 4
)

// ReadAtCloser is a ReaderAt that can be closed as well
type ReadAtCloser interface {
	io.ReaderAt
	io.Closer
}

// CenterRect d4scribes the center point of a the rectangle for a given lat/lng
type CenterRect struct {
	Lat  float64
	Lng  float64
	Rect image.Rectangle
	// for backing store
	offset   int64
	length   int
	imgWidth int
}

// Image draws a raster image of the vector image of the request widht and height
type Image struct {
	// Width of the desired image, it will be multipiled by the PPIRatio to get the final width
	width int
	// Height of the desired image, it will be multipiled by the PPIRatio to get the final height.
	height int

	// PPIRatio
	ppiratio float64

	// These are the centers and the rectangles where the image will be
	// placed
	centers []CenterRect

	// the offset from the top, this is for clip
	offsetHeight int
	offsetWidth  int

	// Style to use to generate the tile
	style string
	// The zoom level
	zoom float64

	// Projection
	prj bounds.AProjection

	// We will write the data to this and then use this for the
	// At function.
	backingStore *os.File
	initLck      sync.Mutex
	initilized   bool

	numberOfTilesNeeded int
	centerXY            [2]float64

	// this is for debugging.
	drawBounds bool
	// bounds need to be in the coordinate system of the image.
	// the color will be black
	bounds     [4]float64
	fullBounds image.Rectangle
}

// SetDebugBounds draws a black line around the border of the image
func (img *Image) SetDebugBounds(extent *geom.Extent, zoom float64) {

	// for lat lng geom.Extent should be laid out as follows:
	// {west, south, east, north}
	ne := [2]float64{extent[3], extent[2]}
	sw := [2]float64{extent[1], extent[0]}

	swPt := bounds.LatLngToPoint(img.prj, sw[0], sw[1], zoom, tilesize)
	nePt := bounds.LatLngToPoint(img.prj, ne[0], ne[1], zoom, tilesize)
	img.drawBounds = true
	img.bounds = [4]float64{
		float64(int((nePt[0] - float64(img.fullBounds.Min.X)) / scale)),
		float64(int((swPt[1] - float64(img.fullBounds.Min.Y)) / scale)),
		float64(int((swPt[0] - float64(img.fullBounds.Min.X)) / scale)),
		float64(int((nePt[1] - float64(img.fullBounds.Min.Y)) / scale)),
	}
}

// ColorModel returns that the image is a RGBA image
func (Image) ColorModel() color.Model { return color.RGBAModel }

// Bounds is the size of the actual image
func (img Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, int(float64(img.width)*img.ppiratio), int(float64(img.height)*img.ppiratio))
}

// Close will close the backing store and remove it.
func (im Image) Close() {
	if im.backingStore == nil {
		return
	}
	// Want to make sure generate/At don't try to use this if it's closing
	im.initLck.Lock()
	defer im.initLck.Unlock()
	if err := im.backingStore.Close(); err != nil {
		log.Printf("warning failed to close %v : %v", im.backingStore.Name(), err)
	}
	log.Printf("removing backing store %v", im.backingStore.Name())
	// ignore any errors.
	if err := os.Remove(im.backingStore.Name()); err != nil {
		log.Printf("warning failed to remove %v : %v", im.backingStore.Name(), err)
	}
	im.backingStore = nil
	im.initilized = false
}

//At returns the color for the x,y position in the image.
func (img Image) At(x, y int) color.Color {
	if !img.initilized {
		if err := img.GenerateImage(); err != nil {
			log.Info("got error generating image")
			// Failed to generate the image, just return black
			return color.RGBA{0, 0, 0, 255}
		}
	}
	rx, ry := x+img.offsetWidth, y+img.offsetHeight
	// rx, ry := x, y
	var data [4]byte

	if img.drawBounds {
		if int(img.bounds[0]) == rx || int(img.bounds[2]) == rx || int(img.bounds[1]) == ry || int(img.bounds[3]) == ry {
			return color.RGBA{0, 0, 0, 255}
		}
	}

	// We need to look through the centers to find the first rect that containts this x,y
	for i := range img.centers {
		rect := img.centers[i].Rect
		if rect.Min.X <= rx && rx <= rect.Max.X && rect.Min.Y <= ry && ry <= rect.Max.Y {
			dx, dy := rx-rect.Min.X, ry-rect.Min.Y
			idx := int64(img.centers[i].imgWidth*4*dy+(4*dx)) + (img.centers[i].offset)
			_, err := img.backingStore.ReadAt(data[:], idx)
			if err != nil {
				panic(fmt.Sprintf("(%v,%v) -> Centers[%v]{ %v }: %v Got an error reading backing store: %v", x, y, i, img.centers[i], idx, err))
			}
			return color.RGBA{data[0], data[1], data[2], data[3]}
		}
	}
	panic(fmt.Sprintf("Did not find expected offset %v,%v -- %v,%v", x, y, rx, ry))
	return color.RGBA{}
}

// New returns a new image with the desired properties
func New(
	prj bounds.AProjection,
	desiredWidth, desiredHeight int,
	centerXY [2]float64,
	zoom float64,
	ppi, pitch, bearing float64,
	style string,
	tempDir, tempFilename string,
) (*Image, error) {

	const tilesize = 4096 / 2
	const scale = 4

	numTilesNeeded := int(
		math.Ceil((math.Max(
			float64(desiredWidth),
			float64(desiredHeight),
		)/tilesize + 1) / 2,
		),
	)
	log.Infoln("desiredWidth", desiredWidth)
	log.Infoln("desiredHeight", desiredHeight)

	tmpDir := "."
	if tempDir == "" {
		tmpDir = tempDir
	}
	tmpFilename := "image_backingstore.bin."
	if tempFilename == "" {
		tmpFilename = tempFilename
	}

	tmpfile, err := ioutil.TempFile(tmpDir, tmpFilename)
	if err != nil {
		return nil, fmt.Errorf("Failed to setup backing store: %v", err)
	}

	log.Infoln("numbTilesNeeded", numTilesNeeded)

	img := Image{
		prj:                 prj,
		style:               style,
		zoom:                zoom,
		width:               desiredWidth,
		height:              desiredHeight,
		ppiratio:            ppi,
		numberOfTilesNeeded: numTilesNeeded,
		centers:             make([]CenterRect, 0, numTilesNeeded*numTilesNeeded),
		centerXY:            centerXY,
		backingStore:        tmpfile,
	}

	return &img, nil
}

// GenerateImage will attempt to generate the backing store.
// This will be call automatically when At() is called, but
// the error will be lost.
func (img *Image) GenerateImage() error {
	if img == nil || img.initilized {
		return nil
	}
	img.initLck.Lock()
	defer img.initLck.Unlock()
	numTilesNeeded := img.numberOfTilesNeeded
	centerXY := img.centerXY
	prj := img.prj
	zoom := img.zoom
	style := img.style
	ppi := img.ppiratio
	desiredWidth := img.width
	desiredHeight := img.height
	centerTileLength := int(math.Ceil((tilesize - 1) * ppi))

	ry := 0
	rx := 0
	bsOffset := int64(0)
	var min image.Point
	for y := -numTilesNeeded; y <= numTilesNeeded; y++ {
		rx = 0
		for x := -numTilesNeeded; x <= numTilesNeeded; x++ {

			var crect CenterRect
			center := [2]float64{centerXY[0] + (float64(x*tilesize) * scale), centerXY[1] + (float64(y*tilesize) * scale)}
			crect.Lat, crect.Lng = bounds.PointToLatLng(prj, center, zoom, tilesize)
			crect.Rect = image.Rect(rx, ry, rx+centerTileLength, ry+centerTileLength)
			// This is the top-left most coner
			if rx == 0 && ry == 0 {
				min.X, min.Y = int(center[0]-(tilesize/2*(scale))), int(center[1]-(tilesize/2*(scale)))
			}
			snpsht := mbgl.Snapshotter{
				Style:    style,
				Width:    uint32(tilesize),
				Height:   uint32(tilesize),
				PPIRatio: ppi,
				Lat:      crect.Lat,
				Lng:      crect.Lng,
				Zoom:     zoom,
			}
			snpImage, err := mbgl.Snapshot(snpsht)
			if err != nil {
				// Delete the tempfile -- don't need to worry about the error.
				// we want to shadow the err here
				img.Close()
				return err
			}
			crect.length, err = img.backingStore.Write(snpImage.Data)
			if err != nil {
				// Delete the tempfile -- don't need to worry about the error.
				// we want to shadow the err here
				img.Close()
				return err
			}
			crect.offset = bsOffset
			crect.imgWidth = snpImage.Width
			img.centers = append(img.centers, crect)

			bsOffset += int64(crect.length)
			rx += centerTileLength
		}
		ry += centerTileLength
	}
	img.fullBounds.Min = min
	img.offsetWidth = (rx / 2) - int(float64(desiredWidth/2)*ppi)
	img.offsetHeight = (ry / 2) - int(float64(desiredHeight/2)*ppi)

	img.initilized = true
	err := img.backingStore.Sync()
	// Move to the top of the file.
	log.Infof("Backing store has been sync'd : %v -- %v", img.backingStore.Name(), err)
	if err != nil {
		return err
	}
	_, _ = img.backingStore.Seek(0, 0)
	return nil
}
