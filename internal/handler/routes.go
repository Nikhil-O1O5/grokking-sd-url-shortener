package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func RegisterRoutes(r *chi.Mux, urlHandler *URLHandler) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Post("/api/v1/shorten", urlHandler.ShortenURL)
	r.Get("/{key}", urlHandler.RedirectURL)
}
