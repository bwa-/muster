# Tool Naming System Improvements

**Status**: Proposed  
**Date**: 2025-12-18  
**Context**: Current three-level prefix system (`x_server-name_tool-name`) is verbose and inflexible

## Current System

Muster uses hierarchical prefixing to aggregate tools from multiple MCP servers:

```
{muster_prefix}_{server_prefix}_{original_tool_name}
```

**Examples:**
- `x_github_list_files` - GitHub tool accessed through muster
- `x_docs-mcp-server_search_docs` - Docs tool (29 chars!)
- `core_service_list` - Built-in muster functionality
- `workflow_deploy-app` - User-defined workflow

**Prefix Categories:**
1. `x_*` - External MCP server tools (aggregation layer)
2. `core_*` - Built-in muster functionality
3. `workflow_*` - User-defined workflows as tools

## Problems

1. **Verbose Tool Names**: `x_docs-mcp-server_search_docs` is 29 characters
2. **Server Name Duplication**: Renaming a server breaks all workflows using its tools
3. **No Automatic Short Names**: Must manually configure `toolPrefix` for ergonomics
4. **Limited Aliasing**: Can't create tool shortcuts or hide complexity
5. **Inconsistent URI Handling**: Resources with schemes bypass prefixing

## Proposed Solutions

### Option 1: Hierarchical with Smart Defaults (Evolutionary)

**Enhancement to existing system - MINIMAL BREAKING CHANGES**

```yaml
# MCPServer CRD Enhancement
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: docs-mcp-server
spec:
  toolPrefix: "docs"  # Short prefix (existing)
  toolNamespace: "documentation"  # Optional logical grouping (NEW)
  aliases:  # NEW
    search: "search_docs"  # Expose as x_docs_search
    index: "index_libraries"
```

**Usage:**
```yaml
steps:
  - tool: x_docs_search  # Alias (short form)
  - tool: x_docs_search_docs  # Full name still works
```

**Pros:**
- Backward compatible
- Opt-in ergonomics
- Preserves uniqueness guarantees

**Cons:**
- Still requires configuration
- Doesn't solve server rename problem

---

### Option 2: DNS-Style Namespacing (Revolutionary)

**Complete redesign using familiar DNS/package patterns**

```
{category}.{server}.{tool}
```

**Examples:**
- `ext.github.list_files` (external MCP server)
- `core.service.list` (muster core)
- `workflow.deploy.app` (workflow)
- `ext.k8s.pods.get` (multi-level hierarchies)

**Benefits:**
- Familiar to developers (DNS, Java packages, Python imports)
- Supports arbitrary depth
- Easy to parse/validate with regex
- Natural "wildcard" patterns for security: `core.*`, `ext.github.*`

**Challenges:**
- **BREAKING CHANGE** - all existing workflows need migration
- Schema validation more complex
- Templating syntax changes: `{{.tools["ext.github.list_files"]}}`

---

### Option 3: Implicit Context with Path Resolution (Pragmatic) ⭐ RECOMMENDED

**Add workflow-scoped import aliases while preserving backward compatibility**

```yaml
# Workflow YAML
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: example-workflow
  imports:  # NEW SECTION
    docs: x_docs-mcp-server  # Define short alias at workflow level
    k8s: x_kubernetes
    gh: x_github-cli

spec:
  steps:
    - tool: docs.search_docs  # Resolves to x_docs-mcp-server_search_docs
    - tool: k8s.get_pods      # Resolves to x_kubernetes_get_pods
    - tool: core_service_list  # Fully qualified still works (no import needed)
    - tool: x_docs-mcp-server_search_docs  # Original form still works
```

**Resolution Logic (parse-time transformation):**
```go
func resolveToolName(tool string, imports map[string]string) string {
    if strings.Contains(tool, ".") {
        parts := strings.SplitN(tool, ".", 2)
        if prefix, ok := imports[parts[0]]; ok {
            return prefix + "_" + parts[1]
        }
    }
    return tool // Fully qualified, no transformation
}
```

**Pros:**
- **100% backward compatible** - existing workflows work unchanged
- **Low implementation cost** - simple parse-time transformation
- **High ergonomic value** - 50-70% reduction in verbosity
- **Workflow-scoped** - no global conflicts, clear dependencies
- **Familiar pattern** - like import systems in programming languages
- **Clear migration path** - can evolve to Option 4 later

**Cons:**
- Requires workflow-level import declarations
- Doesn't solve server rename brittleness (but makes it less painful)

---

### Option 4: Registry with Semantic Versioning (Enterprise)

**Add abstraction layer with semantic naming and versioning**

```yaml
# Tool Registry (auto-generated from MCPServer definitions)
apiVersion: muster.giantswarm.io/v1alpha1
kind: ToolRegistry
tools:
  search_documentation:
    provider: x_docs-mcp-server_search_docs
    version: v1
    aliases: [search, docs.search]
    deprecated: false
    
  create_pod:
    provider: x_kubernetes_create_pod
    version: v2  # Tracks breaking changes
    aliases: [k8s.pod.create, pod.create]
    changelog:
      v2: "Added resource limits support"
      v1: "Initial release"
```

**Usage in workflows:**
```yaml
steps:
  - tool: search_documentation  # Semantic name
  - tool: docs.search           # Alias
  - tool: x_kubernetes_create_pod@v1  # Version pinning
```

**Pros:**
- **Semantic, stable names** decoupled from implementation
- **Versioning** for tool API changes
- **Migration paths** when tools change
- **Best DX** for end users
- **Centralized** tool catalog

**Cons:**
- Requires significant registry infrastructure
- Complex to implement and maintain
- Need migration/compatibility tooling
- Overhead for simple deployments

---

## Recommendation: Option 3 (Implicit Context)

**Rationale:**
1. **Backward compatible**: Zero-risk deployment, existing workflows unaffected
2. **Low implementation cost**: ~200 lines of code, parse-time only
3. **High ergonomic value**: Dramatic readability improvement
4. **Clear migration**: Can evolve to Option 4 in future
5. **Familiar pattern**: Developers understand import systems

**Implementation Priority:**
- Phase 1: Add `imports` field to Workflow CRD schema
- Phase 2: Implement resolution logic in workflow parser
- Phase 3: Add schema validation for import references
- Phase 4: Update documentation and examples

**Example Impact:**

Before:
```yaml
- tool: x_docs-mcp-server_search_docs
- tool: x_kubernetes_get_pods  
- tool: x_github-cli_clone_repo
```

After:
```yaml
imports:
  docs: x_docs-mcp-server
  k8s: x_kubernetes
  gh: x_github-cli
  
steps:
  - tool: docs.search_docs
  - tool: k8s.get_pods
  - tool: gh.clone_repo
```

**Character savings**: 40-60% reduction in common cases

---

## Future Considerations

- **Server Aliasing**: Allow global aliases in muster config (not just workflow-scoped)
- **Auto-imports**: Generate import suggestions based on tool usage
- **Namespace Scoping**: Support nested imports (`k8s.core.pods`, `k8s.apps.deployments`)
- **Version Pinning**: Add version support to imports (`k8s@v1`, `docs@v2`)

---

## References

- [Name Tracker Implementation](../../internal/aggregator/name_tracker.go)
- [Workflow Schema](../../workflow-schema.json)
- [MCP Server CRD](../../deploy/crds/muster.giantswarm.io_mcpservers.yaml)
