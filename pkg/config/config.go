// Package config loads the per-key settings for opentofu-kms-stackit.
// Auth is resolved by stackit-sdk-go's own credential chain; fields here only override it.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ProjectID string
	Region    string
	KeyRingID string
	KeyID     string
	// KeyVersion to wrap with; 0 resolves the latest active, non-disabled version.
	KeyVersion int64

	// Auth overrides, all optional. Secret-valued fields are env-var-only, never flags (see README.md).
	ServiceAccountKey     string
	ServiceAccountToken   string
	PrivateKey            string
	ServiceAccountKeyPath string
	PrivateKeyPath        string
	Endpoint              string
}

// New parses args into a validated Config, falling back to STACKIT_KMS_*/STACKIT_* env vars.
func New(args []string) (*Config, error) {
	fs := flag.NewFlagSet("opentofu-kms-stackit", flag.ContinueOnError)

	projectID := fs.String("project-id", "", "STACKIT project UUID")
	region := fs.String("region", "", "STACKIT region, e.g. eu01 (default eu01)")
	keyRingID := fs.String("key-ring-id", "", "key ring UUID")
	keyID := fs.String("key-id", "", "key UUID")
	keyVersion := fs.String("key-version", "", "key version to wrap with (default: latest active)")
	serviceAccountKeyPath := fs.String("service-account-key-path", "", "path to a service account key file")
	privateKeyPath := fs.String("private-key-path", "", "path to a private key file")
	endpoint := fs.String("endpoint", "", "override the STACKIT KMS API endpoint")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	cfg := &Config{
		ProjectID:             envOrDefault(*projectID, "STACKIT_KMS_PROJECT_ID", ""),
		Region:                envOrDefault(*region, "STACKIT_KMS_REGION", "eu01"),
		KeyRingID:             envOrDefault(*keyRingID, "STACKIT_KMS_KEY_RING_ID", ""),
		KeyID:                 envOrDefault(*keyID, "STACKIT_KMS_KEY_ID", ""),
		ServiceAccountKeyPath: envOrDefault(*serviceAccountKeyPath, "STACKIT_SERVICE_ACCOUNT_KEY_PATH", ""),
		PrivateKeyPath:        envOrDefault(*privateKeyPath, "STACKIT_PRIVATE_KEY_PATH", ""),
		Endpoint:              envOrDefault(*endpoint, "STACKIT_KMS_ENDPOINT", ""),
		// Secrets: env var only, intentionally not flags (see the Config field comment).
		ServiceAccountKey:   os.Getenv("STACKIT_SERVICE_ACCOUNT_KEY"),
		ServiceAccountToken: os.Getenv("STACKIT_SERVICE_ACCOUNT_TOKEN"),
		PrivateKey:          os.Getenv("STACKIT_PRIVATE_KEY"),
	}

	rawKeyVersion := envOrDefault(*keyVersion, "STACKIT_KMS_KEY_VERSION", "")
	if rawKeyVersion != "" {
		v, err := strconv.ParseInt(rawKeyVersion, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("key-version must be an integer, got %q: %w", rawKeyVersion, err)
		}
		cfg.KeyVersion = v
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validate(cfg *Config) error {
	var errs []error
	if cfg.ProjectID == "" {
		errs = append(errs, errors.New("missing project-id"))
	}
	if cfg.Region == "" {
		errs = append(errs, errors.New("missing region"))
	}
	if cfg.KeyRingID == "" {
		errs = append(errs, errors.New("missing key-ring-id"))
	}
	if cfg.KeyID == "" {
		errs = append(errs, errors.New("missing key-id"))
	}
	return errors.Join(errs...)
}

// envOrDefault prefers flagValue, then the environment variable, then def.
func envOrDefault(flagValue, envKey, def string) string {
	if flagValue != "" {
		return flagValue
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return def
}
