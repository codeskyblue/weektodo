package main

import (
	"fmt"

	"github.com/uiautodev/weektodo/internal/i18n"
)

type trayLabels struct {
	open              string
	quickAdd          string
	checkUpdates      string
	today             string
	hide              string
	quit              string
	upToDate          string
	devBuild          string
	updateAvailable   string
	updateMessage     string
	installing        string
	installingMessage string
	later             string
	install           string
	checkFailed       string
	installFailed     string
	restartFailed     string
	viaProxy          string
}

func trayText(lang i18n.Language) trayLabels {
	if lang == i18n.En {
		return trayLabels{
			open:              "Open WeekTodo",
			quickAdd:          "Quick add task",
			checkUpdates:      "Check for Updates…",
			today:             "%d tasks left today",
			hide:              "Hide Window",
			quit:              "Quit WeekTodo",
			upToDate:          "WeekTodo is up to date.",
			devBuild:          "Development builds cannot check for updates.",
			updateAvailable:   "Update available",
			updateMessage:     "WeekTodo %s is available%s. Install and restart now?",
			installing:        "Installing update",
			installingMessage: "Installing WeekTodo %s. The app will restart automatically.",
			later:             "Later",
			install:           "Install and Restart",
			checkFailed:       "Update check failed",
			installFailed:     "Update installation failed",
			restartFailed:     "Restart failed",
			viaProxy:          " (via gh-proxy)",
		}
	}
	return trayLabels{
		open:              "打开 WeekTodo",
		quickAdd:          "快速添加任务",
		checkUpdates:      "检查更新…",
		today:             "今天还有 %d 项待办",
		hide:              "隐藏窗口",
		quit:              "退出 WeekTodo",
		upToDate:          "当前已是最新版本。",
		devBuild:          "当前是开发版本，无法检查更新。",
		updateAvailable:   "发现新版本",
		updateMessage:     "WeekTodo %s 已发布%s，是否安装并重启？",
		installing:        "正在安装更新",
		installingMessage: "正在安装 WeekTodo %s，应用会自动重启。",
		later:             "稍后",
		install:           "安装并重启",
		checkFailed:       "检查更新失败",
		installFailed:     "安装更新失败",
		restartFailed:     "重启失败",
		viaProxy:          "（通过 gh-proxy）",
	}
}

func todayLabel(lang i18n.Language, count int) string {
	return fmt.Sprintf(trayText(lang).today, count)
}
