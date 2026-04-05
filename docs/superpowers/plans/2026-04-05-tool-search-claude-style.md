# Tool Search Claude Style Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace OpenViking-backed tool search with a Claude Code style local keyword search, and remove the entire OpenViking tool-index sync pipeline.

**Architecture:** Keep the current runtime visibility middleware and `selected_tool_names` contract, but change `tool_search` input to `query/max_results` and move matching into a MySQL-backed local scoring path. Remove all plugin-side OpenViking indexing, queueing, DAO helpers, and config fields that only existed for tool search.

**Tech Stack:** Go, Gin, GORM, Eino ADK, MySQL, existing plugin runtime services

---

### Task 1: Replace OpenViking recall with local runtime-tool search

**Files:**
- Modify: `openIntern_backend/internal/dao/plugin.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin_search.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin_runtime_catalog.go`
- Delete: `openIntern_backend/internal/dao/plugin_search.go`

- [ ] **Step 1: Add a DAO view for enabled runtime-tool search rows**

```go
type EnabledRuntimeToolSearchRow struct {
	ToolID            string `gorm:"column:tool_id"`
	ToolName          string `gorm:"column:tool_name"`
	Description       string `gorm:"column:description"`
	RuntimeType       string `gorm:"column:runtime_type"`
	PluginName        string `gorm:"column:plugin_name"`
	PluginDescription string `gorm:"column:plugin_description"`
}

func (d *PluginDAO) ListEnabledRuntimeToolSearchRows(userID string, runtimeTypes []string) ([]EnabledRuntimeToolSearchRow, error) {
	query := database.DB.
		Table("tool").
		Select("tool.tool_id AS tool_id, tool.tool_name AS tool_name, tool.description AS description, plugin.runtime_type AS runtime_type, plugin.name AS plugin_name, plugin.description AS plugin_description").
		Joins("JOIN plugin ON plugin.plugin_id = tool.plugin_id AND plugin.user_id = tool.user_id").
		Where("tool.user_id = ? AND tool.enabled = ? AND plugin.status = ?", strings.TrimSpace(userID), true, "enabled")
	if len(runtimeTypes) > 0 {
		query = query.Where("plugin.runtime_type IN ?", runtimeTypes)
	}

	var rows []EnabledRuntimeToolSearchRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
```

- [ ] **Step 2: Remove the OpenViking DAO dependency**

Delete `openIntern_backend/internal/dao/plugin_search.go` after moving all remaining tool-search data access to `plugin.go`.

- [ ] **Step 3: Rebuild `plugin_search.go` around Claude-style local matching**

```go
type localRuntimeToolCandidate struct {
	ToolID            string
	ToolName          string
	Description       string
	RuntimeType       string
	PluginName        string
	PluginDescription string
	NameParts         []string
	FullName          string
}

type parsedToolSearchQuery struct {
	Raw           string
	IsSelect      bool
	SelectedNames []string
	RequiredTerms []string
	OptionalTerms []string
}
```

Add helpers in `plugin_search.go` for:

- parsing `select:tool_a,tool_b`
- splitting normal queries into `+required` and optional terms
- splitting tool names by underscore and CamelCase
- scoring candidates by tool-name, description, plugin-name, plugin-description matches
- stable sorting by score desc then `tool_name` asc

- [ ] **Step 4: Keep existing service-facing API stable**

```go
func (s *PluginService) SearchRuntimeTools(ctx context.Context, query string, options ToolSearchOptions) ([]RuntimeToolSearchMatch, error) {
	// local lookup only
}

func (s *PluginService) SearchRuntimeToolIDs(ctx context.Context, query string, options ToolSearchOptions) ([]string, error) {
	// map SearchRuntimeTools output back to ids
}
```

Preserve:

- user scoping
- runtime type filtering
- MCP count limiting

Remove:

- `TargetURI`
- OpenViking URI parsing
- score threshold plumbing that only applied to remote recall

- [ ] **Step 5: Run a focused build for the plugin package**

Run: `go build ./internal/services/plugin`
Expected: exit code `0`

- [ ] **Step 6: Commit**

```bash
git add openIntern_backend/internal/dao/plugin.go openIntern_backend/internal/services/plugin/plugin_search.go openIntern_backend/internal/services/plugin/plugin_runtime_catalog.go openIntern_backend/internal/dao/plugin_search.go
git commit -m "refactor: switch tool search to local matching"
```

Do not actually commit during implementation unless the user explicitly asks for a commit.

### Task 2: Switch the `tool_search` tool contract to Claude Code style

**Files:**
- Modify: `openIntern_backend/internal/services/middlewares/toolsearch/tool.go`
- Inspect: `openIntern_backend/internal/services/middlewares/toolsearch/middleware.go`

- [ ] **Step 1: Replace the old input schema**

```go
type Input struct {
	Query      string `json:"query" jsonschema_description:"Use select:tool_name for direct selection, or keywords to search deferred tools."`
	MaxResults int    `json:"max_results,omitempty" jsonschema_description:"Maximum number of results to return."`
}
```

- [ ] **Step 2: Update the tool prompt text**

Describe:

- `select:tool_a,tool_b`
- normal keyword search
- `+required optional` syntax

Remove all references to:

- `intent`
- `keywords`
- `top_k`

- [ ] **Step 3: Keep the middleware result shape unchanged**

```go
result := Result{
	SelectedToolNames: util.NormalizeUniqueStringList(names),
}
```

`middleware.go` should continue to parse `selected_tool_names` without any structural change.

- [ ] **Step 4: Normalize `max_results` bounds**

Use a small default and cap, mirroring Claude-style usage:

```go
const (
	defaultRequestedMaxResults = 5
	maxRequestedMaxResults     = 8
)
```

- [ ] **Step 5: Run a focused build for the toolsearch middleware package**

Run: `go build ./internal/services/middlewares/toolsearch`
Expected: exit code `0`

### Task 3: Remove the OpenViking tool-index sync pipeline

**Files:**
- Modify: `openIntern_backend/internal/services/plugin/plugin.go`
- Modify: `openIntern_backend/internal/services/plugin/plugin_mcp_sync.go`
- Delete: `openIntern_backend/internal/services/plugin/plugin_openviking_sync.go`
- Delete: `openIntern_backend/internal/dao/plugin_store.go`

- [ ] **Step 1: Remove plugin lifecycle queue hooks**

Delete OpenViking-specific branches from:

- `PluginService.Create`
- `PluginService.Update`
- `PluginService.Delete`
- `PluginService.UpdateStatus`

The resulting flow should only keep runtime data persistence and MCP sync logic.

- [ ] **Step 2: Remove direct OpenViking sync calls from MCP sync**

In `plugin_mcp_sync.go`, change:

```go
return s.syncMCPPluginRecord(syncCtx, plugin, false)
```

and collapse `syncMCPPluginRecord` so it no longer accepts or propagates `enqueueOpenVikingSync`.

- [ ] **Step 3: Delete dead OpenViking tool-index code**

Delete these files entirely:

- `openIntern_backend/internal/services/plugin/plugin_openviking_sync.go`
- `openIntern_backend/internal/dao/plugin_store.go`

- [ ] **Step 4: Run a global reference sweep**

Run: `rg -n "queueOpenViking|ToolStoreConfigured|SyncPluginToolsToOpenViking|openVikingSync|ToolsRoot\\(" openIntern_backend`
Expected: no remaining matches related to plugin tool indexing

- [ ] **Step 5: Run a focused build for the plugin service package**

Run: `go build ./internal/services/plugin`
Expected: exit code `0`

### Task 4: Remove obsolete config and documentation references

**Files:**
- Modify: `openIntern_backend/internal/config/config.go`
- Modify: `openIntern_backend/internal/database/context_store.go`
- Modify: `openIntern_backend/main.go`
- Modify: `README.md`
- Optionally Modify: `compose.yaml`

- [ ] **Step 1: Remove plugin-side OpenViking sync config**

Delete from `PluginConfig`:

```go
OpenVikingSyncDelaySeconds   int
OpenVikingSyncPollSeconds    int
OpenVikingSyncTimeoutSeconds int
OpenVikingSyncRetrySeconds   int
```

- [ ] **Step 2: Remove tool-index root config if now unused**

Delete from `OpenVikingConfig` and `ContextStore`:

```go
ToolsRoot string
func (s *ContextStore) ToolsRoot() string
```

Only do this after confirming no memory/skills path still depends on it.

- [ ] **Step 3: Remove unused initialization**

Delete:

```go
initPluginOpenVikingSync(cfg)
```

from plugin initialization if no callers remain.

- [ ] **Step 4: Update README wording**

Document that:

- OpenViking is still used by memory/skills-related backend capabilities
- `tool_search` no longer depends on OpenViking
- local startup no longer needs OpenViking for plugin tool search

- [ ] **Step 5: Run a backend build from the service root**

Run: `go build ./...`
Expected: exit code `0`

This repository does not allow adding new Go test files or running `go test` without approval, so build verification is the completion gate for this change.

- [ ] **Step 6: Final diff sanity check**

Run: `git diff -- openIntern_backend README.md docs/superpowers/specs/2026-04-05-tool-search-claude-style-design.md docs/superpowers/plans/2026-04-05-tool-search-claude-style.md`
Expected: only tool-search-localization, OpenViking tool-index removal, and docs updates
