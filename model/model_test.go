package model

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAccountJSON tests JSON marshaling and unmarshaling for the Account struct.
func TestAccountJSON(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		// Arrange
		originalAccount := Account{
			AccountID: 123,
			Balance:   decimal.NewFromFloat(1500.75),
		}
		expectedJSON := `{"account_id":123,"balance":"1500.75"}`

		// Act: Marshal
		jsonData, err := json.Marshal(originalAccount)
		require.NoError(t, err)
		assert.JSONEq(t, expectedJSON, string(jsonData))

		// Act: Unmarshal
		var unmarshaledAccount Account
		err = json.Unmarshal(jsonData, &unmarshaledAccount)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, originalAccount.AccountID, unmarshaledAccount.AccountID)
		assert.True(t, originalAccount.Balance.Equal(unmarshaledAccount.Balance))
	})

	t.Run("unmarshal with invalid balance format", func(t *testing.T) {
		// Arrange
		invalidJSON := `{"account_id":123,"balance":"not-a-number"}`

		// Act
		var acc Account
		err := json.Unmarshal([]byte(invalidJSON), &acc)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't convert not-a-number to decimal")
	})
}

// TestCreateAccountRequestJSON tests JSON marshaling and unmarshaling for the CreateAccountRequest struct.
func TestCreateAccountRequestJSON(t *testing.T) {
	t.Run("successful marshal and unmarshal with precision", func(t *testing.T) {
		// Arrange
		// Use New to preserve trailing zeros for exact representation
		originalReq := CreateAccountRequest{
			AccountID:      456,
			InitialBalance: decimal.New(200001, -2), // Represents 2000.00
		}
		expectedJSON := `{"account_id":456,"initial_balance":"2000.01"}`

		// Act: Marshal
		jsonData, err := json.Marshal(originalReq)
		require.NoError(t, err)
		assert.JSONEq(t, expectedJSON, string(jsonData))

		// Act: Unmarshal
		var unmarshaledReq CreateAccountRequest
		err = json.Unmarshal(jsonData, &unmarshaledReq)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, originalReq.AccountID, unmarshaledReq.AccountID)
		assert.True(t, originalReq.InitialBalance.Equal(unmarshaledReq.InitialBalance))
	})
}

// TestTransactionRequestJSON tests JSON marshaling and unmarshaling for the TransactionRequest struct.
func TestTransactionRequestJSON(t *testing.T) {
	t.Run("successful marshal and unmarshal", func(t *testing.T) {
		// Arrange
		originalReq := TransactionRequest{
			SourceAccountID:      101,
			DestinationAccountID: 102,
			Amount:               decimal.NewFromFloat(250.25),
		}
		expectedJSON := `{"source_account_id":101,"destination_account_id":102,"amount":"250.25"}`

		// Act: Marshal
		jsonData, err := json.Marshal(originalReq)
		require.NoError(t, err)
		assert.JSONEq(t, expectedJSON, string(jsonData))

		// Act: Unmarshal
		var unmarshaledReq TransactionRequest
		err = json.Unmarshal(jsonData, &unmarshaledReq)
		require.NoError(t, err)

		// Assert
		assert.Equal(t, originalReq.SourceAccountID, unmarshaledReq.SourceAccountID)
		assert.Equal(t, originalReq.DestinationAccountID, unmarshaledReq.DestinationAccountID)
		assert.True(t, originalReq.Amount.Equal(unmarshaledReq.Amount))
	})

	t.Run("unmarshal with invalid amount type", func(t *testing.T) {
		// Arrange
		invalidJSON := `{"source_account_id":101,"destination_account_id":102,"amount":true}` // amount is a boolean

		// Act
		var req TransactionRequest
		err := json.Unmarshal([]byte(invalidJSON), &req)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't convert true to decimal")
	})
}
