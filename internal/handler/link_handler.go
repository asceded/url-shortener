package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asceded/url-shortener/internal/service"
)

type LinkHandler struct {
	svc *service.LinkService
}

func NewLinkHandler(svc *service.LinkService) *LinkHandler {
	return &LinkHandler{svc: svc}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, createResponse{
		Code:        link.Code,
		ShortURL:    h.svc.BaseURL() + "/" + link.Code,
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Status(http.StatusNoContent)
}
