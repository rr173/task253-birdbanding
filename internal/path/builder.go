package path

import (
	"sort"
	"sync"
	"time"

	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/store"
)

// Builder 依据个体有效事件按时间排序构建迁徙边。
type Builder struct {
	db    *store.DB
	mu    sync.Mutex
	idGen func(prefix string) string
}

// NewBuilder 构造构建器。
func NewBuilder(db *store.DB, idGen func(prefix string) string) *Builder {
	return &Builder{db: db, idGen: idGen}
}

// Build 为个体构建迁徙边：取有效事件按日期排序，对连续事件对评估并落库。
func (b *Builder) Build(individualID string) ([]*model.MigrationEdge, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	events, err := b.db.EventsByIndividualSorted(individualID)
	if err != nil {
		return nil, err
	}
	var valid []*model.Event
	for _, e := range events {
		if e.Status == model.EventValid {
			valid = append(valid, e)
		}
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].EventDate.Equal(valid[j].EventDate) {
			return valid[i].ID < valid[j].ID
		}
		return valid[i].EventDate.Before(valid[j].EventDate)
	})

	locCache := map[string]*model.Location{}
	getLoc := func(id string) (*model.Location, error) {
		if l, ok := locCache[id]; ok {
			return l, nil
		}
		l, err := b.db.GetLocation(id)
		if err != nil {
			return nil, err
		}
		locCache[id] = l
		return l, nil
	}

	var edges []*model.MigrationEdge
	for i := 1; i < len(valid); i++ {
		prev, cur := valid[i-1], valid[i]
		pl, err := getLoc(prev.LocationID)
		if err != nil {
			return nil, err
		}
		cl, err := getLoc(cur.LocationID)
		if err != nil {
			return nil, err
		}
		ev := Evaluate(pl.Lat, pl.Lon, prev.EventDate, cl.Lat, cl.Lon, cur.EventDate)
		edge := &model.MigrationEdge{
			ID:           b.idGen("edge"),
			IndividualID: individualID,
			FromEventID:  prev.ID,
			ToEventID:    cur.ID,
			DistanceKm:   ev.DistanceKm,
			Days:         ev.Days,
			SpeedKmDay:   ev.SpeedKmDay,
			Status:       ev.Status,
			Reason:       ev.Reason,
			CreatedAt:    time.Now().UTC(),
		}
		if err := b.db.SaveEdge(edge); err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, nil
}
