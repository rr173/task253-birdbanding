package httpapi

import (
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

func TestRouterExposesHealthAndTimelineRoutes(t *testing.T) {
	f, err := os.CreateTemp("", "birdbanding-router-*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	db, err := store.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(f.Name())
	})

	h := NewServer(service.New(db)).Handler()
	for _, path := range []string{"/api/health", "/api/stats", "/api/selfcheck"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("%s 返回 %d: %s", path, res.Code, res.Body.String())
		}
	}
}

// TestIndividualTimelineRoute 验证研究者读取个体观测时间线：
// 存在的个体按日期升序返回观测记录；不存在的个体仍返回未找到。
func TestIndividualTimelineRoute(t *testing.T) {
	f, err := os.CreateTemp("", "birdbanding-timeline-*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	db, err := store.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(f.Name())
	})

	svc := service.New(db)
	h := NewServer(svc).Handler()

	batch, err := svc.CreateBatch("测试批次")
	if err != nil {
		t.Fatal(err)
	}
	loc, err := svc.CreateLocation("繁殖地", 60.0, 10.0, 500)
	if err != nil {
		t.Fatal(err)
	}
	ind, _, err := svc.ResolveIndividual("AB1234", "Anser anser")
	if err != nil {
		t.Fatal(err)
	}
	// 故意按非时间顺序导入，验证时间线是否按日期升序返回。
	late, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "AB1234", Type: model.EventRecapture,
		LocationID: loc.ID, EventDate: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser",
	})
	if err != nil {
		t.Fatal(err)
	}
	early, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "AB1234", Type: model.EventBanding,
		LocationID: loc.ID, EventDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 存在的个体：时间线可达且按日期升序。
	req := httptest.NewRequest(http.MethodGet, "/api/individuals/"+ind.ID+"/events", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("时间线应返回 200, 实际 %d: %s", res.Code, res.Body.String())
	}
	var timeline []*model.Event
	if err := json.Unmarshal(res.Body.Bytes(), &timeline); err != nil {
		t.Fatalf("解析时间线: %v", err)
	}
	if len(timeline) != 2 {
		t.Fatalf("期望 2 条事件, 实际 %d", len(timeline))
	}
	if timeline[0].ID != early.ID || timeline[1].ID != late.ID {
		t.Fatalf("时间线未按日期升序: 期望 %s,%s 实际 %s,%s",
			early.ID, late.ID, timeline[0].ID, timeline[1].ID)
	}

	// 不存在的个体：仍返回未找到。
	req = httptest.NewRequest(http.MethodGet, "/api/individuals/ind-no-such/events", nil)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("不存在的个体应返回 404, 实际 %d: %s", res.Code, res.Body.String())
	}
}

