package service
import ("testing"; "time"; "task253-birdbanding/internal/event"; "task253-birdbanding/internal/model")
func TestBulkImportContinuesAfterInvalidRecord(t *testing.T) {
	svc:=New(newTestDB(t)); b,_:=svc.CreateBatch("批量"); l,_:=svc.CreateLocation("点",20,10,10); base:=time.Date(2025,1,1,0,0,0,0,time.UTC)
	items,errs:=svc.BulkImport([]event.ImportInput{{BatchID:b.ID,RingCode:"bad",Type:model.EventBanding,LocationID:l.ID,EventDate:base,Species:"duck"},{BatchID:b.ID,RingCode:"AB1234",Type:model.EventBanding,LocationID:l.ID,EventDate:base.AddDate(0,0,1),Species:"duck"}})
	if len(errs)!=1 || len(items)!=1 { t.Fatalf("批量导入应保留有效记录，items=%d errs=%d",len(items),len(errs)) }
}
