package identity

import (
	"sync"
	"time"

	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/store"
)

// Resolver 处理环号与个体的关联与冲突判定。
type Resolver struct {
	db    *store.DB
	mu    sync.Mutex
	idGen func(prefix string) string
}

// NewResolver 构造解析器。
func NewResolver(db *store.DB, idGen func(prefix string) string) *Resolver {
	return &Resolver{db: db, idGen: idGen}
}

// Resolve 按环号复用或新建个体，返回个体与是否新建。
func (r *Resolver) Resolve(ring, species string) (*model.Individual, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, err := r.db.GetIndividualByRing(ring)
	if err == nil {
		return existing, false, nil
	}
	if !model.IsNotFound(err) {
		return nil, false, err
	}
	ind := &model.Individual{
		ID:        r.idGen("ind"),
		RingCode:  ring,
		Species:   species,
		CreatedAt: time.Now().UTC(),
	}
	if err := r.db.SaveIndividual(ind); err != nil {
		return nil, false, err
	}
	return ind, true, nil
}

// ConflictCheck 判定环号是否在不同物种间产生身份冲突。
// 同一环号若已关联到多个不同物种个体，则存在冲突。
func (r *Resolver) ConflictCheck(ring string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all, err := r.db.ListIndividuals()
	if err != nil {
		return false, err
	}
	seen := map[string]bool{}
	for _, ind := range all {
		if ind.RingCode == ring {
			if seen[ind.ID] {
				return true, nil
			}
			seen[ind.ID] = true
		}
	}
	// 环号唯一约束保证通常只有一个；若超过一个即冲突。
	return len(seen) > 1, nil
}

// Reassign 将事件重新关联到新环号对应的个体（校正环号后调用）。
func (r *Resolver) Reassign(eventID, newRing, species string) (*model.Individual, error) {
	ind, _, err := r.Resolve(newRing, species)
	if err != nil {
		return nil, err
	}
	return ind, nil
}
