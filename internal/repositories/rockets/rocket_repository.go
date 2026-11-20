package rockets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rockets/internal/models"
	"rockets/internal/xerrors"
)

const rocketColumns = `channel, status, rocket_type, speed, mission, explosion_reason,
	last_processed_number, last_message_time`

const selectRocket = `
	SELECT ` + rocketColumns + `,
	       (SELECT count(*) FROM rocket_events e WHERE e.channel = r.channel AND e.status = 'pending')
	FROM rockets r`

const lockRocket = `
	INSERT INTO rockets (channel) VALUES ($1)
	ON CONFLICT (channel) DO UPDATE SET channel = EXCLUDED.channel
	RETURNING ` + rocketColumns + `, 0::bigint`

var sortColumns = map[string]string{
	"channel": "r.channel",
	"type":    "r.rocket_type",
	"speed":   "r.speed",
	"mission": "r.mission",
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(ctx context.Context, dsn string) (*Repository, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 10
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Repository{pool: pool}, nil
}

func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }
func (r *Repository) Close()                         { r.pool.Close() }

func (r *Repository) WithRocketLock(ctx context.Context, channel string, fn func(*Tx, models.Rocket) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		rbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(rbCtx)
	}()

	rocket, err := scanRocket(tx.QueryRow(ctx, lockRocket, channel))
	if err != nil {
		return err
	}
	if err := fn(&Tx{tx: tx, channel: channel}, rocket); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Rocket(ctx context.Context, channel string) (models.Rocket, error) {
	return scanRocket(r.pool.QueryRow(ctx, selectRocket+` WHERE r.channel = $1`, channel))
}

func (r *Repository) Rockets(ctx context.Context, opts models.ListOptions) ([]models.Rocket, error) {
	column, ok := sortColumns[opts.Sort]
	if !ok {
		return nil, xerrors.ErrInvalidQuery
	}
	direction := "ASC"
	if opts.Order == "desc" {
		direction = "DESC"
	}

	rows, err := r.pool.Query(ctx, selectRocket+` WHERE r.status <> 'pending'
		ORDER BY `+column+` `+direction+`, r.channel
		LIMIT $1 OFFSET $2`, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rockets := make([]models.Rocket, 0, opts.Limit)
	for rows.Next() {
		rocket, err := scanRocket(rows)
		if err != nil {
			return nil, err
		}
		rockets = append(rockets, rocket)
	}
	return rockets, rows.Err()
}

func scanRocket(row pgx.Row) (models.Rocket, error) {
	var r models.Rocket
	var lastTime *time.Time
	err := row.Scan(&r.Channel, &r.Status, &r.Type, &r.Speed, &r.Mission, &r.ExplosionReason,
		&r.LastProcessed, &lastTime, &r.PendingCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, xerrors.ErrNotFound
	}
	if lastTime != nil {
		r.LastMessageTime = lastTime.UTC()
	}
	return r, err
}
