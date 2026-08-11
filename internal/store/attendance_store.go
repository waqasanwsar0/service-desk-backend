package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

var ErrAlreadyCheckedIn = errors.New("engineer already checked in today")
var ErrNotCheckedInYet = errors.New("engineer has not checked in today")
var ErrAlreadyCheckedOut = errors.New("engineer already checked out today")
var ErrLeaveNotPending = errors.New("leave request already decided")

type AttendanceStore interface {
	CheckIn(engineerID, location string) (*models.AttendanceRecord, error)
	CheckOut(engineerID string) (*models.AttendanceRecord, error)
	RequestLeave(engineerID, fromDate, toDate, reason string) (*models.LeaveRequest, error)
	DecideLeave(leaveID, decidedBy string, approve bool) (*models.LeaveRequest, error)
	List(engineerID string) []*models.AttendanceRecord
	ListLeaveRequests(engineerID string) []*models.LeaveRequest
}

type memoryAttendanceStore struct {
	mu      sync.RWMutex
	records map[string]*models.AttendanceRecord // key: engineerID+date
	leaves  map[string]*models.LeaveRequest
	seq     int
}

func NewMemoryAttendanceStore() AttendanceStore {
	return &memoryAttendanceStore{
		records: make(map[string]*models.AttendanceRecord),
		leaves:  make(map[string]*models.LeaveRequest),
	}
}

func (s *memoryAttendanceStore) CheckIn(engineerID, location string) (*models.AttendanceRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	key := engineerID + "|" + date

	if _, exists := s.records[key]; exists {
		return nil, ErrAlreadyCheckedIn
	}

	s.seq++
	rec := &models.AttendanceRecord{
		ID:         "ATT-" + padLeft(s.seq, 4),
		EngineerID: engineerID,
		Date:       date,
		Status:     models.AttendancePresent,
		CheckInAt:  now,
		Location:   location,
	}
	s.records[key] = rec
	return rec, nil
}

func (s *memoryAttendanceStore) CheckOut(engineerID string) (*models.AttendanceRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	key := engineerID + "|" + date

	rec, exists := s.records[key]
	if !exists {
		return nil, ErrNotCheckedInYet
	}
	if rec.CheckOutAt != nil {
		return nil, ErrAlreadyCheckedOut
	}
	rec.CheckOutAt = &now
	return rec, nil
}

func (s *memoryAttendanceStore) RequestLeave(engineerID, fromDate, toDate, reason string) (*models.LeaveRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now().UTC()
	lr := &models.LeaveRequest{
		ID:         "LVE-" + padLeft(s.seq, 4),
		EngineerID: engineerID,
		FromDate:   fromDate,
		ToDate:     toDate,
		Reason:     reason,
		Status:     models.LeavePending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.leaves[lr.ID] = lr
	return lr, nil
}

func (s *memoryAttendanceStore) DecideLeave(leaveID, decidedBy string, approve bool) (*models.LeaveRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lr, ok := s.leaves[leaveID]
	if !ok {
		return nil, ErrNotFound
	}
	if lr.Status != models.LeavePending {
		return nil, ErrLeaveNotPending
	}
	if approve {
		lr.Status = models.LeaveApproved
	} else {
		lr.Status = models.LeaveRejected
	}
	lr.DecidedBy = decidedBy
	lr.UpdatedAt = time.Now().UTC()
	return lr, nil
}

func (s *memoryAttendanceStore) List(engineerID string) []*models.AttendanceRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.AttendanceRecord, 0)
	for _, rec := range s.records {
		if engineerID != "" && rec.EngineerID != engineerID {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out
}

func (s *memoryAttendanceStore) ListLeaveRequests(engineerID string) []*models.LeaveRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.LeaveRequest, 0)
	for _, lr := range s.leaves {
		if engineerID != "" && lr.EngineerID != engineerID {
			continue
		}
		out = append(out, lr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
