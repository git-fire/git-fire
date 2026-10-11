package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/git-fire/git-fire/internal/auth"
	"github.com/git-fire/git-fire/internal/executor"
	"github.com/git-fire/git-harness/git"
)

const jsonOutputSchemaVersion = 1

// statusJSONDocument is the machine-readable output for `git-fire --status --json`.
type statusJSONDocument struct {
	SchemaVersion int              `json:"schema_version"`
	CLIVersion    string           `json:"cli_version"`
	SSH           sshStatusJSON    `json:"ssh"`
	Scan          statusScanJSON   `json:"scan"`
}

type sshStatusJSON struct {
	AgentRunning bool         `json:"agent_running"`
	AgentSocket  string       `json:"agent_socket,omitempty"`
	AgentKeysKnown bool       `json:"agent_keys_known"`
	AgentKeysLoaded int       `json:"agent_keys_loaded"`
	AgentError   string       `json:"agent_error,omitempty"`
	AvailableKeys []sshKeyJSON `json:"available_keys"`
}

type sshKeyJSON struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Path              string `json:"path"`
	LoadedInAgent     bool   `json:"loaded_in_agent"`
	NeedsPassphrase   bool   `json:"needs_passphrase"`
	FingerprintError  string `json:"fingerprint_error,omitempty"`
}

type statusScanJSON struct {
	ScanPath      string           `json:"scan_path"`
	DisableScan   bool             `json:"disable_scan"`
	NoScanThisRun bool             `json:"no_scan_this_run"`
	ReposFound    int              `json:"repos_found"`
	DirtyRepos    int              `json:"dirty_repos"`
	Repositories  []repoStatusJSON `json:"repositories"`
}

type repoStatusJSON struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	IsDirty bool   `json:"is_dirty"`
}

// runSummaryJSONDocument is the machine-readable run summary for `--json` backup/dry-run flows.
type runSummaryJSONDocument struct {
	SchemaVersion int                    `json:"schema_version"`
	CLIVersion    string                 `json:"cli_version"`
	DryRun        bool                   `json:"dry_run"`
	Plan          *planSummaryJSON       `json:"plan,omitempty"`
	Result        *executionSummaryJSON  `json:"result,omitempty"`
	LogPath       string                 `json:"log_path,omitempty"`
}

type planSummaryJSON struct {
	TotalRepos   int `json:"total_repos"`
	DirtyRepos   int `json:"dirty_repos"`
	Conflicts    int `json:"conflicts"`
	FireBranches int `json:"fire_branches"`
}

type executionSummaryJSON struct {
	Success       int                `json:"success"`
	Failed        int                `json:"failed"`
	Skipped       int                `json:"skipped"`
	Duration      string             `json:"duration"`
	TotalActions  int                `json:"total_actions"`
	FailedActions int                `json:"failed_actions"`
	RepoResults   []repoResultJSON   `json:"repo_results,omitempty"`
}

type repoResultJSON struct {
	Path           string         `json:"path"`
	Success        bool           `json:"success"`
	Error          string         `json:"error,omitempty"`
	PushedBranches []string       `json:"pushed_branches,omitempty"`
	Duration       string         `json:"duration"`
	Actions        []actionJSON   `json:"actions,omitempty"`
}

type actionJSON struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Remote      string `json:"remote,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Error       string `json:"error,omitempty"`
}

func writeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func emitStatusJSON(sshStatus *auth.SSHStatus, cfgScanPath string, disableScan, noScanRun bool, repos []git.Repository) error {
	doc := statusJSONDocument{
		SchemaVersion: jsonOutputSchemaVersion,
		CLIVersion:    resolvedCLIVersion(Version),
		SSH:           sshStatusToJSON(sshStatus),
		Scan:          buildStatusScanJSON(cfgScanPath, disableScan, noScanRun, repos),
	}
	return writeJSON(os.Stdout, doc)
}

func sshStatusToJSON(s *auth.SSHStatus) sshStatusJSON {
	out := sshStatusJSON{
		AvailableKeys: make([]sshKeyJSON, 0, len(s.AvailableKeys)),
	}
	if s.Agent != nil {
		out.AgentRunning = s.Agent.Running
		out.AgentSocket = s.Agent.Socket
		out.AgentKeysKnown = s.Agent.KeysKnown
		if s.Agent.KeysKnown {
			out.AgentKeysLoaded = len(s.Agent.Keys)
		}
		out.AgentError = s.Agent.Error
	}
	for _, k := range s.AvailableKeys {
		out.AvailableKeys = append(out.AvailableKeys, sshKeyJSON{
			Name:             k.Name,
			Type:             k.Type,
			Path:             k.Path,
			LoadedInAgent:    k.IsLoaded,
			NeedsPassphrase:  k.NeedsPass,
			FingerprintError: k.FingerprintError,
		})
	}
	return out
}

func buildStatusScanJSON(scanPath string, disableScan, noScanRun bool, repos []git.Repository) statusScanJSON {
	dirty := 0
	repoRows := make([]repoStatusJSON, 0, len(repos))
	for _, r := range repos {
		if r.IsDirty {
			dirty++
		}
		repoRows = append(repoRows, repoStatusJSON{
			Path:    r.Path,
			Name:    r.Name,
			IsDirty: r.IsDirty,
		})
	}
	return statusScanJSON{
		ScanPath:      scanPath,
		DisableScan:   disableScan,
		NoScanThisRun: noScanRun,
		ReposFound:    len(repos),
		DirtyRepos:    dirty,
		Repositories:  repoRows,
	}
}

func emitRunSummaryJSON(dryRun bool, plan *executor.PushPlan, result *executor.ExecutionResult, logPath string) error {
	doc := runSummaryJSONDocument{
		SchemaVersion: jsonOutputSchemaVersion,
		CLIVersion:    resolvedCLIVersion(Version),
		DryRun:        dryRun,
		LogPath:       logPath,
	}
	if plan != nil {
		doc.Plan = &planSummaryJSON{
			TotalRepos:   plan.TotalRepos,
			DirtyRepos:   plan.DirtyRepos,
			Conflicts:    plan.Conflicts,
			FireBranches: plan.FireBranches,
		}
	}
	if result != nil {
		doc.Result = executionResultToJSON(result)
	}
	return writeJSON(os.Stdout, doc)
}

func executionResultToJSON(result *executor.ExecutionResult) *executionSummaryJSON {
	out := &executionSummaryJSON{
		Success:       result.Success,
		Failed:        result.Failed,
		Skipped:       result.Skipped,
		Duration:      result.Duration.String(),
		TotalActions:  result.TotalActions,
		FailedActions: result.FailedActions,
	}
	if len(result.RepoResults) > 0 {
		out.RepoResults = make([]repoResultJSON, 0, len(result.RepoResults))
		for _, rr := range result.RepoResults {
			row := repoResultJSON{
				Path:           rr.Path,
				Success:        rr.Success,
				PushedBranches: rr.PushedBranches,
				Duration:       rr.Duration.String(),
			}
			if rr.Error != nil {
				row.Error = rr.Error.Error()
			}
			for _, a := range rr.Actions {
				aj := actionJSON{
					Type:        a.Type.String(),
					Description: a.Description,
					Remote:      a.Remote,
					Branch:      a.Branch,
				}
				if a.Error != nil {
					aj.Error = a.Error.Error()
				}
				row.Actions = append(row.Actions, aj)
			}
			out.RepoResults = append(out.RepoResults, row)
		}
	}
	return out
}

// cliPrintln writes human-oriented lines to stderr when --json is set so stdout stays machine-readable.
func cliPrintln(args ...interface{}) {
	w := os.Stdout
	if jsonOutput {
		w = os.Stderr
	}
	fmt.Fprintln(w, args...)
}

func cliPrintf(format string, args ...interface{}) {
	w := os.Stdout
	if jsonOutput {
		w = os.Stderr
	}
	fmt.Fprintf(w, format, args...)
}
