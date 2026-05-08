package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func SlogError(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func Error(w http.ResponseWriter, error string, code int, slogAttrs ...slog.Attr) {
	attrs := []any{
		slog.Int("status", code),
		slog.String("error", error),
	}
	for _, attr := range slogAttrs {
		attrs = append(attrs, attr)
	}

	slog.Error("response", attrs...)
	http.Error(w, error, code)
}

func JSONResponse(w http.ResponseWriter, data any, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}
	if data == nil {
		data = struct{}{}
	}

	content, err := json.Marshal(data)
	if err != nil {
		slog.Error("response marshal failed", SlogError(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	slog.Debug("response",
		slog.Int("status", code),
		slog.String("content", string(content)),
	)

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	w.Write(content)
}

func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		next.ServeHTTP(w, r)
	})
}

func UnknownRouteHandler(w http.ResponseWriter, r *http.Request) {
	slog.Warn("unknown route",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)

	http.Error(w, "unknown route", http.StatusNotFound)
}
