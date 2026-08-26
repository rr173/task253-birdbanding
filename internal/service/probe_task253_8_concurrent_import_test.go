package service
import ("sync"; "testing"; "time"; "task253-birdbanding/internal/event"; "task253-birdbanding/internal/model")
func TestConcurrentImportReturnsOneCanonicalEvent(t *testing.T) {
	svc:=New(newTestDB(t)); b,_:=svc.CreateBatch("并发"); l,_:=svc.CreateLocation("点",20,10,10)
	in:=event.ImportInput{BatchID:b.ID,RingCode:"AB1234",Type:model.EventBanding,LocationID:l.ID,EventDate:time.Date(2025,1,1,0,0,0,0,time.UTC),Species:"duck"}
	const n=20; ids:=make(chan string,n); errs:=make(chan error,n); var wg sync.WaitGroup
	for i:=0;i<n;i++ { wg.Add(1); go func(){ defer wg.Done(); e,_,err:=svc.ImportEvent(in); if err!=nil {errs<-err;return}; ids<-e.ID }() }; wg.Wait(); close(ids); close(errs)
	seen:=map[string]bool{}; for id:=range ids {seen[id]=true}; for err:=range errs {t.Fatal(err)}
	if len(seen)!=1 { t.Fatalf("并发幂等应返回同一事件，实际返回 %d 个 ID",len(seen)) }
}
