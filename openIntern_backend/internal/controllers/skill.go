package controllers

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"openIntern/internal/dao"
	"openIntern/internal/models"
	"openIntern/internal/response"
	accountsvc "openIntern/internal/services/account"
	skillsvc "openIntern/internal/services/skill"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

type skillFileItem struct {
	ID   string    `json:"id"`
	Type string    `json:"type"`
	Size int64     `json:"size"`
	Date time.Time `json:"date"`
}

func ListSkillFiles(c *gin.Context) {
	rawPath := c.DefaultQuery("path", "/")
	if _, ok := getAuthUser(c); !ok {
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
	if !dao.SkillStore.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "skill storage not configured")
		return
	}
	skillURI, err := dao.SkillStore.BuildURI(relPath)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	entries, err := dao.SkillStore.ListFiles(c.Request.Context(), relPath, true)
	if err != nil {
		if isStoreNotFound(err) {
			response.JSONSuccess(c, http.StatusOK, []skillFileItem{})
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	items := make([]skillFileItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		rel := strings.TrimPrefix(strings.TrimSpace(entry.Path), strings.TrimRight(skillURI, "/")+"/")
		if rel == "" {
			rel = entry.Name
		}
		if rel == "" {
			continue
		}
		items = append(items, skillFileItem{
			ID:   "/" + path.Join(relPath, rel),
			Type: entry.Type,
			Size: entry.Size,
			Date: entry.ModifiedAt,
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
	if _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}
	if !dao.SkillStore.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "skill storage not configured")
		return
	}
	relPath := strings.TrimPrefix(cleaned, "/")
	if relPath == "" {
		response.BadRequest(c)
		return
	}
	content, err := dao.SkillStore.ReadFile(c.Request.Context(), relPath)
	if err != nil {
		if isStoreNotFound(err) {
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
	if _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}
	if !dao.SkillStore.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "skill storage not configured")
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
	if err := skillsvc.SkillFrontmatter.CreateOrReplaceByName(&entry); err != nil {
		response.InternalError(c)
		return
	}
	// SkillStore.Import now performs a remote upload/import flow instead of exposing backend-local paths.
	if err := dao.SkillStore.Import(c.Request.Context(), rootDir); err != nil {
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
	if _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}
	if !dao.SkillStore.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "skill storage not configured")
		return
	}
	if err := dao.SkillStore.Delete(c.Request.Context(), name); err != nil {
		if isStoreNotFound(err) {
			response.NotFound(c, "skill not found")
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	if err := skillsvc.SkillFrontmatter.DeleteByName(name); err != nil {
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

	if _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	entry, err := skillsvc.SkillFrontmatter.GetByName(name)
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

	if _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	if !dao.SkillStore.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "skill storage not configured")
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
	content, err := dao.SkillStore.ReadFile(c.Request.Context(), path.Join(name, rel))
	if err != nil {
		if isStoreNotFound(err) {
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

	if _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	if !dao.SkillStore.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "skill storage not configured")
		return
	}
	skills, total, err := dao.SkillStore.ListSkillCatalog(c.Request.Context(), keyword, page, pageSize)
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
	frontmatters, err := skillsvc.SkillFrontmatter.ListByNames(names)
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

func isStoreNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}

var errSkillFound = errors.New("skill found")

func getAuthUser(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", false
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	claims, err := accountsvc.ParseToken(parts[1])
	if err != nil {
		return "", false
	}
	refreshedToken, expiresAt, err := accountsvc.GenerateToken(claims.UserID)
	if err == nil {
		c.Header("X-Access-Token", refreshedToken)
		c.Header("X-Token-Expires", strconv.FormatInt(expiresAt, 10))
	}
	return claims.UserID, true
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
