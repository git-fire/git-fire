# Bugbot review guidance
## Autonomy, cross-repo, and history (reviewers)

- **Autonomy first:** Prefer review feedback that keeps agents unblocked. Ask for repository splits or extra ceremony only when risk truly warrants it.
- **Multi-repo efforts:** A change may be intentionally split across repositories; read the PR body for links to sibling PRs before calling the work incomplete for only touching one repo.
- **Force-push:** Never suggest force-push as a silent or automatic fix. If history rewrite is raised, require clear **user-visible** justification: **why** it is proposed, **blast radius** and effects on others and CI, and why non-destructive options are insufficient—default to safer paths.
