package linear

const mutationCreateIssue = `mutation CreateIssue($input: IssueCreateInput!) {
	issueCreate(input: $input) {
		success
		issue {` + issueFragment + `
		}
	}
}`

const mutationUpdateIssue = `mutation UpdateIssue($id: String!, $input: IssueUpdateInput!) {
	issueUpdate(id: $id, input: $input) {
		success
		issue {` + issueFragment + `
		}
	}
}`

const mutationCreateLabel = `mutation CreateLabel($name: String!, $teamId: String!) {
	issueLabelCreate(input: { name: $name, teamId: $teamId }) {
		success
		issueLabel { id }
	}
}`

const mutationCreateComment = `mutation CreateComment($issueId: String!, $body: String!) {
	commentCreate(input: { issueId: $issueId, body: $body }) {
		success
		comment { id }
	}
}`
