package auth

import (
	"testing"
	"time"
)

func TestTokenRoundTrip(t *testing.T) {
	ti := NewTokenIssuer("test-secret-test-secret-test-secret!", time.Hour)
	tok, err := ti.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := ti.Parse(tok)
	if err != nil || id != 42 {
		t.Fatalf("Parse = %d, %v", id, err)
	}
	other := NewTokenIssuer("another-secret-another-secret-12345", time.Hour)
	if _, err := other.Parse(tok); err == nil {
		t.Fatal("token signed with a different secret must be rejected")
	}
	expired := NewTokenIssuer("test-secret-test-secret-test-secret!", -time.Minute)
	tok, _ = expired.Issue(1)
	if _, err := ti.Parse(tok); err == nil {
		t.Fatal("expired token must be rejected")
	}
}
