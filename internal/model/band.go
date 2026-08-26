package model

import "time"

// Batch 观测批次：一次野外调查导入的事件集合。
type Batch struct {
	ID          string
	Name        string
	Status      BatchStatus
	CreatedAt   time.Time
	PublishedAt *time.Time
}

// Location 观测地点，含坐标与定位精度（米）。
type Location struct {
	ID         string
	Name       string
	Lat        float64
	Lon        float64
	PrecisionM float64
	CreatedAt  time.Time
}

// NewLocation 构造地点并校验精度非空。
func NewLocation(id, name string, lat, lon, precisionM float64) (*Location, error) {
	if precisionM <= 0 {
		return nil, ErrLocationPrecisionMissing
	}
	return &Location{
		ID:         id,
		Name:       name,
		Lat:        lat,
		Lon:        lon,
		PrecisionM: precisionM,
		CreatedAt:  time.Now().UTC(),
	}, nil
}
