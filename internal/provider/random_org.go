package provider

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"time"
)

// RandomOrgClient provides true random numbers from RANDOM.ORG with CSPRNG fallback.
type RandomOrgClient struct {
	apiKey string
	logger *slog.Logger
	client *http.Client
}

// NewRandomOrgClient creates a new RANDOM.ORG client.
func NewRandomOrgClient(apiKey string, logger *slog.Logger) *RandomOrgClient {
	return &RandomOrgClient{
		apiKey: apiKey,
		logger: logger,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// RandomIntegers returns n random integers in [min, max] from RANDOM.ORG.
// Falls back to crypto/rand if the API is unavailable.
func (c *RandomOrgClient) RandomIntegers(ctx context.Context, n, min, max int) ([]int, error) {
	if c.apiKey == "" {
		c.logger.Debug("random.org api key not set, using CSPRNG fallback")
		return csprngIntegers(n, min, max)
	}

	result, err := c.fetchFromAPI(ctx, n, min, max)
	if err != nil {
		c.logger.Warn("random.org unavailable, falling back to CSPRNG", "error", err)
		return csprngIntegers(n, min, max)
	}

	return result, nil
}

func (c *RandomOrgClient) fetchFromAPI(ctx context.Context, n, min, max int) ([]int, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "generateIntegers",
		"params": map[string]interface{}{
			"apiKey":      c.apiKey,
			"n":           n,
			"min":         min,
			"max":         max,
			"replacement": true,
		},
		"id": 1,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.random.org/json-rpc/4/invoke", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned %d", resp.StatusCode)
	}

	var response struct {
		Result struct {
			Random struct {
				Data []int `json:"data"`
			} `json:"random"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("api error: %s", response.Error.Message)
	}

	return response.Result.Random.Data, nil
}

// csprngIntegers generates cryptographically secure random integers as fallback.
func csprngIntegers(n, min, max int) ([]int, error) {
	if min > max {
		return nil, fmt.Errorf("min (%d) > max (%d)", min, max)
	}

	rangeSize := big.NewInt(int64(max - min + 1))
	result := make([]int, n)

	for i := 0; i < n; i++ {
		r, err := rand.Int(rand.Reader, rangeSize)
		if err != nil {
			return nil, fmt.Errorf("csprng: %w", err)
		}
		result[i] = int(r.Int64()) + min
	}

	return result, nil
}
