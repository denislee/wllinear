package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	apiURL         = "https://api.linear.app/graphql"
	requestTimeout = 30 * time.Second
	maxProjects    = 5000 // safety cap on paginated project fetch
	cycleFanout    = 5    // max concurrent issue-by-cycle fetches
)

// Client is a Linear GraphQL client.
type Client struct {
	token string
	http  *http.Client
}

// NewClient returns a new Client with a tuned, shared HTTP client.
func NewClient(token string) *Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   8,
		MaxConnsPerHost:       16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &Client{
		token: token,
		http: &http.Client{
			Transport: transport,
			Timeout:   requestTimeout,
		},
	}
}

// extractOperationName reads the operation name from `query Name(...)` or
// `mutation Name(...)`. Returns the bare keyword if unnamed.
func extractOperationName(query string) string {
	q := strings.TrimLeftFunc(query, isGraphQLSpace)
	var keyword string
	switch {
	case strings.HasPrefix(q, "query"):
		keyword = "query"
	case strings.HasPrefix(q, "mutation"):
		keyword = "mutation"
	case strings.HasPrefix(q, "subscription"):
		keyword = "subscription"
	default:
		return "anonymous"
	}
	rest := strings.TrimLeftFunc(q[len(keyword):], isGraphQLSpace)
	end := 0
	for end < len(rest) {
		c := rest[end]
		if c == '(' || c == '{' || isGraphQLSpace(rune(c)) {
			break
		}
		end++
	}
	if end == 0 {
		return keyword
	}
	return rest[:end]
}

func isGraphQLSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ','
}

func (c *Client) execute(query string, vars map[string]any, resp any) error {
	opName := extractOperationName(query)
	log.Printf("[Linear API] Executing query: %s", opName)
	if len(vars) > 0 {
		varsJSON, _ := json.Marshal(vars)
		log.Printf("[Linear API] Variables: %s", string(varsJSON))
	}

	body, err := json.Marshal(map[string]any{
		"query":         query,
		"variables":     vars,
		"operationName": opName,
	})
	if err != nil {
		log.Printf("[Linear API] Error marshaling request: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("[Linear API] Error creating request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.token)

	start := time.Now()
	r, err := c.http.Do(req)
	if err != nil {
		log.Printf("[Linear API] HTTP Request failed: %v", err)
		return err
	}
	defer r.Body.Close()

	log.Printf("[Linear API] Received response %d in %v (%s)", r.StatusCode, time.Since(start), opName)

	if r.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(r.Body)
		log.Printf("[Linear API] HTTP Error: %s", string(b))
		return fmt.Errorf("linear api error (%d): %s", r.StatusCode, string(b))
	}

	var wrapper struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(r.Body).Decode(&wrapper); err != nil {
		log.Printf("[Linear API] Error decoding response: %v", err)
		return err
	}
	if len(wrapper.Errors) > 0 {
		log.Printf("[Linear API] GraphQL Error: %s", wrapper.Errors[0].Message)
		return fmt.Errorf("linear api error: %s", wrapper.Errors[0].Message)
	}
	return json.Unmarshal(wrapper.Data, resp)
}

// GetViewer returns the current authenticated user.
func (c *Client) GetViewer() (*User, error) {
	var resp struct {
		Viewer User `json:"viewer"`
	}
	if err := c.execute(queryViewer, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Viewer, nil
}

// GetTeams returns all teams the user has access to.
func (c *Client) GetTeams() ([]Team, error) {
	var resp struct {
		Teams struct {
			Nodes []Team `json:"nodes"`
		} `json:"teams"`
	}
	if err := c.execute(queryTeams, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Teams.Nodes, nil
}

// TeamMetadata holds members, projects, cycles, and workflow states for a team.
type TeamMetadata struct {
	Members  []User
	Projects []Project
	Cycles   []Cycle
	States   []WorkflowState
	Labels   []Label
}

// GetTeamMetadata returns members, projects, and cycles for a team.
func (c *Client) GetTeamMetadata(teamID string) (*TeamMetadata, error) {
	vars := map[string]any{
		"teamId": teamID,
	}
	var resp struct {
		Team struct {
			Members struct {
				Nodes []User `json:"nodes"`
			} `json:"members"`
			States struct {
				Nodes []WorkflowState `json:"nodes"`
			} `json:"states"`
			Labels struct {
				Nodes []Label `json:"nodes"`
			} `json:"labels"`
		} `json:"team"`
	}
	if err := c.execute(queryTeamMetadata, vars, &resp); err != nil {
		return nil, err
	}

	projects, err := c.getAllProjects()
	if err != nil {
		return nil, err
	}

	cycles, err := c.getAllTeamCycles(teamID)
	if err != nil {
		return nil, err
	}

	return &TeamMetadata{
		Members:  resp.Team.Members.Nodes,
		Projects: projects,
		Cycles:   cycles,
		States:   resp.Team.States.Nodes,
		Labels:   resp.Team.Labels.Nodes,
	}, nil
}

// getAllProjects fetches all projects across all pages.
func (c *Client) getAllProjects() ([]Project, error) {
	var all []Project
	var cursor *string
	for {
		vars := map[string]any{}
		if cursor != nil {
			vars["after"] = *cursor
		}
		var resp struct {
			Projects struct {
				Nodes    []Project `json:"nodes"`
				PageInfo PageInfo  `json:"pageInfo"`
			} `json:"projects"`
		}
		if err := c.execute(queryProjects, vars, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Projects.Nodes...)
		if !resp.Projects.PageInfo.HasNextPage || len(all) >= maxProjects {
			break
		}
		cursor = &resp.Projects.PageInfo.EndCursor
	}

	return all, nil
}

// getAllTeamCycles fetches all cycles for a team across all pages.
func (c *Client) getAllTeamCycles(teamID string) ([]Cycle, error) {
	var all []Cycle
	var cursor *string
	for {
		vars := map[string]any{
			"teamId": teamID,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}
		var resp struct {
			Team struct {
				Cycles struct {
					Nodes    []Cycle  `json:"nodes"`
					PageInfo PageInfo `json:"pageInfo"`
				} `json:"cycles"`
			} `json:"team"`
		}
		if err := c.execute(queryTeamCycles, vars, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Team.Cycles.Nodes...)
		if !resp.Team.Cycles.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Team.Cycles.PageInfo.EndCursor
	}
	log.Printf("[Linear API] Fetched %d cycles for team %s", len(all), teamID)
	return all, nil
}

// GetLeadingProjects returns projects where the user is the lead and status is "Developing".
func (c *Client) GetLeadingProjects(userID string) ([]Project, error) {
	vars := map[string]any{
		"userId": userID,
	}
	var resp struct {
		Projects struct {
			Nodes []Project `json:"nodes"`
		} `json:"projects"`
	}
	if err := c.execute(queryLeadingProjects, vars, &resp); err != nil {
		return nil, err
	}
	return resp.Projects.Nodes, nil
}

// GetProjectIssuesFromLastCycle returns names of completed issues from the most
// recently ended cycle of a project.
func (c *Client) GetProjectIssuesFromLastCycle(projectID string) ([]string, error) {
	cycles, err := c.GetProjectIssuesByCycles(projectID)
	if err != nil {
		return nil, err
	}
	if len(cycles) == 0 {
		return nil, nil
	}
	last := cycles[0]
	titles := make([]string, 0, len(last.Issues))
	for _, iss := range last.Issues {
		if !strings.EqualFold(iss.State.Name, "Done") {
			continue
		}
		titles = append(titles, iss.Title)
	}
	return titles, nil
}

// fetchProjectCycleIDs scans up to 500 issues in a project and returns the unique
// cycles attached to them. Issues without a cycle are skipped.
func (c *Client) fetchProjectCycleIDs(projectID string) ([]Cycle, error) {
	cycleMap := make(map[string]Cycle)
	var cursor *string
	for page := 0; page < 5; page++ {
		vars := map[string]any{
			"projectId": projectID,
			"first":     100,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}
		var resp struct {
			Issues struct {
				Nodes []struct {
					Cycle *Cycle `json:"cycle"`
				} `json:"nodes"`
				PageInfo PageInfo `json:"pageInfo"`
			} `json:"issues"`
		}
		if err := c.execute(queryProjectAllCycles, vars, &resp); err != nil {
			return nil, err
		}
		for _, n := range resp.Issues.Nodes {
			if n.Cycle == nil || n.Cycle.ID == "" {
				continue
			}
			cycleMap[n.Cycle.ID] = *n.Cycle
		}
		if !resp.Issues.PageInfo.HasNextPage || resp.Issues.PageInfo.EndCursor == "" {
			break
		}
		next := resp.Issues.PageInfo.EndCursor
		cursor = &next
	}
	cycles := make([]Cycle, 0, len(cycleMap))
	for _, cyc := range cycleMap {
		cycles = append(cycles, cyc)
	}
	sort.Slice(cycles, func(i, j int) bool {
		return cycles[i].EndsAt.After(cycles[j].EndsAt)
	})
	return cycles, nil
}

// GetProjectIssuesByCycles returns all completed issues for a project, grouped by cycle.
// Per-cycle requests run concurrently with a bounded fan-out (cycleFanout).
func (c *Client) GetProjectIssuesByCycles(projectID string) ([]ProjectCycleIssues, error) {
	cycles, err := c.fetchProjectCycleIDs(projectID)
	if err != nil {
		return nil, err
	}
	if len(cycles) == 0 {
		return nil, nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]ProjectCycleIssues, 0, len(cycles))
	errs := make([]error, 0)
	sem := make(chan struct{}, cycleFanout)

	for _, cyc := range cycles {
		wg.Add(1)
		sem <- struct{}{}
		go func(cyc Cycle) {
			defer wg.Done()
			defer func() { <-sem }()
			vars := map[string]any{
				"projectId": projectID,
				"cycleId":   cyc.ID,
				"first":     250,
			}
			var issuesResp struct {
				Issues struct {
					Nodes []Issue `json:"nodes"`
				} `json:"issues"`
			}
			if err := c.execute(queryProjectIssuesByCycle, vars, &issuesResp); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			results = append(results, ProjectCycleIssues{
				Cycle:  cyc,
				Issues: issuesResp.Issues.Nodes,
			})
			mu.Unlock()
		}(cyc)
	}
	wg.Wait()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Cycle.EndsAt.After(results[j].Cycle.EndsAt)
	})

	return results, nil
}

// CreateIssue creates a new issue and returns it.
func (c *Client) CreateIssue(input IssueCreateInput) (*Issue, error) {
	vars := map[string]any{
		"input": input,
	}
	var resp struct {
		IssueCreate struct {
			Success bool  `json:"success"`
			Issue   Issue `json:"issue"`
		} `json:"issueCreate"`
	}
	if err := c.execute(mutationCreateIssue, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.IssueCreate.Issue, nil
}

// GetIssues fetches issues for a team.
func (c *Client) GetIssues(teamID string, first int, after string, filter map[string]any, includeDescription bool) (*IssueConnection, error) {
	vars := map[string]any{
		"teamId":          teamID,
		"first":           first,
		"includeArchived": false,
	}
	if after != "" {
		vars["after"] = after
	}
	if filter != nil {
		vars["filter"] = filter
	}

	var resp struct {
		Team struct {
			Issues IssueConnection `json:"issues"`
		} `json:"team"`
	}
	if err := c.execute(queryIssues, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.Team.Issues, nil
}

// GetWorkflowStates returns all workflow states for a team.
func (c *Client) GetWorkflowStates(teamID string) ([]WorkflowState, error) {
	vars := map[string]any{
		"teamId": teamID,
	}
	var resp struct {
		Team struct {
			States struct {
				Nodes []WorkflowState `json:"nodes"`
			} `json:"states"`
		} `json:"team"`
	}
	if err := c.execute(queryWorkflowStates, vars, &resp); err != nil {
		return nil, err
	}
	return resp.Team.States.Nodes, nil
}

// UpdateIssue updates an existing issue.
func (c *Client) UpdateIssue(id string, input IssueUpdateInput) (*Issue, error) {
	vars := map[string]any{
		"id":    id,
		"input": input,
	}
	var resp struct {
		IssueUpdate struct {
			Success bool  `json:"success"`
			Issue   Issue `json:"issue"`
		} `json:"issueUpdate"`
	}
	if err := c.execute(mutationUpdateIssue, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.IssueUpdate.Issue, nil
}

// filterCountSampleSize bounds the per-filter response size for GetFilterCounts.
// Counts up to this number are exact; anything above shows as N+ to the caller.
const filterCountSampleSize = 25

// GetFilterCounts returns the number of issues matching each of the provided filters.
// Each alias requests only `pageInfo { hasNextPage }` plus a small node slice for the
// count; if hasNextPage is true the returned count is the sample size and the bool is true.
func (c *Client) GetFilterCounts(teamID string, filters map[string]map[string]any) (map[string]int, map[string]bool, error) {
	var aliases []string
	var varDefs []string
	vars := map[string]any{"teamId": teamID}

	varDefs = append(varDefs, "$teamId: String!")

	i := 0
	aliasToName := make(map[string]string)
	for name, filter := range filters {
		alias := fmt.Sprintf("f%d", i)
		aliasToName[alias] = name
		varName := fmt.Sprintf("$filter%d", i)
		varDefs = append(varDefs, fmt.Sprintf("%s: IssueFilter", varName))

		f := filter
		if name == "My Unlabeled Issues" {
			// Deep copy filter and add labels length=0 check.
			newFilter := make(map[string]any)
			for k, v := range filter {
				newFilter[k] = v
			}
			newFilter["labels"] = map[string]any{"length": map[string]any{"eq": 0}}
			f = newFilter
		}

		aliases = append(aliases, fmt.Sprintf(
			"%s: issues(first: %d, filter: %s) { nodes { id } pageInfo { hasNextPage } }",
			alias, filterCountSampleSize, varName,
		))
		vars[fmt.Sprintf("filter%d", i)] = f
		i++
	}

	query := fmt.Sprintf("query FilterCounts(%s) {\n  team(id: $teamId) {\n    %s\n  }\n}",
		strings.Join(varDefs, ", "), strings.Join(aliases, "\n    "))

	var respWrapper struct {
		Team map[string]struct {
			Nodes []struct {
				ID string `json:"id"`
			} `json:"nodes"`
			PageInfo PageInfo `json:"pageInfo"`
		} `json:"team"`
	}

	if err := c.execute(query, vars, &respWrapper); err != nil {
		return nil, nil, err
	}

	counts := make(map[string]int, len(respWrapper.Team))
	more := make(map[string]bool, len(respWrapper.Team))
	for alias, val := range respWrapper.Team {
		counts[aliasToName[alias]] = len(val.Nodes)
		more[aliasToName[alias]] = val.PageInfo.HasNextPage
	}
	return counts, more, nil
}

// GetAllIssueLabels fetches all labels in the organization.
func (c *Client) GetAllIssueLabels() ([]Label, error) {
	var all []Label
	var cursor *string
	for {
		vars := map[string]any{}
		if cursor != nil {
			vars["after"] = *cursor
		}
		var resp struct {
			IssueLabels struct {
				Nodes    []Label  `json:"nodes"`
				PageInfo PageInfo `json:"pageInfo"`
			} `json:"issueLabels"`
		}
		if err := c.execute(queryAllIssueLabels, vars, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.IssueLabels.Nodes...)
		if !resp.IssueLabels.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.IssueLabels.PageInfo.EndCursor
	}
	return all, nil
}

// CreateLabel creates a new label in a team.
func (c *Client) CreateLabel(name, teamID string) (string, error) {
	vars := map[string]any{
		"input": map[string]any{
			"name":   name,
			"teamId": teamID,
		},
	}
	var resp struct {
		IssueLabelCreate struct {
			Success    bool  `json:"success"`
			IssueLabel Label `json:"issueLabel"`
		} `json:"issueLabelCreate"`
	}
	if err := c.execute(mutationCreateLabel, vars, &resp); err != nil {
		return "", err
	}
	return resp.IssueLabelCreate.IssueLabel.ID, nil
}

// UpdateIssueLabels replaces the labels on an issue.
func (c *Client) UpdateIssueLabels(issueID string, labelIDs []string) error {
	vars := map[string]any{
		"id": issueID,
		"input": map[string]any{
			"labelIds": labelIDs,
		},
	}
	var resp struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}
	return c.execute(mutationUpdateIssue, vars, &resp)
}
