package http

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/maragudk/errors"
	g "github.com/maragudk/gomponents"
	ghttp "github.com/maragudk/gomponents/http"
	"github.com/maragudk/httph"

	"github.com/maragudk/service/html"
	"github.com/maragudk/service/model"
)

type contextKey string

const contextUserKey = contextKey("user")
const sessionUserIDKey = "userID"

// getUserFromContext, which may be nil if the user is not authenticated.
func getUserFromContext(ctx context.Context) *model.User {
	user := ctx.Value(contextUserKey)
	if user == nil {
		return nil
	}
	return user.(*model.User)
}

type signupper interface {
	Signup(ctx context.Context, u *model.User) error
}

type signupRequest struct {
	Name   string
	Email  model.Email
	Accept bool
}

func (s signupRequest) Validate() error {
	if s.Name == "" {
		return errors.New("name cannot be empty")
	}

	if !s.Email.IsValid() {
		return errors.New("email is invalid")
	}

	if !s.Accept {
		return errors.New("not accepted")
	}

	return nil
}

func Signup(mux chi.Router, log *log.Logger, db signupper) {
	mux.Get("/signup", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		user := getUserFromContext(r.Context())
		if user != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return nil, nil
		}

		return html.SignupPage(html.PageProps{}, model.User{}), nil
	}))

	mux.Post("/signup", httph.FormHandler(func(w http.ResponseWriter, r *http.Request, req signupRequest) {
		ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
			// TODO this should be middleware
			user := getUserFromContext(r.Context())
			if user != nil {
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return nil, nil
			}

			u := model.User{Name: req.Name, Email: req.Email}

			if err := db.Signup(r.Context(), &u); err != nil {
				if errors.Is(err, model.ErrorEmailConflict) {
					return html.SignupPage(html.PageProps{}, u), nil
				}
				log.Println("Error signing up:", err)
				return html.ErrorPage(), err
			}

			http.Redirect(w, r, "/signup/thanks", http.StatusFound)
			return nil, nil
		})(w, r)
	}))

	mux.Get("/signup/thanks", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		return html.SignupThanksPage(html.PageProps{}), nil
	}))
}

type loginner interface {
	Login(ctx context.Context, token string) (*model.ID, error)
	LoginWithEmail(ctx context.Context, email model.Email) error
}

type sessionPutter interface {
	RenewToken(ctx context.Context) error
	Put(ctx context.Context, key string, value interface{})
}

type loginEmailRequest struct {
	Email model.Email
}

func (l loginEmailRequest) Validate() error {
	if !l.Email.IsValid() {
		return errors.New("email is invalid")
	}
	return nil
}

type loginTokenRequest struct {
	Token string
}

func (l loginTokenRequest) Validate() error {
	if l.Token == "" {
		return errors.New("token is invalid")
	}
	return nil
}

func Login(mux chi.Router, log *log.Logger, db loginner, sp sessionPutter) {
	mux.Get("/login", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		user := getUserFromContext(r.Context())
		if user != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return nil, nil
		}

		token := r.URL.Query().Get("token")
		if token != "" {
			return html.LoginTokenPage(html.PageProps{}, token), nil
		}
		return html.LoginPage(html.PageProps{}, ""), nil
	}))

	mux.Post("/login/email", httph.FormHandler(func(w http.ResponseWriter, r *http.Request, req loginEmailRequest) {
		ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
			if err := db.LoginWithEmail(r.Context(), req.Email); err != nil {
				switch {
				case errors.Is(err, model.ErrorUserInactive):
					return html.UserInactivePage(html.PageProps{}), nil
				case errors.Is(err, model.ErrorUserNotFound):
					return html.LoginPage(html.PageProps{}, req.Email.String()), nil
				default:
					log.Println("Error logging in with email:", err)
					return html.ErrorPage(), err
				}
			}

			http.Redirect(w, r, "/login/email", http.StatusFound)
			return nil, nil
		})(w, r)
	}))

	mux.Get("/login/email", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		user := getUserFromContext(r.Context())
		if user != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return nil, nil
		}

		return html.LoginCheckPage(html.PageProps{}), nil
	}))

	mux.Post("/login/token", httph.FormHandler(func(w http.ResponseWriter, r *http.Request, req loginTokenRequest) {
		ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
			userID, err := db.Login(r.Context(), req.Token)
			if err != nil {
				switch {
				case errors.Is(err, model.ErrorUserInactive):
					return html.UserInactivePage(html.PageProps{}), nil
				case errors.Is(err, model.ErrorTokenExpired), errors.Is(err, model.ErrorTokenNotFound):
					return html.TokenExpiredPage(html.PageProps{}), nil
				default:
					log.Println("Error logging in with token:", err)
					return html.ErrorPage(), err
				}
			}

			// Renew the session token to avoid session fixation attacks
			if err := sp.RenewToken(r.Context()); err != nil {
				log.Println("Error renewing session token during token confirm:", err)
				return html.ErrorPage(), err
			}

			sp.Put(r.Context(), sessionUserIDKey, string(*userID))

			http.Redirect(w, r, "/", http.StatusFound)
			return nil, nil
		})(w, r)
	}))
}

type sessionDestroyer interface {
	Destroy(ctx context.Context) error
}

// Logout creates an http.Handler for logging out.
// It just destroys the current user session.
func Logout(mux chi.Router, s sessionDestroyer, log *log.Logger) {
	mux.Post("/logout", ghttp.Adapt(func(w http.ResponseWriter, r *http.Request) (g.Node, error) {
		if err := s.Destroy(r.Context()); err != nil {
			log.Println("Error logging out:", err)
			return html.ErrorPage(), err
		}

		http.Redirect(w, r, "/", http.StatusFound)
		return nil, nil
	}))
}

type sessionGetter interface {
	sessionDestroyer
	Exists(ctx context.Context, key string) bool
	GetString(ctx context.Context, key string) string
}

type userGetter interface {
	GetUser(ctx context.Context, id model.ID) (*model.User, error)
}

// Authenticate checks that there's a user logged in, and otherwise either:
// - redirects to the login page,
// - or calls the next handler
// depending on the passed parameter.
func Authenticate(redirect bool, sg sessionGetter, db userGetter, log *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !sg.Exists(r.Context(), sessionUserIDKey) {
				if redirect {
					http.Redirect(w, r, "/login", http.StatusFound)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			userID := model.ID(sg.GetString(r.Context(), sessionUserIDKey))
			user, err := db.GetUser(r.Context(), userID)
			if err != nil || user == nil {
				log.Println("Error getting user after authentication:", err, user)
				http.Error(w, "error getting user after authentication", http.StatusInternalServerError)
				return
			}

			if !user.Active {
				if err := sg.Destroy(r.Context()); err != nil {
					log.Println("Error destroying session for inactive user:", err, user.ID)
					http.Error(w, "error after authentication", http.StatusInternalServerError)
					return
				}

				if redirect {
					http.Redirect(w, r, "/login", http.StatusFound)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			// We store the user directly in the context instead of having to use the session manager
			ctx := context.WithValue(r.Context(), contextUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
