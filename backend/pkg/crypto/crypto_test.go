package crypto

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestSigner(t *testing.T) *JWTSigner {
	t.Helper()
	dir := t.TempDir()
	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")
	if err := EnsureDevKeyPair(privPath, pubPath); err != nil {
		t.Fatalf("generate test keypair: %v", err)
	}
	signer, err := NewJWTSigner(privPath, pubPath, 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("NewJWTSigner: %v", err)
	}
	return signer
}

func TestJWTSigner_IssueAndVerify(t *testing.T) {
	signer := newTestSigner(t)

	tests := []struct {
		name   string
		userID string
		role   string
		issue  func(userID, role string) (string, error)
	}{
		{"access token", "user-1", "student", signer.IssueAccessToken},
		{"refresh token", "user-2", "teacher", signer.IssueRefreshToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tt.issue(tt.userID, tt.role)
			if err != nil {
				t.Fatalf("issue: %v", err)
			}
			if token == "" {
				t.Fatal("issued token is empty")
			}

			claims, err := signer.Verify(token)
			if err != nil {
				t.Fatalf("verify: %v", err)
			}
			if claims.UserID != tt.userID {
				t.Errorf("UserID = %q, want %q", claims.UserID, tt.userID)
			}
			if claims.Role != tt.role {
				t.Errorf("Role = %q, want %q", claims.Role, tt.role)
			}
		})
	}
}

func TestJWTSigner_Verify_RejectsExpiredToken(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")
	if err := EnsureDevKeyPair(privPath, pubPath); err != nil {
		t.Fatalf("generate test keypair: %v", err)
	}
	// Negative TTL: the token is already expired the instant it's issued.
	signer, err := NewJWTSigner(privPath, pubPath, -1*time.Minute, -1*time.Minute)
	if err != nil {
		t.Fatalf("NewJWTSigner: %v", err)
	}

	token, err := signer.IssueAccessToken("user-1", "student")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := signer.Verify(token); err == nil {
		t.Fatal("Verify accepted an already-expired token, want an error")
	}
}

func TestJWTSigner_Verify_RejectsTokenFromDifferentSigner(t *testing.T) {
	signerA := newTestSigner(t)
	signerB := newTestSigner(t) // different RSA keypair

	token, err := signerA.IssueAccessToken("user-1", "student")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	// A token signed by A's private key must not verify against B's
	// public key -- this is the actual security property RS256 buys
	// over a shared-secret scheme (a compromised verifier can't forge
	// tokens the signer would accept).
	if _, err := signerB.Verify(token); err == nil {
		t.Fatal("Verify accepted a token signed by a different keypair, want an error")
	}
}

func TestJWTSigner_Verify_RejectsTamperedToken(t *testing.T) {
	signer := newTestSigner(t)

	token, err := signer.IssueAccessToken("user-1", "student")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	// Flip a character in the middle of the payload segment, not the
	// token's literal last character: an RSA signature's base64url tail
	// sits at a padding-bit boundary (a 256-byte/2048-bit signature's
	// final byte encodes as 2 base64 characters, the second of which
	// only carries 2 significant bits), so changing the very last
	// character can decode to byte-identical signature bytes depending
	// on which random keypair a given test run generated -- this isn't
	// hypothetical, it flaked exactly this way in a real run. A mid-
	// payload character has no such ambiguity.
	mid := len(token) / 2
	flip := byte('x')
	if token[mid] == flip {
		flip = 'y'
	}
	tampered := token[:mid] + string(flip) + token[mid+1:]
	if tampered == token {
		t.Fatal("test setup produced an unmodified token")
	}

	if _, err := signer.Verify(tampered); err == nil {
		t.Fatal("Verify accepted a tampered token, want an error")
	}
}

// TestIssueAccessToken_UniqueEvenWithinSameSecond is a regression test:
// RS256 is a deterministic signature scheme and IssuedAt/ExpiresAt only
// carry second-granularity precision, so two tokens issued for the same
// user back-to-back used to come out byte-identical -- which collided
// on refresh_tokens.token_hash's unique constraint in production (found
// via internal/auth/service's integration test hitting Register then
// Login back-to-back, not a hypothetical). The jti (RegisteredClaims.ID)
// fix is what this asserts.
func TestIssueAccessToken_UniqueEvenWithinSameSecond(t *testing.T) {
	signer := newTestSigner(t)

	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		token, err := signer.IssueAccessToken("same-user", "student")
		if err != nil {
			t.Fatalf("issue #%d: %v", i, err)
		}
		if seen[token] {
			t.Fatalf("issue #%d produced a duplicate token -- jti uniqueness regressed", i)
		}
		seen[token] = true
	}
}

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "correct-horse-battery-staple" {
		t.Fatal("HashPassword returned the plaintext password unchanged")
	}

	if err := ComparePassword(hash, "correct-horse-battery-staple"); err != nil {
		t.Errorf("ComparePassword rejected the correct password: %v", err)
	}
	if err := ComparePassword(hash, "wrong-password"); err == nil {
		t.Error("ComparePassword accepted an incorrect password")
	}
}
