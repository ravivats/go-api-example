package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"go-api-example/model"
	"go-api-example/storage"

	"github.com/gorilla/mux"
)

// AccountHandler holds dependencies for account-related handlers.
type AccountHandler struct {
	store storage.Store
}

// NewAccountHandler creates a new AccountHandler.
func NewAccountHandler(store storage.Store) *AccountHandler {
	return &AccountHandler{store: store}
}

// CreateAccountHandler handles the creation of a new bank account.
// It expects a JSON body with "account_id" and "initial_balance".
// This endpoint is idempotent.
//
// Method: POST
// Path: /accounts
// Success: 201 Created (if new) or 200 OK (if exists)
// Error: 400 Bad Request (for invalid JSON or validation failure)
// Error: 500 Internal Server Error (for database errors)
func (h *AccountHandler) CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req model.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.InitialBalance.IsNegative() {
		http.Error(w, "Initial balance cannot be negative", http.StatusBadRequest)
		return
	}

	// Check if account already exists to determine status code
	existingAcc, err := h.store.GetAccount(r.Context(), req.AccountID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		http.Error(w, "Could not check for existing account", http.StatusInternalServerError)
		return
	}

	acc := model.Account{
		AccountID: req.AccountID,
		Balance:   req.InitialBalance,
	}

	if err := h.store.CreateAccount(r.Context(), acc); err != nil {
		log.Printf("Error creating account: %v", err)
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	if existingAcc == nil {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// GetAccountHandler handles retrieving a specific account's balance.
// It expects an "account_id" as a URL path parameter.
//
// Method: GET
// Path: /accounts/{account_id}
// Success: 200 OK
// Error: 400 Bad Request (for invalid account ID format)
// Error: 404 Not Found (if account does not exist)
// Error: 500 Internal Server Error (for database errors)
func (h *AccountHandler) GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, ok := vars["account_id"]
	if !ok {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	accountID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid account ID format", http.StatusBadRequest)
		return
	}

	account, err := h.store.GetAccount(r.Context(), accountID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "Account not found", http.StatusNotFound)
		} else {
			log.Printf("Error getting account: %v", err)
			http.Error(w, "Failed to retrieve account", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(account); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Error writing JSON response: %v", err)
	}
}
