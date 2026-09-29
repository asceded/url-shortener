package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/asceded/url-shortener/internal/model"
)

type ClickRepository struct {
	pool *pgxpool.Pool
}

func NewClickRepository(pool *pgxpool.Pool) *ClickRepository {
	return &ClickRepository{pool: pool}
}

func (r *ClickRepository) SaveBatch(ctx context.Context, clicks []model.Click) error {
	if len(clicks) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, c := range clicks {
		batch.Queue(
			`INSERT INTO clicks (link_id, ip, user_agent) VALUES ($1, $2, $3)`,
			c.LinkID, c.IP, c.UserAgent,
		)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range clicks {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("save click batch: %w", err)
		}
	}

	return nil
}

func (r *ClickRepository) GetStats(ctx context.Context, code string) (*model.LinkStats, error) {
	query := `
		SELECT l.code, COUNT(c.id), MAX(c.clicked_at)
		FROM links l
		LEFT JOIN clicks c ON c.link_id = l.id
		WHERE l.code = $1
		GROUP BY l.code
	`

	var stats model.LinkStats
	var lastClick *time.Time

	err := r.pool.QueryRow(ctx, query, code).Scan(
		&stats.Code, &stats.TotalClicks, &lastClick,
	)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	stats.LastClick = lastClick
	return &stats, nil
}
