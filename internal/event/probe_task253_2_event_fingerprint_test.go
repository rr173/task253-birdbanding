package event_test

import (
	"os"
	"testing"
	"time"

	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/service"
	"task253-birdbanding/internal/store"
	"task253-birdbanding/internal/event"
)

func TestBandingAndRecaptureDoNotShareFingerprint(t *testing.T) {
	f, err := os.CreateTemp("", "task253-bug2-*.db")
	if err != nil { t.Fatal(err) }
	_ = f.Close()
	db, err := store.Open(f.Name())
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = db.Close(); _ = os.Remove(f.Name()) })
	svc := service.New(db)
	b, _ := svc.CreateBatch("指纹")
	l, _ := svc.CreateLocation("同点", 20, 10, 10)
	d := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	first, existed, err := svc.ImportEvent(event.ImportInput{BatchID: b.ID, RingCode: "AB1234", Type: model.EventBanding, LocationID: l.ID, EventDate: d, Species: "duck"})
	if err != nil || existed { t.Fatalf("环志导入失败: %v existed=%v", err, existed) }
	second, existed, err := svc.ImportEvent(event.ImportInput{BatchID: b.ID, RingCode: "AB1234", Type: model.EventRecapture, LocationID: l.ID, EventDate: d, Species: "duck"})
	if err != nil || existed { t.Fatalf("重捕不应被幂等合并: %v existed=%v", err, existed) }
	if first.ID == second.ID { t.Fatal("环志和重捕被错误合并") }
}
