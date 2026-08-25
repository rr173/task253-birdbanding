package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"task253-birdbanding/internal/event"
	"task253-birdbanding/internal/httpapi"
	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/service"
	"task253-birdbanding/internal/store"
)

func main() {
	var (
		dbPath    string
		addr      string
		smokeTest bool
	)
	flag.StringVar(&dbPath, "db", "birdbanding.db", "SQLite 数据库路径")
	flag.StringVar(&addr, "addr", ":8080", "HTTP 监听地址")
	flag.BoolVar(&smokeTest, "smoke-test", false, "运行内置冒烟测试后退出")
	flag.Parse()

	if smokeTest {
		if err := runSmokeTest(dbPath); err != nil {
			fmt.Fprintln(os.Stderr, "SMOKE TEST FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("SMOKE TEST OK")
		os.Exit(0)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	svc := service.New(db)
	srv := httpapi.NewServer(svc)
	fmt.Printf("birdbanding listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

// runSmokeTest 端到端验证业务闭环：导入→校验→建边→裁决→版本→自检。
func runSmokeTest(dbPath string) error {
	if dbPath == "" || dbPath == "birdbanding.db" {
		f, err := os.CreateTemp("", "birdbanding-smoke-*.db")
		if err != nil {
			return err
		}
		dbPath = f.Name()
		_ = f.Close()
		defer os.Remove(dbPath)
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	svc := service.New(db)

	// 1) 观测批次
	batch, err := svc.CreateBatch("野外调查#1")
	if err != nil {
		return fmt.Errorf("创建批次: %w", err)
	}
	// 2) 观测地点
	locA, err := svc.CreateLocation("繁殖地", 60.0, 10.0, 500)
	if err != nil {
		return fmt.Errorf("创建地点A: %w", err)
	}
	locB, err := svc.CreateLocation("越冬地", 30.0, 10.0, 500)
	if err != nil {
		return fmt.Errorf("创建地点B: %w", err)
	}
	// 3) 个体环号关联
	ind, _, err := svc.ResolveIndividual("AB1234", "Anser anser")
	if err != nil {
		return fmt.Errorf("关联个体: %w", err)
	}
	// 4) 导入环志 + 重捕事件
	banding, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "AB1234", Type: model.EventBanding,
		LocationID: locA.ID, EventDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser",
	})
	if err != nil {
		return fmt.Errorf("导入环志: %w", err)
	}
	recap, _, err := svc.ImportEvent(event.ImportInput{
		BatchID: batch.ID, RingCode: "AB1234", Type: model.EventRecapture,
		LocationID: locB.ID, EventDate: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), Species: "Anser anser",
	})
	if err != nil {
		return fmt.Errorf("导入重捕: %w", err)
	}
	// 5) 校验（重捕需环志日期）
	if err := svc.ValidateEvent(banding.ID, time.Time{}); err != nil {
		return fmt.Errorf("校验环志: %w", err)
	}
	if err := svc.ValidateEvent(recap.ID, banding.EventDate); err != nil {
		return fmt.Errorf("校验重捕: %w", err)
	}
	// 6) 构建迁徙边
	edges, err := svc.BuildEdges(ind.ID)
	if err != nil {
		return fmt.Errorf("构建迁徙边: %w", err)
	}
	if len(edges) == 0 {
		return fmt.Errorf("期望至少构建 1 条迁徙边，实际 0")
	}
	// 7) 异常裁决：确认边进入路径
	if err := svc.ConfirmEdge(edges[0].ID); err != nil {
		return fmt.Errorf("确认边: %w", err)
	}
	// 8) 路径版本生命周期
	ver, err := svc.CreateVersion(ind.ID, "秋季迁徙路径v1")
	if err != nil {
		return fmt.Errorf("创建版本: %w", err)
	}
	if err := svc.AddEdgeToVersion(ver.ID, edges[0].ID); err != nil {
		return fmt.Errorf("追加边: %w", err)
	}
	if err := svc.TransitionVersion(ver.ID, model.VersionShared); err != nil {
		return fmt.Errorf("共享版本: %w", err)
	}
	if err := svc.TransitionVersion(ver.ID, model.VersionFrozen); err != nil {
		return fmt.Errorf("冻结版本: %w", err)
	}
	// 9) 校验数据库不变量
	problems, err := svc.SelfCheck()
	if err != nil {
		return fmt.Errorf("自检: %w", err)
	}
	if len(problems) > 0 {
		return fmt.Errorf("self-check 发现问题: %v", problems)
	}
	return nil
}
