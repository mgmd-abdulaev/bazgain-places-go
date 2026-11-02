package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/tx-mi/bazgain-places-go/internal/auth"
)

func (r *Repository) FindAuthUser(ctx context.Context, id string) (*auth.User, error) {
	ctx, cancel := r.withQueryTimeout(ctx)
	defer cancel()
	var user auth.User
	err := r.pool.QueryRow(ctx, `SELECT id::text,state::text,role::text,token_version FROM users WHERE id=$1::uuid`, id).Scan(&user.ID, &user.State, &user.Role, &user.TokenVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
