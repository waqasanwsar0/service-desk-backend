package models

import "time"

// Contract mirrors the SOW's "Contract & SOW (Statement of Work) Tracking"
// requirement: scope, billing terms, and lifecycle dates with renewal
// reminders.
type Contract struct {
	ID           string `json:"id"`
	ClientName   string `json:"client_name"`
	EntityID     string `json:"entity_id"`
	ProjectScope string `json:"project_scope"`
	BillingTerms string `json:"billing_terms"` // e.g. "Hourly", "Fixed price"

	StartDate  string `json:"start_date"`  // YYYY-MM-DD
	ExpiryDate string `json:"expiry_date"` // YYYY-MM-DD
	AutoRenew  bool   `json:"auto_renew"`

	CreatedAt time.Time `json:"created_at"`
}

type NewContractInput struct {
	ClientName   string `json:"client_name"`
	EntityID     string `json:"entity_id"`
	ProjectScope string `json:"project_scope"`
	BillingTerms string `json:"billing_terms"`
	StartDate    string `json:"start_date"`
	ExpiryDate   string `json:"expiry_date"`
	AutoRenew    bool   `json:"auto_renew"`
}
