package models

import "time"

type AttendanceStatus string

const (
	AttendancePresent AttendanceStatus = "Present"
	AttendanceLeave   AttendanceStatus = "Leave"
	AttendanceAbsent  AttendanceStatus = "Absent"
)

// AttendanceRecord is one engineer's attendance for one calendar day.
// Created by the "I am ON-SITE" portal button described in the SOW.
type AttendanceRecord struct {
	ID         string           `json:"id"`
	EngineerID string           `json:"engineer_id"`
	Date       string           `json:"date"` // YYYY-MM-DD
	Status     AttendanceStatus `json:"status"`
	CheckInAt  time.Time        `json:"check_in_at"`
	Location   string           `json:"location,omitempty"`
	Notes      string           `json:"notes,omitempty"`
}

type LeaveStatus string

const (
	LeavePending  LeaveStatus = "Pending"
	LeaveApproved LeaveStatus = "Approved"
	LeaveRejected LeaveStatus = "Rejected"
)

// LeaveRequest is the in-portal leave request flow from the SOW
// ("Leave request system inside portal").
type LeaveRequest struct {
	ID         string      `json:"id"`
	EngineerID string      `json:"engineer_id"`
	FromDate   string      `json:"from_date"` // YYYY-MM-DD
	ToDate     string      `json:"to_date"`   // YYYY-MM-DD
	Reason     string      `json:"reason"`
	Status     LeaveStatus `json:"status"`
	DecidedBy  string      `json:"decided_by,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}
