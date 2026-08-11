package handlers

import (
	"net/http"
	"time"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

// DashboardHandler aggregates read-only views across tickets, engineers,
// timesheets and leave — this backs SOW section 8 ("Dashboard Requirements
// — VERY IMPORTANT"): client-wise tickets, project-wise counts, upcoming
// visits, daily/weekly/month views, timesheet-missing list, engineer
// availability, and the leave calendar.
type DashboardHandler struct {
	Tickets    store.TicketStore
	Engineers  store.EngineerStore
	Timesheets store.TimesheetStore
	Attendance store.AttendanceStore
}

func NewDashboardHandler(t store.TicketStore, e store.EngineerStore, ts store.TimesheetStore, a store.AttendanceStore) *DashboardHandler {
	return &DashboardHandler{Tickets: t, Engineers: e, Timesheets: ts, Attendance: a}
}

// GetDashboard handles GET /api/dashboard
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	allTickets := h.Tickets.List(store.ListFilter{})
	allEngineers := h.Engineers.Search(store.EngineerFilter{})
	allLeaves := h.Attendance.ListLeaveRequests("")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tickets_by_client":     ticketsByClient(allTickets),
		"tickets_by_project":    ticketsByProject(allTickets),
		"upcoming_visits":       upcomingVisits(allTickets),
		"period_counts":         periodCounts(allTickets),
		"timesheet_missing":     timesheetMissing(allTickets, h.Timesheets),
		"engineer_availability": engineerAvailability(allEngineers),
		"leave_calendar":        upcomingLeave(allLeaves),
	})
}

func ticketsByClient(tickets []*models.Ticket) map[string]int {
	out := map[string]int{}
	for _, t := range tickets {
		out[t.ClientName]++
	}
	return out
}

func ticketsByProject(tickets []*models.Ticket) map[string]int {
	out := map[string]int{}
	for _, t := range tickets {
		name := t.ProjectName
		if name == "" {
			name = "(no project)"
		}
		out[name]++
	}
	return out
}

// upcomingVisits lists open tickets with an SLA due time in the future,
// soonest first — the "upcoming visits" filter from the SOW.
func upcomingVisits(tickets []*models.Ticket) []*models.Ticket {
	now := time.Now().UTC()
	out := make([]*models.Ticket, 0)
	for _, t := range tickets {
		if t.Status == models.StatusPaid || t.SLADueAt == nil {
			continue
		}
		if t.SLADueAt.After(now) {
			out = append(out, t)
		}
	}
	// simple insertion sort by SLA due time — lists here are small enough
	// that this stays clear and dependency-free
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j].SLADueAt.Before(*out[j-1].SLADueAt) {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out
}

type periodCount struct {
	Today     int `json:"today"`
	ThisWeek  int `json:"this_week"`
	ThisMonth int `json:"this_month"`
}

// periodCounts gives the daily / weekly / monthly ticket-creation views
// required by the SOW dashboard.
func periodCounts(tickets []*models.Ticket) periodCount {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startOfWeek := startOfDay.AddDate(0, 0, -int(startOfDay.Weekday()))
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var pc periodCount
	for _, t := range tickets {
		if t.CreatedAt.After(startOfDay) {
			pc.Today++
		}
		if t.CreatedAt.After(startOfWeek) {
			pc.ThisWeek++
		}
		if t.CreatedAt.After(startOfMonth) {
			pc.ThisMonth++
		}
	}
	return pc
}

// timesheetMissing flags tickets that are Onsite or later but have no
// timesheet on file yet — the SOW's "timesheet missing list".
func timesheetMissing(tickets []*models.Ticket, timesheets store.TimesheetStore) []*models.Ticket {
	out := make([]*models.Ticket, 0)
	pastOnsite := map[models.TicketStatus]bool{
		models.StatusOnsite:           true,
		models.StatusTimesheetPending: true,
	}
	for _, t := range tickets {
		if !pastOnsite[t.Status] {
			continue
		}
		existing := timesheets.List(t.ID, "")
		if len(existing) == 0 {
			out = append(out, t)
		}
	}
	return out
}

type availabilitySummary struct {
	Available   int `json:"available"`
	Unavailable int `json:"unavailable"`
}

func engineerAvailability(engineers []*models.Engineer) availabilitySummary {
	var s availabilitySummary
	for _, e := range engineers {
		if e.Available {
			s.Available++
		} else {
			s.Unavailable++
		}
	}
	return s
}

// upcomingLeave returns approved leave requests whose window hasn't ended
// yet — the "leave calendar" filter from the SOW.
func upcomingLeave(leaves []*models.LeaveRequest) []*models.LeaveRequest {
	today := time.Now().UTC().Format("2006-01-02")
	out := make([]*models.LeaveRequest, 0)
	for _, l := range leaves {
		if l.Status != models.LeaveApproved {
			continue
		}
		if l.ToDate >= today {
			out = append(out, l)
		}
	}
	return out
}
