package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/uzzalhcse/proxypool/internal/config"
	"github.com/uzzalhcse/proxypool/internal/proxy"
	"github.com/uzzalhcse/proxypool/internal/redis"
)

type Handler struct {
	cfg     *config.Config
	manager *proxy.Manager
	redis   *redis.Client
}

func NewHandler(cfg *config.Config, manager *proxy.Manager, redis *redis.Client) *Handler {
	return &Handler{cfg: cfg, manager: manager, redis: redis}
}

func (h *Handler) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/proxies", h.authMiddleware(h.handleProxies))

	addr := fmt.Sprintf(":%d", h.cfg.APIPort)
	log.Printf("[API] Server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (h *Handler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer "+h.cfg.APIAuthToken {
			h.sendError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.sendJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) handleProxies(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.manager.GetAllProxies()
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.sendJSON(w, proxies)
}

func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
