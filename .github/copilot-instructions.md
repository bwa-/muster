# Muster AI Coding Instructions

You are an expert AI developer working on `muster`, a Universal Control Plane for AI Agents built on the Model Context Protocol (MCP).

## 🏗 Architecture & Core Concepts

- **Meta-MCP Server**: `muster` acts as an aggregator. `muster serve` manages processes (MCP servers), and `muster agent` exposes a unified MCP interface to clients (IDEs, agents).
- **Service Locator Pattern**:
  - **Central API**: `internal/api` is the ONLY allowed dependency for inter-package communication.
  - **Interfaces**: Defined in `internal/api/handlers.go`.
  - **Registration**: Packages implement interfaces and register via `api.Register*` (e.g., `api.RegisterServiceManager`).
  - **Consumption**: Packages retrieve handlers via `api.Get*`.
  - **Strict Rule**: NEVER import `workflow`, `mcpserver`, `serviceclass`, or `service` packages directly into each other.

## 🛠 Critical Workflows

- **Build**: `make build` (generates binaries in `bin-dist/`).
- **Unit Tests**: `make test`.
- **Behavioral Tests (Scenarios)**:
  - Run specific scenario: `muster test --scenario <scenario_name> --verbose`
  - Debug scenario: `muster test --scenario <scenario_name> --verbose --debug`
  - **Note**: Scenarios run in isolated `muster serve` instances.
- **Development Loop**:
  1.  **Restart Service**: `./scripts/dev-restart.sh` (rebuilds and restarts systemd service).
  2.  **Check Logs**: `journalctl --user -u muster.service --no-pager | tail -n 50`.
  3.  **Debug**: Use `mcp-debug` tools (e.g., `mcp_mcp-debug_call_tool`) to verify changes.

## 📝 Coding Conventions

- **Go Style**:
  - Run `goimports -w .` and `go fmt ./...` before every commit.
  - Wrap errors: `fmt.Errorf("context: %w", err)`.
  - File size limit: Keep files under **400 lines**.
- **Testing**:
  - **Coverage**: Aim for >80% unit test coverage.
  - **Determinism**: NEVER use `time.Sleep`. Use dependency injection for time/clocks.
  - **Schema**: `schema.json` is generated. Run `muster test --generate-schema` to update it.
- **Dependencies**: Use `web_search` to verify latest versions before adding `go get` dependencies.
- **Commits**: Format as `<Type>: <Description> (closes #<issue_num>)`.

## 🧩 Integration & Patterns

- **MCP Tools**:
  - `core_*`: Native muster tools (e.g., `core_service_list`).
  - `x_<server>_*`: Tools from managed MCP servers (e.g., `x_kubernetes_list_pods`).
- **Workflows**:
  - Defined as `action_<name>` in API/internal.
  - Exposed as `workflow_<name>` to users.
  - Aggregator handles the mapping.
- **Kubernetes**: CRDs in `deploy/crds/` define the data model (`MCPServer`, `ServiceClass`, `Workflow`).

## 🚫 Anti-Patterns

- **Direct Coupling**: Importing sibling packages instead of using `internal/api`.
- **Flaky Tests**: Using `time.Sleep` to fix race conditions.
- **Manual Schema Edits**: Editing `schema.json` by hand.

## 🌿 Branch Strategy & Feature Development

This fork follows a strict branch workflow to maintain clean history and enable selective PRs to upstream:

### Branch Structure
- **`main`**: Clean mirror of upstream. NEVER commit features directly here.
- **`dev`**: Daily working branch with all features merged. Used for local development.
- **`feature/*`**: Individual feature branches for PRs. Each contains ONE logical feature.

### Feature Development Workflow

**Creating a new feature:**
1. Start from clean `main`: `git checkout main && git pull upstream main`
2. Create feature branch: `git checkout -b feature/descriptive-name`
3. Implement feature with commits (NO version bumps in feature branch)
4. Push feature branch: `git push origin feature/descriptive-name`

**Merging to dev branch:**
1. Switch to dev: `git checkout dev`
2. Merge with no-commit: `git merge feature/descriptive-name --no-commit --no-ff`
3. Update version in `main.go` (increment minor version)
4. Commit merge: `git commit -m "Merge feature/descriptive-name (vX.X.X)"`
5. Push dev: `git push origin dev`

**Syncing with upstream:**
1. Update main: `git checkout main && git pull upstream main && git push origin main`
2. Rebase each feature: `git checkout feature/X && git rebase main && git push origin feature/X --force`
3. Rebuild dev: `git checkout dev && git reset --hard main`
4. Re-merge all features in order (oldest to newest)
5. Push dev: `git push origin dev --force`

### Version Management
- **Feature branches**: NO version changes (keeps them reusable)
- **Dev branch**: Version bumped during merge commits
- **Version scheme**: Semantic versioning (MAJOR.MINOR.PATCH)
- **Increment**: Minor version per feature, patch for fixes

### PR Creation
- Create PRs from `feature/*` branches to upstream `main`
- Each PR is independent and can be accepted/rejected separately
- Feature branch stays at upstream's current version (no conflicts)
- If PR rejected, simply don't merge that feature into next dev rebuild

### Key Principles
- Keep `main` synced with upstream (enables clean rebases)
- Keep feature branches small and focused (one logical change)
- Keep dev as throw-away branch (rebuilt when needed)
- Never version bump in feature branches (done during merge to dev)

**IMPORTANT:** Only merge *complete* features into the `dev` branch. Do not merge incomplete or partially implemented features. Every feature branch must be fully implemented, tested, and documented before merging to `dev`.
