// Package payment processes payment transactions.
package payment

import (
	"errors"
	"fmt"
)

// ErrInsufficientFunds is returned when an account lacks the required balance.
var ErrInsufficientFunds = errors.New("insufficient funds")

// ErrInvalidAmount is returned when an amount is zero or negative.
var ErrInvalidAmount = errors.New("invalid amount: must be positive")

// Account holds a named balance in cents.
type Account struct {
	ID      string
	Balance int64 // in cents
}

// Charge deducts amount cents from the account.
// Returns ErrInvalidAmount when amount ≤ 0.
// Returns ErrInsufficientFunds when the account balance is too low.
func (a *Account) Charge(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.Balance < amount {
		return fmt.Errorf("%w: have %d, need %d", ErrInsufficientFunds, a.Balance, amount)
	}
	a.Balance -= amount
	return nil
}

// Refund credits amount cents back to the account.
func (a *Account) Refund(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	a.Balance += amount
	return nil
}

// Transfer moves amount cents from src to dst atomically.
func Transfer(src, dst *Account, amount int64) error {
	if err := src.Charge(amount); err != nil {
		return fmt.Errorf("debit source: %w", err)
	}
	if err := dst.Refund(amount); err != nil {
		// Roll back the charge if credit fails (should not happen in practice).
		_ = src.Refund(amount)
		return fmt.Errorf("credit destination: %w", err)
	}
	return nil
}
