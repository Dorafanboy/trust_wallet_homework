package restapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/pkg/ethparser"
)

// HTTPHandler handles incoming HTTP requests for the parser API.
type HTTPHandler struct {
	parserService ethparser.Parser
	logger        *slog.Logger
}

// NewHTTPHandler creates a new handler with the necessary service dependency.
func NewHTTPHandler(parserService ethparser.Parser, logger *slog.Logger) (*HTTPHandler, error) {
	if parserService == nil {
		return nil, errors.New("parserService cannot be nil for HTTPHandler")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil for HTTPHandler")
	}
	return &HTTPHandler{
		parserService: parserService,
		logger:        logger,
	}, nil
}

// HandleGetCurrentBlock handles requests to GET /current_block
func (h *HTTPHandler) HandleGetCurrentBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("Method not allowed for GetCurrentBlock", "method", r.Method, "path", r.URL.Path)
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", h.logger)
		return
	}

	blockNum, err := h.parserService.GetCurrentBlock(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Error getting current block", "error", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve current block", h.logger)
		return
	}

	respondWithJSON(w, http.StatusOK, GetCurrentBlockResponse{BlockNumber: blockNum}, h.logger)
}

// HandleSubscribe handles requests to POST /subscribe
func (h *HTTPHandler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("Method not allowed for Subscribe", "method", r.Method, "path", r.URL.Path)
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", h.logger)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			h.logger.Warn("Failed to close request body in HandleSubscribe", "error", err)
		}
	}()

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(r.Context(), "Invalid request body for Subscribe", "error", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request body: "+err.Error(), h.logger)
		return
	}

	if req.Address == "" {
		h.logger.WarnContext(r.Context(), "Empty address in Subscribe request")
		respondWithError(w, http.StatusBadRequest, "Address cannot be empty", h.logger)
		return
	}

	err := h.parserService.Subscribe(r.Context(), req.Address)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAddressFormat) {
			h.logger.WarnContext(r.Context(), "Subscribe validation failed", "address", req.Address, "error", err)
			respondWithError(w, http.StatusBadRequest, err.Error(), h.logger)
		} else {
			h.logger.ErrorContext(r.Context(), "Error subscribing address", "address", req.Address, "error", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to subscribe address", h.logger)
		}
		return
	}

	h.logger.InfoContext(r.Context(), "Address subscribed successfully", "address", req.Address)
	respondWithJSON(w, http.StatusOK, SubscribeResponse{
		Success: true,
		Message: "Address subscribed successfully",
	}, h.logger)
}

// HandleGetTransactions handles requests to GET /transactions/{address}
func (h *HTTPHandler) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("Method not allowed for GetTransactions", "method", r.Method, "path", r.URL.Path)
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", h.logger)
		return
	}

	address := r.PathValue("address")

	if address == "" {
		h.logger.WarnContext(r.Context(), "Empty address in GetTransactions URL path")
		respondWithError(w, http.StatusBadRequest, "Address cannot be empty in URL path", h.logger)
		return
	}

	txs, err := h.parserService.GetTransactions(r.Context(), address)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAddressFormat) {
			h.logger.WarnContext(r.Context(), "GetTransactions validation failed", "address", address, "error", err)
			respondWithError(w, http.StatusBadRequest, err.Error(), h.logger)
		} else {
			h.logger.ErrorContext(r.Context(), "Error getting transactions", "address", address, "error", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to retrieve transactions", h.logger)
		}
		return
	}

	h.logger.InfoContext(r.Context(), "Successfully retrieved transactions", "address", address, "count", len(txs))

	respondWithJSON(w, http.StatusOK, txs, h.logger)
}

// respondWithError logs a warning and sends a JSON error response with the given code and message.
func respondWithError(w http.ResponseWriter, code int, message string, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	logger.Warn("Responding with error", "http_code", code, "message", message)
	respondWithJSON(w, code, ErrorResponse{Error: message}, logger)
}

// respondWithJSON marshals the given payload into JSON and writes it to the response writer.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}

	response, err := json.Marshal(payload)
	if err != nil {
		logger.Error("!!! Critical: Error marshaling JSON response !!!",
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
		logger.Error("Error writing response body", "error", writeErr, "bytes_written", n)
	}
}
