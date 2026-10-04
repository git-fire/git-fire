package executor

import (
	"testing"
	"time"

	"github.com/git-fire/git-harness/git"
	testutil "github.com/git-fire/git-testkit"
)

const unauthenticatedNetworkTimeout = 15 * time.Second

func TestSummarizePushKnownRemote_UnauthenticatedHTTPSFailsWithoutPrompt(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "https-push-known-repo",
		Remotes: map[string]string{
			"origin": "https://github.com/git-fire/nonexistent-repo-auth-test.git",
		},
	})

	start := time.Now()
	_, err := summarizePushKnownRemote(repo, "origin")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected summarizePushKnownRemote to fail without credentials")
	}
	if elapsed > unauthenticatedNetworkTimeout {
		t.Errorf("summarizePushKnownRemote should fail fast without prompting, took %v", elapsed)
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

	start := time.Now()
	_, _, _, err = git.DetectConflict(repo, main, "origin")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected DetectConflict fetch to fail without credentials")
	}
	if elapsed > unauthenticatedNetworkTimeout {
		t.Errorf("DetectConflict fetch should fail fast without prompting, took %v", elapsed)
	}
}
