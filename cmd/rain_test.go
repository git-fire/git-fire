package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	testutil "github.com/git-fire/git-testkit"
)

func TestRainCommand_Aliases(t *testing.T) {
	if rainCmd.Name() != "rain" {
		t.Fatalf("expected command name rain, got %q", rainCmd.Name())
	}
	aliases := rainCmd.Aliases
	if len(aliases) != 2 || aliases[0] != "hydrate" || aliases[1] != "hydrant" {
		t.Fatalf("unexpected rain aliases: %v", aliases)
	}
}

func TestRainCommand_FlagParsing_Risky(t *testing.T) {
	resetFlags()
	if err := rainCmd.ParseFlags([]string{"--risky"}); err != nil {
		t.Fatalf("rainCmd.ParseFlags(--risky) error = %v", err)
	}
	if !rainRisky {
		t.Fatal("expected rainRisky flag to be set")
	}
}

func TestRunRain_SafeModeSkipsLocalAheadBranch(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("rain-safe-cmd").
		WithRemote("origin", remote).
		AddFile("tracked.txt", "v1\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)
	repo.AddFile("local-only.txt", "ahead\n").Commit("local ahead")
	localAheadSHA := testutil.GetCurrentSHA(t, repo.Path())

	resetFlags()
	rainPath = filepath.Dir(repo.Path())

	var runErr error
	output := captureStdout(t, func() {
		runErr = runRain(rainCmd, []string{})
	})
	if runErr != nil {
		t.Fatalf("runRain() safe mode error = %v", runErr)
	}
	if !strings.Contains(output, "skipped-local-ahead") {
		t.Fatalf("expected safe mode output to mention skipped-local-ahead, got:\n%s", output)
	}
	if got := testutil.GetCurrentSHA(t, repo.Path()); got != localAheadSHA {
		t.Fatalf("safe mode should preserve local-ahead SHA (want=%s got=%s)", localAheadSHA, got)
	}
}

func TestRunRain_RiskyFlagResetsLocalAheadBranch(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("rain-risky-cmd").
		WithRemote("origin", remote).
		AddFile("tracked.txt", "v1\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)
	remoteSHA := testutil.GetCurrentSHA(t, repo.Path())

	repo.AddFile("local-only.txt", "ahead\n").Commit("local ahead")
	if aheadSHA := testutil.GetCurrentSHA(t, repo.Path()); aheadSHA == remoteSHA {
		t.Fatal("test setup error: local-ahead SHA must differ from remote SHA")
	}

	resetFlags()
	rainPath = filepath.Dir(repo.Path())
	rainRisky = true

	var runErr error
	output := captureStdout(t, func() {
		runErr = runRain(rainCmd, []string{})
	})
	if runErr != nil {
		t.Fatalf("runRain() risky mode error = %v", runErr)
	}
	if !strings.Contains(output, "updated-risky") {
		t.Fatalf("expected risky output to mention updated-risky, got:\n%s", output)
	}
	if got := testutil.GetCurrentSHA(t, repo.Path()); got != remoteSHA {
		t.Fatalf("risky mode should reset branch to remote SHA (want=%s got=%s)", remoteSHA, got)
	}
	if !hasRainBackupBranchInCmdTest(t, repo.Path()) {
		t.Fatal("expected runRain risky mode to create a backup branch")
	}
}

func hasRainBackupBranchInCmdTest(t *testing.T, repoPath string) bool {
	t.Helper()
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git branch listing failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "git-fire-rain-backup-") {
			return true
		}
	}
	return false
}
