package model

import "time"

type Link struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type Click struct {
	ID        int64     `json:"id"`
	LinkID    int64     `json:"link_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	ClickedAt time.Time `json:"clicked_at"`
}

type LinkStats struct {
	Code        string     `json:"code"`
	TotalClicks int64      `json:"total_clicks"`
	LastClick   *time.Time `json:"last_click,omitempty"`
}
