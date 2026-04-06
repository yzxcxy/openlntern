# Plugin Default Icon MinIO Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将仓库内的默认插件图标切换为 MinIO 固定公共对象，并在新环境初始化脚本中自动预热该对象。

**Architecture:** 默认插件图标继续使用后端存储的 `public/...` 对象 key，但对前端返回时统一解析成后端资产代理 `/v1/assets/...`，避免直连私有 MinIO。初始化职责放在 `scripts/init-dev-data.sh`，通过固定 object key 保证所有新建插件默认图标一致，并通过存在性检查保证脚本幂等。

**Tech Stack:** Bash, MinIO `mc`, Go backend runtime config

---

### Task 1: Relocate The Default Icon Asset

**Files:**
- Create: `openIntern_backend/assets/plugin/default-icon.jpg`
- Remove: `tool.jpg`

- [ ] **Step 1: Move the provided image into the backend asset directory**

```bash
mkdir -p openIntern_backend/assets/plugin
mv tool.jpg openIntern_backend/assets/plugin/default-icon.jpg
```

- [ ] **Step 2: Verify the asset exists at the new location**

Run: `file openIntern_backend/assets/plugin/default-icon.jpg`
Expected: output identifies a JPEG image file

### Task 2: Switch The Default Plugin Icon To A Stable MinIO Object Key

**Files:**
- Modify: `openIntern_backend/config.yaml`

- [ ] **Step 1: Update the plugin default icon config**

```yaml
plugin:
    builtin_manifest_path: builtin_plugins.yaml
    default_icon_url: public/plugin/icon/default-plugin.jpg
```

- [ ] **Step 2: Verify the config now points to the public object key**

Run: `rg -n "default_icon_url" openIntern_backend/config.yaml`
Expected: one match with `public/plugin/icon/default-plugin.jpg`

### Task 3: Initialize The Default Icon Object During New Environment Setup

**Files:**
- Modify: `scripts/init-dev-data.sh`

- [ ] **Step 1: Add script variables for the local asset path and object key**

```bash
DEFAULT_PLUGIN_ICON_PATH="${ROOT_DIR}/openIntern_backend/assets/plugin/default-icon.jpg"
DEFAULT_PLUGIN_ICON_OBJECT_KEY="${OPENINTERN_DEFAULT_PLUGIN_ICON_OBJECT_KEY:-public/plugin/icon/default-plugin.jpg}"
```

- [ ] **Step 2: Add an idempotent MinIO upload helper**

```bash
ensure_minio_object_from_file() {
  local object_key="$1"
  local file_path="$2"
  # Validate the source file, then upload only when mc stat says the object does not exist.
}
```

- [ ] **Step 3: Invoke the helper after bucket creation**

```bash
echo "初始化默认插件图标..."
ensure_minio_object_from_file "${DEFAULT_PLUGIN_ICON_OBJECT_KEY}" "${DEFAULT_PLUGIN_ICON_PATH}"
```

- [ ] **Step 4: Verify the script syntax**

Run: `bash -n scripts/init-dev-data.sh`
Expected: no output and exit code `0`

### Task 4: Route Public Asset URLs Through The Backend

**Files:**
- Modify: `openIntern_backend/internal/services/storage/file.go`

- [ ] **Step 1: Change public object URL resolution to return the backend asset route**

```go
func BuildPublicObjectURL(publicObjectKey string) (string, error) {
	key := normalizeObjectKey(publicObjectKey)
	if !strings.HasPrefix(key, "public/") {
		return "", errors.New("public object key must start with public/")
	}
	return backendAssetRoutePrefix + key, nil
}
```

- [ ] **Step 2: Verify backend compilation**

Run: `go build ./...`
Expected: build succeeds without output
### Task 5: Document The New Initialization Behavior

**Files:**
- Modify: `docs/local-development.md`

- [ ] **Step 1: Update the one-time initialization description**

```md
- upload the default plugin icon object to MinIO
```

- [ ] **Step 2: Add a note about the fixed default icon object key**

```md
- The default icon for newly created plugins now points to the fixed MinIO object key `public/plugin/icon/default-plugin.jpg`, and `scripts/init-dev-data.sh` will initialize that object for a new local environment.
```

- [ ] **Step 3: Re-read the updated section for consistency**

Run: `sed -n '18,80p' docs/local-development.md`
Expected: initialization flow and notes mention the default plugin icon object
