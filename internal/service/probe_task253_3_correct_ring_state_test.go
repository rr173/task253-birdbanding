package service

import (
	"testing"
	"time"

	"task253-birdbanding/internal/event"
	"task253-birdbanding/internal/model"
)

func TestCorrectRingResetsValidatedEvent(t *testing.T) {
	svc := New(newTestDB(t))
	b, _ := svc.CreateBatch("校正")
	l, _ := svc.CreateLocation("点", 10, 10, 10)
	ev, _, err := svc.ImportEvent(event.ImportInput{BatchID: b.ID, RingCode: "AB1234", Type: model.EventBanding, LocationID: l.ID, EventDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Species: "duck"})
	if err != nil { t.Fatal(err) }
	if err := svc.ValidateEvent(ev.ID, time.Time{}); err != nil { t.Fatal(err) }
	if err := svc.CorrectRing(ev.ID, "CD5678", "duck"); err != nil { t.Fatal(err) }
	got, err := svc.DB().GetEvent(ev.ID)
	if err != nil { t.Fatal(err) }
	if got.Status != model.EventPending { t.Fatalf("校正后应回到待校验，实际 %q", got.Status) }
}
