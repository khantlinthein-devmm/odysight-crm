package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("user not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const userColumns = `id, name, email, role, created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	return u, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]User, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		args = append(args, like, like, like)
		conds = append(conds, "(name ILIKE $1 OR email ILIKE $2 OR role ILIKE $3)")
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, fmt.Sprintf("role = $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + conds[0]
		for _, c := range conds[1:] {
			where += " AND " + c
		}
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + userColumns + ` FROM users ` + where +
		fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	items := []User{}
	for rows.Next() {
		item, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate users: %w", err)
	}
	return items, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user %d: %w", id, err)
	}
	return u, nil
}

func (r *Repository) Create(ctx context.Context, name, email, hash, role string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, $3, $4)
		 RETURNING `+userColumns,
		name, email, hash, role).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

type Patch struct {
	Name *string
	Role *string
}

func (r *Repository) Update(ctx context.Context, id int64, patch Patch) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET
			name = COALESCE($2, name),
			role = COALESCE($3, role)
		 WHERE id = $1
		 RETURNING `+userColumns,
		id, patch.Name, patch.Role).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("update user %d: %w", id, err)
	}
	return u, nil
}

func (r *Repository) UpdatePassword(ctx context.Context, id int64, hash string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, hash)
	if err != nil {
		return fmt.Errorf("update password %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CountByRole(ctx context.Context, role string) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = $1`, role).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users by role: %w", err)
	}
	return n, nil
}
