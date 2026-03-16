package service

import (
	"context"
	"net/url"
	"testing"

	"linuxdospace/backend/internal/cloudflare"
	"linuxdospace/backend/internal/config"
	"linuxdospace/backend/internal/linuxdocredit"
	"linuxdospace/backend/internal/storage"
	"linuxdospace/backend/internal/storage/sqlite"
)

// fakeConfiguredLinuxDOCredit keeps payment-service tests focused on business
// rules that run before the gateway request is attempted.
type fakeConfiguredLinuxDOCredit struct{}

// Configured reports a fully enabled fake gateway client.
func (fakeConfiguredLinuxDOCredit) Configured() bool {
	return true
}

// SubmitOrder should stay unreachable in tests that reject requests before the
// gateway is contacted.
func (fakeConfiguredLinuxDOCredit) SubmitOrder(ctx context.Context, request linuxdocredit.SubmitOrderRequest) (linuxdocredit.SubmitOrderResult, error) {
	return linuxdocredit.SubmitOrderResult{}, nil
}

// QueryOrder is unused by the current unit tests.
func (fakeConfiguredLinuxDOCredit) QueryOrder(ctx context.Context, outTradeNo string) (linuxdocredit.QueryOrderResult, error) {
	return linuxdocredit.QueryOrderResult{}, nil
}

// VerifyNotification is unused by the current unit tests.
func (fakeConfiguredLinuxDOCredit) VerifyNotification(values url.Values) (linuxdocredit.Notification, error) {
	return linuxdocredit.Notification{}, nil
}

// TestListPublicProductsHidesInternalDomainPurchaseProduct verifies that the
// dedicated dynamic domain-purchase product never leaks into the public Linux
// Do Credit product list, even if an operator mistakenly enables it.
func TestListPublicProductsHidesInternalDomainPurchaseProduct(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)

	product, err := store.GetPaymentProduct(ctx, PaymentProductDomainAllocationPurchase)
	if err != nil {
		t.Fatalf("load internal domain purchase product: %v", err)
	}
	if _, err := store.UpsertPaymentProduct(ctx, sqlite.UpsertPaymentProductInput{
		Key:            product.Key,
		DisplayName:    product.DisplayName,
		Description:    product.Description,
		Enabled:        true,
		UnitPriceCents: product.UnitPriceCents,
		GrantQuantity:  product.GrantQuantity,
		GrantUnit:      product.GrantUnit,
		EffectType:     product.EffectType,
		SortOrder:      product.SortOrder,
	}); err != nil {
		t.Fatalf("enable internal domain purchase product: %v", err)
	}

	service := NewPaymentService(config.Config{}, store, nil, fakeConfiguredLinuxDOCredit{})
	items, err := service.ListPublicProducts(ctx)
	if err != nil {
		t.Fatalf("list public payment products: %v", err)
	}
	for _, item := range items {
		if item.Key == PaymentProductDomainAllocationPurchase {
			t.Fatalf("expected internal domain purchase product to stay hidden from public products")
		}
	}
}

// TestCreateOrderRejectsInternalDomainPurchaseProduct verifies that even if an
// operator enables the internal product row, the generic order API still
// refuses to create checkout orders without the dedicated purchase context.
func TestCreateOrderRejectsInternalDomainPurchaseProduct(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user, err := store.UpsertUser(ctx, storage.UpsertUserInput{
		LinuxDOUserID: 2001,
		Username:      "buyer",
		DisplayName:   "Buyer",
		AvatarURL:     "https://example.com/avatar.png",
		TrustLevel:    2,
	})
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}

	product, err := store.GetPaymentProduct(ctx, PaymentProductDomainAllocationPurchase)
	if err != nil {
		t.Fatalf("load internal domain purchase product: %v", err)
	}
	if _, err := store.UpsertPaymentProduct(ctx, sqlite.UpsertPaymentProductInput{
		Key:            product.Key,
		DisplayName:    product.DisplayName,
		Description:    product.Description,
		Enabled:        true,
		UnitPriceCents: product.UnitPriceCents,
		GrantQuantity:  product.GrantQuantity,
		GrantUnit:      product.GrantUnit,
		EffectType:     product.EffectType,
		SortOrder:      product.SortOrder,
	}); err != nil {
		t.Fatalf("enable internal domain purchase product: %v", err)
	}

	service := NewPaymentService(config.Config{
		LinuxDOCredit: config.LinuxDOCreditConfig{
			PID: "pid",
			Key: "key",
		},
	}, store, nil, fakeConfiguredLinuxDOCredit{})
	if _, err := service.CreateOrder(ctx, user, CreatePaymentOrderRequest{
		ProductKey: PaymentProductDomainAllocationPurchase,
		Units:      1,
	}); err == nil {
		t.Fatalf("expected generic payment order creation to reject the internal domain purchase product")
	} else if normalized := NormalizeError(err); normalized.StatusCode != 403 {
		t.Fatalf("expected forbidden error, got %+v", normalized)
	}
}

// TestListLiveDomainPurchaseBlockedPrefixes verifies that the Cloudflare DNS
// snapshot is reduced to the real second-level namespace labels already in use
// under the selected root domain.
func TestListLiveDomainPurchaseBlockedPrefixes(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)

	if _, err := store.UpsertManagedDomain(ctx, sqlite.UpsertManagedDomainInput{
		RootDomain:         "linuxdo.space",
		CloudflareZoneID:   "zone-default",
		DefaultQuota:       1,
		AutoProvision:      true,
		IsDefault:          true,
		Enabled:            true,
		SaleEnabled:        false,
		SaleBasePriceCents: 1000,
	}); err != nil {
		t.Fatalf("upsert managed domain: %v", err)
	}

	cloudflareClient := &fakeEmailRoutingCloudflare{
		dnsRecordsByZone: map[string][]cloudflare.DNSRecord{
			"zone-default": {
				{ID: "1", Name: "linuxdo.space", Type: "TXT", Content: "root"},
				{ID: "2", Name: "mail.linuxdo.space", Type: "A", Content: "1.1.1.1"},
				{ID: "3", Name: "api.chat.linuxdo.space", Type: "CNAME", Content: "example.com"},
				{ID: "4", Name: "*.bot.linuxdo.space", Type: "TXT", Content: "wildcard"},
			},
		},
	}

	service := NewPaymentService(config.Config{
		Cloudflare: config.CloudflareConfig{APIToken: "token"},
	}, store, cloudflareClient, nil)
	blocked, err := service.listLiveDomainPurchaseBlockedPrefixes(ctx, "linuxdo.space")
	if err != nil {
		t.Fatalf("list blocked purchase prefixes: %v", err)
	}

	blockedSet := make(map[string]struct{}, len(blocked))
	for _, value := range blocked {
		blockedSet[value] = struct{}{}
	}
	for _, expected := range []string{"mail", "chat", "bot"} {
		if _, ok := blockedSet[expected]; !ok {
			t.Fatalf("expected blocked prefixes to contain %q, got %v", expected, blocked)
		}
	}
	if _, ok := blockedSet["linuxdo"]; ok {
		t.Fatalf("root-domain records should not create a fake blocked namespace: %v", blocked)
	}
}

// TestEnsureBuiltInManagedDomainsDoesNotOverrideExistingConfiguration verifies
// that restart-time bootstrapping only inserts missing domains and never
// rewrites administrator-managed pricing or enable flags.
func TestEnsureBuiltInManagedDomainsDoesNotOverrideExistingConfiguration(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)

	existing, err := store.UpsertManagedDomain(ctx, sqlite.UpsertManagedDomainInput{
		RootDomain:         "cifang.love",
		CloudflareZoneID:   "zone-cifang",
		DefaultQuota:       9,
		AutoProvision:      true,
		IsDefault:          false,
		Enabled:            false,
		SaleEnabled:        true,
		SaleBasePriceCents: 2500,
	})
	if err != nil {
		t.Fatalf("seed existing managed domain: %v", err)
	}

	service := NewDomainService(config.Config{
		Cloudflare: config.CloudflareConfig{APIToken: "token"},
	}, store, &fakeEmailRoutingCloudflare{
		zoneIDsByRoot: map[string]string{
			"openapi.best": "zone-openapi",
			"metapi.cc":    "zone-metapi",
		},
	})
	if err := service.EnsureBuiltInManagedDomains(ctx); err != nil {
		t.Fatalf("ensure built-in managed domains: %v", err)
	}

	reloaded, err := store.GetManagedDomainByRoot(ctx, existing.RootDomain)
	if err != nil {
		t.Fatalf("reload existing managed domain: %v", err)
	}
	if reloaded.DefaultQuota != existing.DefaultQuota || reloaded.AutoProvision != existing.AutoProvision || reloaded.Enabled != existing.Enabled || reloaded.SaleEnabled != existing.SaleEnabled || reloaded.SaleBasePriceCents != existing.SaleBasePriceCents {
		t.Fatalf("expected existing managed domain to stay unchanged, got %+v", reloaded)
	}
}

// TestEnsureBuiltInManagedDomainsSkipsMissingOptionalZones verifies that one
// not-yet-delegated optional sale root does not break backend startup.
func TestEnsureBuiltInManagedDomainsSkipsMissingOptionalZones(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)

	service := NewDomainService(config.Config{
		Cloudflare: config.CloudflareConfig{APIToken: "token"},
	}, store, &fakeEmailRoutingCloudflare{
		zoneIDsByRoot: map[string]string{},
	})
	if err := service.EnsureBuiltInManagedDomains(ctx); err != nil {
		t.Fatalf("expected missing optional built-in zones to be skipped, got %v", err)
	}
}
