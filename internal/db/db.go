package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
	"github.com/denislee/wllinear/internal/linear"
)

// DB provides a local SQLite cache for issues.
type DB struct {
	conn *sql.DB
}

// Open initializes the SQLite database at path.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS issues_cache (
			team_id TEXT,
			filter_name TEXT,
			issue_id TEXT,
			data BLOB,
			pos INTEGER,
			PRIMARY KEY (team_id, filter_name, issue_id)
		);
		CREATE INDEX IF NOT EXISTS idx_issues_cache_lookup ON issues_cache (team_id, filter_name, pos);
		CREATE TABLE IF NOT EXISTS project_cycles_cache (
			project_id TEXT,
			cycle_id TEXT,
			data BLOB,
			pos INTEGER,
			PRIMARY KEY (project_id, cycle_id)
		);
		CREATE INDEX IF NOT EXISTS idx_project_cycles_lookup ON project_cycles_cache (project_id, pos);
		CREATE TABLE IF NOT EXISTS workflow_states_cache (
			team_id TEXT,
			state_id TEXT,
			data BLOB,
			pos INTEGER,
			PRIMARY KEY (team_id, state_id)
		);
		CREATE INDEX IF NOT EXISTS idx_workflow_states_lookup ON workflow_states_cache (team_id, pos);
	`)
	if err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &DB{conn: db}, nil
}

// Close closes the database connection.
func Close(d *DB) error {
	if d == nil {
		return nil
	}
	return d.conn.Close()
}

// GetIssues retrieves cached issues for a team and filter.
func (d *DB) GetIssues(teamID, filterName string) ([]linear.Issue, error) {
	if d == nil {
		return nil, nil
	}
	rows, err := d.conn.Query("SELECT data FROM issues_cache WHERE team_id = ? AND filter_name = ? ORDER BY pos", teamID, filterName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []linear.Issue
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var is linear.Issue
		if err := json.Unmarshal(data, &is); err != nil {
			return nil, err
		}
		issues = append(issues, is)
	}
	return issues, nil
}

// SaveIssues overwrites the cached issues for a team and filter.
func (d *DB) SaveIssues(teamID, filterName string, issues []linear.Issue) error {
	if d == nil {
		return nil
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM issues_cache WHERE team_id = ? AND filter_name = ?", teamID, filterName)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO issues_cache (team_id, filter_name, issue_id, data, pos) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, is := range issues {
		data, err := json.Marshal(is)
		if err != nil {
			return err
		}
		_, err = stmt.Exec(teamID, filterName, is.ID, data, i)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetProjectCycles retrieves cached cycles (with their completed issues) for a project.
func (d *DB) GetProjectCycles(projectID string) ([]linear.ProjectCycleIssues, error) {
	if d == nil {
		return nil, nil
	}
	rows, err := d.conn.Query("SELECT data FROM project_cycles_cache WHERE project_id = ? ORDER BY pos", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cycles []linear.ProjectCycleIssues
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var c linear.ProjectCycleIssues
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		cycles = append(cycles, c)
	}
	return cycles, nil
}

// SaveProjectCycles overwrites the cached cycles for a project.
func (d *DB) SaveProjectCycles(projectID string, cycles []linear.ProjectCycleIssues) error {
	if d == nil {
		return nil
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM project_cycles_cache WHERE project_id = ?", projectID)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO project_cycles_cache (project_id, cycle_id, data, pos) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, c := range cycles {
		data, err := json.Marshal(c)
		if err != nil {
			return err
		}
		_, err = stmt.Exec(projectID, c.Cycle.ID, data, i)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetWorkflowStates retrieves cached workflow states for a team.
func (d *DB) GetWorkflowStates(teamID string) ([]linear.WorkflowState, error) {
	if d == nil {
		return nil, nil
	}
	rows, err := d.conn.Query("SELECT data FROM workflow_states_cache WHERE team_id = ? ORDER BY pos", teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []linear.WorkflowState
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var st linear.WorkflowState
		if err := json.Unmarshal(data, &st); err != nil {
			return nil, err
		}
		states = append(states, st)
	}
	return states, nil
}

// SaveWorkflowStates overwrites the cached workflow states for a team.
func (d *DB) SaveWorkflowStates(teamID string, states []linear.WorkflowState) error {
	if d == nil {
		return nil
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM workflow_states_cache WHERE team_id = ?", teamID)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO workflow_states_cache (team_id, state_id, data, pos) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, st := range states {
		data, err := json.Marshal(st)
		if err != nil {
			return err
		}
		_, err = stmt.Exec(teamID, st.ID, data, i)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
