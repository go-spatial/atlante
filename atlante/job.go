//go:generate protoc "--go_out=." "job.proto"
package atlante

import (
	"encoding/base64"

	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
)

type Notifier interface {
	Notify(map[string]string) error
}

func job_grid_mdgid_for_mdgid(mdgid grids.MDGID) *Job_Grid_MDGID {
	return &Job_Grid_MDGID{
		Id:   mdgid.ID,
		Part: uint32(mdgid.Part),
	}
}

func (jgln *Job_Grid_LatLng) LatLng() (float64, float64) {
	if jgln == nil {
		return 0, 0
	}
	return float64(jgln.Lat), float64(jgln.Lng)
}

func job_grid_latlng_for_lat_lng(lat, lng float64) *Job_Grid_LatLng {
	return &Job_Grid_LatLng{
		Lat: float32(lat),
		Lng: float32(lng),
	}
}

func job_grid_editinfo_for_editinfo(ei *grids.EditInfo) *Job_Grid_EditInfo {
	ts, _ := ptypes.TimestampProto(ei.Date)
	return &Job_Grid_EditInfo{
		By:   ei.By,
		Date: ts,
	}
}

func job_grid_for_grid(grid *grids.Grid) *Job_Grid {
	if grid == nil {
		return nil
	}

	ts, _ := ptypes.TimestampProto(grid.PublicationDate)

	return &Job_Grid{
		Mdgid: &Job_Grid_MDGID{
			Id:   grid.MdgID.ID,
			Part: uint32(grid.MdgID.Part),
		},
		Sw:  job_grid_latlng_for_lat_lng(grid.SWLat, grid.SWLng),
		Ne:  job_grid_latlng_for_lat_lng(grid.NELat, grid.NELng),
		Len: job_grid_latlng_for_lat_lng(grid.LatLen, grid.LngLen),

		Nrn:     grid.NRN,
		Country: grid.Country,
		City:    grid.City,
		Sheet:   grid.Sheet,

		Srid:        uint32(grid.SRID),
		MetaData:    grid.Metadata,
		PublishedAt: ts,

		Edited: job_grid_editinfo_for_editinfo(grid.Edited),
	}
}

// NewJob returns a new job object for the given sheet, grid and metadata
func NewJob(sheet string, grid *grids.Grid, metadata map[string]string) *Job {
	return &Job{
		SheetName: sheet,
		Grid:      job_grid_for_grid(grid),
		MetaData:  metadata,
	}
}

func (j *Job_Grid) MdgID() grids.MDGID {
	m := j.GetMdgid()
	if m == nil {
		return grids.MDGID{}
	}
	return grids.MDGID{
		ID:   m.Id,
		Part: uint(m.Part),
	}
}

func (j *Job) GridsGrid() *grids.Grid {
	g := j.Grid
	if j == nil || g == nil {
		return nil
	}

	publishedAt, _ := ptypes.Timestamp(g.GetPublishedAt())
	edited, _ := ptypes.Timestamp(g.GetEdited().GetDate())

	return &grids.Grid{
		MdgID:   g.MdgID(),
		SRID:    uint(g.GetSrid()),
		Country: g.GetCountry(),
		City:    g.GetCity(),
		NRN:     g.GetNrn(),

		SWLat:  float64(g.GetSw().GetLat()),
		SWLng:  float64(g.GetSw().GetLng()),
		NELat:  float64(g.GetNe().GetLat()),
		NELng:  float64(g.GetNe().GetLng()),
		LatLen: float64(g.GetLen().GetLat()),
		LngLen: float64(g.GetLen().GetLng()),
		Sheet:  g.GetSheet(),

		Metadata:        g.GetMetaData(),
		PublicationDate: publishedAt,
		Edited: &grids.EditInfo{
			Date: edited,
			By:   g.GetEdited().GetBy(),
		},
	}
}

// Base64Marshal returns the job encode in a based64 string
func (j *Job) Base64Marshal() (string, error) {
	// first marshal to pbf
	data, err := proto.Marshal(j)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal")
	}
	// Now marshal the []byte to base64
	return base64.StdEncoding.EncodeToString(data), nil
}

// Base64UnmarshalJob will return a Job object for the encode job string
func Base64UnmarshalJob(str string) (*Job, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, errors.Wrap(err, "failed to base64 decode")
	}
	var jb Job
	if err := proto.Unmarshal(data, &jb); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal protobuf")
	}
	return &jb, nil
}
