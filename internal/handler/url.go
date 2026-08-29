package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	appMiddleware "github.com/Nikhil-O1O5/url-shortener/internal/middleware"
	"github.com/Nikhil-O1O5/url-shortener/internal/service"
)

type URLHandler struct {
	urlService  *service.URLService
	authService *service.AuthService
}

func NewURLHandler(urlService *service.URLService, authService *service.AuthService) *URLHandler {
	return &URLHandler{urlService: urlService, authService: authService}
}

func (h *URLHandler) RegisterRoutes(r chi.Router) {
	r.With(appMiddleware.OptionalAuth(h.authService)).Post("/api/v1/shorten", h.ShortenURL)
	r.Get("/{key}", h.RedirectURL)
}

type shortenRequest struct {
	LongURL     string `json:"long_url"     validate:"required,url"`
	CustomAlias string `json:"custom_alias" validate:"omitempty,min=3,max=16,alphanum"`
	ExpireAt    string `json:"expire_at"    validate:"omitempty"`
}

type shortenResponse struct {
	ShortURL  string `json:"short_url"`
	Hash      string `json:"hash"`
	ExpiresAt string `json:"expires_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *URLHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if err := validateStruct(req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	userID := appMiddleware.GetUserID(r.Context())

	if req.CustomAlias != "" && userID == nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "custom aliases require an account"})
		return
	}

	var expireAt *time.Time
	if req.ExpireAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpireAt)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "expire_at must be RFC3339 format"})
			return
		}
		expireAt = &t
	}

	result, err := h.urlService.ShortenURL(r.Context(), service.ShortenRequest{
		OriginalURL: req.LongURL,
		CustomAlias: req.CustomAlias,
		UserID:      userID,
		ExpireAt:    expireAt,
	})
	if err != nil {
		if errors.Is(err, service.ErrCustomAliasTaken) {
			writeJSON(w, http.StatusConflict, errorResponse{Error: "custom alias already taken"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to shorten URL"})
		return
	}

	writeJSON(w, http.StatusCreated, shortenResponse{
		ShortURL:  result.ShortURL,
		Hash:      result.Hash,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *URLHandler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "key")

	url, err := h.urlService.ResolveURL(r.Context(), hash)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to resolve URL"})
		return
	}
	if url == nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "short URL not found"})
		return
	}

	http.Redirect(w, r, url.OriginalURL, http.StatusFound)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
