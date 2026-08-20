package models

import "time"

// EngineerDocument is a reference to an uploaded file on the engineer's
// profile — ID card, certification, contract, etc.
type EngineerDocument struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Engineer struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Email  string   `json:"email"`
	Phone  string   `json:"phone,omitempty"`
	Skills []string `json:"skills"`
	// Location is kept as a plain string (city/region) for now — swap for
	// lat/lng once the GPS/mobile-app future-scope item is built.
	Location string `json:"location"`

	// AreaCoverage lists the cities/regions this engineer can be
	// dispatched to, separate from where they're based.
	AreaCoverage []string `json:"area_coverage,omitempty"`

	HourlyRate  float64 `json:"hourly_rate"`
	HalfDayRate float64 `json:"half_day_rate,omitempty"`
	DayRate     float64 `json:"day_rate"`
	Currency    string  `json:"currency"` // e.g. "EUR"

	// TravelCost is a default per-dispatch travel allowance for this
	// engineer, used as a starting point when billing (can be overridden
	// per project assignment — see EngineerAssignment).
	TravelCost float64 `json:"travel_cost,omitempty"`

	ResumeURL string             `json:"resume_url,omitempty"`
	Documents []EngineerDocument `json:"documents,omitempty"`

	// Named vs Open project eligibility, per the SOW:
	// "Some clients have Named Projects — only pre-approved engineers
	// can be sent. Some are Open Projects — any engineer can be assigned."
	ApprovedProjects []string `json:"approved_projects,omitempty"`

	Available bool `json:"available"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NewEngineerInput struct {
	Name             string             `json:"name"`
	Email            string             `json:"email"`
	Phone            string             `json:"phone,omitempty"`
	Skills           []string           `json:"skills"`
	Location         string             `json:"location"`
	AreaCoverage     []string           `json:"area_coverage,omitempty"`
	HourlyRate       float64            `json:"hourly_rate"`
	HalfDayRate      float64            `json:"half_day_rate,omitempty"`
	DayRate          float64            `json:"day_rate"`
	Currency         string             `json:"currency"`
	TravelCost       float64            `json:"travel_cost,omitempty"`
	ResumeURL        string             `json:"resume_url,omitempty"`
	Documents        []EngineerDocument `json:"documents,omitempty"`
	ApprovedProjects []string           `json:"approved_projects,omitempty"`
}
