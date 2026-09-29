package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"time"

	"github.com/asceded/url-shortener/internal/model"
	"github.com/asceded/url-shortener/internal/repository"
	"github.com/asceded/url-shortener/internal/worker"
)

const (
	codeLength   = 7
	cacheTTL     = 24 * time.Hour
	maxAttempts  = 5
	codeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

var (
	ErrInvalidURL    = errors.New("invalid url")
	ErrCodeGenFailed = errors.New("failed to generate unique code")
	ErrLinkNotFound  = repository.ErrLinkNotFound
)

type LinkService struct {
	linkRepo  *repository.LinkRepository
	clickRepo *repository.ClickRepository
	cache     *repository.LinkCache
	worker    *worker.ClickWorker
	baseURL   string
	log       *slog.Logger
}

func NewLinkService(
	linkRepo *repository.LinkRepository,
	clickRepo *repository.ClickRepository,
	cache *repository.LinkCache,
	worker *worker.ClickWorker,
	baseURL string,
	log *slog.Logger,
) *LinkService {
	return &LinkService{
		linkRepo:  linkRepo,
		clickRepo: clickRepo,
		cache:     cache,
		worker:    worker,
		baseURL:   baseURL,
		log:       log,
	}
}

func (s *LinkService) BaseURL() string {
	return s.baseURL
}

func (s *LinkService) CreateLink(ctx context.Context, originalURL string) (*model.Link, error) {
	parsed, err := url.ParseRequestURI(originalURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, ErrInvalidURL
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		code, err := generateCode(codeLength)
		if err != nil {
			return nil, fmt.Errorf("generate code: %w", err)
		}

		link, err := s.linkRepo.Create(ctx, code, originalURL)
		if err == nil {
			return link, nil
		}
		if !repository.IsUniqueViolation(err) {
			return nil, err
		}
	}
	return nil, ErrCodeGenFailed
}

func (s *LinkService) Resolve(ctx context.Context, code, ip, userAgent string) (string, error) {
	linkID, cachedURL, err := s.cache.Get(ctx, code)
	if err == nil {
		s.worker.Enqueue(model.Click{
			LinkID:    linkID,
			IP:        ip,
			UserAgent: userAgent,
		})
		return cachedURL, nil
	}
	if !errors.Is(err, repository.ErrCacheMiss) {
		s.log.Warn("cache get failed", "error", err, "code", code)
	}

	link, err := s.linkRepo.GetByCode(ctx, code)
	if err != nil {
		return "", err
	}

	if err := s.cache.Set(ctx, code, link.ID, link.OriginalURL, cacheTTL); err != nil {
		s.log.Warn("cache set failed", "error", err, "code", code)
	}

	s.worker.Enqueue(model.Click{
		LinkID:    link.ID,
		IP:        ip,
		UserAgent: userAgent,
	})
	return link.OriginalURL, nil
}

func (s *LinkService) GetStats(ctx context.Context, code string) (*model.LinkStats, error) {
	return s.clickRepo.GetStats(ctx, code)
}

func (s *LinkService) Delete(ctx context.Context, code string) error {
	if err := s.linkRepo.Delete(ctx, code); err != nil {
		return err
	}
	if err := s.cache.Delete(ctx, code); err != nil {
		s.log.Warn("cache delete failed", "error", err, "code", code)
	}
	return nil
}

func generateCode(length int) (string, error) {
	max := big.NewInt(int64(len(codeAlphabet)))
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		buf[i] = codeAlphabet[n.Int64()]
	}
	return string(buf), nil
}
