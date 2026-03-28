# 🗺️ Git-Fire Implementation Roadmap

**Current Phase:** Phase 2 - Refinement & First Plugins
**Target:** Launch git-fire v1.0 with proven extensibility

---

## Backlog

- [ ] **golangci-lint v2 config migration** — `.golangci.yml` and CI lint job are stubbed out pending full migration from v1 config format. Key work: rename `linters-settings` → `linters.settings`, move `gofmt`/`goimports` to `formatters` block, remove merged linters (`gosimple`, `stylecheck`), re-tune exclusions. Re-enable the `lint` job in `.github/workflows/ci.yml` when complete.

---

## Phase 2: Weeks 5-8 (Current)

### Week 5: Plugin System Foundation

**Goal:** Build plugin infrastructure, ship first plugin

#### Tasks:
- [ ] **Plugin Registry** (`internal/plugins/registry.go`)
  - Plugin interface definition
  - Registration system
  - Lifecycle management (init, validate, execute, cleanup)

- [ ] **Command Plugin** (`internal/plugins/command.go`)
  - Execute external commands
  - Variable substitution ({repo_path}, {timestamp}, etc.)
  - Timeout handling
  - Error capture

- [ ] **Plugin Loader** (`internal/plugins/loader.go`)
  - Load from config
  - Validate plugins before execution
  - Dependency checking

- [ ] **S3 Upload Plugin** (first real plugin!)
  - As command plugin: `aws s3 sync`
  - Config validation
  - Integration test
  - Documentation

**Deliverables:**
- Plugin system working
- S3 plugin tested and documented
- Example in README

**Success Criteria:**
- Can run: `git-fire --plugin s3-upload`
- S3 plugin successfully uploads a repo
- Config-driven plugin loading works

---

### Week 6: More Plugins + Backup Mode Design

**Goal:** Prove plugin architecture, design backup mode

#### Tasks:
- [ ] **Slack Notification Plugin**
  - Webhook plugin type
  - Template system for messages
  - Error handling
  - Example config

- [ ] **Local Backup Plugin**
  - rsync or cp-based
  - Timestamped backups
  - Compression option
  - Rotation/cleanup

- [ ] **Webhook Plugin Type** (`internal/plugins/webhook.go`)
  - Generic HTTP POST/GET
  - Header customization
  - Body templating
  - Retry logic

- [ ] **Backup Mode Design**
  - Spec out GitHub API integration
  - Design auto-create repos flow
  - Plan SSH key UI
  - Write design doc

**Deliverables:**
- 3 working plugins (S3, Slack, local)
- Plugin docs complete
- Backup mode design doc

**Success Criteria:**
- All 3 plugins work together
- Can run custom plugin from config
- Backup mode design reviewed

---

### Week 7: Backup Mode Implementation

**Goal:** Ship the killer feature

#### Tasks:
- [ ] **GitHub API Client** (`internal/backup/github.go`)
  - Auth with token
  - Create repo API
  - List repos API
  - Error handling

- [ ] **GitLab API Client** (`internal/backup/gitlab.go`)
  - Same as GitHub
  - Different API patterns

- [ ] **Backup Orchestrator** (`internal/backup/orchestrator.go`)
  - Auto-create repo if not exists
  - Add remote to local repo
  - Push to new remote
  - Generate manifest

- [ ] **Backup Manifest** (`internal/backup/manifest.go`)
  - JSON metadata file
  - Track what was backed up
  - Timestamp, source, destination
  - Reversibility info

- [ ] **CLI Integration**
  - `--backup-to <url>` flag
  - Auto-detect platform (GitHub/GitLab)
  - Prompt for API token if needed

**Deliverables:**
- Backup mode working
- GitHub + GitLab support
- Manifest generation
- Documentation

**Success Criteria:**
- Can run: `git-fire --backup-to github.com:user/backup`
- Repos auto-created on GitHub
- Manifest file generated
- All repos backed up successfully

---

### Week 8: Polish + Launch Prep

**Goal:** Ship v1.0, get first users

#### Tasks:
- [ ] **Fire UI Integration**
  - Wire up existing fire UI (already built!)
  - Add to `--fire` flag
  - Animated flames during push
  - Success/failure animations

- [ ] **Distribution Setup**
  - GitHub Actions for releases
  - Build for Linux (amd64, arm64)
  - Build for macOS (Intel, M1)
  - Build for Windows (if time)
  - Auto-publish to releases

- [ ] **Documentation Polish**
  - Update README with all features
  - Plugin development guide
  - Video demo (screen recording)
  - Examples for common use cases

- [ ] **Launch Materials**
  - Blog post draft
  - HN launch post
  - Twitter announcement thread
  - Reddit post for r/golang

- [ ] **Testing & Fixes**
  - Integration tests
  - User acceptance testing
  - Bug fixes
  - Performance tuning

**Deliverables:**
- git-fire v1.0 released
- Binaries available
- Launch announcement live
- First users onboarded

**Success Criteria:**
- 100+ GitHub stars
- 10+ production users
- 3+ community plugins
- Featured on 1+ tech blog

---

## Phase 3: Months 3-4 (Platform Foundation)

### Month 3: docker-fire

**Goal:** Build second fire tool, prove multi-tool concept

#### High-level tasks:
- [ ] docker-fire MVP
  - List running containers
  - Export containers
  - Backup volumes
  - Export compose files
  - Push to registry

- [ ] Orchestrator design
  - Detector interface
  - Scheduler design
  - Resource management spec

- [ ] Integration testing
  - git-fire + docker-fire together
  - Resource coordination
  - Progress aggregation

**Deliverables:**
- docker-fire working standalone
- Integration with git-fire (manual for now)
- Orchestrator design doc

---

### Month 4: Orchestrator Extraction

**Goal:** Build platform orchestration layer

#### High-level tasks:
- [ ] Detector system
- [ ] Scheduler implementation
- [ ] Executor with progress
- [ ] Unified `fire` CLI
- [ ] Documentation

**Deliverables:**
- Orchestrator working
- `fire` command runs both tools
- Resource management functional

---

## Task Tracking

### Use GitHub Projects

**Columns:**
1. Backlog
2. This Week
3. In Progress
4. Review
5. Done

**Labels:**
- `P0-critical` - Blocker
- `P1-high` - Important
- `P2-medium` - Should have
- `P3-low` - Nice to have
- `plugin` - Plugin related
- `backup-mode` - Backup feature
- `docs` - Documentation
- `bug` - Bug fix

---

## Metrics Dashboard

### Weekly Tracking

| Metric | Week 5 | Week 6 | Week 7 | Week 8 | Target |
|--------|--------|--------|--------|--------|--------|
| GitHub Stars | TBD | TBD | TBD | TBD | 100 |
| Production Users | 0 | TBD | TBD | TBD | 10 |
| Plugins | 0 | 1 | 2 | 3 | 3 |
| Tests Passing | 43 | TBD | TBD | TBD | 60+ |
| Blog Mentions | 0 | TBD | TBD | TBD | 5 |

---

## Risk Register

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Plugin API too rigid | High | Medium | Design for extensibility |
| Backup mode too complex | Medium | Low | Start with GitHub only |
| GitHub API rate limits | Medium | Medium | Token rotation, caching |
| Users don't need plugins | High | Low | S3 plugin proves value |
| Launch gets no traction | High | Medium | Pre-launch marketing |

---

## Dependencies

### External:
- GitHub/GitLab APIs (backup mode)
- AWS SDK (S3 plugin example)
- Slack API (webhook example)

### Internal:
- Plugin system (Week 5) → All other plugins
- Backup mode (Week 7) → GitHub Actions
- Documentation (Week 8) → Launch

---

## Launch Checklist

**Pre-launch (Week 8):**
- [ ] All tests passing
- [ ] Documentation complete
- [ ] Binaries built for major platforms
- [ ] Examples working
- [ ] Blog post ready
- [ ] Social media planned
- [ ] Demo video recorded

**Launch Day:**
- [ ] Publish v1.0 release
- [ ] Post to Hacker News
- [ ] Post to Reddit (r/golang, r/programming)
- [ ] Tweet announcement
- [ ] Post to Dev.to
- [ ] Email to personal network

**Post-launch (Week 9+):**
- [ ] Monitor issues
- [ ] Respond to feedback
- [ ] Fix critical bugs
- [ ] Update docs based on questions
- [ ] Plan Phase 3

---

## Questions to Answer

**Week 5:**
- [ ] Plugin API final design?
- [ ] How to handle plugin dependencies?
- [ ] Plugin versioning strategy?

**Week 6:**
- [ ] Webhook retry policy?
- [ ] Plugin execution timeout defaults?
- [ ] Error handling strategy?

**Week 7:**
- [ ] GitHub vs GitLab - both or GitHub first?
- [ ] Manifest format - JSON or YAML?
- [ ] Repo naming convention for backups?

**Week 8:**
- [ ] Windows support priority?
- [ ] Which platforms for binaries?
- [ ] Launch timing (weekday/time)?

---

**Next Review:** End of Week 5  
**Update Frequency:** Weekly

