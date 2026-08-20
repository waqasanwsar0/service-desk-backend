package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BackupHandler exports the full PostgreSQL database as a single
// downloadable JSON file. It exists because Railway (and most managed
// hosts) don't guarantee automatic backups on lower-cost plans — this
// gives an admin a one-click way to pull a full snapshot at any time,
// so it can be saved somewhere safe (a shared drive, email, etc.).
//
// It only does anything when the app is running against PostgreSQL
// (DATABASE_URL set). In-memory mode has nothing durable to back up.
type BackupHandler struct {
	DB *sql.DB // nil when running in in-memory mode
}

func NewBackupHandler(db *sql.DB) *BackupHandler {
	return &BackupHandler{DB: db}
}

// backupTables lists every table in the schema, in an order that keeps
// referenced rows (e.g. entities before invoices) ahead of the tables
// that reference them, in case this snapshot is ever used to restore.
var backupTables = []string{
	"users", "engineers", "tickets", "attendance_records", "leave_requests",
	"timesheets", "entities", "invoices", "vendor_bills", "adjustment_notes",
	"applicants", "assignments", "contracts", "outreach_contacts",
}

// DownloadBackup handles GET /api/admin/backup
// Streams a JSON file containing every row from every table.
func (h *BackupHandler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeError(w, http.StatusConflict, "backup is only available when running with a PostgreSQL database (DATABASE_URL set) — nothing to back up in in-memory mode")
		return
	}

	snapshot := map[string]interface{}{
		"generated_at": time.Now().UTC(),
	}
	tables := map[string]interface{}{}

	for _, table := range backupTables {
		rows, err := dumpTable(h.DB, table)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed exporting table "+table+": "+err.Error())
			return
		}
		tables[table] = rows
	}
	snapshot["tables"] = tables

	filename := fmt.Sprintf("servicedesk-backup-%s.json", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(snapshot)
}

// BackupStatus handles GET /api/admin/backup/status
// Reports row counts per table without downloading anything — a quick
// sanity check that the database has the data everyone expects.
func (h *BackupHandler) BackupStatus(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"mode":    "in-memory",
			"message": "Running without a database — data is not persisted and there is nothing to back up.",
		})
		return
	}

	counts := map[string]int{}
	for _, table := range backupTables {
		var n int
		if err := h.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			writeError(w, http.StatusInternalServerError, "failed counting table "+table+": "+err.Error())
			return
		}
		counts[table] = n
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mode":       "postgresql",
		"row_counts": counts,
		"checked_at": time.Now().UTC(),
	})
}

// dumpTable runs SELECT * on a table and returns every row as a
// map[string]interface{}, JSON-friendly (byte slices from the driver are
// converted to strings).
func dumpTable(db *sql.DB, table string) ([]map[string]interface{}, error) {
	rows, err := db.Query("SELECT * FROM " + table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			row[col] = normalizeValue(values[i])
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// normalizeValue converts driver-native values into plain JSON-friendly
// types. The pq driver returns []byte for text/JSONB columns — those
// need to become strings, or json.Marshal would base64-encode them.
func normalizeValue(v interface{}) interface{} {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}
