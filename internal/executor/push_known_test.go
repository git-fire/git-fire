package executor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-fire/git-harness/git"
	testutil "github.com/git-fire/git-testkit"
)

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, b)
	}
	return string(b)
}

func TestExecutePushKnownBranches_BehindRemoteDoesNotFailOrMoveHEAD(t *testing.T) {
	remote := testutil.CreateBareRemote(t, "origin")
	local := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:    "local",
		Remotes: map[string]string{"origin": remote},
		Files:   map[string]string{"x.txt": "x"},
	})
	main, err := git.GetCurrentBranch(local)
	if err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "push", "-u", "origin", main)

	testutil.RunGitCmd(t, local, "checkout", "-b", "trail")
	if err := os.WriteFile(filepath.Join(local, "t.txt"), []byte("t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "add", "t.txt")
	testutil.RunGitCmd(t, local, "commit", "-m", "trail1")
	testutil.RunGitCmd(t, local, "push", "-u", "origin", "trail")

	peer := filepath.Join(t.TempDir(), "peer-behind")
	testutil.RunGitCmd(t, filepath.Dir(local), "clone", remote, peer)
	testutil.RunGitCmd(t, peer, "checkout", "trail")
	if err := os.WriteFile(filepath.Join(peer, "ahead.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, peer, "add", "ahead.txt")
	testutil.RunGitCmd(t, peer, "commit", "-m", "remote moves")
	testutil.RunGitCmd(t, peer, "push", "origin", "trail")

	testutil.RunGitCmd(t, local, "fetch", "origin")
	testutil.RunGitCmd(t, local, "checkout", main)

	headBefore := testutil.GetCurrentSHA(t, local)
	if err := executePushKnownBranches(local, "origin", "new-branch"); err != nil {
		t.Fatalf("behind-remote must not error: %v", err)
	}
	if testutil.GetCurrentSHA(t, local) != headBefore {
		t.Fatal("HEAD must not move")
	}
	afterMain, err := git.GetCurrentBranch(local)
	if err != nil || afterMain != main {
		t.Fatalf("still on %s, got %s err=%v", main, afterMain, err)
	}
}

func TestExecutePushKnownBranches_DivergedCreatesBackupRef(t *testing.T) {
	remote := testutil.CreateBareRemote(t, "origin")
	local := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:    "local",
		Remotes: map[string]string{"origin": remote},
		Files:   map[string]string{"x.txt": "x"},
	})
	main, err := git.GetCurrentBranch(local)
	if err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "push", "-u", "origin", main)

	testutil.RunGitCmd(t, local, "checkout", "-b", "topic")
	if err := os.WriteFile(filepath.Join(local, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "add", "a.txt")
	testutil.RunGitCmd(t, local, "commit", "-m", "topic1")
	testutil.RunGitCmd(t, local, "push", "-u", "origin", "topic")

	peer := filepath.Join(t.TempDir(), "peer-div")
	testutil.RunGitCmd(t, filepath.Dir(local), "clone", remote, peer)
	testutil.RunGitCmd(t, peer, "checkout", "topic")
	if err := os.WriteFile(filepath.Join(peer, "r.txt"), []byte("r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, peer, "add", "r.txt")
	testutil.RunGitCmd(t, peer, "commit", "-m", "remote only")
	testutil.RunGitCmd(t, peer, "push", "origin", "topic")

	testutil.RunGitCmd(t, local, "fetch", "origin")
	testutil.RunGitCmd(t, local, "checkout", "topic")
	if err := os.WriteFile(filepath.Join(local, "l.txt"), []byte("l\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "add", "l.txt")
	testutil.RunGitCmd(t, local, "commit", "-m", "local only")
	topicTip, err := git.GetCommitSHA(local, "topic")
	if err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "checkout", main)

	if err := executePushKnownBranches(local, "origin", "new-branch"); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "fetch", "origin")
	remoteTip, _ := git.GetCommitSHA(local, "origin/topic")
	peerTip, _ := git.GetCommitSHA(peer, "topic")
	if remoteTip != peerTip {
		t.Fatalf("shared branch must not move: %s vs %s", remoteTip, peerTip)
	}
	lsOut := gitOut(t, local, "ls-remote", "origin", "git-fire-backup-topic*")
	if !strings.Contains(lsOut, "git-fire-backup-topic-") {
		t.Fatalf("missing backup on remote: %q", lsOut)
	}
	fields := strings.Fields(strings.TrimSpace(strings.Split(lsOut, "\n")[0]))
	if len(fields) < 1 || fields[0] != topicTip {
		t.Fatalf("backup tip want %s got line %q", topicTip, lsOut)
	}
}

func TestExecutePushKnownBranches_DivergedAbortNoBackup(t *testing.T) {
	remote := testutil.CreateBareRemote(t, "origin")
	local := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:    "local",
		Remotes: map[string]string{"origin": remote},
		Files:   map[string]string{"x.txt": "x"},
	})
	main, errBr := git.GetCurrentBranch(local)
	if errBr != nil {
		t.Fatal(errBr)
	}
	testutil.RunGitCmd(t, local, "push", "-u", "origin", main)
	testutil.RunGitCmd(t, local, "checkout", "-b", "dtopic")
	if err := os.WriteFile(filepath.Join(local, "d.txt"), []byte("d\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "add", "d.txt")
	testutil.RunGitCmd(t, local, "commit", "-m", "d1")
	testutil.RunGitCmd(t, local, "push", "-u", "origin", "dtopic")
	peer := filepath.Join(t.TempDir(), "p2")
	testutil.RunGitCmd(t, filepath.Dir(local), "clone", remote, peer)
	testutil.RunGitCmd(t, peer, "checkout", "dtopic")
	if err := os.WriteFile(filepath.Join(peer, "r2.txt"), []byte("r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, peer, "add", "r2.txt")
	testutil.RunGitCmd(t, peer, "commit", "-m", "r")
	testutil.RunGitCmd(t, peer, "push", "origin", "dtopic")
	testutil.RunGitCmd(t, local, "fetch", "origin")
	testutil.RunGitCmd(t, local, "checkout", "dtopic")
	if err := os.WriteFile(filepath.Join(local, "l2.txt"), []byte("l\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "add", "l2.txt")
	testutil.RunGitCmd(t, local, "commit", "-m", "l")
	testutil.RunGitCmd(t, local, "checkout", main)

	if err := executePushKnownBranches(local, "origin", "abort"); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, local, "fetch", "origin")
	lsOut := gitOut(t, local, "ls-remote", "origin", "git-fire-backup-dtopic*")
	if strings.Contains(lsOut, "git-fire-backup-dtopic") {
		t.Fatalf("abort should not create backup, got %q", lsOut)
	}
}
