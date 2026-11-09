package main

import (
	"github.com/uiautodev/weektodo/internal/i18n"
)

type trayLabels struct {
	open         string
	quickAdd     string
	checkUpdates string
	today        string
	hide         string
	quit         string
}

func trayText(lang i18n.Language) trayLabels {
	if lang == i18n.En {
		return trayLabels{
			open:         "Open WeekTodo",
			quickAdd:     "Quick add task",
			checkUpdates: "Check for Updates…",
			today:        "%d tasks left today",
			hide:         "Hide Window",
			quit:         "Quit WeekTodo",
		}
	}
	return trayLabels{
		open:         "打开 WeekTodo",
		quickAdd:     "快速添加任务",
		checkUpdates: "检查更新…",
		today:        "今天还有 %d 项待办",
		hide:         "隐藏窗口",
		quit:         "退出 WeekTodo",
	}
}
