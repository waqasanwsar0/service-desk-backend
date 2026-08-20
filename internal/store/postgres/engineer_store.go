package postgres

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"servicedesk/internal/db"
	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type engineerStore struct {
	db *sql.DB
}

func NewEngineerStore(conn *sql.DB) store.EngineerStore {
	return &engineerStore{db: conn}
}

const engineerColumns = `id, name, email, phone, skills, location, area_coverage, hourly_rate, half_day_rate, day_rate,
	currency, travel_cost, resume_url, documents, approved_projects, available, created_at, updated_at`

func (s *engineerStore) Create(e *models.Engineer) error {
	id, err := db.NextID(s.db, "seq_engineer", "ENG", 4)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	e.ID = id
	e.Available = true
	e.CreatedAt = now
	e.UpdatedAt = now

	skills, _ := json.Marshal(e.Skills)
	approved, _ := json.Marshal(e.ApprovedProjects)
	area, _ := json.Marshal(e.AreaCoverage)
	docs, _ := json.Marshal(e.Documents)

	_, err = s.db.Exec(`
		INSERT INTO engineers (id, name, email, phone, skills, location, area_coverage, hourly_rate,
			half_day_rate, day_rate, currency, travel_cost, resume_url, documents, approved_projects,
			available, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		e.ID, e.Name, e.Email, e.Phone, skills, e.Location, area, e.HourlyRate,
		e.HalfDayRate, e.DayRate, e.Currency, e.TravelCost, e.ResumeURL, docs, approved,
		e.Available, e.CreatedAt, e.UpdatedAt)
	return err
}

func (s *engineerStore) Get(id string) (*models.Engineer, error) {
	row := s.db.QueryRow(`SELECT `+engineerColumns+` FROM engineers WHERE id = $1`, id)
	e, err := scanEngineer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return e, err
}

func (s *engineerStore) Search(filter store.EngineerFilter) []*models.Engineer {
	query := `SELECT ` + engineerColumns + ` FROM engineers WHERE 1=1`
	args := []interface{}{}
	if filter.OnlyAvailable {
		query += " AND available = true"
	}
	if filter.Location != "" {
		args = append(args, filter.Location)
		query += " AND lower(location) = lower($" + strconv.Itoa(len(args)) + ")"
	}
	if filter.Skill != "" {
		args = append(args, filter.Skill)
		query += " AND skills @> to_jsonb($" + strconv.Itoa(len(args)) + "::text)"
	}
	// Named-project filtering (engineers with a non-empty approved_projects
	// list must include this project) is applied in Go below, since it
	// needs an "empty list OR contains" rule that's simplest post-fetch.
	query += " ORDER BY CASE WHEN hourly_rate > 0 THEN hourly_rate ELSE day_rate END ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make([]*models.Engineer, 0)
	for rows.Next() {
		e, err := scanEngineerRows(rows)
		if err != nil {
			continue
		}
		if filter.Project != "" && len(e.ApprovedProjects) > 0 && !containsFold(e.ApprovedProjects, filter.Project) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (s *engineerStore) SetAvailability(id string, available bool) (*models.Engineer, error) {
	e, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE engineers SET available = $1, updated_at = $2 WHERE id = $3`, available, now, id)
	if err != nil {
		return nil, err
	}
	e.Available = available
	e.UpdatedAt = now
	return e, nil
}

func containsFold(list []string, target string) bool {
	for _, s := range list {
		if len(s) == len(target) && (s == target) {
			return true
		}
	}
	return false
}

func scanEngineer(row *sql.Row) (*models.Engineer, error) {
	return scanEngineerGeneric(row)
}

func scanEngineerRows(rows *sql.Rows) (*models.Engineer, error) {
	return scanEngineerGeneric(rows)
}

func scanEngineerGeneric(row rowScanner) (*models.Engineer, error) {
	var e models.Engineer
	var skills, approved, area, docs []byte
	err := row.Scan(&e.ID, &e.Name, &e.Email, &e.Phone, &skills, &e.Location, &area, &e.HourlyRate,
		&e.HalfDayRate, &e.DayRate, &e.Currency, &e.TravelCost, &e.ResumeURL, &docs, &approved,
		&e.Available, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(skills, &e.Skills)
	_ = json.Unmarshal(approved, &e.ApprovedProjects)
	_ = json.Unmarshal(area, &e.AreaCoverage)
	_ = json.Unmarshal(docs, &e.Documents)
	return &e, nil
}
