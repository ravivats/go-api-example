package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-api-example/model"
	"go-api-example/storage"

	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// MockStore provides a mock implementation of the storage.Store for testing.
type MockStore struct {
	CreateAccountFunc   func(ctx context.Context, acc model.Account) error
	GetAccountFunc      func(ctx context.Context, id int64) (*model.Account, error)
	ExecuteTransferFunc func(ctx context.Context, req model.TransactionRequest) error
}

func (m *MockStore) CreateAccount(ctx context.Context, acc model.Account) error {
	return m.CreateAccountFunc(ctx, acc)
}

func (m *MockStore) GetAccount(ctx context.Context, id int64) (*model.Account, error) {
	return m.GetAccountFunc(ctx, id)
}

func (m *MockStore) ExecuteTransfer(ctx context.Context, req model.TransactionRequest) error {
	return m.ExecuteTransferFunc(ctx, req)
}

func TestCreateAccountHandler(t *testing.T) {
	t.Run("success - new account", func(t *testing.T) {
		mockStore := &MockStore{
			GetAccountFunc: func(ctx context.Context, id int64) (*model.Account, error) {
				return nil, storage.ErrNotFound
			},
			CreateAccountFunc: func(ctx context.Context, acc model.Account) error {
				return nil
			},
		}
		handler := NewAccountHandler(mockStore)
		body := `{"account_id": 123, "initial_balance": "100.50"}`
		req := httptest.NewRequest("POST", "/accounts", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.CreateAccountHandler(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	})

	t.Run("success - existing account", func(t *testing.T) {
		mockStore := &MockStore{
			GetAccountFunc: func(ctx context.Context, id int64) (*model.Account, error) {
				return &model.Account{}, nil // Simulate account exists
			},
			CreateAccountFunc: func(ctx context.Context, acc model.Account) error {
				return nil
			},
		}
		handler := NewAccountHandler(mockStore)
		body := `{"account_id": 123, "initial_balance": "100.50"}`
		req := httptest.NewRequest("POST", "/accounts", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.CreateAccountHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := NewAccountHandler(&MockStore{})
		body := `{"account_id": 123, "initial_balance": "100.50"` // Malformed
		req := httptest.NewRequest("POST", "/accounts", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.CreateAccountHandler(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestGetAccountHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expectedAccount := &model.Account{
			AccountID: 123,
			Balance:   decimal.NewFromFloat(100.50),
		}
		mockStore := &MockStore{
			GetAccountFunc: func(ctx context.Context, id int64) (*model.Account, error) {
				assert.Equal(t, int64(123), id)
				return expectedAccount, nil
			},
		}
		handler := NewAccountHandler(mockStore)
		req := httptest.NewRequest("GET", "/accounts/123", nil)
		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/accounts/{account_id}", handler.GetAccountHandler)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resultAccount model.Account
		err := json.Unmarshal(rr.Body.Bytes(), &resultAccount)
		assert.NoError(t, err)
		assert.Equal(t, expectedAccount.AccountID, resultAccount.AccountID)
		assert.True(t, expectedAccount.Balance.Equal(resultAccount.Balance))
	})

	t.Run("not found", func(t *testing.T) {
		mockStore := &MockStore{
			GetAccountFunc: func(ctx context.Context, id int64) (*model.Account, error) {
				return nil, storage.ErrNotFound
			},
		}
		handler := NewAccountHandler(mockStore)
		req := httptest.NewRequest("GET", "/accounts/404", nil)
		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/accounts/{account_id}", handler.GetAccountHandler)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
