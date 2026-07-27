package app

import (
	"log"
	"regexp"
	"strconv"
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

	FocusIdx     int  // 0: Title, 1: Desc, 2: Prio, 3: Status, 4: Assignee, 5: Project, 6: Cycle, 7: Submit
	FocusSet     bool // Tracks if initial focus has been set
	FocusReq     bool // Set to true when we programmatically want to push focus
	LastFocusIdx int  // Tracks last focus index applied to scroll-into-view logic

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
	m.applyDefaultStatus(s)
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

// applyDefaultStatus sets m.StateIdx based on the user's preferred default
// workflow-state type (e.g. "started" → "In Progress"). Safe to call
// repeatedly: it only runs while StateIdx is still -1, so a user-picked
// status is preserved. Called both from metaReady (when team metadata
// arrives) and from the create-issue layout (as a safety net for any
// timing race between modal-open and metadata-load).
func (m *CreateModal) applyDefaultStatus(s *State) {
	if m == nil || m.Meta == nil || len(m.Meta.States) == 0 {
		return
	}
	if m.StateIdx != -1 {
		return
	}
	preferred := "started"
	if s != nil && s.Saved != nil && s.Saved.DefaultCreateStatusType != "" {
		preferred = s.Saved.DefaultCreateStatusType
	}
	// 1) Match by display name first. Teams commonly have multiple states
	//    that share a workflow-state type (e.g. "In Progress", "In Review",
	//    "Blocked" all have type=="started"), so picking by type alone is
	//    ambiguous. Name match — e.g. "In Progress" for the "started"
	//    preference — picks the canonical one when present.
	wantName := createStatusLabel(preferred)
	for i, st := range m.Meta.States {
		if strings.EqualFold(st.Name, wantName) {
			m.StateIdx = i
			return
		}
	}
	// 2) Fall back to matching by workflow-state type. Picks the first
	//    state in that bucket — fine when there's only one, the best we
	//    can do without further guidance when there are several.
	for i, st := range m.Meta.States {
		if st.Type == preferred {
			m.StateIdx = i
			return
		}
	}
	// 3) Fall back to a sensible non-completed bucket, then first state.
	for i, st := range m.Meta.States {
		if st.Type == "unstarted" || st.Type == "backlog" {
			m.StateIdx = i
			return
		}
	}
	m.StateIdx = 0
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
	LabelIdx    int // -1 = none
	Priority    int // 0..4

	FocusIdx int  // 0: Title, 1: Desc, 2: Prio, 3: Status, 4: Assignee, 5: Label, 6: Project, 7: Cycle, 8: Submit
	FocusSet bool // Tracks if initial focus has been set
	FocusReq bool // Set to true when we programmatically want to push focus

	StatusExpanded   bool
	ProjectExpanded  bool
	CycleExpanded    bool
	AssigneeExpanded bool
	LabelExpanded    bool
	StatusToggle     widget.Clickable
	ProjectToggle    widget.Clickable
	CycleToggle      widget.Clickable
	AssigneeToggle   widget.Clickable
	LabelToggle      widget.Clickable

	Submit widget.Clickable
	Cancel widget.Clickable
	// Click-targets for selection rows.
	StateClicks    []widget.Clickable
	AssigneeClicks []widget.Clickable
	ProjectClicks  []widget.Clickable
	CycleClicks    []widget.Clickable
	LabelClicks    []widget.Clickable
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
		LabelIdx:    -1,
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
	if len(m.LabelClicks) != len(m.Meta.Labels) {
		m.LabelClicks = make([]widget.Clickable, len(m.Meta.Labels))
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
	if len(m.Issue.Labels.Nodes) > 0 {
		for i, l := range m.Meta.Labels {
			if l.ID == m.Issue.Labels.Nodes[0].ID {
				m.LabelIdx = i
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
		if m.LabelIdx >= 0 && m.LabelIdx < len(m.Meta.Labels) {
			in.LabelIDs = []string{m.Meta.Labels[m.LabelIdx].ID}
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

	// lastQuery is the query text as of the last identifier-lookup check, so
	// checkIdentifierLookup only reacts to actual edits.
	lastQuery string
	// LookupIdentifier is the identifier-shaped query (e.g. "TECH-12762") a
	// direct API lookup has been requested for, since the locally-loaded
	// Issues list only contains a recent subset assigned to the current user
	// in the current team.
	LookupIdentifier string
	// LookupLoading is true while a direct identifier lookup is in flight.
	LookupLoading bool
	// LookupNotFound is true when the last completed lookup found nothing.
	LookupNotFound bool
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

// issueIdentifierPattern matches a full Linear issue identifier such as
// "TECH-12762".
var issueIdentifierPattern = regexp.MustCompile(`^[A-Za-z]+-\d+$`)

// checkIdentifierLookup fires a direct API lookup when the query text looks
// like a full issue code (e.g. "TECH-12762") and no locally-loaded issue
// matches it. Issues only holds a recent subset of issues assigned to the
// current user in the current team, so this is a fallback to find issues
// outside that set. No-op when the query text hasn't changed since the last
// check.
func (m *SearchModal) checkIdentifierLookup(s *State) {
	text := m.Query.Text()
	if text == m.lastQuery {
		return
	}
	m.lastQuery = text

	q := strings.ToUpper(strings.TrimSpace(text))
	if !issueIdentifierPattern.MatchString(q) {
		m.LookupIdentifier = ""
		m.LookupLoading = false
		m.LookupNotFound = false
		return
	}
	if q == m.LookupIdentifier {
		return
	}
	if len(m.Filter()) > 0 {
		return
	}

	m.LookupIdentifier = q
	m.LookupLoading = true
	m.LookupNotFound = false
	go lookupIssueByIdentifier(s, q)
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

// CommentModal is the new-comment overlay (N on issue detail).
type CommentModal struct {
	Issue    linear.Issue
	Body     widget.Editor
	Submit   widget.Clickable
	Cancel   widget.Clickable
	FocusSet bool
}

func NewCommentModal(issue linear.Issue) *CommentModal {
	return &CommentModal{Issue: issue}
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

// ProjectInfoRow is one label/value pair in the read-only project overlay.
type ProjectInfoRow struct {
	Label string
	Value string
}

// ProjectInfoModal is the read-only project information overlay (Ctrl+I). It
// opens immediately with the few fields already known from the sidebar, then
// swaps in the full field set once the detail fetch completes.
type ProjectInfoModal struct {
	ProjectID string
	Name      string
	Loading   bool
	Rows      []ProjectInfoRow
	Selected  int
	List      widget.List
}

func NewProjectInfoModal(p linear.Project) *ProjectInfoModal {
	m := &ProjectInfoModal{
		ProjectID: p.ID,
		Name:      cleanProjectName(p.Name),
		Loading:   true,
		Rows:      projectBasicRows(p),
	}
	m.List.Axis = layout.Vertical
	return m
}

// SetDetail replaces the modal's rows with the full project detail, preserving
// the current selection where possible.
func (m *ProjectInfoModal) SetDetail(d linear.ProjectDetail) {
	m.Loading = false
	if n := cleanProjectName(d.Name); n != "" {
		m.Name = n
	}
	m.Rows = projectDetailRows(d)
	if m.Selected >= len(m.Rows) {
		m.Selected = 0
	}
}

// projectBasicRows builds the placeholder rows shown before the detail fetch
// returns, from the fields already available on the sidebar Project.
func projectBasicRows(p linear.Project) []ProjectInfoRow {
	rows := []ProjectInfoRow{{Label: "Name", Value: cleanProjectName(p.Name)}}
	if p.Status.Name != "" {
		rows = append(rows, ProjectInfoRow{Label: "Status", Value: p.Status.Name})
	}
	if p.Lead != nil && p.Lead.Name != "" {
		rows = append(rows, ProjectInfoRow{Label: "Lead", Value: p.Lead.Name})
	}
	return rows
}

// projectDetailRows flattens a ProjectDetail into ordered label/value rows,
// skipping any field that is empty.
func projectDetailRows(d linear.ProjectDetail) []ProjectInfoRow {
	rows := make([]ProjectInfoRow, 0, 18)
	add := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		rows = append(rows, ProjectInfoRow{Label: label, Value: value})
	}
	add("Name", cleanProjectName(d.Name))
	status := d.Status.Name
	if status == "" {
		status = d.State
	}
	add("Status", status)
	add("Priority", d.PriorityLabel)
	if d.Progress > 0 {
		add("Progress", strconv.Itoa(int(d.Progress*100+0.5))+"%")
	}
	if d.Lead != nil {
		lead := d.Lead.Name
		if d.Lead.Email != "" {
			lead += " <" + d.Lead.Email + ">"
		}
		add("Lead", lead)
	}
	if len(d.Members.Nodes) > 0 {
		names := make([]string, 0, len(d.Members.Nodes))
		for _, u := range d.Members.Nodes {
			names = append(names, u.Name)
		}
		add("Members", strings.Join(names, ", "))
	}
	add("Start date", d.StartDate)
	add("Target date", d.TargetDate)
	if d.StartedAt != nil && !d.StartedAt.IsZero() {
		add("Started", d.StartedAt.Format("2006-01-02"))
	}
	if d.CompletedAt != nil && !d.CompletedAt.IsZero() {
		add("Completed", d.CompletedAt.Format("2006-01-02"))
	}
	if d.CanceledAt != nil && !d.CanceledAt.IsZero() {
		add("Canceled", d.CanceledAt.Format("2006-01-02"))
	}
	if !d.CreatedAt.IsZero() {
		add("Created", d.CreatedAt.Format("2006-01-02"))
	}
	if !d.UpdatedAt.IsZero() {
		add("Updated", d.UpdatedAt.Format("2006-01-02"))
	}
	add("Description", strings.TrimSpace(d.Description))
	add("Content", strings.TrimSpace(d.Content))
	add("URL", d.URL)
	add("ID", d.ID)
	return rows
}
