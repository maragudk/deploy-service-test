package jobs

import (
	"context"
	"log"

	"github.com/maragudk/errors"

	"github.com/maragudk/service/model"
)

type emailSender interface {
	SendSignupEmail(ctx context.Context, name string, email model.Email, token string) error
	SendLoginEmail(ctx context.Context, name string, email model.Email, token string) error
}

type emailInfoGetter interface {
	GetUserFromToken(ctx context.Context, token string) (*model.User, error)
}

func SendEmail(r registry, log *log.Logger, e emailSender, db emailInfoGetter) {
	r.Register("send-email", func(ctx context.Context, m model.Map) error {
		switch m["type"] {
		case "signup":
			user, err := db.GetUserFromToken(ctx, m["token"])
			if err != nil {
				return errors.Wrap(err, `error getting user from token "%v"`, m["token"])
			}
			if user == nil {
				return nil
			}
			return e.SendSignupEmail(ctx, user.Name, user.Email, m["token"])
		case "login":
			user, err := db.GetUserFromToken(ctx, m["token"])
			if err != nil {
				return errors.Wrap(err, `error getting user from token "%v"`, m["token"])
			}
			if user == nil {
				return nil
			}
			return e.SendLoginEmail(ctx, user.Name, user.Email, m["token"])
		default:
			return errors.Newf(`unknown email type "%v"`, m["type"])
		}
	})
}
