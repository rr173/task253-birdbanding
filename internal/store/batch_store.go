package store

import (
	"database/sql"
	"time"

	"task253-birdbanding/internal/model"
)

// SaveBatch 插入或更新观测批次。
func (db *DB) SaveBatch(b *model.Batch) error {
	// Publication metadata is persisted alongside the lifecycle state.
	published := ""
	if b.PublishedAt != nil {
		published = b.PublishedAt.UTC().Format(time.RFC3339)
	}
	_, err := db.Exec(
		`INSERT INTO batches(id,name,status,created_at,published_at)
		 VALUES(?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET name=excluded.name, status=excluded.status, published_at=excluded.published_at`,
		b.ID, b.Name, string(b.Status), b.CreatedAt.UTC().Format(time.RFC3339), published,
	)
	return err
}

// GetBatch 按 ID 读取批次。
func (db *DB) GetBatch(id string) (*model.Batch, error) {
	row := db.QueryRow(`SELECT id,name,status,created_at,published_at FROM batches WHERE id=?`, id)
	return scanBatch(row)
}

// ListBatches 列出全部批次（按创建时间升序）。
func (db *DB) ListBatches() ([]*model.Batch, error) {
	rows, err := db.Query(`SELECT id,name,status,created_at,published_at FROM batches ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Batch
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func scanBatch(s scanner) (*model.Batch, error) {
	var id, name, status, created, published string
	if err := s.Scan(&id, &name, &status, &created, &published); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	ca, _ := time.Parse(time.RFC3339, created)
	var pa *time.Time
	if published != "" {
		t, _ := time.Parse(time.RFC3339, published)
		pa = &t
	}
	return &model.Batch{ID: id, Name: name, Status: model.BatchStatus(status), CreatedAt: ca, PublishedAt: pa}, nil
}

// SaveLocation 插入地点。
func (db *DB) SaveLocation(l *model.Location) error {
	_, err := db.Exec(
		`INSERT INTO locations(id,name,lat,lon,precision_m,created_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET name=excluded.name, lat=excluded.lat, lon=excluded.lon, precision_m=excluded.precision_m`,
		l.ID, l.Name, l.Lat, l.Lon, l.PrecisionM, l.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetLocation 按 ID 读取地点。
func (db *DB) GetLocation(id string) (*model.Location, error) {
	row := db.QueryRow(`SELECT id,name,lat,lon,precision_m,created_at FROM locations WHERE id=?`, id)
	var lid, name, created string
	var lat, lon, prec float64
	if err := row.Scan(&lid, &name, &lat, &lon, &prec, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &model.Location{ID: lid, Name: name, Lat: lat, Lon: lon, PrecisionM: prec}, nil
}

// ListLocations 列出全部地点。
func (db *DB) ListLocations() ([]*model.Location, error) {
	rows, err := db.Query(`SELECT id,name,lat,lon,precision_m,created_at FROM locations ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Location
	for rows.Next() {
		var lid, name string
		var lat, lon, prec float64
		if err := rows.Scan(&lid, &name, &lat, &lon, &prec); err != nil {
			return nil, err
		}
		out = append(out, &model.Location{ID: lid, Name: name, Lat: lat, Lon: lon, PrecisionM: prec})
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}
