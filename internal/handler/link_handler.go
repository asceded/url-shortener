package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asceded/url-shortener/internal/model"
	"github.com/asceded/url-shortener/internal/service"
)

type LinkService interface {
	CreateLink(ctx context.Context, url string) (*model.Link, error)
	Resolve(ctx context.Context, code, ip, userAgent string) (string, error)
	GetStats(ctx context.Context, code string) (*model.LinkStats, error)
	Delete(ctx context.Context, code string) error
}

type LinkHandler struct {
	svc     LinkService
	baseURL string
	log     *slog.Logger
}

func NewLinkHandler(svc LinkService, baseURL string, log *slog.Logger) *LinkHandler {
	return &LinkHandler{svc: svc, baseURL: baseURL, log: log}
}

type createRequest struct {
	URL string `json:"url" binding:"required"`
}

type createResponse struct {
	Code        string `json:"code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *LinkHandler) Create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	link, err := h.svc.CreateLink(c.Request.Context(), req.URL)
	if err != nil {
		if errors.Is(err, service.ErrInvalidURL) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
			return
		}
		h.log.Error("create link failed", "error", err, "url", req.URL)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, createResponse{
		Code:        link.Code,
		ShortURL:    h.baseURL + "/" + link.Code,
		OriginalURL: link.OriginalURL,
	})
}

func (h *LinkHandler) Redirect(c *gin.Context) {
	code := c.Param("code")

	originalURL, err := h.svc.Resolve(
		c.Request.Context(),
		code,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		h.log.Error("resolve link failed", "error", err, "code", code)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}

func (h *LinkHandler) Stats(c *gin.Context) {
	code := c.Param("code")

	stats, err := h.svc.GetStats(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		h.log.Error("get stats failed", "error", err, "code", code)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *LinkHandler) Delete(c *gin.Context) {
	code := c.Param("code")

	if err := h.svc.Delete(c.Request.Context(), code); err != nil {
		if errors.Is(err, service.ErrLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		h.log.Error("delete link failed", "error", err, "code", code)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Status(http.StatusNoContent)
}
