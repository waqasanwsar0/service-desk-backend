package postgres

import (
	"database/sql"
	"errors"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type assignmentStore struct {
	db *sql.DB
}

func NewAssignmentStore(conn *sql.DB) store.AssignmentStore {
	return &assignmentStore{db: conn}
}

func (s *assignmentStore) Create(a *models.EngineerAssignment) error {
	id, err := db.NextID(s.db, "seq_assignment", "ASG", 4)
	if err != nil {
		return err
	}
	a.ID = id
	a.Active = true
	a.CreatedAt = time.Now().UTC()
	_, err = s.db.Exec(`INSERT INTO assignments (id, engineer_id, project_name, client_name, hourly_rate, day_rate,
		currency, travel_allowance, tools_allowance, start_date, end_date, active, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		a.ID, a.EngineerID, a.ProjectName, a.ClientName, a.HourlyRate, a.DayRate,
		a.Currency, a.TravelAllowance, a.ToolsAllowance, a.StartDate, a.EndDate, a.Active, a.CreatedAt)
	return err
}

func (s *assignmentStore) ListByEngineer(engineerID string) []*models.EngineerAssignment {
	query := `SELECT id, engineer_id, project_name, client_name, hourly_rate, day_rate,
		currency, travel_allowance, tools_allowance, start_date, end_date, active, created_at FROM assignments WHERE 1=1`
	args := []interface{}{}
	if engineerID != "" {
		args = append(args, engineerID)
		query += " AND engineer_id = $1"
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]*models.EngineerAssignment, 0)
	for rows.Next() {
		var a models.EngineerAssignment
		if err := rows.Scan(&a.ID, &a.EngineerID, &a.ProjectName, &a.ClientName, &a.HourlyRate, &a.DayRate,
			&a.Currency, &a.TravelAllowance, &a.ToolsAllowance, &a.StartDate, &a.EndDate, &a.Active, &a.CreatedAt); err == nil {
			out = append(out, &a)
		}
	}
	return out
}

func (s *assignmentStore) End(id string) (*models.EngineerAssignment, error) {
	endDate := time.Now().UTC().Format("2006-01-02")
	res, err := s.db.Exec(`UPDATE assignments SET active=false, end_date=$1 WHERE id=$2`, endDate, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, store.ErrNotFound
	}
	var a models.EngineerAssignment
	err = s.db.QueryRow(`SELECT id, engineer_id, project_name, client_name, hourly_rate, day_rate,
		currency, travel_allowance, tools_allowance, start_date, end_date, active, created_at FROM assignments WHERE id=$1`, id).
		Scan(&a.ID, &a.EngineerID, &a.ProjectName, &a.ClientName, &a.HourlyRate, &a.DayRate,
			&a.Currency, &a.TravelAllowance, &a.ToolsAllowance, &a.StartDate, &a.EndDate, &a.Active, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return &a, err
}
