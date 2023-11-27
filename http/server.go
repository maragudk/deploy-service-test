package http

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/maragudk/aws/s3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/maragudk/service/sql"
)

type Server struct {
	address       string
	adminPassword string
	bucket        *s3.Bucket
	database      *sql.Database
	log           *log.Logger
	metrics       *prometheus.Registry
	mux           chi.Router
	server        *http.Server
	sm            *scs.SessionManager
}

type NewServerOptions struct {
	AdminPassword string
	Bucket        *s3.Bucket
	Database      *sql.Database
	Host          string
	Log           *log.Logger
	Metrics       *prometheus.Registry
	Port          int
	SecureCookie  bool
}

// NewServer returns an initialized, but unstarted Server.
// If no logger is provided, logs are discarded.
func NewServer(opts NewServerOptions) *Server {
	if opts.Log == nil {
		opts.Log = log.New(io.Discard, "", 0)
	}

	if opts.Metrics == nil {
		opts.Metrics = prometheus.NewRegistry()
	}

	address := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	mux := chi.NewMux()

	sm := scs.New()
	sm.Store = sqlite3store.New(opts.Database.DB.DB)
	sm.Lifetime = 365 * 24 * time.Hour
	sm.Cookie.Secure = opts.SecureCookie

	return &Server{
		address:       address,
		adminPassword: opts.AdminPassword,
		bucket:        opts.Bucket,
		database:      opts.Database,
		log:           opts.Log,
		metrics:       opts.Metrics,
		mux:           mux,
		server: &http.Server{
			Addr:              address,
			Handler:           mux,
			ErrorLog:          opts.Log,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       5 * time.Second,
		},
		sm: sm,
	}
}

func (s *Server) Start() error {
	s.log.Println("Starting")

	s.setupRoutes()

	s.log.Println("Listening on http://" + s.address)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop() error {
	s.log.Println("Stopping")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}
	s.log.Println("Stopped")
	return nil
}
