// Package keyprovider implements OpenTofu's external key provider protocol for STACKIT KMS.
package keyprovider

// HeaderMagic identifies the program as an OpenTofu external key provider.
const HeaderMagic = "OpenTofu-External-Key-Provider"

// ProtocolVersion is the only protocol version currently supported.
const ProtocolVersion = 1

// Header is the single-line greeting the key provider writes to stdout first.
type Header struct {
	Magic   string `json:"magic"`
	Version int    `json:"version"`
}

// Metadata is stored alongside the encrypted data and replayed on the next decryption run.
type Metadata struct {
	ExternalData map[string]any `json:"external_data"`
}

// Input is nil ("null") for encryption only, or the stored metadata when decrypting.
type Input *Metadata

// Keys holds the produced key material.
type Keys struct {
	// EncryptionKey must always be provided. As a []byte it is marshaled to base64.
	EncryptionKey []byte `json:"encryption_key,omitempty"`
	// DecryptionKey is only provided when input metadata was present.
	DecryptionKey []byte `json:"decryption_key,omitempty"`
}

// Output is the data structure written to stdout.
type Output struct {
	Keys Keys     `json:"keys"`
	Meta Metadata `json:"meta"`
}
