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

func TestCorrectedRingRebuildsEventIdentityAndFingerprint(t *testing.T) {
	f, err := os.CreateTemp("", "birdbanding-correction-*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	db, err := store.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close(); _ = os.Remove(f.Name()) }()
	h := NewServer(service.New(db)).Handler()
	post := func(path string, body any) map[string]any {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code < 200 || res.Code >= 300 {
			t.Fatalf("POST %s: %d %s", path, res.Code, res.Body.String())
		}
		var out map[string]any
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	batch := post("/api/batches", map[string]string{"name": "校正批次"})["ID"].(string)
	locID := "loc-probe"
	if err := db.SaveLocation(&model.Location{ID: locID, Name: "地点", Lat: 40, Lon: 10, PrecisionM: 10}); err != nil {
		t.Fatal(err)
	}
	event := post("/api/events", map[string]string{"batch_id": batch, "ring_code": "AB1234", "type": "banding", "location_id": locID, "event_date": "2025-04-01", "species": "Anas"})
	eventID := event["ID"].(string)
	post("/api/events/"+eventID+"/correct-ring", map[string]string{"ring_code": "CD5678", "species": "Anas"})
	corrected, err := db.GetEvent(eventID)
	if err != nil {
		t.Fatal(err)
	}
	if corrected.RingCode != "CD5678" || corrected.Fingerprint != "CD5678|banding|"+locID+"|2025-04-01" {
		t.Fatalf("correction must update identity and fingerprint: %+v", corrected)
	}
	second, existed, err := service.New(db).ImportEvent(structureImport(batch, "CD5678", locID))
	if err != nil || !existed || second.ID != eventID {
		t.Fatalf("same corrected event must remain idempotent: event=%v existed=%v err=%v", second, existed, err)
	}
}

// Kept local to avoid depending on HTTP request model details in the probe.
func structureImport(batch, ring, loc string) event.ImportInput {
	return event.ImportInput{BatchID: batch, RingCode: ring, Type: model.EventBanding, LocationID: loc, EventDate: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), Species: "Anas"}
}
