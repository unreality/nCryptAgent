package ncrypt

import (
	"crypto"
	"crypto/rsa"
	"fmt"
	"github.com/lxn/win"
	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
	"io"
	"math/big"
	"time"
)

type Signer struct {
	algorithmGroup string
	keyHandle      uintptr
	hwnd           uintptr
	publicKey      crypto.PublicKey
	timeout        int
	timeractive    bool
}

func newNCryptSigner(kh uintptr, timeout int) (crypto.Signer, error) {
	pub, err := getPublicKey(kh)
	if err != nil {
		return nil, fmt.Errorf("unable to get public key: %w", err)
	}

	algGroup, err := NCryptGetPropertyStr(kh, NCRYPT_ALGORITHM_GROUP_PROPERTY)
	if err != nil {
		return nil, fmt.Errorf("unable to get NCRYPT_ALGORITHM_GROUP_PROPERTY: %w", err)
	}

	signer := Signer{
		algorithmGroup: algGroup,
		keyHandle:      kh,
		publicKey:      pub,
		hwnd:           0,
		timeout:        timeout,
	}

	return &signer, nil
}

func (s *Signer) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if _, isRSAPSS := opts.(*rsa.PSSOptions); isRSAPSS {
		return nil, fmt.Errorf("RSA-PSS signing is not supported")
	}

	// Only popup our window if we have a parent window handle, and pin cache timer is not active
	// The active timer implies we already have a cached pin
	if s.hwnd != 0 && !s.timeractive {

		// TODO: if pageant is disabled, make a nicer experience by using win.GetForegroundWindow() as NCRYPT_WINDOW_HANDLE_PROPERTY
		err := NCryptSetProperty(s.keyHandle, NCRYPT_WINDOW_HANDLE_PROPERTY, s.hwnd, 0)
		if err != nil {
			fmt.Printf("%v", err)
		}

		curWnd := win.GetForegroundWindow()
		myID := win.GetCurrentThreadId()
		curID := win.GetWindowThreadProcessId(curWnd, nil)
		win.AttachThreadInput(int32(curID), int32(myID), true)
		win.ShowWindow(win.HWND(s.hwnd), win.SW_NORMAL)
		win.SetForegroundWindow(win.HWND(s.hwnd))
		win.SetFocus(win.HWND(s.hwnd))
		win.SetActiveWindow(win.HWND(s.hwnd))
		win.AttachThreadInput(int32(curID), int32(myID), false)

		defer func() {
			win.ShowWindow(win.HWND(s.hwnd), win.SW_HIDE)
			win.AttachThreadInput(int32(myID), int32(curID), true)
			win.ShowWindow(curWnd, win.SW_NORMAL)
			win.SetForegroundWindow(curWnd)
			win.SetFocus(curWnd)
			win.SetActiveWindow(curWnd)
			win.AttachThreadInput(int32(myID), int32(curID), false)
		}()
	}

	switch s.algorithmGroup {
	case "ECDSA":
		signatureBytes, err := NCryptSignHash(s.keyHandle, digest, "")

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
		hashAlg, ok := hashAlgorithms[hf]
		if !ok {
			return nil, fmt.Errorf("unsupported RSA hash algorithm %v", hf)
		}
		signatureBytes, err := NCryptSignHash(s.keyHandle, digest, hashAlg)

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
			NCryptSetProperty(s.keyHandle, NCRYPT_PIN_PROPERTY, "", 0)
			fmt.Printf("PIN Cache purged\n")
		})
		s.timeractive = true
	} else if s.timeout == 0 {
		NCryptSetProperty(s.keyHandle, NCRYPT_PIN_PROPERTY, "", 0)
	}
}

func (s *Signer) Public() crypto.PublicKey {
	return s.publicKey
}

func (s *Signer) SetHwnd(hwnd uintptr) {
	s.hwnd = uintptr(hwnd)
}

func (s *Signer) SetPINTimeout(timeout int) {
	s.timeout = timeout
}
