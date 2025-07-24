package model

import "github.com/shopspring/decimal"

// Package model defines the data structures used in the banking application.

// Why we have used external package "github.com/shopspring/decimal"?
// The "github.com/shopspring/decimal" package is used for precise decimal arithmetic.
// It is particularly useful in financial applications where floating-point arithmetic can lead to inaccuracies.

// I have not used float64 knowlingly and intentionally because:
// 1. Floating-point numbers can introduce rounding errors, especially in financial calculations.
// 2. The decimal package provides arbitrary precision, which is crucial for handling monetary values accurately
// and avoiding issues with precision that can arise with standard floating-point types.

// You can't use float64 for financial balances because it cannot accurately represent most decimal values.
// This leads to small rounding errors that accumulate over time, causing data corruption and incorrect balances.
// This is a fundamental limitation of binary floating-point arithmetic.
// For example, a float64 cannot precisely store a value like 0.1.
// func main() {
//   // Adding 0.1 and 0.2 doesn't equal 0.3 with floats
// 	 var a, b float64 = 0.1, 0.2
// 	 fmt.Printf("0.1 + 0.2 = %.20f\n", a+b) // Outputs 0.30000000000000004
// }

// Hence we use the "github.com/shopspring/decimal" package instead of float64 to ensure that all monetary values are
// handled with the necessary precision and accuracy.

// Account represents a bank account with its ID and balance.
type Account struct {
	AccountID int64           `json:"account_id"`
	Balance   decimal.Decimal `json:"balance"`
}

// CreateAccountRequest defines the expected JSON body for creating an account.
type CreateAccountRequest struct {
	AccountID      int64           `json:"account_id"`
	InitialBalance decimal.Decimal `json:"initial_balance"`
}

// TransactionRequest defines the expected JSON body for submitting a transaction.
type TransactionRequest struct {
	SourceAccountID      int64           `json:"source_account_id"`
	DestinationAccountID int64           `json:"destination_account_id"`
	Amount               decimal.Decimal `json:"amount"`
}
