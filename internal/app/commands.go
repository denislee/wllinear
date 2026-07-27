package app

import (
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"

	"github.com/denislee/wllinear/internal/ai"
	"github.com/denislee/wllinear/internal/linear"
)

// cacheFreshness is the stale-while-revalidate window. A cache hit younger
// than this skips the background refresh; older entries trigger a refresh.
const cacheFreshness = 30 * time.Second

// post is shorthand to send an event back to the UI loop.
func post(s *State, e Event) {
	s.PostEvent(e)
}

func errStatus(err error) Event {
	return StatusMsg{Text: err.Error(), Kind: StatusErr}
}

func fetchViewer(s *State) {
	hadCache := false
	if cached, err := s.DB.GetViewer(); err == nil && cached != nil {
		hadCache = true
		post(s, ViewerLoaded{User: *cached})
	}
	u, err := s.Client.GetViewer()
	if err != nil {
		if !hadCache {
			post(s, errStatus(err))
		}
		return
	}
	_ = s.DB.SaveViewer(*u)
	post(s, ViewerLoaded{User: *u})
}

func fetchTeams(s *State) {
	hadCache := false
	if cached, err := s.DB.GetTeams(); err == nil && len(cached) > 0 {
		hadCache = true
		post(s, TeamsLoaded{Teams: cached})
	}
	t, err := s.Client.GetTeams()
	if err != nil {
		if !hadCache {
			post(s, errStatus(err))
		}
		return
	}
	_ = s.DB.SaveTeams(t)
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

func fetchProjectDetail(s *State, projectID string) {
	d, err := s.Client.GetProject(projectID)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, ProjectDetailLoaded{Detail: *d})
}

// CopyProjectInfoRow copies the value of the project overlay's selected row.
func CopyProjectInfoRow(s *State, m *ProjectInfoModal) {
	if m == nil || m.Selected < 0 || m.Selected >= len(m.Rows) {
		return
	}
	row := m.Rows[m.Selected]
	if err := clipboard.WriteAll(row.Value); err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, StatusMsg{Text: "Copied " + row.Label + ": " + truncate(row.Value, 60), Kind: StatusOk})
}

func fetchProjectCycles(s *State, project linear.Project, force bool) {
	// 1. Try loading from local cache for immediate display. Skip the network
	//    fetch entirely if the cache is still fresh and the caller didn't force.
	hadCache := false
	if cached, ts, err := s.DB.GetProjectCyclesWithTime(project.ID); err == nil && len(cached) > 0 {
		hadCache = true
		post(s, ProjectCyclesLoaded{ProjectID: project.ID, Cycles: cached, FromCache: true})
		if !force && time.Since(ts) < cacheFreshness {
			return
		}
	}

	// 2. Fetch from network in the background.
	cycles, err := s.Client.GetProjectIssuesByCycles(project.ID)
	if err != nil {
		if !hadCache {
			post(s, errStatus(err))
		}
		return
	}

	// 3. Update cache and UI with fresh data.
	_ = s.DB.SaveProjectCycles(project.ID, cycles)
	post(s, ProjectCyclesLoaded{ProjectID: project.ID, Cycles: cycles})
}

// fetchIssues runs on a worker goroutine. The filter must be pre-built on the
// UI goroutine (see buildIssueFilter) so no shared state is read here. When
// force is false, a still-fresh cache hit skips the network round-trip.
func fetchIssues(s *State, teamID, filterName string, filter map[string]any, force bool) {
	// 1. Try loading from local cache for immediate display.
	hadCache := false
	if cached, ts, err := s.DB.GetIssuesWithTime(teamID, filterName); err == nil && len(cached) > 0 {
		hadCache = true
		post(s, IssuesLoaded{FilterName: filterName, Issues: cached})
		if !force && time.Since(ts) < cacheFreshness {
			return
		}
	}

	// 2. Fetch from network in the background.
	conn, err := s.Client.GetIssues(teamID, 50, "", filter, false)
	if err != nil {
		if !hadCache {
			post(s, errStatus(err))
		}
		return
	}

	// 3. Update cache and UI with fresh data.
	_ = s.DB.SaveIssues(teamID, filterName, conn.Nodes)
	post(s, IssuesLoaded{FilterName: filterName, Issues: conn.Nodes})
}

// fetchFilterCounts runs on a worker goroutine. Filters must be pre-built
// on the UI goroutine (see snapshotFilters).
func fetchFilterCounts(s *State, teamID string, filters map[string]map[string]any) {
	if len(filters) == 0 {
		return
	}
	counts, more, err := s.Client.GetFilterCounts(teamID, filters)
	if err != nil {
		// Don't surface — counts are decorative.
		return
	}
	post(s, FilterCountsLoaded{Counts: counts, More: more})
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

	// 1. Try loading from local cache for immediate display.
	const cacheKey = "__search_my_issues__"
	if cached, err := s.DB.GetIssues(s.Team.ID, cacheKey); err == nil && len(cached) > 0 {
		post(s, MyIssuesLoaded{Issues: cached})
	}

	// 2. Fetch from network in the background.
	filter := map[string]any{
		"assignee": map[string]any{"id": map[string]any{"eq": s.User.ID}},
	}
	conn, err := s.Client.GetIssues(s.Team.ID, 250, "", filter, true)
	if err != nil {
		post(s, errStatus(err))
		return
	}

	// 3. Update cache and UI with fresh data.
	_ = s.DB.SaveIssues(s.Team.ID, cacheKey, conn.Nodes)
	post(s, MyIssuesLoaded{Issues: conn.Nodes})
}

// lookupIssueByIdentifier fetches a single issue directly by its
// human-readable identifier (e.g. "TECH-123"), used as a fallback in the
// search modal when the identifier isn't among the locally-loaded "my
// issues". Linear's issue query accepts either the identifier or the UUID
// and isn't scoped to a team, so this can find any issue the current API
// key can access.
func lookupIssueByIdentifier(s *State, identifier string) {
	issue, err := s.Client.GetIssue(identifier)
	if err != nil {
		post(s, IssueLookupResult{Identifier: identifier, Err: err})
		return
	}
	post(s, IssueLookupResult{Identifier: identifier, Issue: issue})
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

func createComment(s *State, issueID, identifier, body string) {
	if err := s.Client.CreateComment(issueID, body); err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, StatusMsg{Text: "Posted update on " + identifier, Kind: StatusOk})
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
		name = "Cycle " + strconv.Itoa(c.Cycle.Number)
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
		Text: "Copied " + strconv.Itoa(len(lines)) + " issue titles to clipboard",
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
		Text: "Copied " + strconv.Itoa(len(titles)) + " issues from " + project.Name,
		Kind: StatusOk,
	})
}

// CopyWeeklyReportPrompt fetches the user's completed issues from the team's
// most recently ended cycle and copies a structured LLM prompt to the
// clipboard. The prompt asks an LLM to turn the issue list into a concise
// Slack weekly status update.
func CopyWeeklyReportPrompt(s *State) {
	if s.Team == nil || s.User == nil {
		post(s, StatusMsg{Text: "Select a team first", Kind: StatusWarn})
		return
	}
	post(s, StatusMsg{Text: "Building weekly report...", Kind: StatusInfo})

	report, err := s.Client.GetUserLastCycleCompletedIssues(s.Team.ID, s.User.ID)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	if len(report.Issues) == 0 {
		post(s, StatusMsg{Text: "No completed issues in last cycle", Kind: StatusWarn})
		return
	}

	cycleLabel := report.Cycle.Name
	if cycleLabel == "" {
		cycleLabel = "Cycle " + strconv.Itoa(report.Cycle.Number)
	}
	dateRange := report.Cycle.StartsAt.Format("2006-01-02") + " → " + report.Cycle.EndsAt.Format("2006-01-02")

	var b strings.Builder
	b.WriteString("You are writing a concise weekly status update for Slack.\n")
	b.WriteString("Summarize the work I completed in the cycle below into a short, professional Slack message.\n")
	b.WriteString("Output the result in Markdown format only — no preamble, no surrounding code fences, no commentary.\n")
	b.WriteString("Guidelines:\n")
	b.WriteString("- Lead with a one-line summary of the week's focus.\n")
	b.WriteString("- Group related work; do not just list every issue verbatim.\n")
	b.WriteString("- Use Markdown: `#`/`##` for headings, `-` for bullets, `**bold**` for emphasis.\n")
	b.WriteString("- Mention the cycle name and date range once at the top.\n")
	b.WriteString("- Keep it under ~200 words.\n")
	b.WriteString("- Do not invent details that are not in the data.\n\n")

	b.WriteString("Author: " + s.User.Name + " <" + s.User.Email + ">\n")
	b.WriteString("Team: " + s.Team.Name + "\n")
	b.WriteString("Cycle: " + cycleLabel + " (" + dateRange + ")\n")
	b.WriteString("Completed issues: " + strconv.Itoa(len(report.Issues)) + "\n\n")
	b.WriteString("---\n")

	for _, is := range report.Issues {
		b.WriteString("\n## " + is.Identifier + ": " + is.Title + "\n")
		if is.Project != nil && is.Project.Name != "" {
			b.WriteString("Project: " + is.Project.Name + "\n")
		}
		if len(is.Labels.Nodes) > 0 {
			names := make([]string, 0, len(is.Labels.Nodes))
			for _, l := range is.Labels.Nodes {
				names = append(names, l.Name)
			}
			b.WriteString("Labels: " + strings.Join(names, ", ") + "\n")
		}
		b.WriteString("URL: " + is.URL + "\n")
		if desc := strings.TrimSpace(is.Description); desc != "" {
			if len(desc) > 800 {
				desc = desc[:800] + "..."
			}
			b.WriteString("Description:\n" + desc + "\n")
		}
	}

	if err := clipboard.WriteAll(b.String()); err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, StatusMsg{
		Text: "Copied weekly-report prompt (" + strconv.Itoa(len(report.Issues)) + " issues from " + cycleLabel + ")",
		Kind: StatusOk,
	})
}

// CopyTeamWeeklyReportPrompt fetches every completed issue from the team's
// most recently ended cycle and copies a structured LLM prompt to the
// clipboard. The prompt asks an LLM to summarize the whole team's work as a
// Slack weekly status update.
func CopyTeamWeeklyReportPrompt(s *State) {
	if s.Team == nil {
		post(s, StatusMsg{Text: "Select a team first", Kind: StatusWarn})
		return
	}
	post(s, StatusMsg{Text: "Building team weekly report...", Kind: StatusInfo})

	report, err := s.Client.GetTeamLastCycleCompletedIssues(s.Team.ID)
	if err != nil {
		post(s, errStatus(err))
		return
	}
	if len(report.Issues) == 0 {
		post(s, StatusMsg{Text: "No completed issues in last cycle", Kind: StatusWarn})
		return
	}

	cycleLabel := report.Cycle.Name
	if cycleLabel == "" {
		cycleLabel = "Cycle " + strconv.Itoa(report.Cycle.Number)
	}
	dateRange := report.Cycle.StartsAt.Format("2006-01-02") + " → " + report.Cycle.EndsAt.Format("2006-01-02")

	var b strings.Builder
	b.WriteString("You are writing a concise weekly status update for Slack on behalf of an engineering team.\n")
	b.WriteString("Summarize the work the team completed in the cycle below into a short, professional Slack message.\n")
	b.WriteString("Output the result in Markdown format only — no preamble, no surrounding code fences, no commentary.\n")
	b.WriteString("Guidelines:\n")
	b.WriteString("- Lead with a one-line summary of the cycle's focus.\n")
	b.WriteString("- Group related work by theme or project; do not list every issue verbatim.\n")
	b.WriteString("- Use Markdown: `#`/`##` for headings, `-` for bullets, `**bold**` for emphasis.\n")
	b.WriteString("- Mention the cycle name and date range once at the top.\n")
	b.WriteString("- Keep it under ~250 words.\n")
	b.WriteString("- Do not invent details that are not in the data.\n\n")

	b.WriteString("Team: " + s.Team.Name + "\n")
	b.WriteString("Cycle: " + cycleLabel + " (" + dateRange + ")\n")
	b.WriteString("Completed issues: " + strconv.Itoa(len(report.Issues)) + "\n\n")
	b.WriteString("---\n")

	for _, is := range report.Issues {
		b.WriteString("\n## " + is.Identifier + ": " + is.Title + "\n")
		if is.Assignee != nil && is.Assignee.Name != "" {
			b.WriteString("Assignee: " + is.Assignee.Name + "\n")
		}
		if is.Project != nil && is.Project.Name != "" {
			b.WriteString("Project: " + is.Project.Name + "\n")
		}
		if len(is.Labels.Nodes) > 0 {
			names := make([]string, 0, len(is.Labels.Nodes))
			for _, l := range is.Labels.Nodes {
				names = append(names, l.Name)
			}
			b.WriteString("Labels: " + strings.Join(names, ", ") + "\n")
		}
		b.WriteString("URL: " + is.URL + "\n")
		if desc := strings.TrimSpace(is.Description); desc != "" {
			if len(desc) > 800 {
				desc = desc[:800] + "..."
			}
			b.WriteString("Description:\n" + desc + "\n")
		}
	}

	if err := clipboard.WriteAll(b.String()); err != nil {
		post(s, errStatus(err))
		return
	}
	post(s, StatusMsg{
		Text: "Copied team weekly-report prompt (" + strconv.Itoa(len(report.Issues)) + " issues from " + cycleLabel + ")",
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

	// Resolve labelIDs serially so that label-creation for a new name happens
	// at most once. Then dispatch the actual UpdateIssueLabels calls in
	// parallel with a bounded fan-out.
	type job struct {
		i         int
		issue     linear.Issue
		suggested string
		labelID   string
	}
	jobs := make([]job, 0, len(issues))
	for i, issue := range issues {
		suggested := suggestions[issue.Identifier]
		if suggested == "" {
			post(s, StatusMsg{
				Text: "Auto-label [" + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(issues)) + "] " + issue.Identifier + ": skipped",
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
		jobs = append(jobs, job{i: i, issue: issue, suggested: suggested, labelID: labelID})
	}

	const autoLabelFanout = 4
	sem := make(chan struct{}, autoLabelFanout)
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.Client.UpdateIssueLabels(j.issue.ID, []string{j.labelID}); err != nil {
				post(s, errStatus(err))
				return
			}
			post(s, StatusMsg{
				Text: "Auto-label [" + strconv.Itoa(j.i+1) + "/" + strconv.Itoa(len(issues)) + "] " + j.issue.Identifier + " → " + j.suggested,
				Kind: StatusOk,
			})
		}(j)
	}
	wg.Wait()

	post(s, RefreshRequested{})
	post(s, StatusMsg{Text: "Auto-labeling complete", Kind: StatusOk})
}
