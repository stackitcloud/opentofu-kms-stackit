package config

import (
	"strings"
	"testing"
)

func TestNewFromFlags(t *testing.T) {
	cfg, err := New([]string{
		"-project-id", "proj",
		"-region", "eu01",
		"-key-ring-id", "ring",
		"-key-id", "key",
		"-key-version", "3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectID != "proj" || cfg.Region != "eu01" || cfg.KeyRingID != "ring" || cfg.KeyID != "key" {
		t.Fatalf("unexpected key coordinates: %+v", cfg)
	}
	if cfg.KeyVersion != 3 {
		t.Fatalf("expected key version 3, got %d", cfg.KeyVersion)
	}
}

func TestNewDefaultsRegion(t *testing.T) {
	cfg, err := New([]string{
		"-project-id", "proj",
		"-key-ring-id", "ring",
		"-key-id", "key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Region != "eu01" {
		t.Fatalf("expected default region eu01, got %q", cfg.Region)
	}
	if cfg.KeyVersion != 0 {
		t.Fatalf("expected key version to default to 0 (auto-resolve), got %d", cfg.KeyVersion)
	}
}

func TestNewFallsBackToEnv(t *testing.T) {
	t.Setenv("STACKIT_KMS_PROJECT_ID", "env-proj")
	t.Setenv("STACKIT_KMS_KEY_RING_ID", "env-ring")
	t.Setenv("STACKIT_KMS_KEY_ID", "env-key")

	cfg, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectID != "env-proj" || cfg.KeyRingID != "env-ring" || cfg.KeyID != "env-key" {
		t.Fatalf("unexpected key coordinates from env: %+v", cfg)
	}
}

func TestNewFlagOverridesEnv(t *testing.T) {
	t.Setenv("STACKIT_KMS_PROJECT_ID", "env-proj")

	cfg, err := New([]string{
		"-project-id", "flag-proj",
		"-key-ring-id", "ring",
		"-key-id", "key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectID != "flag-proj" {
		t.Fatalf("expected flag to override env, got %q", cfg.ProjectID)
	}
}

func TestNewRejectsMissingRequiredFields(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("expected an error when no key coordinates are provided, got nil")
	}
}

func TestNewRejectsNonNumericKeyVersion(t *testing.T) {
	_, err := New([]string{
		"-project-id", "proj",
		"-key-ring-id", "ring",
		"-key-id", "key",
		"-key-version", "not-a-number",
	})
	if err == nil {
		t.Fatal("expected an error for a non-numeric key-version, got nil")
	}
}

// Secrets must only come from the environment, never a flag default (see README.md).
func TestNewReadsSecretsFromEnvOnly(t *testing.T) {
	t.Setenv("STACKIT_SERVICE_ACCOUNT_KEY", "secret-key-json")
	t.Setenv("STACKIT_SERVICE_ACCOUNT_TOKEN", "secret-token")
	t.Setenv("STACKIT_PRIVATE_KEY", "secret-private-key")

	cfg, err := New([]string{
		"-project-id", "proj",
		"-key-ring-id", "ring",
		"-key-id", "key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServiceAccountKey != "secret-key-json" {
		t.Fatalf("expected service account key from env, got %q", cfg.ServiceAccountKey)
	}
	if cfg.ServiceAccountToken != "secret-token" {
		t.Fatalf("expected service account token from env, got %q", cfg.ServiceAccountToken)
	}
	if cfg.PrivateKey != "secret-private-key" {
		t.Fatalf("expected private key from env, got %q", cfg.PrivateKey)
	}
}

func TestNewRejectsSecretsAsFlags(t *testing.T) {
	for _, flag := range []string{"-service-account-key", "-service-account-token", "-private-key"} {
		_, err := New([]string{
			"-project-id", "proj",
			"-key-ring-id", "ring",
			"-key-id", "key",
			flag, "whatever",
		})
		if err == nil {
			t.Fatalf("expected %s to be rejected as an unknown flag, got nil error", flag)
		}
		if !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("expected an unknown-flag error for %s, got: %v", flag, err)
		}
	}
}
