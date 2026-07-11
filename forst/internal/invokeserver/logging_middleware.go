package invokeserver

import (
	"net/http"
	"time"
)

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &statusResponseWriter{ResponseWriter: w}
		next.ServeHTTP(lw, r)
		if s.log == nil {
			return
		}
		status := lw.status
		if status == 0 {
			status = http.StatusOK
		}
		s.log.Infof("http %s %s %d %s", r.Method, r.URL.Path, status, time.Since(start).Round(time.Millisecond))
	})
}
