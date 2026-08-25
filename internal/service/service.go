package service

import (
	"sync"
	"time"

	"task253-birdbanding/internal/event"
	"task253-birdbanding/internal/identity"
	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/path"
	"task253-birdbanding/internal/ruling"
	"task253-birdbanding/internal/store"
)

// Service 编排各业务包，对外暴露高层操作。
type Service struct {
	db        *store.DB
	importer  *event.Importer
	resolver  *identity.Resolver
	builder   *path.Builder
	ruling    *ruling.Ruling
	versions  *ruling.VersionManager
	idGen     *IDGen
	mu        sync.Mutex
}

// New 构造编排服务。所有业务包共享同一 DB 与 ID 生成器。
func New(db *store.DB) *Service {
	idGen := NewIDGen()
	return &Service{
		db:       db,
		importer: event.NewImporter(db, idGen.New),
		resolver: identity.NewResolver(db, idGen.New),
		builder:  path.NewBuilder(db, idGen.New),
		ruling:   ruling.NewRuling(db),
		versions: ruling.NewVersionManager(db, idGen.New),
		idGen:    idGen,
	}
}

// DB 暴露底层数据库，供 HTTP 层只读查询使用。
func (s *Service) DB() *store.DB { return s.db }

// ---- 批次 ----

// CreateBatch 创建观测批次（录入中）。
func (s *Service) CreateBatch(name string) (*model.Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := &model.Batch{ID: s.idGen.New("bat"), Name: name, Status: model.BatchDraft, CreatedAt: time.Now().UTC()}
	if err := s.db.SaveBatch(b); err != nil {
		return nil, err
	}
	return b, nil
}

// TransitionBatch 推进批次状态。
func (s *Service) TransitionBatch(id string, to model.BatchStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.db.GetBatch(id)
	if err != nil {
		return err
	}
	if !model.BatchCanTransition(b.Status, to) {
		return model.ErrInvalidTransition
	}
	if to == model.BatchPublished {
		t := time.Now().UTC()
		b.PublishedAt = &t
	}
	b.Status = to
	return s.db.SaveBatch(b)
}

// ---- 地点 ----

// CreateLocation 创建带精度的观测地点。
func (s *Service) CreateLocation(name string, lat, lon, precisionM float64) (*model.Location, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, err := model.NewLocation(s.idGen.New("loc"), name, lat, lon, precisionM)
	if err != nil {
		return nil, err
	}
	if err := s.db.SaveLocation(l); err != nil {
		return nil, err
	}
	return l, nil
}

// ---- 事件导入与校验 ----

// ImportEvent 导入单个事件（幂等去重 + 身份关联）。
func (s *Service) ImportEvent(in event.ImportInput) (*model.Event, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.importer.Import(in)
}

// BulkImport 批量导入。
func (s *Service) BulkImport(inputs []event.ImportInput) ([]*model.Event, []error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.importer.BulkImport(inputs)
}

// ValidateEvent 校验事件并置终态。
func (s *Service) ValidateEvent(eventID string, bandingDate time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.importer.ValidateEvent(eventID, bandingDate)
}

// CorrectRing 校正环号（抄录错误修正）。
func (s *Service) CorrectRing(eventID, newRing, species string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.importer.CorrectRing(eventID, newRing, species)
}

// ExcludeEvent 排除事件。
func (s *Service) ExcludeEvent(eventID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.importer.ExcludeEvent(eventID, reason)
}

// ResolveIndividual 按环号复用或新建个体，返回个体与是否新建。
func (s *Service) ResolveIndividual(ring, species string) (*model.Individual, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolver.Resolve(ring, species)
}

// ---- 迁徙边构建 ----

// BuildEdges 为个体构建迁徙边。
func (s *Service) BuildEdges(individualID string) ([]*model.MigrationEdge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.builder.Build(individualID)
}

// ---- 异常裁决 ----

// KeepRare 保留罕见路线证据。
func (s *Service) KeepRare(edgeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ruling.KeepRare(edgeID)
}

// ExcludeOverSpeed 排除超限边。
func (s *Service) ExcludeOverSpeed(edgeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ruling.ExcludeOverSpeed(edgeID)
}

// ConfirmEdge 确认边进入路径。
func (s *Service) ConfirmEdge(edgeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ruling.ConfirmEdge(edgeID)
}

// ---- 路径版本 ----

// CreateVersion 创建路径版本（草稿）。
func (s *Service) CreateVersion(individualID, name string) (*model.PathVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.versions.Create(individualID, name)
}

// AddEdgeToVersion 向版本追加边（冻结不可改）。
func (s *Service) AddEdgeToVersion(versionID, edgeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.versions.AddEdge(versionID, edgeID)
}

// RemoveEdgeFromVersion 从版本移除边（冻结不可改）。
func (s *Service) RemoveEdgeFromVersion(versionID, edgeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.versions.RemoveEdge(versionID, edgeID)
}

// TransitionVersion 推进版本状态。
func (s *Service) TransitionVersion(versionID string, to model.VersionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.versions.Transition(versionID, to)
}

// ---- 查询 ----

// GetIndividualTimeline 返回个体事件时间线（按日期升序）。
func (s *Service) GetIndividualTimeline(individualID string) ([]*model.Event, error) {
	return s.db.EventsByIndividualSorted(individualID)
}

// Stats 返回各实体计数，用于看板概览。
func (s *Service) Stats() (map[string]int, error) {
	batches, err := s.db.ListBatches()
	if err != nil {
		return nil, err
	}
	inds, err := s.db.ListIndividuals()
	if err != nil {
		return nil, err
	}
	events, err := s.db.ListEvents("", "", "")
	if err != nil {
		return nil, err
	}
	var edgeCount, versionCount int
	for _, ind := range inds {
		es, err := s.db.ListEdges(ind.ID)
		if err != nil {
			return nil, err
		}
		edgeCount += len(es)
		vs, err := s.db.ListVersions(ind.ID)
		if err != nil {
			return nil, err
		}
		versionCount += len(vs)
	}
	return map[string]int{
		"batches":  len(batches),
		"ind":      len(inds),
		"events":   len(events),
		"edges":    edgeCount,
		"versions": versionCount,
	}, nil
}

// SelfCheck 校验数据库不变量，返回问题列表（空表示通过）。
func (s *Service) SelfCheck() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var problems []string
	events, err := s.db.ListEvents("", "", "")
	if err != nil {
		return nil, err
	}
	// 1) 重捕时间倒退不应出现在有效事件中。
	for _, e := range events {
		if e.Status == model.EventValid && e.Type == model.EventRecapture {
			// 找到同个体更早的环志事件。
			for _, o := range events {
				if o.IndividualID == e.IndividualID && o.Type == model.EventBanding && o.EventDate.After(e.EventDate) {
					problems = append(problems, "有效重捕事件 "+e.ID+" 时间早于环志事件 "+o.ID)
				}
			}
		}
	}
	// 2) 冻结版本不应包含被修改的边（这里仅检查版本状态一致性）。
	versions, err := s.listAllVersions()
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.Status == model.VersionFrozen && v.FrozenAt == nil {
			problems = append(problems, "冻结版本 "+v.ID+" 缺少冻结时间")
		}
	}
	return problems, nil
}

func (s *Service) listAllVersions() ([]*model.PathVersion, error) {
	inds, err := s.db.ListIndividuals()
	if err != nil {
		return nil, err
	}
	var out []*model.PathVersion
	for _, ind := range inds {
		vs, err := s.db.ListVersions(ind.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, vs...)
	}
	return out, nil
}
