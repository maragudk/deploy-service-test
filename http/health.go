package http

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type pinger interface {
	Ping(ctx context.Context) error
}

func Health(mux chi.Router, db pinger, log *log.Logger) {
	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			log.Println("Error in health check:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("OK"))
	})
}
