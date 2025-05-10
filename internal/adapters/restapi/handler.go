package restapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/logger"
	"trust_wallet_homework/pkg/ethparser"
)

// HTTPHandler handles incoming HTTP requests for the parser API.
type HTTPHandler struct {
	parserService ethparser.Parser
	logger        logger.AppLogger
}

// NewHTTPHandler creates a new handler with the necessary service dependency.
func NewHTTPHandler(parserService ethparser.Parser, appLogger logger.AppLogger) (*HTTPHandler, error) {
	if parserService == nil {
		return nil, errors.New("parserService cannot be nil for HTTPHandler")
	}
	if appLogger == nil {
		return nil, errors.New("logger cannot be nil for HTTPHandler")
	}
	return &HTTPHandler{
		parserService: parserService,
		logger:        appLogger,
	}, nil
}

// HandleGetCurrentBlock handles requests to GET /current_block
func (h *HTTPHandler) HandleGetCurrentBlock(w http.ResponseWriter, r *http.Request) {
	requestLogger := h.logger.With("method", r.Method, "path", r.URL.Path)

	if r.Method != http.MethodGet {
		requestLogger.Warn("Method not allowed for GetCurrentBlock")
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", requestLogger)
		return
	}

	blockNum, err := h.parserService.GetCurrentBlock(r.Context())
	if err != nil {
		requestLogger.Error("Error getting current block", "error", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve current block", requestLogger)
		return
	}

	respondWithJSON(w, http.StatusOK, GetCurrentBlockResponse{BlockNumber: blockNum}, requestLogger)
}

// HandleSubscribe handles requests to POST /subscribe
func (h *HTTPHandler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	requestLogger := h.logger.With("method", r.Method, "path", r.URL.Path)

	if r.Method != http.MethodPost {
		requestLogger.Warn("Method not allowed for Subscribe")
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", requestLogger)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			requestLogger.Warn("Failed to close request body in HandleSubscribe", "error", err)
		}
	}()

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		requestLogger.Warn("Invalid request body for Subscribe", "error", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request body: "+err.Error(), requestLogger)
		return
	}

	if req.Address == "" {
		requestLogger.Warn("Empty address in Subscribe request")
		respondWithError(w, http.StatusBadRequest, "Address cannot be empty", requestLogger)
		return
	}

	requestLogger = requestLogger.With("address", req.Address)

	err := h.parserService.Subscribe(r.Context(), req.Address)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAddressFormat) {
			requestLogger.Warn("Subscribe validation failed", "error", err)
			respondWithError(w, http.StatusBadRequest, err.Error(), requestLogger)
		} else {
			requestLogger.Error("Error subscribing address", "error", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to subscribe address", requestLogger)
		}
		return
	}

	requestLogger.Info("Address subscribed successfully")
	respondWithJSON(w, http.StatusOK, SubscribeResponse{
		Success: true,
		Message: "Address subscribed successfully",
	}, requestLogger)
}

// HandleGetTransactions handles requests to GET /transactions/{address}
func (h *HTTPHandler) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	requestLogger := h.logger.With("method", r.Method, "path", r.URL.Path)

	if r.Method != http.MethodGet {
		requestLogger.Warn("Method not allowed for GetTransactions")
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", requestLogger)
		return
	}

	address := r.PathValue("address")
	requestLogger = requestLogger.With("address", address)

	if address == "" {
		requestLogger.Warn("Empty address in GetTransactions URL path")
		respondWithError(w, http.StatusBadRequest, "Address cannot be empty in URL path", requestLogger)
		return
	}

	txs, err := h.parserService.GetTransactions(r.Context(), address)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAddressFormat) {
			requestLogger.Warn("GetTransactions validation failed", "error", err)
			respondWithError(w, http.StatusBadRequest, err.Error(), requestLogger)
		} else {
			requestLogger.Error("Error getting transactions", "error", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to retrieve transactions", requestLogger)
		}
		return
	}

	requestLogger.Info("Successfully retrieved transactions", "count", len(txs))

	respondWithJSON(w, http.StatusOK, txs, requestLogger)
}

// respondWithError logs a warning and sends a JSON error response with the given code and message.
func respondWithError(w http.ResponseWriter, code int, message string, l logger.AppLogger) {
	if l == nil {
		serviceLogger := logger.NewSlogAdapter(slog.Default())
		serviceLogger.Warn("Responding with error (fallback logger)", "http_code", code, "message", message)
	} else {
		l.Warn("Responding with error", "http_code", code, "message", message)
	}
	respondWithJSON(w, code, ErrorResponse{Error: message}, l)
}

// respondWithJSON marshals the given payload into JSON and writes it to the response writer.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}, l logger.AppLogger) {
	if l == nil {
		serviceLogger := logger.NewSlogAdapter(slog.Default())
		serviceLogger.Error("!!! Critical: Error marshaling JSON response (fallback logger) !!!",
			"payload_type", fmt.Sprintf("%T", payload),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"Failed to marshal response"}`))
		return
	}

	response, err := json.Marshal(payload)
	if err != nil {
		l.Error("!!! Critical: Error marshaling JSON response !!!",
			"error", err.Error(),
			"payload_type", fmt.Sprintf("%T", payload),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"Failed to marshal response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)

	n, writeErr := w.Write(response)
	if writeErr != nil {
		l.Error("Error writing response body", "error", writeErr, "bytes_written", n)
	}
}
