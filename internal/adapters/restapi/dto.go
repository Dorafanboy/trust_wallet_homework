// Package restapi implements the RESTful API layer, including DTOs and handlers.
package restapi

// SubscribeRequest defines the expected JSON body for the POST /subscribe endpoint.
type SubscribeRequest struct {
	Address string `json:"address"`
}

// ErrorResponse defines a standard structure for JSON error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// GetCurrentBlockResponse defines the structure for the GET /current_block endpoint.
type GetCurrentBlockResponse struct {
	BlockNumber int64 `json:"current_block"`
}

// SubscribeResponse defines the structure for the POST /subscribe endpoint response (on success).
type SubscribeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
