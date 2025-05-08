package restapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"trust_wallet_homework/internal/config"
	"trust_wallet_homework/pkg/ethparser"
)

var (
	// ErrServiceIsNil indicates that a nil parser service was provided during server initialization.
	ErrServiceIsNil = errors.New("service cannot be nil for Server")

	// ErrLoggerIsNil indicates that a nil logger was provided during server initialization.
	ErrLoggerIsNil = errors.New("logger cannot be nil for Server")

	// ErrConfigIsNil indicates that a nil config was provided during server initialization.
	ErrConfigIsNil = errors.New("config cannot be nil for Server")

	// ErrHandlerInitFailed indicates that the HTTP handler dependency failed to initialize.
	ErrHandlerInitFailed = errors.New("failed to initialize handler")
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	httpServer *http.Server
	service    ethparser.Parser
	logger     *slog.Logger
	cfg        *config.Config
}

// NewServer creates a new instance of the REST API server.
func NewServer(service ethparser.Parser, logger *slog.Logger, cfg *config.Config) (*Server, error) {
	if service == nil {
		return nil, ErrServiceIsNil
	}
	if logger == nil {
		return nil, ErrLoggerIsNil
	}
	if cfg == nil {
		return nil, ErrConfigIsNil
	}

	h, err := NewHTTPHandler(service, logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHandlerInitFailed, err)
	}

	smux := setupRouter(h)

	server := &http.Server{
		Addr:              cfg.Server.Port,
		Handler:           smux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	return &Server{
		httpServer: server,
		service:    service,
		logger:     logger,
		cfg:        cfg,
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
func setupRouter(h *HTTPHandler) *http.ServeMux {
	smux := http.NewServeMux()

	smux.HandleFunc("/current_block", h.HandleGetCurrentBlock)
	smux.HandleFunc("/subscribe", h.HandleSubscribe)
	smux.HandleFunc("/transactions/{address}", h.HandleGetTransactions)

	return smux
}
