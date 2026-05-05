package app

import (
	"fmt"
	"strings"

	"github.com/denislee/wllinear/internal/linear"
)

// --- Events posted by goroutines ---

type ViewerLoaded struct{ User linear.User }

func (e ViewerLoaded) apply(s *State) {
	u := e.User
	s.User = &u
	if s.Team != nil {
		go fetchIssues(s, s.Team.ID, s.ActiveFilter)
	}
	go fetchLeadingProjects(s)
}

type TeamsLoaded struct{ Teams []linear.Team }

func (e TeamsLoaded) apply(s *State) {
	s.Teams = e.Teams
	if s.Team == nil && len(e.Teams) > 0 {
		// Try restoring from saved state.
		idx := 0
		if s.Saved != nil && s.Saved.LastTeamID != "" {
			for i, t := range e.Teams {
				if t.ID == s.Saved.LastTeamID {
					idx = i
					break
				}
			}
		}
		t := e.Teams[idx]
		s.Team = &t
		go fetchIssues(s, t.ID, s.ActiveFilter)
		go fetchTeamMetadata(s, t.ID)
	}
}

type LeadingProjectsLoaded struct{ Projects []linear.Project }

func (e LeadingProjectsLoaded) apply(s *State) {
	s.LeadingProjects = e.Projects
}

type TeamSelected struct{ Team linear.Team }

func (e TeamSelected) apply(s *State) {
	t := e.Team
	s.Team = &t
	s.Selected = 0
	s.View = ViewIssueList
	s.Detail = nil
	s.StatusText = "Loading issues for " + t.Name + "..."
	s.StatusKind = StatusInfo
	go fetchIssues(s, t.ID, s.ActiveFilter)
	go fetchTeamMetadata(s, t.ID)
}

type FilterSelected struct{ Filter string }

func (e FilterSelected) apply(s *State) {
	if e.Filter == "---" {
		return
	}
	s.ActiveFilter = e.Filter
	s.Selected = 0
	s.View = ViewIssueList
	if s.Team != nil {
		s.StatusText = "Loading " + e.Filter + "..."
		s.StatusKind = StatusInfo
		go fetchIssues(s, s.Team.ID, e.Filter)
	}
}

type ProjectSelected struct {
	Project linear.Project
}

func (e ProjectSelected) apply(s *State) {
	filter := "Project: " + e.Project.Name
	s.ActiveFilter = filter
	s.Selected = 0
	s.View = ViewProjectCycles
	if s.Team != nil {
		s.StatusText = "Loading cycles for " + e.Project.Name + "..."
		s.StatusKind = StatusInfo
		go fetchProjectCycles(s, e.Project)
	}
}

type ProjectCyclesLoaded struct {
	ProjectID string
	Cycles    []linear.ProjectCycleIssues
}

func (e ProjectCyclesLoaded) apply(s *State) {
	s.ProjectCycles = e.Cycles
	if s.Selected >= len(s.ProjectCycles) {
		s.Selected = 0
	}
	s.StatusText = fmt.Sprintf("Loaded %d cycles", len(e.Cycles))
	s.StatusKind = StatusOk
}

type IssuesLoaded struct {
	FilterName string
	Issues     []linear.Issue
}

func (e IssuesLoaded) apply(s *State) {
	if e.FilterName != s.ActiveFilter {
		return // race: a more recent fetch took precedence
	}
	s.Issues = e.Issues
	if s.Selected >= len(s.Issues) {
		s.Selected = 0
	}
	s.StatusText = fmt.Sprintf("Loaded %d issues", len(e.Issues))
	s.StatusKind = StatusOk
}

type FilterCountsLoaded struct{ Counts map[string]int }

func (e FilterCountsLoaded) apply(s *State) {
	s.FilterCounts = e.Counts
}

type TeamMetadataLoaded struct{ Meta *linear.TeamMetadata }

func (e TeamMetadataLoaded) apply(s *State) {
	s.Meta = e.Meta
	if e.Meta != nil {
		s.Projects = e.Meta.Projects
	}
	s.rebuildFilters()
	if s.Team != nil {
		go fetchFilterCounts(s, s.Team.ID, s.Filters)
	}
	// Forward to the active create screen or open modal.
	if s.Create != nil {
		s.Create.metaReady(s)
	}
	if md, ok := s.ModalState.(*CreateModal); ok && s.Modal == ModalCreate {
		md.metaReady(s)
	}
	if md, ok := s.ModalState.(*EditModal); ok && s.Modal == ModalEdit {
		md.metaReady(s)
	}
}

type WorkflowStatesLoaded struct{ States []linear.WorkflowState }

func (e WorkflowStatesLoaded) apply(s *State) {
	if md, ok := s.ModalState.(*StatusModal); ok && s.Modal == ModalStatus {
		md.SetStates(e.States)
	}
}

type IssueUpdated struct{ Issue linear.Issue }

func (e IssueUpdated) apply(s *State) {
	for i, is := range s.Issues {
		if is.ID == e.Issue.ID {
			s.Issues[i] = e.Issue
		}
	}
	if s.Detail != nil && s.Detail.ID == e.Issue.ID {
		issue := e.Issue
		s.Detail = &issue
	}
	s.StatusText = "Updated " + e.Issue.Identifier
	s.StatusKind = StatusOk
	if s.Team != nil {
		go fetchIssues(s, s.Team.ID, s.ActiveFilter)
		go fetchFilterCounts(s, s.Team.ID, s.Filters)
	}
}

type IssueCreated struct{ Issue linear.Issue }

func (e IssueCreated) apply(s *State) {
	s.StatusText = "Created " + e.Issue.Identifier
	s.StatusKind = StatusOk
	if s.Team != nil {
		go fetchIssues(s, s.Team.ID, s.ActiveFilter)
		go fetchFilterCounts(s, s.Team.ID, s.Filters)
	}
}

type MyIssuesLoaded struct{ Issues []linear.Issue }

func (e MyIssuesLoaded) apply(s *State) {
	if md, ok := s.ModalState.(*SearchModal); ok && s.Modal == ModalSearch {
		md.SetIssues(e.Issues)
	}
}

// StatusMsg sets the status bar text.
type StatusMsg struct {
	Text string
	Kind StatusKind
}

func (e StatusMsg) apply(s *State) {
	s.StatusText = e.Text
	s.StatusKind = e.Kind
}

// rebuildFilters recomputes the sidebar filter list from current user/projects.
func (s *State) rebuildFilters() {
	s.Filters = []string{
		"My Issues",
		"My Issues + Active",
		"My Unlabeled Issues",
	}
}

// buildIssueFilter converts a sidebar filter name to a Linear GraphQL IssueFilter.
func buildIssueFilter(filterName string, s *State) map[string]any {
	switch filterName {
	case "My Issues":
		if s.User != nil {
			return map[string]any{"assignee": map[string]any{"id": map[string]any{"eq": s.User.ID}}}
		}
	case "My Unlabeled Issues":
		if s.User != nil {
			return map[string]any{
				"assignee": map[string]any{"id": map[string]any{"eq": s.User.ID}},
				"labels":   map[string]any{"length": map[string]any{"eq": 0}},
			}
		}
	case "My Issues + Active":
		if s.User != nil {
			return map[string]any{"and": []map[string]any{
				{"assignee": map[string]any{"id": map[string]any{"eq": s.User.ID}}},
				{"state": map[string]any{"type": map[string]any{"eq": "started"}}},
			}}
		}
	}

	// Project filters.
	if strings.HasPrefix(filterName, "Project: ") {
		name := strings.TrimPrefix(filterName, "Project: ")
		for _, p := range s.LeadingProjects {
			if p.Name == name {
				return map[string]any{"project": map[string]any{"id": map[string]any{"eq": p.ID}}}
			}
		}
	}

	return nil
}
