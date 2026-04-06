package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/wb-go/wbf/dbpg"
	wbfredis "github.com/wb-go/wbf/redis"

	"github.com/ustithegod/url-shortener/internal/storage"
)

const (
	cacheKeyPrefix = "short-url:"
	groupByRaw     = "raw"
	groupByDay     = "day"
	groupByMonth   = "month"
	groupByAgent   = "user_agent"
)

type Storage struct {
	db       *dbpg.DB
	cache    *wbfredis.Client
	cacheTTL time.Duration
}

func New(db *dbpg.DB, cache *wbfredis.Client, cacheTTL time.Duration) *Storage {
	return &Storage{
		db:       db,
		cache:    cache,
		cacheTTL: cacheTTL,
	}
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.postgres.SaveURL"

	var id int64
	err := s.db.Master.QueryRow(
		"INSERT INTO urls (url, alias) VALUES ($1, $2) RETURNING id",
		urlToSave,
		alias,
	).Scan(&id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if s.cache != nil {
		_ = s.cache.SetWithExpiration(context.Background(), cacheKey(alias), urlToSave, s.cacheTTL)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.postgres.GetURL"

	if s.cache != nil {
		url, err := s.cache.Get(context.Background(), cacheKey(alias))
		if err == nil {
			return url, nil
		}

		if !errors.Is(err, wbfredis.NoMatches) {
			return "", fmt.Errorf("%s: cache get: %w", op, err)
		}
	}

	var url string
	err := s.db.QueryRowContext(context.Background(), "SELECT url FROM urls WHERE alias = $1", alias).Scan(&url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if s.cache != nil {
		_ = s.cache.SetWithExpiration(context.Background(), cacheKey(alias), url, s.cacheTTL)
	}

	return url, nil
}

func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.postgres.DeleteURL"

	res, err := s.db.ExecContext(context.Background(), "DELETE FROM urls WHERE alias = $1", alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected > 0 && s.cache != nil {
		_ = s.cache.Del(context.Background(), cacheKey(alias))
	}

	return nil
}

func (s *Storage) AliasExists(alias string) (bool, error) {
	const op = "storage.postgres.AliasExists"

	var exists bool
	err := s.db.QueryRowContext(
		context.Background(),
		"SELECT EXISTS(SELECT 1 FROM urls WHERE alias = $1)",
		alias,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) RecordClick(alias string, userAgent string, clickedAt time.Time) error {
	const op = "storage.postgres.RecordClick"

	res, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO click_events (url_id, user_agent, clicked_at)
		 SELECT id, $2, $3 FROM urls WHERE alias = $1`,
		alias,
		userAgent,
		clickedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
	}

	return nil
}

func (s *Storage) GetAnalytics(alias string, groupBy string) (storage.Analytics, error) {
	const op = "storage.postgres.GetAnalytics"

	if groupBy == "" {
		groupBy = groupByRaw
	}

	if !isValidGroupBy(groupBy) {
		return storage.Analytics{}, fmt.Errorf("%s: %w", op, storage.ErrInvalidGroupBy)
	}

	var (
		url   string
		urlID int64
	)
	err := s.db.QueryRowContext(
		context.Background(),
		"SELECT id, url FROM urls WHERE alias = $1",
		alias,
	).Scan(&urlID, &url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Analytics{}, fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
		}
		return storage.Analytics{}, fmt.Errorf("%s: %w", op, err)
	}

	analytics := storage.Analytics{
		Alias:   alias,
		URL:     url,
		GroupBy: groupBy,
	}

	if err := s.db.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM click_events WHERE url_id = $1",
		urlID,
	).Scan(&analytics.TotalClicks); err != nil {
		return storage.Analytics{}, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := s.db.QueryContext(context.Background(), analyticsQuery(groupBy), urlID)
	if err != nil {
		return storage.Analytics{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var entry storage.AnalyticsEntry

		switch groupBy {
		case groupByRaw:
			if err := rows.Scan(&entry.UserAgent, &entry.ClickedAt); err != nil {
				return storage.Analytics{}, fmt.Errorf("%s: %w", op, err)
			}
		default:
			if err := rows.Scan(&entry.Bucket, &entry.Count); err != nil {
				return storage.Analytics{}, fmt.Errorf("%s: %w", op, err)
			}
		}

		analytics.Entries = append(analytics.Entries, entry)
	}

	if err := rows.Err(); err != nil {
		return storage.Analytics{}, fmt.Errorf("%s: %w", op, err)
	}

	return analytics, nil
}

func analyticsQuery(groupBy string) string {
	switch groupBy {
	case groupByDay:
		return `SELECT TO_CHAR(DATE_TRUNC('day', clicked_at), 'YYYY-MM-DD') AS bucket, COUNT(*)
			FROM click_events
			WHERE url_id = $1
			GROUP BY 1
			ORDER BY 1`
	case groupByMonth:
		return `SELECT TO_CHAR(DATE_TRUNC('month', clicked_at), 'YYYY-MM') AS bucket, COUNT(*)
			FROM click_events
			WHERE url_id = $1
			GROUP BY 1
			ORDER BY 1`
	case groupByAgent:
		return `SELECT user_agent AS bucket, COUNT(*)
			FROM click_events
			WHERE url_id = $1
			GROUP BY 1
			ORDER BY COUNT(*) DESC, 1`
	default:
		return `SELECT user_agent, clicked_at
			FROM click_events
			WHERE url_id = $1
			ORDER BY clicked_at DESC`
	}
}

func isValidGroupBy(groupBy string) bool {
	switch groupBy {
	case groupByRaw, groupByDay, groupByMonth, groupByAgent:
		return true
	default:
		return false
	}
}

func cacheKey(alias string) string {
	return cacheKeyPrefix + alias
}
