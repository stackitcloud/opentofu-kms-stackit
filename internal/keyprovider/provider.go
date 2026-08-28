package keyprovider

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
)

const (
	ciphertextField = "ciphertext"
	versionField    = "key_version"

	// dataKeyLength is fixed at 32 bytes (AES-256), one of aes_gcm's three valid sizes.
	dataKeyLength = 32
)

// keyManagementClient hides v1api's request types: their fields are unexported and unmockable.
type keyManagementClient interface {
	Encrypt(ctx context.Context, version int64, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, version int64, ciphertext []byte) ([]byte, error)
	ListVersions(ctx context.Context) ([]v1api.Version, error)
}

// apiClient adapts v1api.DefaultAPI, bound to one key, to keyManagementClient.
type apiClient struct {
	svc       v1api.DefaultAPI
	projectID string
	region    string
	keyRingID string
	keyID     string
}

// Encrypt base64-encodes manually: v1api's Data field is a string, not []byte.
func (a *apiClient) Encrypt(ctx context.Context, version int64, plaintext []byte) ([]byte, error) {
	result, err := a.svc.Encrypt(ctx, a.projectID, a.region, a.keyRingID, a.keyID, version).
		EncryptPayload(v1api.EncryptPayload{
			Data: base64.StdEncoding.EncodeToString(plaintext),
		}).Execute()
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(result.GetData())
}

// Decrypt requires the version that produced this ciphertext.
func (a *apiClient) Decrypt(ctx context.Context, version int64, ciphertext []byte) ([]byte, error) {
	result, err := a.svc.Decrypt(ctx, a.projectID, a.region, a.keyRingID, a.keyID, version).
		DecryptPayload(v1api.DecryptPayload{
			Data: base64.StdEncoding.EncodeToString(ciphertext),
		}).Execute()
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(result.GetData())
}

// ListVersions returns every version: the API does not paginate this response.
func (a *apiClient) ListVersions(ctx context.Context) ([]v1api.Version, error) {
	vl, err := a.svc.ListVersions(ctx, a.projectID, a.region, a.keyRingID, a.keyID).Execute()
	if err != nil {
		return nil, err
	}
	return vl.GetVersions(), nil
}

// resolveLatestActiveVersion picks the highest-numbered active, non-disabled version.
func resolveLatestActiveVersion(ctx context.Context, svc keyManagementClient) (int64, error) {
	versions, err := svc.ListVersions(ctx)
	if err != nil {
		return 0, fmt.Errorf("list key versions: %w", err)
	}
	var best int64
	for _, v := range versions {
		if v.GetState() != v1api.VERSIONSTATE_ACTIVE || v.GetDisabled() {
			continue
		}
		if n := v.GetNumber(); n > best {
			best = n
		}
	}
	if best == 0 {
		return 0, fmt.Errorf("no active key version found")
	}
	return best, nil
}

// KeyProvider generates and unwraps data keys using STACKIT KMS.
type KeyProvider struct {
	svc     keyManagementClient
	version int64
}

// New builds a KeyProvider from a v1api client; version 0 resolves the latest active version.
func New(ctx context.Context, apiSvc v1api.DefaultAPI, projectID, region, keyRingID, keyID string, version int64) (*KeyProvider, error) {
	svc := &apiClient{svc: apiSvc, projectID: projectID, region: region, keyRingID: keyRingID, keyID: keyID}
	return newKeyProvider(ctx, svc, version)
}

// newKeyProvider takes keyManagementClient directly, so tests can pass a mock.
func newKeyProvider(ctx context.Context, svc keyManagementClient, version int64) (*KeyProvider, error) {
	if version == 0 {
		resolved, err := resolveLatestActiveVersion(ctx, svc)
		if err != nil {
			return nil, err
		}
		version = resolved
	}
	return &KeyProvider{svc: svc, version: version}, nil
}

// Provide decrypts using the version stored in the input metadata, not the configured one.
func (k *KeyProvider) Provide(ctx context.Context, in Input) (Output, error) {
	plaintext := make([]byte, dataKeyLength)
	if _, err := rand.Read(plaintext); err != nil {
		return Output{}, fmt.Errorf("generate data key: %w", err)
	}

	ciphertext, err := k.svc.Encrypt(ctx, k.version, plaintext)
	if err != nil {
		return Output{}, fmt.Errorf("encrypt data key (key_version=%d): %w", k.version, err)
	}

	out := Output{
		Keys: Keys{EncryptionKey: plaintext},
		Meta: Metadata{ExternalData: map[string]any{
			ciphertextField: ciphertext,
			versionField:    k.version,
		}},
	}

	if in != nil {
		storedCiphertext, storedVersion, err := readStoredMetadata(in.ExternalData)
		if err != nil {
			return Output{}, err
		}
		if storedCiphertext != nil {
			decrypted, err := k.svc.Decrypt(ctx, storedVersion, storedCiphertext)
			if err != nil {
				return Output{}, fmt.Errorf("decrypt data key (key_version=%d): %w", storedVersion, err)
			}
			out.Keys.DecryptionKey = decrypted
		}
	}

	return out, nil
}

// readStoredMetadata: JSON round-trips ciphertext/version as a string and a float64.
func readStoredMetadata(data map[string]any) (ciphertext []byte, version int64, err error) {
	rawCiphertext, present := data[ciphertextField]
	if !present {
		return nil, 0, nil
	}
	encoded, ok := rawCiphertext.(string)
	if !ok {
		return nil, 0, fmt.Errorf("invalid input metadata: %q must be a string, got %T", ciphertextField, rawCiphertext)
	}
	ciphertext, err = base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid input metadata: %q is not valid base64: %w", ciphertextField, err)
	}

	rawVersion, present := data[versionField]
	if !present {
		return nil, 0, fmt.Errorf("invalid input metadata: %q is missing", versionField)
	}
	versionNumber, ok := rawVersion.(float64)
	if !ok {
		return nil, 0, fmt.Errorf("invalid input metadata: %q must be a number, got %T", versionField, rawVersion)
	}
	return ciphertext, int64(versionNumber), nil
}
