package store

import (
	"database/sql"
	"time"

	"task253-birdbanding/internal/model"
)

// SaveEvent 插入事件。fingerprint 唯一约束保证幂等去重。
func (db *DB) SaveEvent(e *model.Event) error {
	_, err := db.Exec(
		`INSERT INTO events(id,batch_id,individual_id,ring_code,type,location_id,event_date,status,fingerprint,error_reason,created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(fingerprint) DO NOTHING`,
		e.ID, e.BatchID, e.IndividualID, e.RingCode, string(e.Type), e.LocationID,
		e.EventDate.UTC().Format(time.RFC3339), string(e.Status), e.Fingerprint, e.ErrorReason,
		e.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	return nil
}

// GetEventByFingerprint 按指纹读取事件（幂等去重查询）。
func (db *DB) GetEventByFingerprint(fp string) (*model.Event, error) {
	row := db.QueryRow(
		`SELECT id,batch_id,individual_id,ring_code,type,location_id,event_date,status,fingerprint,error_reason,created_at
			 FROM events WHERE fingerprint=? ORDER BY created_at ASC LIMIT 1`, fp)
	return scanEvent(row)
}

// EventExistsByFingerprint 判断指纹是否已存在（幂等去重前置检查）。
func (db *DB) EventExistsByFingerprint(fp string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM events WHERE fingerprint=?`, fp).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetEvent 按 ID 读取事件。
func (db *DB) GetEvent(id string) (*model.Event, error) {
	row := db.QueryRow(
		`SELECT id,batch_id,individual_id,ring_code,type,location_id,event_date,status,fingerprint,error_reason,created_at
		 FROM events WHERE id=?`, id)
	return scanEvent(row)
}

// ListEvents 按可选批次/个体/状态过滤列出事件（按日期升序）。
func (db *DB) ListEvents(batchID, individualID, status string) ([]*model.Event, error) {
	q := `SELECT id,batch_id,individual_id,ring_code,type,location_id,event_date,status,fingerprint,error_reason,created_at FROM events WHERE 1=1`
	args := []interface{}{}
	if batchID != "" {
		q += ` AND batch_id=?`
		args = append(args, batchID)
	}
	if individualID != "" {
		q += ` AND individual_id=?`
		args = append(args, individualID)
	}
	if status != "" {
		q += ` AND status=?`
		args = append(args, status)
	}
	q += ` ORDER BY event_date ASC, id ASC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpdateEventStatus 更新事件状态与错误原因（用于校验、校正环号、排除）。
func (db *DB) UpdateEventStatus(id string, status model.EventStatus, reason string, individualID string) error {
	_, err := db.Exec(
		`UPDATE events SET status=?, error_reason=?, individual_id=? WHERE id=?`,
		string(status), reason, individualID, id)
	return err
}

// EventsByIndividualSorted 返回某个体所有事件，按日期升序（建边用）。
func (db *DB) EventsByIndividualSorted(individualID string) ([]*model.Event, error) {
	return db.ListEvents("", individualID, "")
}

// UpdateEventRing 校正环号：更新环号、关联个体、指纹并重置为待校验。
func (db *DB) UpdateEventRing(id, ringCode, individualID, fingerprint string, status model.EventStatus, reason string) error {
	_, err := db.Exec(
		`UPDATE events SET ring_code=?, individual_id=?, fingerprint=?, status=?, error_reason=? WHERE id=?`,
		ringCode, individualID, fingerprint, string(status), reason, id)
	return err
}

func scanEvent(s scanner) (*model.Event, error) {
	var id, batchID, indID, ring, typ, locID, date, status, fp, reason, created string
	if err := s.Scan(&id, &batchID, &indID, &ring, &typ, &locID, &date, &status, &fp, &reason, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	ed, _ := time.Parse(time.RFC3339, date)
	ca, _ := time.Parse(time.RFC3339, created)
	return &model.Event{
		ID: id, BatchID: batchID, IndividualID: indID, RingCode: ring,
		Type: model.EventType(typ), LocationID: locID, EventDate: ed,
		Status: model.EventStatus(status), Fingerprint: fp, ErrorReason: reason, CreatedAt: ca,
	}, nil
}
