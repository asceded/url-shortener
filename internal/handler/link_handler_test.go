package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/asceded/url-shortener/internal/model"
	"github.com/asceded/url-shortener/internal/service"
)

type mockService struct {
	createLinkFn func(ctx context.Context, url string) (*model.Link, error)
	resolveFn    func(ctx context.Context, code, ip, ua string) (string, error)
	getStatsFn   func(ctx context.Context, code string) (*model.LinkStats, error)
	deleteFn     func(ctx context.Context, code string) error
}

func (m *mockService) CreateLink(ctx context.Context, url string) (*model.Link, error) {
	return m.createLinkFn(ctx, url)
}

func (m *mockService) Resolve(ctx context.Context, code, ip, ua string) (string, error) {
	return m.resolveFn(ctx, code, ip, ua)
}

func (m *mockService) GetStats(ctx context.Context, code string) (*model.LinkStats, error) {
	return m.getStatsFn(ctx, code)
}

func (m *mockService) Delete(ctx context.Context, code string) error {
	return m.deleteFn(ctx, code)
}

func setupRouter(svc LinkService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewLinkHandler(svc, "http://test.local", slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := gin.New()
	r.POST("/api/links", h.Create)
	r.GET("/api/links/:code/stats", h.Stats)
	r.DELETE("/api/links/:code", h.Delete)
	r.GET("/:code", h.Redirect)
	return r
}

func TestCreateLink_Success(t *testing.T) {
	svc := &mockService{
		createLinkFn: func(ctx context.Context, url string) (*model.Link, error) {
			return &model.Link{ID: 1, Code: "abc1234", OriginalURL: url}, nil
		},
	}
	r := setupRouter(svc)

	body, _ := json.Marshal(map[string]string{"url": "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp createResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != "abc1234" {
		t.Errorf("want code abc1234, got %s", resp.Code)
	}
	if resp.ShortURL != "http://test.local/abc1234" {
		t.Errorf("unexpected short_url: %s", resp.ShortURL)
	}
}

func TestCreateLink_InvalidURL(t *testing.T) {
	svc := &mockService{
		createLinkFn: func(ctx context.Context, url string) (*model.Link, error) {
			return nil, service.ErrInvalidURL
		},
	}
	r := setupRouter(svc)

	body, _ := json.Marshal(map[string]string{"url": "bad"})
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCreateLink_MissingURL(t *testing.T) {
	svc := &mockService{}
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestRedirect_Success(t *testing.T) {
	svc := &mockService{
		resolveFn: func(ctx context.Context, code, ip, ua string) (string, error) {
			return "https://example.com/target", nil
		},
	}
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/abc1234", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/target" {
		t.Errorf("unexpected Location: %s", loc)
	}
}

func TestRedirect_NotFound(t *testing.T) {
	svc := &mockService{
		resolveFn: func(ctx context.Context, code, ip, ua string) (string, error) {
			return "", service.ErrLinkNotFound
		},
	}
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := &mockService{
		deleteFn: func(ctx context.Context, code string) error {
			return service.ErrLinkNotFound
		},
	}
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/links/nope", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestDelete_Success(t *testing.T) {
	svc := &mockService{
		deleteFn: func(ctx context.Context, code string) error {
			return nil
		},
	}
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/links/abc1234", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
}

func TestStats_InternalError(t *testing.T) {
	svc := &mockService{
		getStatsFn: func(ctx context.Context, code string) (*model.LinkStats, error) {
			return nil, errors.New("db exploded")
		},
	}
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links/abc/stats", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
