package store

import (
	"database/sql"
	"time"

	"task253-birdbanding/internal/model"
)

// SaveIndividual 插入个体（环号唯一）。
func (db *DB) SaveIndividual(ind *model.Individual) error {
	_, err := db.Exec(
		`INSERT INTO individuals(id,ring_code,species,created_at) VALUES(?,?,?,?)
		 ON CONFLICT(ring_code) DO UPDATE SET species=excluded.species`,
		ind.ID, ind.RingCode, ind.Species, ind.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetIndividual 按 ID 读取个体。
func (db *DB) GetIndividual(id string) (*model.Individual, error) {
	row := db.QueryRow(`SELECT id,ring_code,species,created_at FROM individuals WHERE id=?`, id)
	return scanIndividual(row)
}

// GetIndividualByRing 按环号读取个体（用于身份关联）。
func (db *DB) GetIndividualByRing(ringCode string) (*model.Individual, error) {
	row := db.QueryRow(`SELECT id,ring_code,species,created_at FROM individuals WHERE ring_code=?`, ringCode)
	return scanIndividual(row)
}

// ListIndividuals 列出全部个体。
func (db *DB) ListIndividuals() ([]*model.Individual, error) {
	rows, err := db.Query(`SELECT id,ring_code,species,created_at FROM individuals ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Individual
	for rows.Next() {
		ind, err := scanIndividual(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ind)
	}
	return out, rows.Err()
}

func scanIndividual(s scanner) (*model.Individual, error) {
	var id, ring, species, created string
	if err := s.Scan(&id, &ring, &species, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	ca, _ := time.Parse(time.RFC3339, created)
	return &model.Individual{ID: id, RingCode: ring, Species: species, CreatedAt: ca}, nil
}
