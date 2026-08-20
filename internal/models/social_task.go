package models

import "time"

type SocialTaskStatus string

const (
	SocialTaskDraft    SocialTaskStatus = "Draft"
	SocialTaskInReview SocialTaskStatus = "In Review"
	SocialTaskApproved SocialTaskStatus = "Approved"
	SocialTaskPosted   SocialTaskStatus = "Posted"
)

func IsValidSocialTaskStatus(s SocialTaskStatus) bool {
	switch s {
	case SocialTaskDraft, SocialTaskInReview, SocialTaskApproved, SocialTaskPosted:
		return true
	}
	return false
}

// SocialMediaTask is one piece of content/work for the social media
// team — a post, graphic, or campaign item — tracked from draft through
// approval to posting, per platform.
type SocialMediaTask struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Platform    string           `json:"platform"` // e.g. "Instagram", "LinkedIn", "Facebook"
	Description string           `json:"description,omitempty"`
	OwnerID     string           `json:"owner_id,omitempty"`
	DueDate     string           `json:"due_date,omitempty"` // YYYY-MM-DD
	Status      SocialTaskStatus `json:"status"`
	AssetURL    string           `json:"asset_url,omitempty"`

	Notes []SocialTaskNote `json:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SocialTaskNote struct {
	AuthorID  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type NewSocialTaskInput struct {
	Title       string `json:"title"`
	Platform    string `json:"platform"`
	Description string `json:"description,omitempty"`
	OwnerID     string `json:"owner_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	AssetURL    string `json:"asset_url,omitempty"`
}
