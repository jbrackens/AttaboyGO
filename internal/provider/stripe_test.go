package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := "whsec_test_secret"
	p := NewStripeProvider("", secret)

	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{}}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())

	// Compute valid signature
	signedPayload := ts + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))

	sigHeader := fmt.Sprintf("t=%s,v1=%s", ts, sig)

	event, err := p.VerifyWebhookSignature(payload, sigHeader)
	require.NoError(t, err)
	assert.Equal(t, "evt_123", event.ID)
	assert.Equal(t, "checkout.session.completed", event.Type)
}

func TestVerifyWebhookSignature_InvalidSignature(t *testing.T) {
	p := NewStripeProvider("", "whsec_test_secret")

	payload := []byte(`{"id":"evt_123","type":"test"}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sigHeader := fmt.Sprintf("t=%s,v1=invalid_signature", ts)

	_, err := p.VerifyWebhookSignature(payload, sigHeader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook signature")
}

func TestVerifyWebhookSignature_ExpiredTimestamp(t *testing.T) {
	secret := "whsec_test_secret"
	p := NewStripeProvider("", secret)

	payload := []byte(`{"id":"evt_123","type":"test"}`)
	ts := fmt.Sprintf("%d", time.Now().Unix()-600) // 10 minutes ago

	signedPayload := ts + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))

	sigHeader := fmt.Sprintf("t=%s,v1=%s", ts, sig)

	_, err := p.VerifyWebhookSignature(payload, sigHeader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp too old")
}

func TestVerifyWebhookSignature_MissingHeader(t *testing.T) {
	p := NewStripeProvider("", "whsec_test_secret")
	_, err := p.VerifyWebhookSignature([]byte(`{}`), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature header format")
}
