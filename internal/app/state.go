// Package app holds the central application state for wllinear.
//
// State is updated only from the UI goroutine. Async work happens in
// goroutines that send Event values through Events; the UI loop applies
// each event with Apply and then redraws.
package app

import (
	"sync"

	"github.com/denislee/wllinear/internal/config"
	"github.com/denislee/wllinear/internal/db"
	"github.com/denislee/wllinear/internal/linear"
)

// View identifies which screen is shown in the main panel.
type View int

const (
	ViewIssueList View = iota
	ViewIssueDetail
	ViewProjectCycles
	ViewCreateIssue
	ViewEditIssue
)

// Modal identifies which overlay is open (or none).
type Modal int

const (
	ModalNone Modal = iota
	ModalCreate
	ModalEdit
	ModalStatus
	ModalSearch
	ModalTeam
	ModalHelp
)

// Focus identifies which top-level panel currently receives keyboard input.
type Focus int

const (
	FocusSidebar Focus = iota
	FocusMain
)

// State is the full application state. All fields are mutated from the UI
// goroutine; goroutine workers communicate through Events.
type State struct {
	Client *linear.Client
	DB     *db.DB
	Cfg    *config.Config
	Saved  *config.State

	User     *linear.User
	Teams    []linear.Team
	Team     *linear.Team
	Projects []linear.Project // current team's projects (from metadata)
	LeadingProjects []linear.Project // projects user leads (Developing)
	Meta     *linear.TeamMetadata // current team metadata

	Filters      []string        // sidebar filter list ("---" denotes a separator)
	FilterCounts map[string]int  // filter name -> count
	FilterMore   map[string]bool // filter name -> truncated indicator (sample exceeded)
	ActiveFilter string

	Issues           []linear.Issue
	ProjectCycles    []linear.ProjectCycleIssues
	ExpandedCycles   map[string]bool // cycle ID -> expanded
	CurrentProjectID string // project whose cycles are being viewed (guards stale loads)
	Selected         int    // index into Issues or ProjectCycles
	Compact       bool
	ShowLabels    bool
	ShowPriority  bool
	HideHints     bool
	SidebarWidth  int

	View   View
	Detail *linear.Issue
	Create *CreateModal // populated while View == ViewCreateIssue
	Edit   *EditModal   // populated while View == ViewEditIssue

	Focus Focus

	Modal      Modal
	ModalState any // pointer to a modal struct; type depends on Modal

	ShowSettings bool
	Settings     *SettingsModal

	// Status bar text.
	StatusText  string
	StatusKind  StatusKind
	HintsText   string

	Events chan Event
	Wakeup func() // called after an event is posted, to request a UI redraw
	Inflight sync.WaitGroup
}

type StatusKind int

const (
	StatusInfo StatusKind = iota
	StatusOk
	StatusWarn
	StatusErr
)

// Event is anything that mutates state.
type Event interface{ apply(*State) }

// PostEvent enqueues an event and wakes the UI so it gets applied promptly.
func (s *State) PostEvent(e Event) {
	if s == nil || s.Events == nil {
		return
	}
	s.Events <- e
	if s.Wakeup != nil {
		s.Wakeup()
	}
}

// Drain applies every pending event without blocking. Returns true if any
// event was applied (i.e., the UI should redraw).
func (s *State) Drain() bool {
	any := false
	for {
		select {
		case e := <-s.Events:
			e.apply(s)
			any = true
		default:
			return any
		}
	}
}

