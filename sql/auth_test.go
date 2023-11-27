package sql_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/maragudk/is"

	"github.com/maragudk/service/model"
	"github.com/maragudk/service/sqltest"
)

func TestDatabase_Signup(t *testing.T) {
	t.Run("signs up an account, group, and user, and creates a token and a job", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		u := model.User{
			Name:  "Me",
			Email: "Me@example.com",
		}
		err := db.Signup(context.Background(), &u)
		is.NotError(t, err)

		is.Equal(t, 34, len(u.ID))
		is.True(t, strings.HasPrefix(u.ID.String(), "u_"))
		is.True(t, time.Since(u.Created.T) < time.Second)
		is.True(t, time.Since(u.Updated.T) < time.Second)
		is.Equal(t, "Me", u.Name)
		is.Equal(t, "me@example.com", u.Email.String())
		is.True(t, !u.Confirmed)
		is.True(t, u.Active)

		var a model.Account
		err = db.DB.Get(&a, `select * from accounts where id = ?`, u.AccountID)
		is.NotError(t, err)
		is.Equal(t, 34, len(a.ID))
		is.True(t, strings.HasPrefix(a.ID.String(), "a_"))
		is.True(t, time.Since(a.Created.T) < time.Second)
		is.True(t, time.Since(a.Updated.T) < time.Second)
		is.Equal(t, "Me", a.Name)

		var g model.Group
		err = db.DB.Get(&g, `select * from groups where accountID = ?`, u.AccountID)
		is.NotError(t, err)
		is.Equal(t, 34, len(g.ID))
		is.True(t, strings.HasPrefix(g.ID.String(), "g_"))
		is.True(t, time.Since(g.Created.T) < time.Second)
		is.True(t, time.Since(g.Updated.T) < time.Second)
		is.Equal(t, "Me", g.Name)

		var exists bool
		err = db.DB.Get(&exists, `select exists (select * from group_membership where userID = ? and groupID = ?)`,
			u.ID, g.ID)
		is.NotError(t, err)
		is.True(t, exists)

		var token string
		err = db.DB.Get(&token, `select value from tokens where userID = ?`, u.ID)
		is.NotError(t, err)
		is.Equal(t, 34, len(token))
		is.True(t, strings.HasPrefix(token, "t_"))

		job, err := db.GetJob(context.Background())
		is.NotError(t, err)
		is.NotNil(t, job)
		is.Equal(t, "send-email", job.Name)
		is.Equal(t, "signup", job.Payload["type"])
		is.Equal(t, token, job.Payload["token"])
	})

	t.Run("errors on duplicate email", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		u := model.User{
			Name:  "Me",
			Email: "Me@example.com",
		}
		err := db.Signup(context.Background(), &u)
		is.NotError(t, err)

		err = db.Signup(context.Background(), &u)
		is.Error(t, model.ErrorEmailConflict, err)
	})
}

func TestDatabase_Login(t *testing.T) {
	t.Run("marks token used and user confirmed and returns user id", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		u := model.User{
			Name:  "Me",
			Email: "Me@example.com",
		}
		err := db.Signup(context.Background(), &u)
		is.NotError(t, err)

		var token string
		err = db.DB.Get(&token, `select value from tokens where userID = ?`, u.ID)
		is.NotError(t, err)

		userID, err := db.Login(context.Background(), token)
		is.NotError(t, err)
		is.NotNil(t, userID)
		is.Equal(t, u.ID, *userID)

		var used bool
		err = db.DB.Get(&used, `select used from tokens where value = ?`, token)
		is.NotError(t, err)
		is.True(t, used)

		var confirmed bool
		err = db.DB.Get(&confirmed, `select confirmed from users where id = ?`, userID)
		is.NotError(t, err)
		is.True(t, confirmed)
	})

	t.Run("returns token expired error when token is expired", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		u := model.User{
			Name:  "Me",
			Email: "Me@example.com",
		}
		err := db.Signup(context.Background(), &u)
		is.NotError(t, err)

		var token string
		err = db.DB.Get(&token, `select value from tokens where userID = ?`, u.ID)
		is.NotError(t, err)

		_, err = db.DB.Exec(`update tokens set expires = '2001-01-01T00:00:00.000Z' where value = ?`, token)
		is.NotError(t, err)

		userID, err := db.Login(context.Background(), token)
		is.Error(t, model.ErrorTokenExpired, err)
		is.Nil(t, userID)
	})

	t.Run("returns user inactive error when user not active", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		u := model.User{
			Name:  "Me",
			Email: "Me@example.com",
		}
		err := db.Signup(context.Background(), &u)
		is.NotError(t, err)

		var token string
		err = db.DB.Get(&token, `select value from tokens where userID = ?`, u.ID)
		is.NotError(t, err)

		_, err = db.DB.Exec(`update users set active = false where id = ?`, u.ID)
		is.NotError(t, err)

		userID, err := db.Login(context.Background(), token)
		is.Error(t, model.ErrorUserInactive, err)
		is.Nil(t, userID)
	})

	t.Run("returns error if no such token", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		userID, err := db.Login(context.Background(), "t_018743ccfbed090e8f5eebc810ff797d")
		is.Error(t, model.ErrorTokenNotFound, err)
		is.Nil(t, userID)
	})
}

func TestDatabase_LoginWithEmail(t *testing.T) {
	t.Run("creates token and job to send login email", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		u := model.User{
			Name:  "Me",
			Email: "me@example.com",
		}
		err := db.Signup(context.Background(), &u)
		is.NotError(t, err)

		job, err := db.GetJob(context.Background())
		is.NotError(t, err)
		token1 := job.Payload["token"]

		err = db.LoginWithEmail(context.Background(), "me@example.com")
		is.NotError(t, err)

		var token2 string
		err = db.DB.Get(&token2, `select value from tokens where userID = ? and value != ?`, u.ID, token1)
		is.NotError(t, err)

		job, err = db.GetJob(context.Background())
		is.NotError(t, err)
		is.NotNil(t, job)
		is.Equal(t, "send-email", job.Name)
		is.Equal(t, "login", job.Payload["type"])
		is.Equal(t, token2, job.Payload["token"])
	})

	t.Run("errors when user not found", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		err := db.LoginWithEmail(context.Background(), "doesnotexist@example.com")
		is.Error(t, model.ErrorUserNotFound, err)
	})

	t.Run("errors when user inactive", func(t *testing.T) {
		db := sqltest.CreateDatabase(t)

		u := model.User{
			Name:  "Me",
			Email: "me@example.com",
		}
		err := db.Signup(context.Background(), &u)
		is.NotError(t, err)

		_, err = db.DB.Exec(`update users set active = false where id = ?`, u.ID)
		is.NotError(t, err)

		err = db.LoginWithEmail(context.Background(), "me@example.com")
		is.Error(t, model.ErrorUserInactive, err)
	})
}
