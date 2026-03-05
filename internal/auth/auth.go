// Package auth provides authentication and authorization primitives for Gantry.
// It handles password hashing via bcrypt and token generation/validation via JWT.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the work factor used for password hashing.
const bcryptCost = 10

// tokenExpiry is the duration a JWT remains valid after issuance.
const tokenExpiry = 24 * time.Hour

// User represents an authenticated user in the system.
type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Password    string    `json:"-"` // never serialized
	DisplayName string    `json:"displayName,omitempty"`
	Email       string    `json:"email,omitempty"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Claims holds the JWT claims for an authenticated session.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Service provides authentication operations including password hashing
// and JWT token generation/validation.
type Service struct {
	jwtSecret []byte
}

// NewService creates a new auth service with the given JWT signing secret.
func NewService(jwtSecret string) *Service {
	return &Service{
		jwtSecret: []byte(jwtSecret),
	}
}

// HashPassword returns a bcrypt hash of the given plaintext password.
func (s *Service) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a bcrypt hashed password with a plaintext candidate.
// It returns nil on success or an error if they do not match.
func (s *Service) CheckPassword(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("auth: check password: %w", err)
	}
	return nil
}

// GenerateToken creates a signed JWT for the given user with a 24-hour expiry.
func (s *Service) GenerateToken(user *User) (string, error) {
	now := time.Now().UTC()

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenExpiry)),
			Issuer:    "gantry",
		},
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("auth: generate token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT string, returning the embedded claims.
// It returns an error if the token is malformed, expired, or has an invalid signature.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		// Ensure the signing method is HMAC.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("auth: token expired: %w", err)
		}
		return nil, fmt.Errorf("auth: validate token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("auth: invalid token")
	}

	return claims, nil
}
