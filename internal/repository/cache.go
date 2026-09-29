package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type cachedLink struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type LinkCache struct {
	client *redis.Client
}

func NewLinkCache(client *redis.Client) *LinkCache {
	return &LinkCache{client: client}
}

func (c *LinkCache) Get(ctx context.Context, code string) (int64, string, error) {
	data, err := c.client.Get(ctx, cacheKey(code)).Bytes()
	if errors.Is(err, redis.Nil) {
		return 0, "", ErrCacheMiss
	}
	if err != nil {
		return 0, "", err
	}

	var cached cachedLink
	if err := json.Unmarshal(data, &cached); err != nil {
		return 0, "", err
	}
	return cached.ID, cached.URL, nil
}

func (c *LinkCache) Set(ctx context.Context, code string, id int64, url string, ttl time.Duration) error {
	data, err := json.Marshal(cachedLink{ID: id, URL: url})
	if err != nil {
		return err
	}
	return c.client.Set(ctx, cacheKey(code), data, ttl).Err()
}

func (c *LinkCache) Delete(ctx context.Context, code string) error {
	return c.client.Del(ctx, cacheKey(code)).Err()
}

func cacheKey(code string) string {
	return "link:" + code
}
