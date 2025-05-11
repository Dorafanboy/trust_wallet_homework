package restapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"trust_wallet_homework/internal/config"
	"trust_wallet_homework/internal/logger"
	"trust_wallet_homework/pkg/ethparser"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	httpServer *http.Server
	service    ethparser.Parser
	logger     logger.AppLogger
}

// NewServer creates a new instance of the REST API server.
func NewServer(service ethparser.Parser, appLogger logger.AppLogger, cfg *config.ServerConfig) (*Server, error) {
	if service == nil {
		return nil, errors.New("service cannot be nil for Server")
	}
	if appLogger == nil {
		return nil, errors.New("logger cannot be nil for Server")
	}
	if cfg == nil {
		return nil, errors.New("config cannot be nil for Server")
	}

	h, err := NewHTTPHandler(service, appLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize handler: %w", err)
	}

	smux := setupRouter(h, cfg.Port)

	server := &http.Server{
		Addr:              cfg.Port,
		Handler:           smux,
		ReadTimeout:       time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:       time.Duration(cfg.IdleTimeoutSeconds) * time.Second,
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeoutSeconds) * time.Second,
	}

	return &Server{
		httpServer: server,
		service:    service,
		logger:     appLogger,
	}, nil
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("HTTP server starting", "address", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("HTTP server ListenAndServe error", "error", err)
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", "error", err)
		return err
	}
	s.logger.Info("HTTP server stopped gracefully.")
	return nil
}

// setupRouter creates a new ServeMux and registers all API handlers.
func setupRouter(h *HTTPHandler, port string) *http.ServeMux {
	smux := http.NewServeMux()

	smux.HandleFunc("/current_block", h.HandleGetCurrentBlock)
	smux.HandleFunc("/subscribe", h.HandleSubscribe)
	smux.HandleFunc("/transactions/{address}", h.HandleGetTransactions)

	h.logger.Info("-------------------------------------")
	h.logger.Info("API Server starting", "address", port)
	h.logger.Info("Available Endpoints:")
	h.logger.Info("  GET  /current_block")
	h.logger.Info("  POST /subscribe       (Body: {'address':'0x...'})")
	h.logger.Info("  GET  /transactions/{address}")
	h.logger.Info("-------------------------------------")

	return smux
}
