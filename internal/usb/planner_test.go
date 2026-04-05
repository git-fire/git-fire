package usb

import (
	"path/filepath"
	"testing"

	"github.com/git-fire/git-fire/internal/git"
)

func TestBuildPlans_WithOverrides(t *testing.T) {
	repos := []git.Repository{
		{
			Name:    "app",
			Path:    "/tmp/app",
			IsDirty: true,
		},
	}
	target := t.TempDir()
	cfg := map[string]*VolumeConfig{
		target: {
			LayoutDir: "repos",
			Strategy:  StrategyMirror,
		},
	}
	overrides := map[string]RepoOverride{
		"/tmp/app": {
			Strategy:   StrategyClone,
			RepoPath:   "custom/app-backup",
			SyncPolicy: "prune",
		},
	}

	plans := BuildPlans(repos, []string{target}, cfg, overrides, PlanOptions{AutoCommit: true})
	if len(plans) != 1 {
		t.Fatalf("expected one plan, got %d", len(plans))
	}
	if len(plans[0].Actions) != 2 {
		t.Fatalf("expected two actions (auto-commit + sync), got %d", len(plans[0].Actions))
	}
	syncAction := plans[0].Actions[1]
	if syncAction.Strategy != StrategyClone {
		t.Fatalf("expected override strategy git-clone, got %s", syncAction.Strategy)
	}
	if syncAction.SyncPolicy != "prune" {
		t.Fatalf("expected sync policy prune, got %s", syncAction.SyncPolicy)
	}
	wantDest := filepath.Join(target, "repos", "custom/app-backup")
	if syncAction.Destination != wantDest {
		t.Fatalf("expected destination %s, got %s", wantDest, syncAction.Destination)
	}
}
