package models

import "time"

// Project is the SOW's "Projects module" — the core entity tickets,
// engineers, and reporting are organized under, tagged by country/city
// to support multi-country operations. Reuses the ProjectType
// (Dispatch/FTE) already defined in ticket.go.
type Project struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	ClientName string      `json:"client_name"`
	Country    string      `json:"country,omitempty"`
	City       string      `json:"city,omitempty"`
	Type       ProjectType `json:"type"`

	CreatedAt time.Time `json:"created_at"`
}

type NewProjectInput struct {
	Name       string      `json:"name"`
	ClientName string      `json:"client_name"`
	Country    string      `json:"country,omitempty"`
	City       string      `json:"city,omitempty"`
	Type       ProjectType `json:"type"`
}
