package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
)

// Client is a Linear GraphQL client.
type Client struct {
	token string
}

// NewClient returns a new Client.
func NewClient(token string) *Client {
	return &Client{token: token}
}

func (c *Client) execute(query string, vars map[string]any, resp any) error {
	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": vars,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.token)

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(r.Body)
		return fmt.Errorf("linear api error (%d): %s", r.StatusCode, string(b))
	}

	var wrapper struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(r.Body).Decode(&wrapper); err != nil {
		return err
	}
	if len(wrapper.Errors) > 0 {
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
			Cycles struct {
				Nodes []Cycle `json:"nodes"`
			} `json:"cycles"`
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

	return &TeamMetadata{
		Members:  resp.Team.Members.Nodes,
		Projects: projects,
		Cycles:   resp.Team.Cycles.Nodes,
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
	titles := make([]string, len(last.Issues))
	for i, iss := range last.Issues {
		titles[i] = iss.Title
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

	for _, cyc := range cycles {
		wg.Add(1)
		go func(cyc Cycle) {
			defer wg.Done()
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

// truncate shortens s to n runes, appending an ellipsis when truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
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

// GetFilterCounts returns the number of issues matching each of the provided filters.
func (c *Client) GetFilterCounts(teamID string, filters map[string]map[string]any) (map[string]int, error) {
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

		// We fetch nodes to count them.
		aliases = append(aliases, fmt.Sprintf("%s: issues(first: 1, filter: %s) { totalCount }", alias, varName))
		vars[fmt.Sprintf("filter%d", i)] = f
		i++
	}

	query := fmt.Sprintf("query(%s) {\n  team(id: $teamId) {\n    %s\n  }\n}", strings.Join(varDefs, ", "), strings.Join(aliases, "\n    "))

	var respWrapper struct {
		Team map[string]struct {
			TotalCount int `json:"totalCount"`
		} `json:"team"`
	}

	if err := c.execute(query, vars, &respWrapper); err != nil {
		return nil, err
	}

	res := make(map[string]int)
	for alias, val := range respWrapper.Team {
		res[aliasToName[alias]] = val.TotalCount
	}
	return res, nil
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
