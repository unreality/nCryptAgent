package keyman

import (
	"crypto"
	"crypto/rsa"
	"fmt"
	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
	"io"
	"math/big"
	"ncryptagent/ncrypt"
	"time"
)

type Signer struct {
	algorithmGroup string
	keyHandle      uintptr
	publicKey      crypto.PublicKey
	timeout        int
	timeractive    bool
}

func newNCryptSigner(kh uintptr, timeout int) (crypto.Signer, error) {
	pub, err := getPublicKey(kh)
	if err != nil {
		return nil, fmt.Errorf("unable to get public key: %w", err)
	}

	algGroup, err := ncrypt.NCryptGetPropertyStr(kh, ncrypt.NCRYPT_ALGORITHM_GROUP_PROPERTY)
	if err != nil {
		return nil, fmt.Errorf("unable to get NCRYPT_ALGORITHM_GROUP_PROPERTY: %w", err)
	}

	signer := Signer{
		algorithmGroup: algGroup,
		keyHandle:      kh,
		publicKey:      pub,
		timeout:        timeout,
	}

	return &signer, nil
}

func (s *Signer) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if _, isRSAPSS := opts.(*rsa.PSSOptions); isRSAPSS {
		return nil, fmt.Errorf("RSA-PSS signing is not supported")
	}

	switch s.algorithmGroup {
	case "ECDSA":
		signatureBytes, err := ncrypt.NCryptSignHash(s.keyHandle, digest, "")

		if err != nil {
			return nil, err
		}

		s.handlePinTimer()

		if len(signatureBytes) >= len(digest)*2 {
			sigR := signatureBytes[:len(digest)]
			sigS := signatureBytes[len(digest):]

			var b cryptobyte.Builder
			b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
				b.AddASN1BigInt(new(big.Int).SetBytes(sigR))
				b.AddASN1BigInt(new(big.Int).SetBytes(sigS))
			})
			return b.Bytes()
		}

		return nil, fmt.Errorf("signatureBytes not long enough to encode ASN signature")
	case "RSA":
		hf := opts.HashFunc()
		hashAlg, ok := ncrypt.HashAlgorithms[hf]
		if !ok {
			return nil, fmt.Errorf("unsupported RSA hash algorithm %v", hf)
		}
		signatureBytes, err := ncrypt.NCryptSignHash(s.keyHandle, digest, hashAlg)

		if err != nil {
			return nil, fmt.Errorf("NCryptSignHash failed: %w", err)
		}

		s.handlePinTimer()

		return signatureBytes, nil
	default:
		return nil, fmt.Errorf("unsupported algorithm group %v", s.algorithmGroup)
	}
}

func (s *Signer) handlePinTimer() {
	if !s.timeractive && s.timeout > 0 {
		fmt.Printf("Starting pin cache purge timer: %ds\n", s.timeout)
		time.AfterFunc(time.Second*time.Duration(s.timeout), func() {
			s.timeractive = false
			ncrypt.NCryptSetProperty(s.keyHandle, ncrypt.NCRYPT_PIN_PROPERTY, "", 0)
			fmt.Printf("PIN Cache purged\n")
		})
		s.timeractive = true
	} else if s.timeout == 0 {
		ncrypt.NCryptSetProperty(s.keyHandle, ncrypt.NCRYPT_PIN_PROPERTY, "", 0)
	}
}

func (s *Signer) Public() crypto.PublicKey {
	return s.publicKey
}

func (s *Signer) SetPINTimeout(timeout int) {
	s.timeout = timeout
}
