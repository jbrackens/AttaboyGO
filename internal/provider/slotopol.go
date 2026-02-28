package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// SlotopolClient communicates with the Slotopol slot game server.
type SlotopolClient struct {
	baseURL string
	logger  *slog.Logger
	client  *http.Client
}

// NewSlotopolClient creates a new Slotopol HTTP client.
func NewSlotopolClient(baseURL string, logger *slog.Logger) *SlotopolClient {
	return &SlotopolClient{
		baseURL: baseURL,
		logger:  logger,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// SlotopolGame represents a game from the Slotopol catalog.
type SlotopolGame struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Category    string   `json:"category"`
	RTP         float64  `json:"rtp"`
	Volatility  string   `json:"volatility"`
	Lines       int      `json:"lines"`
	Reels       int      `json:"reels"`
	Features    []string `json:"features,omitempty"`
}

// SlotopolSpinRequest is the request body for a spin.
type SlotopolSpinRequest struct {
	GameID string `json:"game_id"`
	Bet    int64  `json:"bet"` // in cents
	Lines  int    `json:"lines"`
}

// SlotopolSpinResult is the response from a spin.
type SlotopolSpinResult struct {
	GameRoundID string      `json:"game_round_id"`
	Win         int64       `json:"win"` // in cents
	Reels       [][]int     `json:"reels"`
	Paylines    []Payline   `json:"paylines,omitempty"`
	FreeSpins   int         `json:"free_spins"`
}

// Payline describes a winning payline.
type Payline struct {
	Line    int    `json:"line"`
	Symbol  string `json:"symbol"`
	Count   int    `json:"count"`
	Payout  int64  `json:"payout"`
}

// ListGames returns the available game catalog from Slotopol.
// Returns an empty list if the Slotopol service is unreachable.
func (c *SlotopolClient) ListGames(ctx context.Context) ([]SlotopolGame, error) {
	url := fmt.Sprintf("%s/games", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Warn("slotopol service unreachable, returning empty game list", "error", err)
		return []SlotopolGame{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("slotopol returned non-200, returning empty game list", "status", resp.StatusCode)
		return []SlotopolGame{}, nil
	}

	var games []SlotopolGame
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("decode games: %w", err)
	}

	return games, nil
}

// Spin executes a spin on the Slotopol server.
func (c *SlotopolClient) Spin(ctx context.Context, spin SlotopolSpinRequest) (*SlotopolSpinResult, error) {
	url := fmt.Sprintf("%s/spin", c.baseURL)

	body, _ := json.Marshal(spin)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, jsonReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spin request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("slotopol spin returned %d", resp.StatusCode)
	}

	var result SlotopolSpinResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode spin result: %w", err)
	}

	return &result, nil
}

// jsonReader creates a reader from a byte slice.
func jsonReader(data []byte) *jsonReadCloser {
	return &jsonReadCloser{data: data, pos: 0}
}

type jsonReadCloser struct {
	data []byte
	pos  int
}

func (r *jsonReadCloser) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *jsonReadCloser) Close() error { return nil }
