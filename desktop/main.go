package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uiautodev/weektodo/internal/ai"
	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/server"
	"github.com/uiautodev/weektodo/internal/store"
	"github.com/uiautodev/weektodo/internal/update"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed build/trayicon.png
var trayIcon []byte

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	st, err := store.New("")
	if err != nil {
		log.Fatalf("load store: %v", err)
	}
	updater := update.New(update.Options{DataDir: st.Dir(), Desktop: true})
	srv := server.New(st, ai.New(), server.Options{
		Version: version,
		Commit:  commit,
		Desktop: true,
		Updater: updater,
	})

	app := application.New(application.Options{
		Name:        "WeekTodo",
		Description: "Weekly task planner",
		Assets: application.AssetOptions{
			Handler: srv.Handler(),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyRegular,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "WeekTodo",
		Width:            1120,
		Height:           800,
		MinWidth:         760,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(250, 250, 250),
		URL:              "/",
	})

	initialLanguage := i18n.Normalize(st.Settings().Language)
	if initialLanguage == "" {
		initialLanguage = i18n.System()
	}
	var languageMu sync.Mutex
	language := initialLanguage
	currentLanguage := func() i18n.Language {
		languageMu.Lock()
		defer languageMu.Unlock()
		return language
	}
	setLanguage := func(next i18n.Language) {
		languageMu.Lock()
		language = next
		languageMu.Unlock()
	}

	var quitting atomic.Bool
	var updateInProgress atomic.Bool
	var tray *application.SystemTray
	var statusItem *application.MenuItem

	showWindow := func() {
		setAppRegular()
		window.Show()
		window.Restore()
		window.Focus()
	}
	hideWindow := func() {
		app.Event.Emit("weektodo:window-hiding")
		setAppAccessory()
		window.Hide()
	}
	quitApp := func() {
		app.Event.Emit("weektodo:window-hiding")
		time.Sleep(150 * time.Millisecond)
		quitting.Store(true)
		app.Quit()
	}

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if quitting.Load() {
			return
		}
		hideWindow()
		event.Cancel()
	})

	checkUpdates := func(lang i18n.Language) {
		if updateInProgress.Load() {
			return
		}
		text := trayText(lang)
		info, err := updater.Check(context.Background(), version)
		if err != nil {
			app.Dialog.Error().SetTitle(text.checkFailed).SetMessage(err.Error()).Show()
			return
		}
		if !info.Available {
			message := text.upToDate
			if info.Dev {
				message = text.devBuild
			}
			app.Dialog.Info().SetTitle(text.checkUpdates).SetMessage(message).Show()
			return
		}
		source := ""
		if info.Source == "gh-proxy" {
			source = text.viaProxy
		}
		notes := strings.TrimSpace(info.Notes)
		if len([]rune(notes)) > 1800 {
			notes = string([]rune(notes)[:1800]) + "…"
		}
		message := fmt.Sprintf(text.updateMessage, info.Latest, source)
		if notes != "" {
			message += "\n\n" + notes
		}
		dialog := app.Dialog.Question().
			SetTitle(text.updateAvailable).
			SetMessage(message)
		dialog.AddButton(text.later).SetAsDefault()
		dialog.AddButton(text.install).OnClick(func() {
			if !updateInProgress.CompareAndSwap(false, true) {
				return
			}
			go func() {
				defer updateInProgress.Store(false)
				app.Dialog.Info().
					SetTitle(text.installing).
					SetMessage(fmt.Sprintf(text.installingMessage, info.Latest)).
					Show()
				if _, err := updater.Apply(context.Background(), info); err != nil {
					app.Dialog.Error().SetTitle(text.installFailed).SetMessage(err.Error()).Show()
					return
				}
				if err := updater.Restart(); err != nil {
					app.Dialog.Error().SetTitle(text.restartFailed).SetMessage(err.Error()).Show()
				}
			}()
		})
		dialog.Show()
	}

	activeCategory := func() string {
		id := st.Settings().ActiveCategory
		if id == "" {
			return model.DefaultCategoryID
		}
		return id
	}

	buildMenu := func() *application.Menu {
		lang := currentLanguage()
		text := trayText(lang)
		menu := app.NewMenu()
		menu.Add(text.open).OnClick(func(*application.Context) { showWindow() })
		menu.Add(text.quickAdd).OnClick(func(*application.Context) {
			showWindow()
			time.AfterFunc(180*time.Millisecond, func() {
				app.Event.Emit("weektodo:focus-quick-add")
			})
		})
		menu.Add(text.checkUpdates).OnClick(func(*application.Context) {
			go checkUpdates(currentLanguage())
		})
		statusItem = menu.Add(fmt.Sprintf(text.today, st.TodayCountInCategory(activeCategory(), time.Now())))
		statusItem.SetEnabled(false)
		menu.AddSeparator()
		menu.Add(text.hide).OnClick(func(*application.Context) { hideWindow() })
		menu.AddSeparator()
		menu.Add(text.quit).OnClick(func(*application.Context) { quitApp() })
		return menu
	}

	syncLanguage := func() {
		next := i18n.Normalize(st.Settings().Language)
		if next != "" && next != currentLanguage() {
			setLanguage(next)
			tray.SetMenu(buildMenu())
		}
	}

	refreshTray := func() {
		syncLanguage()
		lang := currentLanguage()
		text := trayText(lang)
		count := st.TodayCountInCategory(activeCategory(), time.Now())
		application.InvokeSync(func() {
			if count > 0 {
				tray.SetLabel(strconv.Itoa(count))
			} else {
				tray.SetLabel("")
			}
			tray.SetTooltip(fmt.Sprintf("WeekTodo · %s", fmt.Sprintf(text.today, count)))
			statusItem.SetLabel(fmt.Sprintf(text.today, count))
		})
	}

	tray = app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayIcon)
	} else {
		tray.SetIcon(trayIcon)
	}
	tray.OnClick(showWindow)
	tray.OnRightClick(func() { tray.ShowMenu() })
	tray.SetMenu(buildMenu())

	app.Event.On("weektodo:changed", func(*application.CustomEvent) {
		refreshTray()
	})

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		setAppRegular()
		showWindow()
		refreshTray()
		go func() {
			// Poll the in-process store so the tray follows language and
			// today-count changes even if a webview event is missed.
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				refreshTray()
			}
		}()
	})

	if runtime.GOOS == "darwin" {
		app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
			showWindow()
		})
	}

	if err := app.Run(); err != nil {
		log.Fatalf("run desktop app: %v", err)
	}
}
