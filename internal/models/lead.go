package models

import "time"

// LeadStatus is a CRM-style pipeline for business-development leads —
// distinct from the recruitment OutreachContact (which tracks candidate
// outreach); this tracks potential client/business leads.
type LeadStatus string

const (
	LeadNew              LeadStatus = "New"
	LeadContacted        LeadStatus = "Contacted"
	LeadFollowUp         LeadStatus = "Follow-up"
	LeadQualified        LeadStatus = "Qualified"
	LeadMeetingScheduled LeadStatus = "Meeting Scheduled"
	LeadWon              LeadStatus = "Won"
	LeadLost             LeadStatus = "Lost"
)

func IsValidLeadStatus(s LeadStatus) bool {
	switch s {
	case LeadNew, LeadContacted, LeadFollowUp, LeadQualified, LeadMeetingScheduled, LeadWon, LeadLost:
		return true
	}
	return false
}

// Lead is a prospective client/business contact — sourced from LinkedIn
// or anywhere else — tracked through a sales pipeline separate from
// recruitment.
type Lead struct {
	ID           string     `json:"id"`
	CompanyName  string     `json:"company_name"`
	ContactName  string     `json:"contact_name,omitempty"`
	ContactEmail string     `json:"contact_email,omitempty"`
	ContactPhone string     `json:"contact_phone,omitempty"`
	LinkedInURL  string     `json:"linkedin_url,omitempty"`
	Source       string     `json:"source,omitempty"` // e.g. "LinkedIn", "Referral", "Website"
	Status       LeadStatus `json:"status"`
	OwnerID      string     `json:"owner_id,omitempty"`

	Notes []LeadNote `json:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LeadNote struct {
	AuthorID  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type NewLeadInput struct {
	CompanyName  string `json:"company_name"`
	ContactName  string `json:"contact_name,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	LinkedInURL  string `json:"linkedin_url,omitempty"`
	Source       string `json:"source,omitempty"`
	OwnerID      string `json:"owner_id,omitempty"`
}
