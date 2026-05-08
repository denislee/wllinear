package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type State struct {
	LastTeamID              string    `json:"last_team_id"`
	LastFilter              string    `json:"last_filter"`
	CompactMode             bool      `json:"compact_mode"`
	ShowLabels              bool      `json:"show_labels"`
	ShowPriority            bool      `json:"show_priority"`
	HideHints               bool      `json:"hide_hints"`
	SidebarWidth            int       `json:"sidebar_width"`
	WindowW                 int       `json:"window_w,omitempty"`
	WindowH                 int       `json:"window_h,omitempty"`
	Fonts                   FontPrefs `json:"fonts,omitempty"`
	DefaultCreateStatusType string    `json:"default_create_status_type,omitempty"`
	EnableLogging           bool      `json:"enable_logging"`
}

// FontPref is the persisted form of ui.FontStyle. Mirrored here so the
// config package stays free of UI imports.
type FontPref struct {
	Face string  `json:"face,omitempty"`
	Size float32 `json:"size,omitempty"`
}

// FontPrefs mirrors ui.SectionFonts.
type FontPrefs struct {
	Global      FontPref `json:"global,omitempty"`
	Sidebar     FontPref `json:"sidebar,omitempty"`
	IssueList   FontPref `json:"issue_list,omitempty"`
	IssueDetail FontPref `json:"issue_detail,omitempty"`
	CreateIssue FontPref `json:"create_issue,omitempty"`
	StatusBar   FontPref `json:"status_bar,omitempty"`
	Modal       FontPref `json:"modal,omitempty"`
	Code        FontPref `json:"code,omitempty"`
}

func statePath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "wllinear", "state.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "wllinear", "state.json")
}

func DBPath() string {
	return filepath.Join(filepath.Dir(statePath()), "cache.sqlite")
}

// LoadState reads persisted UI state from disk. Missing or corrupt → defaults.
func LoadState() *State {
	state := &State{
		CompactMode:   true,
		ShowPriority:  true,
		EnableLogging: true,
	}
	if data, err := os.ReadFile(statePath()); err == nil {
		_ = json.Unmarshal(data, state)
	}
	if state.LastFilter == "" {
		state.LastFilter = "My Issues + Active"
	}
	if state.SidebarWidth == 0 {
		state.SidebarWidth = 260
	}
	if state.DefaultCreateStatusType == "" {
		state.DefaultCreateStatusType = "started"
	}
	return state
}

func SaveState(state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	path := statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}
