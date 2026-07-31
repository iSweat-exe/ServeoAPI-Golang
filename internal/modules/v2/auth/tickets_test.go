package auth

import (
	"testing"
	"time"
)

func TestConsumeTicketIsSingleUse(t *testing.T) {
	ticket, err := GenerateTicket("jwt-token")
	if err != nil {
		t.Fatalf("GenerateTicket returned an error: %v", err)
	}

	token, ok := ConsumeTicket(ticket)
	if !ok || token != "jwt-token" {
		t.Fatalf("first ConsumeTicket = (%q, %v), want (%q, true)", token, ok, "jwt-token")
	}

	if _, ok := ConsumeTicket(ticket); ok {
		t.Fatal("a ticket must not be reusable")
	}
}

func TestGenerateTicketIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		ticket, err := GenerateTicket("jwt-token")
		if err != nil {
			t.Fatalf("GenerateTicket returned an error: %v", err)
		}
		if len(ticket) != 32 {
			t.Fatalf("ticket length = %d, want 32 hex chars", len(ticket))
		}
		if _, dup := seen[ticket]; dup {
			t.Fatalf("duplicate ticket generated: %q", ticket)
		}
		seen[ticket] = struct{}{}
	}
}

func TestConsumeExpiredTicket(t *testing.T) {
	ticketStore.Store("expired", ticketEntry{
		Token:     "jwt-token",
		ExpiresAt: time.Now().Add(-time.Second),
	})

	if _, ok := ConsumeTicket("expired"); ok {
		t.Fatal("an expired ticket must be rejected")
	}
}

func TestConsumeUnknownTicket(t *testing.T) {
	if _, ok := ConsumeTicket("does-not-exist"); ok {
		t.Fatal("an unknown ticket must be rejected")
	}
}
