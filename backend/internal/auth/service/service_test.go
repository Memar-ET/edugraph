package service_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edugraph-ai/edugraph/internal/auth/dto"
	"github.com/edugraph-ai/edugraph/internal/auth/repository"
	"github.com/edugraph-ai/edugraph/internal/auth/service"
	"github.com/edugraph-ai/edugraph/internal/testutil"
	"github.com/edugraph-ai/edugraph/pkg/crypto"
)

func newTestService(t *testing.T) *service.Service {
	t.Helper()
	pool := testutil.RequirePostgres(t)
	t.Cleanup(pool.Close)

	dir := t.TempDir()
	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")
	if err := crypto.EnsureDevKeyPair(privPath, pubPath); err != nil {
		t.Fatalf("generate test keypair: %v", err)
	}
	signer, err := crypto.NewJWTSigner(privPath, pubPath, 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("NewJWTSigner: %v", err)
	}

	repo := repository.New(pool)
	redis := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:0"}) // never dialed by anything auth.Service actually does
	return service.New(repo, signer, redis)
}

// uniqueEmail keeps repeated local test runs (against a persistent,
// non-CI database) from colliding on users.email's unique constraint.
func uniqueEmail(t *testing.T) string {
	t.Helper()
	return "test-" + uuid.NewString() + "@edugraph.et"
}

func TestRegisterAndLogin(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	email := uniqueEmail(t)

	registerResp, err := svc.Register(ctx, dto.RegisterRequest{
		Email:    email,
		Password: "testpass123",
		FullName: "Test User",
		Role:     "student",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if registerResp.User.Email != email {
		t.Errorf("registered user email = %q, want %q", registerResp.User.Email, email)
	}
	if registerResp.AccessToken == "" || registerResp.RefreshToken == "" {
		t.Error("Register did not issue both tokens")
	}
	// checklist 11.1: AuthResponse's token fields must never serialize to
	// JSON, even though the service layer still returns them internally
	// for the handler to set as cookies -- this is the actual defense,
	// not just handler discipline, see dto.AuthResponse's doc comment.
	body, err := json.Marshal(registerResp)
	if err != nil {
		t.Fatalf("marshal AuthResponse: %v", err)
	}
	if strings.Contains(string(body), registerResp.AccessToken) || strings.Contains(string(body), registerResp.RefreshToken) {
		t.Error("AuthResponse serialized a raw token into JSON -- json:\"-\" tag regressed")
	}

	loginResp, err := svc.Login(ctx, dto.LoginRequest{Email: email, Password: "testpass123"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if loginResp.User.ID != registerResp.User.ID {
		t.Errorf("Login returned a different user id than Register: %s vs %s", loginResp.User.ID, registerResp.User.ID)
	}

	if _, err := svc.Login(ctx, dto.LoginRequest{Email: email, Password: "wrong-password"}); err == nil {
		t.Error("Login succeeded with an incorrect password")
	}
	if _, err := svc.Login(ctx, dto.LoginRequest{Email: "no-such-user-" + uuid.NewString() + "@edugraph.et", Password: "testpass123"}); err == nil {
		t.Error("Login succeeded for a nonexistent email")
	}
}

func TestRefresh_RotatesAndSingleUses(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, dto.RegisterRequest{
		Email: uniqueEmail(t), Password: "testpass123", FullName: "Test User", Role: "student",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	refreshed, err := svc.Refresh(ctx, reg.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshed.RefreshToken == reg.RefreshToken {
		t.Error("Refresh returned the same refresh token instead of rotating it")
	}
	if refreshed.User.ID != reg.User.ID {
		t.Error("Refresh changed the user identity")
	}

	// checklist: refresh tokens are single-use -- reusing the original
	// (now-rotated-away) token must fail, exactly the property that
	// makes a stolen-and-later-replayed refresh token detectable.
	if _, err := svc.Refresh(ctx, reg.RefreshToken); err == nil {
		t.Error("Refresh accepted an already-rotated refresh token")
	}

	if _, err := svc.Refresh(ctx, "not-a-real-token"); err == nil {
		t.Error("Refresh accepted a garbage token")
	}
}

func TestLogout_RevokesRefreshToken(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, dto.RegisterRequest{
		Email: uniqueEmail(t), Password: "testpass123", FullName: "Test User", Role: "student",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := svc.Logout(ctx, reg.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// The whole point of Logout: the same refresh token must no longer
	// work afterward (this is the server-side half of the checklist
	// 11.1 fix -- clearing a client-side cookie alone can't revoke it).
	if _, err := svc.Refresh(ctx, reg.RefreshToken); err == nil {
		t.Error("Refresh succeeded with a token that was just logged out")
	}
}

func TestExpiresIn_ReflectsAccessTokenTTL(t *testing.T) {
	// Regression test for a real bug found and fixed alongside checklist
	// 11.1: ExpiresIn used to be computed from the refresh token's ~7-day
	// TTL instead of the access token's ~15-minute one.
	svc := newTestService(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, dto.RegisterRequest{
		Email: uniqueEmail(t), Password: "testpass123", FullName: "Test User", Role: "student",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	const accessTTLSeconds = 15 * 60
	if reg.ExpiresIn < accessTTLSeconds-5 || reg.ExpiresIn > accessTTLSeconds+5 {
		t.Errorf("ExpiresIn = %d, want ~%d (the access token's TTL, not the refresh token's)", reg.ExpiresIn, accessTTLSeconds)
	}
}
