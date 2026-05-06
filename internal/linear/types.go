package linear

import "time"

// User represents a Linear user.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ProjectStatus represents a Linear project status.
type ProjectStatus struct {
	Name string `json:"name"`
}

// Project represents a Linear project.
type Project struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Status ProjectStatus `json:"status"`
	Lead   *User         `json:"lead"`
}

// Cycle represents a Linear cycle.
type Cycle struct {
	ID          string     `json:"id"`
	Number      int        `json:"number"`
	Name        string     `json:"name"`
	StartsAt    time.Time  `json:"startsAt"`
	EndsAt      time.Time  `json:"endsAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

// ProjectCycleIssues contains completed issues for a specific cycle.
type ProjectCycleIssues struct {
	Cycle  Cycle
	Issues []Issue
}

// Team represents a Linear team.
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// WorkflowState represents a workflow state in Linear.
type WorkflowState struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Color    string  `json:"color"`
	Type     string  `json:"type"` // triage, backlog, unstarted, started, completed, canceled
	Position float64 `json:"position"`
}

// Label represents a Linear label.
type Label struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	IsGroup bool   `json:"isGroup"`
	TeamID  string `json:"teamId"`
}

// Issue represents a Linear issue.
type Issue struct {
	ID          string        `json:"id"`
	Identifier  string        `json:"identifier"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Priority    int           `json:"priority"`
	State       WorkflowState `json:"state"`
	Assignee    *User         `json:"assignee"`
	Project     *Project      `json:"project"`
	Cycle       *Cycle        `json:"cycle"`
	Labels      struct {
		Nodes []Label `json:"nodes"`
	} `json:"labels"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// IssueConnection represents a paginated list of issues.
type IssueConnection struct {
	Nodes    []Issue  `json:"nodes"`
	PageInfo PageInfo `json:"pageInfo"`
}

// PageInfo contains pagination information.
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// IssueCreateInput represents the input for creating an issue.
type IssueCreateInput struct {
	TeamID      string  `json:"teamId"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	StateID     *string `json:"stateId,omitempty"`
	AssigneeID  *string `json:"assigneeId,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	ProjectID   *string `json:"projectId,omitempty"`
	CycleID     *string `json:"cycleId,omitempty"`
}

// IssueUpdateInput represents the input for updating an issue.
type IssueUpdateInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	StateID     *string `json:"stateId,omitempty"`
	AssigneeID  *string `json:"assigneeId,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	ProjectID   *string `json:"projectId,omitempty"`
	CycleID     *string `json:"cycleId,omitempty"`
}
