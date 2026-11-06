package update

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(out)
	for name, data := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractWindowsExecutable(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "update.zip")
	writeZip(t, archive, map[string][]byte{
		"WeekTodo.exe": []byte("new executable"),
		"README.txt":   []byte("readme"),
	})

	extracted, err := extractWindowsExecutable(archive, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new executable" {
		t.Fatalf("data = %q", data)
	}
	if filepath.Dir(extracted) != dir {
		t.Fatalf("extracted outside target dir: %s", extracted)
	}
}

func TestExtractWindowsExecutableRejectsMissingExe(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "update.zip")
	writeZip(t, archive, map[string][]byte{"README.txt": []byte("readme")})
	if _, err := extractWindowsExecutable(archive, dir); err == nil {
		t.Fatal("expected missing executable error")
	}
}
