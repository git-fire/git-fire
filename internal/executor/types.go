// Package executor plans backup actions, runs them with concurrency limits, and logs results.
package executor

import (
	"time"

	"github.com/git-fire/git-fire/internal/git"
)

// PushPlan represents the plan of what will be pushed
type PushPlan struct {
	Repos       []RepoPlan
	TotalRepos  int
	DirtyRepos  int
	Conflicts   int
	FireBranches int
	DryRun      bool
	CreatedAt   time.Time
}

// RepoPlan represents the plan for a single repository
type RepoPlan struct {
	Repo        git.Repository
	Actions     []Action
	HasConflict bool
	FireBranch  string // New branch name if conflict detected
	Skip        bool   // Skip this repo
	SkipReason  string
}

// Action represents a single action to perform
type Action struct {
	Type        ActionType
	Description string
	Remote      string
	Branch      string
	Error       error // Set if action failed
}

// ActionType defines types of actions
type ActionType int

const (
	ActionAutoCommit ActionType = iota
	ActionPushBranch
	ActionPushAll
	ActionPushKnown
	ActionCreateFireBranch
	ActionSkip
)

func (a ActionType) String() string {
	switch a {
	case ActionAutoCommit:
		return "auto-commit"
	case ActionPushBranch:
		return "push-branch"
	case ActionPushAll:
		return "push-all"
	case ActionPushKnown:
		return "push-known"
	case ActionCreateFireBranch:
		return "create-fire-branch"
	case ActionSkip:
		return "skip"
	default:
		return "unknown"
	}
}

// ExecutionResult represents the result of executing a plan
type ExecutionResult struct {
	Plan           *PushPlan
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	Success        int // Repos pushed successfully
	Failed         int // Repos that failed
	Skipped        int // Repos skipped
	TotalActions   int
	FailedActions  int
	RepoResults    []RepoResult
}

// RepoResult represents the result for a single repo
type RepoResult struct {
	Path           string
	Success        bool
	Error          error
	Actions        []Action
	PushedBranches []string
	Duration       time.Duration
}

// Progress represents progress updates during execution
type Progress struct {
	CurrentRepo  int
	TotalRepos   int
	RepoName     string
	Action       string
	Status       ProgressStatus
	Error        error
}

// ProgressStatus represents the status of an operation
type ProgressStatus int

const (
	StatusStarting ProgressStatus = iota
	StatusInProgress
	StatusSuccess
	StatusFailed
	StatusSkipped
)

func (s ProgressStatus) String() string {
	switch s {
	case StatusStarting:
		return "starting"
	case StatusInProgress:
		return "in-progress"
	case StatusSuccess:
		return "success"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}
