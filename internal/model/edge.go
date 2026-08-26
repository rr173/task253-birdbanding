package model

import "time"

// MigrationEdge 同一体连续两次观测之间的迁徙边。
type MigrationEdge struct {
	ID           string
	IndividualID string
	FromEventID  string
	ToEventID    string
	DistanceKm   float64
	Days         float64
	SpeedKmDay   float64
	Status       EdgeStatus
	Reason       string
	CreatedAt    time.Time
}

// PathVersion 个体迁徙路径假设版本（不可变快照由冻结保证）。
type PathVersion struct {
	ID           string
	IndividualID string
	Name         string
	Status       VersionStatus
	EdgeIDs      []string
	CreatedAt    time.Time
	FrozenAt     *time.Time
}

// EdgeMembership 表示一条迁徙边属于某个路径版本。
type EdgeMembership struct {
	VersionID string
	EdgeID    string
	Seq       int
}
