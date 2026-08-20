package models

import "time"

// TicketStatus represents where a ticket is in its lifecycle.
// This mirrors the workflow defined in the SOW:
// New -> Assigned -> Onsite -> Timesheet Pending -> Completed -> Invoice -> Paid
type TicketStatus string

const (
	StatusNew              TicketStatus = "New"
	StatusAssigned         TicketStatus = "Assigned"
	StatusOnsite           TicketStatus = "Onsite"
	StatusTimesheetPending TicketStatus = "Timesheet Pending"
	StatusCompleted        TicketStatus = "Completed"
	StatusInvoice          TicketStatus = "Invoice"
	StatusPaid             TicketStatus = "Paid"
)

// statusOrder defines the only forward transitions allowed.
// Kept as a slice (not a map) so we can validate "is this a legal next step".
var statusOrder = []TicketStatus{
	StatusNew,
	StatusAssigned,
	StatusOnsite,
	StatusTimesheetPending,
	StatusCompleted,
	StatusInvoice,
	StatusPaid,
}

// IsValidTransition returns true if moving from `from` to `to` is allowed.
// Rule: you may only move one step forward, or stay the same.
// (Cancellation/reopen flows can be added later as explicit exceptions.)
func IsValidTransition(from, to TicketStatus) bool {
	fromIdx, toIdx := -1, -1
	for i, s := range statusOrder {
		if s == from {
			fromIdx = i
		}
		if s == to {
			toIdx = i
		}
	}
	if fromIdx == -1 || toIdx == -1 {
		return false
	}
	return toIdx == fromIdx+1
}

func IsValidStatus(s TicketStatus) bool {
	for _, v := range statusOrder {
		if v == s {
			return true
		}
	}
	return false
}

// Priority mirrors the "Priority / SLA selection" requirement.
type Priority string

const (
	PriorityLow      Priority = "Low"
	PriorityMedium   Priority = "Medium"
	PriorityHigh     Priority = "High"
	PriorityCritical Priority = "Critical"
)

// Source records where the ticket came from — required because the SOW
// calls for Outlook, web form, WhatsApp, and manual entry as intake channels.
type Source string

const (
	SourceManual   Source = "manual"
	SourceWeb      Source = "web"
	SourceEmail    Source = "email"    // Outlook / Microsoft Graph
	SourceWhatsApp Source = "whatsapp" // future scope
)

// ProjectType distinguishes the two project categories from the SOW.
type ProjectType string

const (
	ProjectTypeFTE      ProjectType = "FTE"      // long-term deployment
	ProjectTypeDispatch ProjectType = "Dispatch" // hourly / day-based
)

type Ticket struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	ClientName  string       `json:"client_name"`
	ProjectID   string       `json:"project_id,omitempty"` // links to the Projects module
	ProjectName string       `json:"project_name"`
	ProjectType ProjectType  `json:"project_type"`
	Country     string       `json:"country,omitempty"`
	Domain      string       `json:"domain"` // e.g. "Windows Support", "Networking"
	SiteAddress string       `json:"site_address"`
	Priority    Priority     `json:"priority"`
	SLADueAt    *time.Time   `json:"sla_due_at,omitempty"`
	Source      Source       `json:"source"`
	Status      TicketStatus `json:"status"`
	ImageURLs   []string     `json:"image_urls,omitempty"`

	AssignedEngineerID   string `json:"assigned_engineer_id,omitempty"`
	AssignedEngineerName string `json:"assigned_engineer_name,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewTicketInput is what callers (API clients, email importer, web form) submit.
// Kept separate from Ticket so we control which fields the system sets itself
// (id, status, timestamps) versus what the caller provides.
type NewTicketInput struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	ClientName  string      `json:"client_name"`
	ProjectID   string      `json:"project_id,omitempty"`
	ProjectName string      `json:"project_name"`
	ProjectType ProjectType `json:"project_type"`
	Country     string      `json:"country,omitempty"`
	Domain      string      `json:"domain"`
	SiteAddress string      `json:"site_address"`
	Priority    Priority    `json:"priority"`
	SLAHours    *int        `json:"sla_hours,omitempty"` // e.g. 4 -> due 4h from now
	Source      Source      `json:"source"`
	ImageURLs   []string    `json:"image_urls,omitempty"`
}
