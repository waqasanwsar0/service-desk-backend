package models

import "time"

// EngineerAssignment lets one engineer work on multiple projects at once,
// each with its own rate and cost structure — the SOW's "Re-hire
// functionality" and "Multi-Project Assignment Capability":
//
//	"Allow engineers/employees to work on multiple projects
//	 simultaneously... different hourly/day rates... different
//	 additional costs per project (travel and tools reimbursements)."
type EngineerAssignment struct {
	ID          string `json:"id"`
	EngineerID  string `json:"engineer_id"`
	ProjectName string `json:"project_name"`
	ClientName  string `json:"client_name"`

	HourlyRate float64 `json:"hourly_rate,omitempty"`
	DayRate    float64 `json:"day_rate,omitempty"`
	Currency   string  `json:"currency"`

	// Additional per-project reimbursements, e.g. travel or tool costs.
	TravelAllowance float64 `json:"travel_allowance,omitempty"`
	ToolsAllowance  float64 `json:"tools_allowance,omitempty"`

	StartDate string `json:"start_date"` // YYYY-MM-DD
	EndDate   string `json:"end_date,omitempty"`
	Active    bool   `json:"active"`

	CreatedAt time.Time `json:"created_at"`
}

type NewAssignmentInput struct {
	ProjectName     string  `json:"project_name"`
	ClientName      string  `json:"client_name"`
	HourlyRate      float64 `json:"hourly_rate,omitempty"`
	DayRate         float64 `json:"day_rate,omitempty"`
	Currency        string  `json:"currency"`
	TravelAllowance float64 `json:"travel_allowance,omitempty"`
	ToolsAllowance  float64 `json:"tools_allowance,omitempty"`
	StartDate       string  `json:"start_date"`
	EndDate         string  `json:"end_date,omitempty"`
}
