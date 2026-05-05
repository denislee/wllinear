package linear

const mutationCreateIssue = `mutation($input: IssueCreateInput!) {
	issueCreate(input: $input) {
		success
		issue {` + issueFragment + `
		}
	}
}`

const mutationUpdateIssue = `mutation($id: String!, $input: IssueUpdateInput!) {
	issueUpdate(id: $id, input: $input) {
		success
		issue {` + issueFragment + `
		}
	}
}`

const mutationCreateLabel = `mutation($name: String!, $teamId: String!) {
	issueLabelCreate(input: { name: $name, teamId: $teamId }) {
		success
		issueLabel { id }
	}
}`
