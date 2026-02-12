# Roadmap: Git Fire 🔥

**Philosophy:** Ship fast, iterate based on real usage.

---

## Phase 1: MVP (Current - Ship in 2-4 weeks)

**Goal:** Emergency backup tool that just works.

### Core Features
- ✅ Multi-repo scanning (hybrid strategy: cache + quick paths + background indexing)
- ✅ Zero-config operation (works out of box with safe defaults)
- ✅ Interactive prompt with 10-second countdown + ASCII flame animations
- ✅ Auto-commit uncommitted changes (git add -A && commit)
- ✅ Intelligent conflict handling (create fire branches, never force-push)
- ✅ **Dual modes:**
  - **Normal Fire:** Push to existing remotes
  - **Backup Mode:** Push to new remote location (GitHub/GitLab/etc)
- ✅ **Backup to new remote:**
  - Auto-create repos on target (API integration)
  - Repo renaming with templates (hostname, date, etc)
  - Add new remote to each repo (keeps original remotes)
  - Generate backup manifest (JSON metadata)
- ✅ Push to all remotes by default
- ✅ SSH passphrase collection with validation
- ✅ Beautiful TUI (Bubble Tea + Lipgloss)
- ✅ Fire drill mode (dry-run preview)
- ✅ Comprehensive logging (JSON format, reversibility)
- ✅ Config file support (optional TOML)

### CLI
```bash
git-fire              # Interactive mode (fast, uses cache)
git-fire ~/projects   # Scan specific path
git-fire --dry-run    # Fire drill (show what would happen)
git-fire --full-scan  # Full filesystem scan
git-fire --init       # Generate config template
```

### Testing
- Unit tests for core logic
- Integration tests with test repos
- Manual testing on Linux + macOS

### Success Criteria
- Works on Linux and macOS
- Handles 100+ repos in < 2 minutes
- Zero data loss in all scenarios
- 5+ beta testers validate

**Release Target:** v0.1.0 - End of February 2026

---

## Phase 2: Polish & Launch (Weeks 5-8)

**Goal:** Make it easy to discover and install.

### Distribution
- Homebrew formula
- Debian/Ubuntu apt
- Arch AUR
- `go install` support
- Windows support (if demand)

### Features
- Performance optimizations
- Better error messages
- Log rotation
- Remote health checks

### Marketing
- HackerNews launch
- Reddit r/ProgrammerHumor
- Blog post + demo video
- Package manager listings

**Release Target:** v1.0.0 - End of March 2026

---

## Phase 3: Community (Months 3-6)

**Goal:** Build community and add requested features.

### Potential Features (Based on Feedback)
- IDE integrations (VS Code, JetBrains)
- Scheduled/background backups
- Proper recursive submodule handling
- GitHub auto-create repos
- Team features (shared configs, compliance reporting)

### Community
- Accept PRs enthusiastically
- Good first issue labels
- Contributor recognition

**Release Target:** v1.5.0+ - Mid 2026

---

## Out of Scope (For MVP)

- Full backup solution (use Time Machine, Backblaze)
- Git hosting replacement
- CI/CD pipeline
- Code review tools
- Project management

**Focus:** Do ONE thing perfectly - emergency git backup.

---

## Current Status

**Phase:** MVP Development (spec complete)
**Next:** Start implementation
**ETA:** v0.1.0 beta by end of February 2026

