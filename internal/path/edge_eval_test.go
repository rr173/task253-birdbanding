package path

import (
	"testing"
	"time"

	"task253-birdbanding/internal/model"
)

func TestEvaluateMigrationEdgeBoundaries(t *testing.T) {
	base := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	feasible := Evaluate(30, 10, base, 30.1, 10, base.AddDate(0, 0, 30))
	if feasible.Status != model.EdgeFeasible || feasible.SpeedKmDay <= 0 {
		t.Fatalf("正常迁徙边评估错误: %+v", feasible)
	}
	over := Evaluate(0, 0, base, 40, 0, base.AddDate(0, 0, 1))
	if over.Status != model.EdgeOverSpeed {
		t.Fatalf("超速迁徙边未被识别: %+v", over)
	}
	reversed := Evaluate(30, 10, base, 31, 10, base.Add(-24*time.Hour))
	if reversed.Status != model.EdgeCandidate || reversed.Reason != model.ErrRecaptureTimeReversed.Error() {
		t.Fatalf("时间倒退边界错误: %+v", reversed)
	}
}
