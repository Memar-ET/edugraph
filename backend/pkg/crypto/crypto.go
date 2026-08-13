// Package crypto provides password hashing and RS256 JWT signing used by
// the auth domain.
package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ── Passwords ──────────────────────────────────────────────────

// bcryptCost is set above bcrypt.DefaultCost (10) to match the minimum cost
// mandated by the DB design doc (identity.users.password_hash: "bcrypt hash,
// cost 12 minimum").
const bcryptCost = 12

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ── JWT ────────────────────────────────────────────────────────

type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type JWTSigner struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWTSigner(privateKeyPath, publicKeyPath string, accessTTL, refreshTTL time.Duration) (*JWTSigner, error) {
	priv, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}
	pub, err := loadPublicKey(publicKeyPath)
	if err != nil {
		return nil, err
	}
	return &JWTSigner{privateKey: priv, publicKey: pub, accessTTL: accessTTL, refreshTTL: refreshTTL}, nil
}

func (s *JWTSigner) IssueAccessToken(userID, role string) (string, error) {
	return s.sign(userID, role, s.accessTTL)
}

func (s *JWTSigner) IssueRefreshToken(userID, role string) (string, error) {
	return s.sign(userID, role, s.refreshTTL)
}

func (s *JWTSigner) RefreshTTL() time.Duration { return s.refreshTTL }
func (s *JWTSigner) AccessTTL() time.Duration  { return s.accessTTL }

func (s *JWTSigner) sign(userID, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			// ID (jti): RS256 is a deterministic signature scheme (no
			// randomized nonce like RSA-PSS/ECDSA), and IssuedAt/
			// ExpiresAt only carry second-granularity precision
			// (jwt.NewNumericDate) -- without this, two tokens issued
			// for the same user within the same wall-clock second are
			// byte-identical, which previously collided on
			// refresh_tokens.token_hash's unique constraint and turned
			// into a hard 500 (found via a real test hitting Register
			// then Login back-to-back, not a hypothetical).
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a token, returning its claims.
func (s *JWTSigner) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM block for private key")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
		return rsaKey, nil
	}
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return rsaKey, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM block for public key")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return rsaKey, nil
}

// EnsureDevKeyPair generates a local RSA keypair at privPath/pubPath if they
// don't already exist. Intended for local/dev use only — production
// deployments must mount real keys (see JWT_PRIVATE_KEY_PATH in .env.example).
func EnsureDevKeyPair(privPath, pubPath string) error {
	if _, err := os.Stat(privPath); err == nil {
		return nil // already generated
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate dev rsa key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(privPath), 0o700); err != nil {
		return fmt.Errorf("create key dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(pubPath), 0o700); err != nil {
		return fmt.Errorf("create key dir: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(key)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		return fmt.Errorf("write dev private key: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal dev public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	if err := os.WriteFile(pubPath, pubPEM, 0o600); err != nil {
		return fmt.Errorf("write dev public key: %w", err)
	}

	return nil
}
