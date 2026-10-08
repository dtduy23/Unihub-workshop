package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"unihub-workshop/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
)

// AuthMiddleware validates JWT token and extracts claims
func AuthMiddleware(secret string, resolvers ...func(context.Context, string) (*model.User, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				authError(w, http.StatusUnauthorized, "Missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				authError(w, http.StatusUnauthorized, "Invalid Authorization format")
				return
			}

			tokenStr := parts[1]
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())

			if err != nil || !token.Valid {
				authError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				authError(w, http.StatusUnauthorized, "Invalid token claims")
				return
			}

			// Check expiration
			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().Unix() > int64(exp) {
					authError(w, http.StatusUnauthorized, "Token expired")
					return
				}
			}

			userID, _ := claims["sub"].(string)
			role, _ := claims["role"].(string)
			if userID == "" || !model.Role(role).Valid() {
				authError(w, http.StatusUnauthorized, "Invalid token claims")
				return
			}
			if len(resolvers) > 0 {
				user, err := resolvers[0](r.Context(), userID)
				version := 1
				if v, ok := claims["ver"].(float64); ok {
					version = int(v)
				}
				if err != nil || user == nil || user.Role != model.Role(role) || user.AuthVersion != version {
					authError(w, http.StatusUnauthorized, "Session expired")
					return
				}
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, RoleKey, model.Role(role))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks that the user has one of the allowed roles
func RequireRole(roles ...model.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := r.Context().Value(RoleKey).(model.Role)
			if !ok {
				authError(w, http.StatusForbidden, "Role not found in context")
				return
			}

			for _, allowed := range roles {
				if userRole == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			authError(w, http.StatusForbidden, "Forbidden: insufficient permissions")
		})
	}
}

// GenerateJWT creates a signed JWT token
func GenerateJWT(secret, userID string, role model.Role, versions ...int) (string, error) {
	if !role.Valid() || userID == "" {
		return "", fmt.Errorf("invalid identity")
	}
	version := 1
	if len(versions) > 0 {
		version = versions[0]
	}
	claims := jwt.MapClaims{
		"sub":  userID,
		"ver":  version,
		"role": string(role),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// GetUserRole extracts role from context
func GetUserRole(ctx context.Context) model.Role {
	if v, ok := ctx.Value(RoleKey).(model.Role); ok {
		return v
	}
	return ""
}

func authError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.APIResponse{Success: false, Error: message})
}
