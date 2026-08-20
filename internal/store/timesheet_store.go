package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"servicedesk/internal/models"
)

var ErrTimesheetNotPending = errors.New("timesheet is not pending review")
var ErrTimesheetAlreadyInvoiced = errors.New("timesheet has already been invoiced")
var ErrTimesheetNotApproved = errors.New("timesheet must be approved before invoicing")

type TimesheetStore interface {
	Create(t *models.Timesheet) error
	Get(id string) (*models.Timesheet, error)
	List(ticketID, engineerID string) []*models.Timesheet
	// Approve computes the billed amount from the engineer's rate and
	// marks the timesheet Approved. Rejects with ErrTimesheetNotPending
	// if it has already been decided.
	Approve(id string, hourlyRate, dayRate float64, currency string) (*models.Timesheet, error)
	Reject(id string) (*models.Timesheet, error)
	// MarkInvoiced attaches an invoice ID to an approved, not-yet-invoiced
	// timesheet. Returns ErrTimesheetAlreadyInvoiced or ErrTimesheetNotApproved
	// if the timesheet isn't eligible.
	MarkInvoiced(id, invoiceID string) (*models.Timesheet, error)
	// Sign marks the timesheet as confirmed by the engineer (FTE
	// sign-off) — required before it can be approved for billing.
	Sign(id string) (*models.Timesheet, error)
}

type memoryTimesheetStore struct {
	mu         sync.RWMutex
	timesheets map[string]*models.Timesheet
	seq        int
}

func NewMemoryTimesheetStore() TimesheetStore {
	return &memoryTimesheetStore{timesheets: make(map[string]*models.Timesheet)}
}

func (s *memoryTimesheetStore) Create(t *models.Timesheet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now().UTC()
	t.ID = "TSH-" + padLeft(s.seq, 4)
	t.Status = models.TimesheetPendingReview
	t.CreatedAt = now
	t.UpdatedAt = now

	s.timesheets[t.ID] = t
	return nil
}

func (s *memoryTimesheetStore) Get(id string) (*models.Timesheet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.timesheets[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *memoryTimesheetStore) List(ticketID, engineerID string) []*models.Timesheet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Timesheet, 0)
	for _, t := range s.timesheets {
		if ticketID != "" && t.TicketID != ticketID {
			continue
		}
		if engineerID != "" && t.EngineerID != engineerID {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// Approve calculates billing based on job type:
//   - Hourly:  (checkout - checkin) in hours * hourlyRate
//   - HalfDay: dayRate / 2
//   - FullDay: dayRate
func (s *memoryTimesheetStore) Approve(id string, hourlyRate, dayRate float64, currency string) (*models.Timesheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.timesheets[id]
	if !ok {
		return nil, ErrNotFound
	}
	if t.Status != models.TimesheetPendingReview {
		return nil, ErrTimesheetNotPending
	}

	var amount float64
	switch t.JobType {
	case models.JobHourly:
		hours := t.CheckOutAt.Sub(t.CheckInAt).Hours()
		if hours < 0 {
			hours = 0
		}
		amount = hours * hourlyRate
	case models.JobHalfDay:
		amount = dayRate / 2
	case models.JobFullDay:
		amount = dayRate
	}

	t.BilledAmount = round2(amount)
	t.Currency = currency
	t.Status = models.TimesheetApproved
	t.UpdatedAt = time.Now().UTC()
	return t, nil
}

func (s *memoryTimesheetStore) Reject(id string) (*models.Timesheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.timesheets[id]
	if !ok {
		return nil, ErrNotFound
	}
	if t.Status != models.TimesheetPendingReview {
		return nil, ErrTimesheetNotPending
	}
	t.Status = models.TimesheetRejected
	t.UpdatedAt = time.Now().UTC()
	return t, nil
}

func (s *memoryTimesheetStore) MarkInvoiced(id, invoiceID string) (*models.Timesheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.timesheets[id]
	if !ok {
		return nil, ErrNotFound
	}
	if t.Status != models.TimesheetApproved {
		return nil, ErrTimesheetNotApproved
	}
	if t.InvoiceID != "" {
		return nil, ErrTimesheetAlreadyInvoiced
	}
	t.InvoiceID = invoiceID
	t.UpdatedAt = time.Now().UTC()
	return t, nil
}

func (s *memoryTimesheetStore) Sign(id string) (*models.Timesheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.timesheets[id]
	if !ok {
		return nil, ErrNotFound
	}
	now := time.Now().UTC()
	t.SignedByEngineer = true
	t.SignedAt = &now
	t.UpdatedAt = now
	return t, nil
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
