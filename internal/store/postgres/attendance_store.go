package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type attendanceStore struct {
	db *sql.DB
}

func NewAttendanceStore(conn *sql.DB) store.AttendanceStore {
	return &attendanceStore{db: conn}
}

func (s *attendanceStore) CheckIn(engineerID, location string) (*models.AttendanceRecord, error) {
	now := time.Now().UTC()
	date := now.Format("2006-01-02")

	var exists bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM attendance_records WHERE engineer_id=$1 AND date=$2)`, engineerID, date).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		return nil, store.ErrAlreadyCheckedIn
	}

	id, err := db.NextID(s.db, "seq_attendance", "ATT", 4)
	if err != nil {
		return nil, err
	}
	rec := &models.AttendanceRecord{
		ID: id, EngineerID: engineerID, Date: date,
		Status: models.AttendancePresent, CheckInAt: now, Location: location,
	}
	_, err = s.db.Exec(`INSERT INTO attendance_records (id, engineer_id, date, status, check_in_at, location) VALUES ($1,$2,$3,$4,$5,$6)`,
		rec.ID, rec.EngineerID, rec.Date, rec.Status, rec.CheckInAt, rec.Location)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *attendanceStore) CheckOut(engineerID string) (*models.AttendanceRecord, error) {
	now := time.Now().UTC()
	date := now.Format("2006-01-02")

	var rec models.AttendanceRecord
	var checkOutAt sql.NullTime
	row := s.db.QueryRow(`SELECT id, engineer_id, date, status, check_in_at, check_out_at, location, notes
		FROM attendance_records WHERE engineer_id=$1 AND date=$2`, engineerID, date)
	if err := row.Scan(&rec.ID, &rec.EngineerID, &rec.Date, &rec.Status, &rec.CheckInAt, &checkOutAt, &rec.Location, &rec.Notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotCheckedInYet
		}
		return nil, err
	}
	if checkOutAt.Valid {
		return nil, store.ErrAlreadyCheckedOut
	}

	_, err := s.db.Exec(`UPDATE attendance_records SET check_out_at=$1 WHERE id=$2`, now, rec.ID)
	if err != nil {
		return nil, err
	}
	rec.CheckOutAt = &now
	return &rec, nil
}

func (s *attendanceStore) RequestLeave(engineerID, fromDate, toDate, reason string) (*models.LeaveRequest, error) {
	id, err := db.NextID(s.db, "seq_leave", "LVE", 4)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	lr := &models.LeaveRequest{
		ID: id, EngineerID: engineerID, FromDate: fromDate, ToDate: toDate, Reason: reason,
		Status: models.LeavePending, CreatedAt: now, UpdatedAt: now,
	}
	_, err = s.db.Exec(`INSERT INTO leave_requests (id, engineer_id, from_date, to_date, reason, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		lr.ID, lr.EngineerID, lr.FromDate, lr.ToDate, lr.Reason, lr.Status, lr.CreatedAt, lr.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return lr, nil
}

func (s *attendanceStore) DecideLeave(leaveID, decidedBy string, approve bool) (*models.LeaveRequest, error) {
	var lr models.LeaveRequest
	row := s.db.QueryRow(`SELECT id, engineer_id, from_date, to_date, reason, status, decided_by, created_at, updated_at FROM leave_requests WHERE id=$1`, leaveID)
	if err := row.Scan(&lr.ID, &lr.EngineerID, &lr.FromDate, &lr.ToDate, &lr.Reason, &lr.Status, &lr.DecidedBy, &lr.CreatedAt, &lr.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	if lr.Status != models.LeavePending {
		return nil, store.ErrLeaveNotPending
	}
	newStatus := models.LeaveRejected
	if approve {
		newStatus = models.LeaveApproved
	}
	now := time.Now().UTC()
	_, err := s.db.Exec(`UPDATE leave_requests SET status=$1, decided_by=$2, updated_at=$3 WHERE id=$4`, newStatus, decidedBy, now, leaveID)
	if err != nil {
		return nil, err
	}
	lr.Status = newStatus
	lr.DecidedBy = decidedBy
	lr.UpdatedAt = now
	return &lr, nil
}

func (s *attendanceStore) List(engineerID string) []*models.AttendanceRecord {
	query := `SELECT id, engineer_id, date, status, check_in_at, check_out_at, location, notes FROM attendance_records WHERE 1=1`
	args := []interface{}{}
	if engineerID != "" {
		args = append(args, engineerID)
		query += " AND engineer_id = $1"
	}
	query += " ORDER BY date DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.AttendanceRecord, 0)
	for rows.Next() {
		var r models.AttendanceRecord
		var checkOutAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.EngineerID, &r.Date, &r.Status, &r.CheckInAt, &checkOutAt, &r.Location, &r.Notes); err == nil {
			if checkOutAt.Valid {
				r.CheckOutAt = &checkOutAt.Time
			}
			out = append(out, &r)
		}
	}
	return out
}

func (s *attendanceStore) ListLeaveRequests(engineerID string) []*models.LeaveRequest {
	query := `SELECT id, engineer_id, from_date, to_date, reason, status, decided_by, created_at, updated_at FROM leave_requests WHERE 1=1`
	args := []interface{}{}
	if engineerID != "" {
		args = append(args, engineerID)
		query += " AND engineer_id = $1"
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.LeaveRequest, 0)
	for rows.Next() {
		var l models.LeaveRequest
		if err := rows.Scan(&l.ID, &l.EngineerID, &l.FromDate, &l.ToDate, &l.Reason, &l.Status, &l.DecidedBy, &l.CreatedAt, &l.UpdatedAt); err == nil {
			out = append(out, &l)
		}
	}
	return out
}
