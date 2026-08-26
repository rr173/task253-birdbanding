package service

import (
	"sync"
	"testing"
	"time"

	"task253-birdbanding/internal/event"
	"task253-birdbanding/internal/model"
)

func TestConcurrentSameFingerprintConverges(t *testing.T) {
	svc := New(newTestDB(t))
	batch, _ := svc.CreateBatch("批次")
	loc, _ := svc.CreateLocation("地点", 60.0, 10.0, 500)
	in := event.ImportInput{
		BatchID: batch.ID, RingCode: "GH2468", Type: model.EventBanding,
		LocationID: loc.ID, EventDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), Species: "Larus argentatus",
	}
	const n = 64
	var wg sync.WaitGroup
	results := make([]*model.Event, n)
	errs := make([]error, n)
	start := make(chan struct{})
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			ev, _, err := svc.ImportEvent(in)
			errs[i] = err
			results[i] = ev
		}(i)
	}
	close(start)
	wg.Wait()

	// 1) 所有调用方均成功且无错误。
	for i, err := range errs {
		if err != nil {
			t.Fatalf("并发导入 %d 出错: %v", i, err)
		}
		if results[i] == nil {
			t.Fatalf("并发导入 %d 返回 nil 事件", i)
		}
	}
	// 2) 所有调用方拿到同一稳定标识（收敛）。
	want := results[0].ID
	for i, ev := range results {
		if ev.ID != want {
			t.Fatalf("并发导入未收敛: id[0]=%s id[%d]=%s", want, i, ev.ID)
		}
	}
	// 3) 返回的标识必须真实持久化：按该 ID 能读回记录（验证落败方未拿到幻影 ID）。
	got, err := svc.DB().GetEvent(want)
	if err != nil {
		t.Fatalf("并发导入返回的事件标识无法读回: %v (id=%s)", err, want)
	}
	if got.ID != want {
		t.Fatalf("读回事件 ID=%s 与返回 ID=%s 不一致", got.ID, want)
	}
	// 4) 数据库仅一条事件记录、一条个体记录。
	all, _ := svc.DB().ListEvents("", "", "")
	if len(all) != 1 {
		t.Fatalf("期望仅 1 条事件记录, 实际 %d", len(all))
	}
	if all[0].ID != want {
		t.Fatalf("持久化事件 ID=%s 与返回 ID=%s 不一致", all[0].ID, want)
	}
	inds, _ := svc.DB().ListIndividuals()
	if len(inds) != 1 {
		t.Fatalf("期望仅 1 条个体记录, 实际 %d", len(inds))
	}
}

// TestConcurrentDistinctEventsDoNotInterfere 验证不同事件并发导入互不影响：
// 多个不同指纹（同环号、不同地点/日期）并发导入应各自落库，数量与标识一一对应。
func TestConcurrentDistinctEventsDoNotInterfere(t *testing.T) {
	svc := New(newTestDB(t))
	batch, _ := svc.CreateBatch("批次")
	locA, _ := svc.CreateLocation("A", 60.0, 10.0, 500)
	locB, _ := svc.CreateLocation("B", 30.0, 10.0, 500)
	locC, _ := svc.CreateLocation("C", 45.0, 5.0, 500)
	inputs := []event.ImportInput{
		{BatchID: batch.ID, RingCode: "IJ0001", Type: model.EventBanding, LocationID: locA.ID, EventDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser"},
		{BatchID: batch.ID, RingCode: "IJ0001", Type: model.EventRecapture, LocationID: locB.ID, EventDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser"},
		{BatchID: batch.ID, RingCode: "KL2345", Type: model.EventBanding, LocationID: locC.ID, EventDate: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), Species: "Anas platyrhynchos"},
	}
	const repeats = 32
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(len(inputs) * repeats)
	for _, in := range inputs {
		in := in
		for r := 0; r < repeats; r++ {
			go func() {
				defer wg.Done()
				<-start
				_, _, err := svc.ImportEvent(in)
				if err != nil {
					t.Errorf("并发导入不同事件出错: %v", err)
				}
			}()
		}
	}
	close(start)
	wg.Wait()

	all, _ := svc.DB().ListEvents("", "", "")
	if len(all) != len(inputs) {
		t.Fatalf("期望 %d 条不同事件记录, 实际 %d", len(inputs), len(all))
	}
	// 指纹去重：无重复记录。
	seen := map[string]bool{}
	for _, e := range all {
		if seen[e.Fingerprint] {
			t.Fatalf("不同事件出现重复指纹: %s", e.Fingerprint)
		}
		seen[e.Fingerprint] = true
	}
	inds, _ := svc.DB().ListIndividuals()
	if len(inds) != 2 {
		t.Fatalf("期望 2 条个体记录, 实际 %d", len(inds))
	}
}
