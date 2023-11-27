package sql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/maragudk/errors"

	"github.com/maragudk/service/model"
)

// Signup creates an account, a personal group, an unconfirmed user, and a token.
// Also creates a job to send an email with a token.
func (d *Database) Signup(ctx context.Context, u *model.User) error {
	return d.inTransaction(ctx, func(tx *sqlx.Tx) error {
		token, err := createToken()
		if err != nil {
			return err
		}

		var a model.Account
		if err := tx.GetContext(ctx, &a, `insert into accounts (name) values (?) returning *`, u.Name); err != nil {
			return errors.Wrap(err, "error creating account")
		}

		var g model.Group
		query := `insert into groups (accountID, name) values (?, ?) returning *`
		if err := tx.GetContext(ctx, &g, query, a.ID, u.Name); err != nil {
			return errors.Wrap(err, "error creating group")
		}

		var exists bool
		query = `select exists (select * from users where email = ?)`
		if err := tx.GetContext(ctx, &exists, query, u.Email.ToLower()); err != nil {
			return errors.Wrap(err, "error getting user by email")
		}
		if exists {
			return model.ErrorEmailConflict
		}

		query = `insert into users (accountID, name, email) values (?, ?, ?) returning *`
		if err := tx.GetContext(ctx, u, query, a.ID, u.Name, u.Email.ToLower()); err != nil {
			return errors.Wrap(err, "error creating user")
		}

		query = `insert into group_membership (groupID, userID) values (?, ?)`
		if _, err := tx.ExecContext(ctx, query, g.ID, u.ID); err != nil {
			return err
		}

		query = `insert into tokens (value, userID) values (?, ?)`
		if _, err := tx.ExecContext(ctx, query, token, u.ID); err != nil {
			return errors.Wrap(err, "error creating token")
		}

		m := model.Map{
			"type":  "signup",
			"token": token,
		}
		if err := d.createJobInTx(ctx, tx, "send-email", m, 10*time.Second); err != nil {
			return err
		}

		return nil
	})
}

// Login with the given token. It marks the token as used (but this isn't currently checked anywhere)
// if it's not expired and if the user is marked active.
// It also sets the user confirmed.
func (d *Database) Login(ctx context.Context, token string) (*model.ID, error) {
	var userID model.ID
	err := d.inTransaction(ctx, func(tx *sqlx.Tx) error {
		var expired bool
		query := `select exists (select 1 from tokens where value = ? and expires <= strftime('%Y-%m-%dT%H:%M:%fZ'))`
		if err := tx.GetContext(ctx, &expired, query, token); err != nil {
			return err
		}
		if expired {
			return model.ErrorTokenExpired
		}

		var inactive bool
		query = `select exists (select 1 from users where id = (select userID from tokens where value = ?) and not active)`
		if err := tx.GetContext(ctx, &inactive, query, token); err != nil {
			return err
		}
		if inactive {
			return model.ErrorUserInactive
		}

		query = `update tokens set used = 1 where value = ? returning userID`
		if err := tx.GetContext(ctx, &userID, query, token); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrorTokenNotFound
			}
			return err
		}

		query = `update users set confirmed = 1 where id = ? and not confirmed`
		if _, err := tx.ExecContext(ctx, query, userID); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &userID, nil
}

// LoginWithEmail checks whether the user exists and is active, creates a login token, and creates a job to send
// an email with the token in it.
func (d *Database) LoginWithEmail(ctx context.Context, email model.Email) error {
	return d.inTransaction(ctx, func(tx *sqlx.Tx) error {
		var exists bool
		query := `select exists (select 1 from users where email = ?)`
		if err := tx.GetContext(ctx, &exists, query, email); err != nil {
			return err
		}
		if !exists {
			return model.ErrorUserNotFound
		}

		var inactive bool
		query = `select exists (select 1 from users where email = ? and not active)`
		if err := tx.GetContext(ctx, &inactive, query, email); err != nil {
			return err
		}
		if inactive {
			return model.ErrorUserInactive
		}

		token, err := createToken()
		if err != nil {
			return err
		}
		query = `insert into tokens (value, userID) values (?, (select id from users where email = ?))`
		if _, err := tx.ExecContext(ctx, query, token, email); err != nil {
			return errors.Wrap(err, "error creating token")
		}

		m := model.Map{
			"type":  "login",
			"token": token,
		}
		if err := d.createJobInTx(ctx, tx, "send-email", m, 10*time.Second); err != nil {
			return err
		}

		return nil
	})
}

func (d *Database) GetUserFromToken(ctx context.Context, token string) (*model.User, error) {
	var u model.User
	query := `select users.* from users join tokens on tokens.userID = users.id where tokens.value = ?`
	if err := d.DB.GetContext(ctx, &u, query, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (d *Database) GetUser(ctx context.Context, id model.ID) (*model.User, error) {
	var u model.User
	if err := d.DB.GetContext(ctx, &u, `select * from users where id = ?`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrorUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func createToken() (string, error) {
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return fmt.Sprintf("t_%x", secret), nil
}
