package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	pgdb "servicedesk/internal/db"
	"servicedesk/internal/handlers"
	"servicedesk/internal/outlook"
	"servicedesk/internal/store"
	"servicedesk/internal/store/postgres"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var (
		ticketStore      store.TicketStore
		userStore        store.UserStore
		engineerStore    store.EngineerStore
		attendanceStore  store.AttendanceStore
		timesheetStore   store.TimesheetStore
		accountingStore  store.AccountingStore
		applicantStore   store.ApplicantStore
		assignmentStore  store.AssignmentStore
		contractStore    store.ContractStore
		outreachStore    store.OutreachStore
		projectStore     store.ProjectStore
		dispatchStore    store.DispatchStore
		requirementStore store.RequirementStore
		leadStore        store.LeadStore
		socialTaskStore  store.SocialTaskStore
		salaryStore      store.SalaryStore
		fileStore        store.FileStore
		rawDB            *sql.DB // kept for the backup/export endpoint; nil in in-memory mode
	)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		conn := mustConnectAndMigrate(databaseURL)
		defer conn.Close()
		rawDB = conn

		ticketStore = postgres.NewTicketStore(conn)
		userStore = postgres.NewUserStore(conn)
		engineerStore = postgres.NewEngineerStore(conn)
		attendanceStore = postgres.NewAttendanceStore(conn)
		timesheetStore = postgres.NewTimesheetStore(conn)
		accountingStore = postgres.NewAccountingStore(conn)
		applicantStore = postgres.NewApplicantStore(conn)
		assignmentStore = postgres.NewAssignmentStore(conn)
		contractStore = postgres.NewContractStore(conn)
		outreachStore = postgres.NewOutreachStore(conn)
		projectStore = postgres.NewProjectStore(conn)
		dispatchStore = postgres.NewDispatchStore(conn)
		requirementStore = postgres.NewRequirementStore(conn)
		leadStore = postgres.NewLeadStore(conn)
		socialTaskStore = postgres.NewSocialTaskStore(conn)
		salaryStore = postgres.NewSalaryStore(conn)
		fileStore = postgres.NewFileStore(conn)

		log.Println("Using PostgreSQL storage (DATABASE_URL set) — data persists across restarts.")
	} else {
		ticketStore = store.NewMemoryStore()
		userStore = store.NewMemoryUserStore()
		engineerStore = store.NewMemoryEngineerStore()
		attendanceStore = store.NewMemoryAttendanceStore()
		timesheetStore = store.NewMemoryTimesheetStore()
		accountingStore = store.NewMemoryAccountingStore()
		applicantStore = store.NewMemoryApplicantStore()
		assignmentStore = store.NewMemoryAssignmentStore()
		contractStore = store.NewMemoryContractStore()
		outreachStore = store.NewMemoryOutreachStore()
		projectStore = store.NewMemoryProjectStore()
		dispatchStore = store.NewMemoryDispatchStore()
		requirementStore = store.NewMemoryRequirementStore()
		leadStore = store.NewMemoryLeadStore()
		socialTaskStore = store.NewMemorySocialTaskStore()
		salaryStore = store.NewMemorySalaryStore()
		fileStore = store.NewMemoryFileStore()

		log.Println("Using in-memory storage (no DATABASE_URL set) — data resets on restart.")
	}

	notifier := handlers.NewNotifierFromEnv()

	// Outlook ticket intake: only starts if AZURE_TENANT_ID, AZURE_CLIENT_ID,
	// AZURE_CLIENT_SECRET, and AZURE_MAILBOX are all set. Polls the shared
	// mailbox every 2 minutes and turns unread emails into tickets.
	if cfg, ok := outlook.FromEnv(); ok {
		client := outlook.NewClient(cfg)
		go outlook.StartPoller(client, ticketStore, 2*time.Minute)
	} else {
		log.Println("Outlook integration not configured (set AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_MAILBOX) — email ticket intake is off.")
	}

	ticketHandler := handlers.NewTicketHandler(ticketStore, engineerStore, notifier)
	authHandler := handlers.NewAuthHandler(userStore)
	engineerHandler := handlers.NewEngineerHandler(engineerStore)
	attendanceHandler := handlers.NewAttendanceHandler(attendanceStore, engineerStore, notifier)
	timesheetHandler := handlers.NewTimesheetHandler(timesheetStore, ticketStore, engineerStore)
	accountingHandler := handlers.NewAccountingHandler(accountingStore, timesheetStore, ticketStore)
	applicantHandler := handlers.NewApplicantHandler(applicantStore)
	dashboardHandler := handlers.NewDashboardHandler(ticketStore, engineerStore, timesheetStore, attendanceStore)
	assignmentHandler := handlers.NewAssignmentHandler(assignmentStore, engineerStore)
	contractHandler := handlers.NewContractHandler(contractStore)
	outreachHandler := handlers.NewOutreachHandler(outreachStore)
	backupHandler := handlers.NewBackupHandler(rawDB)
	projectHandler := handlers.NewProjectHandler(projectStore)
	dispatchHandler := handlers.NewDispatchHandler(dispatchStore, ticketStore, engineerStore, notifier)
	requirementHandler := handlers.NewRequirementHandler(requirementStore)
	leadHandler := handlers.NewLeadHandler(leadStore)
	socialTaskHandler := handlers.NewSocialTaskHandler(socialTaskStore)
	salaryHandler := handlers.NewSalaryHandler(salaryStore)
	fileHandler := handlers.NewFileHandler(fileStore)

	router := handlers.NewRouter(ticketHandler, authHandler, engineerHandler, attendanceHandler, timesheetHandler, accountingHandler, applicantHandler, dashboardHandler, assignmentHandler, contractHandler, outreachHandler, backupHandler, projectHandler, dispatchHandler, requirementHandler, leadHandler, socialTaskHandler, salaryHandler, fileHandler)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      logRequests(handlers.CORS(router)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Service Desk Ticket Intake API listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// mustConnectAndMigrate opens the PostgreSQL connection and applies the
// schema. schema.sql uses CREATE TABLE/SEQUENCE IF NOT EXISTS throughout,
// so running it on every boot is safe and keeps a fresh database in sync
// without a separate migration step.
func mustConnectAndMigrate(databaseURL string) *sql.DB {
	conn, err := pgdb.Connect(databaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	if _, err := conn.Exec(pgdb.SchemaSQL); err != nil {
		log.Fatalf("schema migration failed: %v", err)
	}
	return conn
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
