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

// TestBulkImportIsolatesErrors 验证批量导入的错误隔离：首条非法记录报错，
// 但不阻断同一批次中其余合法记录的导入。
func TestBulkImportIsolatesErrors(t *testing.T) {
	svc := New(newTestDB(t))
	batch, _ := svc.CreateBatch("批量批次")
	loc, _ := svc.CreateLocation("地点", 60.0, 10.0, 500)
	inputs := []event.ImportInput{
		// 第一条环号格式错误（非法），应被隔离。
		{BatchID: batch.ID, RingCode: "AB-12", Type: model.EventBanding,
			LocationID: loc.ID, EventDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser"},
		// 第二条合法，应被正常导入。
		{BatchID: batch.ID, RingCode: "CD5678", Type: model.EventBanding,
			LocationID: loc.ID, EventDate: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), Species: "Anas platyrhynchos"},
		// 第三条合法，应被正常导入。
		{BatchID: batch.ID, RingCode: "EF9012", Type: model.EventBanding,
			LocationID: loc.ID, EventDate: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), Species: "Cygnus cygnus"},
	}

	events, errs := svc.BulkImport(inputs)
	if len(events) != 2 {
		t.Fatalf("应导入 2 条合法记录, 实际 %d", len(events))
	}
	if len(errs) != 1 {
		t.Fatalf("应报告 1 条错误, 实际 %d", len(errs))
	}
	// 合法记录的环号应被正确保存。
	got := map[string]bool{}
	for _, ev := range events {
		got[ev.RingCode] = true
	}
	for _, ring := range []string{"CD5678", "EF9012"} {
		if !got[ring] {
			t.Fatalf("合法记录 %s 未被保存", ring)
		}
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
