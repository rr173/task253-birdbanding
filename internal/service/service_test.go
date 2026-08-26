package service

import (
	"os"
	"testing"
	"time"

	"task253-birdbanding/internal/event"
	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/store"
)

// newTestDB 创建临时数据库供测试使用。
func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	f, err := os.CreateTemp("", "birdbanding-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	_ = f.Close()
	db, err := store.Open(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(f.Name())
	})
	return db
}

// TestClosedLoop 验证完整业务闭环：导入→校验→建边→裁决→版本→自检。
func TestClosedLoop(t *testing.T) {
	svc := New(newTestDB(t))

	batch, err := svc.CreateBatch("测试批次")
	if err != nil {
		t.Fatalf("创建批次: %v", err)
	}
	locA, err := svc.CreateLocation("繁殖地", 60.0, 10.0, 500)
	if err != nil {
		t.Fatalf("地点A: %v", err)
	}
	locB, err := svc.CreateLocation("越冬地", 30.0, 10.0, 500)
	if err != nil {
		t.Fatalf("地点B: %v", err)
	}
	ind, created, err := svc.ResolveIndividual("AB1234", "Anser anser")
	if err != nil || !created {
		t.Fatalf("关联个体: %v created=%v", err, created)
	}
	banding, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "AB1234", Type: model.EventBanding,
		LocationID: locA.ID, EventDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser",
	})
	if err != nil {
		t.Fatalf("导入环志: %v", err)
	}
	recap, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "AB1234", Type: model.EventRecapture,
		LocationID: locB.ID, EventDate: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser",
	})
	if err != nil {
		t.Fatalf("导入重捕: %v", err)
	}
	if err := svc.ValidateEvent(banding.ID, time.Time{}); err != nil {
		t.Fatalf("校验环志: %v", err)
	}
	if err := svc.ValidateEvent(recap.ID, banding.EventDate); err != nil {
		t.Fatalf("校验重捕: %v", err)
	}
	edges, err := svc.BuildEdges(ind.ID)
	if err != nil {
		t.Fatalf("构建迁徙边: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("期望 1 条迁徙边, 实际 %d", len(edges))
	}
	if err := svc.ConfirmEdge(edges[0].ID); err != nil {
		t.Fatalf("确认边: %v", err)
	}
	ver, err := svc.CreateVersion(ind.ID, "v1")
	if err != nil {
		t.Fatalf("创建版本: %v", err)
	}
	if err := svc.AddEdgeToVersion(ver.ID, edges[0].ID); err != nil {
		t.Fatalf("追加边: %v", err)
	}
	if err := svc.TransitionVersion(ver.ID, model.VersionShared); err != nil {
		t.Fatalf("共享: %v", err)
	}
	if err := svc.TransitionVersion(ver.ID, model.VersionFrozen); err != nil {
		t.Fatalf("冻结: %v", err)
	}
	problems, err := svc.SelfCheck()
	if err != nil {
		t.Fatalf("自检: %v", err)
	}
	if len(problems) > 0 {
		t.Fatalf("self-check 发现问题: %v", problems)
	}
}

// TestIdempotentImport 验证重复指纹导入为幂等（不重复落库）。
func TestIdempotentImport(t *testing.T) {
	svc := New(newTestDB(t))
	batch, _ := svc.CreateBatch("批次")
	loc, _ := svc.CreateLocation("地点", 60.0, 10.0, 500)
	in := event.ImportInput{
		BatchID: batch.ID, RingCode: "CD5678", Type: model.EventBanding,
		LocationID: loc.ID, EventDate: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), Species: "Anas platyrhynchos",
	}
	e1, existed1, err := svc.ImportEvent(in)
	if err != nil || existed1 {
		t.Fatalf("首次导入应成功且未重复: %v existed=%v", err, existed1)
	}
	e2, existed2, err := svc.ImportEvent(in)
	if err != nil || !existed2 {
		t.Fatalf("重复导入应幂等: %v existed=%v", err, existed2)
	}
	if e1.ID != e2.ID {
		t.Fatalf("幂等导入应返回同一事件 ID")
	}
}

// TestInvalidTransition 验证状态机非法流转被拒绝。
func TestInvalidTransition(t *testing.T) {
	svc := New(newTestDB(t))
	batch, _ := svc.CreateBatch("批次")
	loc, _ := svc.CreateLocation("地点", 60.0, 10.0, 500)
	ev, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "EF9012", Type: model.EventBanding,
		LocationID: loc.ID, EventDate: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), Species: "Cygnus cygnus",
	})
	if err != nil {
		t.Fatalf("导入: %v", err)
	}
	// 待校验→非法目标（封存事件态不应存在，这里测事件直接排除再校验应失败）
	if err := svc.ExcludeEvent(ev.ID, "录入错误"); err != nil {
		t.Fatalf("排除: %v", err)
	}
	if err := svc.ValidateEvent(ev.ID, time.Time{}); err != model.ErrInvalidTransition {
		t.Fatalf("已排除事件再次校验应非法流转, 实际: %v", err)
	}
}

// TestBatchPublishedAtPersists 验证发布时间随发布动作固化并持久保留，
// 草稿与待复核批次不伪造发布时间。
func TestBatchPublishedAtPersists(t *testing.T) {
	svc := New(newTestDB(t))

	batch, err := svc.CreateBatch("发布批次")
	if err != nil {
		t.Fatalf("创建批次: %v", err)
	}
	// 草稿态：不应存在发布时间。
	draft, err := svc.DB().GetBatch(batch.ID)
	if err != nil {
		t.Fatalf("读取草稿批次: %v", err)
	}
	if draft.PublishedAt != nil {
		t.Fatalf("草稿批次不应有发布时间, 实际: %v", draft.PublishedAt)
	}
	if draft.Status != model.BatchDraft {
		t.Fatalf("期望草稿态, 实际: %v", draft.Status)
	}
	// 推进到待复核：仍不应伪造发布时间。
	if err := svc.TransitionBatch(batch.ID, model.BatchReview); err != nil {
		t.Fatalf("流转到待复核: %v", err)
	}
	review, err := svc.DB().GetBatch(batch.ID)
	if err != nil {
		t.Fatalf("读取待复核批次: %v", err)
	}
	if review.PublishedAt != nil {
		t.Fatalf("待复核批次不应有发布时间, 实际: %v", review.PublishedAt)
	}

	// 发布：应固化发布时间。
	if err := svc.TransitionBatch(batch.ID, model.BatchPublished); err != nil {
		t.Fatalf("发布: %v", err)
	}
	first, err := svc.DB().GetBatch(batch.ID)
	if err != nil {
		t.Fatalf("读取已发布批次: %v", err)
	}
	if first.Status != model.BatchPublished {
		t.Fatalf("期望已发布态, 实际: %v", first.Status)
	}
	if first.PublishedAt == nil {
		t.Fatalf("已发布批次的发布时间为空（旧 BUG 回归）")
	}
	if first.PublishedAt.After(time.Now().UTC().Add(5 * time.Second)) {
		t.Fatalf("发布时间不应指向未来: %v", first.PublishedAt)
	}

	// 重新读取（模拟研究者再次打开批次详情）：发布时间应持续保留。
	reopened, err := svc.DB().GetBatch(batch.ID)
	if err != nil {
		t.Fatalf("重读批次: %v", err)
	}
	if reopened.PublishedAt == nil {
		t.Fatalf("重新读取批次后发布时间丢失")
	}
	if !reopened.PublishedAt.Equal(*first.PublishedAt) {
		t.Fatalf("发布时间不一致: 首次 %v 重读 %v", first.PublishedAt, reopened.PublishedAt)
	}

	// 封存已发布批次：发布时间应保留，不被清空。
	if err := svc.TransitionBatch(batch.ID, model.BatchSealed); err != nil {
		t.Fatalf("封存: %v", err)
	}
	sealed, err := svc.DB().GetBatch(batch.ID)
	if err != nil {
		t.Fatalf("读取封存批次: %v", err)
	}
	if sealed.PublishedAt == nil || !sealed.PublishedAt.Equal(*first.PublishedAt) {
		t.Fatalf("封存后发布时间应保留不变, 实际: %v", sealed.PublishedAt)
	}
}
