package store

import (
	"database/sql"
	"time"

	"task253-birdbanding/internal/model"
)

// SaveEdge 插入迁徙边。
func (db *DB) SaveEdge(e *model.MigrationEdge) error {
	_, err := db.Exec(
		`INSERT INTO edges(id,individual_id,from_event_id,to_event_id,distance_km,days,speed_km_day,status,reason,created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET status=excluded.status, reason=excluded.reason`,
		e.ID, e.IndividualID, e.FromEventID, e.ToEventID, e.DistanceKm, e.Days, e.SpeedKmDay,
		string(e.Status), e.Reason, e.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetEdge 按 ID 读取迁徙边。
func (db *DB) GetEdge(id string) (*model.MigrationEdge, error) {
	row := db.QueryRow(
		`SELECT id,individual_id,from_event_id,to_event_id,distance_km,days,speed_km_day,status,reason,created_at
		 FROM edges WHERE id=?`, id)
	return scanEdge(row)
}

// ListEdges 按个体列出迁徙边（按创建时间升序）。
func (db *DB) ListEdges(individualID string) ([]*model.MigrationEdge, error) {
	rows, err := db.Query(
		`SELECT id,individual_id,from_event_id,to_event_id,distance_km,days,speed_km_day,status,reason,created_at
		 FROM edges WHERE individual_id=? ORDER BY created_at ASC`, individualID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.MigrationEdge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpdateEdgeStatus 更新迁徙边状态与原因。
func (db *DB) UpdateEdgeStatus(id string, status model.EdgeStatus, reason string) error {
	_, err := db.Exec(`UPDATE edges SET status=?, reason=? WHERE id=?`, string(status), reason, id)
	return err
}

func scanEdge(s scanner) (*model.MigrationEdge, error) {
	var id, indID, fromID, toID, status, reason, created string
	var dist, days, speed float64
	if err := s.Scan(&id, &indID, &fromID, &toID, &dist, &days, &speed, &status, &reason, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	ca, _ := time.Parse(time.RFC3339, created)
	return &model.MigrationEdge{
		ID: id, IndividualID: indID, FromEventID: fromID, ToEventID: toID,
		DistanceKm: dist, Days: days, SpeedKmDay: speed,
		Status: model.EdgeStatus(status), Reason: reason, CreatedAt: ca,
	}, nil
}
