package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"linuxdospace/backend/internal/config"
	"linuxdospace/backend/internal/mailrelay"
	"linuxdospace/backend/internal/model"
	"linuxdospace/backend/internal/security"
	"linuxdospace/backend/internal/storage"
)

const (
	// APITokenTargetTypeEmail means the route forwards to one verified external mailbox.
	APITokenTargetTypeEmail = model.EmailRouteTargetKindEmail

	// APITokenTargetTypeAPIToken means the route forwards to one live API token stream.
	APITokenTargetTypeAPIToken = model.EmailRouteTargetKindAPIToken

	// maxDynamicMailboxSuffixesPerToken bounds one live token stream's dynamic
	// mailbox-domain filter set so a single bearer token cannot pin unbounded
	// in-memory suffix state into the relay.
	maxDynamicMailboxSuffixesPerToken = 256
)

// UserAPITokenView is the public frontend representation of one user-managed token.
type UserAPITokenView struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	PublicID     string     `json:"public_id"`
	Scopes       []string   `json:"scopes"`
	EmailEnabled bool       `json:"email_enabled"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

// CreateMyAPITokenRequest describes one new token request from the public frontend.
type CreateMyAPITokenRequest struct {
	Name         string `json:"name"`
	EmailEnabled bool   `json:"email_enabled"`
}

// CreateMyAPITokenResult returns the persisted token metadata and the one-time raw secret.
type CreateMyAPITokenResult struct {
	Token    UserAPITokenView `json:"token"`
	RawToken string           `json:"raw_token"`
}

// UpdateTokenEmailFiltersRequest replaces the currently active dynamic mailbox
// suffix-fragment set for one live API-token email stream connection.
type UpdateTokenEmailFiltersRequest struct {
	Suffixes []string `json:"suffixes"`
}

// UpdateTokenEmailFiltersResult returns the normalized suffix fragments and the
// fully derived mailbox domains now active for this token stream.
type UpdateTokenEmailFiltersResult struct {
	Suffixes []string `json:"suffixes"`
	Domains  []string `json:"domains"`
}

// TokenService owns user-managed API tokens and the live email stream hub.
type TokenService struct {
	cfg config.Config
	db  Store
	hub *mailrelay.TokenStreamHub
}

// NewTokenService creates the user-facing token-management service.
func NewTokenService(cfg config.Config, db Store, hub *mailrelay.TokenStreamHub) *TokenService {
	return &TokenService{cfg: cfg, db: db, hub: hub}
}

// ListMyAPITokens returns the current user's tokens, newest first.
func (s *TokenService) ListMyAPITokens(ctx context.Context, user model.User) ([]UserAPITokenView, error) {
	items, err := s.db.ListAPITokensByOwner(ctx, user.ID)
	if err != nil {
		return nil, InternalError("failed to list api tokens", err)
	}

	views := make([]UserAPITokenView, 0, len(items))
	for _, item := range items {
		views = append(views, userAPITokenFromModel(item))
	}
	return views, nil
}

// CreateMyAPIToken issues one new bearer token for the current user.
func (s *TokenService) CreateMyAPIToken(ctx context.Context, user model.User, request CreateMyAPITokenRequest) (CreateMyAPITokenResult, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return CreateMyAPITokenResult{}, ValidationError("token name must not be empty")
	}
	if len([]rune(name)) > 64 {
		return CreateMyAPITokenResult{}, ValidationError("token name must be 64 characters or fewer")
	}

	scopes := []string{model.APITokenScopeEmail}

	publicSuffix, err := security.RandomToken(8)
	if err != nil {
		return CreateMyAPITokenResult{}, InternalError("failed to generate token public id", err)
	}
	secret, err := security.RandomToken(32)
	if err != nil {
		return CreateMyAPITokenResult{}, InternalError("failed to generate token secret", err)
	}

	publicID := "ldt_" + strings.ToLower(strings.Trim(strings.ReplaceAll(publicSuffix, "-", ""), "_"))
	rawToken := "lds_pat_" + publicID + "." + secret
	tokenHash := hashAPIToken(rawToken)

	item, err := s.db.CreateAPIToken(ctx, storage.CreateAPITokenInput{
		OwnerUserID: user.ID,
		Name:        name,
		PublicID:    publicID,
		TokenHash:   tokenHash,
		Scopes:      scopes,
	})
	if err != nil {
		return CreateMyAPITokenResult{}, InternalError("failed to create api token", err)
	}

	metadata, _ := json.Marshal(map[string]any{
		"api_token_id": item.ID,
		"public_id":    item.PublicID,
		"scopes":       item.Scopes,
	})
	logAuditWriteFailure("api_token.create", s.db.WriteAuditLog(ctx, storage.AuditLogInput{
		ActorUserID:  &user.ID,
		Action:       "api_token.create",
		ResourceType: "api_token",
		ResourceID:   item.PublicID,
		MetadataJSON: string(metadata),
	}))

	return CreateMyAPITokenResult{
		Token:    userAPITokenFromModel(item),
		RawToken: rawToken,
	}, nil
}

// RevokeMyAPIToken disables one existing token owned by the current user.
func (s *TokenService) RevokeMyAPIToken(ctx context.Context, user model.User, publicID string) (UserAPITokenView, error) {
	item, err := s.db.GetAPITokenByPublicID(ctx, strings.TrimSpace(publicID))
	if err != nil {
		if storage.IsNotFound(err) {
			return UserAPITokenView{}, NotFoundError("api token not found")
		}
		return UserAPITokenView{}, InternalError("failed to load api token", err)
	}
	if item.OwnerUserID != user.ID {
		return UserAPITokenView{}, NotFoundError("api token not found")
	}
	if item.RevokedAt != nil {
		return userAPITokenFromModel(item), nil
	}

	now := time.Now().UTC()
	item, err = s.db.UpdateAPIToken(ctx, storage.UpdateAPITokenInput{
		ID:        item.ID,
		RevokedAt: &now,
	})
	if err != nil {
		return UserAPITokenView{}, InternalError("failed to revoke api token", err)
	}

	metadata, _ := json.Marshal(map[string]any{
		"api_token_id": item.ID,
		"public_id":    item.PublicID,
	})
	logAuditWriteFailure("api_token.revoke", s.db.WriteAuditLog(ctx, storage.AuditLogInput{
		ActorUserID:  &user.ID,
		Action:       "api_token.revoke",
		ResourceType: "api_token",
		ResourceID:   item.PublicID,
		MetadataJSON: string(metadata),
	}))
	if s.hub != nil {
		s.hub.DisconnectToken(item.PublicID)
	}

	return userAPITokenFromModel(item), nil
}

// AuthenticateEmailStreamToken validates one raw bearer token for the live email stream.
func (s *TokenService) AuthenticateEmailStreamToken(ctx context.Context, rawToken string) (model.APIToken, error) {
	tokenHash := hashAPIToken(rawToken)
	item, err := s.db.GetAPITokenByTokenHash(ctx, tokenHash)
	if err != nil {
		if storage.IsNotFound(err) {
			return model.APIToken{}, UnauthorizedError("invalid api token")
		}
		return model.APIToken{}, InternalError("failed to load api token", err)
	}
	if item.RevokedAt != nil {
		return model.APIToken{}, UnauthorizedError("api token has been revoked")
	}
	if !apiTokenHasScope(item, model.APITokenScopeEmail) {
		return model.APIToken{}, ForbiddenError("api token does not allow email streaming")
	}
	control, err := s.db.GetUserControlByUserID(ctx, item.OwnerUserID)
	if err == nil && control.IsBanned {
		return model.APIToken{}, ForbiddenError("api token owner is banned")
	}
	if err != nil && !storage.IsNotFound(err) {
		return model.APIToken{}, InternalError("failed to load api token owner control", err)
	}

	now := time.Now().UTC()
	if _, err = s.db.UpdateAPIToken(ctx, storage.UpdateAPITokenInput{
		ID:         item.ID,
		LastUsedAt: &now,
	}); err != nil {
		return model.APIToken{}, InternalError("failed to update api token last_used_at", err)
	}
	item.LastUsedAt = &now
	return item, nil
}

// Hub returns the live email-stream broker used by the HTTP stream endpoint and SMTP relay.
func (s *TokenService) Hub() *mailrelay.TokenStreamHub {
	return s.hub
}

// ResolveStreamOwnerUsername returns the normalized Linux Do username that
// owns the currently authenticated API token stream.
func (s *TokenService) ResolveStreamOwnerUsername(ctx context.Context, token model.APIToken) (string, error) {
	user, err := s.db.GetUserByID(ctx, token.OwnerUserID)
	if err != nil {
		if storage.IsNotFound(err) {
			return "", NotFoundError("api token owner not found")
		}
		return "", InternalError("failed to load api token owner", err)
	}
	return strings.ToLower(strings.TrimSpace(user.Username)), nil
}

// UpdateEmailStreamMailboxFilters replaces the active dynamic mailbox-domain
// registrations attached to one live token stream connection.
func (s *TokenService) UpdateEmailStreamMailboxFilters(ctx context.Context, token model.APIToken, ownerUsername string, request UpdateTokenEmailFiltersRequest) (UpdateTokenEmailFiltersResult, error) {
	if s == nil || s.hub == nil {
		return UpdateTokenEmailFiltersResult{}, UnavailableError("api token stream is not configured", nil)
	}

	defaultRootDomain := strings.ToLower(strings.TrimSpace(s.cfg.Cloudflare.DefaultRootDomain))
	if defaultRootDomain == "" {
		return UpdateTokenEmailFiltersResult{}, UnavailableError("default root domain is not configured for dynamic api-token mailbox suffixes", nil)
	}

	normalizedOwnerUsername, err := normalizedUserPrefix(ownerUsername)
	if err != nil {
		return UpdateTokenEmailFiltersResult{}, ValidationError("api token owner username cannot be mapped to a valid dynamic mailbox prefix")
	}

	managedDomain, err := s.db.GetManagedDomainByRoot(ctx, defaultRootDomain)
	if err != nil {
		if storage.IsNotFound(err) {
			return UpdateTokenEmailFiltersResult{}, UnavailableError("default root domain is not available for dynamic api-token mailbox suffixes", err)
		}
		return UpdateTokenEmailFiltersResult{}, InternalError("failed to load default root domain for dynamic api-token mailbox suffixes", err)
	}
	if !managedDomain.Enabled {
		return UpdateTokenEmailFiltersResult{}, ForbiddenError("default root domain is disabled for dynamic api-token mailbox suffixes")
	}

	if len(request.Suffixes) > 0 {
		ownerUser, err := s.db.GetUserByID(ctx, token.OwnerUserID)
		if err != nil {
			if storage.IsNotFound(err) {
				return UpdateTokenEmailFiltersResult{}, NotFoundError("api token owner not found")
			}
			return UpdateTokenEmailFiltersResult{}, InternalError("failed to load api token owner", err)
		}
		permissionService := NewPermissionService(s.cfg, s.db, nil)
		permission, err := permissionService.loadEmailCatchAllPermission(ctx, ownerUser)
		if err != nil {
			return UpdateTokenEmailFiltersResult{}, err
		}
		if permission.Status != "approved" {
			return UpdateTokenEmailFiltersResult{}, ForbiddenError("the catch-all email permission has not been approved")
		}
		if permission.CatchAllAccess == nil || !permission.CatchAllAccess.DeliveryAvailable {
			return UpdateTokenEmailFiltersResult{}, ConflictError("catch-all delivery is currently unavailable for this user")
		}
	}

	normalizedSuffixes := make([]string, 0, len(request.Suffixes))
	seenSuffixes := make(map[string]struct{}, len(request.Suffixes))
	domains := make([]string, 0, len(request.Suffixes))
	for _, rawSuffix := range request.Suffixes {
		normalizedSuffix, suffixErr := normalizeTokenMailboxSuffixFragment(rawSuffix)
		if suffixErr != nil {
			return UpdateTokenEmailFiltersResult{}, suffixErr
		}
		if _, exists := seenSuffixes[normalizedSuffix]; exists {
			continue
		}
		seenSuffixes[normalizedSuffix] = struct{}{}
		if len(seenSuffixes) > maxDynamicMailboxSuffixesPerToken {
			return UpdateTokenEmailFiltersResult{}, ValidationError("too many dynamic mailbox suffixes in one request")
		}
		normalizedSuffixes = append(normalizedSuffixes, normalizedSuffix)

		label := normalizedOwnerUsername + catchAllNamespaceSuffix + normalizedSuffix
		if len(label) > 63 {
			return UpdateTokenEmailFiltersResult{}, ValidationError("dynamic mailbox suffix is too long for one dns label")
		}

		allocation, allocationErr := s.db.FindAllocationByNormalizedPrefix(ctx, managedDomain.ID, label)
		switch {
		case allocationErr == nil && allocation.ID > 0:
			return UpdateTokenEmailFiltersResult{}, ConflictError("dynamic mailbox suffix conflicts with an allocated namespace")
		case allocationErr != nil && !storage.IsNotFound(allocationErr):
			return UpdateTokenEmailFiltersResult{}, InternalError("failed to check dynamic mailbox suffix conflicts", allocationErr)
		}

		domains = append(domains, label+"."+defaultRootDomain)
	}

	sort.Strings(normalizedSuffixes)
	sort.Strings(domains)

	if err := s.hub.UpdateMailboxDomains(token.PublicID, token.OwnerUserID, normalizedOwnerUsername, domains); err != nil {
		switch err {
		case mailrelay.ErrTokenStreamSubscriptionRequired:
			return UpdateTokenEmailFiltersResult{}, ConflictError("an active api token email stream connection is required before mailbox suffix filters can be updated")
		case mailrelay.ErrTokenStreamMailboxDomainConflict:
			return UpdateTokenEmailFiltersResult{}, ConflictError("one requested dynamic mailbox domain is already attached to another live api token stream")
		default:
			return UpdateTokenEmailFiltersResult{}, InternalError("failed to update api token mailbox suffix filters", err)
		}
	}

	return UpdateTokenEmailFiltersResult{
		Suffixes: normalizedSuffixes,
		Domains:  domains,
	}, nil
}

// RequireOwnedEmailAPIToken validates that the given token exists, belongs to the
// current user, is active, and supports the email stream capability.
func (s *TokenService) RequireOwnedEmailAPIToken(ctx context.Context, user model.User, publicID string) (model.APIToken, error) {
	item, err := s.db.GetAPITokenByPublicID(ctx, strings.TrimSpace(publicID))
	if err != nil {
		if storage.IsNotFound(err) {
			return model.APIToken{}, ValidationError("target_token_public_id is invalid")
		}
		return model.APIToken{}, InternalError("failed to load target api token", err)
	}
	if item.OwnerUserID != user.ID {
		return model.APIToken{}, ValidationError("target api token does not belong to the current user")
	}
	if item.RevokedAt != nil {
		return model.APIToken{}, ValidationError("target api token has been revoked")
	}
	if !apiTokenHasScope(item, model.APITokenScopeEmail) {
		return model.APIToken{}, ValidationError("target api token does not allow email streaming")
	}
	return item, nil
}

func userAPITokenFromModel(item model.APIToken) UserAPITokenView {
	return UserAPITokenView{
		ID:           item.ID,
		Name:         item.Name,
		PublicID:     item.PublicID,
		Scopes:       append([]string(nil), item.Scopes...),
		EmailEnabled: apiTokenHasScope(item, model.APITokenScopeEmail),
		LastUsedAt:   item.LastUsedAt,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
		RevokedAt:    item.RevokedAt,
	}
}

func apiTokenHasScope(item model.APIToken, scope string) bool {
	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	if normalizedScope == "" {
		return false
	}
	for _, itemScope := range item.Scopes {
		if strings.ToLower(strings.TrimSpace(itemScope)) == normalizedScope {
			return true
		}
	}
	return false
}

func hashAPIToken(rawToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	return hex.EncodeToString(sum[:])
}

func normalizeTokenMailboxSuffixFragment(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", nil
	}

	var builder strings.Builder
	lastWasDash := false
	for _, runeValue := range value {
		switch {
		case runeValue >= 'a' && runeValue <= 'z':
			builder.WriteRune(runeValue)
			lastWasDash = false
		case runeValue >= '0' && runeValue <= '9':
			builder.WriteRune(runeValue)
			lastWasDash = false
		default:
			if !lastWasDash {
				builder.WriteRune('-')
				lastWasDash = true
			}
		}
	}

	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		return "", ValidationError("mailbox suffix does not contain any valid dns characters")
	}
	if strings.Contains(normalized, ".") {
		return "", ValidationError("mailbox suffix must stay inside one dns label")
	}
	if len(normalized) > 48 {
		return "", ValidationError("mailbox suffix is too long")
	}
	if _, err := NormalizePrefix("mail" + normalized); err != nil {
		return "", ValidationError(fmt.Sprintf("mailbox suffix is invalid: %s", err.Error()))
	}
	return normalized, nil
}
