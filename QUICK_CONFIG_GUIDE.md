# Quick Configuration Guide

This guide walks through the most common `gh-cost-center` configuration scenarios. Every example assumes you have already:

1. Installed the extension: `gh extension install renan-alm/gh-cost-center`
2. Created your config file: `cp config/config.example.yaml config/config.yaml`
3. Set up authentication (see [README → Authentication](README.md#authentication))

All examples use `--mode plan` (dry-run). Replace with `--mode apply --yes` when you are ready to push changes.

---

## Table of Contents

- [1. PRU-Based (Default)](#1-pru-based-default)
- [2. Teams-Based — Auto Mode](#2-teams-based--auto-mode)
- [3. Teams-Based — Manual Mode](#3-teams-based--manual-mode)
- [4. Repository-Based](#4-repository-based)
- [5. Complex Scenarios](#5-complex-scenarios)
  - [5a. Teams + Budgets + Auto-Create](#5a-teams--budgets--auto-create)
  - [5b. Multi-Org Manual Teams Mapping](#5b-multi-org-manual-teams-mapping)
  - [5c. Repository Mode with Multiple Properties](#5c-repository-mode-with-multiple-properties)
  - [5d. GHE Data Resident / GHES with Teams](#5d-ghe-data-resident--ghes-with-teams)

---

## 1. PRU-Based (Default)

The simplest mode. All Copilot users go to a "no PRU overages" cost center, except for a list of exception users who go to a "PRU overages allowed" cost center.

### Config

```yaml
github:
  enterprise: "my-enterprise"

cost_centers:
  auto_create: true
  no_prus_cost_center_name: "00 - No PRU overages"
  prus_allowed_cost_center_name: "01 - PRU overages allowed"
  prus_exception_users:
    - "alice"
    - "bob"
```

### Run

```bash
gh cost-center assign --mode plan
gh cost-center assign --mode apply --yes
```

### What happens

| User | Assigned to |
|------|-------------|
| alice | 01 - PRU overages allowed |
| bob | 01 - PRU overages allowed |
| everyone else | 00 - No PRU overages |

---

## 2. Teams-Based — Auto Mode

One cost center is created automatically for each GitHub team. Users are assigned to the cost center that matches their team membership.

### Config

```yaml
github:
  enterprise: "my-enterprise"

teams:
  enabled: true
  scope: "organization"        # query teams from organizations
  mode: "auto"                 # one cost center per team (auto-named)
  organizations:
    - "my-org"
  auto_create_cost_centers: true
  remove_users_no_longer_in_teams: true
```

### Run

```bash
gh cost-center assign --teams --mode plan
```

### Cost center naming

| Scope | Naming pattern | Example |
|-------|---------------|---------|
| `organization` | `[org team] {org}/{team}` | `[org team] my-org/frontend` |
| `enterprise` | `[enterprise team] {team}` | `[enterprise team] platform` |

---

## 3. Teams-Based — Manual Mode

You control exactly which team maps to which cost center name.

### Config

```yaml
github:
  enterprise: "my-enterprise"

teams:
  enabled: true
  scope: "organization"
  mode: "manual"
  organizations:
    - "my-org"
  auto_create_cost_centers: true
  remove_users_no_longer_in_teams: true
  team_mappings:
    "my-org/frontend": "CC-Frontend"
    "my-org/backend": "CC-Backend"
    "my-org/infra": "CC-Platform"
```

### Run

```bash
gh cost-center assign --teams --mode plan
```

### What happens

| Team | Cost center |
|------|-------------|
| my-org/frontend | CC-Frontend |
| my-org/backend | CC-Backend |
| my-org/infra | CC-Platform |
| Teams not listed | Skipped |

---

## 4. Repository-Based

Assign **repositories** (not users) to cost centers based on organization custom property values.

> **Prerequisite:** Configure [custom properties](https://docs.github.com/en/organizations/managing-organization-settings/managing-custom-properties-for-repositories-in-your-organization) on your GitHub organization repositories first.

### Config

```yaml
github:
  enterprise: "my-enterprise"
  cost_centers:
    mode: "repository"
    repository_config:
      explicit_mappings:
        - cost_center: "Platform Engineering"
          property_name: "team"
          property_values:
            - "platform"
            - "infrastructure"

        - cost_center: "Product"
          property_name: "team"
          property_values:
            - "product"

teams:
  organizations:
    - "my-org"          # required — tells the CLI which orgs to scan
```

### Run

```bash
gh cost-center assign --repo --mode plan
```

### Notes

- Property names and values are **case-sensitive**.
- Repositories without the specified property are skipped.
- A repository can match multiple mappings.

---

## 5. Complex Scenarios

### 5a. Teams + Budgets + Auto-Create

Automatically create one cost center per team **and** provision Copilot and Actions budgets for each new cost center.

```yaml
github:
  enterprise: "my-enterprise"

teams:
  enabled: true
  scope: "organization"
  mode: "auto"
  organizations:
    - "my-org"
  auto_create_cost_centers: true
  remove_users_no_longer_in_teams: true

budgets:
  enabled: true
  products:
    copilot:
      amount: 200       # USD per cost center
      enabled: true
    actions:
      amount: 150
      enabled: true
```

```bash
gh cost-center assign --teams --mode plan --create-cost-centers --create-budgets
```

---

### 5b. Multi-Org Manual Teams Mapping

Your enterprise has multiple organizations and you want to map teams from each org to shared cost centers.

```yaml
github:
  enterprise: "my-enterprise"

teams:
  enabled: true
  scope: "organization"
  mode: "manual"
  organizations:
    - "org-alpha"
    - "org-beta"
  auto_create_cost_centers: true
  remove_users_no_longer_in_teams: true
  team_mappings:
    # Alpha org
    "org-alpha/backend": "CC-Engineering"
    "org-alpha/frontend": "CC-Engineering"
    "org-alpha/data": "CC-Data"
    # Beta org — maps into the same cost centers
    "org-beta/api-team": "CC-Engineering"
    "org-beta/analytics": "CC-Data"
```

```bash
gh cost-center assign --teams --mode plan --create-cost-centers
```

Users from both organizations are consolidated into shared cost centers (`CC-Engineering`, `CC-Data`).

---

### 5c. Repository Mode with Multiple Properties

Map repositories using different custom properties to different cost centers. For example, separate by both `team` and `environment`.

```yaml
github:
  enterprise: "my-enterprise"
  cost_centers:
    mode: "repository"
    repository_config:
      explicit_mappings:
        # By team
        - cost_center: "Platform Engineering"
          property_name: "team"
          property_values: ["platform", "infrastructure", "devops"]

        - cost_center: "Frontend Applications"
          property_name: "team"
          property_values: ["web", "mobile", "ui"]

        # By environment
        - cost_center: "Production Services"
          property_name: "environment"
          property_values: ["production"]

        - cost_center: "Staging"
          property_name: "environment"
          property_values: ["staging", "qa"]

teams:
  organizations:
    - "my-org"
```

```bash
gh cost-center assign --repo --mode plan --create-cost-centers
```

> A repository with `team=platform` **and** `environment=production` will match **both** "Platform Engineering" and "Production Services".

---

### 5d. GHE Data Resident / GHES with Teams

If your enterprise runs on GitHub Enterprise Data Resident or a self-hosted GitHub Enterprise Server, set `api_base_url` and use any mode as normal.

```yaml
github:
  enterprise: "my-enterprise"

  # GHE Data Resident
  api_base_url: "https://api.octocorp.ghe.com"

  # — or GHES (self-hosted) —
  # api_base_url: "https://github.company.com/api/v3"

teams:
  enabled: true
  scope: "organization"
  mode: "auto"
  organizations:
    - "my-org"
  auto_create_cost_centers: true
```

```bash
gh cost-center assign --teams --mode plan
```

---

## Quick Reference: Key Flags

| Flag | Purpose |
|------|---------|
| `--mode plan` | Dry-run — preview changes without applying |
| `--mode apply` | Push changes to GitHub |
| `--yes` / `-y` | Skip confirmation prompt in apply mode |
| `--teams` | Use teams-based assignment |
| `--repo` | Use repository-based assignment |
| `--create-cost-centers` | Create cost centers that don't exist yet |
| `--create-budgets` | Create budgets for new cost centers |
| `--incremental` | Only process users added since last run (PRU mode) |
| `--check-current` | Check current cost center membership before assigning |
| `--token <PAT>` | Pass a GitHub token directly |
| `--config <path>` | Use a custom config file path |
| `--verbose` / `-v` | Enable debug logging |

---

## Verifying Your Config

Before running an assignment, check the resolved configuration:

```bash
gh cost-center config
```

This shows the final merged values after applying environment variables, `.env` files, and YAML defaults.
