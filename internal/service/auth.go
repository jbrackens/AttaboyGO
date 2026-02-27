package service

import (
	"context"
	"fmt"

	"github.com/attaboy/platform/internal/auth"
	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/repository"
	"github.com/google/uuid"
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
}

// Login authenticates a player and returns a JWT.
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	user, err := s.users.FindByEmail(ctx, s.pool, input.Email)
	if err != nil {
		return nil, domain.ErrInternal("find user", err)
	}
	if user == nil {
		return nil, domain.ErrUnauthorized("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrUnauthorized("invalid credentials")
	}

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
