package repository

import (
	"context"
	"errors"

	"github.com/attaboy/platform/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PgProfileRepository implements ProfileRepository using pgx.
type PgProfileRepository struct{}

// NewPgProfileRepository creates a new PgProfileRepository.
func NewPgProfileRepository() *PgProfileRepository {
	return &PgProfileRepository{}
}

// FindByPlayerID returns a player profile, or nil if not found.
func (r *PgProfileRepository) FindByPlayerID(ctx context.Context, db DBTX, playerID uuid.UUID) (*domain.PlayerProfile, error) {
	row := db.QueryRow(ctx,
		`SELECT player_id, email, first_name, last_name, date_of_birth,
		        country, currency, language, mobile_phone, address, post_code,
		        verified, account_status, risk_profile, created_at
		 FROM player_profiles WHERE player_id = $1`, playerID)

	p := &domain.PlayerProfile{}
	err := row.Scan(
		&p.PlayerID, &p.Email, &p.FirstName, &p.LastName, &p.DateOfBirth,
		&p.Country, &p.Currency, &p.Language, &p.MobilePhone, &p.Address, &p.PostCode,
		&p.Verified, &p.AccountStatus, &p.RiskProfile, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Create inserts a new player profile.
func (r *PgProfileRepository) Create(ctx context.Context, db DBTX, profile *domain.PlayerProfile) error {
	_, err := db.Exec(ctx,
		`INSERT INTO player_profiles (player_id, email, first_name, last_name, date_of_birth,
		 country, currency, language, mobile_phone, address, post_code,
		 verified, account_status, risk_profile)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		profile.PlayerID, profile.Email, profile.FirstName, profile.LastName, profile.DateOfBirth,
		profile.Country, profile.Currency, profile.Language, profile.MobilePhone, profile.Address, profile.PostCode,
		profile.Verified, profile.AccountStatus, profile.RiskProfile,
	)
	return err
}

// Update modifies a player profile.
func (r *PgProfileRepository) Update(ctx context.Context, db DBTX, profile *domain.PlayerProfile) error {
	_, err := db.Exec(ctx,
		`UPDATE player_profiles SET
		 first_name = $2, last_name = $3, date_of_birth = $4,
		 country = $5, currency = $6, language = $7,
		 mobile_phone = $8, address = $9, post_code = $10,
		 verified = $11, account_status = $12, risk_profile = $13
		 WHERE player_id = $1`,
		profile.PlayerID, profile.FirstName, profile.LastName, profile.DateOfBirth,
		profile.Country, profile.Currency, profile.Language,
		profile.MobilePhone, profile.Address, profile.PostCode,
		profile.Verified, profile.AccountStatus, profile.RiskProfile,
	)
	return err
}
