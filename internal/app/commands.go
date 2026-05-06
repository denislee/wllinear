package app

import (
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/atotto/clipboard"

	"github.com/denislee/wllinear/internal/ai"
	"github.com/denislee/wllinear/internal/linear"
)

// post is shorthand to send an event back to the UI loop.
func post(s *State, e Event) {
	s.PostEvent(e)
}

func errStatus(err error) Event {
	return StatusMsg{Text: err.Error(), Kind: StatusErr}
}

func fetchViewer(s *State) {
	u, err := s.Client.GetViewer()
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, ViewerLoaded{User: *u})
}

func fetchTeams(s *State) {
	t, err := s.Client.GetTeams()
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, TeamsLoaded{Teams: t})
}

func fetchLeadingProjects(s *State) {
	if s.User == nil {
		return
	}
	p, err := s.Client.GetLeadingProjects(s.User.ID)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, LeadingProjectsLoaded{Projects: p})
}

func fetchProjectCycles(s *State, project linear.Project) {
	// 1. Try loading from local cache for immediate display.
	if cached, err := s.DB.GetProjectCycles(project.ID); err == nil && len(cached) > 0 {
		post(s, ProjectCyclesLoaded{ProjectID: project.ID, Cycles: cached, FromCache: true})
	}

	// 2. Fetch from network in the background.
	cycles, err := s.Client.GetProjectIssuesByCycles(project.ID)
	if err != nil {
		// Only surface error if we have nothing to show.
		if len(s.ProjectCycles) == 0 {
			post(s, errStatus(err))
		}
		return
	}

	// 3. Update cache and UI with fresh data.
	_ = s.DB.SaveProjectCycles(project.ID, cycles)
	post(s, ProjectCyclesLoaded{ProjectID: project.ID, Cycles: cycles})
}

func fetchIssues(s *State, teamID, filterName string) {
	// 1. Try loading from local cache for immediate display.
	if cached, err := s.DB.GetIssues(teamID, filterName); err == nil && len(cached) > 0 {
		post(s, IssuesLoaded{FilterName: filterName, Issues: cached})
	}

	// 2. Fetch from network in the background.
	filter := buildIssueFilter(filterName, s)
	conn, err := s.Client.GetIssues(teamID, 50, "", filter, false)
	if err != nil {
		// Only post error if we have no cached issues to show.
		if len(s.Issues) == 0 {
			post(s, errStatus(err))
		}
		return
	}

	// 3. Update cache and UI with fresh data.
	_ = s.DB.SaveIssues(teamID, filterName, conn.Nodes)
	post(s, IssuesLoaded{FilterName: filterName, Issues: conn.Nodes})
}

func fetchFilterCounts(s *State, teamID string, filters []string) {
	m := make(map[string]map[string]any)
	for _, n := range filters {
		if n == "---" {
			continue
		}
		m[n] = buildIssueFilter(n, s)
	}
	if len(m) == 0 {
		return
	}
	counts, err := s.Client.GetFilterCounts(teamID, m)
	if err != nil {
		// Don't surface — counts are decorative.
		return
	}
	post(s, FilterCountsLoaded{Counts: counts})
}

func fetchTeamMetadata(s *State, teamID string) {
	if s.Meta != nil && s.Team != nil && s.Team.ID == teamID {
		return
	}
	meta, err := s.Client.GetTeamMetadata(teamID)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, TeamMetadataLoaded{Meta: meta})
}

func fetchWorkflowStates(s *State, teamID string) {
	// 1. Try loading from local cache for immediate display.
	if cached, err := s.DB.GetWorkflowStates(teamID); err == nil && len(cached) > 0 {
		post(s, WorkflowStatesLoaded{TeamID: teamID, States: cached, FromCache: true})
	}

	// 2. Fetch from network in the background.
	states, err := s.Client.GetWorkflowStates(teamID)
	if err != nil {
		post(s, errStatus(err))
		return
	}

	// 3. Update cache and UI with fresh data.
	_ = s.DB.SaveWorkflowStates(teamID, states)
	post(s, WorkflowStatesLoaded{TeamID: teamID, States: states})
}

func fetchMyIssues(s *State) {
	if s.Team == nil || s.User == nil {
		return
	}
	filter := map[string]any{
		"assignee": map[string]any{"id": map[string]any{"eq": s.User.ID}},
	}
	conn, err := s.Client.GetIssues(s.Team.ID, 250, "", filter, true)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, MyIssuesLoaded{Issues: conn.Nodes})
}

func updateIssueStatus(s *State, issueID, stateID string) {
	updated, err := s.Client.UpdateIssue(issueID, linear.IssueUpdateInput{StateID: &stateID})
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, IssueUpdated{Issue: *updated})
}

func createIssue(s *State, in linear.IssueCreateInput) {
	issue, err := s.Client.CreateIssue(in)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, IssueCreated{Issue: *issue})
	_ = clipboard.WriteAll("chore(" + issue.Identifier + "): " + strings.ReplaceAll(issue.Title, ":", ","))
}

func editIssue(s *State, id string, in linear.IssueUpdateInput) {
	issue, err := s.Client.UpdateIssue(id, in)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, IssueUpdated{Issue: *issue})
}

// OpenBrowser opens a URL in the system default browser.
func OpenBrowser(url string) {
	if url == "" {
		return
	}
	var c *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		c = exec.Command("xdg-open", url)
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = c.Start()
}

// CopyCycleIssues copies the titles of the given cycle's issues as a markdown list.
// Only issues whose workflow state name is "Done" are included.
func CopyCycleIssues(s *State, c linear.ProjectCycleIssues) {
	name := c.Cycle.Name
	if name == "" {
		name = "Cycle " + intStr(c.Cycle.Number)
	}
	if len(c.Issues) == 0 {
		post(s, StatusMsg{Text: "No issues in " + name, Kind: StatusWarn})
		return
	}
	lines := make([]string, 0, len(c.Issues))
	for _, is := range c.Issues {
		if !strings.EqualFold(is.State.Name, "Done") {
			continue
		}
		lines = append(lines, "- "+is.Title)
	}
	if len(lines) == 0 {
		post(s, StatusMsg{Text: "No Done issues in " + name, Kind: StatusWarn})
		return
	}
	if err := clipboard.WriteAll(strings.Join(lines, "\n")); err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, StatusMsg{
		Text: "Copied " + intStr(len(lines)) + " issue titles to clipboard",
		Kind: StatusOk,
	})
}

// CopyProjectLastCycle copies titles of last-cycle issues to the clipboard.
func CopyProjectLastCycle(s *State, project linear.Project) {
	titles, err := s.Client.GetProjectIssuesFromLastCycle(project.ID)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	if len(titles) == 0 {
		post(s, StatusMsg{Text: "No completed issues in last cycle of " + project.Name, Kind: StatusWarn})
		return
	}
	for i, t := range titles {
		titles[i] = "- " + t
	}
	if err := clipboard.WriteAll(strings.Join(titles, "\n")); err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, StatusMsg{
		Text: "Copied " + intStr(len(titles)) + " issues from " + project.Name,
		Kind: StatusOk,
	})
}

// CopyIssue copies the issue's identifier and title to the clipboard using
// the standard commit-message style template.
func CopyIssue(s *State, issue linear.Issue) {
	text := "chore(" + issue.Identifier + "): " + strings.ReplaceAll(issue.Title, ":", ",")
	if err := clipboard.WriteAll(text); err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, StatusMsg{
		Text: "Copied: " + text,
		Kind: StatusOk,
	})
}

// AutoLabel runs the Gemini-based auto-labeling flow on the given issues.
// All work happens in a single goroutine; progress is reported via StatusMsg.
func AutoLabel(s *State, issues []linear.Issue) {
	if len(issues) == 0 || s.Team == nil {
		return
	}
	teamID := s.Team.ID
	post(s, StatusMsg{Text: "Auto-labeling: fetching label set...", Kind: StatusInfo})

	all, err := s.Client.GetAllIssueLabels()
	if err != nil {
		post(s, errStatus(err))
		return
	}
	teamLabels := map[string]string{}
	orgLabels := map[string]string{}
	nameSet := map[string]struct{}{}
	for _, l := range all {
		if l.IsGroup {
			continue
		}
		if l.TeamID == "" {
			orgLabels[l.Name] = l.ID
		} else if l.TeamID == teamID {
			teamLabels[l.Name] = l.ID
		}
		nameSet[l.Name] = struct{}{}
	}
	allowed := make([]string, 0, len(nameSet))
	for n := range nameSet {
		allowed = append(allowed, n)
	}
	sort.Strings(allowed)
	if len(allowed) == 0 {
		post(s, StatusMsg{Text: "No labels found in workspace", Kind: StatusWarn})
		return
	}

	inputs := make([]ai.IssueInput, 0, len(issues))
	for _, i := range issues {
		inputs = append(inputs, ai.IssueInput{
			Identifier:  i.Identifier,
			Title:       i.Title,
			Description: i.Description,
		})
	}

	post(s, StatusMsg{Text: "Auto-labeling: querying Gemini...", Kind: StatusInfo})
	suggestions, err := ai.NewGeminiClient().CategorizeIssues(inputs, allowed)
	if err != nil {
		post(s, errStatus(err))
		return
	}

	for i, issue := range issues {
		suggested := suggestions[issue.Identifier]
		if suggested == "" {
			post(s, StatusMsg{
				Text: "Auto-label [" + intStr(i+1) + "/" + intStr(len(issues)) + "] " + issue.Identifier + ": skipped",
				Kind: StatusWarn,
			})
			continue
		}
		labelID := teamLabels[suggested]
		if labelID == "" {
			labelID = orgLabels[suggested]
		}
		if labelID == "" {
			id, err := s.Client.CreateLabel(suggested, teamID)
			if err != nil {
				post(s, errStatus(err))
				continue
			}
			teamLabels[suggested] = id
			labelID = id
		}
		if err := s.Client.UpdateIssueLabels(issue.ID, []string{labelID}); err != nil {
			post(s, errStatus(err))
			continue
		}
		post(s, StatusMsg{
			Text: "Auto-label [" + intStr(i+1) + "/" + intStr(len(issues)) + "] " + issue.Identifier + " → " + suggested,
			Kind: StatusOk,
		})
	}

	if s.Team != nil {
		fetchIssues(s, s.Team.ID, s.ActiveFilter)
	}
	post(s, StatusMsg{Text: "Auto-labeling complete", Kind: StatusOk})
}

func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
