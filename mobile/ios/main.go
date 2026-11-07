package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/uiautodev/weektodo/internal/ai"
	"github.com/uiautodev/weektodo/internal/server"
	"github.com/uiautodev/weektodo/internal/store"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	dataDir := application.Mobile.StoragePath()
	if dataDir == "" {
		// Should not happen on iOS; keep the app usable if the native bridge
		// cannot provide a path.
		dataDir = filepath.Join(os.TempDir(), "weektodo")
	}

	st, err := store.New(dataDir)
	if err != nil {
		log.Fatalf("load store: %v", err)
	}
	srv := server.New(st, ai.New(), server.Options{
		Version: version,
		Commit:  commit,
		Mobile:  true,
	})

	app := application.New(application.Options{
		Name:        "WeekTodo",
		Description: "Weekly task planner",
		Assets: application.AssetOptions{
			Handler: srv.Handler(),
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "WeekTodo",
		URL:              "/",
		BackgroundColour: application.NewRGB(250, 250, 250),
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
