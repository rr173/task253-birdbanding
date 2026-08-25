package service
import ("testing"; "task253-birdbanding/internal/model")
func TestPublishedBatchKeepsPublishTime(t *testing.T) {
	svc:=New(newTestDB(t)); b,err:=svc.CreateBatch("发布"); if err!=nil {t.Fatal(err)}
	if err:=svc.TransitionBatch(b.ID,model.BatchReview); err!=nil {t.Fatal(err)}
	if err:=svc.TransitionBatch(b.ID,model.BatchPublished); err!=nil {t.Fatal(err)}
	got,err:=svc.DB().GetBatch(b.ID); if err!=nil {t.Fatal(err)}
	if got.PublishedAt == nil { t.Fatal("已发布批次必须保存发布时间") }
}
