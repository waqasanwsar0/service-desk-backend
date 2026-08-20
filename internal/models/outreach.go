package models

import "time"

// OutreachStatus tracks a manual LinkedIn (or any channel) outreach
// message through to a reply — since LinkedIn has no public API for
// message status, this is logged by the recruiter after they send a
// message themselves.
type OutreachStatus string

const (
	OutreachSent          OutreachStatus = "Sent"
	OutreachReplied       OutreachStatus = "Replied"
	OutreachInterested    OutreachStatus = "Interested"
	OutreachNotInterested OutreachStatus = "Not Interested"
	OutreachNoResponse    OutreachStatus = "No Response"
)

func IsValidOutreachStatus(s OutreachStatus) bool {
	switch s {
	case OutreachSent, OutreachReplied, OutreachInterested, OutreachNotInterested, OutreachNoResponse:
		return true
	}
	return false
}

// OutreachContact is one person a recruiter has approached — typically
// via LinkedIn — logged manually since LinkedIn doesn't expose message
// status through any API available to a small business.
type OutreachContact struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	LinkedInURL   string         `json:"linkedin_url,omitempty"`
	TargetRole    string         `json:"target_role,omitempty"` // job/project being recruited for
	RecruiterID   string         `json:"recruiter_id,omitempty"`
	Status        OutreachStatus `json:"status"`
	MessageSentAt time.Time      `json:"message_sent_at"`

	Notes []OutreachNote `json:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OutreachNote struct {
	AuthorID  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type NewOutreachInput struct {
	Name          string    `json:"name"`
	LinkedInURL   string    `json:"linkedin_url,omitempty"`
	TargetRole    string    `json:"target_role,omitempty"`
	RecruiterID   string    `json:"recruiter_id,omitempty"`
	MessageSentAt time.Time `json:"message_sent_at"`
}
