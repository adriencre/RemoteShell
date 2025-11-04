package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware est un middleware d'authentification pour Gin
func AuthMiddleware(tokenManager *TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Récupérer le token depuis l'header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token d'authentification manquant"})
			c.Abort()
			return
		}

		// Vérifier le format "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "format de token invalide"})
			c.Abort()
			return
		}

		token := parts[1]

		// Valider le token
		claims, err := tokenManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token invalide"})
			c.Abort()
			return
		}

		// Ajouter les claims au contexte
		// Pour les utilisateurs web, utiliser UserID/UserName
		// Pour les agents, utiliser AgentID/AgentName
		if claims.UserID != "" {
			c.Set("user_id", claims.UserID)
			c.Set("user_name", claims.UserName)
		}
		if claims.AgentID != "" {
			c.Set("agent_id", claims.AgentID)
			c.Set("agent_name", claims.AgentName)
		}
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}

// OptionalAuthMiddleware est un middleware d'authentification optionnel
func OptionalAuthMiddleware(tokenManager *TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]
		claims, err := tokenManager.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("agent_id", claims.AgentID)
		c.Set("agent_name", claims.AgentName)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}

// RequireRole vérifie que l'utilisateur a le rôle requis
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "rôle non défini"})
			c.Abort()
			return
		}

		if role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "permissions insuffisantes"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetAgentIDFromContext récupère l'ID de l'agent depuis le contexte
func GetAgentIDFromContext(c *gin.Context) (string, bool) {
	agentID, exists := c.Get("agent_id")
	if !exists {
		return "", false
	}
	return agentID.(string), true
}

// GetClaimsFromContext récupère les claims depuis le contexte
func GetClaimsFromContext(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	return claims.(*Claims), true
}


