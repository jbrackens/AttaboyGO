package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBetSolutionsAdapter_VerifySignature(t *testing.T) {
	adapter := NewBetSolutionsAdapter("test-secret", nil)

	// Body without Hash field — compute valid signature
	body := []byte(`{"Token":"abc","PlayerId":"123"}`)
	validHash := adapter.ComputeSignature(body)

	// Valid hash should pass
	assert.True(t, adapter.VerifySignature(body, validHash))

	// Body with Hash field included — signature should still work
	// because VerifySignature strips the Hash field before computing
	bodyWithHash := []byte(`{"Token":"abc","PlayerId":"123","Hash":"` + validHash + `"}`)
	assert.True(t, adapter.VerifySignature(bodyWithHash, validHash))

	// Invalid hash should fail
	assert.False(t, adapter.VerifySignature(body, "invalid-hash"))
}

func TestPragmaticAdapter_VerifySignature(t *testing.T) {
	adapter := NewPragmaticAdapter("test-secret", nil)

	body := []byte(`{"userId":"abc","action":"balance"}`)
	validHash := adapter.ComputeSignature(body)

	assert.True(t, adapter.VerifySignature(body, validHash))

	// Body with hash field included
	bodyWithHash := []byte(`{"userId":"abc","action":"balance","hash":"` + validHash + `"}`)
	assert.True(t, adapter.VerifySignature(bodyWithHash, validHash))

	assert.False(t, adapter.VerifySignature(body, "wrong-hash"))
}

func TestParseDecimalToCents(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"10.50", 1050},
		{"0.99", 99},
		{"100", 10000},
		{"1.5", 150},
		{"0.01", 1},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDecimalToCents(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFormatCents(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{1050, "10.50"},
		{99, "0.99"},
		{10000, "100.00"},
		{1, "0.01"},
		{0, "0.00"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatCents(tt.input))
		})
	}
}

func TestToCentsFromCents(t *testing.T) {
	assert.Equal(t, int64(1500), ToCents(1500))
	assert.Equal(t, int64(1500), FromCents(1500))
}
