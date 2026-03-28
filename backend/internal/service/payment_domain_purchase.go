package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"linuxdospace/backend/internal/linuxdocredit"
	"linuxdospace/backend/internal/model"
	"linuxdospace/backend/internal/storage"
)

const (
	// PaymentProductDomainAllocationPurchase is the stable internal product key
	// used by dynamic paid namespace orders. The administrator does not price it
	// from `payment_products`; per-root-domain pricing lives on `managed_domains`.
	PaymentProductDomainAllocationPurchase = "domain_allocation_purchase"

	// DomainPurchaseModeExact keeps the exact prefix selected by the buyer.
	DomainPurchaseModeExact = "exact"

	// DomainPurchaseModeRandom hides the final prefix until after payment and
	// then assigns one random 12+ character label.
	DomainPurchaseModeRandom = "random"
)

// CreateDomainPurchaseOrderRequest describes one dynamic paid namespace order
// initiated from the public domain search page.
type CreateDomainPurchaseOrderRequest struct {
	RootDomain   string `json:"root_domain"`
	Mode         string `json:"mode"`
	Prefix       string `json:"prefix"`
	RandomLength int    `json:"random_length"`
}

// CreateDomainPurchaseOrder reserves one local dynamic order for a paid
// namespace purchase and returns the upstream checkout URL.
func (s *PaymentService) CreateDomainPurchaseOrder(ctx context.Context, user model.User, request CreateDomainPurchaseOrderRequest) (model.PaymentOrder, error) {
	if s.credit == nil || !s.credit.Configured() || !s.cfg.LinuxDOCreditConfigured() {
		return model.PaymentOrder{}, UnavailableError("linux.do credit payment is not configured", nil)
	}

	managedDomain, normalizedMode, normalizedPrefix, requestedLength, totalPriceCents, title, err := s.validateDomainPurchaseOrder(ctx, user, request)
	if err != nil {
		return model.PaymentOrder{}, err
	}

	outTradeNo, err := generateOutTradeNo()
	if err != nil {
		return model.PaymentOrder{}, InternalError("failed to generate payment order number", err)
	}

	order, err := s.db.CreatePaymentOrder(ctx, storage.CreatePaymentOrderInput{
		UserID:                   user.ID,
		ProductKey:               PaymentProductDomainAllocationPurchase,
		ProductName:              "域名购买",
		Title:                    title,
		GatewayType:              model.PaymentGatewayLinuxDOCredit,
		OutTradeNo:               outTradeNo,
		Status:                   model.PaymentOrderStatusCreated,
		Units:                    1,
		GrantQuantity:            1,
		GrantedTotal:             1,
		GrantUnit:                "allocation",
		UnitPriceCents:           totalPriceCents,
		TotalPriceCents:          totalPriceCents,
		EffectType:               model.PaymentEffectDomainAllocationPurchase,
		PurchaseRootDomain:       managedDomain.RootDomain,
		PurchaseMode:             normalizedMode,
		PurchasePrefix:           strings.TrimSpace(request.Prefix),
		PurchaseNormalizedPrefix: normalizedPrefix,
		PurchaseRequestedLength:  requestedLength,
	})
	if err != nil {
		if isDomainPurchaseReservationConflictError(err) {
			return model.PaymentOrder{}, ConflictError("the requested namespace is already reserved by another active checkout")
		}
		return model.PaymentOrder{}, InternalError("failed to create local domain purchase order", err)
	}

	metadata, _ := json.Marshal(map[string]any{
		"out_trade_no":               order.OutTradeNo,
		"purchase_root_domain":       order.PurchaseRootDomain,
		"purchase_mode":              order.PurchaseMode,
		"purchase_normalized_prefix": order.PurchaseNormalizedPrefix,
		"purchase_requested_length":  order.PurchaseRequestedLength,
		"total_price_cents":          order.TotalPriceCents,
	})
	logAuditWriteFailure("payment.domain_order.create", s.db.WriteAuditLog(ctx, storage.AuditLogInput{
		ActorUserID:  &user.ID,
		Action:       "payment.domain_order.create",
		ResourceType: "payment_order",
		ResourceID:   order.OutTradeNo,
		MetadataJSON: string(metadata),
	}))

	submitResult, submitErr := s.credit.SubmitOrder(ctx, linuxDOCreditSubmitRequest(order))
	if submitErr != nil {
		now := time.Now().UTC()
		_, updateErr := s.db.UpdatePaymentOrderGatewayState(ctx, storage.UpdatePaymentOrderGatewayStateInput{
			OutTradeNo:    outTradeNo,
			Status:        model.PaymentOrderStatusFailed,
			LastCheckedAt: &now,
		})
		logPostMutationFailure("payment.domain_order.mark_failed", updateErr)
		return model.PaymentOrder{}, UnavailableError("failed to create linux.do credit checkout", submitErr)
	}

	order, err = s.db.UpdatePaymentOrderGatewayState(ctx, storage.UpdatePaymentOrderGatewayStateInput{
		OutTradeNo: outTradeNo,
		Status:     model.PaymentOrderStatusPending,
		PaymentURL: submitResult.PaymentURL,
	})
	if err != nil {
		return model.PaymentOrder{}, InternalError("failed to persist linux.do credit checkout url", err)
	}

	logAuditWriteFailure("payment.domain_order.checkout_ready", s.db.WriteAuditLog(ctx, storage.AuditLogInput{
		ActorUserID:  &user.ID,
		Action:       "payment.domain_order.checkout_ready",
		ResourceType: "payment_order",
		ResourceID:   order.OutTradeNo,
		MetadataJSON: fmt.Sprintf(`{"purchase_root_domain":"%s","purchase_mode":"%s","status":"%s"}`, order.PurchaseRootDomain, order.PurchaseMode, order.Status),
	}))

	return order, nil
}

// validateDomainPurchaseOrder resolves the root domain, normalizes the chosen
// mode, checks namespace availability, and calculates the final payable amount.
func (s *PaymentService) validateDomainPurchaseOrder(ctx context.Context, user model.User, request CreateDomainPurchaseOrderRequest) (model.ManagedDomain, string, string, int, int64, string, error) {
	managedDomain, err := s.db.GetManagedDomainByRoot(ctx, strings.ToLower(strings.TrimSpace(request.RootDomain)))
	if err != nil {
		if storage.IsNotFound(err) {
			return model.ManagedDomain{}, "", "", 0, 0, "", NotFoundError("managed domain not found")
		}
		return model.ManagedDomain{}, "", "", 0, 0, "", InternalError("failed to load managed domain", err)
	}
	if !managedDomain.Enabled {
		return model.ManagedDomain{}, "", "", 0, 0, "", ForbiddenError("the selected root domain is disabled")
	}
	if !managedDomain.SaleEnabled || managedDomain.SaleBasePriceCents <= 0 {
		return model.ManagedDomain{}, "", "", 0, 0, "", ForbiddenError("the selected root domain is not open for purchase yet")
	}

	switch normalizeDomainPurchaseMode(request.Mode) {
	case DomainPurchaseModeExact:
		normalizedPrefix, normalizeErr := NormalizePrefix(request.Prefix)
		if normalizeErr != nil {
			return model.ManagedDomain{}, "", "", 0, 0, "", ValidationError(normalizeErr.Error())
		}
		totalPriceCents, priceErr := calculateExactDomainPurchasePriceCents(managedDomain.SaleBasePriceCents, len(normalizedPrefix))
		if priceErr != nil {
			return model.ManagedDomain{}, "", "", 0, 0, "", ValidationError(priceErr.Error())
		}

		userPrefix, userPrefixErr := normalizedUserPrefix(user.Username)
		if userPrefixErr == nil && normalizedPrefix == userPrefix && managedDomain.AutoProvision {
			return model.ManagedDomain{}, "", "", 0, 0, "", ForbiddenError("this exact namespace is already eligible for the temporary free registration flow")
		}

		availability, availabilityErr := NewDomainService(s.cfg, s.db, s.cf).CheckAvailability(ctx, managedDomain.RootDomain, normalizedPrefix)
		if availabilityErr != nil {
			return model.ManagedDomain{}, "", "", 0, 0, "", availabilityErr
		}
		if !availability.Available {
			return model.ManagedDomain{}, "", "", 0, 0, "", ConflictError(readableDomainPurchaseAvailabilityMessage(availability.Reasons))
		}

		return managedDomain, DomainPurchaseModeExact, normalizedPrefix, len(normalizedPrefix), totalPriceCents, fmt.Sprintf("购买 %s", availability.FQDN), nil
	case DomainPurchaseModeRandom:
		requestedLength := request.RandomLength
		if requestedLength == 0 {
			requestedLength = 12
		}
		totalPriceCents, priceErr := calculateRandomDomainPurchasePriceCents(managedDomain.SaleBasePriceCents, requestedLength)
		if priceErr != nil {
			return model.ManagedDomain{}, "", "", 0, 0, "", ValidationError(priceErr.Error())
		}

		return managedDomain, DomainPurchaseModeRandom, "", requestedLength, totalPriceCents, fmt.Sprintf("购买随机 %d 位子域名 · %s", requestedLength, managedDomain.RootDomain), nil
	default:
		return model.ManagedDomain{}, "", "", 0, 0, "", ValidationError("mode must be exact or random")
	}
}

// normalizeDomainPurchaseMode keeps the public API strict and predictable.
func normalizeDomainPurchaseMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case DomainPurchaseModeExact:
		return DomainPurchaseModeExact
	case DomainPurchaseModeRandom:
		return DomainPurchaseModeRandom
	default:
		return ""
	}
}

// calculateExactDomainPurchasePriceCents applies the fixed length multiplier
// table for buyer-selected exact prefixes.
func calculateExactDomainPurchasePriceCents(basePriceCents int64, prefixLength int) (int64, error) {
	if basePriceCents <= 0 {
		return 0, fmt.Errorf("base price must be greater than 0")
	}

	switch prefixLength {
	case 0:
		return 0, fmt.Errorf("prefix is required")
	case 1:
		return 0, fmt.Errorf("1-character prefixes are not sold")
	case 2:
		return multiplyInt64Checked(basePriceCents, 15)
	case 3:
		return multiplyInt64Checked(basePriceCents, 10)
	case 4:
		return multiplyInt64Checked(basePriceCents, 5)
	case 5:
		return multiplyInt64Checked(basePriceCents, 2)
	default:
		return basePriceCents, nil
	}
}

// calculateRandomDomainPurchasePriceCents prices the hidden random 12+ label
// mode at half of the exact 6+ base price and rounds up to the nearest cent.
func calculateRandomDomainPurchasePriceCents(basePriceCents int64, requestedLength int) (int64, error) {
	if basePriceCents <= 0 {
		return 0, fmt.Errorf("base price must be greater than 0")
	}
	if requestedLength < 12 || requestedLength > 63 {
		return 0, fmt.Errorf("random mode only supports 12 to 63 characters")
	}
	return (basePriceCents + 1) / 2, nil
}

// readableDomainPurchaseAvailabilityMessage keeps checkout failures aligned
// with the same reason vocabulary used by the public search page.
func readableDomainPurchaseAvailabilityMessage(reasons []string) string {
	if len(reasons) == 0 {
		return "the requested namespace is not available"
	}
	if slicesContain(reasons, "reserved_in_database") {
		return "the requested namespace has already been reserved"
	}
	if slicesContain(reasons, "existing_dns_records") {
		return "live dns records already exist for the requested namespace"
	}
	if slicesContain(reasons, "reserved_dynamic_mail_namespace") || slicesContain(reasons, "reserved_mail_namespace") {
		return "the requested namespace is reserved for the platform-managed mail namespace"
	}
	return "the requested namespace is not available"
}

// linuxDOCreditSubmitRequest converts one local order into the small upstream
// payload expected by the Linux Do Credit client.
func linuxDOCreditSubmitRequest(order model.PaymentOrder) linuxdocredit.SubmitOrderRequest {
	return linuxdocredit.SubmitOrderRequest{
		OutTradeNo: order.OutTradeNo,
		Name:       order.Title,
		Money:      formatMoneyFromCents(order.TotalPriceCents),
	}
}

// slicesContain keeps tiny string membership checks readable without pulling
// in a broader helper dependency for one short list.
func slicesContain(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

// listLiveDomainPurchaseBlockedPrefixes collapses the current Cloudflare zone
// snapshot into the set of top-level namespace labels that are already in use
// under one managed root domain.
func (s *PaymentService) listLiveDomainPurchaseBlockedPrefixes(ctx context.Context, rootDomain string) ([]string, error) {
	managedDomain, err := s.db.GetManagedDomainByRoot(ctx, strings.ToLower(strings.TrimSpace(rootDomain)))
	if err != nil {
		if storage.IsNotFound(err) {
			return nil, NotFoundError("managed domain not found")
		}
		return nil, InternalError("failed to load managed domain", err)
	}
	if s.cf == nil || !s.cfg.CloudflareConfigured() || strings.TrimSpace(managedDomain.CloudflareZoneID) == "" {
		return nil, nil
	}

	records, err := s.cf.ListAllDNSRecords(ctx, managedDomain.CloudflareZoneID)
	if err != nil {
		return nil, UnavailableError("failed to list cloudflare dns records", err)
	}

	normalizedRoot := strings.ToLower(strings.TrimSpace(managedDomain.RootDomain))
	blocked := make(map[string]struct{}, len(records))
	for _, record := range records {
		recordName := strings.ToLower(strings.TrimSpace(record.Name))
		if recordName == normalizedRoot {
			continue
		}
		if !strings.HasSuffix(recordName, "."+normalizedRoot) {
			continue
		}

		relativeToRoot := strings.TrimSuffix(recordName, "."+normalizedRoot)
		relativeToRoot = strings.TrimSuffix(relativeToRoot, ".")
		if relativeToRoot == "" {
			continue
		}

		labels := strings.Split(relativeToRoot, ".")
		prefixCandidate := strings.TrimSpace(labels[len(labels)-1])
		if prefixCandidate == "" || prefixCandidate == "*" {
			continue
		}
		if normalizedPrefix, normalizeErr := NormalizePrefix(prefixCandidate); normalizeErr == nil {
			blocked[normalizedPrefix] = struct{}{}
		}
	}

	values := make([]string, 0, len(blocked))
	for value := range blocked {
		values = append(values, value)
	}
	return values, nil
}

// listReservedDynamicMailAliasUserPrefixes returns every currently known user
// prefix whose derived `-mail` namespace family must stay unsellable under the
// configured default root at final payment-apply time as well.
func (s *PaymentService) listReservedDynamicMailAliasUserPrefixes(ctx context.Context, rootDomain string) ([]string, error) {
	if !strings.EqualFold(strings.TrimSpace(rootDomain), strings.TrimSpace(s.cfg.Cloudflare.DefaultRootDomain)) {
		return nil, nil
	}

	users, err := s.db.ListAdminUsers(ctx)
	if err != nil {
		return nil, InternalError("failed to load user list for dynamic mail namespace reservation", err)
	}

	seen := make(map[string]struct{}, len(users))
	values := make([]string, 0, len(users))
	for _, user := range users {
		normalizedPrefix, normalizeErr := normalizedUserPrefix(user.Username)
		if normalizeErr != nil || normalizedPrefix == "" {
			continue
		}
		if _, exists := seen[normalizedPrefix]; exists {
			continue
		}
		seen[normalizedPrefix] = struct{}{}
		values = append(values, normalizedPrefix)
	}
	sort.Strings(values)
	return values, nil
}

// isDomainPurchaseReservationConflictError normalizes database-specific unique
// failures so the public checkout endpoint can return one stable 409 outcome.
func isDomainPurchaseReservationConflictError(err error) bool {
	if err == nil {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(normalized, "unique") ||
		strings.Contains(normalized, "constraint failed") ||
		strings.Contains(normalized, "duplicate key")
}
