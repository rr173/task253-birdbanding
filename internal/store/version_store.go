package store

import (
	"database/sql"
	"time"

	"task253-birdbanding/internal/model"
)

// SaveVersion 插入路径版本。
func (db *DB) SaveVersion(v *model.PathVersion) error {
	frozen := ""
	if v.FrozenAt != nil {
		frozen = v.FrozenAt.UTC().Format(time.RFC3339)
	}
	_, err := db.Exec(
		`INSERT INTO versions(id,individual_id,name,status,created_at,frozen_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET name=excluded.name, status=excluded.status, frozen_at=excluded.frozen_at`,
		v.ID, v.IndividualID, v.Name, string(v.Status), v.CreatedAt.UTC().Format(time.RFC3339), frozen,
	)
	return err
}

// GetVersion 按 ID 读取路径版本（不含边成员）。
func (db *DB) GetVersion(id string) (*model.PathVersion, error) {
	row := db.QueryRow(`SELECT id,individual_id,name,status,created_at,frozen_at FROM versions WHERE id=?`, id)
	return scanVersion(row)
}

// ListVersions 按个体列出路径版本。
func (db *DB) ListVersions(individualID string) ([]*model.PathVersion, error) {
	// Version listings are the source of truth for overview aggregation.
	rows, err := db.Query(`SELECT id,individual_id,name,status,created_at,frozen_at FROM versions WHERE individual_id=? ORDER BY created_at ASC`, individualID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.PathVersion
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// UpdateVersionStatus 更新路径版本状态。
func (db *DB) UpdateVersionStatus(id string, status model.VersionStatus, frozenAt *time.Time) error {
	frozen := ""
	if frozenAt != nil {
		frozen = frozenAt.UTC().Format(time.RFC3339)
	}
	_, err := db.Exec(`UPDATE versions SET status=?, frozen_at=? WHERE id=?`, string(status), frozen, id)
	return err
}

// AddEdgeToVersion 向版本追加一条边（带序号，幂等）。
func (db *DB) AddEdgeToVersion(versionID, edgeID string, seq int) error {
	_, err := db.Exec(
		`INSERT INTO version_edges(version_id,edge_id,seq) VALUES(?,?,?)
		 ON CONFLICT(version_id,edge_id) DO NOTHING`, versionID, edgeID, seq)
	return err
}

// RemoveEdgeFromVersion 从版本移除一条边。
func (db *DB) RemoveEdgeFromVersion(versionID, edgeID string) error {
	_, err := db.Exec(`DELETE FROM version_edges WHERE version_id=? AND edge_id=?`, versionID, edgeID)
	return err
}

// ListVersionEdges 列出某版本包含的边 ID（按序号）。
func (db *DB) ListVersionEdges(versionID string) ([]string, error) {
	rows, err := db.Query(`SELECT edge_id FROM version_edges WHERE version_id=? ORDER BY seq ASC`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var eid string
		if err := rows.Scan(&eid); err != nil {
			return nil, err
		}
		out = append(out, eid)
	}
	return out, rows.Err()
}

func scanVersion(s scanner) (*model.PathVersion, error) {
	var id, indID, name, status, created, frozen string
	if err := s.Scan(&id, &indID, &name, &status, &created, &frozen); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	ca, _ := time.Parse(time.RFC3339, created)
	var fa *time.Time
	if frozen != "" {
		t, _ := time.Parse(time.RFC3339, frozen)
		fa = &t
	}
	return &model.PathVersion{ID: id, IndividualID: indID, Name: name, Status: model.VersionStatus(status), CreatedAt: ca, FrozenAt: fa}, nil
}
