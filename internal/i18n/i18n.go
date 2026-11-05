package i18n

import (
	"net/http"
	"os"
	"strings"
	"time"
)

// Language is one of the supported UI languages.
type Language string

const (
	Zh Language = "zh"
	En Language = "en"
)

// Normalize converts a language tag such as zh-CN or en-US to a supported language.
func Normalize(value string) Language {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(value, "en"):
		return En
	case strings.HasPrefix(value, "zh"):
		return Zh
	default:
		return ""
	}
}

// Detect reads Accept-Language from an HTTP request.
func Detect(r *http.Request) Language {
	if lang := Normalize(r.Header.Get("Accept-Language")); lang != "" {
		return lang
	}
	return Zh
}

// System detects the desktop/system language from the environment.
func System() Language {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		value = strings.SplitN(value, ".", 2)[0]
		if lang := Normalize(value); lang != "" {
			return lang
		}
	}
	return En
}

// T picks the localized variant.
func T(lang Language, zh, en string) string {
	if lang == En {
		return en
	}
	return zh
}

var weekdayZh = [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
var weekdayEn = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
var monthEn = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// Weekday returns a localized short weekday.
func Weekday(lang Language, t time.Time) string {
	if lang == En {
		return weekdayEn[int(t.Weekday())]
	}
	return weekdayZh[int(t.Weekday())]
}

// MonthDay returns a localized date such as 11月2日 or Nov 2.
func MonthDay(lang Language, t time.Time) string {
	if lang == En {
		return monthEn[int(t.Month())-1] + " " + itoa(t.Day())
	}
	return itoa(int(t.Month())) + "月" + itoa(t.Day()) + "日"
}

// FormatRange returns a localized week date range.
func FormatRange(lang Language, start, end time.Time) string {
	return MonthDay(lang, start) + " – " + MonthDay(lang, end)
}

func itoa(value int) string {
	if value < 10 {
		return string(rune('0' + value))
	}
	return string(rune('0'+value/10)) + string(rune('0'+value%10))
}

var errorTranslations = map[string]string{
	"这一周还没有记录":                    "There is no record for this week",
	"无法计算上一周":                     "Could not calculate the previous week",
	"当前已经是最新版本":                   "Already up to date",
	"已有更新正在进行":                    "An update is already in progress",
	"not found":                   "Not found",
	"method not allowed":          "Method not allowed",
	"frontend is not built":       "Frontend is not built",
	"frontend assets unavailable": "Frontend assets are unavailable",
	"任务层级最多 8 层":                  "Tasks can be nested at most 8 levels deep",
	"单周任务数量不能超过 2000 个":           "A week can contain at most 2000 tasks",
}

// TranslateError translates known server messages. Unknown technical messages fall back.
func TranslateError(lang Language, message string) string {
	if lang != En {
		return message
	}
	if translated, ok := errorTranslations[message]; ok {
		return translated
	}
	return message
}
