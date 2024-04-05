package middleware

import (
	"net/http"
	"text/template"
)

type ApiMetrics struct {
	fileServerHits int
}

func (m *ApiMetrics) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.fileServerHits += 1
		next.ServeHTTP(w, r)
	})
}

func (m *ApiMetrics) HandleGetMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseGlob("*.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	tmpl.ExecuteTemplate(w, "metrics.html", map[string]interface{}{
		"Hits": m.fileServerHits,
	})
}

func (m *ApiMetrics) HandleResetMetrics(w http.ResponseWriter, _ *http.Request) {
	m.fileServerHits = 0
	w.WriteHeader(http.StatusOK)
}
