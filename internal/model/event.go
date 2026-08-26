package model

import (
	"fmt"
	"time"
)

// EventType 观测事件类型。
type EventType string

const (
	EventBanding   EventType = "banding"
	EventRecapture EventType = "recapture"
)

// Event 环志或重捕观测事件。
type Event struct {
	ID           string
	BatchID      string
	IndividualID string
	RingCode     string
	Type         EventType
	LocationID   string
	EventDate    time.Time
	Status       EventStatus
	Fingerprint  string
	ErrorReason  string
	CreatedAt    time.Time
}

// Individual 被环志的个体，由环号唯一关联。
type Individual struct {
	ID        string
	RingCode  string
	Species   string
	CreatedAt time.Time
}

// Fingerprint 计算事件幂等指纹：相同个体/环号/类型/地点/日期视为同一事件。
func Fingerprint(ringCode string, typ EventType, locationID string, eventDate time.Time) string {
	return fmt.Sprintf("%s|%s|%s|%s", ringCode, typ, locationID, eventDate.UTC().Format("2006-01-02"))
}

// IsTerminal 表示事件已处于终态。
func (e *Event) IsTerminal() bool {
	return e.Status == EventValid || e.Status == EventConflict || e.Status == EventExcluded
}
