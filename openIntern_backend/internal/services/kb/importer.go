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

	"openIntern/internal/models"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// saveMultipartFile saves a multipart uploaded file to a local path.
func saveMultipartFile(fileHeader *multipart.FileHeader, destPath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// validateZipArchive validates zip contents without extracting.
// It checks for path traversal attacks and ensures the archive contains valid files.
func validateZipArchive(zipPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	hasValidFiles := false
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Check for path traversal attacks.
		cleaned, err := cleanZipPath(file.Name)
		if err != nil {
			return err
		}
		if cleaned == "" {
			continue
		}

		// Skip macOS garbage files.
		if !shouldIncludePath(cleaned) {
			continue
		}

		hasValidFiles = true
	}

	if !hasValidFiles {
		return ErrInvalidZipPath
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

// ZipEntry 表示zip中的条目信息。
type ZipEntry struct {
	Path  string
	Name  string
	IsDir bool
	Size  int64
}

// ExtractZipTree 解析zip文件并返回原始目录结构。
// 同时将文件解压到目标目录。
func ExtractZipTree(zipPath string, destDir string) ([]models.KBTreeEntry, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// 用于去重目录
	seenDirs := make(map[string]struct{})
	var entries []models.KBTreeEntry

	for _, file := range reader.File {
		if !shouldIncludePath(file.Name) {
			continue
		}

		cleaned, err := cleanZipPath(file.Name)
		if err != nil {
			return nil, err
		}
		if cleaned == "" {
			continue
		}

		isDir := file.FileInfo().IsDir()
		name := path.Base(cleaned)
		if name == "" || name == "." {
			continue
		}

		// 添加目录（确保父目录存在）
		if !isDir {
			// 添加所有父目录
			parts := strings.Split(cleaned, "/")
			for i := 0; i < len(parts)-1; i++ {
				dirPath := strings.Join(parts[:i+1], "/")
				if _, seen := seenDirs[dirPath]; !seen {
					seenDirs[dirPath] = struct{}{}
					entries = append(entries, models.KBTreeEntry{
						Path:  dirPath,
						Name:  parts[i],
						IsDir: true,
						Size:  0,
					})
				}
			}
		}

		// 添加条目
		entries = append(entries, models.KBTreeEntry{
			Path:  cleaned,
			Name:  name,
			IsDir: isDir,
			Size:  file.FileInfo().Size(),
		})

		// 解压文件到目标目录
		if !isDir && destDir != "" {
			if err := extractFile(file, destDir, cleaned); err != nil {
				return nil, err
			}
		}
	}

	return entries, nil
}

// extractFile 解压单个文件到目标目录。
func extractFile(file *zip.File, destDir string, relPath string) error {
	// 构建目标路径
	targetPath := filepath.Join(destDir, filepath.FromSlash(relPath))

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// 打开zip中的文件
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// 复制内容
	_, err = io.Copy(dst, src)
	return err
}