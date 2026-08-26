package ruling

import (
	"sync"

	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/store"
)

// Ruling 处理异常重捕裁决（保留罕见证据 / 排除超限）。
type Ruling struct {
	db *store.DB
	mu sync.Mutex
}

// NewRuling 构造裁决器。
func NewRuling(db *store.DB) *Ruling {
	return &Ruling{db: db}
}

// KeepRare 保留罕见路线证据：将「罕见」边确认为有效候选（状态→确认）。
func (r *Ruling) KeepRare(edgeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	edge, err := r.db.GetEdge(edgeID)
	if err != nil {
		return err
	}
	if edge.Status != model.EdgeRare && edge.Status != model.EdgeFeasible {
		return model.ErrInvalidTransition
	}
	return r.db.UpdateEdgeStatus(edgeID, model.EdgeConfirmed, "研究者保留罕见路线证据")
}

// ExcludeOverSpeed 排除超限边：将其排除并重算（状态置回候选，理由记录）。
func (r *Ruling) ExcludeOverSpeed(edgeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	edge, err := r.db.GetEdge(edgeID)
	if err != nil {
		return err
	}
	if edge.Status != model.EdgeOverSpeed {
		return model.ErrInvalidTransition
	}
	return r.db.UpdateEdgeStatus(edgeID, model.EdgeCandidate, "研究者判定为录入错误并排除")
}

// ConfirmEdge 将可行边确认为最终路径组成。
func (r *Ruling) ConfirmEdge(edgeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	edge, err := r.db.GetEdge(edgeID)
	if err != nil {
		return err
	}
	if edge.Status == model.EdgeCandidate || edge.Status == model.EdgeFeasible ||
		edge.Status == model.EdgeRare || edge.Status == model.EdgeOverSpeed {
		return r.db.UpdateEdgeStatus(edgeID, model.EdgeConfirmed, "确认为路径组成")
	}
	return model.ErrInvalidTransition
}
