package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/git-fire/git-fire/internal/auth"
	"github.com/git-fire/git-fire/internal/executor"
	"github.com/git-fire/git-harness/git"
)

func TestSSHStatusToJSON(t *testing.T) {
	sshStatus := &auth.SSHStatus{
		AvailableKeys: []auth.SSHKey{{Name: "id_ed25519", Type: "ed25519", Path: "/home/u/.ssh/id_ed25519"}},
		Agent:         &auth.SSHAgent{Running: true, Socket: "/tmp/agent", KeysKnown: true, Keys: []auth.SSHKey{}},
	}
	out := sshStatusToJSON(sshStatus)
	if !out.AgentRunning || out.AgentSocket != "/tmp/agent" {
		t.Fatalf("agent fields: %+v", out)
	}
	if len(out.AvailableKeys) != 1 || out.AvailableKeys[0].Name != "id_ed25519" {
		t.Fatalf("keys: %+v", out.AvailableKeys)
	}
}

func TestBuildStatusScanJSON(t *testing.T) {
	repos := []git.Repository{
		{Path: "/tmp/a", Name: "a", IsDirty: true},
		{Path: "/tmp/b", Name: "b", IsDirty: false},
	}
	scan := buildStatusScanJSON("/tmp", true, true, repos)
	if scan.ReposFound != 2 || scan.DirtyRepos != 1 || !scan.DisableScan || !scan.NoScanThisRun {
		t.Fatalf("scan: %+v", scan)
	}
}

func TestExecutionResultToJSON_roundTrip(t *testing.T) {
	result := &executor.ExecutionResult{
		Success:  1,
		Failed:   0,
		Skipped:  0,
		Duration: 2 * time.Second,
		RepoResults: []executor.RepoResult{
			{Path: "/r", Success: true, Duration: time.Second, Actions: []executor.Action{
				{Type: executor.ActionPushBranch, Description: "push", Branch: "main"},
			}},
		},
	}
	out := executionResultToJSON(result)
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), `"push-branch"`) {
		t.Fatalf("expected action type in JSON: %s", data)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && stringIndex(s, sub) >= 0))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
