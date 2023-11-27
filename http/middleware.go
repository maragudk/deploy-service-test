package http

import (
	"net/http"
)

// Middleware is an alias for a function that takes a handler and returns one, too.
type Middleware = func(http.Handler) http.Handler
