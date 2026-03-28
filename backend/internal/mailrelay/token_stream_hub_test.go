package mailrelay

import "testing"

// TestTokenStreamHubAllowsOnlyOneActiveSubscription verifies that a single API
// token can own only one live NDJSON stream at a time.
func TestTokenStreamHubAllowsOnlyOneActiveSubscription(t *testing.T) {
	hub := NewTokenStreamHub()

	first, err := hub.Subscribe("ldt_token")
	if err != nil {
		t.Fatalf("subscribe first token stream: %v", err)
	}
	defer first.Cancel()

	second, err := hub.Subscribe("ldt_token")
	if err != ErrTokenStreamAlreadyConnected {
		t.Fatalf("expected ErrTokenStreamAlreadyConnected, got subscription=%v err=%v", second, err)
	}
}

// TestTokenStreamHubDisconnectTokenClosesSubscription verifies that token
// revocation can actively terminate the live stream.
func TestTokenStreamHubDisconnectTokenClosesSubscription(t *testing.T) {
	hub := NewTokenStreamHub()

	subscription, err := hub.Subscribe("ldt_token")
	if err != nil {
		t.Fatalf("subscribe token stream: %v", err)
	}

	if disconnected := hub.DisconnectToken("ldt_token"); disconnected != 1 {
		t.Fatalf("expected one disconnected subscriber, got %d", disconnected)
	}

	select {
	case <-subscription.Done():
	default:
		t.Fatalf("expected subscription done channel to be closed after disconnect")
	}
}

// TestTokenStreamHubTracksMailboxDomains verifies that one live token stream
// can register dynamic mailbox domains and that disconnect cleans them up.
func TestTokenStreamHubTracksMailboxDomains(t *testing.T) {
	hub := NewTokenStreamHub()

	subscription, err := hub.Subscribe("ldt_token")
	if err != nil {
		t.Fatalf("subscribe token stream: %v", err)
	}
	defer subscription.Cancel()

	if err := hub.UpdateMailboxDomains("ldt_token", 42, "alice", []string{
		"alice-mailfoo.linuxdo.space",
		"alice-mail.linuxdo.space",
	}); err != nil {
		t.Fatalf("update mailbox domains: %v", err)
	}

	item, ok := hub.LookupMailboxDomain("alice-mailfoo.linuxdo.space")
	if !ok {
		t.Fatalf("expected dynamic mailbox domain lookup to succeed")
	}
	if item.TokenPublicID != "ldt_token" || item.OwnerUserID != 42 || item.OwnerUsername != "alice" {
		t.Fatalf("unexpected mailbox domain owner info: %+v", item)
	}

	if disconnected := hub.DisconnectToken("ldt_token"); disconnected != 1 {
		t.Fatalf("expected one disconnected subscriber, got %d", disconnected)
	}
	if _, ok := hub.LookupMailboxDomain("alice-mailfoo.linuxdo.space"); ok {
		t.Fatalf("expected dynamic mailbox domain to be cleared after disconnect")
	}
}

// TestTokenStreamHubRejectsMailboxDomainConflicts verifies that two live API
// tokens cannot claim the same dynamic mailbox domain at the same time.
func TestTokenStreamHubRejectsMailboxDomainConflicts(t *testing.T) {
	hub := NewTokenStreamHub()

	first, err := hub.Subscribe("ldt_first")
	if err != nil {
		t.Fatalf("subscribe first token stream: %v", err)
	}
	defer first.Cancel()
	second, err := hub.Subscribe("ldt_second")
	if err != nil {
		t.Fatalf("subscribe second token stream: %v", err)
	}
	defer second.Cancel()

	if err := hub.UpdateMailboxDomains("ldt_first", 1, "alice", []string{"alice-mailfoo.linuxdo.space"}); err != nil {
		t.Fatalf("register first dynamic mailbox domain: %v", err)
	}
	if err := hub.UpdateMailboxDomains("ldt_second", 2, "bob", []string{"alice-mailfoo.linuxdo.space"}); err != ErrTokenStreamMailboxDomainConflict {
		t.Fatalf("expected ErrTokenStreamMailboxDomainConflict, got %v", err)
	}
}
