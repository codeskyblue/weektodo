package server

import (
	"net/http"

	"github.com/uiautodev/weektodo/internal/i18n"
)

type languageWriter struct {
	http.ResponseWriter
	lang i18n.Language
}

func (w *languageWriter) Language() i18n.Language {
	return w.lang
}

func withLanguage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&languageWriter{ResponseWriter: w, lang: i18n.Detect(r)}, r)
	})
}

func responseLanguage(w http.ResponseWriter) i18n.Language {
	if localized, ok := w.(interface{ Language() i18n.Language }); ok {
		return localized.Language()
	}
	return i18n.Zh
}

func requestLanguage(r *http.Request, fallback string) i18n.Language {
	if lang := i18n.Normalize(r.URL.Query().Get("lang")); lang != "" {
		return lang
	}
	if lang := i18n.Normalize(fallback); lang != "" {
		return lang
	}
	return i18n.Detect(r)
}
