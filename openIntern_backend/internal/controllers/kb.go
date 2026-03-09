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

	"openIntern/internal/dao"
	"openIntern/internal/response"

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
	if !dao.KnowledgeBase.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "knowledge base storage not configured")
		return
	}
	items, err := dao.KnowledgeBase.List(c.Request.Context())
	if err != nil {
		if isStoreNotFound(err) {
			response.JSONSuccess(c, http.StatusOK, []kbItem{})
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	result := make([]kbItem, 0, len(items))
	for _, item := range items {
		result = append(result, kbItem{
			Name: item.Name,
			URI:  item.URI,
		})
	}
	response.JSONSuccess(c, http.StatusOK, result)
}

func GetKnowledgeBaseTree(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if !dao.KnowledgeBase.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "knowledge base storage not configured")
		return
	}
	entries, err := dao.KnowledgeBase.Tree(c.Request.Context(), name)
	if err != nil {
		if isStoreNotFound(err) {
			response.JSONSuccess(c, http.StatusOK, []map[string]any{})
			return
		}
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, entries)
}

func ImportKnowledgeBase(c *gin.Context) {
	if !dao.KnowledgeBase.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "knowledge base storage not configured")
		return
	}
	kbName, err := dao.KnowledgeBase.CleanName(c.PostForm("kb_name"))
	if err != nil {
		response.BadRequest(c)
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

	// Workaround for OpenViking Chinese path bug: create nested directory <kb>/<kb>/.
	if err := dao.KnowledgeBase.Ingest(c.Request.Context(), rootDir, dao.KnowledgeBase.URI(kbName), false, 0); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusAccepted, gin.H{
		"name":   kbName,
		"status": "accepted",
		"async":  true,
	})
}

func UploadKnowledgeBaseFile(c *gin.Context) {
	if !dao.KnowledgeBase.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "knowledge base storage not configured")
		return
	}
	kbName, err := dao.KnowledgeBase.CleanName(c.PostForm("kb_name"))
	if err != nil {
		response.BadRequest(c)
		return
	}
	targetDir := strings.TrimSpace(c.PostForm("target"))
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c)
		return
	}
	log.Printf("UploadKnowledgeBaseFile start kb=%s target_dir=%s file=%s", kbName, targetDir, fileHeader.Filename)
	fileName := sanitizeUploadedFileName(fileHeader.Filename)
	tempDir, err := os.MkdirTemp("", "kb-upload-file-*")
	if err != nil {
		response.InternalError(c)
		return
	}
	defer os.RemoveAll(tempDir)
	localPath := filepath.Join(tempDir, fileName)
	if err := c.SaveUploadedFile(fileHeader, localPath); err != nil {
		response.InternalError(c)
		return
	}
	targetURI := resolveKnowledgeBaseUploadTargetURI(kbName, targetDir)
	log.Printf("UploadKnowledgeBaseFile resolved kb=%s file=%s local_path=%s target_uri=%s", kbName, fileName, localPath, targetURI)
	if err := dao.KnowledgeBase.Ingest(c.Request.Context(), localPath, targetURI, false, 0); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusAccepted, gin.H{
		"path":   targetURI,
		"status": "accepted",
		"async":  true,
	})
}

func resolveKnowledgeBaseUploadTargetURI(kbName string, targetDir string) string {
	base := strings.TrimRight(dao.KnowledgeBase.InnerURI(kbName), "/")
	dir := strings.TrimSpace(targetDir)
	dir = strings.TrimLeft(dir, "/")
	dir = path.Clean(dir)
	if dir == ".." || strings.HasPrefix(dir, "../") {
		return base + "/"
	}
	if dir == "." || dir == "" {
		return base + "/"
	}
	firstPrefix := kbName + "/"
	if dir == kbName {
		dir = ""
	} else if strings.HasPrefix(dir, firstPrefix) {
		dir = strings.TrimPrefix(dir, firstPrefix)
	}
	if dir == kbName {
		dir = ""
	} else if strings.HasPrefix(dir, firstPrefix) {
		dir = strings.TrimPrefix(dir, firstPrefix)
	}
	if dir == "." || dir == "" {
		return base + "/"
	}
	return base + "/" + dir + "/"
}

func sanitizeUploadedFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	if name == "" || name == "." || name == ".." {
		return "upload.bin"
	}
	return name
}

func MoveKnowledgeBaseEntry(c *gin.Context) {
	var payload kbMovePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c)
		return
	}
	if !dao.KnowledgeBase.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "knowledge base storage not configured")
		return
	}
	if strings.TrimSpace(payload.FromURI) == "" || strings.TrimSpace(payload.ToURI) == "" {
		response.BadRequest(c)
		return
	}
	if err := dao.KnowledgeBase.MoveEntry(c.Request.Context(), payload.FromURI, payload.ToURI); err != nil {
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
	if !dao.KnowledgeBase.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "knowledge base storage not configured")
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
	if err := dao.KnowledgeBase.MoveEntry(c.Request.Context(), fromURI, toURI); err != nil {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, err.Error())
		return
	}
	response.JSONSuccess(c, http.StatusOK, gin.H{"from": fromURI, "to": toURI})
}

func DeleteKnowledgeBase(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if !dao.KnowledgeBase.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "knowledge base storage not configured")
		return
	}
	kbName, err := dao.KnowledgeBase.CleanName(name)
	if err != nil {
		response.BadRequest(c)
		return
	}
	if err := dao.KnowledgeBase.Delete(c.Request.Context(), name); err != nil {
		if isStoreNotFound(err) {
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
	if !dao.KnowledgeBase.Configured() {
		response.JSONError(c, http.StatusInternalServerError, response.CodeInternal, "knowledge base storage not configured")
		return
	}
	if err := dao.KnowledgeBase.DeleteEntry(c.Request.Context(), rawURI, strings.EqualFold(recursive, "true")); err != nil {
		if isStoreNotFound(err) {
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
		targetPath, err := dao.KnowledgeBase.ResolveLocalPath(rootDir, cleaned)
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
