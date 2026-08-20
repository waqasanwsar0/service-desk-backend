package handlers

import "net/http"

// NewRouter wires up all routes on Go's stdlib ServeMux.
// Go 1.22+ ServeMux supports "METHOD /path/{param}" patterns natively,
// so no external router dependency is needed for this module.
func NewRouter(th *TicketHandler, ah *AuthHandler, eh *EngineerHandler, atth *AttendanceHandler, tsh *TimesheetHandler, ach *AccountingHandler, aph *ApplicantHandler, dh *DashboardHandler, asgh *AssignmentHandler, ch *ContractHandler, oh *OutreachHandler, bh *BackupHandler, ph *ProjectHandler, dsh *DispatchHandler, rh *RequirementHandler, lh *LeadHandler, sth *SocialTaskHandler, salh *SalaryHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// --- Auth ---
	// Register is left open so the very first admin account can be created.
	// Once real deployment starts, protect it with RequireRole(admin) and
	// seed the first admin via a one-off script/migration instead.
	mux.HandleFunc("POST /api/auth/register", ah.Register)
	mux.HandleFunc("POST /api/auth/login", ah.Login)
	mux.Handle("GET /api/auth/me", Chain(http.HandlerFunc(ah.Me), RequireAuth))

	// --- Tickets ---
	mux.Handle("POST /api/tickets", Chain(http.HandlerFunc(th.CreateTicket), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin))))
	mux.Handle("GET /api/tickets", Chain(http.HandlerFunc(th.ListTickets), RequireAuth))
	mux.Handle("GET /api/tickets/{id}", Chain(http.HandlerFunc(th.GetTicket), RequireAuth))
	mux.Handle("PATCH /api/tickets/{id}/status", Chain(http.HandlerFunc(th.UpdateStatus), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin), string(roleEngineer))))
	mux.Handle("PATCH /api/tickets/{id}/assign", Chain(http.HandlerFunc(th.AssignEngineer), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin))))
	mux.Handle("POST /api/tickets/{id}/images", Chain(http.HandlerFunc(th.AddImage), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin), string(roleEngineer))))

	// Landing point for the future Microsoft Graph (Outlook) email importer.
	// The importer service will authenticate with its own service account
	// once built; for now it requires the same roles as manual ticket creation.
	mux.Handle("POST /api/tickets/import/email", Chain(http.HandlerFunc(th.ImportFromEmail), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin))))

	// --- Engineers ---
	mux.Handle("POST /api/engineers", Chain(http.HandlerFunc(eh.CreateEngineer), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin), string(roleRecruiter))))
	mux.Handle("GET /api/engineers", Chain(http.HandlerFunc(eh.SearchEngineers), RequireAuth))
	mux.Handle("GET /api/engineers/{id}", Chain(http.HandlerFunc(eh.GetEngineer), RequireAuth))
	mux.Handle("PATCH /api/engineers/{id}/availability", Chain(http.HandlerFunc(eh.SetAvailability), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin), string(roleEngineer))))

	// --- Attendance (FTE) ---
	mux.Handle("POST /api/attendance/checkin", Chain(http.HandlerFunc(atth.CheckIn), RequireAuth,
		RequireRole(string(roleEngineer), string(roleAdmin))))
	mux.Handle("POST /api/attendance/checkout", Chain(http.HandlerFunc(atth.CheckOut), RequireAuth,
		RequireRole(string(roleEngineer), string(roleAdmin))))
	mux.Handle("POST /api/attendance/leave-request", Chain(http.HandlerFunc(atth.RequestLeave), RequireAuth,
		RequireRole(string(roleEngineer), string(roleAdmin))))
	mux.Handle("PATCH /api/attendance/leave-request/{id}/decision", Chain(http.HandlerFunc(atth.DecideLeave), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin))))
	mux.Handle("GET /api/attendance", Chain(http.HandlerFunc(atth.ListAttendance), RequireAuth))

	// --- Timesheets & Billing ---
	mux.Handle("POST /api/timesheets", Chain(http.HandlerFunc(tsh.CreateTimesheet), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin), string(roleEngineer))))
	mux.Handle("GET /api/timesheets", Chain(http.HandlerFunc(tsh.ListTimesheets), RequireAuth))
	mux.Handle("GET /api/timesheets/{id}", Chain(http.HandlerFunc(tsh.GetTimesheet), RequireAuth))
	mux.Handle("PATCH /api/timesheets/{id}/sign", Chain(http.HandlerFunc(tsh.SignTimesheet), RequireAuth,
		RequireRole(string(roleEngineer), string(roleAdmin))))
	mux.Handle("PATCH /api/timesheets/{id}/approve", Chain(http.HandlerFunc(tsh.ApproveTimesheet), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin), string(roleAccounts))))
	mux.Handle("PATCH /api/timesheets/{id}/reject", Chain(http.HandlerFunc(tsh.RejectTimesheet), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin), string(roleAccounts))))

	// --- Accounting (multi-entity, multi-currency) ---
	mux.Handle("POST /api/entities", Chain(http.HandlerFunc(ach.CreateEntity), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("GET /api/entities", Chain(http.HandlerFunc(ach.ListEntities), RequireAuth))
	mux.Handle("POST /api/invoices", Chain(http.HandlerFunc(ach.CreateInvoice), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("GET /api/invoices", Chain(http.HandlerFunc(ach.ListInvoices), RequireAuth))
	mux.Handle("GET /api/invoices/{id}", Chain(http.HandlerFunc(ach.GetInvoice), RequireAuth))
	mux.Handle("PATCH /api/invoices/{id}/send", Chain(http.HandlerFunc(ach.MarkInvoiceSent), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("PATCH /api/invoices/{id}/paid", Chain(http.HandlerFunc(ach.MarkInvoicePaid), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("PATCH /api/invoices/{id}/payment", Chain(http.HandlerFunc(ach.RecordPayment), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("PATCH /api/invoices/{id}/cancel", Chain(http.HandlerFunc(ach.CancelInvoice), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("PATCH /api/invoices/{id}/overdue", Chain(http.HandlerFunc(ach.MarkInvoiceOverdue), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("GET /api/invoices/{id}/adjustment-notes", Chain(http.HandlerFunc(ach.ListAdjustmentNotes), RequireAuth))
	mux.Handle("POST /api/adjustment-notes", Chain(http.HandlerFunc(ach.CreateAdjustmentNote), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))

	// --- Vendor bills & profitability ---
	mux.Handle("POST /api/vendor-bills", Chain(http.HandlerFunc(ach.CreateVendorBill), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("GET /api/vendor-bills", Chain(http.HandlerFunc(ach.ListVendorBills), RequireAuth))
	mux.Handle("PATCH /api/vendor-bills/{id}/approve", Chain(http.HandlerFunc(ach.ApproveVendorBill), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("PATCH /api/vendor-bills/{id}/pay", Chain(http.HandlerFunc(ach.PayVendorBill), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("GET /api/tickets/{id}/profitability", Chain(http.HandlerFunc(ach.TicketProfitability), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts), string(roleServiceDesk))))

	// --- Contracts / SOW tracking ---
	mux.Handle("POST /api/contracts", Chain(http.HandlerFunc(ch.CreateContract), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("GET /api/contracts", Chain(http.HandlerFunc(ch.ListContracts), RequireAuth))
	mux.Handle("GET /api/contracts/{id}", Chain(http.HandlerFunc(ch.GetContract), RequireAuth))

	// --- LinkedIn / manual outreach tracking (recruitment) ---
	mux.Handle("POST /api/outreach", Chain(http.HandlerFunc(oh.CreateOutreach), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/outreach", Chain(http.HandlerFunc(oh.ListOutreach), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/outreach/{id}", Chain(http.HandlerFunc(oh.GetOutreach), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("PATCH /api/outreach/{id}/status", Chain(http.HandlerFunc(oh.UpdateOutreachStatus), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("POST /api/outreach/{id}/notes", Chain(http.HandlerFunc(oh.AddOutreachNote), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))

	// --- Database backup (admin only) ---
	mux.Handle("GET /api/admin/backup", Chain(http.HandlerFunc(bh.DownloadBackup), RequireAuth,
		RequireRole(string(roleAdmin))))
	mux.Handle("GET /api/admin/backup/status", Chain(http.HandlerFunc(bh.BackupStatus), RequireAuth,
		RequireRole(string(roleAdmin))))

	// --- Projects (SOW: country/city, Dispatch/FTE typing) ---
	mux.Handle("POST /api/projects", Chain(http.HandlerFunc(ph.CreateProject), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin))))
	mux.Handle("GET /api/projects", Chain(http.HandlerFunc(ph.ListProjects), RequireAuth))
	mux.Handle("GET /api/projects/{id}", Chain(http.HandlerFunc(ph.GetProject), RequireAuth))

	// --- Dispatches (auto-generates the linked ticket) ---
	mux.Handle("POST /api/dispatches", Chain(http.HandlerFunc(dsh.CreateDispatch), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin))))
	mux.Handle("GET /api/dispatches", Chain(http.HandlerFunc(dsh.ListDispatches), RequireAuth))

	// --- Client requirements (portal upload + email intake landing point) ---
	mux.Handle("POST /api/requirements", Chain(http.HandlerFunc(rh.CreateRequirement), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin))))
	mux.Handle("POST /api/requirements/import/email", Chain(http.HandlerFunc(rh.ImportFromEmail), RequireAuth,
		RequireRole(string(roleServiceDesk), string(roleAdmin))))
	mux.Handle("GET /api/requirements", Chain(http.HandlerFunc(rh.ListRequirements), RequireAuth))
	mux.Handle("GET /api/requirements/{id}", Chain(http.HandlerFunc(rh.GetRequirement), RequireAuth))

	// --- Sales CRM (LinkedIn / business-development leads) ---
	mux.Handle("POST /api/leads", Chain(http.HandlerFunc(lh.CreateLead), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/leads", Chain(http.HandlerFunc(lh.ListLeads), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/leads/report", Chain(http.HandlerFunc(lh.LeadsReport), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/leads/{id}", Chain(http.HandlerFunc(lh.GetLead), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("PATCH /api/leads/{id}/status", Chain(http.HandlerFunc(lh.UpdateLeadStatus), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("POST /api/leads/{id}/notes", Chain(http.HandlerFunc(lh.AddLeadNote), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))

	// --- Social media task management ---
	mux.Handle("POST /api/social-tasks", Chain(http.HandlerFunc(sth.CreateSocialTask), RequireAuth))
	mux.Handle("GET /api/social-tasks", Chain(http.HandlerFunc(sth.ListSocialTasks), RequireAuth))
	mux.Handle("GET /api/social-tasks/dashboard", Chain(http.HandlerFunc(sth.SocialTaskDashboard), RequireAuth))
	mux.Handle("GET /api/social-tasks/{id}", Chain(http.HandlerFunc(sth.GetSocialTask), RequireAuth))
	mux.Handle("PATCH /api/social-tasks/{id}/status", Chain(http.HandlerFunc(sth.UpdateSocialTaskStatus), RequireAuth))
	mux.Handle("POST /api/social-tasks/{id}/notes", Chain(http.HandlerFunc(sth.AddSocialTaskNote), RequireAuth))

	// --- Employee salary management ---
	mux.Handle("POST /api/salaries", Chain(http.HandlerFunc(salh.CreateSalary), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("GET /api/salaries", Chain(http.HandlerFunc(salh.ListSalaries), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))
	mux.Handle("PATCH /api/salaries/{id}/paid", Chain(http.HandlerFunc(salh.MarkSalaryPaid), RequireAuth,
		RequireRole(string(roleAdmin), string(roleAccounts))))

	// --- Recruitment ATS ---
	mux.Handle("POST /api/applicants", Chain(http.HandlerFunc(aph.CreateApplicant), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/applicants", Chain(http.HandlerFunc(aph.ListApplicants), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/applicants/{id}", Chain(http.HandlerFunc(aph.GetApplicant), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("PATCH /api/applicants/{id}/stage", Chain(http.HandlerFunc(aph.MoveStage), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	// Notes ("Chat with Recruiter") — recruiter and the applicant's own
	// portal session both post here once applicant-facing auth exists;
	// for now it's recruiter/admin like the rest of the ATS.
	mux.Handle("POST /api/applicants/{id}/notes", Chain(http.HandlerFunc(aph.AddNote), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/applicants/dashboard", Chain(http.HandlerFunc(aph.RecruitmentDashboard), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))
	mux.Handle("GET /api/recruiters/performance", Chain(http.HandlerFunc(aph.RecruiterPerformance), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))

	// --- Engineer multi-project assignments (re-hire) ---
	mux.Handle("POST /api/engineers/{id}/assignments", Chain(http.HandlerFunc(asgh.CreateAssignment), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin), string(roleServiceDesk))))
	mux.Handle("GET /api/engineers/{id}/assignments", Chain(http.HandlerFunc(asgh.ListAssignments), RequireAuth))
	mux.Handle("PATCH /api/assignments/{id}/end", Chain(http.HandlerFunc(asgh.EndAssignment), RequireAuth,
		RequireRole(string(roleRecruiter), string(roleAdmin))))

	// --- Dashboard (SOW section 8) ---
	mux.Handle("GET /api/dashboard", Chain(http.HandlerFunc(dh.GetDashboard), RequireAuth))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return mux
}

// Local role constants mirror models.Role — kept here as unexported string
// aliases purely to keep this file readable without an extra import alias
// on every line. They must stay in sync with internal/models.Role.
type role string

const (
	roleAdmin       role = "admin"
	roleServiceDesk role = "service_desk"
	roleEngineer    role = "engineer"
	roleRecruiter   role = "recruiter"
	roleAccounts    role = "accounts"
)
