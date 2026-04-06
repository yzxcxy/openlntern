# OpenViking Remote Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace local-path-based OpenViking imports with pure HTTP remote upload and `temp_file_id` imports for knowledge bases and skills.

**Architecture:** Keep the existing controller and service APIs stable, but move the import boundary inside the OpenViking DAO layer. Local files and directories remain temporary artifacts inside the backend container only; the backend uploads them to OpenViking with `temp_upload`, then calls resource or skill import APIs using `temp_file_id`.

**Tech Stack:** Go, Gin, standard library `archive/zip`, `mime/multipart`, existing OpenViking HTTP client wrappers

---

## File Map

- Modify: `openIntern_backend/internal/database/context_store.go`
  - Add multipart temp upload, archive packaging, and temp-file import helpers.
- Modify: `openIntern_backend/internal/database/context_store_test.go`
  - Add unit tests for new payload selection and log extraction behavior.
- Modify: `openIntern_backend/internal/dao/context_store.go`
  - Switch resource and skill imports from local path payloads to `temp_file_id` payloads.
- Create: `openIntern_backend/internal/dao/context_store_test.go`
  - Add focused tests for resource/skill import payload generation and directory-vs-file routing.
- Modify: `openIntern_backend/internal/services/kb/service.go`
  - Keep existing import flow, but rely on the DAO’s remote import behavior.
- Modify: `openIntern_backend/internal/controllers/skill.go`
  - Keep existing validation flow, but rely on the DAO’s remote import behavior.
- Modify: `README.md`
  - Document HTTP upload based OpenViking imports.
- Modify: `docs/local-development.md`
  - Clarify that OpenViking no longer reads backend-local paths.

## Execution Constraints

- Repository rule: do not run `go test` unless the user explicitly allows it.
- TDD adaptation for this repo: write tests first, but leave test execution commands as deferred verification steps until approval is granted.
- Allowed verification in this round: static code inspection and `go build`-style compilation checks if needed.

### Task 1: Add OpenViking temp upload primitives

**Files:**
- Modify: `openIntern_backend/internal/database/context_store.go`
- Modify: `openIntern_backend/internal/database/context_store_test.go`

- [ ] **Step 1: Write the failing tests for temp upload payload extraction**

```go
func TestExtractPayloadSummarySupportsTempFileID(t *testing.T) {
	target, pathValue := extractPayloadSummary(map[string]any{
		"target":       "viking://resources/demo/",
		"temp_file_id": "tmp_123",
	})
	if target != "viking://resources/demo/" {
		t.Fatalf("unexpected target: %q", target)
	}
	if pathValue != "temp_file_id:tmp_123" {
		t.Fatalf("unexpected path summary: %q", pathValue)
	}
}
```

- [ ] **Step 2: Defer test execution until approval**

Run later after approval: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go test ./internal/database -run 'TestExtractPayloadSummary'`
Expected after implementation: `ok  	openIntern/internal/database`

- [ ] **Step 3: Add multipart temp upload and archive helpers**

```go
type TempUploadResult struct {
	TempFileID string `json:"temp_file_id"`
	FileName   string `json:"file_name"`
}

func (s *ContextStore) UploadTempFile(ctx context.Context, localPath string) (*TempUploadResult, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return s.uploadMultipart(ctx, "/api/v1/resources/temp_upload", body, writer.FormDataContentType())
}
```

- [ ] **Step 4: Add directory archive packaging helper**

```go
func CreateZipArchiveFromDir(rootDir string, archivePath string) error {
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer archiveFile.Close()

	zipWriter := zip.NewWriter(archiveFile)
	defer zipWriter.Close()

	return filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		headerPath := filepath.ToSlash(relPath)
		entry, err := zipWriter.Create(headerPath)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(entry, src)
		return err
	})
}
```

- [ ] **Step 5: Add temp upload response decoding and logging support**

```go
func extractPayloadSummary(payload any) (string, string) {
	data, ok := payload.(map[string]any)
	if !ok {
		return "", ""
	}
	target := strings.TrimSpace(fmt.Sprint(data["target_uri"]))
	if target == "" {
		target = strings.TrimSpace(fmt.Sprint(data["target"]))
	}
	if pathValue, ok := data["path"]; ok {
		return target, sanitizeLogPath(strings.TrimSpace(fmt.Sprint(pathValue)))
	}
	if tempID, ok := data["temp_file_id"]; ok {
		return target, "temp_file_id:" + strings.TrimSpace(fmt.Sprint(tempID))
	}
	return target, ""
}
```

- [ ] **Step 6: Verify code compiles locally**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./internal/database`
Expected: command exits with code `0`

### Task 2: Route resource imports through temp uploads

**Files:**
- Modify: `openIntern_backend/internal/dao/context_store.go`
- Create: `openIntern_backend/internal/dao/context_store_test.go`
- Modify: `openIntern_backend/internal/services/kb/service.go`

- [ ] **Step 1: Write the failing DAO tests for file-vs-directory routing**

```go
func TestBuildAddResourcePayloadFromTempFile(t *testing.T) {
	payload := buildAddResourcePayload("tmp_123", "viking://resources/demo/", false, 0)
	if payload["temp_file_id"] != "tmp_123" {
		t.Fatalf("unexpected temp file id: %#v", payload["temp_file_id"])
	}
	if payload["target"] != "viking://resources/demo/" {
		t.Fatalf("unexpected target: %#v", payload["target"])
	}
	if _, exists := payload["path"]; exists {
		t.Fatalf("path should not be sent once temp_file_id is used")
	}
}
```

- [ ] **Step 2: Defer DAO test execution until approval**

Run later after approval: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go test ./internal/dao -run 'TestBuildAddResourcePayloadFromTempFile'`
Expected after implementation: `ok  	openIntern/internal/dao`

- [ ] **Step 3: Split resource import into upload and import phases**

```go
func addResourceWithRootURI(ctx context.Context, resourcePath string, targetURI string, wait bool, timeoutSeconds float64) (string, error) {
	info, err := os.Stat(resourcePath)
	if err != nil {
		return "", err
	}

	var upload *database.TempUploadResult
	if info.IsDir() {
		upload, err = database.Context.UploadTempArchive(ctx, resourcePath, filepath.Base(resourcePath))
	} else {
		upload, err = database.Context.UploadTempFile(ctx, resourcePath)
	}
	if err != nil {
		return "", err
	}

	return addResourceFromTempFileID(ctx, upload.TempFileID, targetURI, wait, timeoutSeconds)
}
```

- [ ] **Step 4: Add a small payload builder for deterministic tests**

```go
func buildAddResourcePayload(tempFileID string, targetURI string, wait bool, timeoutSeconds float64) map[string]any {
	payload := map[string]any{
		"temp_file_id": strings.TrimSpace(tempFileID),
		"target":       strings.TrimSpace(targetURI),
		"wait":         wait,
	}
	if timeoutSeconds > 0 {
		payload["timeout"] = timeoutSeconds
	}
	return payload
}
```

- [ ] **Step 5: Keep knowledge base service logic unchanged except for comments**

```go
// Import keeps zip extraction locally for validation, then delegates remote upload/import to the DAO layer.
if err := dao.KnowledgeBase.Ingest(ctx, rootDir, dao.KnowledgeBase.URI(kbName), false, 0); err != nil {
	return nil, err
}
```

- [ ] **Step 6: Verify backend package build**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./internal/dao ./internal/services/kb`
Expected: command exits with code `0`

### Task 3: Route skill imports through temp uploads

**Files:**
- Modify: `openIntern_backend/internal/dao/context_store.go`
- Modify: `openIntern_backend/internal/controllers/skill.go`
- Modify: `openIntern_backend/internal/dao/context_store_test.go`

- [ ] **Step 1: Write the failing DAO test for skill temp import payloads**

```go
func TestBuildAddSkillPayloadFromTempFile(t *testing.T) {
	payload := buildAddSkillPayload("tmp_skill_1", false, 0)
	if payload["temp_file_id"] != "tmp_skill_1" {
		t.Fatalf("unexpected temp file id: %#v", payload["temp_file_id"])
	}
	if _, exists := payload["data"]; exists {
		t.Fatalf("data should not be sent once temp_file_id is used")
	}
}
```

- [ ] **Step 2: Defer skill DAO test execution until approval**

Run later after approval: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go test ./internal/dao -run 'TestBuildAddSkillPayloadFromTempFile'`
Expected after implementation: `ok  	openIntern/internal/dao`

- [ ] **Step 3: Replace local-directory skill import with archive upload**

```go
func importSkill(ctx context.Context, rootDir string) error {
	upload, err := database.Context.UploadTempArchive(ctx, rootDir, filepath.Base(rootDir))
	if err != nil {
		return err
	}
	body, err := database.Context.Post(ctx, "/api/v1/skills", buildAddSkillPayload(upload.TempFileID, false, 0))
	if err != nil {
		return err
	}
	return decodeStoreResult(body, nil)
}
```

- [ ] **Step 4: Keep controller validation order stable and document the remote import boundary**

```go
if err := dao.SkillStore.Import(c.Request.Context(), rootDir); err != nil {
	response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
	return
}
```

- [ ] **Step 5: Verify backend package build**

Run: `cd /Users/fqc/project/agent/openIntern/openIntern_backend && go build ./internal/controllers ./internal/dao`
Expected: command exits with code `0`

### Task 4: Update docs for container-safe OpenViking imports

**Files:**
- Modify: `README.md`
- Modify: `docs/local-development.md`

- [ ] **Step 1: Add documentation first-draft changes**

```md
- OpenViking import flows now upload files over HTTP before importing them.
- `openIntern_backend` no longer passes backend-local file paths to OpenViking.
- OpenViking does not require a shared Docker volume with the backend for skill or knowledge-base imports.
```

- [ ] **Step 2: Update the local development notes**

```md
- The OpenViking container may persist its own workspace internally, but import requests from openIntern are sent through HTTP upload APIs.
- Backend-local temp files are never referenced directly by OpenViking.
```

- [ ] **Step 3: Review docs for contradictions with the spec**

Check:
- No sentence claims that OpenViking can read backend-local paths directly
- No sentence implies a required shared volume between backend and OpenViking
- Existing compose instructions still make sense for local development

- [ ] **Step 4: Verify final diff shape**

Run: `cd /Users/fqc/project/agent/openIntern && git diff -- docs/superpowers/specs/2026-04-06-openviking-remote-import-design.md docs/superpowers/plans/2026-04-06-openviking-remote-import.md README.md docs/local-development.md openIntern_backend/internal/database/context_store.go openIntern_backend/internal/database/context_store_test.go openIntern_backend/internal/dao/context_store.go openIntern_backend/internal/dao/context_store_test.go openIntern_backend/internal/controllers/skill.go openIntern_backend/internal/services/kb/service.go`
Expected: diff only covers the remote-import design, plan, DAO/client changes, targeted tests, and docs

## Self-Review

- Spec coverage:
  - Remote HTTP upload path: Task 1, Task 2, Task 3
  - Knowledge base import flow: Task 2
  - Skill import flow: Task 3
  - Docs updates: Task 4
  - No shared volume dependency: Task 1, Task 2, Task 3, Task 4
- Placeholder scan:
  - No `TODO`, `TBD`, or “implement later” placeholders remain
  - Test execution is explicitly deferred because repo instructions forbid `go test` without approval
- Type consistency:
  - `TempUploadResult`, `buildAddResourcePayload`, and `buildAddSkillPayload` are defined once and reused consistently

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-06-openviking-remote-import.md`. Two execution options:

1. Subagent-Driven (recommended) - I dispatch a fresh subagent per task, review between tasks, fast iteration
2. Inline Execution - Execute tasks in this session using executing-plans, batch execution with checkpoints

In this session I can only use option 2 by default, because subagent dispatch needs your explicit request. If you want subagent-driven execution, say so; otherwise continue inline.
