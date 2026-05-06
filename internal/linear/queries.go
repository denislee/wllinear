package linear

const issueFragment = `
	id
	identifier
	title
	description
	priority
	state {
		id
		name
		color
		type
		position
	}
	assignee {
		id
		name
		email
	}
	project {
		id
		name
	}
	cycle {
		id
		number
		name
		startsAt
		endsAt
		completedAt
	}
	labels {
		nodes {
			id
			name
			color
		}
	}
	url
	createdAt
	updatedAt
`

const queryViewer = `query {
	viewer {
		id
		name
		email
	}
}`

const queryTeams = `query {
	teams {
		nodes {
			id
			name
			key
		}
	}
}`

const queryIssues = `query($teamId: String!, $first: Int!, $after: String, $filter: IssueFilter, $includeArchived: Boolean) {
	team(id: $teamId) {
		issues(first: $first, after: $after, filter: $filter, includeArchived: $includeArchived) {
			nodes {` + issueFragment + `
			}
			pageInfo {
				hasNextPage
				endCursor
			}
		}
	}
}`

const queryMyIssues = `query($first: Int!, $after: String, $filter: IssueFilter) {
	issues(first: $first, after: $after, filter: $filter) {
		nodes {` + issueFragment + `
		}
		pageInfo {
			hasNextPage
			endCursor
		}
	}
}`

const queryIssue = `query($id: String!) {
	issue(id: $id) {` + issueFragment + `
	}
}`

const queryWorkflowStates = `query($teamId: String!) {
	team(id: $teamId) {
		states {
			nodes {
				id
				name
				color
				type
				position
			}
		}
	}
}`

const queryTeamMetadata = `query($teamId: String!) {
	team(id: $teamId) {
		members(first: 100) {
			nodes {
				id
				name
				email
			}
		}
		states {
			nodes {
				id
				name
				color
				type
				position
			}
		}
		labels(includeArchived: false) {
			nodes {
				id
				name
				color
			}
		}
	}
}`

const queryTeamCycles = `query($teamId: String!, $after: String) {
	team(id: $teamId) {
		cycles(first: 250, after: $after) {
			pageInfo { hasNextPage endCursor }
			nodes {
				id
				number
				name
				startsAt
				endsAt
				completedAt
			}
		}
	}
}`

const queryAllIssueLabels = `query($after: String) {
	issueLabels(first: 250, after: $after, includeArchived: false) {
		pageInfo { hasNextPage endCursor }
		nodes {
			id
			name
			isGroup
			parent { name }
			team { id }
		}
	}
}`

const queryTeamLabelByName = `query($teamId: ID!, $name: String!) {
	issueLabels(filter: { team: { id: { eq: $teamId } }, name: { eq: $name } }, includeArchived: false) {
		nodes { id name }
	}
}`

const queryProjects = `query($after: String) {
	projects(first: 250, after: $after, includeArchived: true) {
		nodes {
			id
			name
			status {
				name
			}
			lead {
				id
			}
		}
		pageInfo {
			hasNextPage
			endCursor
		}
	}
}`

const queryLeadingProjects = `query($userId: ID!) {
	projects(filter: { lead: { id: { eq: $userId } }, status: { name: { eq: "Developing" } } }) {
		nodes {
			id
			name
			status {
				name
			}
		}
	}
}`

const queryProjectAllCycles = `query($projectId: ID!, $first: Int!, $after: String) {
	issues(filter: { project: { id: { eq: $projectId } } }, first: $first, after: $after) {
		nodes {
			cycle {
				id
				name
				number
				startsAt
				endsAt
				completedAt
			}
		}
		pageInfo {
			hasNextPage
			endCursor
		}
	}
}`

const queryProjectIssuesByCycle = `query($projectId: ID!, $cycleId: ID!, $first: Int!, $after: String) {
	issues(filter: { project: { id: { eq: $projectId } }, cycle: { id: { eq: $cycleId } }, state: { type: { eq: "completed" } } }, first: $first, after: $after) {
		nodes {` + issueFragment + `
		}
		pageInfo {
			hasNextPage
			endCursor
		}
	}
}`
