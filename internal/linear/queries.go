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

const queryViewer = `query Viewer {
	viewer {
		id
		name
		email
	}
}`

const queryTeams = `query Teams {
	teams {
		nodes {
			id
			name
			key
		}
	}
}`

const queryIssues = `query Issues($teamId: String!, $first: Int!, $after: String, $filter: IssueFilter, $includeArchived: Boolean) {
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

const queryMyIssues = `query MyIssues($first: Int!, $after: String, $filter: IssueFilter) {
	issues(first: $first, after: $after, filter: $filter) {
		nodes {` + issueFragment + `
		}
		pageInfo {
			hasNextPage
			endCursor
		}
	}
}`

const queryIssue = `query Issue($id: String!) {
	issue(id: $id) {` + issueFragment + `
	}
}`

const queryWorkflowStates = `query WorkflowStates($teamId: String!) {
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

const queryTeamMetadata = `query TeamMetadata($teamId: String!) {
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

const queryTeamCycles = `query TeamCycles($teamId: String!, $after: String) {
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

const queryAllIssueLabels = `query AllIssueLabels($after: String) {
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

const queryTeamLabelByName = `query TeamLabelByName($teamId: ID!, $name: String!) {
	issueLabels(filter: { team: { id: { eq: $teamId } }, name: { eq: $name } }, includeArchived: false) {
		nodes { id name }
	}
}`

const queryProjects = `query Projects($after: String) {
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

const queryLeadingProjects = `query LeadingProjects($userId: ID!) {
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

const queryProjectDetail = `query ProjectDetail($id: String!) {
	project(id: $id) {
		id
		name
		description
		content
		status { name }
		priority
		priorityLabel
		progress
		scope
		startDate
		targetDate
		startedAt
		completedAt
		canceledAt
		createdAt
		updatedAt
		url
		slugId
		lead { name email }
		members { nodes { name email } }
	}
}`

const queryProjectAllCycles = `query ProjectAllCycles($projectId: ID!, $first: Int!, $after: String) {
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

const queryProjectIssuesByCycle = `query ProjectIssuesByCycle($projectId: ID!, $cycleId: ID!, $first: Int!, $after: String) {
	issues(filter: { project: { id: { eq: $projectId } }, cycle: { id: { eq: $cycleId } }, state: { type: { eq: "completed" } } }, first: $first, after: $after) {
		nodes {` + issueFragment + `
		}
		pageInfo {
			hasNextPage
			endCursor
		}
	}
}`
