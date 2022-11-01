package keyman

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"sync"
)

type KeyManagerAgent struct {
	km     *KeyManager
	locked bool
	mu     sync.Mutex
}

// List returns the identities known to the agent.
func (kma *KeyManagerAgent) List() ([]*agent.Key, error) {
	kma.mu.Lock()
	defer kma.mu.Unlock()

	if kma.km.Keys == nil {
		return nil, nil
	}

	var ids []*agent.Key
	for _, k := range kma.km.KeysList() {
		if k.SSHPublicKey != nil {
			pub := *k.SSHPublicKey
			ids = append(ids, &agent.Key{
				Format:  pub.Type(),
				Blob:    pub.Marshal(),
				Comment: k.Name})
		}

		// Check for a cert
		k.LoadCertificate("")

		if k.SSHCertificate != nil {
			pub := *k.SSHCertificate
			ids = append(ids, &agent.Key{
				Format:  pub.Type(),
				Blob:    pub.Marshal(),
				Comment: k.Name})
		}
	}
	return ids, nil
}

// Sign has the agent sign the data using a protocol 2 key as defined
// in [PROTOCOL.agent] section 2.6.2.
func (kma *KeyManagerAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	return kma.SignWithFlags(key, data, 0)
}

func (kma *KeyManagerAgent) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	for _, k := range kma.km.KeysList() {
		// Some clients might send the certificate blob as a key instead, so check equality for that
		var certMatches = false
		if k.SSHCertificate != nil {
			certMatches = bytes.Equal(k.SSHCertificate.Marshal(), key.Marshal())
		}

		pub := *k.SSHPublicKey
		if bytes.Equal(pub.Marshal(), key.Marshal()) || certMatches {
			if flags == 0 {
				sig, err := k.SignSSH(data)

				if err != nil {
					kma.km.Notify(NotifyMsg{
						Title:   "SSH Sign Failed",
						Message: fmt.Sprintf("Failed to sign message with key \"%s\"", k.Name),
						Icon: struct {
							DLL   string
							Index int32
							Size  int
						}{
							DLL:   "imageres",
							Index: 100,
							Size:  32,
						},
					})

					return nil, err
				}

				kma.km.Notify(NotifyMsg{
					Title:   "SSH Sign Successful",
					Message: fmt.Sprintf("Signed message with key \"%s\"", k.Name),
					Icon: struct {
						DLL   string
						Index int32
						Size  int
					}{
						DLL:   "imageres",
						Index: 101,
						Size:  32,
					},
				})

				return sig, err

			} else {
				var algorithm string

				switch flags {
				case agent.SignatureFlagRsaSha256:
					algorithm = ssh.KeyAlgoRSASHA256
				case agent.SignatureFlagRsaSha512:
					algorithm = ssh.KeyAlgoRSASHA512
				default:
					return nil, fmt.Errorf("agent: unsupported signature flags: %d", flags)
				}
				sig, err := k.SignWithAlgorithmSSH(data, algorithm)

				if err != nil {
					kma.km.Notify(NotifyMsg{
						Title:   "SSH Sign Failed",
						Message: fmt.Sprintf("Failed to sign message with key \"%s\"", k.Name),
						Icon: struct {
							DLL   string
							Index int32
							Size  int
						}{
							DLL:   "imageres",
							Index: 100,
							Size:  32,
						},
					})

					return nil, err
				}

				kma.km.Notify(NotifyMsg{
					Title:   "SSH Sign Successful",
					Message: fmt.Sprintf("Signed message with key \"%s\"", k.Name),
					Icon: struct {
						DLL   string
						Index int32
						Size  int
					}{
						DLL:   "imageres",
						Index: 101,
						Size:  32,
					},
				})

				return sig, err
			}
		}
	}

	return nil, fmt.Errorf("not found")
}

// Add adds a private key to the agent.
func (kma *KeyManagerAgent) Add(key agent.AddedKey) error {
	return fmt.Errorf("not implemented")
}

// Remove removes all identities with the given public key.
func (kma *KeyManagerAgent) Remove(key ssh.PublicKey) error {
	return fmt.Errorf("not implemented")
}

// RemoveAll removes all identities.
func (kma *KeyManagerAgent) RemoveAll() error {
	return fmt.Errorf("not implemented")
}

// Lock locks the agent. Sign and Remove will fail, and List will empty an empty list.
func (kma *KeyManagerAgent) Lock(passphrase []byte) error {
	kma.locked = true
	return nil
}

// Unlock undoes the effect of Lock
func (kma *KeyManagerAgent) Unlock(passphrase []byte) error {
	kma.locked = false
	return nil
}

// Signers returns signers for all the known keys.
func (kma *KeyManagerAgent) Signers() ([]ssh.Signer, error) {
	return nil, fmt.Errorf("not implemented")
}

func (kma *KeyManagerAgent) Extension(extensionType string, contents []byte) ([]byte, error) {
	return nil, agent.ErrExtensionUnsupported
}
