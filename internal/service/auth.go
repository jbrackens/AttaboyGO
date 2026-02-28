package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/attaboy/platform/internal/auth"
	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/guard"
	"github.com/attaboy/platform/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles player registration and login.
type AuthService struct {
	pool     *pgxpool.Pool
	users    repository.AuthUserRepository
	players  repository.PlayerRepository
	profiles repository.ProfileRepository
	jwtMgr   *auth.JWTManager
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	pool *pgxpool.Pool,
	users repository.AuthUserRepository,
	players repository.PlayerRepository,
	profiles repository.ProfileRepository,
	jwtMgr *auth.JWTManager,
) *AuthService {
	return &AuthService{
		pool:     pool,
		users:    users,
		players:  players,
		profiles: profiles,
		jwtMgr:   jwtMgr,
	}
}

// RegisterInput holds the registration request fields.
type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Currency string `json:"currency"`
}

// AuthResult is returned on successful registration or login.
type AuthResult struct {
	Token    string        `json:"token"`
	PlayerID uuid.UUID     `json:"player_id"`
	Email    string        `json:"email"`
	Balance  domain.Balances `json:"balance"`
}

// Register creates a new player account within a single transaction.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	if err := domain.ValidateEmail(input.Email); err != nil {
		return nil, domain.ErrValidation(err.Error())
	}
	if len(input.Password) < 8 {
		return nil, domain.ErrValidation("password must be at least 8 characters")
	}
	if input.Currency == "" {
		input.Currency = "EUR"
	}
	if err := domain.ValidateCurrency(input.Currency); err != nil {
		return nil, domain.ErrValidation(err.Error())
	}

	// Check for existing user
	existing, err := s.users.FindByEmail(ctx, s.pool, input.Email)
	if err != nil {
		return nil, domain.ErrInternal("find user", err)
	}
	if existing != nil {
		return nil, domain.ErrConflict("email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domain.ErrInternal("hash password", err)
	}

	// Run in transaction: create auth_user + v2_player + player_profile
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, domain.ErrInternal("begin tx", err)
	}
	defer tx.Rollback(ctx)

	playerID := uuid.New()

	authUser := &domain.AuthUser{
		ID:           playerID,
		Email:        input.Email,
		PasswordHash: string(hash),
	}
	if err := s.users.Create(ctx, tx, authUser); err != nil {
		return nil, domain.ErrInternal("create auth user", err)
	}

	player := &domain.Player{
		ID:       playerID,
		Currency: input.Currency,
	}
	if err := s.players.Create(ctx, tx, player); err != nil {
		return nil, domain.ErrInternal("create player", err)
	}

	profile := &domain.PlayerProfile{
		PlayerID:      playerID,
		Email:         input.Email,
		Currency:      input.Currency,
		Language:      "en",
		AccountStatus: "active",
		RiskProfile:   "low",
	}
	if err := s.profiles.Create(ctx, tx, profile); err != nil {
		return nil, domain.ErrInternal("create profile", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.ErrInternal("commit tx", err)
	}

	// Generate JWT
	token, err := s.jwtMgr.GenerateToken(auth.RealmPlayer, playerID, input.Email, "", "")
	if err != nil {
		return nil, domain.ErrInternal("generate token", err)
	}

	return &AuthResult{
		Token:    token,
		PlayerID: playerID,
		Email:    input.Email,
		Balance:  domain.Balances{},
	}, nil
}

// LoginInput holds the login request fields.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	IP       string `json:"-"`
}

// Login authenticates a player and returns a JWT.
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	if err := guard.CheckLocked(ctx, s.pool, input.Email, "player"); err != nil {
		return nil, err
	}

	user, err := s.users.FindByEmail(ctx, s.pool, input.Email)
	if err != nil {
		return nil, domain.ErrInternal("find user", err)
	}
	if user == nil {
		guard.RecordAttempt(ctx, s.pool, input.Email, "player", input.IP, false)
		return nil, domain.ErrUnauthorized("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		guard.RecordAttempt(ctx, s.pool, input.Email, "player", input.IP, false)
		return nil, domain.ErrUnauthorized("invalid credentials")
	}

	guard.RecordAttempt(ctx, s.pool, input.Email, "player", input.IP, true)

	// Fetch player for balance
	player, err := s.players.FindByID(ctx, s.pool, user.ID)
	if err != nil {
		return nil, domain.ErrInternal("find player", err)
	}
	if player == nil {
		return nil, domain.ErrInternal("player record missing", fmt.Errorf("no v2_players row for %s", user.ID))
	}

	token, err := s.jwtMgr.GenerateToken(auth.RealmPlayer, user.ID, user.Email, "", "")
	if err != nil {
		return nil, domain.ErrInternal("generate token", err)
	}

	return &AuthResult{
		Token:    token,
		PlayerID: user.ID,
		Email:    user.Email,
		Balance:  player.Balances,
	}, nil
}

// PasswordResetResult is returned when a reset token is requested.
type PasswordResetResult struct {
	Token string `json:"token"`
}

// RequestPasswordReset generates a reset token for the given email.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (*PasswordResetResult, error) {
	user, err := s.users.FindByEmail(ctx, s.pool, email)
	if err != nil {
		return nil, domain.ErrInternal("find user", err)
	}
	if user == nil {
		// Return success even if user not found (don't leak existence)
		return &PasswordResetResult{Token: ""}, nil
	}

	// Generate 32-byte random token
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return nil, domain.ErrInternal("generate token", err)
	}
	tokenHex := hex.EncodeToString(rawToken)

	// SHA-256 hash for storage
	hash := sha256.Sum256([]byte(tokenHex))
	tokenHash := hex.EncodeToString(hash[:])

	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = s.pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (email, realm, token_hash, expires_at)
		VALUES ($1, 'player', $2, $3)`,
		email, tokenHash, expiresAt)
	if err != nil {
		return nil, domain.ErrInternal("store reset token", err)
	}

	return &PasswordResetResult{Token: tokenHex}, nil
}

// ConfirmPasswordReset validates the token and updates the password.
func (s *AuthService) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return domain.ErrValidation("password must be at least 8 characters")
	}

	// Hash the input token to look up
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	var email string
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT id, email FROM password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()`,
		tokenHash).Scan(&id, &email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrValidation("invalid or expired reset token")
		}
		return domain.ErrInternal("lookup reset token", err)
	}

	// Hash new password
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return domain.ErrInternal("hash password", err)
	}

	// Update password + mark token as used in a transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ErrInternal("begin tx", err)
	}
	defer tx.Rollback(ctx)

	if err := s.users.UpdatePasswordHash(ctx, tx, email, string(bcryptHash)); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at = now() WHERE id = $1`, id)
	if err != nil {
		return domain.ErrInternal("mark token used", err)
	}

	return tx.Commit(ctx)
}
