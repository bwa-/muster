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

This fork uses a consolidated development workflow with the ability to extract features for upstream PRs:

### Branch Structure
- **`main`**: Clean mirror of upstream. NEVER commit features directly here. Keep synced with `git pull upstream main`.
- **`dev`**: Primary development branch where all features are developed and merged together. This is the daily working branch.

### Development Workflow

**Daily Development (in dev branch):**
1. Work directly in `dev` branch for all feature development
2. Commit frequently with descriptive, tagged messages using format: `[feature-name]: description`
   - Example: `[env-vars]: Add environment variable substitution to workflow executor`
   - Example: `[foreach]: Implement forEach loop support in workflow CRD`
3. Test after each significant change: `make build && make test`
4. Push to origin regularly: `git push origin dev`

**Commit Tagging Convention:**
- Use square brackets with feature name at start of commit message
- Common tags: `[env-vars]`, `[foreach]`, `[validation]`, `[yaml]`, `[text-transform]`, etc.
- Keep tags consistent for related commits to enable easy filtering later
- Multi-feature changes: `[env-vars][foreach]: Combined fix for...`

**Syncing with Upstream:**
1. Update main: `git checkout main && git pull upstream main && git push origin main`
2. Merge upstream into dev: `git checkout dev && git merge main`
3. Resolve any conflicts, test, and push: `git push origin dev`

### Extracting Features for Upstream PRs

When a feature in `dev` is ready for an upstream PR:

1. **Create extraction branch** from clean upstream main:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feature/descriptive-name
   ```

2. **Cherry-pick relevant commits** using git log filtering:
   ```bash
   git log dev --grep="\[feature-name\]" --oneline  # Find commits
   git cherry-pick <commit-hash>  # Pick each relevant commit
   ```

3. **Test extracted feature in isolation:**
   ```bash
   make build && make test
   muster test --scenario <relevant-scenario> --verbose
   ```

4. **Adjust if needed** - Feature may need small adjustments to work standalone
5. **Submit PR** to upstream from the extraction branch
6. **Keep dev unchanged** - No rebasing or modification of dev branch

### Key Principles
- **Develop in dev**: All features built together in dev branch, dependencies are natural
- **Tag commits**: Use `[feature-name]` prefixes for easy filtering
- **Extract when ready**: Only create feature branches when preparing upstream PR
- **Test extractions**: Extracted features must work standalone
- **Sync regularly**: Pull upstream changes into main, merge main into dev
- **Complete features only**: Only commit complete, tested features to dev

**IMPORTANT:** Features in `dev` can depend on each other freely. Only when extracting for upstream PR do we need to ensure the feature works independently.
