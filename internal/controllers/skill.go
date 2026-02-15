package controllers

import (
	"errors"
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
	"gopkg.in/yaml.v3"
)

type skillFileItem struct {
	ID   string    `json:"id"`
	Type string    `json:"type"`
	Size int64     `json:"size"`
	Date time.Time `json:"date"`
}

const officialSkillDirName = "offical"

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	Icon        string `yaml:"icon"`
}

func ListSkillFiles(c *gin.Context) {
	scope := strings.TrimSpace(c.DefaultQuery("scope", "official"))
	rawPath := c.DefaultQuery("path", "/")
	userID, _, ok := getAuthUser(c)

	if scope != "official" && scope != "user" {
		response.BadRequest(c)
		return
	}

	if scope == "user" && !ok {
		response.Unauthorized(c)
		return
	}

	baseSegment := "official"
	if scope == "user" {
		baseSegment = userID
	}

	cleaned, err := cleanSkillPath(rawPath)
	if err != nil {
		response.BadRequest(c)
		return
	}

	absPath, relPath, err := resolveSkillPath(cleaned, baseSegment)
	if err != nil {
		response.Forbidden(c)
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			response.JSONSuccess(c, http.StatusOK, []skillFileItem{})
			return
		}
		response.InternalError(c)
		return
	}

	items := make([]skillFileItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			response.InternalError(c)
			return
		}
		entryPath := "/" + path.Join(relPath, entry.Name())
		items = append(items, skillFileItem{
			ID:   entryPath,
			Type: "file",
			Size: info.Size(),
			Date: info.ModTime(),
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

	userID, _, ok := getAuthUser(c)
	baseSegment, err := resolveReadBase(cleaned, userID, ok)
	if err != nil {
		if errors.Is(err, errUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.Forbidden(c)
		return
	}

	absPath, _, err := resolveSkillPath(cleaned, baseSegment)
	if err != nil {
		response.Forbidden(c)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "file not found")
			return
		}
		response.InternalError(c)
		return
	}
	if info.IsDir() {
		response.BadRequest(c)
		return
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"path":    cleaned,
		"content": string(content),
	})
}

func CreateSkillEntry(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c)
		return
	}

	cleaned, err := cleanSkillPath(req.Path)
	if err != nil || cleaned == "/" {
		response.BadRequest(c)
		return
	}

	userID, role, ok := getAuthUser(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	baseSegment, err := resolveWriteBase(cleaned, userID, role)
	if err != nil {
		if errors.Is(err, errUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.Forbidden(c)
		return
	}

	absPath, _, err := resolveSkillPath(cleaned, baseSegment)
	if err != nil {
		response.Forbidden(c)
		return
	}

	if strings.TrimSpace(req.Type) == "" {
		response.BadRequest(c)
		return
	}

	if req.Type == "folder" {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			response.InternalError(c)
			return
		}
		response.JSONMessage(c, http.StatusCreated, "folder created successfully")
		return
	}

	if req.Type != "file" {
		response.BadRequest(c)
		return
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		response.InternalError(c)
		return
	}

	f, err := os.OpenFile(absPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			response.BadRequest(c)
			return
		}
		response.InternalError(c)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(req.Content); err != nil {
		response.InternalError(c)
		return
	}

	response.JSONMessage(c, http.StatusCreated, "file created successfully")
}

func UpdateSkillFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c)
		return
	}

	cleaned, err := cleanSkillPath(req.Path)
	if err != nil || cleaned == "/" {
		response.BadRequest(c)
		return
	}

	userID, role, ok := getAuthUser(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	baseSegment, err := resolveWriteBase(cleaned, userID, role)
	if err != nil {
		if errors.Is(err, errUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.Forbidden(c)
		return
	}

	absPath, _, err := resolveSkillPath(cleaned, baseSegment)
	if err != nil {
		response.Forbidden(c)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "file not found")
			return
		}
		response.InternalError(c)
		return
	}
	if info.IsDir() {
		response.BadRequest(c)
		return
	}

	if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
		response.InternalError(c)
		return
	}

	response.JSONMessage(c, http.StatusOK, "file updated successfully")
}

func DeleteSkillEntry(c *gin.Context) {
	rawPath := c.Query("path")
	if strings.TrimSpace(rawPath) == "" {
		response.BadRequest(c)
		return
	}

	cleaned, err := cleanSkillPath(rawPath)
	if err != nil || cleaned == "/" {
		response.BadRequest(c)
		return
	}

	userID, role, ok := getAuthUser(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	baseSegment, err := resolveWriteBase(cleaned, userID, role)
	if err != nil {
		if errors.Is(err, errUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.Forbidden(c)
		return
	}

	absPath, _, err := resolveSkillPath(cleaned, baseSegment)
	if err != nil {
		response.Forbidden(c)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "path not found")
			return
		}
		response.InternalError(c)
		return
	}

	if info.IsDir() {
		if err := os.RemoveAll(absPath); err != nil {
			response.InternalError(c)
			return
		}
		response.JSONMessage(c, http.StatusOK, "folder deleted successfully")
		return
	}

	if err := os.Remove(absPath); err != nil {
		response.InternalError(c)
		return
	}
	response.JSONMessage(c, http.StatusOK, "file deleted successfully")
}

func CreateSkillMeta(c *gin.Context) {
	response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "skill meta is read-only")
}

func GetOfficialSkillMetaByName(c *gin.Context) {
	name, err := parseSkillName(c.Param("name"))
	if err != nil {
		response.BadRequest(c)
		return
	}

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}
	skillDir := filepath.Join(baseDir, officialSkillDirName, name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	skill, err := readSkillFromFile(skillFile, path.Join("official", name), "", "official")
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "skill not found")
			return
		}
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusOK, skill)
}

func GetCustomSkillMetaByName(c *gin.Context) {
	name, err := parseSkillName(c.Param("name"))
	if err != nil {
		response.BadRequest(c)
		return
	}

	userID, _, ok := getAuthUser(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}
	skillDir := filepath.Join(baseDir, userID, name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	skill, err := readSkillFromFile(skillFile, path.Join(userID, name), userID, userID)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "skill not found")
			return
		}
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusOK, skill)
}

func GetSkillMeta(c *gin.Context) {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		response.BadRequest(c)
		return
	}

	decoded, err := url.PathUnescape(id)
	if err != nil {
		response.BadRequest(c)
		return
	}

	cleaned, err := cleanSkillPath(decoded)
	if err != nil {
		response.BadRequest(c)
		return
	}

	userID, _, ok := getAuthUser(c)
	baseSegment, err := resolveReadBase(cleaned, userID, ok)
	if err != nil {
		if errors.Is(err, errUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.Forbidden(c)
		return
	}

	absPath, relPath, err := resolveSkillMetaPath(cleaned, baseSegment)
	if err != nil {
		response.Forbidden(c)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "skill not found")
			return
		}
		response.InternalError(c)
		return
	}

	skillFile := absPath
	if info.IsDir() {
		skillFile = filepath.Join(absPath, "SKILL.md")
	}

	skill, err := readSkillFromFile(skillFile, relPath, userID, baseSegment)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "skill not found")
			return
		}
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusOK, skill)
}

func UpdateSkillMeta(c *gin.Context) {
	response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "skill meta is read-only")
}

func DeleteSkillMeta(c *gin.Context) {
	response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "skill meta is read-only")
}

func ReadOfficialSkillContent(c *gin.Context) {
	name, err := parseSkillName(c.Param("name"))
	if err != nil {
		response.BadRequest(c)
		return
	}

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}
	skillFile := filepath.Join(baseDir, officialSkillDirName, name, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "file not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"name":    name,
		"content": string(content),
	})
}

func ReadCustomSkillContent(c *gin.Context) {
	name, err := parseSkillName(c.Param("name"))
	if err != nil {
		response.BadRequest(c)
		return
	}

	userID, _, ok := getAuthUser(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}
	skillFile := filepath.Join(baseDir, userID, name, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "file not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{
		"name":    name,
		"content": string(content),
	})
}

func ListOfficialSkills(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}

	skills, total, err := listSkillMetas(filepath.Join(baseDir, officialSkillDirName), "official", "", keyword, page, pageSize)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  skills,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func ListCustomSkills(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	userID := c.Query("user_id")
	keyword := c.Query("keyword")

	authUserID, _, ok := getAuthUser(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if userID == "" {
		userID = authUserID
	}
	if userID != authUserID {
		response.Forbidden(c)
		return
	}

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}

	skills, total, err := listSkillMetas(filepath.Join(baseDir, userID), userID, userID, keyword, page, pageSize)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.JSONSuccess(c, http.StatusOK, gin.H{
		"data":  skills,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

var errUnauthorized = errors.New("unauthorized")

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
	return claims.UserID, claims.Role, true
}

func skillBaseDir() (string, error) {
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

func resolveSkillPath(cleaned string, baseSegment string) (string, string, error) {
	if baseSegment == "" {
		return "", "", errors.New("invalid base")
	}
	relPath := ""
	if cleaned == "/" {
		relPath = baseSegment
	} else {
		relPath = strings.TrimPrefix(cleaned, "/")
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) == 0 || parts[0] != baseSegment {
			return "", "", errors.New("out of scope")
		}
	}
	baseDir, err := skillBaseDir()
	if err != nil {
		return "", "", err
	}
	absPath := filepath.Join(baseDir, filepath.FromSlash(relPath))
	return absPath, relPath, nil
}

func resolveReadBase(cleaned string, userID string, hasAuth bool) (string, error) {
	if hasSegmentPrefix(cleaned, "official") {
		return "official", nil
	}
	if !hasAuth {
		return "", errUnauthorized
	}
	if hasSegmentPrefix(cleaned, userID) {
		return userID, nil
	}
	return "", errors.New("out of scope")
}

func resolveWriteBase(cleaned string, userID string, role string) (string, error) {
	if userID == "" {
		return "", errUnauthorized
	}
	if hasSegmentPrefix(cleaned, "official") {
		if role != models.RoleAdmin {
			return "", errors.New("forbidden")
		}
		return "official", nil
	}
	if hasSegmentPrefix(cleaned, userID) {
		return userID, nil
	}
	return "", errors.New("forbidden")
}

func hasSegmentPrefix(cleaned string, segment string) bool {
	if segment == "" {
		return false
	}
	prefix := "/" + segment
	if !strings.HasPrefix(cleaned, prefix) {
		return false
	}
	if len(cleaned) == len(prefix) {
		return true
	}
	return cleaned[len(prefix)] == '/'
}

func resolveSkillMetaPath(cleaned string, baseSegment string) (string, string, error) {
	if baseSegment == "" {
		return "", "", errors.New("invalid base")
	}
	relPath := ""
	if cleaned == "/" {
		relPath = baseSegment
	} else {
		relPath = strings.TrimPrefix(cleaned, "/")
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) == 0 || parts[0] != baseSegment {
			return "", "", errors.New("out of scope")
		}
	}
	fsRelPath := relPath
	if baseSegment == "official" {
		parts := strings.SplitN(relPath, "/", 2)
		parts[0] = officialSkillDirName
		fsRelPath = parts[0]
		if len(parts) > 1 {
			fsRelPath = fsRelPath + "/" + parts[1]
		}
	}
	baseDir, err := skillBaseDir()
	if err != nil {
		return "", "", err
	}
	absPath := filepath.Join(baseDir, filepath.FromSlash(fsRelPath))
	return absPath, relPath, nil
}

func listSkillMetas(baseDir string, baseSegment string, userID string, keyword string, page int, pageSize int) ([]models.Skill, int64, error) {
	skills, err := collectSkills(baseDir, baseSegment, userID)
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
		return skills[i].Path < skills[j].Path
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

func collectSkills(baseDir string, baseSegment string, userID string) ([]models.Skill, error) {
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
		skillPath := path.Join(baseSegment, dirName)
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
		skill, err := readSkillFromFile(skillFile, skillPath, userID, baseSegment)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

func readSkillFromFile(skillFile string, skillPath string, userID string, baseSegment string) (models.Skill, error) {
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return models.Skill{}, err
	}
	frontmatter, err := parseSkillFrontmatter(string(data))
	if err != nil {
		return models.Skill{}, err
	}

	dirName := filepath.Base(filepath.Dir(skillFile))
	author := parseAuthorFromDir(dirName)
	skillType := models.SkillTypeCustom
	if baseSegment == "official" {
		skillType = models.SkillTypeOfficial
	}
	return models.Skill{
		SkillID:     url.PathEscape(skillPath),
		Name:        frontmatter.Name,
		Description: frontmatter.Description,
		Type:        skillType,
		Source:      author,
		Icon:        frontmatter.Icon,
		Path:        skillPath,
		UserID:      userID,
	}, nil
}

func parseSkillFrontmatter(content string) (skillFrontmatter, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return skillFrontmatter{}, errors.New("invalid frontmatter")
	}
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}
	if endIndex == -1 {
		return skillFrontmatter{}, errors.New("invalid frontmatter")
	}
	block := strings.Join(lines[1:endIndex], "\n")
	var frontmatter skillFrontmatter
	if err := yaml.Unmarshal([]byte(block), &frontmatter); err != nil {
		return skillFrontmatter{}, err
	}
	return frontmatter, nil
}

func parseAuthorFromDir(dirName string) string {
	parts := strings.SplitN(dirName, "-", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func matchSkillKeyword(skill models.Skill, keyword string) bool {
	keyword = strings.ToLower(keyword)
	fields := []string{
		skill.Name,
		skill.Description,
		skill.Source,
		skill.Path,
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), keyword) {
			return true
		}
	}
	return false
}
