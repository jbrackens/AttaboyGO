package infra

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration parsed from environment variables.
type Config struct {
	// Database
	DatabaseURL string `env:"DATABASE_URL"`
	PGHost      string `env:"PGHOST" envDefault:"localhost"`
	PGPort      int    `env:"PGPORT" envDefault:"5435"`
	PGUser      string `env:"PGUSER" envDefault:"attaboy"`
	PGPassword  string `env:"PGPASSWORD" envDefault:"attaboy"`
	PGDatabase  string `env:"PGDATABASE" envDefault:"attaboy"`

	// Redis
	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6380"`

	// JWT
	JWTSecret         string `env:"JWT_SECRET" envDefault:"change-me-in-production"`
	JWTPlayerExpiry    string `env:"JWT_PLAYER_EXPIRY" envDefault:"24h"`
	JWTAdminExpiry     string `env:"JWT_ADMIN_EXPIRY" envDefault:"8h"`
	JWTAffiliateExpiry string `env:"JWT_AFFILIATE_EXPIRY" envDefault:"12h"`

	// Server ports
	APIPort          int `env:"API_PORT" envDefault:"3100"`
	WalletServerPort int `env:"WALLET_SERVER_PORT" envDefault:"4001"`

	// Kafka
	KafkaBrokers string `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
	KafkaEnabled bool   `env:"KAFKA_ENABLED" envDefault:"false"`

	// CORS
	CORSAllowedOrigins string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`

	// Dev
	AllowInsecureDefaults bool `env:"ALLOW_INSECURE_DEFAULTS" envDefault:"false"`

	// External services
	RandomOrgAPIKey     string `env:"RANDOM_ORG_API_KEY"`
	StripeSecretKey     string `env:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret string `env:"STRIPE_WEBHOOK_SECRET"`
}

// LoadConfig parses environment variables into a Config struct.
func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Validate checks for insecure configuration that must not run in production.
// Set ALLOW_INSECURE_DEFAULTS=true to bypass (local dev only).
func (c *Config) Validate() error {
	if c.AllowInsecureDefaults {
		return nil
	}
	if c.JWTSecret == "change-me-in-production" {
		return fmt.Errorf("JWT_SECRET is set to the insecure default; set a strong secret or set ALLOW_INSECURE_DEFAULTS=true for local dev")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET is too short (%d chars); minimum 32 characters required", len(c.JWTSecret))
	}
	return nil
}

// DSN returns the PostgreSQL connection string, preferring DATABASE_URL if set.
func (c *Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.PGUser, c.PGPassword, c.PGHost, c.PGPort, c.PGDatabase)
}
