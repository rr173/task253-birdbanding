package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
