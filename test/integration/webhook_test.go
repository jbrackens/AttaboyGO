//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/attaboy/platform/test/integration/testutil"
	"github.com/stretchr/testify/assert"
)

// ─── Stripe Webhook Tests (5) ─────────────────────────────────────────────

func TestStripeWebhook_MissingSignatureHeader(t *testing.T) {
	env := testutil.NewTestEnv(t)

	payload := []byte(`{"type":"checkout.session.completed","data":{}}`)
	resp := env.RawPOST("/webhooks/stripe", payload, map[string]string{
		"Content-Type": "application/json",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestStripeWebhook_EmptyBody(t *testing.T) {
	env := testutil.NewTestEnv(t)

	sig := testutil.StripeWebhookSignature([]byte{})
	resp := env.RawPOST("/webhooks/stripe", []byte{}, map[string]string{
		"Content-Type":     "application/json",
		"Stripe-Signature": sig,
	})
	defer resp.Body.Close()

	// Empty body passes sig verification but fails event parsing
	assert.True(t, resp.StatusCode >= 400, "expected 4xx error, got %d", resp.StatusCode)
}

func TestStripeWebhook_InvalidSignature(t *testing.T) {
	env := testutil.NewTestEnv(t)

	payload := []byte(`{"type":"checkout.session.completed","data":{"object":{"id":"cs_test"}}}`)
	resp := env.RawPOST("/webhooks/stripe", payload, map[string]string{
		"Content-Type":     "application/json",
		"Stripe-Signature": "t=1234567890,v1=invalid_signature_here",
	})
	defer resp.Body.Close()

	// Invalid signature should be rejected
	assert.True(t, resp.StatusCode >= 400, "expected 4xx error, got %d", resp.StatusCode)
}

func TestStripeWebhook_NoAuthRequired(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Webhook endpoint uses Stripe signature auth, not JWT.
	// A valid signature but unknown event type should still return a non-401 error.
	payload := []byte(`{"type":"unknown.event.type","data":{}}`)
	sig := testutil.StripeWebhookSignature(payload)
	resp := env.RawPOST("/webhooks/stripe", payload, map[string]string{
		"Content-Type":     "application/json",
		"Stripe-Signature": sig,
	})
	defer resp.Body.Close()

	// Route is accessible without JWT — signature auth only
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestStripeWebhook_ContentTypeRawBody(t *testing.T) {
	env := testutil.NewTestEnv(t)

	// Stripe sends application/json but handler reads raw body for sig verification
	payload := []byte(`{"id":"evt_test","type":"checkout.session.completed","data":{}}`)
	resp := env.RawPOST("/webhooks/stripe", payload, map[string]string{
		"Content-Type":     "application/json",
		"Stripe-Signature": "t=1234567890,v1=badhash",
	})
	defer resp.Body.Close()

	// Should process the body and fail on signature, not on content-type
	assert.True(t, resp.StatusCode >= 400, "expected signature error, got %d", resp.StatusCode)
	assert.NotEqual(t, http.StatusUnsupportedMediaType, resp.StatusCode)
}
