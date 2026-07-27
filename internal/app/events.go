package app

import (
	"fmt"
	"log"
	"strings"

	"gioui.org/widget"

	"github.com/denislee/wllinear/internal/linear"
)

// --- Events posted by goroutines ---

type ViewerLoaded struct{ User linear.User }

func (e ViewerLoaded) apply(s *State) {
	u := e.User
	s.User = &u
	if s.Team != nil {
		filter := buildIssueFilter(s.ActiveFilter, s)
		go fetchIssues(s, s.Team.ID, s.ActiveFilter, filter, false)
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
		filter := buildIssueFilter(s.ActiveFilter, s)
		go fetchIssues(s, t.ID, s.ActiveFilter, filter, false)
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
	filter := buildIssueFilter(s.ActiveFilter, s)
	go fetchIssues(s, t.ID, s.ActiveFilter, filter, false)
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
		filter := buildIssueFilter(e.Filter, s)
		go fetchIssues(s, s.Team.ID, e.Filter, filter, false)
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
	// Clear stale cycles from a previously-viewed project so we don't flash old data.
	if s.CurrentProjectID != e.Project.ID {
		s.ProjectCycles = nil
		s.ExpandedCycles = nil
	}
	s.CurrentProjectID = e.Project.ID
	if s.Team != nil {
		s.StatusText = "Loading cycles for " + e.Project.Name + "..."
		s.StatusKind = StatusInfo
		go fetchProjectCycles(s, e.Project, false)
	}
}

type ProjectDetailLoaded struct{ Detail linear.ProjectDetail }

func (e ProjectDetailLoaded) apply(s *State) {
	if s.Modal != ModalProjectInfo {
		return
	}
	m, ok := s.ModalState.(*ProjectInfoModal)
	if !ok || m.ProjectID != e.Detail.ID {
		return
	}
	m.SetDetail(e.Detail)
}

type ProjectCyclesLoaded struct {
	ProjectID string
	Cycles    []linear.ProjectCycleIssues
	FromCache bool
}

func (e ProjectCyclesLoaded) apply(s *State) {
	if e.ProjectID != s.CurrentProjectID {
		return // race: user navigated away
	}
	s.ProjectCycles = e.Cycles
	if s.Selected >= len(s.ProjectCycles) {
		s.Selected = 0
	}
	if e.FromCache {
		s.StatusText = fmt.Sprintf("Loaded %d cycles (cached, refreshing...)", len(e.Cycles))
		s.StatusKind = StatusInfo
	} else {
		s.StatusText = fmt.Sprintf("Loaded %d cycles", len(e.Cycles))
		s.StatusKind = StatusOk
	}
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

type FilterCountsLoaded struct {
	Counts map[string]int
	More   map[string]bool
}

func (e FilterCountsLoaded) apply(s *State) {
	s.FilterCounts = e.Counts
	s.FilterMore = e.More
}

type TeamMetadataLoaded struct{ Meta *linear.TeamMetadata }

func (e TeamMetadataLoaded) apply(s *State) {
	s.Meta = e.Meta
	if e.Meta != nil {
		s.Projects = e.Meta.Projects
		log.Printf("[App] TeamMetadataLoaded: %d cycles, %d states, %d projects", len(e.Meta.Cycles), len(e.Meta.States), len(e.Meta.Projects))
	}
	s.rebuildFilters()
	if s.Team != nil {
		snap := snapshotFilters(s, s.Filters)
		go fetchFilterCounts(s, s.Team.ID, snap)
	}
	// Forward to the active create/edit screen or open modal.
	if s.Create != nil {
		s.Create.metaReady(s)
	}
	if s.Edit != nil {
		s.Edit.metaReady(s)
	}
	if md, ok := s.ModalState.(*CreateModal); ok && s.Modal == ModalCreate {
		md.metaReady(s)
	}
}

type WorkflowStatesLoaded struct {
	TeamID    string
	States    []linear.WorkflowState
	FromCache bool
}

func (e WorkflowStatesLoaded) apply(s *State) {
	if s.Team != nil && e.TeamID != "" && e.TeamID != s.Team.ID {
		return // race: user switched teams
	}
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
		filter := buildIssueFilter(s.ActiveFilter, s)
		snap := snapshotFilters(s, s.Filters)
		go fetchIssues(s, s.Team.ID, s.ActiveFilter, filter, true)
		go fetchFilterCounts(s, s.Team.ID, snap)
	}
}

type IssueCreated struct{ Issue linear.Issue }

func (e IssueCreated) apply(s *State) {
	s.StatusText = "Created " + e.Issue.Identifier
	s.StatusKind = StatusOk
	if s.Team != nil {
		filter := buildIssueFilter(s.ActiveFilter, s)
		snap := snapshotFilters(s, s.Filters)
		go AutoLabel(s, []linear.Issue{e.Issue})
		go fetchIssues(s, s.Team.ID, s.ActiveFilter, filter, true)
		go fetchFilterCounts(s, s.Team.ID, snap)
	}
}

type MyIssuesLoaded struct{ Issues []linear.Issue }

func (e MyIssuesLoaded) apply(s *State) {
	if md, ok := s.ModalState.(*SearchModal); ok && s.Modal == ModalSearch {
		md.SetIssues(e.Issues)
	}
}

// IssueLookupResult carries the result of a direct issue-identifier lookup
// fired from the search modal (see SearchModal.checkIdentifierLookup). Issue
// is nil when the identifier wasn't found.
type IssueLookupResult struct {
	Identifier string
	Issue      *linear.Issue
	Err        error
}

func (e IssueLookupResult) apply(s *State) {
	md, ok := s.ModalState.(*SearchModal)
	if !ok || s.Modal != ModalSearch {
		return
	}
	if e.Identifier != md.LookupIdentifier {
		return // stale result for a query the user has since changed
	}
	md.LookupLoading = false
	if e.Err != nil || e.Issue == nil {
		md.LookupNotFound = true
		return
	}
	md.LookupNotFound = false
	for _, is := range md.Issues {
		if is.ID == e.Issue.ID {
			return // already present in the locally-loaded list
		}
	}
	md.Issues = append([]linear.Issue{*e.Issue}, md.Issues...)
	md.Clicks = make([]widget.Clickable, len(md.Issues))
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

// RefreshRequested asks the UI to refire the current-filter fetch on the UI
// goroutine. Used by background workers (e.g. AutoLabel) that finished a
// mutation and want the issue list to refresh — they cannot safely read
// s.Team / s.ActiveFilter themselves.
type RefreshRequested struct{}

func (RefreshRequested) apply(s *State) {
	if s.Team == nil {
		return
	}
	filter := buildIssueFilter(s.ActiveFilter, s)
	snap := snapshotFilters(s, s.Filters)
	go fetchIssues(s, s.Team.ID, s.ActiveFilter, filter, true)
	go fetchFilterCounts(s, s.Team.ID, snap)
}

// rebuildFilters recomputes the sidebar filter list from current user/projects.
func (s *State) rebuildFilters() {
	s.Filters = []string{
		"My Issues",
		"My Issues + Active",
		"My Unlabeled Issues",
	}
}

// snapshotFilters builds a name->filter map for every non-separator entry in
// filters. Must be called on the UI goroutine.
func snapshotFilters(s *State, filters []string) map[string]map[string]any {
	m := make(map[string]map[string]any, len(filters))
	for _, n := range filters {
		if n == "---" {
			continue
		}
		m[n] = buildIssueFilter(n, s)
	}
	return m
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
