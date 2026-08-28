package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/stackitcloud/opentofu-kms-stackit/internal/config"
	"github.com/stackitcloud/opentofu-kms-stackit/internal/keyprovider"

	stackitconfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
)

// Set by goreleaser via ldflags; "dev" for local builds.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// stdout is reserved for the protocol; all diagnostics go to stderr.
	log.SetOutput(os.Stderr)

	// Checked before the protocol claims stdout.
	if hasVersionFlag(os.Args[1:]) {
		fmt.Printf("opentofu-kms-stackit %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	if err := writeHeader(os.Stdout); err != nil {
		log.Fatalf("failed to write header: %v", err)
	}

	rawInput, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("failed to read stdin: %v", err)
	}

	var input keyprovider.Input
	if err := json.Unmarshal(rawInput, &input); err != nil {
		log.Fatalf("failed to parse input: %v", err)
	}

	cfg, err := config.New(os.Args[1:])
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	apiSvc, err := newAPIClient(cfg)
	if err != nil {
		log.Fatalf("failed to create STACKIT KMS client: %v", err)
	}

	ctx := context.Background()
	kp, err := keyprovider.New(ctx, apiSvc, cfg.ProjectID, cfg.Region, cfg.KeyRingID, cfg.KeyID, cfg.KeyVersion)
	if err != nil {
		log.Fatalf("failed to initialize key provider: %v", err)
	}

	output, err := kp.Provide(ctx, input)
	if err != nil {
		log.Fatalf("failed to provide key: %v", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		log.Fatalf("failed to write output: %v", err)
	}
}

func hasVersionFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-version" || arg == "--version" {
			return true
		}
	}
	return false
}

// writeHeader writes the single-line protocol header followed by a newline.
func writeHeader(w io.Writer) error {
	header := keyprovider.Header{Magic: keyprovider.HeaderMagic, Version: keyprovider.ProtocolVersion}
	marshalled, err := json.Marshal(header)
	if err != nil {
		return err
	}
	_, err = w.Write(append(marshalled, '\n'))
	return err
}

// No stackitconfig.WithRegion: KMS is global-endpoint, region goes per-call via keyprovider.New.
func newAPIClient(cfg *config.Config) (v1api.DefaultAPI, error) {
	var opts []stackitconfig.ConfigurationOption
	if cfg.ServiceAccountKey != "" {
		opts = append(opts, stackitconfig.WithServiceAccountKey(cfg.ServiceAccountKey))
	}
	if cfg.ServiceAccountKeyPath != "" {
		opts = append(opts, stackitconfig.WithServiceAccountKeyPath(cfg.ServiceAccountKeyPath))
	}
	if cfg.ServiceAccountToken != "" {
		opts = append(opts, stackitconfig.WithToken(cfg.ServiceAccountToken))
	}
	if cfg.PrivateKey != "" {
		opts = append(opts, stackitconfig.WithPrivateKey(cfg.PrivateKey))
	}
	if cfg.PrivateKeyPath != "" {
		opts = append(opts, stackitconfig.WithPrivateKeyPath(cfg.PrivateKeyPath))
	}
	if cfg.Endpoint != "" {
		opts = append(opts, stackitconfig.WithEndpoint(cfg.Endpoint))
	}
	opts = append(opts, stackitconfig.WithUserAgent("opentofu-kms-stackit"))

	client, err := v1api.NewAPIClient(opts...)
	if err != nil {
		return nil, err
	}
	return client.DefaultAPI, nil
}
