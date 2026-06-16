package executor

import (
	"strings"
	"testing"

	"github.com/git-fire/git-harness/git"
	testutil "github.com/git-fire/git-testkit"
)

func TestSummarizePushKnownRemote_UnauthenticatedHTTPSFailsWithoutPrompt(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "https-push-known-repo",
		Remotes: map[string]string{
			"origin": "https://github.com/git-fire/nonexistent-repo-auth-test.git",
		},
	})

	_, err := summarizePushKnownRemote(repo, "origin")
	if err == nil {
		t.Fatal("expected summarizePushKnownRemote to fail without credentials")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "terminal prompts disabled") &&
		!strings.Contains(msg, "could not read username") {
		t.Fatalf("expected non-interactive auth failure, got: %v", err)
	}
}

func TestDetectConflict_UnauthenticatedHTTPSFailsWithoutPrompt(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "https-detect-conflict-repo",
		Remotes: map[string]string{
			"origin": "https://github.com/git-fire/nonexistent-repo-auth-test.git",
		},
	})
	main, err := git.GetCurrentBranch(repo)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = git.DetectConflict(repo, main, "origin")
	if err == nil {
		t.Fatal("expected DetectConflict fetch to fail without credentials")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "terminal prompts disabled") &&
		!strings.Contains(msg, "could not read username") {
		t.Fatalf("expected non-interactive auth failure, got: %v", err)
	}
}
