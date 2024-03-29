package keyman

import (
	"bytes"
	"context"
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
	"github.com/lxn/win"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"math/big"
	"ncryptagent/keyman/listeners"
	"ncryptagent/ncrypt"
	"ncryptagent/webauthn"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	OPENSSH_SK_ECDSA        = "sk-ecdsa-sha2-nistp256@openssh.com"
	OPENSSH_SK_ED25519      = "sk-ssh-ed25519@openssh.com"
	OPENSSH_SK_ECDSA_CERT   = "sk-ecdsa-sha2-nistp256-cert-v01@openssh.com"
	OPENSSH_SK_ED25519_CERT = "sk-ssh-ed25519-cert-v01@openssh.com"
)

type NotifyMsg struct {
	Title   string
	Message string
	Icon    struct {
		DLL   string
		Index int32
		Size  int
	}
}

type sshPrivateKeySKECDSA struct {
	Type        string
	ID          string
	Key         []byte
	Application string
	Flags       byte
	KeyHandle   []byte
	Reserved    string
}

type sshPrivateKeySKED25519 struct {
	Type        string
	Key         []byte
	Application string
	Flags       byte
	KeyHandle   []byte
	Reserved    string
}

type KeyConfig struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	ContainerName  string `json:"containerName"`
	ProviderName   string `json:"providerName,omitempty"`
	SSHPublicKey   string `json:"sshPublicKey,omitempty"`
	SKPrivateHalf  string `json:"skPrivateHalf,omitempty"`
	Algorithm      string `json:"algorithm,omitempty"`
	Length         int    `json:"length,omitempty"`
	VerifyRequired bool   `json:"verifyRequired,omitempty"`
	NoPin          bool   `json:"noPin,omitempty"`
}

type KeyManagerConfig struct {
	Keys                 []*KeyConfig `json:"keys,omitempty"`
	PinTimeout           int          `json:"pinTimeout,omitempty"`
	PageantEnabled       bool         `json:"pageant"`
	VSockEnabled         bool         `json:"vsock"`
	NamedPipeEnabled     bool         `json:"namedpipe"`
	CygwinEnabled        bool         `json:"cygwin"`
	DisableNotifications bool         `json:"disableNotifications,omitempty"`
	USBEvents            bool         `json:"usbEvents,omitempty"`
}

type Key struct {
	Name                 string
	Type                 string
	SSHPublicKey         *ssh.PublicKey
	SSHCertificate       *ssh.Certificate
	SSHPublicKeyLocation string
	Missing              bool

	LoadError error

	algorithm string
	length    int

	config *KeyConfig
	handle uintptr
	signer *crypto.Signer
	hwnd   uintptr

	focusData struct {
		victimHWND win.HWND
		victimID   uint32
		myID       uint32
	}
}

func (k *Key) TakeFocus() bool {
	if !k.config.NoPin && k.hwnd != 0 {
		if k.Type == "NCRYPT" && k.signer != nil {
			if ncryptSigner, ok := (*k.signer).(*Signer); ok {
				if ncryptSigner.timeractive {
					return false
				}
			}
		}
		hwnd := win.HWND(k.hwnd)

		k.focusData.victimHWND = win.GetForegroundWindow()
		k.focusData.myID = win.GetCurrentThreadId()
		k.focusData.victimID = win.GetWindowThreadProcessId(k.focusData.victimHWND, nil)
		win.AttachThreadInput(int32(k.focusData.victimID), int32(k.focusData.myID), true)
		win.ShowWindow(hwnd, win.SW_NORMAL)
		win.SetForegroundWindow(hwnd)
		win.SetFocus(hwnd)
		win.SetActiveWindow(hwnd)
		win.AttachThreadInput(int32(k.focusData.victimID), int32(k.focusData.myID), false)

		return true
	}

	return false
}

func (k *Key) ReturnFocus() {
	if !k.config.NoPin {
		win.ShowWindow(win.HWND(k.hwnd), win.SW_HIDE)
		win.AttachThreadInput(int32(k.focusData.myID), int32(k.focusData.victimID), true)
		win.ShowWindow(k.focusData.victimHWND, win.SW_SHOW)
		win.SetForegroundWindow(k.focusData.victimHWND)
		win.SetFocus(k.focusData.victimHWND)
		win.SetActiveWindow(k.focusData.victimHWND)
		win.AttachThreadInput(int32(k.focusData.myID), int32(k.focusData.victimID), false)
	}
}

func (k *Key) AlgorithmReadable() string {
	if k.algorithm == ncrypt.ALG_RSA {
		return fmt.Sprintf("%s-%d", k.algorithm, k.length)
	} else {
		return k.algorithm
	}
}

func (k *Key) SSHPublicKeyString() string {
	if k.SSHPublicKey != nil {
		pkBytes := ssh.MarshalAuthorizedKey(*k.SSHPublicKey)

		if k.Type == "WEBAUTHN" && k.config.VerifyRequired {
			pkBytes = append([]byte("verify-required "), pkBytes...)
		}

		return string(pkBytes)
	}

	return "unknown"
}

func (k *Key) SSHPublicKeyFingerprint() string {
	if k.SSHPublicKey != nil {
		return ssh.FingerprintSHA256(*k.SSHPublicKey)
	}

	return "unknown"
}

func (k *Key) SSHPublicKeyType() string {
	if k.SSHPublicKey != nil {
		return (*k.SSHPublicKey).Type()
	}

	return "unknown"
}

func (k *Key) SaveSSHPublicKey(publicKeysDir string) error {
	if k.SSHPublicKey != nil {
		fingerprint := ssh.FingerprintLegacyMD5(*k.SSHPublicKey)
		filename := fmt.Sprintf("%s.pub", strings.ReplaceAll(fingerprint, ":", ""))
		k.SSHPublicKeyLocation = filepath.Join(publicKeysDir, filename)
		log.Printf("Saving public key to %s\n", k.SSHPublicKeyLocation)

		f, err := os.OpenFile(k.SSHPublicKeyLocation, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(k.SSHPublicKeyString()))
		f.Close()
	}

	return nil
}

func (k *Key) LoadCertificate(copyFromPath string) error {
	if k.SSHPublicKeyLocation != "" && k.SSHPublicKey != nil {
		publicKeysDir := filepath.Dir(k.SSHPublicKeyLocation)
		fingerprint := ssh.FingerprintLegacyMD5(*k.SSHPublicKey)
		filename := fmt.Sprintf("%s-cert.pub", strings.ReplaceAll(fingerprint, ":", ""))
		certPath := filepath.Join(publicKeysDir, filename)

		var certContents []byte
		var err error

		if copyFromPath != "" {
			certContents, err = ioutil.ReadFile(copyFromPath)
			if err != nil {
				return err
			}

			f, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			_, err = f.Write(certContents)
			f.Close()
		} else {
			certContents, err = ioutil.ReadFile(certPath)
			if err != nil {
				return fmt.Errorf("could not open cert file")
			}
		}

		pub, _, _, _, err := ssh.ParseAuthorizedKey(certContents)
		if err != nil {
			return err
		}
		if cert, ok := pub.(*ssh.Certificate); ok {
			if bytes.Equal(cert.Key.Marshal(), (*k.SSHPublicKey).Marshal()) {
				k.SSHCertificate = cert
				return nil
			} else {
				return fmt.Errorf("certificate does not match selected key")
			}

		}
	}

	return nil
}

func (k *Key) ContainerName() string {
	if k.config != nil {
		return k.config.ContainerName
	}

	return ""
}

func (k *Key) Close() {
	if k.handle != 0 {
		ncrypt.NCryptFreeObject(k.handle)
	}
	k.handle = 0
}

func (k *Key) SignSSH(b []byte) (*ssh.Signature, error) {
	if k.TakeFocus() {
		defer k.ReturnFocus()
	}

	if k.Type == "NCRYPT" && k.signer != nil {
		sshSigner, err := ssh.NewSignerFromSigner(*k.signer)

		if err != nil {
			return nil, err
		}

		signature, err := sshSigner.Sign(rand.Reader, b)
		if err == nil {
			k.Missing = false
		}

		return signature, err
	} else if k.Type == "WEBAUTHN" {
		return k.signWebAuthN(b)
	}

	return nil, fmt.Errorf("invalid signer")
}

func (k *Key) SignWithAlgorithmSSH(b []byte, algorithm string) (*ssh.Signature, error) {
	if k.TakeFocus() {
		defer k.ReturnFocus()
	}

	if k.Type == "NCRYPT" && k.signer != nil {
		sshSigner, err := ssh.NewSignerFromSigner(*k.signer)

		if err != nil {
			return nil, err
		}

		if algorithmSigner, ok := sshSigner.(ssh.AlgorithmSigner); ok {

			signature, err := algorithmSigner.SignWithAlgorithm(rand.Reader, b, algorithm)
			if err == nil {
				k.Missing = false
			}

			return signature, err
		} else {
			return nil, fmt.Errorf("invalid signer type %T", algorithmSigner)
		}
	} else if k.Type == "WEBAUTHN" {
		return k.signWebAuthN(b)
	}

	return nil, fmt.Errorf("invalid signer")
}

func (k *Key) signWebAuthN(signData []byte) (*ssh.Signature, error) {

	privBytes, err := base64.StdEncoding.DecodeString(k.config.SKPrivateHalf)
	if err != nil {
		return nil, fmt.Errorf("error decoding base64 private half: %w", err)
	}

	var application string
	var keyHandle []byte

	if (*k.SSHPublicKey).Type() == OPENSSH_SK_ECDSA || (*k.SSHPublicKey).Type() == OPENSSH_SK_ECDSA_CERT {
		priv := sshPrivateKeySKECDSA{}
		err = ssh.Unmarshal(privBytes, &priv)

		if err != nil {
			return nil, fmt.Errorf("error unmarshalling private half: %w", err)
		}

		application = priv.Application
		keyHandle = priv.KeyHandle
	} else if (*k.SSHPublicKey).Type() == OPENSSH_SK_ED25519 || (*k.SSHPublicKey).Type() == OPENSSH_SK_ED25519_CERT {
		priv := sshPrivateKeySKED25519{}
		err = ssh.Unmarshal(privBytes, &priv)

		if err != nil {
			return nil, fmt.Errorf("error unmarshalling private half: %w", err)
		}

		application = priv.Application
		keyHandle = priv.KeyHandle
	}

	clientData := webauthn.CLIENT_DATA{
		Version:              webauthn.CLIENT_DATA_CURRENT_VERSION,
		ClientDataJSONLength: uint32(len(signData)),
		ClientDataJSON:       uintptr(unsafe.Pointer(&signData[0])),
		HashAlgId:            webauthn.LPCWSTR(webauthn.HASH_ALGORITHM_SHA_256),
	}

	credentials := []webauthn.CREDENTIAL{
		{
			Version:        webauthn.CREDENTIAL_CURRENT_VERSION,
			IdLen:          uint32(len(keyHandle)),
			Id:             uintptr(unsafe.Pointer(&keyHandle[0])),
			CredentialType: webauthn.LPCWSTR(webauthn.CREDENTIAL_TYPE_PUBLIC_KEY),
		},
	}

	userVerification := webauthn.USER_VERIFICATION_REQUIREMENT_DISCOURAGED
	if k.config.VerifyRequired {
		userVerification = webauthn.USER_VERIFICATION_REQUIREMENT_REQUIRED
	}

	assertionOptions := webauthn.AUTHENTICATOR_GET_ASSERTION_OPTIONS{
		Version: webauthn.AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS_CURRENT_VERSION,
		CredentialList: webauthn.CREDENTIALS{
			Count:       1,
			Credentials: uintptr(unsafe.Pointer(&credentials[0])),
		},
		UserVerificationRequirement: uint32(userVerification),
	}

	assertion, err := webauthn.AuthenticatorGetAssertion(k.hwnd, application, clientData, assertionOptions)

	if err != nil {
		return nil, fmt.Errorf("AuthenticatorGetAssertion failed: %w", err)
	}

	defer webauthn.FreeAssertion(assertion)

	authDataBytes := webauthn.UintptrToBytes(assertion.AuthenticatorData, assertion.AuthenticatorDataLen)
	assertionSignatureBytes := webauthn.UintptrToBytes(assertion.Signature, assertion.SignatureLen)

	authData := webauthn.AuthenticatorData{}
	reader := bytes.NewReader(authDataBytes)
	err = binary.Read(reader, binary.BigEndian, &authData.RPIDHash)
	err = binary.Read(reader, binary.BigEndian, &authData.Flags)
	err = binary.Read(reader, binary.BigEndian, &authData.Counter)

	additionalData := struct {
		Flags   byte
		Counter uint32
	}{
		Flags:   authData.Flags,
		Counter: authData.Counter,
	}

	var signatureBytes []byte

	if (*k.SSHPublicKey).Type() == "sk-ecdsa-sha2-nistp256@openssh.com" || (*k.SSHPublicKey).Type() == "sk-ecdsa-sha2-nistp256-cert-v01@openssh.com" {
		signatureParsed := struct {
			R *big.Int
			S *big.Int
		}{}

		_, err = asn1.Unmarshal(assertionSignatureBytes, &signatureParsed)
		if err != nil {
			return nil, fmt.Errorf("asn1.Unmarshal of ECDSA signature failed: %w", err)
		}

		signatureBytes = ssh.Marshal(signatureParsed)
	} else {
		signatureBytes = assertionSignatureBytes
	}

	sig := ssh.Signature{
		Format: (*k.SSHPublicKey).Type(),
		Blob:   signatureBytes,
		Rest:   ssh.Marshal(additionalData),
	}

	return &sig, nil
}

func (k *Key) SetHWND(hwnd uintptr) {
	if k.Type == "NCRYPT" && k.handle != 0 {
		err := ncrypt.NCryptSetProperty(k.handle, ncrypt.NCRYPT_WINDOW_HANDLE_PROPERTY, hwnd, 0)
		if err != nil {
			log.Printf("Setting NCryptWindow handle failed: %v", err)
		}
	}
	k.hwnd = hwnd
}

func (k *Key) SetTimeout(timeout int) {
	if k.Type == "NCRYPT" && k.signer != nil {
		if ncryptSigner, ok := (*k.signer).(*Signer); ok {
			ncryptSigner.SetPINTimeout(timeout)
		}
	}
}

func (k *Key) SSHCertificateSerial() string {
	if k.SSHCertificate != nil {
		return strconv.FormatUint(k.SSHCertificate.Serial, 10)
	}
	return ""
}

type KeyManager struct {
	Keys            map[string]*Key
	providerHandles map[string]uintptr
	configPath      string
	publicKeysDir   string
	config          *KeyManagerConfig

	lwg    *sync.WaitGroup
	lctx   context.Context
	cancel context.CancelFunc
	hwnd   win.HWND

	namedPipeListener *listeners.NamedPipe
	cygwinListener    *listeners.Cygwin
	vSockListener     *listeners.VSock
	pageantListener   *listeners.Pageant
	sshAgent          KeyManagerAgent
	notifyChan        chan NotifyMsg
}

func NewKeyManager(configPath string) (*KeyManager, error) {
	var kmc KeyManagerConfig

	configDir := filepath.Dir(configPath)
	os.MkdirAll(configDir, os.ModePerm)

	publicKeysDir := filepath.Join(configDir, "PublicKeys")
	os.MkdirAll(publicKeysDir, os.ModePerm)

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		log.Printf("Using default config\n")
		// create a default config
		kmc = KeyManagerConfig{
			Keys:                 nil,
			PinTimeout:           5,
			CygwinEnabled:        true,
			PageantEnabled:       true,
			VSockEnabled:         true,
			NamedPipeEnabled:     true,
			DisableNotifications: true,
			USBEvents:            false,
		}
	} else {
		//log.Printf("Loading %s\n", configPath)
		content, err := ioutil.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read KeyManager config file at %s: %w", configPath, err)
		}

		err = json.Unmarshal(content, &kmc)
		if err != nil {
			return nil, fmt.Errorf("unable to parse KeyManager config file at %s: %w", configPath, err)
		}
	}

	km := KeyManager{
		Keys:              make(map[string]*Key),
		providerHandles:   make(map[string]uintptr),
		configPath:        configPath,
		config:            &kmc,
		hwnd:              0,
		publicKeysDir:     publicKeysDir,
		cygwinListener:    nil,
		pageantListener:   nil,
		namedPipeListener: nil,
		vSockListener:     nil,
	}
	km.providerHandles = make(map[string]uintptr)
	km.configPath = configPath

	km.sshAgent = KeyManagerAgent{
		km:     &km,
		locked: false,
		mu:     sync.Mutex{},
	}

	return &km, nil
}

func (km *KeyManager) Start() error {
	saveConfig := false
	km.lctx, km.cancel = context.WithCancel(context.Background())

	km.lwg = new(sync.WaitGroup)

	if km.config.CygwinEnabled {
		_, disableListenerInConfig, _ := km.StartListener(listeners.TYPE_CYGWIN)
		if disableListenerInConfig {
			km.config.CygwinEnabled = false
			saveConfig = true
		}
	}

	if km.config.VSockEnabled {
		_, disableListenerInConfig, _ := km.StartListener(listeners.TYPE_VSOCK)
		if disableListenerInConfig {
			km.config.VSockEnabled = false
			saveConfig = true
		}
	}

	if km.config.NamedPipeEnabled {
		_, disableListenerInConfig, _ := km.StartListener(listeners.TYPE_NAMED_PIPE)
		if disableListenerInConfig {
			km.config.NamedPipeEnabled = false
			saveConfig = true
		}
	}

	if km.config.PageantEnabled {
		_, disableListenerInConfig, _ := km.StartListener(listeners.TYPE_PAGEANT)
		if disableListenerInConfig {
			km.config.PageantEnabled = false
			saveConfig = true
		}
	}

	for _, k := range km.config.Keys {
		log.Printf("Loading key %s\n", k.Name)
		var err error

		if k.Type == "NCRYPT" {
			if k.ProviderName == "" {
				k.ProviderName = ncrypt.ProviderMSSC
			}

			_, err = km.getProviderHandle(k.ProviderName)
			if err != nil {
				return fmt.Errorf("unable to open provider %s for %s: %w", k.ProviderName, k.Name, err)
			}

			_, err = km.LoadNCryptKey(k)
		} else {
			_, err = km.LoadWebAuthNKey(k)
		}

		if err != nil {
			km.Keys[k.Name] = &Key{
				Name:                 k.Name,
				Type:                 k.Type,
				algorithm:            "unknown",
				length:               0,
				SSHPublicKey:         nil,
				SSHPublicKeyLocation: "",
				config:               k,
				handle:               0,
				LoadError:            err,
				Missing:              true,
			}

			if k.SSHPublicKey != "" {
				if sshPublicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(k.SSHPublicKey)); err == nil {
					km.Keys[k.Name].SSHPublicKey = &sshPublicKey
				} else {
					log.Printf("Unable to load stored public key: %v", err)
				}
			}
		}
	}

	if saveConfig {
		km.SaveConfig()
	}

	return nil
}

// StartListener attempts top start a listenerType, returning (success, disableListenerInConfig, error)
// If disableListenerInConfig is true, the caller should disable the listener in the config and save
func (km *KeyManager) StartListener(listenerType string) (bool, bool, error) {
	var err error
	var listener listeners.Listener

	switch listenerType {
	case listeners.TYPE_VSOCK:
		if km.vSockListener != nil {
			km.vSockListener.Stop()
		}

		km.vSockListener, err = listeners.NewVSockListener()

		if err != nil {
			var listenerErr *listeners.ListenerError
			if errors.As(err, &listenerErr) {
				switch listenerErr.Code() {
				case listeners.ERR_DISABLE:
					log.Printf("disabled vSock listener: %s", err)
					return false, true, nil
				case listeners.ERR_ABORTED:
					log.Printf("disabled vSock listener, but config wont be saved: %s", err)
					return false, false, nil
				}
			} else {
				log.Printf("could not create vsock listener: %s", err)
				return false, false, err
			}
		}

		if km.vSockListener == nil {
			return false, false, nil
		}

		listener = km.vSockListener
	case listeners.TYPE_CYGWIN:
		km.cygwinListener = new(listeners.Cygwin)
		km.cygwinListener.Sockfile = filepath.Join(filepath.Dir(km.configPath), "cygwin-agent.sock")
		listener = km.cygwinListener
	case listeners.TYPE_NAMED_PIPE:
		km.namedPipeListener = new(listeners.NamedPipe)
		listener = km.namedPipeListener
	case listeners.TYPE_PAGEANT:
		km.pageantListener = new(listeners.Pageant)
		listener = km.pageantListener
	default:
		return false, false, fmt.Errorf("invalid listener type %s", listenerType)
	}

	km.lwg.Add(1)
	go func(l listeners.Listener) {
		log.Printf("Starting listener %T\n", l)
		err := l.Run(km.lctx, &km.sshAgent)
		if err != nil {
			log.Printf("Error result from listener Run(): %s\n", err)
			return
		}
		km.lwg.Done()
	}(listener)

	return true, false, nil
}

func (km *KeyManager) RescanNCryptKeys() error {

	log.Printf("Rescanning for all NCrypt keys")
	for _, k := range km.KeysList() {
		if k.Type == "NCRYPT" {
			_, err := km.LoadNCryptKey(k.config)

			if err != nil {
				k.Missing = true
			}
		}
	}

	return nil
}

func (km *KeyManager) LoadNCryptKey(kc *KeyConfig) (*Key, error) {
	providerHandle, err := km.getProviderHandle(kc.ProviderName)
	if err != nil {
		return nil, err
	}

	// silently determine if the key is available
	keyHandle, err := ncrypt.NCryptOpenKey(providerHandle, kc.ContainerName, 0, ncrypt.NCRYPT_SILENT_FLAG)
	if err != nil {
		return nil, err
	}

	// close and reopen the handle allowing user interaction now
	ncrypt.NCryptFreeObject(keyHandle)
	keyHandle, err = ncrypt.NCryptOpenKey(providerHandle, kc.ContainerName, 0, 0)
	if err != nil {
		return nil, err
	}

	algorithmName, err := ncrypt.NCryptGetPropertyStr(keyHandle, ncrypt.NCRYPT_ALGORITHM_PROPERTY)
	if err != nil {
		ncrypt.NCryptFreeObject(keyHandle)
		return nil, err
	}

	var keyLength = 0
	if algorithmName == ncrypt.ALG_RSA {
		keyLength, err = ncrypt.NCryptGetPropertyInt(keyHandle, ncrypt.NCRYPT_LENGTH_PROPERTY)
		if err == nil {
			log.Printf("Got length %d\n", keyLength)
		} else {
			log.Printf("%v", err)
			keyLength = 0
		}
	}

	signer, err := newNCryptSigner(keyHandle, km.config.PinTimeout)
	if err != nil {
		ncrypt.NCryptFreeObject(keyHandle)
		return nil, err
	}

	sshPub, err := ssh.NewPublicKey(signer.Public())
	if err != nil {
		ncrypt.NCryptFreeObject(keyHandle)
		return nil, err
	}

	sshPub.Type()

	km.Keys[kc.Name] = &Key{
		Name:                 kc.Name,
		Type:                 "NCRYPT",
		algorithm:            algorithmName,
		length:               0,
		SSHPublicKey:         &sshPub,
		SSHPublicKeyLocation: "",
		config:               kc,
		handle:               keyHandle,
		LoadError:            nil,
		signer:               &signer,
		Missing:              false,
	}

	if km.hwnd != 0 {
		km.Keys[kc.Name].SetHWND(uintptr(km.hwnd))
	}

	km.Keys[kc.Name].SaveSSHPublicKey(km.publicKeysDir)
	km.Keys[kc.Name].LoadCertificate("")

	return km.Keys[kc.Name], nil
}

func (km *KeyManager) getProviderHandle(providerName string) (uintptr, error) {
	var pHandle uintptr
	var handleOpen bool
	var err error

	if pHandle, handleOpen = km.providerHandles[providerName]; !handleOpen {
		pHandle, err = ncrypt.NCryptOpenStorageProvider(providerName)
		if err != nil {
			return 0, fmt.Errorf("unable to open provider %s: %w", providerName, err)
		}

		km.providerHandles[providerName] = pHandle
	}

	return pHandle, nil
}

func (km *KeyManager) CreateNewNCryptKey(keyName string, containerName string, providerName string, algorithm string, bits int, password string) (*Key, error) {

	if _, keyNameExists := km.Keys[keyName]; keyNameExists {
		return nil, fmt.Errorf("key named %s already exists", keyName)
	}

	if containerName == "" {
		containerUUID, _ := uuid.NewRandom()
		containerName = containerUUID.String()
	}

	if providerName == ncrypt.ProviderMSSC {
		return nil, fmt.Errorf("creating keys on smartcards is not supported")
	}

	//TODO: investigate replacing this with NCryptIsAlgSupported()
	//https://learn.microsoft.com/en-us/windows/win32/api/ncrypt/nf-ncrypt-ncryptisalgsupported
	algorithmOK := false
	for _, i := range ncrypt.AVAILABLE_ALGORITHMS {
		if i == algorithm {
			algorithmOK = true
			break
		}
	}
	if !algorithmOK {
		return nil, fmt.Errorf("unsupported algorithm %v", algorithm)
	}

	providerHandle, err := km.getProviderHandle(providerName)
	if err != nil {
		return nil, err
	}

	kh, err := ncrypt.NCryptCreatePersistedKey(providerHandle, containerName, algorithm, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to create persisted key: %w", err)
	}

	if algorithm == ncrypt.ALG_RSA {
		err = ncrypt.NCryptSetProperty(kh, ncrypt.NCRYPT_LENGTH_PROPERTY, uint32(bits), 0)

		if err != nil {
			ncrypt.NCryptFreeObject(kh)
			return nil, fmt.Errorf("unable to set key NCRYPT_LENGTH_PROPERTY: %w", err)
		}
	}

	if password != "" {
		// If the provider is platform, set the property to a UI compatible one
		utf16Str, err := syscall.UTF16FromString(password)
		if err != nil {
			return nil, err
		}
		bytesStr := make([]byte, len(utf16Str)*2)
		for i, utf16 := range utf16Str {
			// LPCSTR (Windows' representation of utf16) is always little endian.
			binary.LittleEndian.PutUint16(bytesStr[i*2:i*2+2], utf16)
		}

		digest := sha1.Sum(bytesStr[:len(bytesStr)-2])

		err = ncrypt.NCryptSetProperty(kh, ncrypt.NCRYPT_PCP_USAGE_AUTH_PROPERTY, digest[:], 0)
		if err != nil {
			log.Printf("error setting password: %v\n", err)
		}
	}

	err = ncrypt.NCryptFinalizeKey(kh, 0)
	if err != nil {
		ncrypt.NCryptFreeObject(kh)
		return nil, fmt.Errorf("unable to finalize key: %w", err)
	}

	uc, err := ncrypt.NCryptGetPropertyStr(kh, ncrypt.NCRYPT_UNIQUE_NAME_PROPERTY)
	if err != nil {
		ncrypt.NCryptFreeObject(kh)
		return nil, fmt.Errorf("unable to retrieve NCRYPT_UNIQUE_NAME_PROPERTY: %w", err)
	}

	signer, err := newNCryptSigner(kh, km.config.PinTimeout)
	if err != nil {
		ncrypt.NCryptFreeObject(kh)
		return nil, err
	}

	sshPub, err := ssh.NewPublicKey(signer.Public())
	if err != nil {
		ncrypt.NCryptFreeObject(kh)
		return nil, err
	}

	kc := KeyConfig{
		Name:          keyName,
		Type:          "NCRYPT",
		ContainerName: uc,
		ProviderName:  providerName,
		Length:        bits,
		Algorithm:     algorithm,
		NoPin:         password == "",
	}

	k := Key{
		Name:                 keyName,
		Type:                 "NCRYPT",
		algorithm:            algorithm,
		length:               bits,
		SSHPublicKey:         &sshPub,
		SSHPublicKeyLocation: "",
		config:               &kc,
		handle:               kh,
		signer:               &signer,
		Missing:              false,
	}

	km.Keys[keyName] = &k

	k.SetHWND(uintptr(km.hwnd))
	k.SaveSSHPublicKey(km.publicKeysDir)
	k.LoadCertificate("")

	err = km.SaveConfig()

	// empty the password so we are prompted on first use
	if password != "" {
		ncrypt.NCryptSetProperty(kh, ncrypt.NCRYPT_PIN_PROPERTY, "", 0)
	}

	return &k, err
}

func (km *KeyManager) CreateNewWebAuthNKey(keyName string, application string, coseAlgorithm int64, coseHash string, resident bool, verifyRequired bool, hwnd uintptr) (*Key, error) {

	if _, keyNameExists := km.Keys[keyName]; keyNameExists {
		return nil, fmt.Errorf("key named %s already exists", keyName)
	}

	if application == "" {
		application = fmt.Sprintf("ssh:%s", keyName)
	} else {
		application = fmt.Sprintf("ssh:%s", application)
	}

	var userName string
	currentUser, err := user.Current()
	if err != nil {
		userName = ""
	} else {
		userName = currentUser.Name
	}

	userId := []byte("(null)")

	entityInfo := webauthn.RP_ENTITY_INFORMATION{
		Version: webauthn.RP_ENTITY_INFORMATION_CURRENT_VERSION,
		Id:      webauthn.LPCWSTR(application),
		Name:    webauthn.LPCWSTR("nCrypt Agent"),
		Icon:    nil,
	}

	userEntityInfo := webauthn.USER_ENTITY_INFORMATION{
		Version:     webauthn.USER_ENTITY_INFORMATION_CURRENT_VERSION,
		IdLen:       uint32(len(userId)),
		Id:          uintptr(unsafe.Pointer(&userId[0])),
		Name:        webauthn.LPCWSTR(userName),
		Icon:        nil,
		DisplayName: webauthn.LPCWSTR(userName),
	}

	coseParameter := []webauthn.COSE_CREDENTIAL_PARAMETER{
		{
			Version:        webauthn.COSE_CREDENTIAL_PARAMETER_CURRENT_VERSION,
			CredentialType: webauthn.LPCWSTR(webauthn.CREDENTIAL_TYPE_PUBLIC_KEY),
			Alg:            coseAlgorithm,
		},
	}

	coseParameters := webauthn.COSE_CREDENTIAL_PARAMETERS{
		Count:                uint32(len(coseParameter)),
		CredentialParameters: uintptr(unsafe.Pointer(&coseParameter[0])),
	}

	sshChallengeData := []byte("{}") // should we make a random data?

	clientData := webauthn.CLIENT_DATA{
		Version:              webauthn.CLIENT_DATA_CURRENT_VERSION,
		ClientDataJSONLength: uint32(len(sshChallengeData)),
		ClientDataJSON:       uintptr(unsafe.Pointer(&sshChallengeData[0])),
		HashAlgId:            webauthn.LPCWSTR(coseHash),
	}

	userVerificationRequirement := webauthn.USER_VERIFICATION_REQUIREMENT_DISCOURAGED

	if verifyRequired {
		userVerificationRequirement = webauthn.USER_VERIFICATION_REQUIREMENT_REQUIRED
	}

	credentialOptions := webauthn.AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS{
		Version:                     webauthn.AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS_CURRENT_VERSION,
		UserVerificationRequirement: uint32(userVerificationRequirement),
		RequireResidentKey:          resident,
	}

	var useWnd uintptr
	if hwnd != 0 {
		useWnd = hwnd
	} else if km.hwnd != 0 {
		useWnd = uintptr(km.hwnd)
	} else {
		useWnd = uintptr(win.GetForegroundWindow())
	}

	credentialAttestation, err := webauthn.AuthenticatorMakeCredential(useWnd, entityInfo, userEntityInfo, coseParameters, clientData, credentialOptions)

	if err != nil {
		return nil, fmt.Errorf("AuthenticatorMakeCredential failed: %w", err)
	}

	defer webauthn.FreeCredentialAttestation(credentialAttestation)

	attestationObjectBytes := webauthn.UintptrToBytes(credentialAttestation.AttestationObject, credentialAttestation.AttestationObjectLen)
	attestationObject := webauthn.AttestationObject{}
	err = cbor.Unmarshal(attestationObjectBytes, &attestationObject)

	if err != nil {
		return nil, fmt.Errorf("cbor.Unmarshal failed to parse attestationObject: %w", err)
	}

	reader := bytes.NewReader(attestationObject.AuthData)
	authData := webauthn.AuthenticatorData{}

	// Format of attestation object from https://www.w3.org/TR/webauthn/#attestation-object
	// Read Authenticator Data Header
	err = binary.Read(reader, binary.BigEndian, &authData.RPIDHash)
	err = binary.Read(reader, binary.BigEndian, &authData.Flags)
	err = binary.Read(reader, binary.BigEndian, &authData.Counter)

	//TODO: Look at authData.Flags to see if there is credential data or extensions

	// Read the attested credential data
	authData.AttestedCredentialData = &webauthn.AttestedCredentialData{}
	err = binary.Read(reader, binary.BigEndian, &authData.AttestedCredentialData.AAGUID)
	err = binary.Read(reader, binary.BigEndian, &authData.AttestedCredentialData.CredentialIDLen)
	authData.AttestedCredentialData.CredentialID = make([]byte, authData.AttestedCredentialData.CredentialIDLen)
	err = binary.Read(reader, binary.BigEndian, &authData.AttestedCredentialData.CredentialID)

	credentialPublicKey := make([]byte, reader.Len()) // Read the rest of the AttestedCredentialData in as the public key
	//TODO: check for CBOR extensions?!
	_, err = reader.Read(credentialPublicKey)

	coseKey := webauthn.COSEKey{}
	err = cbor.Unmarshal(credentialPublicKey, &coseKey)
	if err != nil {
		return nil, fmt.Errorf("cbor.Unmarshal failed to parse credentialPublicKey: %w", err)
	}

	var sshPrivBytes []byte
	var sshPubBytes []byte

	if coseKey.Kty == webauthn.COSE_KEY_TYPE_EC2 {
		if coseKey.Alg == webauthn.COSE_ALGORITHM_ECDSA_P256_WITH_SHA256 {
			x := new(big.Int)
			x.SetBytes(coseKey.X[:])
			y := new(big.Int)
			y.SetBytes(coseKey.Y[:])

			publicKeyBytes := elliptic.Marshal(elliptic.P256(), x, y)

			keyType := OPENSSH_SK_ECDSA
			curveName := "nistp256"

			sshPub := struct {
				Type        string
				ID          string
				Key         []byte
				Application string
			}{
				Type:        keyType,
				ID:          curveName,
				Key:         publicKeyBytes,
				Application: application,
			}

			sshPriv := sshPrivateKeySKECDSA{
				Type:        keyType,
				ID:          curveName,
				Key:         publicKeyBytes,
				Application: application,
				Flags:       authData.Flags,
				KeyHandle:   authData.AttestedCredentialData.CredentialID,
				Reserved:    "",
			}

			sshPubBytes = ssh.Marshal(&sshPub)
			sshPrivBytes = ssh.Marshal(&sshPriv)
		} else {
			return nil, fmt.Errorf("invalid algorithm cose identifier: %d", coseKey.Alg)
		}
	} else if coseKey.Kty == webauthn.COSE_KEY_TYPE_OKP {
		if coseKey.Alg == webauthn.COSE_ALGORITHM_EDDSA_ED25519 {
			keyType := OPENSSH_SK_ED25519

			sshPub := struct {
				Type        string
				Key         []byte
				Application string
			}{
				Type:        keyType,
				Key:         coseKey.X[:],
				Application: application,
			}

			sshPriv := struct {
				Type        string
				Key         []byte
				Application string
				Flags       byte
				KeyHandle   []byte
				Reserved    string
			}{
				Type:        keyType,
				Key:         coseKey.X[:],
				Application: application,
				Flags:       authData.Flags,
				KeyHandle:   authData.AttestedCredentialData.CredentialID,
				Reserved:    "",
			}

			sshPubBytes = ssh.Marshal(&sshPub)
			sshPrivBytes = ssh.Marshal(&sshPriv)

		} else {
			return nil, fmt.Errorf("invalid algorithm cose identifier: %d", coseKey.Alg)
		}
	} else {
		return nil, fmt.Errorf("openSSH SK keys only available for ECDSA or ED25519 key types (got %d)", coseKey.Kty)
	}

	sshPublicKeyObj, err := ssh.ParsePublicKey(sshPubBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse previously generated public key: %w", err)
	}

	k := Key{
		Name:           keyName,
		Type:           "WEBAUTHN",
		SSHPublicKey:   &sshPublicKeyObj,
		SSHCertificate: nil,
		Missing:        false,
		LoadError:      nil,
		algorithm:      "",
		config: &KeyConfig{
			Name:           keyName,
			Type:           "WEBAUTHN",
			ContainerName:  "",
			ProviderName:   "",
			SSHPublicKey:   "",
			SKPrivateHalf:  base64.StdEncoding.EncodeToString(sshPrivBytes),
			Algorithm:      "",
			Length:         0,
			VerifyRequired: verifyRequired,
			NoPin:          false,
		},
		handle: 0,
		signer: nil,
	}

	km.Keys[keyName] = &k

	k.SetHWND(uintptr(km.hwnd))
	k.SaveSSHPublicKey(km.publicKeysDir)
	k.LoadCertificate("")

	err = km.SaveConfig()

	return &k, err
}

func (km *KeyManager) KeysList() []*Key {
	if km.Keys == nil {
		return nil
	}

	keys := make([]*Key, 0, len(km.Keys))

	for _, k := range km.Keys {
		keys = append(keys, k)
	}

	return keys
}

func (km *KeyManager) Close() {
	for _, k := range km.Keys {
		k.Close()
	}

	for _, p := range km.providerHandles {
		if p != 0 {
			ncrypt.NCryptFreeObject(p)
		}
	}

	km.cancel()
}

func (km *KeyManager) SetHwnd(hwnd win.HWND) {
	km.hwnd = hwnd

	for _, k := range km.Keys {
		k.SetHWND(uintptr(hwnd))
	}
}

func (km *KeyManager) SaveConfig() error {
	var keyConfigs []*KeyConfig
	for _, k := range km.KeysList() {
		if k.config.SSHPublicKey == "" {
			k.config.SSHPublicKey = k.SSHPublicKeyString()
		}
		keyConfigs = append(keyConfigs, k.config)
	}

	km.config.Keys = keyConfigs
	jsonString, err := json.MarshalIndent(km.config, "", "    ")

	f, err := os.OpenFile(km.configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	_, err = f.Write(jsonString)
	f.Close()

	return err
}

func (km *KeyManager) DeleteKey(keyToDelete *Key, deleteFromKeystore bool) error {
	log.Printf("Deleting %s - from keystore %v\n", keyToDelete.Name, deleteFromKeystore)

	if deleteFromKeystore && keyToDelete.Type == "NCRYPT" {
		err := ncrypt.NCryptDeleteKey(keyToDelete.handle, 0)

		if err != nil {
			log.Printf("err: %s", err)
			return err
		}
	}

	delete(km.Keys, keyToDelete.Name)

	return km.SaveConfig()
}

func (km *KeyManager) SetPinTimeout(timeout int) {
	km.config.PinTimeout = timeout
	for _, k := range km.Keys {
		k.SetTimeout(timeout)
	}
}

func (km *KeyManager) GetPinTimeout() int {
	return km.config.PinTimeout
}
func (km *KeyManager) GetUSBEventsEnabled() bool {
	return km.config.USBEvents
}
func (km *KeyManager) GetNotificationsEnabled() bool {
	return !km.config.DisableNotifications
}

func (km *KeyManager) EnableListener(listenerType string, enabled bool) {

	running := false

	switch listenerType {
	case listeners.TYPE_PAGEANT:
		if km.pageantListener != nil {
			running = km.pageantListener.Running()
		}
	case listeners.TYPE_CYGWIN:
		if km.cygwinListener != nil {
			running = km.cygwinListener.Running()
		}
	case listeners.TYPE_VSOCK:
		if km.vSockListener != nil {
			running = km.vSockListener.Running()
		}
	case listeners.TYPE_NAMED_PIPE:
		if km.namedPipeListener != nil {
			running = km.namedPipeListener.Running()
		}
	default:
		return
	}

	if running == enabled {
		return
	}

	if running == true && enabled == false {
		switch listenerType {
		case listeners.TYPE_PAGEANT:
			if km.pageantListener != nil {
				km.pageantListener.Stop()
			}
		case listeners.TYPE_CYGWIN:
			if km.cygwinListener != nil {
				km.cygwinListener.Stop()
			}
		case listeners.TYPE_VSOCK:
			if km.vSockListener != nil {
				km.vSockListener.Stop()
			}
		case listeners.TYPE_NAMED_PIPE:
			if km.namedPipeListener != nil {
				km.namedPipeListener.Stop()
			}
		default:
			return
		}
	}

	if running == false && enabled == true {
		listenerStarted, disableListenerInConfig, _ := km.StartListener(listenerType)

		if disableListenerInConfig {
			enabled = false
		}

		enabled = listenerStarted
	}

	switch listenerType {
	case listeners.TYPE_PAGEANT:
		km.config.PageantEnabled = enabled
	case listeners.TYPE_CYGWIN:
		km.config.CygwinEnabled = enabled
	case listeners.TYPE_VSOCK:
		km.config.VSockEnabled = enabled
	case listeners.TYPE_NAMED_PIPE:
		km.config.NamedPipeEnabled = enabled
	}
}

func (km *KeyManager) GetListenerEnabled(listenerType string) bool {
	switch listenerType {
	case listeners.TYPE_PAGEANT:
		return km.config.PageantEnabled
	case listeners.TYPE_CYGWIN:
		return km.config.CygwinEnabled
	case listeners.TYPE_VSOCK:
		return km.config.VSockEnabled
	case listeners.TYPE_NAMED_PIPE:
		return km.config.NamedPipeEnabled
	}

	return false
}

func (km *KeyManager) SetNotifyChan(c chan NotifyMsg) {
	km.notifyChan = c
}

func (km *KeyManager) Notify(n NotifyMsg) {
	if km.GetNotificationsEnabled() {
		if km.notifyChan != nil {
			km.notifyChan <- n
		}
	}
}

func (km *KeyManager) CygwinSocketLocation() string {
	if km.cygwinListener != nil {
		return km.cygwinListener.Sockfile
	}

	return ""
}

func (km *KeyManager) LoadWebAuthNKey(kc *KeyConfig) (*Key, error) {
	out, _, _, _, err := ssh.ParseAuthorizedKey([]byte(kc.SSHPublicKey))

	if err != nil {
		log.Printf("Error parsing authorized: %w\n", err)
		return nil, err
	}

	k := Key{
		Name:           kc.Name,
		Type:           "WEBAUTHN",
		SSHPublicKey:   &out,
		SSHCertificate: nil,
		Missing:        false,
		LoadError:      nil,
		algorithm:      "",
		config:         kc,
		handle:         0,
		signer:         nil,
	}

	km.Keys[kc.Name] = &k

	k.SetHWND(uintptr(km.hwnd))
	k.SaveSSHPublicKey(km.publicKeysDir)
	k.LoadCertificate("")

	return &k, nil
}

func (km *KeyManager) SetNotificationsEnabled(enabled bool) {
	km.config.DisableNotifications = !enabled
}
