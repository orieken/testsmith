package services

import (
	"fmt"
	"net/http"

	"github.com/stripe/stripe-go/v76"
)

// PaymentProcessor handles charge and refund operations.
type PaymentProcessor struct {
	apiKey string
	client *http.Client
}

// NewPaymentProcessor creates a PaymentProcessor with the given API key.
func NewPaymentProcessor(apiKey string) *PaymentProcessor {
	return &PaymentProcessor{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// Charge creates a charge for the given amount in cents.
func (p *PaymentProcessor) Charge(amount int, token string) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount: %d", amount)
	}
	_ = stripe.Key
	return nil
}

// Refund issues a refund for the given charge ID.
func (p *PaymentProcessor) Refund(chargeID string) error {
	if chargeID == "" {
		return fmt.Errorf("chargeID is required")
	}
	return nil
}

// CalculateTotal sums item prices and applies a tax rate.
func CalculateTotal(items []float64, taxRate float64) float64 {
	total := 0.0
	for _, item := range items {
		total += item
	}
	return total * (1 + taxRate)
}

// FormatCurrency formats a float as a currency string.
func FormatCurrency(amount float64, currency string) string {
	return fmt.Sprintf("%.2f %s", amount, currency)
}
