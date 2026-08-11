package models

import "time"

// JobType mirrors the SOW's billing categories for dispatch work.
type JobType string

const (
	JobHourly  JobType = "Hourly"
	JobHalfDay JobType = "HalfDay"
	JobFullDay JobType = "FullDay"
)

func IsValidJobType(j JobType) bool {
	switch j {
	case JobHourly, JobHalfDay, JobFullDay:
		return true
	}
	return false
}

type TimesheetStatus string

const (
	TimesheetPendingReview TimesheetStatus = "Pending Review"
	TimesheetApproved      TimesheetStatus = "Approved"
	TimesheetRejected      TimesheetStatus = "Rejected"
)

// Timesheet is the SOW's "Service desk will upload PDF/JPG of timesheet"
// feature, plus the automated billing calculation that follows it.
type Timesheet struct {
	ID         string `json:"id"`
	TicketID   string `json:"ticket_id"`
	EngineerID string `json:"engineer_id"`

	CheckInAt  time.Time `json:"check_in_at"`
	CheckOutAt time.Time `json:"check_out_at"`
	JobType    JobType   `json:"job_type"`

	// FileURL points at the uploaded PDF/JPG proof of work. Actual file
	// storage (S3 / local disk) is out of scope for this module — this
	// just records the reference.
	FileURL string `json:"file_url,omitempty"`

	Status TimesheetStatus `json:"status"`

	// Populated once approved.
	BilledAmount float64 `json:"billed_amount"`
	Currency     string  `json:"currency"`

	// Set once this timesheet has been pulled into an invoice — prevents
	// billing the same work twice.
	InvoiceID string `json:"invoice_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NewTimesheetInput struct {
	TicketID   string    `json:"ticket_id"`
	EngineerID string    `json:"engineer_id"`
	CheckInAt  time.Time `json:"check_in_at"`
	CheckOutAt time.Time `json:"check_out_at"`
	JobType    JobType   `json:"job_type"`
	FileURL    string    `json:"file_url,omitempty"`
}
