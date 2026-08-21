package models

import "time"

// File is a generic attachment — image, video, or PDF — usable from any
// module (ticket photos, engineer resumes/documents, timesheet proof,
// requirement files, social media assets). Stored directly in the
// database (bytea) since the deployment has no external object storage
// configured; a max size is enforced by the handler to keep this
// reasonable for images/PDFs and short videos.
type File struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	UploadedBy  string    `json:"uploaded_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	// Data is never included in JSON responses (list/metadata calls) —
	// only the dedicated download endpoint streams it.
	Data []byte `json:"-"`
}
