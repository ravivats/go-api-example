package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-api-example/model"
	"go-api-example/storage"

	"github.com/stretchr/testify/assert"
)

func TestCreateTransactionHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockStore := &MockStore{
			ExecuteTransferFunc: func(ctx context.Context, req model.TransactionRequest) error {
				return nil
			},
		}
		handler := NewTransactionHandler(mockStore)
		body := `{"source_account_id": 1, "destination_account_id": 2, "amount": "100"}`
		req := httptest.NewRequest("POST", "/transactions", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.CreateTransactionHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("insufficient funds", func(t *testing.T) {
		mockStore := &MockStore{
			ExecuteTransferFunc: func(ctx context.Context, req model.TransactionRequest) error {
				return storage.ErrInsufficientFunds
			},
		}
		handler := NewTransactionHandler(mockStore)
		body := `{"source_account_id": 1, "destination_account_id": 2, "amount": "1000"}`
		req := httptest.NewRequest("POST", "/transactions", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.CreateTransactionHandler(rr, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
		assert.Contains(t, rr.Body.String(), "Insufficient funds")
	})

	t.Run("account not found", func(t *testing.T) {
		mockStore := &MockStore{
			ExecuteTransferFunc: func(ctx context.Context, req model.TransactionRequest) error {
				return storage.ErrNotFound
			},
		}
		handler := NewTransactionHandler(mockStore)
		body := `{"source_account_id": 99, "destination_account_id": 2, "amount": "100"}`
		req := httptest.NewRequest("POST", "/transactions", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.CreateTransactionHandler(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "One or both accounts not found")
	})

	t.Run("same account", func(t *testing.T) {
		handler := NewTransactionHandler(&MockStore{})
		body := `{"source_account_id": 1, "destination_account_id": 1, "amount": "100"}`
		req := httptest.NewRequest("POST", "/transactions", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.CreateTransactionHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("negative amount", func(t *testing.T) {
		handler := NewTransactionHandler(&MockStore{})
		body := `{"source_account_id": 1, "destination_account_id": 2, "amount": "-100"}`
		req := httptest.NewRequest("POST", "/transactions", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.CreateTransactionHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
