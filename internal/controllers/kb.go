package controllers

import (
	"archive/zip"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"openIntern/internal/response"
	"openIntern/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type kbItem struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type kbMovePayload struct {
	FromURI string `json:"from_uri"`
	ToURI   string `json:"to_uri"`
}

func ListKnowledgeBases(c *gin.Context) {
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	root := services.KnowledgeBaseRootURI()
	entries, err := services.OpenVikingList(c.Request.Context(), root, false)
	if err != nil {
		if isOpenVikingNotFound(err) {
			response.JSONSuccess(c, http.StatusOK, []kbItem{})
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	items := make([]kbItem, 0, len(entries))
	for _, entry := range entries {
		if !services.OpenVikingEntryIsDir(entry) {
			continue
		}
		entryPath := services.OpenVikingEntryString(entry, "path", "uri")
		entryName := services.OpenVikingEntryString(entry, "name")
		rel := services.OpenVikingRelativePath(root, entryPath)
		if rel == "" {
			rel = entryName
		}
		rel = strings.Trim(rel, "/")
		if rel == "" {
			continue
		}
		items = append(items, kbItem{
			Name: rel,
			URI:  strings.TrimRight(root, "/") + "/" + rel + "/",
		})
	}
	response.JSONSuccess(c, http.StatusOK, items)
}

func GetKnowledgeBaseTree(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	kbName, err := services.CleanKBName(name)
	if err != nil {
		response.BadRequest(c)
		return
	}
	kbURI := services.KnowledgeBaseURI(kbName)
	entries, err := services.OpenVikingTree(c.Request.Context(), kbURI)
	if err != nil {
		if isOpenVikingNotFound(err) {
			response.JSONSuccess(c, http.StatusOK, []map[string]any{})
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, entries)
}

func ImportKnowledgeBase(c *gin.Context) {
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	kbName, err := services.CleanKBName(c.PostForm("kb_name"))
	if err != nil {
		response.BadRequest(c)
		return
	}
	if services.File == nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "file service not configured")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil && !isMissingFormFile(err) {
		response.BadRequest(c)
		return
	}
	tempDir, err := os.MkdirTemp("", "kb-import-*")
	if err != nil {
		response.InternalError(c)
		return
	}
	defer os.RemoveAll(tempDir)
	rootDir := filepath.Join(tempDir, kbName)
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		response.InternalError(c)
		return
	}
	if fileHeader != nil {
		if err := extractZipToDir(fileHeader, rootDir); err != nil {
			if errors.Is(err, errInvalidZipPath) {
				response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
				return
			}
			response.InternalError(c)
			return
		}
	}
	files, err := listLocalFiles(rootDir)
	if err != nil {
		response.InternalError(c)
		return
	}
	if len(files) == 0 {
		if err := services.OpenVikingAddResourceWithOptions(c.Request.Context(), rootDir, services.KnowledgeBaseRootURI(), true, 0); err != nil {
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
			return
		}
		response.JSONSuccess(c, http.StatusOK, gin.H{"name": kbName})
		return
	}
	for _, rel := range files {
		absPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		cosKey := path.Join("kbs", kbName, rel)
		cosURL, err := services.File.UploadPath(c.Request.Context(), cosKey, absPath)
		if err != nil {
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
			return
		}
		baseURI := strings.TrimRight(services.KnowledgeBaseURI(kbName), "/")
		dir := path.Dir(strings.TrimLeft(rel, "/"))
		targetURI := baseURI + "/"
		if dir != "." && dir != "" {
			targetURI = baseURI + "/" + dir + "/"
		}
		log.Printf("ImportKnowledgeBase upload kb=%s rel=%s cos_key=%s target_uri=%s", kbName, rel, cosKey, targetURI)
		if err := services.OpenVikingAddResourceWithOptions(c.Request.Context(), cosURL, targetURI, true, 0); err != nil {
			response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
			return
		}
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"name": kbName})
}

func UploadKnowledgeBaseFile(c *gin.Context) {
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	kbName, err := services.CleanKBName(c.PostForm("kb_name"))
	if err != nil {
		response.BadRequest(c)
		return
	}
	if services.File == nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "file service not configured")
		return
	}
	targetDir := strings.TrimSpace(c.PostForm("target"))
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c)
		return
	}
	log.Printf("UploadKnowledgeBaseFile start kb=%s target_dir=%s file=%s", kbName, targetDir, fileHeader.Filename)
	rel := services.NormalizeUploadPath(targetDir, fileHeader.Filename)
	if rel == "" {
		response.JSONError(c, http.StatusBadRequest, response.CodeBadRequest, "invalid target path")
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c)
		return
	}
	cosKey := path.Join("kbs", kbName, rel)
	baseURI := strings.TrimRight(services.KnowledgeBaseURI(kbName), "/")
	targetURI := baseURI + "/"
	if targetDir != "" {
		dir := strings.TrimLeft(targetDir, "/")
		dir = path.Clean(dir)
		if dir != "." && dir != "" {
			targetURI = baseURI + "/" + dir + "/"
		}
	}
	log.Printf("UploadKnowledgeBaseFile resolved kb=%s rel=%s cos_key=%s target_uri=%s", kbName, rel, cosKey, targetURI)
	cosURL, err := services.File.UploadWithKey(c.Request.Context(), cosKey, src, fileHeader)
	src.Close()
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	if err := services.OpenVikingAddResourceWithOptions(c.Request.Context(), cosURL, targetURI, true, 0); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"path": targetURI})
}

func MoveKnowledgeBaseEntry(c *gin.Context) {
	var payload kbMovePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c)
		return
	}
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	if strings.TrimSpace(payload.FromURI) == "" || strings.TrimSpace(payload.ToURI) == "" {
		response.BadRequest(c)
		return
	}
	if err := services.OpenVikingMove(c.Request.Context(), payload.FromURI, payload.ToURI); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"from": payload.FromURI, "to": payload.ToURI})
}

func DragKnowledgeBaseEntry(c *gin.Context) {
	var payload kbMovePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c)
		return
	}
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	fromURI := strings.TrimSpace(payload.FromURI)
	toURI := strings.TrimSpace(payload.ToURI)
	if fromURI == "" || toURI == "" {
		response.BadRequest(c)
		return
	}
	if !strings.HasSuffix(toURI, "/") {
		toURI += "/"
	}
	if err := services.OpenVikingMove(c.Request.Context(), fromURI, toURI); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"from": fromURI, "to": toURI})
}

func DeleteKnowledgeBase(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	kbName, err := services.CleanKBName(name)
	if err != nil {
		response.BadRequest(c)
		return
	}
	if err := services.OpenVikingDeleteResource(c.Request.Context(), services.KnowledgeBaseURI(kbName), true); err != nil {
		if isOpenVikingNotFound(err) {
			response.NotFound(c, "kb not found")
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"name": kbName})
}

func DeleteKnowledgeBaseEntry(c *gin.Context) {
	rawURI := strings.TrimSpace(c.Query("uri"))
	recursive := strings.TrimSpace(c.DefaultQuery("recursive", "false"))
	if rawURI == "" {
		response.BadRequest(c)
		return
	}
	if !services.OpenViking.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "openviking not configured")
		return
	}
	if err := services.OpenVikingDeleteResource(c.Request.Context(), rawURI, strings.EqualFold(recursive, "true")); err != nil {
		if isOpenVikingNotFound(err) {
			response.NotFound(c, "entry not found")
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"uri": rawURI})
}

var errInvalidZipPath = errors.New("invalid zip entry path")

func isMissingFormFile(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, http.ErrMissingFile) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "missing file")
}

func extractZipToDir(fileHeader *multipart.FileHeader, rootDir string) error {
	if fileHeader == nil {
		return nil
	}
	tempZip, err := os.CreateTemp("", "kb-upload-*.zip")
	if err != nil {
		return err
	}
	tempZipPath := tempZip.Name()
	defer os.Remove(tempZipPath)
	src, err := fileHeader.Open()
	if err != nil {
		tempZip.Close()
		return err
	}
	if _, err := io.Copy(tempZip, src); err != nil {
		tempZip.Close()
		src.Close()
		return err
	}
	if err := tempZip.Close(); err != nil {
		src.Close()
		return err
	}
	if err := src.Close(); err != nil {
		return err
	}
	reader, err := zip.OpenReader(tempZipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	files := make([]*zip.File, 0, len(reader.File))
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		cleaned, err := cleanZipPath(f.Name)
		if err != nil {
			return err
		}
		if cleaned == "" {
			continue
		}
		if !shouldIncludePath(cleaned) {
			continue
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil
	}
	stripRoot := detectZipRoot(files)
	for _, f := range files {
		cleaned, err := cleanZipPath(f.Name)
		if err != nil {
			return err
		}
		if cleaned == "" {
			continue
		}
		if stripRoot != "" {
			cleaned = strings.TrimPrefix(cleaned, stripRoot+"/")
		}
		if cleaned == "" || cleaned == "." {
			continue
		}
		if !shouldIncludePath(cleaned) {
			continue
		}
		targetPath, err := services.ResolveUploadPath(rootDir, cleaned)
		if err != nil {
			return errInvalidZipPath
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(targetPath)
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(dst, rc); err != nil {
			dst.Close()
			rc.Close()
			return err
		}
		if err := dst.Close(); err != nil {
			rc.Close()
			return err
		}
		if err := rc.Close(); err != nil {
			return err
		}
	}
	return nil
}

func cleanZipPath(name string) (string, error) {
	trimmed := decodeZipName(strings.TrimSpace(name))
	if trimmed == "" {
		return "", nil
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "", nil
	}
	parts := strings.Split(trimmed, "/")
	for _, part := range parts {
		if part == ".." {
			return "", errInvalidZipPath
		}
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "" {
		return "", nil
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", errInvalidZipPath
	}
	return cleaned, nil
}

func detectZipRoot(files []*zip.File) string {
	root := ""
	for _, f := range files {
		cleaned, err := cleanZipPath(f.Name)
		if err != nil || cleaned == "" {
			return ""
		}
		if !strings.Contains(cleaned, "/") {
			return ""
		}
		parts := strings.Split(cleaned, "/")
		if len(parts) == 0 {
			return ""
		}
		if root == "" {
			root = parts[0]
			continue
		}
		if root != parts[0] {
			return ""
		}
	}
	return root
}

func listLocalFiles(rootDir string) ([]string, error) {
	result := []string{}
	err := filepath.WalkDir(rootDir, func(entryPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(rootDir, entryPath)
		if err != nil {
			return err
		}
		if rel == "." || rel == "" {
			return nil
		}
		normalized := filepath.ToSlash(rel)
		if !shouldIncludePath(normalized) {
			return nil
		}
		result = append(result, normalized)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func shouldIncludePath(rel string) bool {
	rel = strings.TrimPrefix(strings.TrimSpace(rel), "/")
	if rel == "" {
		return false
	}
	if strings.HasPrefix(rel, "__MACOSX/") || rel == "__MACOSX" {
		return false
	}
	base := path.Base(rel)
	if base == ".DS_Store" || strings.HasPrefix(base, "._") {
		return false
	}
	return true
}

func decodeZipName(name string) string {
	if name == "" {
		return name
	}
	if utf8.ValidString(name) {
		return name
	}
	decoded, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), name)
	if err != nil {
		return name
	}
	if !utf8.ValidString(decoded) {
		return name
	}
	return decoded
}
