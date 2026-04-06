# OpenViking User-Private Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 OpenViking 的用户记忆检索、skills、知识库改为按 openIntern 登录用户私有隔离，直接使用 `user_id` UUID 作为用户命名空间。

**Architecture:** 通过一层统一的 OpenViking 用户路径构造与上下文注入能力，消除代码中所有 `default` 用户路径与全局 skill/KB 根路径。`memory` 仅改检索路径，不改 session 驱动的 memory sync 写入链路；`skills` 与 `知识库` 全量改为当前登录用户私有根。由于仓库约束禁止随意新增 Go test 文件和执行 `go test`，验证以 `go build ./...` 和双用户手工校验为主。

**Tech Stack:** Go, Gin, GORM, OpenViking HTTP API, MySQL

---

### Task 1: Add A Shared OpenViking User Scope Helper

**Files:**
- Create: `openIntern_backend/internal/dao/openviking_user_scope.go`

- [ ] **Step 1: Add context helpers and user-scoped URI builders**

```go
package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type openVikingUserContextKey string

const openVikingUserIDContextKey openVikingUserContextKey = "openviking_user_id"

func WithOpenVikingUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openVikingUserIDContextKey, strings.TrimSpace(userID))
}

func OpenVikingUserIDFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("openviking user id is required")
	}
	userID := strings.TrimSpace(fmt.Sprintf("%v", ctx.Value(openVikingUserIDContextKey)))
	if userID == "" {
		return "", errors.New("openviking user id is required")
	}
	return userID, nil
}

func UserMemoryRootURI(userID string) string {
	return "viking://user/" + strings.TrimSpace(userID) + "/memories/"
}

func UserSkillRootURI(userID string) string {
	return "viking://user/" + strings.TrimSpace(userID) + "/skills"
}

func UserKnowledgeBaseRootURI(userID string) string {
	return "viking://resources/users/" + strings.TrimSpace(userID) + "/kbs/"
}

func UserKnowledgeBaseURI(userID, kbName string) string {
	return strings.TrimRight(UserKnowledgeBaseRootURI(userID), "/") + "/" + strings.Trim(kbName, "/") + "/"
}
```

- [ ] **Step 2: Verify no old hard-coded user memory roots remain the planned source of truth**

Run: `rg -n "viking://user/default/memories|viking://agent/skills" openIntern_backend/internal`

Expected: matches still exist before refactor, confirming the next tasks have concrete replacements to make

### Task 2: Refactor Memory Retrieval To Use The Authenticated User Space

**Files:**
- Modify: `openIntern_backend/internal/dao/memory_search.go`
- Modify: `openIntern_backend/internal/services/memory/openviking/retriever.go`
- Modify: `openIntern_backend/internal/services/agent/agent_entry.go`
- Modify: `openIntern_backend/internal/services/agent/agent_service.go`

- [ ] **Step 1: Replace fixed user memory root helpers with explicit user-scoped helpers**

```go
// memory_search.go
func (d *MemorySearchDAO) UserRootURI(userID string) string {
	return UserMemoryRootURI(userID)
}
```

- [ ] **Step 2: Change the memory retriever contract to require user id**

```go
// agent_service.go
type MemoryRetriever interface {
	Retrieve(ctx context.Context, userID string, inputText string) ([]contracts.RetrievedMemory, error)
}

// retriever.go
func (r *Retriever) Retrieve(ctx context.Context, userID string, inputText string) ([]contracts.RetrievedMemory, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	query := strings.TrimSpace(inputText)
	if query == "" {
		return nil, nil
	}
	matches, err := r.findRelevantMemoryMatches(ctx, userID, query)
	if err != nil {
		return nil, err
	}
	return toRetrievedMemories(matches), nil
}
```

- [ ] **Step 3: Update retrieval to stop reading from the default user namespace**

```go
userMatches, err := dao.MemorySearch.FindMemoryMatches(ctx, dao.MemorySearchFilter{
	Query:          query,
	TargetURI:      dao.MemorySearch.UserRootURI(userID),
	Limit:          defaultUserMemoryFindLimit,
	ScoreThreshold: defaultMemoryScoreThreshold,
})
```

- [ ] **Step 4: Pass the authenticated owner id into memory retrieval from the chat runtime**

```go
preparedInput, err = injectRetrievedMemoryContext(
	ctx,
	s.deps.MemoryRetriever,
	ownerID,
	mergedInput,
)
```

- [ ] **Step 5: Update the memory injection helper signature**

```go
func injectRetrievedMemoryContext(
	ctx context.Context,
	retriever MemoryRetriever,
	userID string,
	input *types.RunAgentInput,
) (*types.RunAgentInput, error)
```

- [ ] **Step 6: Verify the backend still compiles after the interface change**

Run: `go build ./...`

Expected: build succeeds without output

### Task 3: Make Skill Frontmatter And Skill Storage User-Scoped

**Files:**
- Modify: `openIntern_backend/internal/models/skill_frontmatter.go`
- Modify: `openIntern_backend/internal/dao/skill_frontmatter.go`
- Modify: `openIntern_backend/internal/services/skill/frontmatter.go`
- Modify: `openIntern_backend/internal/services/skill/store_adapter.go`
- Modify: `openIntern_backend/internal/dao/skill_store.go`
- Modify: `openIntern_backend/internal/services/middlewares/skill/backend.go`

- [ ] **Step 1: Add `user_id` to the skill frontmatter model**

```go
type SkillFrontmatter struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	UserID    string         `gorm:"column:user_id;index:ux_skill_frontmatter_user_name,priority:1;size:36;not null" json:"user_id"`
	SkillName string         `gorm:"column:skill_name;index:ux_skill_frontmatter_user_name,priority:2;size:255;not null" json:"skill_name"`
	Raw       string         `gorm:"type:text;not null" json:"raw"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
```

- [ ] **Step 2: Change DAO queries to always scope by `(user_id, skill_name)`**

```go
func (d *SkillFrontmatterDAO) ReplaceByUserIDAndSkillName(frontmatter *models.SkillFrontmatter) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND skill_name = ?", frontmatter.UserID, frontmatter.SkillName).
			Delete(&models.SkillFrontmatter{}).Error; err != nil {
			return err
		}
		return tx.Create(frontmatter).Error
	})
}
```

- [ ] **Step 3: Thread `userID` through the frontmatter service and adapter**

```go
func (s *SkillFrontmatterService) GetByUserIDAndName(userID, name string) (*models.SkillFrontmatter, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(name) == "" {
		return nil, errors.New("user_id and skill name are required")
	}
	return dao.SkillFrontmatter.GetLatestByUserIDAndName(userID, name)
}
```

- [ ] **Step 4: Change the skill repository root builder to use the user id from context**

```go
func (d *SkillStoreDAO) RootURI(ctx context.Context) (string, error) {
	userID, err := OpenVikingUserIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(UserSkillRootURI(userID), "/"), nil
}

func (d *SkillStoreDAO) BuildURI(ctx context.Context, skillPath string) (string, error) {
	root, err := d.RootURI(ctx)
	if err != nil {
		return "", err
	}
	skillPath = strings.Trim(skillPath, "/")
	if skillPath == "" {
		return root, nil
	}
	parts := strings.Split(skillPath, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	return root + "/" + strings.Join(cleaned, "/"), nil
}
```

- [ ] **Step 5: Update the middleware repository interface so `BuildURI` is context-aware**

```go
type SkillRepository interface {
	ListSkillNames(ctx context.Context) ([]string, error)
	ListFilesInDirectory(ctx context.Context, skillName string, relPath string) ([]dao.SkillFileEntry, error)
	ReadSummary(ctx context.Context, skillName string) (string, error)
	ReadFile(ctx context.Context, skillPath string) (string, error)
	BuildURI(ctx context.Context, skillPath string) (string, error)
}
```

- [ ] **Step 6: Update the middleware backend to fetch frontmatter with the current user**

```go
userID, err := dao.OpenVikingUserIDFromContext(ctx)
if err != nil {
	return einoSkill.Skill{}, err
}
record, err := b.store.GetByUserIDAndName(userID, name)
```

- [ ] **Step 7: Verify the old global skill root is no longer used in code paths**

Run: `rg -n "skills_root|viking://agent/skills" openIntern_backend/internal`

Expected: only configuration structs or comments remain; runtime path builders no longer depend on the global root

### Task 4: Pass The Authenticated User Into Skill HTTP And Agent Runtime Paths

**Files:**
- Modify: `openIntern_backend/internal/controllers/skill.go`
- Modify: `openIntern_backend/internal/services/agent/agent_entry.go`
- Modify: `openIntern_backend/internal/services/agent/agent_init.go`
- Modify: `openIntern_backend/internal/services/middlewares/skill/scope.go`

- [ ] **Step 1: Wrap controller request contexts with `WithOpenVikingUserID` before calling skill DAO/service methods**

```go
userID, ok := getAuthUser(c)
if !ok {
	response.Unauthorized(c)
	return
}
ctx := dao.WithOpenVikingUserID(c.Request.Context(), userID)
```

- [ ] **Step 2: Ensure imported frontmatter records store the authenticated user id**

```go
entry := models.SkillFrontmatter{
	UserID:    userID,
	SkillName: frontmatter.Name,
	Raw:       frontmatter.Raw,
}
```

- [ ] **Step 3: Inject the OpenViking user id into the agent runtime context before skill middleware executes**

```go
ctx = dao.WithOpenVikingUserID(ctx, ownerID)
ctx = context.WithValue(ctx, builtinTool.ContextKeyUserID, ownerID)
```

- [ ] **Step 4: Update scoped skill backends to keep the user-scoped lookup contract intact**

```go
baseDir, err := b.repo.BuildURI(ctx, record.SkillName)
if err != nil {
	return einoSkill.Skill{}, err
}
```

- [ ] **Step 5: Verify skill metadata endpoints still compile after user-scoped method changes**

Run: `go build ./...`

Expected: build succeeds without output

### Task 5: Make Knowledge Bases User-Private Under The Resources Namespace

**Files:**
- Modify: `openIntern_backend/internal/dao/knowledge_base.go`
- Modify: `openIntern_backend/internal/services/kb/service.go`
- Modify: `openIntern_backend/internal/controllers/kb.go`

- [ ] **Step 1: Replace the global knowledge base root with a user-private root**

```go
func (d *KnowledgeBaseDAO) RootURI(userID string) string {
	return UserKnowledgeBaseRootURI(userID)
}

func (d *KnowledgeBaseDAO) URI(userID, name string) string {
	return UserKnowledgeBaseURI(userID, name)
}
```

- [ ] **Step 2: Change list/tree/ingest/delete helpers to require `userID`**

```go
func (d *KnowledgeBaseDAO) List(ctx context.Context, userID string) ([]KnowledgeBaseItem, error) {
	root := d.RootURI(userID)
	entries, err := listEntries(ctx, root, false)
	if err != nil {
		return nil, err
	}
	items := make([]KnowledgeBaseItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir {
			continue
		}
		rel := strings.Trim(relativePath(root, entry.Path), "/")
		if rel == "" {
			rel = strings.Trim(entry.Name, "/")
		}
		if rel == "" {
			continue
		}
		items = append(items, KnowledgeBaseItem{
			Name: rel,
			URI:  d.URI(userID, rel),
		})
	}
	return items, nil
}
```

- [ ] **Step 3: Thread `userID` through the KB service**

```go
func (s *Service) List(ctx context.Context, userID string) ([]Item, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	items, err := dao.KnowledgeBase.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]Item, 0, len(items))
	for _, item := range items {
		result = append(result, Item{Name: item.Name, URI: item.URI})
	}
	return result, nil
}

func (s *Service) Import(ctx context.Context, userID, rawName string, fileHeader *multipart.FileHeader) (*AsyncResult, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	kbName, err := normalizeKnowledgeBaseName(rawName)
	if err != nil {
		return nil, err
	}
	if err := dao.KnowledgeBase.Ingest(ctx, rootDir, dao.KnowledgeBase.URI(userID, kbName), false, 0); err != nil {
		return nil, err
	}
	return &AsyncResult{Name: kbName, Status: "accepted", Async: true}, nil
}

func (s *Service) ReadContent(ctx context.Context, userID, rawURI string) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	uri, err := normalizeEntryURI(userID, rawURI)
	if err != nil {
		return "", err
	}
	return dao.KnowledgeBase.ReadContent(ctx, uri)
}
```

- [ ] **Step 4: Reject any KB URI outside the current user prefix**

```go
func normalizeEntryURI(userID, rawURI string) (string, error) {
	uri := strings.TrimSpace(rawURI)
	root := dao.KnowledgeBase.RootURI(userID)
	if !strings.HasPrefix(uri, strings.TrimRight(root, "/")+"/") {
		return "", fmt.Errorf("%w: uri is outside the current user knowledge base root", ErrInvalidInput)
	}
	return uri, nil
}
```

- [ ] **Step 5: Update controllers to read `user_id` from Gin context and pass it explicitly**

```go
userID := strings.TrimSpace(c.GetString("user_id"))
if userID == "" {
	response.Unauthorized(c)
	return
}
items, err := kbsvc.KnowledgeBase.List(c.Request.Context(), userID)
```

- [ ] **Step 6: Verify no KB operation still points at the global resources root**

Run: `rg -n "viking://resources/" openIntern_backend/internal/dao openIntern_backend/internal/services/kb openIntern_backend/internal/controllers/kb.go`

Expected: only user-scoped root builder code remains, with `users/{user_id}/kbs`

### Task 6: Run Final Build And Multi-User Manual Checks

**Files:**
- Modify: `docs/superpowers/specs/2026-04-06-openviking-user-private-context-design.md` (only if implementation reveals a spec mismatch)

- [ ] **Step 1: Run the backend build**

Run: `go build ./...`

Expected: build succeeds without output

- [ ] **Step 2: Manually verify user-scoped skills and KB behavior with two accounts**

Run:

```bash
# user A 登录后导入一个 skill、创建一个知识库
# user B 登录后访问 /v1/skills/meta 和 /v1/kbs
```

Expected:
- user B 看不到 user A 的 skills
- user B 看不到 user A 的知识库

- [ ] **Step 3: Manually verify user-scoped memory retrieval**

Run:

```bash
# user A 触发一段会命中其 memory 的对话
# user B 使用相同问题访问聊天接口
```

Expected:
- user A 能命中自己的 user memory
- user B 不会读到 user A 的 user memory

- [ ] **Step 4: Re-scan for obsolete default user paths**

Run: `rg -n "viking://user/default/memories|viking://agent/default/memories|viking://agent/skills" openIntern_backend`

Expected:
- `viking://user/default/memories` 不再出现在运行时代码中
- `viking://agent/skills` 不再出现在运行时代码中
- `viking://agent/default/memories` 若仍存在，只保留在明确声明“暂不处理 agent memory”的代码注释或常量中
