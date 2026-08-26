package path

import (
	"testing"
	"time"

	"task253-birdbanding/internal/model"
)

func TestSpringSameLatitudeEdgeIsRare(t *testing.T) {
	base := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	got := Evaluate(30, 10, base, 30, 11, base.AddDate(0, 0, 30))
	if got.Status != model.EdgeRare {
		t.Fatalf("春季同纬度路线应标为罕见，实际 %+v", got)
	}
}
