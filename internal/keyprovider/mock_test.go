package keyprovider

import (
	"bytes"
	"context"
	"fmt"

	"github.com/stackitcloud/stackit-sdk-go/services/kms/v1api"
)

// mockKMC implements keyManagementClient directly, so tests never touch the
// real v1api SDK types.
type mockKMC struct {
	encrypt      func(version int64, plaintext []byte) ([]byte, error)
	decrypt      func(version int64, ciphertext []byte) ([]byte, error)
	listVersions func() ([]v1api.Version, error)
}

func (m *mockKMC) Encrypt(_ context.Context, version int64, plaintext []byte) ([]byte, error) {
	return m.encrypt(version, plaintext)
}

func (m *mockKMC) Decrypt(_ context.Context, version int64, ciphertext []byte) ([]byte, error) {
	return m.decrypt(version, ciphertext)
}

func (m *mockKMC) ListVersions(_ context.Context) ([]v1api.Version, error) {
	return m.listVersions()
}

// newTestMock's round trip only succeeds if Decrypt gets Encrypt's exact version.
func newTestMock() *mockKMC {
	return &mockKMC{
		encrypt: func(version int64, plaintext []byte) ([]byte, error) {
			prefix := []byte(fmt.Sprintf("v%d:", version))
			return append(prefix, plaintext...), nil
		},
		decrypt: func(version int64, ciphertext []byte) ([]byte, error) {
			prefix := []byte(fmt.Sprintf("v%d:", version))
			if !bytes.HasPrefix(ciphertext, prefix) {
				return nil, fmt.Errorf("ciphertext was not wrapped with key version %d", version)
			}
			return ciphertext[len(prefix):], nil
		},
		listVersions: func() ([]v1api.Version, error) {
			return []v1api.Version{
				{Number: 1, State: v1api.VERSIONSTATE_ACTIVE},
				{Number: 2, State: v1api.VERSIONSTATE_ACTIVE, Disabled: true},
				{Number: 3, State: v1api.VERSIONSTATE_DESTROYED},
			}, nil
		},
	}
}
