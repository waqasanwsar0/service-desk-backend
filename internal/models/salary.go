package models

import "time"

type SalaryStatus string

const (
	SalaryPending SalaryStatus = "Pending"
	SalaryPaid    SalaryStatus = "Paid"
)

// SalaryRecord tracks one employee's pay for one period (month) — the
// SOW's "Employee salary management" item under Accounting & Finance,
// kept separate from engineer/vendor billing since salaries are an
// internal cost, not client-billable.
type SalaryRecord struct {
	ID           string       `json:"id"`
	EmployeeName string       `json:"employee_name"`
	UserID       string       `json:"user_id,omitempty"`
	PayPeriod    string       `json:"pay_period"` // "2026-08"
	Amount       float64      `json:"amount"`
	Currency     string       `json:"currency"`
	Status       SalaryStatus `json:"status"`
	PaidAt       *time.Time   `json:"paid_at,omitempty"`
	Notes        string       `json:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

type NewSalaryInput struct {
	EmployeeName string  `json:"employee_name"`
	UserID       string  `json:"user_id,omitempty"`
	PayPeriod    string  `json:"pay_period"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Notes        string  `json:"notes,omitempty"`
}
