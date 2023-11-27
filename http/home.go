package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	g "github.com/maragudk/gomponents"
	ghttp "github.com/maragudk/gomponents/http"

	"github.com/maragudk/service/html"
)

func Home(mux chi.Router) {
	mux.Get("/", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		user := getUserFromContext(r.Context())
		return html.HomePage(html.PageProps{User: user}), nil
	}))
}
