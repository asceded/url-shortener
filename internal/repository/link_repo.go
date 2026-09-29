package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asceded/url-shortener/internal/model"
)

type LinkRepository struct {
	pool *pgxpool.Pool
}

func NewLinkRepository(pool *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{pool: pool}
}

func (r *LinkRepository) Create(ctx context.Context, code, originalURL string) (*model.Link, error) {
	const query = `
		INSERT INTO links (code, original_url)
		VALUES ($1, $2)
		RETURNING id, code, original_url, created_at
	`

	var link model.Link
	err := r.pool.QueryRow(ctx, query, code, originalURL).Scan(
		&link.ID, &link.Code, &link.OriginalURL, &link.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert link: %w", err)
	}
	return &link, nil
}

func (r *LinkRepository) GetByCode(ctx context.Context, code string) (*model.Link, error) {
	const query = `SELECT id, code, original_url, created_at FROM links WHERE code = $1`

	var link model.Link
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&link.ID, &link.Code, &link.OriginalURL, &link.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLinkNotFound
		}
		return nil, fmt.Errorf("select link: %w", err)
	}
	return &link, nil
}

func (r *LinkRepository) Delete(ctx context.Context, code string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM links WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrLinkNotFound
	}
	return nil
}
