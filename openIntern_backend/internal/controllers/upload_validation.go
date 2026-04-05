package controllers

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// detectImageContentType reads upload bytes and returns a trusted image content type.
func detectImageContentType(fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil {
		return "", errors.New("file is required")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	readSize, readErr := file.Read(buffer)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", readErr
	}
	sniffed := strings.ToLower(strings.TrimSpace(http.DetectContentType(buffer[:readSize])))
	if strings.HasPrefix(sniffed, "image/") {
		return sniffed, nil
	}

	// svg is text-based and may be detected as xml/text; check payload marker before accepting.
	preview := bytes.ToLower(bytes.TrimSpace(buffer[:readSize]))
	if (strings.Contains(sniffed, "xml") || strings.HasPrefix(sniffed, "text/")) && bytes.Contains(preview, []byte("<svg")) {
		return "image/svg+xml", nil
	}
	return "", errors.New("only image files are supported")
}
