package config

import (
	"strings"
	"testing"
)

// TestLoadDisablesAdminConsoleByDefault verifies that the backend no longer
// exposes any administrator access unless the deployment explicitly configures
// both the admin allowlist and the second-factor password.
func TestLoadDisablesAdminConsoleByDefault(t *testing.T) {
	t.Setenv("APP_SESSION_SECRET", "test-session-secret")
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_ADMIN_USERNAMES", "")
	t.Setenv("APP_ADMIN_PASSWORD", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config without admin settings: %v", err)
	}
	if len(cfg.App.AdminUsernames) != 0 {
		t.Fatalf("expected no admin usernames by default, got %+v", cfg.App.AdminUsernames)
	}
	if cfg.App.AdminPassword != "" {
		t.Fatalf("expected no admin password by default, got %q", cfg.App.AdminPassword)
	}
}

// TestLoadRejectsPartialAdminConfiguration verifies that deployments cannot
// accidentally enable only half of the required admin protection.
func TestLoadRejectsPartialAdminConfiguration(t *testing.T) {
	testCases := []struct {
		name             string
		adminUsernames   string
		adminPassword    string
		expectedFragment string
	}{
		{
			name:             "missing password",
			adminUsernames:   "MoYeRanQianZhi,user2996",
			adminPassword:    "",
			expectedFragment: "APP_ADMIN_PASSWORD is required",
		},
		{
			name:             "missing usernames",
			adminUsernames:   "",
			adminPassword:    "strong-password",
			expectedFragment: "APP_ADMIN_USERNAMES is required",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("APP_SESSION_SECRET", "test-session-secret")
			t.Setenv("APP_ENV", "development")
			t.Setenv("APP_ADMIN_USERNAMES", testCase.adminUsernames)
			t.Setenv("APP_ADMIN_PASSWORD", testCase.adminPassword)

			_, err := Load()
			if err == nil {
				t.Fatalf("expected partial admin configuration to fail")
			}
			if !strings.Contains(err.Error(), testCase.expectedFragment) {
				t.Fatalf("expected error to contain %q, got %v", testCase.expectedFragment, err)
			}
		})
	}
}

// TestLoadRequiresExplicitAdminConfigInProduction verifies that production
// deployments fail fast unless the admin protection settings are configured.
func TestLoadRequiresExplicitAdminConfigInProduction(t *testing.T) {
	t.Setenv("APP_SESSION_SECRET", "test-session-secret")
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_ADMIN_USERNAMES", "")
	t.Setenv("APP_ADMIN_PASSWORD", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected production admin misconfiguration to fail")
	}
	if !strings.Contains(err.Error(), "APP_ADMIN_USERNAMES and APP_ADMIN_PASSWORD are required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLoadAcceptsExplicitAdminConfigInProduction verifies that production still
// starts normally once both administrator settings are provided explicitly.
func TestLoadAcceptsExplicitAdminConfigInProduction(t *testing.T) {
	t.Setenv("APP_SESSION_SECRET", "test-session-secret")
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_ADMIN_USERNAMES", "MoYeRanQianZhi,user2996")
	t.Setenv("APP_ADMIN_PASSWORD", "strong-password")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config with explicit production admin settings: %v", err)
	}
	if len(cfg.App.AdminUsernames) != 2 {
		t.Fatalf("expected 2 admin usernames, got %+v", cfg.App.AdminUsernames)
	}
	if cfg.App.AdminPassword != "strong-password" {
		t.Fatalf("expected configured admin password to survive load, got %q", cfg.App.AdminPassword)
	}
}
