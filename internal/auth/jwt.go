package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("token invalide")
	ErrExpiredToken = errors.New("token expiré")
	ErrInvalidClaims = errors.New("claims invalides")
)

// Claims contient les claims JWT
type Claims struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager gère les tokens JWT
type TokenManager struct {
	secretKey []byte
	issuer    string
}

// NewTokenManager crée un nouveau gestionnaire de tokens
func NewTokenManager(secretKey string, issuer string) *TokenManager {
	if secretKey == "" {
		secretKey = generateSecretKey()
	}
	return &TokenManager{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

// GenerateToken génère un nouveau token JWT
func (tm *TokenManager) GenerateToken(agentID, agentName, role string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		AgentID:   agentID,
		AgentName: agentName,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   agentID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secretKey)
}

// ValidateToken valide un token JWT
func (tm *TokenManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue: %v", token.Header["alg"])
		}
		return tm.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// RefreshToken génère un nouveau token basé sur un token existant
func (tm *TokenManager) RefreshToken(tokenString string, duration time.Duration) (string, error) {
	claims, err := tm.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Vérifier que le token n'est pas trop ancien (max 24h)
	if time.Since(claims.IssuedAt.Time) > 24*time.Hour {
		return "", ErrExpiredToken
	}

	return tm.GenerateToken(claims.AgentID, claims.AgentName, claims.Role, duration)
}

// generateSecretKey génère une clé secrète aléatoire
func generateSecretKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback si rand.Read échoue
		return "default-secret-key-change-in-production"
	}
	return hex.EncodeToString(bytes)
}

// GetSecretKey retourne la clé secrète (pour sauvegarde)
func (tm *TokenManager) GetSecretKey() string {
	return hex.EncodeToString(tm.secretKey)
}

// SetSecretKey définit une nouvelle clé secrète
func (tm *TokenManager) SetSecretKey(secretKey string) error {
	if len(secretKey) < 32 {
		return errors.New("la clé secrète doit faire au moins 32 caractères")
	}
	tm.secretKey = []byte(secretKey)
	return nil
}


