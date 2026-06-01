package httpapi

import (
	_ "embed"
	"net/http"
)

//go:embed swagger/index.html
var swaggerUIHTML []byte

//go:embed swagger/openapi.yaml
var openAPISpec []byte

func (h *Handler) redirectToSwaggerUI(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger/", http.StatusPermanentRedirect)
}

func (h *Handler) swaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(swaggerUIHTML)
}

func (h *Handler) swaggerSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(openAPISpec)
}
