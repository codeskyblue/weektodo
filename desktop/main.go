package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/uiautodev/weektodo/internal/ai"
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
		Description: "按周记录、整理和总结你的待办事项",
		Assets: application.AssetOptions{
			Handler: srv.Handler(),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
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

	var quitting atomic.Bool
	var updateInProgress atomic.Bool
	var tray *application.SystemTray
	var statusItem *application.MenuItem
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if quitting.Load() {
			return
		}
		app.Event.Emit("weektodo:window-hiding")
		window.Hide()
		event.Cancel()
	})

	showWindow := func() {
		window.Show()
		window.Restore()
		window.Focus()
	}

	refreshTray := func() {
		count := st.TodayCount(time.Now())
		application.InvokeSync(func() {
			if count > 0 {
				tray.SetLabel(strconv.Itoa(count))
			} else {
				tray.SetLabel("")
			}
			tray.SetTooltip(fmt.Sprintf("WeekTodo · 今天 %d 项待办", count))
			statusItem.SetLabel(fmt.Sprintf("今天还有 %d 项待办", count))
		})
	}

	checkUpdates := func() {
		if updateInProgress.Load() {
			return
		}
		info, err := updater.Check(context.Background(), version)
		if err != nil {
			app.Dialog.Error().SetTitle("检查更新").SetMessage(err.Error()).Show()
			return
		}
		if !info.Available {
			message := "当前已是最新版本。"
			if info.Dev {
				message = "当前是开发版本，无法检查更新。"
			}
			app.Dialog.Info().SetTitle("检查更新").SetMessage(message).Show()
			return
		}
		source := ""
		if info.Source == "gh-proxy" {
			source = "（通过 gh-proxy）"
		}
		notes := strings.TrimSpace(info.Notes)
		if len([]rune(notes)) > 1800 {
			notes = string([]rune(notes)[:1800]) + "…"
		}
		message := fmt.Sprintf("WeekTodo %s 已发布%s，是否安装并重启？", info.Latest, source)
		if notes != "" {
			message += "\n\n" + notes
		}
		dialog := app.Dialog.Question().
			SetTitle("发现新版本").
			SetMessage(message)
		dialog.AddButton("稍后").SetAsDefault()
		dialog.AddButton("安装并重启").OnClick(func() {
			if !updateInProgress.CompareAndSwap(false, true) {
				return
			}
			go func() {
				defer updateInProgress.Store(false)
				app.Dialog.Info().
					SetTitle("正在安装更新").
					SetMessage(fmt.Sprintf("正在安装 WeekTodo %s，应用会自动重启。", info.Latest)).
					Show()
				if _, err := updater.Apply(context.Background(), info); err != nil {
					app.Dialog.Error().SetTitle("安装更新失败").SetMessage(err.Error()).Show()
					return
				}
				if err := updater.Restart(); err != nil {
					app.Dialog.Error().SetTitle("重启失败").SetMessage(err.Error()).Show()
				}
			}()
		})
		dialog.Show()
	}

	tray = app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayIcon)
	} else {
		tray.SetIcon(trayIcon)
	}
	tray.SetTooltip("WeekTodo")
	tray.OnClick(showWindow)
	tray.OnRightClick(func() { tray.ShowMenu() })

	menu := app.NewMenu()
	menu.Add("打开 WeekTodo").OnClick(func(*application.Context) { showWindow() })
	menu.Add("快速添加任务").OnClick(func(*application.Context) {
		showWindow()
		time.AfterFunc(180*time.Millisecond, func() {
			app.Event.Emit("weektodo:focus-quick-add")
		})
	})
	menu.Add("检查更新…").OnClick(func(*application.Context) {
		go checkUpdates()
	})
	statusItem = menu.Add("今天还有 0 项待办")
	statusItem.SetEnabled(false)
	menu.AddSeparator()
	menu.Add("隐藏窗口").OnClick(func(*application.Context) { window.Hide() })
	menu.AddSeparator()
	menu.Add("退出 WeekTodo").OnClick(func(*application.Context) {
		app.Event.Emit("weektodo:window-hiding")
		time.Sleep(150 * time.Millisecond)
		quitting.Store(true)
		app.Quit()
	})
	tray.SetMenu(menu)

	app.Event.On("weektodo:changed", func(*application.CustomEvent) {
		refreshTray()
	})

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		showWindow()
		refreshTray()
		go func() {
			ticker := time.NewTicker(60 * time.Second)
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
