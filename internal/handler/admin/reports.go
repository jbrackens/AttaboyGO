package admin

import (
	"net/http"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/handler"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportsHandler handles admin report generation.
type ReportsHandler struct {
	pool *pgxpool.Pool
}

// NewReportsHandler creates a new ReportsHandler.
func NewReportsHandler(pool *pgxpool.Pool) *ReportsHandler {
	return &ReportsHandler{pool: pool}
}

// GetDashboardStats handles GET /admin/reports/dashboard.
func (h *ReportsHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	type stats struct {
		TotalPlayers      int   `json:"total_players"`
		ActivePlayers     int   `json:"active_players"`
		TotalDeposits     int64 `json:"total_deposits"`
		TotalWithdrawals  int64 `json:"total_withdrawals"`
		PendingWithdrawals int  `json:"pending_withdrawals"`
		OpenBets          int   `json:"open_bets"`
	}

	var s stats

	// Total players
	h.pool.QueryRow(r.Context(), `SELECT COUNT(*) FROM v2_players`).Scan(&s.TotalPlayers)

	// Active players (have transactions in last 30 days)
	h.pool.QueryRow(r.Context(), `
		SELECT COUNT(DISTINCT player_id) FROM v2_transactions
		WHERE created_at > now() - interval '30 days'`).Scan(&s.ActivePlayers)

	// Pending withdrawals
	h.pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM payments WHERE type = 'withdrawal' AND status = 'pending'`).Scan(&s.PendingWithdrawals)

	// Open sportsbook bets
	h.pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM sports_bets WHERE status = 'open'`).Scan(&s.OpenBets)

	handler.RespondJSON(w, http.StatusOK, s)
}

// GetTransactionReport handles GET /admin/reports/transactions.
func (h *ReportsHandler) GetTransactionReport(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "7 days"
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT type, COUNT(*) as count, SUM(amount) as total
		FROM v2_transactions
		WHERE created_at > now() - $1::interval
		GROUP BY type ORDER BY count DESC`, period)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("query report", err))
		return
	}
	defer rows.Close()

	type txSummary struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
		Total int64  `json:"total"`
	}

	var summaries []txSummary
	for rows.Next() {
		var s txSummary
		if err := rows.Scan(&s.Type, &s.Count, &s.Total); err != nil {
			handler.RespondError(w, domain.ErrInternal("scan report", err))
			return
		}
		summaries = append(summaries, s)
	}

	handler.RespondJSON(w, http.StatusOK, summaries)
}
