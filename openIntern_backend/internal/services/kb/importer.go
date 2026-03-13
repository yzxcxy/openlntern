package kb

import (
	"archive/zip"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"openIntern/internal/dao"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

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
	for _, item := range reader.File {
		if item.FileInfo().IsDir() {
			continue
		}
		cleaned, err := cleanZipPath(item.Name)
		if err != nil {
			return err
		}
		if cleaned == "" || !shouldIncludePath(cleaned) {
			continue
		}
		files = append(files, item)
	}
	if len(files) == 0 {
		return nil
	}

	stripRoot := detectZipRoot(files)
	for _, item := range files {
		cleaned, err := cleanZipPath(item.Name)
		if err != nil {
			return err
		}
		if stripRoot != "" {
			cleaned = strings.TrimPrefix(cleaned, stripRoot+"/")
		}
		if cleaned == "" || cleaned == "." || !shouldIncludePath(cleaned) {
			continue
		}
		targetPath, err := dao.KnowledgeBase.ResolveLocalPath(rootDir, cleaned)
		if err != nil {
			return ErrInvalidZipPath
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if err := copyZipFile(item, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func copyZipFile(file *zip.File, targetPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	dst, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, rc); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
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
			return "", ErrInvalidZipPath
		}
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "" {
		return "", nil
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", ErrInvalidZipPath
	}
	return cleaned, nil
}

func detectZipRoot(files []*zip.File) string {
	root := ""
	for _, item := range files {
		cleaned, err := cleanZipPath(item.Name)
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
	if name == "" || utf8.ValidString(name) {
		return name
	}
	decoded, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), name)
	if err != nil || !utf8.ValidString(decoded) {
		return name
	}
	return decoded
}
