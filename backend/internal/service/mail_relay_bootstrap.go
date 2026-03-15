package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"linuxdospace/backend/internal/config"
	"linuxdospace/backend/internal/model"
)

// EnsureDatabaseRelayIngressDNSForRootDomain ensures one namespace-scoped mail
// domain points at the built-in SMTP relay when database-relay mode is active.
// The helper intentionally skips the parent root because the parent mailbox
// flow still relies on Cloudflare's exact-address forwarding.
func EnsureDatabaseRelayIngressDNSForRootDomain(ctx context.Context, cfg config.Config, cf CloudflareClient, rootDomain string) error {
	return ensureDatabaseRelayIngressDNSForRootDomain(ctx, cfg, cf, rootDomain)
}

// EnsureDatabaseRelayIngressDNSState scans the current database state and
// backfills relay ingress DNS for every namespace that should already be served
// by the built-in SMTP relay. This protects production after operators switch
// from Cloudflare catch-all delivery to database-relay mode or move servers.
func EnsureDatabaseRelayIngressDNSState(ctx context.Context, cfg config.Config, store Store, cf CloudflareClient) error {
	if !cfg.UsesDatabaseMailRelay() || !cfg.Mail.EnsureDNS {
		return nil
	}
	if store == nil {
		return fmt.Errorf("database mail relay dns bootstrap store is nil")
	}
	if cf == nil || !cfg.CloudflareConfigured() {
		return fmt.Errorf("database mail relay dns bootstrap requires cloudflare dns access")
	}

	namespaceRoots := map[string]struct{}{}
	addNamespaceRoot := func(rootDomain string) error {
		normalizedRoot := normalizeDNSName(rootDomain)
		if normalizedRoot == "" {
			return nil
		}
		if !usesDatabaseRelayNamespaceRoot(cfg, normalizedRoot) {
			return nil
		}
		namespaceRoots[normalizedRoot] = struct{}{}
		return nil
	}

	emailRoutes, err := store.ListEmailRoutes(ctx)
	if err != nil {
		return fmt.Errorf("list email routes for database mail relay dns bootstrap: %w", err)
	}
	for _, item := range emailRoutes {
		if err := addNamespaceRoot(item.RootDomain); err != nil {
			return err
		}
	}

	applications, err := store.ListAdminApplications(ctx)
	if err != nil {
		return fmt.Errorf("list admin applications for database mail relay dns bootstrap: %w", err)
	}
	for _, item := range applications {
		if item.Type != PermissionKeyEmailCatchAll || !strings.EqualFold(strings.TrimSpace(item.Status), "approved") {
			continue
		}
		_, rootDomain, parseErr := parseCatchAllTargetAddress(item.Target)
		if parseErr != nil {
			return fmt.Errorf("parse approved catch-all application %d target %q: %w", item.ID, item.Target, parseErr)
		}
		if err := addNamespaceRoot(rootDomain); err != nil {
			return err
		}
	}

	orderedRoots := make([]string, 0, len(namespaceRoots))
	for rootDomain := range namespaceRoots {
		orderedRoots = append(orderedRoots, rootDomain)
	}
	sort.Strings(orderedRoots)

	for _, rootDomain := range orderedRoots {
		if err := ensureDatabaseRelayIngressDNSForRootDomain(ctx, cfg, cf, rootDomain); err != nil {
			return fmt.Errorf("ensure database mail relay ingress dns for %s: %w", rootDomain, err)
		}
	}
	return nil
}

// ensureDatabaseRelayIngressDNSForRootDomain keeps the service-layer validation
// in one place so both request-time saves and startup backfills apply the same
// parent-root safety guard before writing relay MX/TXT records.
func ensureDatabaseRelayIngressDNSForRootDomain(ctx context.Context, cfg config.Config, cf CloudflareClient, rootDomain string) error {
	if !cfg.UsesDatabaseMailRelay() {
		return nil
	}

	normalizedRoot := normalizeDNSName(rootDomain)
	defaultRoot := normalizeDNSName(cfg.Cloudflare.DefaultRootDomain)
	if normalizedRoot == "" {
		return ValidationError("catch-all relay namespace is empty")
	}
	if defaultRoot != "" && normalizedRoot == defaultRoot {
		return UnavailableError(
			"refusing to bootstrap mail-relay dns on the parent root domain",
			fmt.Errorf("parent root %s must stay on cloudflare mail routing", defaultRoot),
		)
	}

	return newEmailRoutingProvisioner(cfg, cf).ensureDatabaseRelayIngressDNS(ctx, normalizedRoot)
}

// collectApprovedCatchAllApplications filters one application slice down to the
// currently approved catch-all permission rows. The helper stays local to this
// file but keeps the startup bootstrap logic readable.
func collectApprovedCatchAllApplications(items []model.AdminApplication) []model.AdminApplication {
	filtered := make([]model.AdminApplication, 0, len(items))
	for _, item := range items {
		if item.Type != PermissionKeyEmailCatchAll || !strings.EqualFold(strings.TrimSpace(item.Status), "approved") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}
