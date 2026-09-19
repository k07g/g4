package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func newMockRepo(t *testing.T) (*UserRepository, sqlmock.Sqlmock) {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })

	return NewUserRepository(mockDB), mock
}

func TestUserRepository_Create(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "cognito_sub", "email", "created_at", "updated_at"}).
			AddRow("11111111-1111-1111-1111-111111111111", "sub-123", "user@example.com", now, now)

		mock.ExpectQuery("INSERT INTO users").
			WithArgs("sub-123", "user@example.com").
			WillReturnRows(rows)

		u, err := repo.Create(context.Background(), "sub-123", "user@example.com")
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		if u.ID != "11111111-1111-1111-1111-111111111111" || u.CognitoSub != "sub-123" || u.Email != "user@example.com" {
			t.Errorf("unexpected user: %+v", u)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("duplicate email propagates db error", func(t *testing.T) {
		dbErr := errors.New("duplicate key value violates unique constraint")
		mock.ExpectQuery("INSERT INTO users").
			WithArgs("sub-456", "dup@example.com").
			WillReturnError(dbErr)

		_, err := repo.Create(context.Background(), "sub-456", "dup@example.com")
		if err == nil {
			t.Fatal("Create returned nil error, want the underlying db error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func TestUserRepository_GetByCognitoSub(t *testing.T) {
	repo, mock := newMockRepo(t)
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "cognito_sub", "email", "created_at", "updated_at"}).
			AddRow("11111111-1111-1111-1111-111111111111", "sub-123", "user@example.com", now, now)

		mock.ExpectQuery("SELECT id, cognito_sub, email, created_at, updated_at").
			WithArgs("sub-123").
			WillReturnRows(rows)

		u, err := repo.GetByCognitoSub(context.Background(), "sub-123")
		if err != nil {
			t.Fatalf("GetByCognitoSub returned error: %v", err)
		}
		if u.Email != "user@example.com" {
			t.Errorf("Email = %q, want %q", u.Email, "user@example.com")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, cognito_sub, email, created_at, updated_at").
			WithArgs("missing-sub").
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetByCognitoSub(context.Background(), "missing-sub")
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("GetByCognitoSub error = %v, want %v", err, ErrUserNotFound)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUserRepository_DeleteByCognitoSub(t *testing.T) {
	repo, mock := newMockRepo(t)

	t.Run("deletes existing user", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM users").
			WithArgs("sub-123").
			WillReturnResult(sqlmock.NewResult(0, 1))

		if err := repo.DeleteByCognitoSub(context.Background(), "sub-123"); err != nil {
			t.Errorf("DeleteByCognitoSub returned error: %v", err)
		}
	})

	t.Run("returns ErrUserNotFound when nothing deleted", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM users").
			WithArgs("missing-sub").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.DeleteByCognitoSub(context.Background(), "missing-sub")
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("DeleteByCognitoSub error = %v, want %v", err, ErrUserNotFound)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
