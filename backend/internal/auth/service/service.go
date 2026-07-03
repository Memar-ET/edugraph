package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/edugraph-ai/edugraph/internal/auth/dto"
	"github.com/edugraph-ai/edugraph/internal/auth/repository"
	"github.com/edugraph-ai/edugraph/pkg/crypto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

type Service struct {
	repo  *repository.Repository
	jwt   *crypto.JWTSigner
	redis *goredis.Client
}

func New(repo *repository.Repository, jwt *crypto.JWTSigner, redis *goredis.Client) *Service {
	return &Service{repo: repo, jwt: jwt, redis: redis}
}

func (s *Service) Register(ctx context.Context, req dto.RegisterRequest) (dto.AuthResponse, error) {
	if _, err := s.repo.GetUserByEmail(ctx, req.Email); err == nil {
		return dto.AuthResponse{}, apperrors.Conflict("an account with this email already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}

	user, err := s.repo.CreateUser(ctx, repository.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
		FullName:     req.FullName,
		Phone:        nilIfEmpty(req.Phone),
		RegionID:     nilIfEmpty(req.RegionID),
		SchoolID:     nilIfEmpty(req.SchoolID),
	})
	if err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}

	return s.issueTokens(ctx, user)
}

func (s *Service) Login(ctx context.Context, req dto.LoginRequest) (dto.AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.AuthResponse{}, apperrors.Unauthorized("invalid email or password")
	}
	if err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}
	if !user.IsActive {
		return dto.AuthResponse{}, apperrors.Forbidden("account is deactivated")
	}
	if err := crypto.ComparePassword(user.PasswordHash, req.Password); err != nil {
		return dto.AuthResponse{}, apperrors.Unauthorized("invalid email or password")
	}

	return s.issueTokens(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (dto.AuthResponse, error) {
	hash := hashToken(refreshToken)

	stored, err := s.repo.GetRefreshToken(ctx, hash)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.AuthResponse{}, apperrors.Unauthorized("invalid refresh token")
	}
	if err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}
	if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		return dto.AuthResponse{}, apperrors.Unauthorized("refresh token expired or revoked")
	}

	user, err := s.repo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}

	if err := s.repo.RevokeRefreshToken(ctx, hash); err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}

	return s.issueTokens(ctx, user)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if err := s.repo.RevokeRefreshToken(ctx, hashToken(refreshToken)); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (dto.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.UserResponse{}, apperrors.NotFound("user not found")
	}
	if err != nil {
		return dto.UserResponse{}, apperrors.Internal(err)
	}
	return toUserResponse(user), nil
}

// VerifyAccessToken satisfies pkg/middleware.TokenVerifier.
func (s *Service) VerifyAccessToken(ctx context.Context, token string) (userID string, role string, err error) {
	claims, err := s.jwt.Verify(token)
	if err != nil {
		return "", "", err
	}
	return claims.UserID, claims.Role, nil
}

func (s *Service) issueTokens(ctx context.Context, user repository.User) (dto.AuthResponse, error) {
	access, err := s.jwt.IssueAccessToken(user.ID, user.Role)
	if err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}
	refresh, err := s.jwt.IssueRefreshToken(user.ID, user.Role)
	if err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}

	expiresAt := time.Now().Add(s.jwt.RefreshTTL())
	if err := s.repo.StoreRefreshToken(ctx, user.ID, hashToken(refresh), expiresAt); err != nil {
		return dto.AuthResponse{}, apperrors.Internal(err)
	}

	return dto.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(time.Until(expiresAt).Seconds()),
		User:         toUserResponse(user),
	}, nil
}

func toUserResponse(u repository.User) dto.UserResponse {
	resp := dto.UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
	if u.Phone != nil {
		resp.Phone = *u.Phone
	}
	if u.RegionID != nil {
		resp.RegionID = *u.RegionID
	}
	if u.SchoolID != nil {
		resp.SchoolID = *u.SchoolID
	}
	return resp
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
