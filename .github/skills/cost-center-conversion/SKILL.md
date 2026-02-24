---
name: convert-cost-center-to-gh-extension
description: 'All steps to convert a cost center project from Python to a GitHub extension in Go, delivered across 8 PRs.'
allowed-tools: Read Edit Bash(git:*) Bash(gh issue:*) Bash(gh pr:*) Bash(go:*) Bash(gofmt:*) Bash(golangci-lint:*) Bash(make:*) Bash(mkdir:*) Bash(mv:*) Bash(rm:*) Bash(cp:*) Bash(cat:*) Bash(ls:*) Bash(find:*) Bash(sed:*) Bash(grep:*) Bash(echo:*)
---

# Convert cost-center-automation → gh-cost-center (Go gh Extension)

Convert the Python `cost-center-automation` CLI into `gh-cost-center`, a precompiled Go-based GitHub CLI extension. The repo is renamed to `gh-cost-center`, uses `cobra` for CLI, `go-gh` for authentication (inheriting `gh auth login`), and `gopkg.in/yaml.v3` for config. Incomplete features (file-based cache, `filter_users_by_timestamp`, repo plan mode) are implemented in Go with follow-up issues. Delivered across **8 pull requests**, one per milestone, with Python code removed in the final PR.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Auth model | `go-gh` only | Inherits `gh auth login` — idiomatic for gh extensions |
| CLI framework | Cobra | Standard in Go CLI ecosystem, used by `gh` itself |
| Repo/command name | `gh-cost-center` → `gh cost-center` | Required `gh-` prefix for gh extensions |
| Batch size | 50 users per API call | Preserved from Python version (GitHub API limit) |
| Missing features | Implement in Go + follow-up issues | Full parity plus gap-fill |
| Python removal | Final PR (PR 8) | Clean cutover after all Go functionality is validated |
| Config format | Keep YAML (`config/config.yaml`) | Users keep their existing config files |

---

## PR 1: Go Project Scaffold + CLI Skeleton

**Branch**: `go/scaffold`
**Goal**: Bootstrap the Go module, Cobra command tree, and CI/CD for precompiled binary releases.

### Steps

1. Initialize Go module: `go mod init github.com/renan-alm/gh-cost-center`
2. Create the Cobra command tree in `cmd/`:
   - `cmd/root.go` — root command with `--config`, `--verbose` global flags
   - `cmd/assign.go` — `gh cost-center assign` with `--mode plan|apply`, `--yes`, `--teams`, `--repo`, `--users`, `--incremental`, `--create-cost-centers`, `--create-budgets`, `--check-current`
   - `cmd/list_users.go` — `gh cost-center list-users`
   - `cmd/config.go` — `gh cost-center config` (show config)
   - `cmd/report.go` — `gh cost-center report` with `--teams`
   - `cmd/cache.go` — `gh cost-center cache` with `--stats`, `--clear`, `--cleanup`
   - `cmd/version.go` — `gh cost-center version`
3. Create `main.go` as the entry point calling `cmd.Execute()`
4. Add `gh-extension-precompile` GitHub Actions workflow for cross-compiled release binaries (`linux-amd64`, `darwin-amd64`, `darwin-arm64`, `windows-amd64`)
5. Update `.gitignore` with Go build artifacts, `gh-cost-center` binary
6. Add a `Makefile` with `build`, `install` (via `gh extension install .`), `test`, `lint` targets

### Verification

```bash
go build -o gh-cost-center .
gh extension install .
gh cost-center --help         # prints all subcommands/flags
gh cost-center assign --help  # prints assign-specific flags
gh cost-center version        # prints version
```

---

## PR 2: Configuration System

**Branch**: `go/config`
**Goal**: Port the full YAML config loader, env var overrides, backward-compatible keys, and validation.

### Steps

1. Create `internal/config/config.go` — `ConfigManager` struct with `Load(path string) (*Config, error)`
2. Create `internal/config/models.go` — typed config structs mirroring YAML:
   - `Config`, `GitHubConfig`, `CostCentersConfig`, `TeamsConfig`, `BudgetsConfig`, `LoggingConfig`, `RepositoryConfig`, `ExplicitMapping`
3. Port backward-compatible key fallback chains (e.g., `no_prus_cost_center` → `no_prus_cost_center_id`, `remove_orphaned_users` → `remove_users_no_longer_in_teams`)
4. Port API URL validation supporting standard GitHub, GHE Data Resident (`api.*.ghe.com`), and GHE Server (`*/api/v3`)
5. Port env var overrides: `GITHUB_ENTERPRISE`, `GITHUB_API_BASE_URL` (no `GITHUB_TOKEN` — auth is via `go-gh`)
6. Port placeholder detection (warn if `REPLACE_WITH_*` values found)
7. Port `check_config_warnings()` logic
8. Port `load_last_run_timestamp()` / `save_last_run_timestamp()` (JSON file at `exports/.last_run_timestamp`)
9. Wire config loading into `cmd/root.go` `PersistentPreRunE`
10. Port `--show-config` display logic to `cmd/config.go`

### Config YAML Structure (preserved)

```yaml
github:
  enterprise: "string"
  api_base_url: "string"
  cost_centers:
    mode: "users|teams|repository"
    repository_config:
      explicit_mappings:
        - cost_center: "string"
          property_name: "string"
          property_values: ["string"]

logging:
  level: "INFO|DEBUG|WARNING|ERROR"
  file: "logs/copilot_manager.log"

cost_centers:
  no_prus_cost_center_id: "string"
  prus_allowed_cost_center_id: "string"
  prus_exception_users: ["string"]
  auto_create: bool
  no_prus_cost_center_name: "string"
  prus_allowed_cost_center_name: "string"
  enable_incremental: bool

teams:
  enabled: bool
  scope: "organization|enterprise"
  mode: "auto|manual"
  organizations: ["string"]
  auto_create_cost_centers: bool
  remove_users_no_longer_in_teams: bool
  team_mappings:
    "org/team-slug": "cost_center_id_or_name"

budgets:
  enabled: bool
  products:
    <product_name>:
      amount: int
      enabled: bool
```

### Backward-Compatible Config Key Fallback Chains

| New key | Old key (fallback) |
|---------|-------------------|
| `no_prus_cost_center_id` | `no_prus_cost_center` |
| `prus_allowed_cost_center_id` | `prus_allowed_cost_center` |
| `no_prus_cost_center_name` | `no_pru_name` |
| `prus_allowed_cost_center_name` | `pru_allowed_name` |
| `remove_users_no_longer_in_teams` | `remove_orphaned_users` |

### Verification

```bash
go test ./internal/config/... -v
# Tests: config.example.yaml loading, fallback chains, env var overrides,
#        API URL validation, placeholder warnings
```

---

## PR 3: GitHub API Client + Logging

**Branch**: `go/api-client`
**Goal**: Port all GitHub API interactions and the logging infrastructure.

### Steps

1. Create `internal/github/client.go` — HTTP client using `go-gh` for authentication:
   - `NewClient(config)` — constructs client with `go-gh` REST client, retry logic (3 retries, backoff on 429/500/502/503/504), custom `User-Agent: gh-cost-center`
   - Rate-limit handling: read `X-RateLimit-Reset` header, sleep & retry
2. Create `internal/github/copilot.go` — Copilot seat management:
   - `GetCopilotUsers()` — paginated `GET /enterprises/{enterprise}/copilot/billing/seats` (100/page)
   - `FilterUsersByTimestamp(users, timestamp)` — filter users by `created_at` > timestamp (implement the currently-missing function)
   - User deduplication by login
3. Create `internal/github/costcenters.go` — Cost center CRUD:
   - `GetAllActiveCostCenters()` — `GET /enterprises/{enterprise}/settings/billing/cost-centers`
   - `GetCostCenter(id)` — single CC details
   - `CreateCostCenter(name)` — `POST` with 409 conflict handling (UUID regex extraction from error message)
   - `AddUsersToCostCenter(id, usernames)` — `POST .../resource` with batch-of-50 chunking
   - `RemoveUsersFromCostCenter(id, usernames)` — `DELETE .../resource`
   - `CheckUserCostCenterMembership(username)` — membership check endpoint
   - `AddRepositoriesToCostCenter(id, repoNames)` — batch repo assignment
   - `EnsureCostCentersExist(name1, name2)` — create-or-find with preload optimization
4. Create `internal/github/teams.go` — Teams API:
   - `GetOrgTeams(org)`, `GetOrgTeamMembers(org, slug)` — org-level
   - `GetEnterpriseTeams(enterprise)`, `GetEnterpriseTeamMembers(enterprise, slug)` — enterprise-level
5. Create `internal/github/repos.go` — Repository/custom property APIs:
   - `GetOrgReposWithProperties(org)`, `GetRepoProperties(owner, repo)`
6. Create `internal/github/budgets.go` — Budget APIs:
   - `ListBudgets()`, `CreateBudget()` with `BudgetsAPIUnavailableError` handling
   - Port `_get_budget_type_and_sku()` product→SKU mapping
7. Create `internal/logging/logger.go`:
   - Structured logger using `log/slog` (Go stdlib): console handler (stdout) + file handler (rotating)
   - Log levels: INFO default, DEBUG on `--verbose`
   - SIGPIPE graceful handling

### GitHub API Endpoints (complete reference)

| Method | Endpoint | Function |
|--------|----------|----------|
| GET | `/enterprises/{enterprise}/copilot/billing/seats` | `GetCopilotUsers` |
| GET | `/enterprises/{enterprise}/settings/billing/cost-centers` | `GetAllActiveCostCenters` |
| GET | `/enterprises/{enterprise}/settings/billing/cost-centers/{id}` | `GetCostCenter` |
| POST | `/enterprises/{enterprise}/settings/billing/cost-centers` | `CreateCostCenter` |
| POST | `/enterprises/{enterprise}/settings/billing/cost-centers/{id}/resource` | `AddUsersToCostCenter` / `AddRepositoriesToCostCenter` |
| DELETE | `/enterprises/{enterprise}/settings/billing/cost-centers/{id}/resource` | `RemoveUsersFromCostCenter` |
| GET | `/enterprises/{enterprise}/settings/billing/cost-centers/memberships?resource_type=user&name={username}` | `CheckUserCostCenterMembership` |
| GET | `/enterprises/{enterprise}/settings/billing/budgets` | `ListBudgets` |
| POST | `/enterprises/{enterprise}/settings/billing/budgets` | `CreateBudget` |
| GET | `/enterprises/{enterprise}/teams` | `GetEnterpriseTeams` |
| GET | `/enterprises/{enterprise}/teams/{slug}/memberships` | `GetEnterpriseTeamMembers` |
| GET | `/orgs/{org}/teams` | `GetOrgTeams` |
| GET | `/orgs/{org}/teams/{slug}/members` | `GetOrgTeamMembers` |
| GET | `/orgs/{org}/properties/schema` | `GetOrgPropertySchema` |
| GET | `/orgs/{org}/properties/values` | `GetOrgReposWithProperties` |
| GET | `/repos/{owner}/{repo}/properties/values` | `GetRepoProperties` |
| GET | `/rate_limit` | `CheckRateLimit` |

### Common Headers

```
Accept: application/vnd.github+json
User-Agent: gh-cost-center
X-GitHub-Api-Version: 2022-11-28
```

### Logging Conventions

| Level | Use for |
|-------|---------|
| INFO | Progress/summaries (e.g., "Total users found: 42") |
| DEBUG | Detailed operations/pagination (e.g., "Fetched page 3") |
| WARN | Skipped items/config gaps (e.g., "User not found in any team") |
| ERROR | Failures (e.g., "API call failed after 3 retries") |

### Verification

```bash
go test ./internal/github/... -v
go test ./internal/logging/... -v
# Tests: pagination, retry, rate limiting, 409 conflict handling, batch chunking,
#        FilterUsersByTimestamp, deduplication
```

---

## PR 4: PRU-Based Mode (Default)

**Branch**: `go/pru-mode`
**Goal**: Port the core PRU-based cost center assignment flow — the default mode.

### Steps

1. Create `internal/pru/manager.go` — `PRUManager` struct:
   - `AssignCostCenter(user)` — if login ∈ exception list → PRU-allowed CC, else → no-PRU CC
   - `GenerateSummary(users)` — count by cost center
2. Wire into `cmd/assign.go` — the default (no `--teams`, no `--repo`) flow:
   - Fetch Copilot users → deduplicate → optional `--users` filter → optional `--incremental` filter
   - Build `{cc_id: [usernames]}` map
   - Plan mode: display assignments without making changes
   - Apply mode: `BulkUpdateCostCenterAssignments()` (batches of 50)
   - Port confirmation prompt (`type 'apply'` unless `--yes`)
   - Port `--create-cost-centers` flow (auto-create CCs in apply mode)
   - Port `--check-current-cost-center` flag logic (`ignore_current_cost_center` inversion)
3. Wire `cmd/list_users.go` — fetch and display users with `[PRUs Exception]` markers
4. Wire `cmd/report.go` — generate and display summary
5. Port `_show_success_summary()` with cost center URLs
6. Port incremental timestamp save on successful apply

### PRU Assignment Logic

```
for each user in copilot_users:
    if user.login in prus_exception_users:
        assign → prus_allowed_cost_center_id
    else:
        assign → no_prus_cost_center_id
```

### Verification

```bash
gh cost-center assign --mode plan           # preview assignments
gh cost-center list-users                    # shows users with PRU markers
gh cost-center report                        # shows summary
go test ./internal/pru/... -v
```

---

## PR 5: Teams-Based Mode

**Branch**: `go/teams-mode`
**Goal**: Port teams-based cost center assignment with auto/manual modes, org/enterprise scopes, and user removal.

### Steps

1. Create `internal/teams/manager.go` — `TeamsManager` struct:
   - In-memory caches: `teamsCache`, `membersCache`, `teamCostCenterCache`
   - `SyncTeamAssignments(mode, ignoreCurrent)` — main orchestration
   - `buildTeamAssignments()` — fetch teams → fetch members → resolve CC names → deduplicate (last-team-wins)
   - Auto mode CC naming: `[org team] org/team-name` or `[enterprise team] team-name`
   - Manual mode: lookup from `teams.team_mappings` config
   - `handleUserRemoval()` — detect users in CC but not in current team members, remove them; skip newly-created CCs
   - `preloadActiveCostCenters()` — fetch all active CCs once for efficiency
   - `GenerateSummary()` — teams-aware summary
2. Wire into `cmd/assign.go` under `--teams` flag path:
   - Port teams config display (scope, mode, orgs/enterprise, auto-create, full-sync settings)
   - Port plan/apply flow with confirmation prompt
   - Port apply result tracking (successful/failed counts per CC)
3. Wire into `cmd/report.go` under `--teams` flag

### Teams Assignment Logic

```
1. Fetch teams (org-level or enterprise-level per scope config)
2. For each team → determine CC name:
     auto:   "[org team] {org}/{team-slug}" or "[enterprise team] {team-slug}"
     manual: lookup team_mappings["org/team-slug"]
3. Fetch members for each team
4. Build {cc_name: [(user, org, team)]} — last-team-wins for multi-team users
5. Ensure all CCs exist (preload → create missing)
6. Convert CC names → CC IDs
7. BulkUpdateCostCenterAssignments (batches of 50)
8. If remove_users_no_longer_in_teams: detect orphans, remove from CC
     (skip newly-created CCs — optimization)
```

### Verification

```bash
gh cost-center assign --teams --mode plan     # preview team→CC mapping
gh cost-center report --teams                 # teams-aware summary
go test ./internal/teams/... -v
```

---

## PR 6: Repository Mode + Budgets

**Branch**: `go/repo-mode-budgets`
**Goal**: Port repository-based cost center assignment using custom properties, and the budget creation system.

### Steps

1. Create `internal/repository/manager.go` — `RepositoryManager` struct:
   - `Run(orgName)` — fetch repos with properties → match against explicit mappings → assign to CCs
   - Explicit mapping matching: for each mapping, filter repos where `property_name` has value in `property_values`
   - Implement plan mode (currently missing in Python — marked `# TODO`)
2. Wire into `cmd/assign.go` under `--repo` flag path:
   - Validate org name from config (`teams.organizations[0]`)
   - Port config display
   - Implement both plan and apply flows
3. Create `internal/budgets/manager.go` — `BudgetManager` struct:
   - Port product→SKU mapping from `_get_budget_type_and_sku()`
   - `EnsureBudgetsForCostCenter(ccID, products)` — idempotent budget creation
   - `BudgetsAPIUnavailableError` handling (disable on 404, skip further attempts)
4. Wire `--create-budgets` flag into teams and repository assignment flows

### Budget Product→SKU Mapping

| Product | Budget Type | SKU |
|---------|------------|-----|
| `actions` | ProductPricing | — |
| `packages` | ProductPricing | — |
| `codespaces` | ProductPricing | — |
| `copilot` | ProductPricing | — |
| `ghas` | ProductPricing | — |
| `ghec` | ProductPricing | — |
| `copilot_premium_request` | SkuPricing | `copilot_premium_request` |
| `actions_linux` | SkuPricing | `actions_linux` |
| `ghas_licenses` | SkuPricing | `ghas_licenses` |

### Verification

```bash
gh cost-center assign --repo --mode plan      # preview repo→CC mapping
gh cost-center assign --repo --mode apply --yes --create-budgets
go test ./internal/repository/... -v
go test ./internal/budgets/... -v
```

---

## PR 7: Cache System + Polish

**Branch**: `go/cache-polish`
**Goal**: Implement the file-based cost center cache (was missing in Python), add signal handling, error formatting, and overall polish.

### Steps

1. Create `internal/cache/cache.go` — `CostCenterCache` struct:
   - JSON file at `.cache/cost_centers.json`
   - 24-hour TTL per entry
   - `Get(key)`, `Set(key, value)`, `GetStats()`, `Clear()`, `CleanupExpired()` methods
2. Wire into `cmd/cache.go` — implement `--stats`, `--clear`, `--cleanup` subcommands
3. Integrate cache into cost center lookups (check cache before API call, update cache on API response)
4. Add signal handling: SIGPIPE → graceful exit, SIGINT → cleanup + exit
5. Add colored output using `go-gh` terminal utilities or `fatih/color`
6. Add comprehensive `--help` text with examples (port from Python argparse epilog)
7. Update README.md — installation via `gh extension install renan-alm/gh-cost-center`, usage examples, migration guide from Python version
8. Create GitHub issues for follow-up items:
   - Any remaining bugs discovered during conversion
   - Performance optimizations (connection pooling, concurrent team fetching)

### Cache File Format

```json
{
  "version": 1,
  "entries": {
    "cost_center_name_or_id": {
      "id": "uuid",
      "name": "string",
      "cached_at": "2026-01-01T00:00:00Z",
      "ttl_hours": 24
    }
  }
}
```

### Verification

```bash
gh cost-center cache --stats
gh cost-center cache --clear
gh cost-center cache --cleanup
gh cost-center --help          # complete help with examples
go test ./internal/cache/... -v
```

---

## PR 8: Remove Python Code + Final Cleanup

**Branch**: `go/final-cleanup`
**Goal**: Remove all Python code, Docker artifacts, and update CI/CD for Go-only workflow.

### Steps

1. Remove all Python source files: `src/`, `main.py`, `requirements.txt`
2. Remove Docker artifacts: `Dockerfile`, `docker-compose.yml`
3. Remove shell automation: `automation/`
4. Remove Python-era docs: `REMOVED_USERS_FEATURE.md`, `TEAMS_INTEGRATION.md`, `TEAMS_QUICKSTART.md`, `CACHING_IMPLEMENTATION.md`, `BUDGET_IMPROVEMENTS.md`
5. Update `CHANGELOG.md` — document v2.0.0 Go rewrite
6. Update `VERSION` to `2.0.0`
7. Update CI/CD workflows:
   - Replace Python workflow with Go build/test/lint workflow (`golangci-lint`, `go test ./...`)
   - Keep `gh-extension-precompile` release workflow from PR 1
   - Update Dependabot to track Go modules instead of pip
8. Update `CODEOWNERS` if needed
9. Update `config/config.example.yaml` — remove `GITHUB_TOKEN` references (auth is via `gh auth login`)
10. Final `go vet ./...` and `golangci-lint run` pass

### Verification

```bash
go build -o gh-cost-center .
gh extension install .
gh cost-center assign --mode plan   # end-to-end test
go vet ./...
golangci-lint run
# Confirm: no Python files remain, CI passes
```

---

## Go Project Structure (Final)

```
gh-cost-center/
├── main.go                          # Entry point
├── go.mod
├── go.sum
├── Makefile
├── VERSION
├── LICENSE
├── README.md
├── CHANGELOG.md
├── CODEOWNERS
├── .gitignore
├── .goreleaser.yml                  # (or use gh-extension-precompile action)
├── config/
│   ├── config.yaml
│   └── config.example.yaml
├── cmd/
│   ├── root.go                      # Root command, global flags
│   ├── assign.go                    # assign subcommand (PRU/teams/repo)
│   ├── list_users.go                # list-users subcommand
│   ├── config.go                    # config subcommand
│   ├── report.go                    # report subcommand
│   ├── cache.go                     # cache subcommand
│   └── version.go                   # version subcommand
├── internal/
│   ├── config/
│   │   ├── config.go                # ConfigManager
│   │   ├── models.go                # Config structs
│   │   └── config_test.go
│   ├── github/
│   │   ├── client.go                # HTTP client, retry, rate-limit
│   │   ├── copilot.go               # Copilot seat management
│   │   ├── costcenters.go           # Cost center CRUD
│   │   ├── teams.go                 # Teams API
│   │   ├── repos.go                 # Repository/property APIs
│   │   ├── budgets.go               # Budget APIs
│   │   └── *_test.go
│   ├── pru/
│   │   ├── manager.go               # PRU-based assignment
│   │   └── manager_test.go
│   ├── teams/
│   │   ├── manager.go               # Teams-based assignment
│   │   └── manager_test.go
│   ├── repository/
│   │   ├── manager.go               # Repository-based assignment
│   │   └── manager_test.go
│   ├── budgets/
│   │   ├── manager.go               # Budget creation
│   │   └── manager_test.go
│   ├── cache/
│   │   ├── cache.go                 # File-based CC cache
│   │   └── cache_test.go
│   └── logging/
│       └── logger.go                # Structured logging setup
├── .github/
│   └── workflows/
│       ├── release.yml              # gh-extension-precompile
│       ├── ci.yml                   # go build + test + lint
│       └── dependabot.yml           # Go modules
└── logs/                            # Runtime log output
```

## Go Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/cli/go-gh/v2` | gh CLI extension SDK (auth, REST client, terminal) |
| `github.com/spf13/cobra` | CLI command framework |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/fatih/color` | Colored terminal output |
| `log/slog` (stdlib) | Structured logging |
| `net/http` (stdlib) | HTTP client (underneath go-gh) |

## Feature Parity Matrix

| Feature | Python Status | Go Target |
|---------|--------------|-----------|
| PRU-based assignment | ✅ Working | PR 4 |
| Teams-based (org scope) | ✅ Working | PR 5 |
| Teams-based (enterprise scope) | ✅ Working | PR 5 |
| Repository-based assignment | ✅ Working (apply only) | PR 6 (+ plan mode) |
| Budget creation | ✅ Working | PR 6 |
| Incremental processing | ✅ Working | PR 4 |
| Cost center auto-creation | ✅ Working | PR 4 |
| User removal (teams) | ✅ Working | PR 5 |
| File-based cache | ❌ Missing (import exists, file doesn't) | PR 7 |
| `FilterUsersByTimestamp` | ❌ Missing (called but not defined) | PR 3 |
| Repo plan mode | ❌ Missing (`# TODO`) | PR 6 |
| Config backward compat | ✅ Working | PR 2 |
| GHE Data Resident support | ✅ Working | PR 2 |
| Colored output | ❌ Partial (emoji only) | PR 7 |
