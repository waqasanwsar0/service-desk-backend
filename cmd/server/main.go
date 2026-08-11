package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	pgdb "servicedesk/internal/db"
	"servicedesk/internal/handlers"
	"servicedesk/internal/store"
	"servicedesk/internal/store/postgres"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var (
		ticketStore     store.TicketStore
		userStore       store.UserStore
		engineerStore   store.EngineerStore
		attendanceStore store.AttendanceStore
		timesheetStore  store.TimesheetStore
		accountingStore store.AccountingStore
		applicantStore  store.ApplicantStore
		assignmentStore store.AssignmentStore
		contractStore   store.ContractStore
	)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		conn := mustConnectAndMigrate(databaseURL)
		defer conn.Close()

		ticketStore = postgres.NewTicketStore(conn)
		userStore = postgres.NewUserStore(conn)
		engineerStore = postgres.NewEngineerStore(conn)
		attendanceStore = postgres.NewAttendanceStore(conn)
		timesheetStore = postgres.NewTimesheetStore(conn)
		accountingStore = postgres.NewAccountingStore(conn)
		applicantStore = postgres.NewApplicantStore(conn)
		assignmentStore = postgres.NewAssignmentStore(conn)
		contractStore = postgres.NewContractStore(conn)

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

		log.Println("Using in-memory storage (no DATABASE_URL set) — data resets on restart.")
	}

	ticketHandler := handlers.NewTicketHandler(ticketStore)
	authHandler := handlers.NewAuthHandler(userStore)
	engineerHandler := handlers.NewEngineerHandler(engineerStore)
	attendanceHandler := handlers.NewAttendanceHandler(attendanceStore)
	timesheetHandler := handlers.NewTimesheetHandler(timesheetStore, ticketStore, engineerStore)
	accountingHandler := handlers.NewAccountingHandler(accountingStore, timesheetStore, ticketStore)
	applicantHandler := handlers.NewApplicantHandler(applicantStore)
	dashboardHandler := handlers.NewDashboardHandler(ticketStore, engineerStore, timesheetStore, attendanceStore)
	assignmentHandler := handlers.NewAssignmentHandler(assignmentStore, engineerStore)
	contractHandler := handlers.NewContractHandler(contractStore)

	router := handlers.NewRouter(ticketHandler, authHandler, engineerHandler, attendanceHandler, timesheetHandler, accountingHandler, applicantHandler, dashboardHandler, assignmentHandler, contractHandler)

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
