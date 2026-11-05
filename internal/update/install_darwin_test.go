//go:build darwin

package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyDesktopDMG(t *testing.T) {
	version := "0.2.0"
	stage := filepath.Join(t.TempDir(), "stage")
	app := filepath.Join(stage, "WeekTodo.app")
	appExe := filepath.Join(app, "Contents", "MacOS", "weektodo")
	if err := os.MkdirAll(filepath.Dir(appExe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appExe, []byte("new binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleShortVersionString</key><string>` + version + `</string></dict></plist>`)
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), plist, 0o644); err != nil {
		t.Fatal(err)
	}
	asset, err := AssetName("darwin", "arm64", version, true)
	if err != nil {
		t.Fatal(err)
	}
	dmg := filepath.Join(t.TempDir(), asset)
	if out, err := exec.Command("hdiutil", "create", "-volname", "WeekTodo", "-srcfolder", stage, "-ov", "-format", "UDZO", dmg).CombinedOutput(); err != nil {
		t.Fatalf("create dmg: %v: %s", err, out)
	}
	payload, err := os.ReadFile(dmg)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/missing-manifest", http.NotFound)
	mux.HandleFunc("/missing-latest", http.NotFound)
	mux.HandleFunc("/missing-feed", http.NotFound)
	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Release{
			Tag: "v" + version,
			Assets: []Asset{{
				Name:               asset,
				Size:               int64(len(payload)),
				SHA256:             hash,
				BrowserDownloadURL: "http://" + r.Host + "/" + asset,
			}},
		})
	})
	mux.HandleFunc("/"+asset, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	targetApp := filepath.Join(t.TempDir(), "WeekTodo.app")
	targetExe := filepath.Join(targetApp, "Contents", "MacOS", "weektodo")
	if err := os.MkdirAll(filepath.Dir(targetExe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetExe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetApp, "Contents", "Info.plist"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := New(Options{
		ManifestURL: srv.URL + "/missing-manifest",
		LatestURL:   srv.URL + "/missing-latest",
		FeedURL:     srv.URL + "/missing-feed",
		APIURL:      srv.URL + "/release",
		DataDir:     t.TempDir(),
		Executable:  targetExe,
		Desktop:     true,
		GOOS:        "darwin",
		GOARCH:      "arm64",
		HTTPClient:  srv.Client(),
	})
	info, err := client.Check(context.Background(), "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if info.Asset != asset {
		t.Fatalf("asset = %q, want %q", info.Asset, asset)
	}
	if _, err := client.Apply(context.Background(), info); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(targetExe)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new binary" {
		t.Fatalf("target executable = %q", data)
	}
	updatedPlist, err := os.ReadFile(filepath.Join(targetApp, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updatedPlist), version) {
		t.Fatalf("Info.plist was not updated: %s", updatedPlist)
	}
}
