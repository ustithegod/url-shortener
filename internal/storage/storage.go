package storage

import (
	"errors"
	"time"
)

var (
	ErrURLNotFound    = errors.New("url not found")
	ErrURLExists      = errors.New("url already exists")
	ErrInvalidGroupBy = errors.New("invalid group by")
)

type Analytics struct {
	Alias       string           `json:"alias"`
	URL         string           `json:"url"`
	GroupBy     string           `json:"group_by"`
	TotalClicks int64            `json:"total_clicks"`
	Entries     []AnalyticsEntry `json:"entries"`
}

type AnalyticsEntry struct {
	Bucket    string    `json:"bucket,omitempty"`
	Count     int64     `json:"count,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	ClickedAt time.Time `json:"clicked_at,omitempty"`
}
