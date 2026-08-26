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

// TestDistinctEventTypesNotDeduped 验证同一鸟、同一天、同一地点的环志与重捕
// 因事件类型不同应分别保存，不被指纹去重吞并；同时真正重复导入仍幂等。
func TestDistinctEventTypesNotDeduped(t *testing.T) {
	svc := New(newTestDB(t))
	batch, _ := svc.CreateBatch("批次")
	loc, _ := svc.CreateLocation("地点", 60.0, 10.0, 500)
	date := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	banding, existedB, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "GH2468", Type: model.EventBanding,
		LocationID: loc.ID, EventDate: date, Species: "Turdus merula",
	})
	if err != nil || existedB {
		t.Fatalf("导入环志应成功且非重复: err=%v existed=%v", err, existedB)
	}
	recap, existedR, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "GH2468", Type: model.EventRecapture,
		LocationID: loc.ID, EventDate: date, Species: "Turdus merula",
	})
	if err != nil {
		t.Fatalf("导入重捕应成功: %v", err)
	}
	if existedR {
		t.Fatalf("同日同地不同类型重捕被误判为重复，研究者无法看到两次观测")
	}
	if banding.ID == recap.ID {
		t.Fatalf("环志与重捕应分别保存为不同事件，却返回了同一 ID")
	}
	if banding.Type == recap.Type {
		t.Fatalf("两次观测类型应不同: 都=%s", banding.Type)
	}

	// 真正重复导入（类型完全一致）仍应幂等去重。
	dup, existedDup, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "GH2468", Type: model.EventRecapture,
		LocationID: loc.ID, EventDate: date, Species: "Turdus merula",
	})
	if err != nil || !existedDup {
		t.Fatalf("真正重复导入应幂等: err=%v existed=%v", err, existedDup)
	}
	if dup.ID != recap.ID {
		t.Fatalf("重复导入应返回已存在事件 ID")
	}

	// 数据库应恰好保存两条不同类型事件。
	events, err := svc.DB().ListEvents(batch.ID, "", "")
	if err != nil {
		t.Fatalf("列出事件: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("应保存 2 条事件（环志+重捕），实际 %d", len(events))
	}
}
