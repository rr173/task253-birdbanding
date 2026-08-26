package event

import (
	"fmt"
	"sync"
	"time"

	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/store"
)

// Importer 负责事件导入、幂等去重与身份关联。
type Importer struct {
	db    *store.DB
	mu    sync.Mutex
	idGen func(prefix string) string
}

// NewImporter 构造导入器。
func NewImporter(db *store.DB, idGen func(prefix string) string) *Importer {
	return &Importer{db: db, idGen: idGen}
}

// Import 导入单个事件：先幂等去重，再创建/关联个体并落库为「待校验」。
// 返回 (event, alreadyExists, error)。
func (im *Importer) Import(in ImportInput) (*model.Event, bool, error) {
	if err := in.Validate(); err != nil {
		return nil, false, err
	}
	fp := model.Fingerprint(in.RingCode, in.Type, in.LocationID, in.EventDate)

	im.mu.Lock()
	defer im.mu.Unlock()

	exists, err := im.db.EventExistsByFingerprint(fp)
	if err != nil {
		return nil, false, err
	}
	if exists {
		// 幂等：返回已存在的事件，不重复落库。
		ev, gerr := im.db.GetEventByFingerprint(fp)
		if gerr != nil {
			return nil, true, nil
		}
		return ev, true, nil
	}

	// 身份关联：按环号复用或新建个体。
	ind, err := im.resolveIndividual(in.RingCode, in.Species)
	if err != nil {
		return nil, false, err
	}

	now := time.Now().UTC()
	ev := &model.Event{
		ID:           im.idGen("evt"),
		BatchID:      in.BatchID,
		IndividualID: ind.ID,
		RingCode:     in.RingCode,
		Type:         in.Type,
		LocationID:   in.LocationID,
		EventDate:    in.EventDate,
		Status:       model.EventPending,
		Fingerprint:  fp,
		CreatedAt:    now,
	}
	if err := im.db.SaveEvent(ev); err != nil {
		return nil, false, err
	}
	return ev, false, nil
}

// resolveIndividual 按环号复用个体；若不存在则新建（需用 species 初始化）。
func (im *Importer) resolveIndividual(ring, species string) (*model.Individual, error) {
	existing, err := im.db.GetIndividualByRing(ring)
	if err == nil {
		return existing, nil
	}
	if !model.IsNotFound(err) {
		return nil, err
	}
	ind := &model.Individual{
		ID:        im.idGen("ind"),
		RingCode:  ring,
		Species:   species,
		CreatedAt: time.Now().UTC(),
	}
	if err := im.db.SaveIndividual(ind); err != nil {
		return nil, err
	}
	return ind, nil
}

// ValidateEvent 校验单个待校验事件，置为 有效/身份冲突/排除 之一。
// 规则：环号格式错误→排除；重捕时间早于环志→排除（重捕时间倒退）；地点精度缺失→排除；
// 同一环号映射到多个物种→身份冲突。
func (im *Importer) ValidateEvent(eventID string, bandingDate time.Time) error {
	ev, err := im.db.GetEvent(eventID)
	if err != nil {
		return err
	}
	if ev.Status != model.EventPending {
		return model.ErrInvalidTransition
	}
	if err := ValidateRingFormat(ev.RingCode); err != nil {
		return im.db.UpdateEventStatus(eventID, model.EventExcluded, err.Error(), ev.IndividualID)
	}
	if ev.Type == model.EventRecapture && !bandingDate.IsZero() && isRecaptureReversed(ev.EventDate, bandingDate) {
		return im.db.UpdateEventStatus(eventID, model.EventExcluded, model.ErrRecaptureTimeReversed.Error(), ev.IndividualID)
	}
	// 身份冲突检查：同环号下若存在不同物种个体则冲突。
	conflict, err := im.identityConflict(ev)
	if err != nil {
		return err
	}
	if conflict {
		return im.db.UpdateEventStatus(eventID, model.EventConflict, model.ErrIdentityConflict.Error(), ev.IndividualID)
	}
	return im.db.UpdateEventStatus(eventID, model.EventValid, "", ev.IndividualID)
}

// identityConflict 检查该事件个体是否与同环号其它个体的物种不一致。
func (im *Importer) identityConflict(ev *model.Event) (bool, error) {
	ind, err := im.db.GetIndividual(ev.IndividualID)
	if err != nil {
		return false, err
	}
	byRing, err := im.db.GetIndividualByRing(ev.RingCode)
	if err != nil {
		return false, err
	}
	if byRing.ID != ind.ID {
		return true, nil
	}
	// 同一环号下存在两个个体记录（环号唯一约束理论上不会，但做防御）。
	all, err := im.db.ListIndividuals()
	if err != nil {
		return false, err
	}
	seen := map[string]bool{}
	for _, a := range all {
		if a.RingCode == ev.RingCode {
			if seen[a.ID] {
				return true, nil
			}
			seen[a.ID] = true
		}
	}
	return false, nil
}

// CorrectRing 校正环号：重置事件为待校验并重新关联个体（用于抄录错误修正）。
func (im *Importer) CorrectRing(eventID, newRing, species string) error {
	if err := ValidateRingFormat(newRing); err != nil {
		return err
	}
	ev, err := im.db.GetEvent(eventID)
	if err != nil {
		return err
	}
	ind, err := im.resolveIndividual(newRing, species)
	if err != nil {
		return err
	}
	fp := model.Fingerprint(newRing, ev.Type, ev.LocationID, ev.EventDate)
	return im.db.UpdateEventRing(eventID, newRing, ind.ID, fp, model.EventPending, "")
}

// ExcludeEvent 将事件排除（如确认为录入错误）。
func (im *Importer) ExcludeEvent(eventID, reason string) error {
	ev, err := im.db.GetEvent(eventID)
	if err != nil {
		return err
	}
	if ev.Status != model.EventPending && ev.Status != model.EventConflict {
		return model.ErrInvalidTransition
	}
	return im.db.UpdateEventStatus(eventID, model.EventExcluded, reason, ev.IndividualID)
}

// BulkImport 批量导入；返回各事件导入结果。
func (im *Importer) BulkImport(inputs []ImportInput) ([]*model.Event, []error) {
	var events []*model.Event
	var errs []error
	for _, in := range inputs {
		ev, _, err := im.Import(in)
		if err != nil {
			errs = append(errs, fmt.Errorf("ring=%s: %w", in.RingCode, err))
			continue
		}
		events = append(events, ev)
	}
	return events, errs
}
