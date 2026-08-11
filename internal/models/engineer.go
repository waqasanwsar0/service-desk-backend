package models

import "time"

type Engineer struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Email  string   `json:"email"`
	Phone  string   `json:"phone,omitempty"`
	Skills []string `json:"skills"`
	// Location is kept as a plain string (city/region) for now — swap for
	// lat/lng once the GPS/mobile-app future-scope item is built.
	Location string `json:"location"`

	HourlyRate float64 `json:"hourly_rate"`
	DayRate    float64 `json:"day_rate"`
	Currency   string  `json:"currency"` // e.g. "EUR"

	// Named vs Open project eligibility, per the SOW:
	// "Some clients have Named Projects — only pre-approved engineers
	// can be sent. Some are Open Projects — any engineer can be assigned."
	ApprovedProjects []string `json:"approved_projects,omitempty"`

	Available bool `json:"available"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NewEngineerInput struct {
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	Phone            string   `json:"phone,omitempty"`
	Skills           []string `json:"skills"`
	Location         string   `json:"location"`
	HourlyRate       float64  `json:"hourly_rate"`
	DayRate          float64  `json:"day_rate"`
	Currency         string   `json:"currency"`
	ApprovedProjects []string `json:"approved_projects,omitempty"`
}
