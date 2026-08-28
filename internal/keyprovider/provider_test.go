package keyprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
)

// roundTripJSON simulates the real protocol: Output is JSON-encoded to
// stdout by this program and JSON-decoded from stdin on the next run, so
// map[string]any values come back as string/float64, not []byte/int64.
func roundTripJSON(t *testing.T, meta Metadata) *Metadata {
	t.Helper()
	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Metadata
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	return &decoded
}

func TestProvideRoundTrip(t *testing.T) {
	ctx := context.Background()
	kp, err := newKeyProvider(ctx, newTestMock(), 1)
	if err != nil {
		t.Fatal(err)
	}

	out, err := kp.Provide(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Keys.EncryptionKey) != 32 {
		t.Fatalf("expected a 32-byte encryption key, got %d bytes", len(out.Keys.EncryptionKey))
	}
	if len(out.Keys.DecryptionKey) != 0 {
		t.Fatal("expected no decryption key without input metadata")
	}

	meta := roundTripJSON(t, out.Meta)
	out2, err := kp.Provide(ctx, meta)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out2.Keys.DecryptionKey, out.Keys.EncryptionKey) {
		t.Fatal("decrypted key does not match the originally encrypted key")
	}
}

func TestProvideAutoResolvesVersion(t *testing.T) {
	ctx := context.Background()
	kp, err := newKeyProvider(ctx, newTestMock(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if kp.version != 1 {
		t.Fatalf("expected auto-resolved version 1, got %d", kp.version)
	}
}

func TestProvideNoActiveVersionFound(t *testing.T) {
	ctx := context.Background()
	svc := &mockKMC{
		listVersions: func() ([]v1api.Version, error) {
			return []v1api.Version{
				{Number: 1, State: v1api.VERSIONSTATE_DESTROYED},
				{Number: 2, State: v1api.VERSIONSTATE_ACTIVE, Disabled: true},
			}, nil
		},
	}
	if _, err := newKeyProvider(ctx, svc, 0); err == nil {
		t.Fatal("expected an error when no active key version is available, got nil")
	}
}

// Regression test: decrypt must use the version stored in the input
// metadata, not the version the KeyProvider is currently configured with.
func TestProvideUsesStoredVersionNotConfiguredVersion(t *testing.T) {
	ctx := context.Background()
	mock := newTestMock()

	oldKp, err := newKeyProvider(ctx, mock, 1)
	if err != nil {
		t.Fatal(err)
	}
	oldOut, err := oldKp.Provide(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	newKp, err := newKeyProvider(ctx, mock, 2)
	if err != nil {
		t.Fatal(err)
	}
	meta := roundTripJSON(t, oldOut.Meta)
	out, err := newKp.Provide(ctx, meta)
	if err != nil {
		t.Fatalf("decrypting state wrapped with a since-rotated key version failed: %v", err)
	}
	if !bytes.Equal(out.Keys.DecryptionKey, oldOut.Keys.EncryptionKey) {
		t.Fatal("decrypted key does not match the originally encrypted key")
	}
}

func TestProvideRejectsMissingVersionInMetadata(t *testing.T) {
	ctx := context.Background()
	kp, err := newKeyProvider(ctx, newTestMock(), 1)
	if err != nil {
		t.Fatal(err)
	}
	meta := &Metadata{ExternalData: map[string]any{ciphertextField: "dGVzdA=="}}
	if _, err := kp.Provide(ctx, meta); err == nil {
		t.Fatal("expected an error when key_version is missing from input metadata, got nil")
	}
}

func TestProvideEncryptionOnlyWhenMetadataHasNoCiphertext(t *testing.T) {
	ctx := context.Background()
	kp, err := newKeyProvider(ctx, newTestMock(), 1)
	if err != nil {
		t.Fatal(err)
	}
	// "null" input in protocol terms: Go decodes it into a nil *Metadata.
	meta := &Metadata{ExternalData: map[string]any{}}
	out, err := kp.Provide(ctx, meta)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Keys.DecryptionKey) != 0 {
		t.Fatal("expected no decryption key when metadata has no stored ciphertext")
	}
}
