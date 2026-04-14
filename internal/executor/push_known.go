package executor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-fire/git-harness/git"
)

// gitDirPresent returns true if path looks like a git work tree (has .git file or dir).
func gitDirPresent(repoPath string) bool {
	_, err := os.Stat(filepath.Join(repoPath, ".git"))
	return err == nil
}

// push-known orchestration lives in git-fire; git-harness exposes only primitives
// (fetch, list branches, ancestry, create/push refs).

type knownBranchClass int

const (
	knownBranchEqual knownBranchClass = iota
	knownBranchAheadFastForward
	knownBranchBehindRemote
	knownBranchDiverged
)

func classifyKnownBranchAgainstRemote(repoPath, remote, branch string) (knownBranchClass, error) {
	localSHA, err := git.GetCommitSHA(repoPath, branch)
	if err != nil {
		return 0, fmt.Errorf("local branch %q: %w", branch, err)
	}
	remoteRef := remote + "/" + branch
	remoteSHA, err := git.GetCommitSHA(repoPath, remoteRef)
	if err != nil {
		return 0, fmt.Errorf("remote ref %q: %w", remoteRef, err)
	}
	if localSHA == remoteSHA {
		return knownBranchEqual, nil
	}

	localAncestorOfRemote, err := git.RefIsAncestor(repoPath, branch, remoteRef)
	if err != nil {
		return 0, err
	}
	remoteAncestorOfLocal, err := git.RefIsAncestor(repoPath, remoteRef, branch)
	if err != nil {
		return 0, err
	}

	switch {
	case localAncestorOfRemote && !remoteAncestorOfLocal:
		return knownBranchBehindRemote, nil
	case remoteAncestorOfLocal && !localAncestorOfRemote:
		return knownBranchAheadFastForward, nil
	case remoteAncestorOfLocal && localAncestorOfRemote:
		return knownBranchEqual, nil
	default:
		return knownBranchDiverged, nil
	}
}

// pushKnownRemoteSummary counts how local branches that exist on the remote
// relate to remote tips (after fetch).
type pushKnownRemoteSummary struct {
	Equal   int
	AheadFF int
	Behind  int
	Diverged int
}

func summarizePushKnownRemote(repoPath, remote string) (pushKnownRemoteSummary, error) {
	var out pushKnownRemoteSummary
	if err := git.FetchRemote(repoPath, remote); err != nil {
		return out, err
	}
	remoteBranches, err := git.ListRemoteBranches(repoPath, remote)
	if err != nil {
		return out, fmt.Errorf("list remote branches: %w", err)
	}
	localBranches, err := git.ListLocalBranches(repoPath)
	if err != nil {
		return out, fmt.Errorf("list local branches: %w", err)
	}

	remoteSet := make(map[string]struct{}, len(remoteBranches))
	for _, b := range remoteBranches {
		remoteSet[b] = struct{}{}
	}

	for _, branch := range localBranches {
		if _, ok := remoteSet[branch]; !ok {
			continue
		}
		c, err := classifyKnownBranchAgainstRemote(repoPath, remote, branch)
		if err != nil {
			return out, fmt.Errorf("classify %q: %w", branch, err)
		}
		switch c {
		case knownBranchEqual:
			out.Equal++
		case knownBranchAheadFastForward:
			out.AheadFF++
		case knownBranchBehindRemote:
			out.Behind++
		case knownBranchDiverged:
			out.Diverged++
		}
	}
	return out, nil
}

func normalizePushKnownConflictStrategy(strategy string) string {
	if strings.TrimSpace(strategy) == "abort" {
		return "abort"
	}
	return "new-branch"
}

// executePushKnownBranches runs push-known-branches backup rules for one remote.
func executePushKnownBranches(repoPath, remote, conflictStrategy string) error {
	strategy := normalizePushKnownConflictStrategy(conflictStrategy)

	if err := git.FetchRemote(repoPath, remote); err != nil {
		return err
	}
	remoteBranches, err := git.ListRemoteBranches(repoPath, remote)
	if err != nil {
		return fmt.Errorf("failed to get remote branches: %w", err)
	}
	localBranches, err := git.ListLocalBranches(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get local branches: %w", err)
	}

	remoteSet := make(map[string]struct{}, len(remoteBranches))
	for _, b := range remoteBranches {
		remoteSet[b] = struct{}{}
	}

	var errs []error
	for _, branch := range localBranches {
		if _, ok := remoteSet[branch]; !ok {
			fmt.Fprintf(os.Stderr, "warning: branch '%s' has no remote tracking ref — not backed up\n", branch)
			continue
		}

		class, err := classifyKnownBranchAgainstRemote(repoPath, remote, branch)
		if err != nil {
			errs = append(errs, fmt.Errorf("branch %s: %w", branch, err))
			continue
		}

		switch class {
		case knownBranchEqual:
			continue
		case knownBranchBehindRemote:
			fmt.Fprintf(os.Stderr, "warning: branch '%s' is behind %s/%s — skipping push (remote already has your local commits)\n", branch, remote, branch)
			continue
		case knownBranchAheadFastForward:
			if err := git.PushBranch(repoPath, remote, branch); err != nil {
				errs = append(errs, fmt.Errorf("branch %s: %w", branch, err))
			}
		case knownBranchDiverged:
			if strategy == "abort" {
				fmt.Fprintf(os.Stderr, "warning: branch '%s' diverged from %s/%s — skipping push (conflict_strategy=abort)\n", branch, remote, branch)
				continue
			}
			localSHA, shaErr := git.GetCommitSHA(repoPath, branch)
			if shaErr != nil {
				errs = append(errs, fmt.Errorf("branch %s: %w", branch, shaErr))
				continue
			}
			fireName, fbErr := git.CreateFireBranch(repoPath, branch, localSHA)
			if fbErr != nil {
				errs = append(errs, fmt.Errorf("branch %s: backup: %w", branch, fbErr))
				continue
			}
			if err := git.PushBranch(repoPath, remote, fireName); err != nil {
				errs = append(errs, fmt.Errorf("branch %s: push backup %s: %w", branch, fireName, err))
			}
		}
	}

	return errors.Join(errs...)
}
