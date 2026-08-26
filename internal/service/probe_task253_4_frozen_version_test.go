package service

import (
	"testing"
	"time"

	"task253-birdbanding/internal/event"
	"task253-birdbanding/internal/model"
)

func TestFrozenVersionRejectsMembershipMutation(t *testing.T) {
	svc := New(newTestDB(t))
	b, _ := svc.CreateBatch("冻结")
	a, _ := svc.CreateLocation("起点", 30, 10, 10)
	z, _ := svc.CreateLocation("终点", 30.1, 10, 10)
	first, _, _ := svc.ImportEvent(event.ImportInput{BatchID:b.ID, RingCode:"AB1234", Type:model.EventBanding, LocationID:a.ID, EventDate:time.Date(2025,3,1,0,0,0,0,time.UTC), Species:"duck"})
	last, _, _ := svc.ImportEvent(event.ImportInput{BatchID:b.ID, RingCode:"AB1234", Type:model.EventRecapture, LocationID:z.ID, EventDate:time.Date(2025,4,1,0,0,0,0,time.UTC), Species:"duck"})
	if err:=svc.ValidateEvent(first.ID,time.Time{}); err!=nil { t.Fatal(err) }
	if err:=svc.ValidateEvent(last.ID,first.EventDate); err!=nil { t.Fatal(err) }
	ind, _ := svc.DB().GetIndividual(first.IndividualID)
	edges, err := svc.BuildEdges(ind.ID); if err!=nil || len(edges)!=1 { t.Fatalf("edges=%d err=%v",len(edges),err) }
	ver, err := svc.CreateVersion(ind.ID,"v1"); if err!=nil { t.Fatal(err) }
	if err:=svc.AddEdgeToVersion(ver.ID,edges[0].ID); err!=nil { t.Fatal(err) }
	if err:=svc.TransitionVersion(ver.ID,model.VersionShared); err!=nil { t.Fatal(err) }
	if err:=svc.TransitionVersion(ver.ID,model.VersionFrozen); err!=nil { t.Fatal(err) }
	if err:=svc.AddEdgeToVersion(ver.ID,edges[0].ID); err != model.ErrFrozenImmutable { t.Fatalf("冻结版本追加应拒绝，实际 %v",err) }
}
