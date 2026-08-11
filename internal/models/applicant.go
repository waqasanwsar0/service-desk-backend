package models

import "time"

// ApplicantStage mirrors a typical ATS pipeline. Kept as an ordered list
// like TicketStatus so moves can be validated the same way.
type ApplicantStage string

const (
	StageApplied   ApplicantStage = "Applied"
	StageScreening ApplicantStage = "Screening"
	StageInterview ApplicantStage = "Interview"
	StageOffer     ApplicantStage = "Offer"
	StageHired     ApplicantStage = "Hired"
	StageRejected  ApplicantStage = "Rejected" // terminal, reachable from any stage
)

var applicantStageOrder = []ApplicantStage{
	StageApplied, StageScreening, StageInterview, StageOffer, StageHired,
}

// IsValidStageMove allows moving one step forward, or into Rejected from
// any non-terminal stage (a candidate can be rejected at any point).
func IsValidStageMove(from, to ApplicantStage) bool {
	if to == StageRejected && from != StageHired && from != StageRejected {
		return true
	}
	fromIdx, toIdx := -1, -1
	for i, s := range applicantStageOrder {
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

func IsValidStage(s ApplicantStage) bool {
	if s == StageRejected {
		return true
	}
	for _, v := range applicantStageOrder {
		if v == s {
			return true
		}
	}
	return false
}

// Applicant is one candidate in the pipeline for a given job opening.
type Applicant struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Phone     string         `json:"phone,omitempty"`
	JobTitle  string         `json:"job_title"`
	ResumeURL string         `json:"resume_url,omitempty"`
	Stage     ApplicantStage `json:"stage"`

	// RecruiterID links to the User handling this candidate — used by the
	// "Chat with Recruiter" feature from the SOW.
	RecruiterID string `json:"recruiter_id,omitempty"`

	Notes []ApplicantNote `json:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ApplicantNote struct {
	AuthorID  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type NewApplicantInput struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Phone       string `json:"phone,omitempty"`
	JobTitle    string `json:"job_title"`
	ResumeURL   string `json:"resume_url,omitempty"`
	RecruiterID string `json:"recruiter_id,omitempty"`
}
