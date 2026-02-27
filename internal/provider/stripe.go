package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StripeProvider wraps Stripe API operations.
type StripeProvider struct {
	secretKey      string
	webhookSecret  string
}

// NewStripeProvider creates a Stripe provider.
func NewStripeProvider(secretKey, webhookSecret string) *StripeProvider {
	return &StripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

// CheckoutSession represents a Stripe checkout session response.
type CheckoutSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// StripeWebhookEvent represents a parsed Stripe webhook event.
type StripeWebhookEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// CheckoutSessionData is the nested data.object from a checkout.session.completed event.
type CheckoutSessionData struct {
	ID                string `json:"id"`
	PaymentIntent     string `json:"payment_intent"`
	AmountTotal       int64  `json:"amount_total"`
	Currency          string `json:"currency"`
	Status            string `json:"status"`
	ClientReferenceID string `json:"client_reference_id"`
}

// CreateCheckoutSession creates a Stripe checkout session for a deposit.
// In production, this would call the Stripe API. For now, returns a structured response.
func (s *StripeProvider) CreateCheckoutSession(amountCents int64, currency, playerID, successURL, cancelURL string) (*CheckoutSession, error) {
	if s.secretKey == "" {
		return nil, fmt.Errorf("stripe secret key not configured")
	}

	// Build form data for Stripe API
	form := fmt.Sprintf(
		"mode=payment&line_items[0][price_data][currency]=%s&line_items[0][price_data][unit_amount]=%d&line_items[0][price_data][product_data][name]=Deposit&line_items[0][quantity]=1&client_reference_id=%s&success_url=%s&cancel_url=%s",
		strings.ToLower(currency), amountCents, playerID, successURL, cancelURL,
	)

	req, err := http.NewRequest("POST", "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(form))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stripe error (status %d): %s", resp.StatusCode, string(body))
	}

	var session CheckoutSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("decode stripe response: %w", err)
	}
	return &session, nil
}

// VerifyWebhookSignature verifies a Stripe webhook signature.
// Returns the parsed event if valid.
func (s *StripeProvider) VerifyWebhookSignature(payload []byte, sigHeader string) (*StripeWebhookEvent, error) {
	if s.webhookSecret == "" {
		return nil, fmt.Errorf("stripe webhook secret not configured")
	}

	// Parse Stripe-Signature header: t=timestamp,v1=signature
	parts := strings.Split(sigHeader, ",")
	var timestamp string
	var signatures []string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}

	if timestamp == "" || len(signatures) == 0 {
		return nil, fmt.Errorf("invalid signature header format")
	}

	// Check timestamp tolerance (5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	if time.Now().Unix()-ts > 300 {
		return nil, fmt.Errorf("webhook timestamp too old")
	}

	// Compute expected signature
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	// Compare with provided signatures
	valid := false
	for _, sig := range signatures {
		if hmac.Equal([]byte(expected), []byte(sig)) {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	var event StripeWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("decode webhook event: %w", err)
	}
	return &event, nil
}

// ParseCheckoutSessionData extracts checkout session data from a webhook event.
func ParseCheckoutSessionData(data json.RawMessage) (*CheckoutSessionData, error) {
	var wrapper struct {
		Object CheckoutSessionData `json:"object"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parse checkout session data: %w", err)
	}
	return &wrapper.Object, nil
}
