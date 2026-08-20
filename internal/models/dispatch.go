package models

import "time"

// Dispatch represents sending an engineer out to a client site. Creating
// one auto-generates a linked Ticket (source="dispatch") and, when an
// engineer is specified, assigns them immediately — this is the SOW's
// "tickets automatically generate when a dispatch is created" and
// "client portal automatically shares the ticket with the available
// engineer" requirement. The client-portal side of "sharing" is simply
// that GET /api/tickets/{id} already includes assigned_engineer_name —
// there is no separate share step needed once the ticket is created
// with an engineer attached.
type Dispatch struct {
	ID          string    `json:"id"`
	TicketID    string    `json:"ticket_id"` // the auto-generated ticket
	ProjectID   string    `json:"project_id,omitempty"`
	ClientName  string    `json:"client_name"`
	SiteAddress string    `json:"site_address,omitempty"`
	EngineerID  string    `json:"engineer_id,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type NewDispatchInput struct {
	ProjectID   string    `json:"project_id,omitempty"`
	ClientName  string    `json:"client_name"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	SiteAddress string    `json:"site_address,omitempty"`
	Domain      string    `json:"domain,omitempty"`
	Priority    Priority  `json:"priority,omitempty"`
	EngineerID  string    `json:"engineer_id,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at,omitempty"`
}
