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

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	Icon        string `yaml:"icon"`
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

	absPath, relPath, err := resolveSkillPath(cleaned)
	if err != nil {
		response.Forbidden(c)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			response.JSONSuccess(c, http.StatusOK, []skillFileItem{})
			return
		}
		response.InternalError(c)
		return
	}
	if !info.IsDir() {
		response.JSONSuccess(c, http.StatusOK, []skillFileItem{})
		return
	}

	items := make([]skillFileItem, 0)
	err = filepath.WalkDir(absPath, func(currentPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		relToRoot, err := filepath.Rel(absPath, currentPath)
		if err != nil {
			return err
		}
		relToRoot = filepath.ToSlash(relToRoot)
		entryPath := "/" + path.Join(relPath, relToRoot)
		items = append(items, skillFileItem{
			ID:   entryPath,
			Type: "file",
			Size: stat.Size(),
			Date: stat.ModTime(),
		})
		return nil
	})
	if err != nil {
		response.InternalError(c)
		return
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

	absPath, _, err := resolveSkillPath(cleaned)
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

	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	absPath, _, err := resolveSkillPath(cleaned)
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

	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	absPath, _, err := resolveSkillPath(cleaned)
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

	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	absPath, _, err := resolveSkillPath(cleaned)
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

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}
	skillDir := filepath.Join(baseDir, name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	skill, err := readSkillFromFile(skillFile, name)
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

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}
	skillDir := filepath.Join(baseDir, name)
	skillFile, err := resolveSkillContentPath(skillDir, c.Query("path"))
	if err != nil {
		if errors.Is(err, errInvalidPath) {
			response.BadRequest(c)
			return
		}
		if errors.Is(err, errOutOfScope) {
			response.Forbidden(c)
			return
		}
		response.InternalError(c)
		return
	}
	info, err := os.Stat(skillFile)
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

func ListSkills(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	if _, _, ok := getAuthUser(c); !ok {
		response.Unauthorized(c)
		return
	}

	baseDir, err := skillBaseDir()
	if err != nil {
		response.InternalError(c)
		return
	}

	skills, total, err := listSkillMetas(baseDir, keyword, page, pageSize)
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

var errInvalidPath = errors.New("invalid path")
var errOutOfScope = errors.New("out of scope")

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

func readSkillFromFile(skillFile string, skillPath string) (models.Skill, error) {
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
	return models.Skill{
		SkillID:     url.PathEscape(skillPath),
		Name:        frontmatter.Name,
		Description: frontmatter.Description,
		Source:      author,
		Icon:        frontmatter.Icon,
		Path:        skillPath,
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
