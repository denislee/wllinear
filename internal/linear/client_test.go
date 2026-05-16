package linear

import "testing"

func TestExtractOperationName(t *testing.T) {
	cases := []struct {
		query string
		want  string
	}{
		{queryViewer, "Viewer"},
		{queryTeams, "Teams"},
		{queryIssues, "Issues"},
		{queryTeamMetadata, "TeamMetadata"},
		{queryProjectIssuesByCycle, "ProjectIssuesByCycle"},
		{mutationCreateIssue, "CreateIssue"},
		{mutationUpdateIssue, "UpdateIssue"},
		{mutationCreateLabel, "CreateLabel"},
		// Anonymous fall-throughs.
		{"query { viewer { id } }", "query"},
		{"mutation { x { y } }", "mutation"},
		{"   query   Named  ( $x: Int ) { x }", "Named"},
		{"not a graphql doc", "anonymous"},
	}
	for _, tc := range cases {
		got := extractOperationName(tc.query)
		if got != tc.want {
			t.Errorf("extractOperationName(%q) = %q, want %q",
				firstLine(tc.query), got, tc.want)
		}
	}
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}
