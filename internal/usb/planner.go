package usb

import (
	"path/filepath"
	"strings"

	"github.com/git-fire/git-fire/internal/git"
)

type ActionType string

const (
	ActionAutoCommit ActionType = "auto-commit"
	ActionSync       ActionType = "sync"
)

type Action struct {
	Type        ActionType
	TargetRoot  string
	Strategy    string
	Destination string
	SyncPolicy  string
}

type RepoPlan struct {
	Repo    git.Repository
	Actions []Action
}

type PlanOptions struct {
	AutoCommit bool
}

func BuildPlans(
	repos []git.Repository,
	targetRoots []string,
	targetCfg map[string]*VolumeConfig,
	repoOverrides map[string]RepoOverride,
	opts PlanOptions,
) []RepoPlan {
	plans := make([]RepoPlan, 0, len(repos))
	for _, repo := range repos {
		repoPlan := RepoPlan{Repo: repo, Actions: make([]Action, 0)}
		if repo.IsDirty && opts.AutoCommit {
			repoPlan.Actions = append(repoPlan.Actions, Action{Type: ActionAutoCommit})
		}
		override := repoOverrides[repo.Path]
		for _, target := range targetRoots {
			cfg := targetCfg[target]
			reposRoot := TargetReposRoot(target, cfg)
			destBase := StableRepoName(repo.Path, repo.Name)
			destName := destBase
			if strings.TrimSpace(override.RepoPath) != "" {
				destName = strings.TrimSpace(override.RepoPath)
			}
			strategy := cfg.Strategy
			if strings.TrimSpace(override.Strategy) != "" {
				strategy = strings.TrimSpace(override.Strategy)
			}
			if strings.TrimSpace(override.RepoPath) == "" && strategy == StrategyMirror {
				destName = destBase + ".git"
			}
			dest := filepath.Join(reposRoot, destName)
			syncPolicy := strings.TrimSpace(override.SyncPolicy)
			repoPlan.Actions = append(repoPlan.Actions, Action{
				Type:        ActionSync,
				TargetRoot:  target,
				Strategy:    strategy,
				Destination: dest,
				SyncPolicy:  syncPolicy,
			})
		}
		plans = append(plans, repoPlan)
	}
	return plans
}

type RepoOverride struct {
	Strategy   string
	RepoPath   string
	SyncPolicy string
}
