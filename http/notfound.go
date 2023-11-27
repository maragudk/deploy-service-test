package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	g "github.com/maragudk/gomponents"
	ghttp "github.com/maragudk/gomponents/http"

	"github.com/maragudk/service/html"
)

func NotFound(mux chi.Router) {
	mux.NotFound(ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		return html.NotFoundPage(), nil
	}))
}
