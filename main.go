package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const loggerKey contextKey = "logger"

// LoggerFromContext retrieves the structured logger from the context.
// If no logger is found, it returns the default logger.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// StructuredLogger is a middleware that injects a structured logger with the request ID into the context.
func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())
		logger := slog.Default().With("correlation_id", reqID)
		ctx := context.WithValue(r.Context(), loggerKey, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recoverer is a middleware that recovers from panics, logs the panic with the context-aware logger,
// and returns a 500 Internal Server Error status.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				log := LoggerFromContext(r.Context())
				log.Error("panic recovered",
					"error", rvr,
					"stack", string(debug.Stack()),
				)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// SetupRouter configures the chi router with the required middleware chain.
// Note on Middleware Order:
// 1. middleware.RequestID must be injected first to generate/propagate the X-Request-Id.
// 2. StructuredLogger must be injected next to extract the Request ID and inject the context-aware logger.
// 3. Recoverer must be injected last so that it can access the context-aware logger during panic recovery.
func SetupRouter() *chi.Mux {
	r := chi.NewRouter()

	// 1. Inject Request/Correlation ID into context
	r.Use(middleware.RequestID)

	// 2. Initialize Structured Logger (injects logger with correlation ID into context)
	r.Use(StructuredLogger)

	// 3. Recoverer (now has access to context containing the correlation ID)
	r.Use(Recoverer)

	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("simulated database connection failure")
	})

	return r
}

func main() { 
	r := SetupRouter()
	fmt.Println("Starting server on :8080")
	http.ListenAndServe(":8080", r)
}
