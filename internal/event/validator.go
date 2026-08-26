package event

import (
	"regexp"
	"time"

	"task253-birdbanding/internal/model"
)

// ringPattern 环号格式：2-3 位大写字母 + 4-9 位数字。
var ringPattern = regexp.MustCompile(`^[A-Z]{2,3}[0-9]{4,9}$`)

// ValidateRingFormat 校验环号是否符合格式。
func ValidateRingFormat(ring string) error {
	if !ringPattern.MatchString(ring) {
		return model.ErrInvalidRingFormat
	}
	return nil
}

// ImportInput 单次事件导入请求。
type ImportInput struct {
	BatchID    string
	RingCode   string
	Type       model.EventType
	LocationID string
	EventDate  time.Time
	Species    string // 首次出现时用于创建个体
}

// Validate 校验导入请求的基础合法性（环号格式、类型、日期）。
func (in ImportInput) Validate() error {
	if in.BatchID == "" || in.LocationID == "" {
		return model.ErrInvalidArgument
	}
	if in.Type != model.EventBanding && in.Type != model.EventRecapture {
		return model.ErrInvalidArgument
	}
	if in.EventDate.IsZero() {
		return model.ErrInvalidArgument
	}
	return ValidateRingFormat(in.RingCode)
}

// isRecaptureReversed 判断重捕是否早于该个体的环志时间（时间倒退）。
func isRecaptureReversed(recaptureDate time.Time, bandingDate time.Time) bool {
	if bandingDate.IsZero() {
		return false
	}
	return recaptureDate.Before(bandingDate)
}
