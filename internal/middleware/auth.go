package middleware

import (
	"net/http"
	"strings"

	"github.com/ca-x/vaultwarden-syncer/internal/auth"
	"github.com/ca-x/vaultwarden-syncer/internal/setup"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	authService  *auth.Service
	setupService *setup.SetupService
}

func NewAuthMiddleware(authService *auth.Service, setupService *setup.SetupService) *AuthMiddleware {
	return &AuthMiddleware{
		authService:  authService,
		setupService: setupService,
	}
}

// RequireAuth middleware validates JWT token and sets user claims in context
// It also checks if setup is complete before requiring authentication
func (m *AuthMiddleware) RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// First check if setup is complete
			setupComplete, err := m.setupService.IsSetupComplete(c.Request().Context())
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check setup status"})
			}

			// If setup is not complete, redirect to setup page
			if !setupComplete {
				return c.Redirect(http.StatusFound, "/setup")
			}

			// Try to get token from Authorization header first
			authHeader := c.Request().Header.Get("Authorization")
			var token string

			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				// Try to get token from cookie
				cookie, err := c.Cookie("auth_token")
				if err != nil {
					return c.Redirect(http.StatusFound, "/login")
				}
				token = cookie.Value
			}

			if token == "" {
				return c.Redirect(http.StatusFound, "/login")
			}

			claims, err := m.authService.ValidateToken(token)
			if err != nil {
				return c.Redirect(http.StatusFound, "/login")
			}

			// Set user claims in context
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("is_admin", claims.IsAdmin)
			c.Set("claims", claims)

			return next(c)
		}
	}
}

// RequireAdmin middleware requires the user to be an admin
func (m *AuthMiddleware) RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isAdmin, ok := c.Get("is_admin").(bool)
			if !ok || !isAdmin {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Admin access required",
				})
			}
			return next(c)
		}
	}
}

// GetUserID helper function to extract user ID from context
func GetUserID(c echo.Context) (int, bool) {
	userID, ok := c.Get("user_id").(int)
	return userID, ok
}

// GetUsername helper function to extract username from context
func GetUsername(c echo.Context) (string, bool) {
	username, ok := c.Get("username").(string)
	return username, ok
}

// IsAdmin helper function to check if user is admin
func IsAdmin(c echo.Context) bool {
	isAdmin, ok := c.Get("is_admin").(bool)
	return ok && isAdmin
}

// GetClaims helper function to extract full claims from context
func GetClaims(c echo.Context) (*auth.Claims, bool) {
	claims, ok := c.Get("claims").(*auth.Claims)
	return claims, ok
}
