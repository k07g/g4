package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/k07g/g4/internal/models"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, cognitoSub, email string) (*models.User, error) {
	const q = `
		INSERT INTO users (cognito_sub, email)
		VALUES ($1, $2)
		RETURNING id, cognito_sub, email, created_at, updated_at
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, cognitoSub, email).
		Scan(&u.ID, &u.CognitoSub, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) GetByCognitoSub(ctx context.Context, cognitoSub string) (*models.User, error) {
	const q = `
		SELECT id, cognito_sub, email, created_at, updated_at
		FROM users
		WHERE cognito_sub = $1
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, cognitoSub).
		Scan(&u.ID, &u.CognitoSub, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) DeleteByCognitoSub(ctx context.Context, cognitoSub string) error {
	const q = `DELETE FROM users WHERE cognito_sub = $1`
	res, err := r.db.ExecContext(ctx, q, cognitoSub)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}
