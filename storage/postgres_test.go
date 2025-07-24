// storage/postgres_test.go
package storage

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"go-api-example/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testStore *PostgresStore

// TestMain sets up the test database container and runs the tests.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Create PostgreSQL container using the new API
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:14-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		log.Fatalf("could not start postgres container: %s", err)
	}

	// Clean up the container after the tests are finished
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Fatalf("could not terminate postgres container: %s", err)
		}
	}()

	// Get the connection string
	connString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("could not get connection string: %s", err)
	}

	// Connect to the test database
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("could not connect to test database: %s", err)
	}
	defer pool.Close()

	testStore = &PostgresStore{db: pool}
	if err := testStore.initSchema(ctx); err != nil {
		log.Fatalf("could not initialize schema: %s", err)
	}

	// Run the tests
	code := m.Run()
	os.Exit(code)
}

// truncateTables clears the accounts table between tests to ensure isolation.
func truncateTables(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := testStore.db.Exec(ctx, "TRUNCATE TABLE accounts RESTART IDENTITY")
	require.NoError(t, err, "failed to truncate tables")
}

func TestCreateAndGetAccount(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	t.Run("successfully create and retrieve an account", func(t *testing.T) {
		// Arrange
		initialBalance := decimal.NewFromFloat(100.50)
		acc := model.Account{
			AccountID: 1,
			Balance:   initialBalance,
		}

		// Act
		err := testStore.CreateAccount(ctx, acc)
		require.NoError(t, err)

		// Assert
		retrievedAcc, err := testStore.GetAccount(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, retrievedAcc)
		assert.Equal(t, int64(1), retrievedAcc.AccountID)
		assert.True(t, initialBalance.Equal(retrievedAcc.Balance), "expected balance %s, got %s", initialBalance.String(), retrievedAcc.Balance.String())
	})

	t.Run("creating an account is idempotent", func(t *testing.T) {
		// Arrange
		acc := model.Account{
			AccountID: 2,
			Balance:   decimal.NewFromInt(200),
		}
		err := testStore.CreateAccount(ctx, acc)
		require.NoError(t, err)

		// Act: Create the same account again
		err = testStore.CreateAccount(ctx, acc)

		// Assert: No error should occur
		require.NoError(t, err)
	})

	t.Run("create account with zero balance", func(t *testing.T) {
		// Arrange
		acc := model.Account{
			AccountID: 3,
			Balance:   decimal.Zero,
		}

		// Act
		err := testStore.CreateAccount(ctx, acc)
		require.NoError(t, err)

		// Assert
		retrievedAcc, err := testStore.GetAccount(ctx, 3)
		require.NoError(t, err)
		assert.True(t, decimal.Zero.Equal(retrievedAcc.Balance))
	})

	t.Run("create account with negative balance", func(t *testing.T) {
		// Arrange
		acc := model.Account{
			AccountID: 4,
			Balance:   decimal.NewFromInt(-100),
		}

		// Act
		err := testStore.CreateAccount(ctx, acc)
		require.NoError(t, err)

		// Assert
		retrievedAcc, err := testStore.GetAccount(ctx, 4)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(-100).Equal(retrievedAcc.Balance))
	})

	t.Run("create account with large balance", func(t *testing.T) {
		// Arrange - using a large but valid amount for NUMERIC(19, 5)
		largeBalance, _ := decimal.NewFromString("99999999999999.99999")
		acc := model.Account{
			AccountID: 5,
			Balance:   largeBalance,
		}

		// Act
		err := testStore.CreateAccount(ctx, acc)
		require.NoError(t, err)

		// Assert
		retrievedAcc, err := testStore.GetAccount(ctx, 5)
		require.NoError(t, err)
		assert.True(t, largeBalance.Equal(retrievedAcc.Balance))
	})
}

func TestGetAccount_NotFound(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	// Act
	_, err := testStore.GetAccount(ctx, 999)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetAccount_InvalidID(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	t.Run("negative account ID", func(t *testing.T) {
		_, err := testStore.GetAccount(ctx, -1)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("zero account ID", func(t *testing.T) {
		_, err := testStore.GetAccount(ctx, 0)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestExecuteTransfer_Success(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	// Arrange
	sourceInitialBalance := decimal.NewFromInt(1000)
	destInitialBalance := decimal.NewFromInt(500)
	transferAmount := decimal.NewFromInt(100)

	sourceAcc := model.Account{AccountID: 10, Balance: sourceInitialBalance}
	destAcc := model.Account{AccountID: 20, Balance: destInitialBalance}
	require.NoError(t, testStore.CreateAccount(ctx, sourceAcc))
	require.NoError(t, testStore.CreateAccount(ctx, destAcc))

	req := model.TransactionRequest{
		SourceAccountID:      10,
		DestinationAccountID: 20,
		Amount:               transferAmount,
	}

	// Act
	err := testStore.ExecuteTransfer(ctx, req)
	require.NoError(t, err)

	// Assert
	finalSourceAcc, err := testStore.GetAccount(ctx, 10)
	require.NoError(t, err)
	finalDestAcc, err := testStore.GetAccount(ctx, 20)
	require.NoError(t, err)

	expectedSourceBalance := sourceInitialBalance.Sub(transferAmount)
	expectedDestBalance := destInitialBalance.Add(transferAmount)

	assert.True(t, expectedSourceBalance.Equal(finalSourceAcc.Balance), "source balance mismatch")
	assert.True(t, expectedDestBalance.Equal(finalDestAcc.Balance), "destination balance mismatch")
}

func TestExecuteTransfer_FailureCases(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	// Arrange
	sourceAcc := model.Account{AccountID: 30, Balance: decimal.NewFromInt(50)}
	destAcc := model.Account{AccountID: 40, Balance: decimal.NewFromInt(100)}
	require.NoError(t, testStore.CreateAccount(ctx, sourceAcc))
	require.NoError(t, testStore.CreateAccount(ctx, destAcc))

	t.Run("insufficient funds", func(t *testing.T) {
		req := model.TransactionRequest{
			SourceAccountID: 30, DestinationAccountID: 40, Amount: decimal.NewFromInt(100),
		}
		err := testStore.ExecuteTransfer(ctx, req)
		assert.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("source account not found", func(t *testing.T) {
		req := model.TransactionRequest{
			SourceAccountID: 999, DestinationAccountID: 40, Amount: decimal.NewFromInt(10),
		}
		err := testStore.ExecuteTransfer(ctx, req)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("destination account not found", func(t *testing.T) {
		req := model.TransactionRequest{
			SourceAccountID: 30, DestinationAccountID: 999, Amount: decimal.NewFromInt(10),
		}
		err := testStore.ExecuteTransfer(ctx, req)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("both accounts not found", func(t *testing.T) {
		req := model.TransactionRequest{
			SourceAccountID: 888, DestinationAccountID: 999, Amount: decimal.NewFromInt(10),
		}
		err := testStore.ExecuteTransfer(ctx, req)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("zero transfer amount", func(t *testing.T) {
		req := model.TransactionRequest{
			SourceAccountID: 30, DestinationAccountID: 40, Amount: decimal.Zero,
		}
		err := testStore.ExecuteTransfer(ctx, req)
		require.NoError(t, err) // Zero transfers should be allowed

		// Verify balances remain unchanged
		sourceAcc, _ := testStore.GetAccount(ctx, 30)
		destAcc, _ := testStore.GetAccount(ctx, 40)
		assert.True(t, decimal.NewFromInt(50).Equal(sourceAcc.Balance))
		assert.True(t, decimal.NewFromInt(100).Equal(destAcc.Balance))
	})

	t.Run("negative transfer amount", func(t *testing.T) {
		req := model.TransactionRequest{
			SourceAccountID: 30, DestinationAccountID: 40, Amount: decimal.NewFromInt(-10),
		}
		err := testStore.ExecuteTransfer(ctx, req)
		// This should either fail or be handled as a reverse transfer
		// depending on business logic - currently it will succeed as a reverse transfer
		require.NoError(t, err)
	})

	t.Run("self transfer", func(t *testing.T) {
		// Get initial balance before self transfer
		initialAcc, err := testStore.GetAccount(ctx, 30)
		require.NoError(t, err)
		initialBalance := initialAcc.Balance

		req := model.TransactionRequest{
			SourceAccountID: 30, DestinationAccountID: 30, Amount: decimal.NewFromInt(10),
		}
		err = testStore.ExecuteTransfer(ctx, req)
		require.NoError(t, err) // Self transfers should work

		// Balance should remain unchanged for self transfers
		acc, _ := testStore.GetAccount(ctx, 30)
		assert.True(t, initialBalance.Equal(acc.Balance),
			"Self transfer should not change balance. Expected: %s, Got: %s",
			initialBalance.String(), acc.Balance.String())
	})

	t.Run("exact balance transfer", func(t *testing.T) {
		// Create a new account with exact amount we want to transfer
		exactAcc := model.Account{AccountID: 50, Balance: decimal.NewFromInt(25)}
		require.NoError(t, testStore.CreateAccount(ctx, exactAcc))

		req := model.TransactionRequest{
			SourceAccountID: 50, DestinationAccountID: 40, Amount: decimal.NewFromInt(25),
		}
		err := testStore.ExecuteTransfer(ctx, req)
		require.NoError(t, err)

		// Source should have zero balance
		sourceAcc, _ := testStore.GetAccount(ctx, 50)
		assert.True(t, decimal.Zero.Equal(sourceAcc.Balance))
	})
}

func TestExecuteTransfer_ConcurrentTransfers(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	// Arrange
	initialBalance := decimal.NewFromInt(10000)
	acc1 := model.Account{AccountID: 100, Balance: initialBalance}
	acc2 := model.Account{AccountID: 200, Balance: initialBalance}
	require.NoError(t, testStore.CreateAccount(ctx, acc1))
	require.NoError(t, testStore.CreateAccount(ctx, acc2))

	transferAmount := decimal.NewFromInt(10)
	numTransfers := 100 // Number of concurrent transfers in each direction

	var wg sync.WaitGroup
	errs := make(chan error, numTransfers*2)

	// Act: Concurrently transfer back and forth between two accounts
	for i := 0; i < numTransfers; i++ {
		wg.Add(2)
		go func() { // Acc 100 -> Acc 200
			defer wg.Done()
			req := model.TransactionRequest{SourceAccountID: 100, DestinationAccountID: 200, Amount: transferAmount}
			if err := testStore.ExecuteTransfer(context.Background(), req); err != nil {
				errs <- err
			}
		}()
		go func() { // Acc 200 -> Acc 100
			defer wg.Done()
			req := model.TransactionRequest{SourceAccountID: 200, DestinationAccountID: 100, Amount: transferAmount}
			if err := testStore.ExecuteTransfer(context.Background(), req); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	// Assert
	var errorList []error
	for err := range errs {
		errorList = append(errorList, err)
	}
	require.Empty(t, errorList, "concurrent transfers should not produce errors: %v", errorList)

	finalAcc1, err := testStore.GetAccount(ctx, 100)
	require.NoError(t, err)
	finalAcc2, err := testStore.GetAccount(ctx, 200)
	require.NoError(t, err)

	// Balances should be unchanged because for every debit, there was a corresponding credit.
	assert.True(t, initialBalance.Equal(finalAcc1.Balance), "final balance of account 1 is incorrect")
	assert.True(t, initialBalance.Equal(finalAcc2.Balance), "final balance of account 2 is incorrect")
}

func TestExecuteTransfer_DeadlockPrevention(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	// Arrange - create multiple accounts
	accounts := []model.Account{
		{AccountID: 1001, Balance: decimal.NewFromInt(1000)},
		{AccountID: 1002, Balance: decimal.NewFromInt(1000)},
		{AccountID: 1003, Balance: decimal.NewFromInt(1000)},
		{AccountID: 1004, Balance: decimal.NewFromInt(1000)},
	}

	for _, acc := range accounts {
		require.NoError(t, testStore.CreateAccount(ctx, acc))
	}

	var wg sync.WaitGroup
	errs := make(chan error, 100)
	transferAmount := decimal.NewFromInt(1)

	// Act: Create circular transfers that could cause deadlocks
	for i := 0; i < 25; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			req := model.TransactionRequest{SourceAccountID: 1001, DestinationAccountID: 1002, Amount: transferAmount}
			if err := testStore.ExecuteTransfer(context.Background(), req); err != nil {
				errs <- err
			}
		}()
		go func() {
			defer wg.Done()
			req := model.TransactionRequest{SourceAccountID: 1002, DestinationAccountID: 1003, Amount: transferAmount}
			if err := testStore.ExecuteTransfer(context.Background(), req); err != nil {
				errs <- err
			}
		}()
		go func() {
			defer wg.Done()
			req := model.TransactionRequest{SourceAccountID: 1003, DestinationAccountID: 1004, Amount: transferAmount}
			if err := testStore.ExecuteTransfer(context.Background(), req); err != nil {
				errs <- err
			}
		}()
		go func() {
			defer wg.Done()
			req := model.TransactionRequest{SourceAccountID: 1004, DestinationAccountID: 1001, Amount: transferAmount}
			if err := testStore.ExecuteTransfer(context.Background(), req); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	// Assert - no deadlocks should occur
	var errorList []error
	for err := range errs {
		errorList = append(errorList, err)
	}
	require.Empty(t, errorList, "circular transfers should not cause deadlocks: %v", errorList)
}

func TestExecuteTransfer_LargeAmounts(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	// Arrange - using amounts that fit within NUMERIC(19, 5) constraints
	largeBalance, _ := decimal.NewFromString("99999999999999.99999")
	transferAmount, _ := decimal.NewFromString("12345678901234.12345")

	sourceAcc := model.Account{AccountID: 60, Balance: largeBalance}
	destAcc := model.Account{AccountID: 70, Balance: decimal.Zero}
	require.NoError(t, testStore.CreateAccount(ctx, sourceAcc))
	require.NoError(t, testStore.CreateAccount(ctx, destAcc))

	req := model.TransactionRequest{
		SourceAccountID:      60,
		DestinationAccountID: 70,
		Amount:               transferAmount,
	}

	// Act
	err := testStore.ExecuteTransfer(ctx, req)
	require.NoError(t, err)

	// Assert
	finalSourceAcc, err := testStore.GetAccount(ctx, 60)
	require.NoError(t, err)
	finalDestAcc, err := testStore.GetAccount(ctx, 70)
	require.NoError(t, err)

	expectedSourceBalance := largeBalance.Sub(transferAmount)
	assert.True(t, expectedSourceBalance.Equal(finalSourceAcc.Balance), "large amount transfer failed")
	assert.True(t, transferAmount.Equal(finalDestAcc.Balance), "large amount transfer failed")
}

func TestExecuteTransfer_ContextCancellation(t *testing.T) {
	ctx := context.Background()
	truncateTables(t, ctx)

	// Arrange
	sourceAcc := model.Account{AccountID: 80, Balance: decimal.NewFromInt(1000)}
	destAcc := model.Account{AccountID: 90, Balance: decimal.NewFromInt(500)}
	require.NoError(t, testStore.CreateAccount(ctx, sourceAcc))
	require.NoError(t, testStore.CreateAccount(ctx, destAcc))

	// Create a context that gets cancelled immediately
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	req := model.TransactionRequest{
		SourceAccountID:      80,
		DestinationAccountID: 90,
		Amount:               decimal.NewFromInt(100),
	}

	// Act
	err := testStore.ExecuteTransfer(cancelCtx, req)

	// Assert - should fail due to context cancellation
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
