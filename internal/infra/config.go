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

	// External services
	RandomOrgAPIKey string `env:"RANDOM_ORG_API_KEY"`
	StripeSecretKey string `env:"STRIPE_SECRET_KEY"`
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

// DSN returns the PostgreSQL connection string, preferring DATABASE_URL if set.
func (c *Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.PGUser, c.PGPassword, c.PGHost, c.PGPort, c.PGDatabase)
}
