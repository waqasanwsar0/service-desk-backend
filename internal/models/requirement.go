package models

import "time"

// RequirementSource records where a client requirement came from — the
// SOW asks for both a portal upload and email intake.
type RequirementSource string

const (
	RequirementSourcePortal RequirementSource = "portal"
	RequirementSourceEmail  RequirementSource = "email"
)

// Requirement is a client's stated need for a project — a document or
// description the client uploads (or emails in), which Service Delivery
// then turns into tickets/dispatches.
type Requirement struct {
	ID          string            `json:"id"`
	ProjectID   string            `json:"project_id,omitempty"`
	ClientName  string            `json:"client_name"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	FileURL     string            `json:"file_url,omitempty"`
	Source      RequirementSource `json:"source"`

	CreatedAt time.Time `json:"created_at"`
}

type NewRequirementInput struct {
	ProjectID   string `json:"project_id,omitempty"`
	ClientName  string `json:"client_name"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	FileURL     string `json:"file_url,omitempty"`
}
