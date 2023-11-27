package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	g "github.com/maragudk/gomponents"
	ghttp "github.com/maragudk/gomponents/http"

	"github.com/maragudk/service/html"
)

func Legal(mux chi.Router) {
	mux.Get("/legal/terms-of-service", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		return html.TermsOfServicePage(html.PageProps{}), nil
	}))

	mux.Get("/legal/privacy-policy", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		return html.PrivacyPolicyPage(html.PageProps{}), nil
	}))

	mux.Get("/legal/subprocessors", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		return html.SubProcessorsPage(html.PageProps{}), nil
	}))
}
