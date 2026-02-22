package controllers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"openIntern/internal/models"
	"openIntern/internal/response"
	"openIntern/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

type skillFileItem struct {
	ID   string    `json:"id"`
	Type string    `json:"type"`
	Size int64     `json:"size"`
	Date time.Time `json:"date"`
}

type openVikingResponse struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
	Error  any             `json:"error"`
}

func ListSkillFiles(c *gin.Context) {
	rawPath := c.DefaultQuery("path", "/")
	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	cleaned, err := cleanSkillPath(rawPath)
	if err != nil {
		response.BadRequest(c)
		return
	}
	relPath := strings.TrimPrefix(cleaned, "/")
	if relPath == "" {
		response.BadRequest(c)
		return
	}
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	skillURI, err := buildSkillURI(relPath)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	entries, err := openVikingList(c, skillURI, true)
	if err != nil {
		if isOpenVikingNotFound(err) {
			response.JSONSuccess(c, http.StatusOK, []skillFileItem{})
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	items := make([]skillFileItem, 0, len(entries))
	for _, entry := range entries {
		if openVikingEntryIsDir(entry) {
			continue
		}
		entryPath := openVikingEntryString(entry, "path", "uri")
		entryName := openVikingEntryString(entry, "name")
		rel := openVikingRelativePath(skillURI, entryPath)
		if rel == "" {
			rel = entryName
		}
		if rel == "" {
			continue
		}
		items = append(items, skillFileItem{
			ID:   "/" + path.Join(relPath, rel),
			Type: openVikingEntryString(entry, "type", "kind"),
			Size: openVikingEntryInt64(entry, "size"),
			Date: openVikingEntryTime(entry, "mtime", "modified_at", "date"),
		})
	}
	response.JSONSuccess(c, http.StatusOK, items)
}

func ReadSkillFile(c *gin.Context) {
	rawPath := c.Query("path")
	if strings.TrimSpace(rawPath) == "" {
		response.BadRequest(c)
		return
	}

	cleaned, err := cleanSkillPath(rawPath)
	if err != nil {
		response.BadRequest(c)
		return
	}
	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	relPath := strings.TrimPrefix(cleaned, "/")
	if relPath == "" {
		response.BadRequest(c)
		return
	}
	targetURI, err := buildSkillURI(relPath)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	content, err := openVikingReadContent(c, targetURI)
	if err != nil {
		if isOpenVikingNotFound(err) {
			response.NotFound(c, "file not found")
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"path":    cleaned,
		"content": content,
	})
}

func ImportSkill(c *gin.Context) {
	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c)
		return
	}
	if strings.ToLower(filepath.Ext(fileHeader.Filename)) != ".zip" {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "只支持 zip 文件")
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c)
		return
	}
	defer src.Close()
	tempDir, err := os.MkdirTemp("", "skill-import-*")
	if err != nil {
		response.InternalError(c)
		return
	}
	defer os.RemoveAll(tempDir)
	zipPath := filepath.Join(tempDir, "upload.zip")
	dst, err := os.Create(zipPath)
	if err != nil {
		response.InternalError(c)
		return
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		response.InternalError(c)
		return
	}
	if err := dst.Close(); err != nil {
		response.InternalError(c)
		return
	}
	extractDir := filepath.Join(tempDir, "unzipped")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		response.InternalError(c)
		return
	}
	if err := unzipSkill(zipPath, extractDir); err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	rootDir, _, err := resolveSkillRoot(extractDir, fileHeader.Filename)
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	skillFile, err := findSkillMarkdown(rootDir)
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	content, err := os.ReadFile(skillFile)
	if err != nil {
		response.InternalError(c)
		return
	}
	frontmatter, err := parseFrontmatter(string(content))
	if err != nil {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(frontmatter.Name) == "" {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "frontmatter 缺少 name")
		return
	}
	entry := models.SkillFrontmatter{
		SkillName: frontmatter.Name,
		Raw:       frontmatter.Raw,
	}
	if err := services.SkillFrontmatter.CreateOrReplaceByName(&entry); err != nil {
		response.InternalError(c)
		return
	}
	if err := openVikingAddSkill(c, rootDir); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"name": frontmatter.Name,
	})
}

func DeleteSkill(c *gin.Context) {
	name, err := parseSkillName(c.Param("name"))
	if err != nil {
		response.BadRequest(c)
		return
	}
	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	targetURI, err := buildSkillURI(name)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	if !strings.HasSuffix(targetURI, "/") {
		targetURI += "/"
	}
	if err := openVikingDeleteSkill(c, targetURI); err != nil {
		if isOpenVikingNotFound(err) {
			response.NotFound(c, "skill not found")
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	if err := services.SkillFrontmatter.DeleteByName(name); err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"name": name,
	})
}

func CreateSkillMeta(c *gin.Context) {
	response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "skill meta is read-only")
}

func GetSkillMetaByName(c *gin.Context) {
	name, err := parseSkillName(c.Param("name"))
	if err != nil {
		response.BadRequest(c)
		return
	}

	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	entry, err := services.SkillFrontmatter.GetByName(name)
	if err != nil {
		response.NotFound(c, "skill not found")
		return
	}
	parsed, err := parseFrontmatterRaw(entry.Raw)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, models.Skill{
		Name:        parsed.Name,
		Description: parsed.Description,
		Icon:        parsed.Icon,
		Frontmatter: entry.Raw,
	})
}

func UpdateSkillMeta(c *gin.Context) {
	response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "skill meta is read-only")
}

func DeleteSkillMeta(c *gin.Context) {
	response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "skill meta is read-only")
}

func ReadSkillContent(c *gin.Context) {
	name, err := parseSkillName(c.Param("name"))
	if err != nil {
		response.BadRequest(c)
		return
	}

	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	rawPath := c.Query("path")
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		response.BadRequest(c)
		return
	}
	cleaned, err := cleanSkillPath(decoded)
	if err != nil {
		response.BadRequest(c)
		return
	}
	rel := strings.TrimPrefix(cleaned, "/")
	if rel == "" {
		rel = "SKILL.md"
	}
	skillURI, err := buildSkillURI(name)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	targetURI := strings.TrimRight(skillURI, "/") + "/" + rel
	content, err := openVikingReadContent(c, targetURI)
	if err != nil {
		if isOpenVikingNotFound(err) {
			response.NotFound(c, "file not found")
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"name":    name,
		"content": content,
	})
}

func ListSkills(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	skills, total, err := listOpenVikingSkills(c, keyword, page, pageSize)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	applySkillFrontmatter(skills)
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  skills,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func listOpenVikingSkills(c *gin.Context, keyword string, page int, pageSize int) ([]models.Skill, int64, error) {
	root := services.OpenViking.SkillsRoot()
	entries, err := openVikingList(c, root, false)
	if err != nil {
		return nil, 0, err
	}
	skills := make([]models.Skill, 0, len(entries))
	for _, entry := range entries {
		if !openVikingEntryIsDir(entry) {
			continue
		}
		entryPath := openVikingEntryString(entry, "path", "uri")
		entryName := openVikingEntryString(entry, "name")
		skillPath := openVikingRelativePath(root, entryPath)
		if skillPath == "" {
			skillPath = entryName
		}
		if skillPath == "" {
			continue
		}
		if strings.Contains(skillPath, "/") {
			skillPath = strings.Split(skillPath, "/")[0]
		}
		skills = append(skills, models.Skill{
			Name: skillPath,
		})
	}
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		filtered := make([]models.Skill, 0, len(skills))
		for _, skill := range skills {
			if matchSkillKeyword(skill, keyword) {
				filtered = append(filtered, skill)
			}
		}
		skills = filtered
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	total := int64(len(skills))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(skills) {
		return []models.Skill{}, total, nil
	}
	end := start + pageSize
	if end > len(skills) {
		end = len(skills)
	}
	return skills[start:end], total, nil
}

func applySkillFrontmatter(skills []models.Skill) {
	if len(skills) == 0 {
		return
	}
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		if strings.TrimSpace(skill.Name) != "" {
			names = append(names, skill.Name)
		}
	}
	if len(names) == 0 {
		return
	}
	frontmatters, err := services.SkillFrontmatter.ListByNames(names)
	if err != nil || len(frontmatters) == 0 {
		return
	}
	index := make(map[string]models.SkillFrontmatter, len(frontmatters))
	for _, item := range frontmatters {
		if item.SkillName != "" {
			index[item.SkillName] = item
		}
	}
	for i := range skills {
		item, ok := index[skills[i].Name]
		if !ok {
			continue
		}
		parsed, err := parseFrontmatterRaw(item.Raw)
		if err != nil {
			continue
		}
		if strings.TrimSpace(parsed.Name) != "" {
			skills[i].Name = parsed.Name
		}
		if strings.TrimSpace(parsed.Description) != "" {
			skills[i].Description = parsed.Description
		}
		if strings.TrimSpace(parsed.Icon) != "" {
			skills[i].Icon = parsed.Icon
		}
		skills[i].Frontmatter = item.Raw
	}
}

func buildSkillURI(skillPath string) (string, error) {
	root := strings.TrimRight(strings.TrimSpace(services.OpenViking.SkillsRoot()), "/")
	if root == "" {
		return "", errors.New("openviking skills_root not configured")
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
	if len(cleaned) == 0 {
		return root, nil
	}
	return root + "/" + strings.Join(cleaned, "/"), nil
}

func openVikingList(c *gin.Context, uri string, recursive bool) ([]map[string]any, error) {
	params := url.Values{}
	params.Set("uri", uri)
	if recursive {
		params.Set("recursive", "true")
	}
	params.Set("output", "agent")
	body, err := openVikingGet(c, "/api/v1/fs/ls", params)
	if err != nil {
		return nil, err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return nil, errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return nil, errors.New(msg)
		}
	}
	if len(resp.Result) == 0 {
		return []map[string]any{}, nil
	}
	var result []map[string]any
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func openVikingReadContent(c *gin.Context, uri string) (string, error) {
	params := url.Values{}
	params.Set("uri", uri)
	body, err := openVikingGet(c, "/api/v1/content/read", params)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "", nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		if result, ok := payload["result"]; ok {
			switch value := result.(type) {
			case string:
				return value, nil
			case map[string]any:
				if content, ok := value["content"]; ok {
					return fmt.Sprint(content), nil
				}
				if text, ok := value["text"]; ok {
					return fmt.Sprint(text), nil
				}
			}
		}
		if content, ok := payload["content"]; ok {
			return fmt.Sprint(content), nil
		}
	}
	return string(body), nil
}

func openVikingGet(c *gin.Context, endpoint string, params url.Values) ([]byte, error) {
	if !services.OpenViking.Configured() {
		return nil, errors.New("openviking not configured")
	}
	baseURL := strings.TrimRight(services.OpenViking.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("openviking base_url not configured")
	}
	requestURL := baseURL + endpoint
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if apiKey := services.OpenViking.APIKey(); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	client := services.OpenViking.Client()
	if client == nil {
		return nil, errors.New("openviking client not configured")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	return body, nil
}

func openVikingPost(c *gin.Context, endpoint string, payload any) ([]byte, error) {
	if !services.OpenViking.Configured() {
		return nil, errors.New("openviking not configured")
	}
	baseURL := strings.TrimRight(services.OpenViking.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("openviking base_url not configured")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	requestURL := baseURL + endpoint
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey := services.OpenViking.APIKey(); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	client := services.OpenViking.Client()
	if client == nil {
		return nil, errors.New("openviking client not configured")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	return respBody, nil
}

func openVikingDelete(c *gin.Context, endpoint string, params url.Values) ([]byte, error) {
	if !services.OpenViking.Configured() {
		return nil, errors.New("openviking not configured")
	}
	baseURL := strings.TrimRight(services.OpenViking.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("openviking base_url not configured")
	}
	requestURL := baseURL + endpoint
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodDelete, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if apiKey := services.OpenViking.APIKey(); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	client := services.OpenViking.Client()
	if client == nil {
		return nil, errors.New("openviking client not configured")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, errors.New(msg)
	}
	return body, nil
}

func openVikingDeleteSkill(c *gin.Context, uri string) error {
	params := url.Values{}
	params.Set("uri", uri)
	params.Set("recursive", "true")
	body, err := openVikingDelete(c, "/api/v1/fs", params)
	if err != nil {
		return err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return errors.New(msg)
		}
	}
	return nil
}

func openVikingAddSkill(c *gin.Context, skillDir string) error {
	body, err := openVikingPost(c, "/api/v1/skills", map[string]any{
		"data": skillDir,
	})
	if err != nil {
		return err
	}
	var resp openVikingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.Status) != "" && strings.ToLower(resp.Status) != "ok" {
		return errors.New("openviking error")
	}
	if resp.Error != nil {
		if msg := strings.TrimSpace(fmt.Sprint(resp.Error)); msg != "" && msg != "<nil>" {
			return errors.New(msg)
		}
	}
	return nil
}

type skillFrontmatterPayload struct {
	Name        string
	Description string
	Icon        string
	Raw         string
}

func parseFrontmatter(content string) (skillFrontmatterPayload, error) {
	trimmed := strings.TrimLeft(content, "\ufeff\r\n\t ")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 || strings.TrimSpace(strings.TrimRight(lines[0], "\r")) != "---" {
		return skillFrontmatterPayload{}, errors.New("missing frontmatter")
	}
	var fmLines []string
	endFound := false
	for i := 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if strings.TrimSpace(line) == "---" {
			endFound = true
			break
		}
		fmLines = append(fmLines, line)
	}
	if !endFound {
		return skillFrontmatterPayload{}, errors.New("invalid frontmatter")
	}
	raw := strings.TrimSpace(strings.Join(fmLines, "\n"))
	if raw == "" {
		return skillFrontmatterPayload{}, errors.New("empty frontmatter")
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(raw), &data); err != nil {
		return skillFrontmatterPayload{}, err
	}
	get := func(key string) string {
		value, ok := data[key]
		if !ok {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return skillFrontmatterPayload{
		Name:        get("name"),
		Description: get("description"),
		Icon:        get("icon"),
		Raw:         raw,
	}, nil
}

func parseFrontmatterRaw(raw string) (skillFrontmatterPayload, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return skillFrontmatterPayload{}, errors.New("empty frontmatter")
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(trimmed), &data); err != nil {
		return skillFrontmatterPayload{}, err
	}
	get := func(key string) string {
		value, ok := data[key]
		if !ok {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return skillFrontmatterPayload{
		Name:        get("name"),
		Description: get("description"),
		Icon:        get("icon"),
		Raw:         trimmed,
	}, nil
}

func resolveSkillRoot(extractDir string, zipName string) (string, string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", "", err
	}
	filtered := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "__MACOSX") {
			continue
		}
		filtered = append(filtered, entry)
	}
	if len(filtered) == 1 && filtered[0].IsDir() {
		root := filepath.Join(extractDir, filtered[0].Name())
		return root, filtered[0].Name(), nil
	}
	base := strings.TrimSuffix(filepath.Base(zipName), filepath.Ext(zipName))
	base = strings.TrimSpace(base)
	if base == "" {
		return "", "", errors.New("invalid zip name")
	}
	return extractDir, base, nil
}

func findSkillMarkdown(rootDir string) (string, error) {
	rootFile := filepath.Join(rootDir, "SKILL.md")
	if info, err := os.Stat(rootFile); err == nil && !info.IsDir() {
		return rootFile, nil
	}
	var found string
	walkErr := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "__MACOSX")) {
			return filepath.SkipDir
		}
		if !d.IsDir() && name == "SKILL.md" {
			found = path
			return errSkillFound
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errSkillFound) {
		return "", walkErr
	}
	if found == "" {
		return "", errors.New("SKILL.md not found")
	}
	return found, nil
}

func unzipSkill(zipPath string, dest string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		name := file.Name
		if strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		cleaned := filepath.Clean(name)
		if cleaned == "." || strings.HasPrefix(cleaned, "..") {
			return errors.New("invalid zip entry")
		}
		target := filepath.Join(dest, cleaned)
		rel, err := filepath.Rel(dest, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return errors.New("invalid zip entry")
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return errors.New("invalid zip entry")
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			in.Close()
			return err
		}
		if err := out.Close(); err != nil {
			in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
	}
	return nil
}

func openVikingRelativePath(rootURI string, entryPath string) string {
	if entryPath == "" {
		return ""
	}
	root := strings.TrimRight(rootURI, "/")
	entryPath = strings.TrimPrefix(entryPath, root)
	return strings.TrimPrefix(entryPath, "/")
}

func openVikingEntryString(entry map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case string:
				if v != "" {
					return v
				}
			case fmt.Stringer:
				str := v.String()
				if str != "" {
					return str
				}
			default:
				str := strings.TrimSpace(fmt.Sprint(value))
				if str != "" {
					return str
				}
			}
		}
	}
	return ""
}

func openVikingEntryBool(entry map[string]any, keys ...string) bool {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case bool:
				return v
			case string:
				if strings.EqualFold(strings.TrimSpace(v), "true") {
					return true
				}
			}
		}
	}
	return false
}

func openVikingEntryIsDir(entry map[string]any) bool {
	if openVikingEntryBool(entry, "is_dir", "isDir", "dir", "directory", "isDirectory") {
		return true
	}
	t := strings.ToLower(openVikingEntryString(entry, "type", "kind"))
	return t == "dir" || t == "directory" || t == "folder"
}

func openVikingEntryInt64(entry map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case int64:
				return v
			case int:
				return int64(v)
			case float64:
				return int64(v)
			case json.Number:
				if n, err := v.Int64(); err == nil {
					return n
				}
			case string:
				if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
					return n
				}
			}
		}
	}
	return 0
}

func openVikingEntryTime(entry map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		if value, ok := entry[key]; ok {
			switch v := value.(type) {
			case time.Time:
				return v
			case string:
				if parsed, err := parseOpenVikingTime(v); err == nil {
					return parsed
				}
			case float64:
				return time.Unix(int64(v), 0)
			case json.Number:
				if n, err := v.Int64(); err == nil {
					return time.Unix(n, 0)
				}
			}
		}
	}
	return time.Time{}
}

func parseOpenVikingTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty time")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(n, 0), nil
	}
	return time.Time{}, errors.New("invalid time")
}

func isOpenVikingNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}

var errInvalidPath = errors.New("invalid path")
var errOutOfScope = errors.New("out of scope")
var errSkillFound = errors.New("skill found")

func getAuthUser(c *gin.Context) (string, string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", "", false
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", "", false
	}
	claims, err := services.ParseToken(parts[1])
	if err != nil {
		return "", "", false
	}
	refreshedToken, expiresAt, err := services.GenerateToken(claims.UserID, claims.Role)
	if err == nil {
		c.Header("X-Access-Token", refreshedToken)
		c.Header("X-Token-Expires", strconv.FormatInt(expiresAt, 10))
	}
	return claims.UserID, claims.Role, true
}

func skillBaseDir() (string, error) {
	wd, err := os.Getwd()
	if err == nil {
		candidates := []string{
			filepath.Join(wd, "internal", "skills"),
			filepath.Join(wd, "openIntern_backend", "internal", "skills"),
		}
		for _, candidate := range candidates {
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				return filepath.Abs(candidate)
			}
		}
	}
	base := filepath.Join("internal", "skills")
	return filepath.Abs(base)
}

func cleanSkillPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/", nil
	}
	if strings.Contains(p, "..") {
		return "", errors.New("invalid path")
	}
	cleaned := path.Clean("/" + p)
	return cleaned, nil
}

func resolveSkillContentPath(skillDir string, rawPath string) (string, error) {
	if strings.TrimSpace(rawPath) == "" {
		return filepath.Join(skillDir, "SKILL.md"), nil
	}
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		return "", errInvalidPath
	}
	cleaned, err := cleanSkillPath(decoded)
	if err != nil {
		return "", errInvalidPath
	}
	rel := strings.TrimPrefix(cleaned, "/")
	if rel == "" {
		return "", errInvalidPath
	}
	target := filepath.Join(skillDir, rel)
	skillDirAbs, err := filepath.Abs(skillDir)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	relTo, err := filepath.Rel(skillDirAbs, targetAbs)
	if err != nil {
		return "", errOutOfScope
	}
	if relTo == "." || strings.HasPrefix(relTo, "..") {
		return "", errOutOfScope
	}
	return targetAbs, nil
}

func parseSkillName(name string) (string, error) {
	decoded, err := url.PathUnescape(name)
	if err != nil {
		return "", err
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", errors.New("invalid name")
	}
	if strings.Contains(decoded, "..") || strings.Contains(decoded, "/") || strings.Contains(decoded, "\\") {
		return "", errors.New("invalid name")
	}
	return decoded, nil
}

func resolveSkillPath(cleaned string) (string, string, error) {
	relPath := strings.TrimPrefix(cleaned, "/")
	baseDir, err := skillBaseDir()
	if err != nil {
		return "", "", err
	}
	absPath := filepath.Join(baseDir, filepath.FromSlash(relPath))
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", "", err
	}
	targetAbs, err := filepath.Abs(absPath)
	if err != nil {
		return "", "", err
	}
	relTo, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(relTo, "..") {
		return "", "", errors.New("out of scope")
	}
	return targetAbs, relPath, nil
}

func listSkillMetas(baseDir string, keyword string, page int, pageSize int) ([]models.Skill, int64, error) {
	skills, err := collectSkills(baseDir)
	if err != nil {
		return nil, 0, err
	}

	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		filtered := make([]models.Skill, 0, len(skills))
		for _, skill := range skills {
			if matchSkillKeyword(skill, keyword) {
				filtered = append(filtered, skill)
			}
		}
		skills = filtered
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	total := int64(len(skills))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(skills) {
		return []models.Skill{}, total, nil
	}
	end := start + pageSize
	if end > len(skills) {
		end = len(skills)
	}
	return skills[start:end], total, nil
}

func collectSkills(baseDir string) ([]models.Skill, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Skill{}, nil
		}
		return nil, err
	}

	skills := make([]models.Skill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		skillPath := path.Join(dirName)
		skillFile := filepath.Join(baseDir, dirName, "SKILL.md")
		info, err := os.Stat(skillFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		skill, err := readSkillFromFile(skillFile, skillPath)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

func readSkillFromFile(_ string, skillPath string) (models.Skill, error) {
	return models.Skill{
		Name:        skillPath,
		Description: "",
		Icon:        "",
	}, nil
}

func matchSkillKeyword(skill models.Skill, keyword string) bool {
	keyword = strings.ToLower(keyword)
	fields := []string{
		skill.Name,
		skill.Description,
		skill.Frontmatter,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), keyword) {
			return true
		}
	}
	return false
}
