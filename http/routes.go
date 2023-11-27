package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/honeybadger-io/honeybadger-go"
	"github.com/maragudk/dblens"
	"github.com/maragudk/httph"
)

func (s *Server) setupRoutes() {
	s.mux.Group(func(r chi.Router) {
		r.Use(middleware.Recoverer, honeybadger.Handler)
		r.Use(middleware.Compress(5))
		r.Use(middleware.RealIP)
		r.Use(AddMetrics(s.metrics))

		Health(r, s.database, s.log)
		Metrics(r, s.metrics)

		r.Group(func(r chi.Router) {
			r.Use(VersionedAssets)

			Static(r)
		})

		r.Group(func(r chi.Router) {
			r.Use(httph.NoClickjacking, httph.ContentSecurityPolicy(func(opts *httph.ContentSecurityPolicyOptions) {
				opts.ConnectSrc = "'self' https://cdn.usefathom.com"
				opts.ImgSrc = "'self' https://cdn.usefathom.com"
				opts.ManifestSrc = "'self'"
				opts.ScriptSrc = "'self' https://cdn.usefathom.com"
			}))
			r.Use(middleware.SetHeader("Content-Type", "text/html; charset=utf-8"))
			r.Use(s.sm.LoadAndSave)

			r.Group(func(r chi.Router) {
				r.Use(Authenticate(false, s.sm, s.database, s.log))

				Home(r)
				Signup(r, s.log, s.database)
				Login(r, s.log, s.database, s.sm)
				Logout(r, s.sm, s.log)
			})

			Legal(r)
			NotFound(r)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.BasicAuth("admin", map[string]string{"admin": s.adminPassword}))

			Migrate(r, s.database)
			r.Get("/dblens", dblens.Handler(s.database.DB.DB, "sqlite3"))
		})
	})
}
