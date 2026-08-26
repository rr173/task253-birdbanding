package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"task253-birdbanding/internal/service"
	"task253-birdbanding/internal/store"
)

func TestIndividualTimelineRouteIsReachable(t *testing.T) {
	f, _ := os.CreateTemp("", "task253-bug5-*.db"); _ = f.Close()
	db, err := store.Open(f.Name()); if err != nil { t.Fatal(err) }
	t.Cleanup(func(){ _=db.Close(); _=os.Remove(f.Name()) })
	svc := service.New(db); ind,_,err:=svc.ResolveIndividual("AB1234","duck"); if err!=nil {t.Fatal(err)}
	res:=httptest.NewRecorder(); req:=httptest.NewRequest(http.MethodGet,"/api/individuals/"+ind.ID+"/events",nil)
	NewServer(svc).Handler().ServeHTTP(res,req)
	if res.Code != http.StatusOK { t.Fatalf("时间线接口应可访问，返回 %d: %s",res.Code,res.Body.String()) }
}
