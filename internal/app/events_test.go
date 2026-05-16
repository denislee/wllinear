package app

import (
	"testing"

	"github.com/denislee/wllinear/internal/linear"
)

func TestBuildIssueFilter_MyIssues(t *testing.T) {
	s := &State{User: &linear.User{ID: "u1"}}
	got := buildIssueFilter("My Issues", s)
	assignee, ok := got["assignee"].(map[string]any)
	if !ok {
		t.Fatalf("missing assignee key: %#v", got)
	}
	id, ok := assignee["id"].(map[string]any)
	if !ok || id["eq"] != "u1" {
		t.Fatalf("assignee.id.eq != u1: %#v", got)
	}
}

func TestBuildIssueFilter_MyUnlabeled(t *testing.T) {
	s := &State{User: &linear.User{ID: "u1"}}
	got := buildIssueFilter("My Unlabeled Issues", s)
	if _, ok := got["labels"]; !ok {
		t.Fatalf("missing labels predicate: %#v", got)
	}
	labels := got["labels"].(map[string]any)
	length := labels["length"].(map[string]any)
	if length["eq"] != 0 {
		t.Fatalf("labels.length.eq != 0: %#v", got)
	}
}

func TestBuildIssueFilter_MyIssuesActive(t *testing.T) {
	s := &State{User: &linear.User{ID: "u1"}}
	got := buildIssueFilter("My Issues + Active", s)
	clauses, ok := got["and"].([]map[string]any)
	if !ok || len(clauses) != 2 {
		t.Fatalf("expected 2-clause and: %#v", got)
	}
}

func TestBuildIssueFilter_Project(t *testing.T) {
	s := &State{
		LeadingProjects: []linear.Project{{ID: "p1", Name: "Alpha"}},
	}
	got := buildIssueFilter("Project: Alpha", s)
	proj, ok := got["project"].(map[string]any)
	if !ok {
		t.Fatalf("missing project key: %#v", got)
	}
	id := proj["id"].(map[string]any)
	if id["eq"] != "p1" {
		t.Fatalf("project.id.eq != p1: %#v", got)
	}
}

func TestBuildIssueFilter_UnknownReturnsNil(t *testing.T) {
	s := &State{}
	if got := buildIssueFilter("Project: Nope", s); got != nil {
		t.Fatalf("expected nil for unknown project, got %#v", got)
	}
	if got := buildIssueFilter("garbage", s); got != nil {
		t.Fatalf("expected nil for unknown name, got %#v", got)
	}
}

func TestSnapshotFiltersSkipsSeparator(t *testing.T) {
	s := &State{User: &linear.User{ID: "u1"}}
	snap := snapshotFilters(s, []string{"My Issues", "---", "My Unlabeled Issues"})
	if _, ok := snap["---"]; ok {
		t.Fatalf("separator must be skipped: %#v", snap)
	}
	if len(snap) != 2 {
		t.Fatalf("expected 2 entries, got %d: %#v", len(snap), snap)
	}
}

func TestFilterCountsLoaded_Apply(t *testing.T) {
	s := &State{}
	(FilterCountsLoaded{
		Counts: map[string]int{"My Issues": 7},
		More:   map[string]bool{"My Issues": true},
	}).apply(s)
	if s.FilterCounts["My Issues"] != 7 {
		t.Fatalf("counts not applied: %#v", s.FilterCounts)
	}
	if !s.FilterMore["My Issues"] {
		t.Fatalf("more flag not applied: %#v", s.FilterMore)
	}
}

func TestIssuesLoaded_IgnoresStaleFilter(t *testing.T) {
	s := &State{ActiveFilter: "current"}
	(IssuesLoaded{
		FilterName: "stale",
		Issues:     []linear.Issue{{ID: "i1"}},
	}).apply(s)
	if len(s.Issues) != 0 {
		t.Fatalf("stale issues should be dropped, got %d", len(s.Issues))
	}
}

func TestIssuesLoaded_AppliesMatchingFilter(t *testing.T) {
	s := &State{ActiveFilter: "current"}
	issues := []linear.Issue{{ID: "i1"}, {ID: "i2"}}
	(IssuesLoaded{FilterName: "current", Issues: issues}).apply(s)
	if len(s.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(s.Issues))
	}
}

func TestProjectCyclesLoaded_IgnoresStaleProject(t *testing.T) {
	s := &State{CurrentProjectID: "active"}
	(ProjectCyclesLoaded{
		ProjectID: "old",
		Cycles:    []linear.ProjectCycleIssues{{}},
	}).apply(s)
	if len(s.ProjectCycles) != 0 {
		t.Fatalf("stale project cycles should be dropped")
	}
}
