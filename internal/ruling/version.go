package ruling

import (
	"sync"
	"time"

	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/store"
)

// VersionManager 管理路径版本的创建与状态流转（草稿→共享→冻结→替代）。
type VersionManager struct {
	db    *store.DB
	mu    sync.Mutex
	idGen func(prefix string) string
}

// NewVersionManager 构造版本管理器。
func NewVersionManager(db *store.DB, idGen func(prefix string) string) *VersionManager {
	return &VersionManager{db: db, idGen: idGen}
}

// Create 创建个体路径版本（草稿态）。
func (vm *VersionManager) Create(individualID, name string) (*model.PathVersion, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	v := &model.PathVersion{
		ID:           vm.idGen("ver"),
		IndividualID: individualID,
		Name:         name,
		Status:       model.VersionDraft,
		CreatedAt:    time.Now().UTC(),
	}
	if err := vm.db.SaveVersion(v); err != nil {
		return nil, err
	}
	return v, nil
}

// AddEdge 向版本追加一条边（冻结版本不可修改）。
func (vm *VersionManager) AddEdge(versionID, edgeID string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	v, err := vm.db.GetVersion(versionID)
	if err != nil {
		return err
	}
	// 冻结与已替代均为不可变快照，禁止再追加或移除边成员。
	if v.Status.IsImmutable() {
		return model.ErrFrozenImmutable
	}
	existing, err := vm.db.ListVersionEdges(versionID)
	if err != nil {
		return err
	}
	return vm.db.AddEdgeToVersion(versionID, edgeID, len(existing)+1)
}

// RemoveEdge 从版本移除一条边（冻结版本不可修改）。
func (vm *VersionManager) RemoveEdge(versionID, edgeID string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	v, err := vm.db.GetVersion(versionID)
	if err != nil {
		return err
	}
	if v.Status.IsImmutable() {
		return model.ErrFrozenImmutable
	}
	return vm.db.RemoveEdgeFromVersion(versionID, edgeID)
}

// Transition 校验并推进版本状态。
func (vm *VersionManager) Transition(versionID string, to model.VersionStatus) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	v, err := vm.db.GetVersion(versionID)
	if err != nil {
		return err
	}
	if !model.VersionCanTransition(v.Status, to) {
		return model.ErrInvalidTransition
	}
	var frozenAt *time.Time
	if to == model.VersionFrozen {
		t := time.Now().UTC()
		frozenAt = &t
	}
	return vm.db.UpdateVersionStatus(versionID, to, frozenAt)
}

// Supersede 将被替代版本置为「替代」，并使其边成员失效。
func (vm *VersionManager) Supersede(versionID string) error {
	return vm.Transition(versionID, model.VersionSuperseded)
}
