package service

import (
	"context"
	"log/slog"

	"github.com/attaboy/platform/internal/auth"
	"github.com/attaboy/platform/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// AffiliateService handles affiliate operations.
type AffiliateService struct {
	pool   *pgxpool.Pool
	jwtMgr *auth.JWTManager
	logger *slog.Logger
}

// NewAffiliateService creates an AffiliateService.
func NewAffiliateService(pool *pgxpool.Pool, jwtMgr *auth.JWTManager, logger *slog.Logger) *AffiliateService {
	return &AffiliateService{pool: pool, jwtMgr: jwtMgr, logger: logger}
}

// AffiliateRegisterInput holds affiliate registration fields.
type AffiliateRegisterInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Company   string `json:"company,omitempty"`
	Website   string `json:"website,omitempty"`
}

// Register creates a new affiliate account.
func (s *AffiliateService) Register(ctx context.Context, input AffiliateRegisterInput) (*AuthResult, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domain.ErrInternal("hash password", err)
	}

	affID := uuid.New()
	code := "AFF" + affID.String()[:8]

	_, err = s.pool.Exec(ctx, `
		INSERT INTO affiliates (id, email, password_hash, first_name, last_name, company, affiliate_code, website, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending')`,
		affID, input.Email, string(hash), input.FirstName, input.LastName, input.Company, code, input.Website)
	if err != nil {
		return nil, domain.ErrInternal("create affiliate", err)
	}

	token, err := s.jwtMgr.GenerateToken(auth.RealmAffiliate, affID, input.Email, "", "pending")
	if err != nil {
		return nil, domain.ErrInternal("generate token", err)
	}

	return &AuthResult{Token: token, PlayerID: affID, Email: input.Email}, nil
}

// AffiliateLoginInput holds affiliate login fields.
type AffiliateLoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates an affiliate.
func (s *AffiliateService) Login(ctx context.Context, input AffiliateLoginInput) (*AuthResult, error) {
	var affID uuid.UUID
	var email, passwordHash, status string
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, status FROM affiliates WHERE email = $1`,
		input.Email).Scan(&affID, &email, &passwordHash, &status)
	if err != nil {
		return nil, domain.ErrUnauthorized("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrUnauthorized("invalid credentials")
	}

	token, err := s.jwtMgr.GenerateToken(auth.RealmAffiliate, affID, email, "", status)
	if err != nil {
		return nil, domain.ErrInternal("generate token", err)
	}

	return &AuthResult{Token: token, PlayerID: affID, Email: email}, nil
}

// CalculateCommission computes commission for an affiliate based on their plan.
func (s *AffiliateService) CalculateCommission(ctx context.Context, affiliateID uuid.UUID, ggr int64, model domain.CommissionModel) int64 {
	switch model {
	case domain.CommissionCPA:
		return 5000 // Fixed CPA amount (€50.00)
	case domain.CommissionRevShare:
		return ggr * 25 / 100 // 25% rev share
	case domain.CommissionHybrid:
		return 2500 + (ggr * 15 / 100) // €25 CPA + 15% rev share
	case domain.CommissionTiered:
		if ggr > 100000 {
			return ggr * 35 / 100
		} else if ggr > 50000 {
			return ggr * 30 / 100
		}
		return ggr * 25 / 100
	default:
		return 0
	}
}

// TrackClick records an affiliate click.
func (s *AffiliateService) TrackClick(ctx context.Context, btag, ipAddr, userAgent, referer string) {
	// Find link by btag
	var linkID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM affiliate_links WHERE btag = $1`, btag).Scan(&linkID)
	if err != nil {
		s.logger.Warn("btag not found", "btag", btag)
		return
	}

	// Record click
	_, err = s.pool.Exec(ctx, `
		INSERT INTO affiliate_clicks (link_id, ip_address, user_agent, referrer_url)
		VALUES ($1, $2, $3, $4)`,
		linkID, ipAddr, userAgent, referer)
	if err != nil {
		s.logger.Error("record click", "error", err, "btag", btag)
	}
}
