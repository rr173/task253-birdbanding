package event

import (
	"testing"
	"time"

	"task253-birdbanding/internal/model"
)

func TestValidateRingFormat(t *testing.T) {
	for _, ring := range []string{"AB1234", "ABC123456789"} {
		if err := ValidateRingFormat(ring); err != nil {
			t.Fatalf("有效环号 %q 被拒绝: %v", ring, err)
		}
	}
	for _, ring := range []string{"A1234", "ab1234", "AB12", "AB-1234"} {
		if err := ValidateRingFormat(ring); err != model.ErrInvalidRingFormat {
			t.Fatalf("非法环号 %q 未被正确拒绝: %v", ring, err)
		}
	}
}

func TestImportInputValidate(t *testing.T) {
	in := ImportInput{
		BatchID: "batch-1", RingCode: "AB1234", Type: model.EventRecapture,
		LocationID: "loc-1", EventDate: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := in.Validate(); err != nil {
		t.Fatalf("有效导入请求被拒绝: %v", err)
	}
	in.Type = "unknown"
	if err := in.Validate(); err != model.ErrInvalidArgument {
		t.Fatalf("未知事件类型应被拒绝: %v", err)
	}
}
