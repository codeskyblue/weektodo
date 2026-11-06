package update

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxWindowsExecutableSize = 256 << 20

// extractWindowsExecutable extracts the Windows desktop executable from a
// release zip into dir. Only the WeekTodo.exe entry is used, so archive paths
// can never escape the destination.
func extractWindowsExecutable(archivePath, dir string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open update zip: %w", err)
	}
	defer reader.Close()

	var entry *zip.File
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Base(filepath.ToSlash(file.Name)), "WeekTodo.exe") {
			entry = file
			break
		}
	}
	if entry == nil {
		return "", errors.New("update zip does not contain WeekTodo.exe")
	}
	if entry.UncompressedSize64 > maxWindowsExecutableSize {
		return "", errors.New("update executable is too large")
	}

	source, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()

	tmp, err := os.CreateTemp(dir, ".weektodo-exe-*")
	if err != nil {
		return "", fmt.Errorf("write update next to the executable: %w", err)
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()

	written, err := io.Copy(tmp, io.LimitReader(source, maxWindowsExecutableSize+1))
	if err != nil {
		tmp.Close()
		return "", err
	}
	if written > maxWindowsExecutableSize {
		tmp.Close()
		return "", errors.New("update executable is too large")
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	ok = true
	return tmpName, nil
}
