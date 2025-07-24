package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"go-api-example/model"
	"go-api-example/storage"
)

// TransactionHandler holds dependencies for transaction-related handlers.
type TransactionHandler struct {
	store storage.Store
}

// NewTransactionHandler creates a new TransactionHandler.
func NewTransactionHandler(store storage.Store) *TransactionHandler {
	return &TransactionHandler{store: store}
}

// CreateTransactionHandler handles the submission of a new financial transaction.
// It processes the transfer atomically and ensures data consistency.
//
// Method: POST
// Path: /transactions
// Success: 200 OK
// Error: 400 Bad Request (for invalid JSON or validation failure)
// Error: 422 Unprocessable Entity (for business logic errors like insufficient funds)
// Error: 500 Internal Server Error (for database errors)
func (h *TransactionHandler) CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var req model.TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if req.SourceAccountID == req.DestinationAccountID {
		http.Error(w, "Source and destination accounts cannot be the same", http.StatusBadRequest)
		return
	}
	if !req.Amount.IsPositive() {
		http.Error(w, "Transaction amount must be positive", http.StatusBadRequest)
		return
	}

	err := h.store.ExecuteTransfer(r.Context(), req)
	if err != nil {
		log.Printf("Error executing transfer: %v", err)
		switch {
		case errors.Is(err, storage.ErrInsufficientFunds):
			http.Error(w, "Insufficient funds", http.StatusUnprocessableEntity)
		case errors.Is(err, storage.ErrNotFound):
			http.Error(w, "One or both accounts not found", http.StatusNotFound)
		default:
			http.Error(w, "Failed to process transaction", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
