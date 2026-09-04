package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const userColumns = `id, name, email, password_hash, role`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role)
	return u, err
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

type SeedUser struct {
	Name     string
	Email    string
	Password string
	Role     Role
}

// EnsureSeed inserts the default accounts when the users table is empty.
func (r *Repository) EnsureSeed(ctx context.Context, seeds []SeedUser) error {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}
	for _, s := range seeds {
		hash, err := HashPassword(s.Password)
		if err != nil {
			return fmt.Errorf("hash seed password: %w", err)
		}
		if _, err := r.pool.Exec(ctx,
			`INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, $3, $4)`,
			s.Name, s.Email, hash, s.Role); err != nil {
			return fmt.Errorf("seed user %s: %w", s.Email, err)
		}
	}
	return nil
}
