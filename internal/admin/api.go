package admin

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminServer struct {
	db *pgxpool.Pool
}

func NewAdminServer(db *pgxpool.Pool) *AdminServer {
	return &AdminServer{db: db}
}

// jsonResponse is a helper to send JSON HTTP responses
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// jsonError is a helper to send JSON HTTP error responses
func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}

// authMiddleware enforces JWT authentication on admin routes
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			jsonError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tenantID, err := ValidateJWT(tokenString)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// Inject tenantID into context
		ctx := context.WithValue(r.Context(), "admin_tenant_id", tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// MountRoutes sets up all the admin endpoints on a given mux
func (s *AdminServer) MountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /admin/login", s.handleLogin)
	mux.HandleFunc("POST /admin/tenants", authMiddleware(s.handleCreateTenant))
	mux.HandleFunc("GET /admin/tenants/", authMiddleware(s.handleGetTenant))
	mux.HandleFunc("POST /admin/api-keys", authMiddleware(s.handleCreateAPIKey))
	mux.HandleFunc("DELETE /admin/api-keys/", authMiddleware(s.handleDeleteAPIKey))
	mux.HandleFunc("PUT /admin/policy", authMiddleware(s.handleUpdatePolicy))
	mux.HandleFunc("GET /admin/usage", authMiddleware(s.handleGetUsage))
	mux.HandleFunc("GET /admin/providers/health", authMiddleware(s.handleProviderHealth))
}

func (s *AdminServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Dummy login for testing - in production, check admin credentials
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "admin" && req.Password == "admin" {
		token, err := GenerateJWT("admin_tenant")
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"token": token})
		return
	}

	jsonError(w, http.StatusUnauthorized, "invalid credentials")
}

func (s *AdminServer) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Tier string `json:"tier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Tier == "" {
		req.Tier = "free"
	}

	var id string
	err := s.db.QueryRow(r.Context(), "INSERT INTO tenants (name, tier) VALUES ($1, $2) RETURNING id", req.Name, req.Tier).Scan(&id)
	if err != nil {
		log.Printf("DB error: %v", err)
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": id, "name": req.Name, "tier": req.Tier})
}

func (s *AdminServer) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/tenants/")
	if id == "" {
		jsonError(w, http.StatusBadRequest, "missing tenant id")
		return
	}

	var name, tier string
	err := s.db.QueryRow(r.Context(), "SELECT name, tier FROM tenants WHERE id = $1", id).Scan(&name, &tier)
	if err != nil {
		jsonError(w, http.StatusNotFound, "tenant not found")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"id": id, "name": name, "tier": tier})
}

func (s *AdminServer) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
		Key      string `json:"key"` // Plaintext key provided by user (or generated here)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hash, err := HashAPIKey(req.Key)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to hash key")
		return
	}

	var id string
	err = s.db.QueryRow(r.Context(), "INSERT INTO api_keys (tenant_id, key_hash) VALUES ($1, $2) RETURNING id", req.TenantID, hash).Scan(&id)
	if err != nil {
		log.Printf("DB error: %v", err)
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": id, "tenant_id": req.TenantID})
}

func (s *AdminServer) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/api-keys/")
	if id == "" {
		jsonError(w, http.StatusBadRequest, "missing key id")
		return
	}

	_, err := s.db.Exec(r.Context(), "UPDATE api_keys SET revoked_at = CURRENT_TIMESTAMP WHERE id = $1", id)
	if err != nil {
		log.Printf("DB error: %v", err)
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (s *AdminServer) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID      string `json:"tenant_id"`
		Algorithm     string `json:"algorithm"`
		RPM           int    `json:"rpm"`
		TPM           int    `json:"tpm"`
		TPD           int    `json:"tpd"`
		MaxConcurrent int    `json:"max_concurrent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err := s.db.Exec(r.Context(), 
		`INSERT INTO rate_limit_policies (tenant_id, algorithm, rpm, tpm, tpd, max_concurrent) 
		 VALUES ($1, $2, $3, $4, $5, $6) 
		 ON CONFLICT (tenant_id) DO UPDATE SET 
		 algorithm = EXCLUDED.algorithm, rpm = EXCLUDED.rpm, tpm = EXCLUDED.tpm, tpd = EXCLUDED.tpd, max_concurrent = EXCLUDED.max_concurrent`,
		req.TenantID, req.Algorithm, req.RPM, req.TPM, req.TPD, req.MaxConcurrent)
	if err != nil {
		log.Printf("DB error: %v", err)
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	jsonResponse(w, http.StatusOK, req)
}

func (s *AdminServer) handleGetUsage(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		jsonError(w, http.StatusBadRequest, "missing tenant_id")
		return
	}

	// For dashboard charts: get aggregated usage over last 24h
	rows, err := s.db.Query(r.Context(), `
		SELECT date_trunc('hour', created_at) as hour, 
		       SUM(prompt_tokens + completion_tokens) as tokens,
		       SUM(cost_usd) as cost
		FROM usage_events 
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '24 HOURS'
		GROUP BY hour ORDER BY hour ASC`, tenantID)
	
	if err != nil {
		log.Printf("DB error: %v", err)
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	type UsagePoint struct {
		Hour   time.Time `json:"hour"`
		Tokens int64     `json:"tokens"`
		Cost   float64   `json:"cost"`
	}
	var data []UsagePoint

	for rows.Next() {
		var p UsagePoint
		if err := rows.Scan(&p.Hour, &p.Tokens, &p.Cost); err != nil {
			continue
		}
		data = append(data, p)
	}

	jsonResponse(w, http.StatusOK, data)
}

func (s *AdminServer) handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	// Stub: In a real system, query circuit breaker state from Redis or memory
	health := []map[string]interface{}{
		{"provider": "gemini", "status": "healthy", "error_rate": 0.01},
		{"provider": "openai", "status": "degraded", "error_rate": 0.15},
		{"provider": "anthropic", "status": "healthy", "error_rate": 0.00},
	}
	jsonResponse(w, http.StatusOK, health)
}
