package tenant

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrInvalidKey     = errors.New("invalid or revoked API key")
)

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Tier      string    `json:"tier"`
	CreatedAt time.Time `json:"created_at"`
}

type APIKey struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	KeyHash   string     `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type RateLimitPolicy struct {
	TenantID      string `json:"tenant_id"`
	Algorithm     string `json:"algorithm"`
	RPM           int    `json:"rpm"`
	TPM           int    `json:"tpm"`
	TPD           int    `json:"tpd"`
	MaxConcurrent int    `json:"max_concurrent"`
}

// Store defines the interface for data access.
type Store interface {
	CreateTenant(ctx context.Context, name, tier string) (*Tenant, error)
	GetTenant(ctx context.Context, id string) (*Tenant, error)
	
	CreateAPIKey(ctx context.Context, tenantID, keyHash string) (*APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	RevokeAPIKey(ctx context.Context, id string) error
	
	GetPolicy(ctx context.Context, tenantID string) (*RateLimitPolicy, error)
	UpdatePolicy(ctx context.Context, policy *RateLimitPolicy) error
}

// PostgresStore implements Store using database/sql.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) CreateTenant(ctx context.Context, name, tier string) (*Tenant, error) {
	query := `INSERT INTO tenants (name, tier) VALUES ($1, $2) RETURNING id, name, tier, created_at`
	t := &Tenant{}
	err := s.db.QueryRowContext(ctx, query, name, tier).Scan(&t.ID, &t.Name, &t.Tier, &t.CreatedAt)
	return t, err
}

func (s *PostgresStore) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	query := `SELECT id, name, tier, created_at FROM tenants WHERE id = $1`
	t := &Tenant{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(&t.ID, &t.Name, &t.Tier, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrTenantNotFound
	}
	return t, err
}

func (s *PostgresStore) CreateAPIKey(ctx context.Context, tenantID, keyHash string) (*APIKey, error) {
	query := `INSERT INTO api_keys (tenant_id, key_hash) VALUES ($1, $2) RETURNING id, tenant_id, key_hash, created_at`
	k := &APIKey{}
	err := s.db.QueryRowContext(ctx, query, tenantID, keyHash).Scan(&k.ID, &k.TenantID, &k.KeyHash, &k.CreatedAt)
	return k, err
}

func (s *PostgresStore) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `SELECT id, tenant_id, key_hash, created_at, revoked_at FROM api_keys WHERE key_hash = $1`
	k := &APIKey{}
	err := s.db.QueryRowContext(ctx, query, keyHash).Scan(&k.ID, &k.TenantID, &k.KeyHash, &k.CreatedAt, &k.RevokedAt)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidKey
	}
	return k, err
}

func (s *PostgresStore) RevokeAPIKey(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET revoked_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *PostgresStore) GetPolicy(ctx context.Context, tenantID string) (*RateLimitPolicy, error) {
	query := `SELECT tenant_id, algorithm, rpm, tpm, tpd, max_concurrent FROM rate_limit_policies WHERE tenant_id = $1`
	p := &RateLimitPolicy{}
	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(&p.TenantID, &p.Algorithm, &p.RPM, &p.TPM, &p.TPD, &p.MaxConcurrent)
	if err == sql.ErrNoRows {
		// Return default policy
		return &RateLimitPolicy{
			TenantID:      tenantID,
			Algorithm:     "token_bucket",
			RPM:           60,
			TPM:           10000,
			TPD:           100000,
			MaxConcurrent: 5,
		}, nil
	}
	return p, err
}

func (s *PostgresStore) UpdatePolicy(ctx context.Context, policy *RateLimitPolicy) error {
	query := `
		INSERT INTO rate_limit_policies (tenant_id, algorithm, rpm, tpm, tpd, max_concurrent) 
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id) 
		DO UPDATE SET algorithm = $2, rpm = $3, tpm = $4, tpd = $5, max_concurrent = $6
	`
	_, err := s.db.ExecContext(ctx, query, policy.TenantID, policy.Algorithm, policy.RPM, policy.TPM, policy.TPD, policy.MaxConcurrent)
	return err
}
