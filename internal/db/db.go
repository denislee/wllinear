package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/denislee/wllinear/internal/linear"
)

// DB provides a local SQLite cache for issues and related resources.
//
// Each cache table stores ONE blob row per logical key (team+filter, project,
// team, etc.) and an updated_at timestamp so callers can implement a
// stale-while-revalidate freshness policy.
type DB struct {
	conn *sql.DB
}

// Open initializes the SQLite database at path.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	// Drop legacy row-per-item tables if they exist — the data is purely a
	// cache and the cost of refetching once is negligible.
	if _, err := conn.Exec(`
		DROP TABLE IF EXISTS issues_cache;
		DROP TABLE IF EXISTS project_cycles_cache;
		DROP TABLE IF EXISTS workflow_states_cache;
	`); err != nil {
		return nil, fmt.Errorf("drop legacy caches: %w", err)
	}

	if _, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS issues_blob (
			team_id TEXT NOT NULL,
			filter_name TEXT NOT NULL,
			data BLOB NOT NULL,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY (team_id, filter_name)
		);
		CREATE TABLE IF NOT EXISTS project_cycles_blob (
			project_id TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			updated_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS workflow_states_blob (
			team_id TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			updated_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS viewer_blob (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			data BLOB NOT NULL,
			updated_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS teams_blob (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			data BLOB NOT NULL,
			updated_at INTEGER NOT NULL
		);
	`); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func Close(d *DB) error {
	if d == nil {
		return nil
	}
	return d.conn.Close()
}

func nowSec() int64 { return time.Now().Unix() }

// readBlob fetches data + updated_at for a single-key blob.
func (d *DB) readBlob(query string, args ...any) ([]byte, time.Time, error) {
	if d == nil {
		return nil, time.Time{}, nil
	}
	var data []byte
	var ts int64
	row := d.conn.QueryRow(query, args...)
	if err := row.Scan(&data, &ts); err != nil {
		if err == sql.ErrNoRows {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, err
	}
	return data, time.Unix(ts, 0), nil
}

// --- Issues ---

// GetIssues retrieves cached issues for a team and filter.
func (d *DB) GetIssues(teamID, filterName string) ([]linear.Issue, error) {
	issues, _, err := d.GetIssuesWithTime(teamID, filterName)
	return issues, err
}

// GetIssuesWithTime returns cached issues plus the time they were stored.
// Returns (nil, zero time, nil) when there is no cache entry.
func (d *DB) GetIssuesWithTime(teamID, filterName string) ([]linear.Issue, time.Time, error) {
	data, ts, err := d.readBlob(
		"SELECT data, updated_at FROM issues_blob WHERE team_id = ? AND filter_name = ?",
		teamID, filterName,
	)
	if err != nil || data == nil {
		return nil, ts, err
	}
	var issues []linear.Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, ts, err
	}
	return issues, ts, nil
}

// SaveIssues overwrites the cached issues for a team and filter.
func (d *DB) SaveIssues(teamID, filterName string, issues []linear.Issue) error {
	if d == nil {
		return nil
	}
	data, err := json.Marshal(issues)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO issues_blob (team_id, filter_name, data, updated_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(team_id, filter_name) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		teamID, filterName, data, nowSec(),
	)
	return err
}

// --- Project cycles ---

func (d *DB) GetProjectCycles(projectID string) ([]linear.ProjectCycleIssues, error) {
	cycles, _, err := d.GetProjectCyclesWithTime(projectID)
	return cycles, err
}

func (d *DB) GetProjectCyclesWithTime(projectID string) ([]linear.ProjectCycleIssues, time.Time, error) {
	data, ts, err := d.readBlob(
		"SELECT data, updated_at FROM project_cycles_blob WHERE project_id = ?",
		projectID,
	)
	if err != nil || data == nil {
		return nil, ts, err
	}
	var cycles []linear.ProjectCycleIssues
	if err := json.Unmarshal(data, &cycles); err != nil {
		return nil, ts, err
	}
	return cycles, ts, nil
}

func (d *DB) SaveProjectCycles(projectID string, cycles []linear.ProjectCycleIssues) error {
	if d == nil {
		return nil
	}
	data, err := json.Marshal(cycles)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO project_cycles_blob (project_id, data, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(project_id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		projectID, data, nowSec(),
	)
	return err
}

// --- Workflow states ---

func (d *DB) GetWorkflowStates(teamID string) ([]linear.WorkflowState, error) {
	data, _, err := d.readBlob(
		"SELECT data, updated_at FROM workflow_states_blob WHERE team_id = ?",
		teamID,
	)
	if err != nil || data == nil {
		return nil, err
	}
	var states []linear.WorkflowState
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, err
	}
	return states, nil
}

func (d *DB) SaveWorkflowStates(teamID string, states []linear.WorkflowState) error {
	if d == nil {
		return nil
	}
	data, err := json.Marshal(states)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO workflow_states_blob (team_id, data, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(team_id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		teamID, data, nowSec(),
	)
	return err
}

// --- Viewer & teams (small, frequently re-fetched at startup) ---

func (d *DB) GetViewer() (*linear.User, error) {
	data, _, err := d.readBlob("SELECT data, updated_at FROM viewer_blob WHERE id = 1")
	if err != nil || data == nil {
		return nil, err
	}
	var u linear.User
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (d *DB) SaveViewer(u linear.User) error {
	if d == nil {
		return nil
	}
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO viewer_blob (id, data, updated_at) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		data, nowSec(),
	)
	return err
}

func (d *DB) GetTeams() ([]linear.Team, error) {
	data, _, err := d.readBlob("SELECT data, updated_at FROM teams_blob WHERE id = 1")
	if err != nil || data == nil {
		return nil, err
	}
	var teams []linear.Team
	if err := json.Unmarshal(data, &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

func (d *DB) SaveTeams(teams []linear.Team) error {
	if d == nil {
		return nil
	}
	data, err := json.Marshal(teams)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO teams_blob (id, data, updated_at) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		data, nowSec(),
	)
	return err
}
