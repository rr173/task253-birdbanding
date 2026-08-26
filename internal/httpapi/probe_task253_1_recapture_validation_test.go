package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"task253-birdbanding/internal/event"
	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/service"
	"task253-birdbanding/internal/store"
)

func TestRecaptureReversalCannotBeBypassedThroughHTTP(t *testing.T) {
	f, err := os.CreateTemp("", "task253-bug1-*.db")
	if err != nil { t.Fatal(err) }
	_ = f.Close()
	db, err := store.Open(f.Name())
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = db.Close(); _ = os.Remove(f.Name()) })
	svc := service.New(db)
	batch, err := svc.CreateBatch("倒退校验")
	if err != nil { t.Fatal(err) }
	loc, err := svc.CreateLocation("观测点", 30, 10, 100)
	if err != nil { t.Fatal(err) }
	banding, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "AB1234", Type: model.EventBanding,
		LocationID: loc.ID, EventDate: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), Species: "duck",
	})
	if err != nil { t.Fatal(err) }
	recapture, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "AB1234", Type: model.EventRecapture,
		LocationID: loc.ID, EventDate: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), Species: "duck",
	})
	if err != nil { t.Fatal(err) }
	if err := svc.ValidateEvent(banding.ID, time.Time{}); err != nil { t.Fatal(err) }
	h := NewServer(svc).Handler()
	body := bytes.NewBufferString(`{"banding_date":"2025-05-01"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+recapture.ID+"/validate", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("倒退重捕应返回 400，实际 %d: %s", res.Code, res.Body.String())
	}
	var got model.Event
	if err := json.NewDecoder(bytes.NewReader(res.Body.Bytes())).Decode(&got); err == nil && got.Status == model.EventValid {
		t.Fatal("倒退重捕不应被置为有效")
	}
}
