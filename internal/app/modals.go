package app

import (
	"log"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/denislee/wllinear/internal/linear"
	"github.com/denislee/wllinear/internal/ui"
)

// CreateModal holds the state of the create-issue overlay.
type CreateModal struct {
	List        widget.List
	Title       widget.Editor
	Description widget.Editor

	// Selected indices into Meta lists (-1 = none).
	StateIdx    int
	AssigneeIdx int
	ProjectIdx  int
	CycleIdx    int
	Priority    int // 0..4

	FocusIdx int  // 0: Title, 1: Desc, 2: Prio, 3: Status, 4: Project, 5: Cycle, 6: Submit, 7: Assignee
	FocusSet bool // Tracks if initial focus has been set
	FocusReq bool // Set to true when we programmatically want to push focus

	StatusExpanded   bool
	ProjectExpanded  bool
	CycleExpanded    bool
	AssigneeExpanded bool
	StatusToggle     widget.Clickable
	ProjectToggle    widget.Clickable
	CycleToggle      widget.Clickable
	AssigneeToggle   widget.Clickable

	Submit widget.Clickable
	Cancel widget.Clickable
	// Click-targets for selection rows.
	StateClicks    []widget.Clickable
	AssigneeClicks []widget.Clickable
	ProjectClicks  []widget.Clickable
	CycleClicks    []widget.Clickable
	PrioClicks     [5]widget.Clickable

	Meta *linear.TeamMetadata
}

func NewCreateModal(s *State) *CreateModal {
	m := &CreateModal{
		StateIdx: -1, AssigneeIdx: -1, ProjectIdx: -1, CycleIdx: -1,
	}
	m.Title.SingleLine = true
	if s.User != nil && s.Meta != nil {
		for i, u := range s.Meta.Members {
			if u.ID == s.User.ID {
				m.AssigneeIdx = i
				break
			}
		}
	}
	if s.Meta != nil {
		m.metaReady(s)
	}
	return m
}

func (m *CreateModal) metaReady(s *State) {
	m.Meta = s.Meta
	if m.Meta == nil {
		return
	}
	log.Printf("[App] CreateModal metaReady: %d cycles", len(m.Meta.Cycles))
	if len(m.StateClicks) != len(m.Meta.States) {
		m.StateClicks = make([]widget.Clickable, len(m.Meta.States))
	}
	if len(m.AssigneeClicks) != len(m.Meta.Members) {
		m.AssigneeClicks = make([]widget.Clickable, len(m.Meta.Members))
	}
	if len(m.ProjectClicks) != len(m.Meta.Projects) {
		m.ProjectClicks = make([]widget.Clickable, len(m.Meta.Projects))
	}
	if len(m.CycleClicks) != len(m.Meta.Cycles) {
		m.CycleClicks = make([]widget.Clickable, len(m.Meta.Cycles))
	}
	if m.AssigneeIdx == -1 && s.User != nil {
		for i, u := range m.Meta.Members {
			if u.ID == s.User.ID {
				m.AssigneeIdx = i
				break
			}
		}
	}
	if m.StateIdx == -1 {
		preferred := "started"
		if s.Saved != nil && s.Saved.DefaultCreateStatusType != "" {
			preferred = s.Saved.DefaultCreateStatusType
		}
		for i, st := range m.Meta.States {
			if st.Type == preferred {
				m.StateIdx = i
				break
			}
		}
		// Fall back to unstarted/backlog, then first state.
		if m.StateIdx == -1 {
			for i, st := range m.Meta.States {
				if st.Type == "unstarted" || st.Type == "backlog" {
					m.StateIdx = i
					break
				}
			}
		}
		if m.StateIdx == -1 && len(m.Meta.States) > 0 {
			m.StateIdx = 0
		}
	}
	if m.CycleIdx == -1 {
		now := time.Now()
		for i, c := range m.Meta.Cycles {
			// Skip completed cycles.
			if c.CompletedAt != nil && !c.CompletedAt.IsZero() {
				continue
			}
			// Active cycle check.
			if (c.StartsAt.Before(now) || c.StartsAt.Equal(now)) && (c.EndsAt.After(now) || c.EndsAt.Equal(now)) {
				m.CycleIdx = i
				break
			}
		}
	}
}

func (m *CreateModal) Build(s *State) (linear.IssueCreateInput, bool) {
	if s.Team == nil {
		return linear.IssueCreateInput{}, false
	}
	title := strings.TrimSpace(m.Title.Text())
	if title == "" {
		return linear.IssueCreateInput{}, false
	}
	in := linear.IssueCreateInput{
		TeamID: s.Team.ID,
		Title:  title,
	}
	if d := strings.TrimSpace(m.Description.Text()); d != "" {
		in.Description = &d
	}
	if m.Priority > 0 {
		p := m.Priority
		in.Priority = &p
	}
	if m.Meta != nil {
		if m.AssigneeIdx >= 0 && m.AssigneeIdx < len(m.Meta.Members) {
			id := m.Meta.Members[m.AssigneeIdx].ID
			in.AssigneeID = &id
		}
		if m.StateIdx >= 0 && m.StateIdx < len(m.Meta.States) {
			id := m.Meta.States[m.StateIdx].ID
			in.StateID = &id
		}
		if m.ProjectIdx >= 0 && m.ProjectIdx < len(s.LeadingProjects) {
			id := s.LeadingProjects[m.ProjectIdx].ID
			in.ProjectID = &id
		}
		if m.CycleIdx >= 0 && m.CycleIdx < len(m.Meta.Cycles) {
			id := m.Meta.Cycles[m.CycleIdx].ID
			in.CycleID = &id
		}
	}
	return in, true
}

// EditModal mirrors CreateModal but for an existing issue.
type EditModal struct {
	List        widget.List
	Issue       linear.Issue
	Title       widget.Editor
	Description widget.Editor

	// Selected indices into Meta lists (-1 = none).
	StateIdx    int
	AssigneeIdx int
	ProjectIdx  int
	CycleIdx    int
	Priority    int // 0..4

	FocusIdx int  // 0: Title, 1: Desc, 2: Prio, 3: Status, 4: Project, 5: Cycle, 6: Submit, 7: Assignee
	FocusSet bool // Tracks if initial focus has been set
	FocusReq bool // Set to true when we programmatically want to push focus

	StatusExpanded   bool
	ProjectExpanded  bool
	CycleExpanded    bool
	AssigneeExpanded bool
	StatusToggle     widget.Clickable
	ProjectToggle    widget.Clickable
	CycleToggle      widget.Clickable
	AssigneeToggle   widget.Clickable

	Submit widget.Clickable
	Cancel widget.Clickable
	// Click-targets for selection rows.
	StateClicks    []widget.Clickable
	AssigneeClicks []widget.Clickable
	ProjectClicks  []widget.Clickable
	CycleClicks    []widget.Clickable
	PrioClicks     [5]widget.Clickable

	Meta *linear.TeamMetadata
}

func NewEditModal(s *State, issue linear.Issue) *EditModal {
	m := &EditModal{
		Issue:       issue,
		StateIdx:    -1,
		AssigneeIdx: -1,
		ProjectIdx:  -1,
		CycleIdx:    -1,
		Priority:    issue.Priority,
	}
	m.Title.SingleLine = true
	m.Title.SetText(issue.Title)
	m.Description.SetText(issue.Description)
	if s.Meta != nil {
		m.metaReady(s)
	}
	return m
}

func (m *EditModal) metaReady(s *State) {
	m.Meta = s.Meta
	if m.Meta == nil {
		return
	}
	if len(m.StateClicks) != len(m.Meta.States) {
		m.StateClicks = make([]widget.Clickable, len(m.Meta.States))
	}
	if len(m.AssigneeClicks) != len(m.Meta.Members) {
		m.AssigneeClicks = make([]widget.Clickable, len(m.Meta.Members))
	}
	if len(m.ProjectClicks) != len(s.LeadingProjects) {
		m.ProjectClicks = make([]widget.Clickable, len(s.LeadingProjects))
	}
	if len(m.CycleClicks) != len(m.Meta.Cycles) {
		m.CycleClicks = make([]widget.Clickable, len(m.Meta.Cycles))
	}

	for i, st := range m.Meta.States {
		if st.ID == m.Issue.State.ID {
			m.StateIdx = i
			break
		}
	}
	if m.Issue.Assignee != nil {
		for i, u := range m.Meta.Members {
			if u.ID == m.Issue.Assignee.ID {
				m.AssigneeIdx = i
				break
			}
		}
	}
	if m.Issue.Project != nil {
		for i, p := range s.LeadingProjects {
			if p.ID == m.Issue.Project.ID {
				m.ProjectIdx = i
				break
			}
		}
	}
	if m.Issue.Cycle != nil {
		for i, c := range m.Meta.Cycles {
			if c.ID == m.Issue.Cycle.ID {
				m.CycleIdx = i
				break
			}
		}
	}
}

func (m *EditModal) Build(s *State) (string, linear.IssueUpdateInput, bool) {
	in := linear.IssueUpdateInput{}
	title := strings.TrimSpace(m.Title.Text())
	if title == "" {
		return "", in, false
	}
	t := title
	in.Title = &t
	d := m.Description.Text()
	in.Description = &d
	p := m.Priority
	in.Priority = &p
	if m.Meta != nil {
		if m.StateIdx >= 0 && m.StateIdx < len(m.Meta.States) {
			id := m.Meta.States[m.StateIdx].ID
			in.StateID = &id
		}
		if m.AssigneeIdx >= 0 && m.AssigneeIdx < len(m.Meta.Members) {
			id := m.Meta.Members[m.AssigneeIdx].ID
			in.AssigneeID = &id
		}
		if m.ProjectIdx >= 0 && m.ProjectIdx < len(s.LeadingProjects) {
			id := s.LeadingProjects[m.ProjectIdx].ID
			in.ProjectID = &id
		}
		if m.CycleIdx >= 0 && m.CycleIdx < len(m.Meta.Cycles) {
			id := m.Meta.Cycles[m.CycleIdx].ID
			in.CycleID = &id
		}
	}
	return m.Issue.ID, in, true
}

// StatusModal is the lightweight workflow-state picker.
type StatusModal struct {
	Issue   linear.Issue
	States  []linear.WorkflowState
	Idx     int
	Cancel  widget.Clickable
	Confirm widget.Clickable
	Clicks  []widget.Clickable
	List    widget.List
}

func NewStatusModal(issue linear.Issue) *StatusModal {
	m := &StatusModal{Issue: issue, Idx: -1}
	m.List.Axis = layout.Vertical
	return m
}

func (m *StatusModal) SetStates(states []linear.WorkflowState) {
	// Preserve the user's in-progress selection across a cache→network refresh.
	var selectedID string
	if m.Idx >= 0 && m.Idx < len(m.States) {
		selectedID = m.States[m.Idx].ID
	}
	m.States = states
	m.Clicks = make([]widget.Clickable, len(states))
	m.Idx = -1
	target := selectedID
	if target == "" {
		target = m.Issue.State.ID
	}
	for i, st := range states {
		if st.ID == target {
			m.Idx = i
			break
		}
	}
}

// SearchModal is the issue search overlay (Ctrl+K).
type SearchModal struct {
	Query    widget.Editor
	Issues   []linear.Issue
	Selected int
	Cancel   widget.Clickable
	Clicks   []widget.Clickable
	FocusSet bool
}

func NewSearchModal() *SearchModal {
	m := &SearchModal{Selected: 0}
	m.Query.SingleLine = true
	return m
}

func (m *SearchModal) SetIssues(issues []linear.Issue) {
	m.Issues = issues
	m.Clicks = make([]widget.Clickable, len(issues))
}

// SettingsModal is the per-section font/size configuration overlay.
type SettingsModal struct {
	List  widget.List
	Close widget.Clickable
	Reset widget.Clickable
	Rows  []*SettingsRow

	// Default-create-status cycler.
	StatusPrev widget.Clickable
	StatusNext widget.Clickable

	LogToggle   widget.Bool
	HintsToggle widget.Bool
}

// CreateStatusTypes are the workflow state types selectable as the default
// status when creating a new issue.
var CreateStatusTypes = []string{"started", "unstarted", "backlog"}

func createStatusLabel(t string) string {
	switch t {
	case "started":
		return "In Progress"
	case "unstarted":
		return "Todo"
	case "backlog":
		return "Backlog"
	}
	return t
}

// SettingsRow is one configurable section: a label + face/size click targets.
type SettingsRow struct {
	Label   string
	Target  *ui.FontStyle
	Mono    bool
	PrevF   widget.Clickable
	NextF   widget.Clickable
	Smaller widget.Clickable
	Bigger  widget.Clickable
	Reset   widget.Clickable
}

func NewSettingsModal(th *ui.Theme, logging, hideHints bool) *SettingsModal {
	m := &SettingsModal{}
	m.LogToggle.Value = logging
	m.HintsToggle.Value = !hideHints
	m.List.Axis = layout.Vertical
	m.Rows = []*SettingsRow{
		{Label: "Global (base)", Target: &th.Fonts.Global},
		{Label: "Sidebar", Target: &th.Fonts.Sidebar},
		{Label: "Issue List", Target: &th.Fonts.IssueList},
		{Label: "Issue Detail", Target: &th.Fonts.IssueDetail},
		{Label: "Create Issue", Target: &th.Fonts.CreateIssue},
		{Label: "Status Bar", Target: &th.Fonts.StatusBar},
		{Label: "Modals", Target: &th.Fonts.Modal},
		{Label: "Code (mono)", Target: &th.Fonts.Code, Mono: true},
	}
	return m
}

// Filter returns issues matching the query (substring, case-insensitive).
func (m *SearchModal) Filter() []linear.Issue {
	q := strings.ToLower(strings.TrimSpace(m.Query.Text()))
	if q == "" {
		return m.Issues
	}
	out := make([]linear.Issue, 0, len(m.Issues))
	for _, is := range m.Issues {
		if strings.Contains(strings.ToLower(is.Identifier), q) ||
			strings.Contains(strings.ToLower(is.Title), q) {
			out = append(out, is)
		}
	}
	return out
}

// TeamModal is the team selection overlay (Ctrl+T).
type TeamModal struct {
	Teams    []linear.Team
	Selected int
	Cancel   widget.Clickable
	Clicks   []widget.Clickable
}

func NewTeamModal() *TeamModal {
	m := &TeamModal{Selected: 0}
	return m
}

func (m *TeamModal) SetTeams(teams []linear.Team) {
	m.Teams = teams
	m.Clicks = make([]widget.Clickable, len(teams))
}
