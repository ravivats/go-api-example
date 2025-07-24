// storage/postgres.go

package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-api-example/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	// "github.com/shopspring/decimal"
)

// Custom errors for the storage layer.
var (
	ErrNotFound          = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// Store defines the interface for database operations.
type Store interface {
	CreateAccount(ctx context.Context, acc model.Account) error
	GetAccount(ctx context.Context, id int64) (*model.Account, error)
	ExecuteTransfer(ctx context.Context, req model.TransactionRequest) error
}

// PostgresStore implements the Store interface for PostgreSQL.
type PostgresStore struct {
	db *pgxpool.Pool
}

// NewPostgresStore creates a new PostgresStore, connects to the database, and initializes the schema.
func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	var pool *pgxpool.Pool
	var err error

	// Retry connecting to the database for a few seconds
	for i := 0; i < 5; i++ {
		pool, err = pgxpool.New(ctx, connString)
		if err == nil {
			if err := pool.Ping(ctx); err == nil {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("could not connect to database after retries: %w", err)
	}

	store := &PostgresStore{db: pool}
	if err := store.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("could not initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables if they don't exist.
func (s *PostgresStore) initSchema(ctx context.Context) error {
	query := `
    CREATE TABLE IF NOT EXISTS accounts (
        account_id BIGINT PRIMARY KEY,
        balance NUMERIC(19, 5) NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );`
	_, err := s.db.Exec(ctx, query)
	return err
}

// CreateAccount creates a new account in the database.
// CreateAccount function is idempotent: if an account with the same ID already exists, it does nothing and returns nil.
func (s *PostgresStore) CreateAccount(ctx context.Context, acc model.Account) error {
	query := `
		INSERT INTO accounts (account_id, balance) 
		VALUES ($1, $2) 
		ON CONFLICT (account_id) DO NOTHING`
	_, err := s.db.Exec(ctx, query, acc.AccountID, acc.Balance)
	return err
}

// GetAccount retrieves a single account by its ID.
func (s *PostgresStore) GetAccount(ctx context.Context, id int64) (*model.Account, error) {
	acc := &model.Account{AccountID: id}
	query := "SELECT balance FROM accounts WHERE account_id = $1"
	err := s.db.QueryRow(ctx, query, id).Scan(&acc.Balance)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return acc, nil
}

// ExecuteTransfer performs a financial transfer between two accounts within a database transaction.
// It locks the rows for the source and destination accounts to prevent race conditions.
func (s *PostgresStore) ExecuteTransfer(ctx context.Context, req model.TransactionRequest) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback is a no-op if the transaction has been committed.

	// Lock accounts in a consistent order (by ID) to prevent deadlocks.
	var sourceAccount model.Account
	var foundSource, foundDest bool

	query := `
        SELECT account_id, balance FROM accounts 
        WHERE account_id = $1 OR account_id = $2 
        ORDER BY account_id FOR UPDATE`

	rows, err := tx.Query(ctx, query, req.SourceAccountID, req.DestinationAccountID)
	if err != nil {
		return fmt.Errorf("could not query accounts for update: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var acc model.Account
		if err := rows.Scan(&acc.AccountID, &acc.Balance); err != nil {
			return fmt.Errorf("could not scan account row: %w", err)
		}
		if acc.AccountID == req.SourceAccountID {
			sourceAccount = acc
			foundSource = true
		}
		if acc.AccountID == req.DestinationAccountID {
			// destAccount = acc
			foundDest = true
		}
	}

	if !foundSource || !foundDest {
		return ErrNotFound
	}

	if sourceAccount.Balance.LessThan(req.Amount) {
		return ErrInsufficientFunds
	}

	// Debit source account
	updateQuery := "UPDATE accounts SET balance = balance - $1 WHERE account_id = $2"
	if _, err := tx.Exec(ctx, updateQuery, req.Amount, req.SourceAccountID); err != nil {
		return fmt.Errorf("could not debit source account: %w", err)
	}

	// Credit destination account
	updateQuery = "UPDATE accounts SET balance = balance + $1 WHERE account_id = $2"
	if _, err := tx.Exec(ctx, updateQuery, req.Amount, req.DestinationAccountID); err != nil {
		return fmt.Errorf("could not credit destination account: %w", err)
	}

	return tx.Commit(ctx)
}
