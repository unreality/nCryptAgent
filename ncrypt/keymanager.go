package ncrypt

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/lxn/win"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"ncryptagent/ncrypt/listeners"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

type KeyConfig struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	ContainerName string `json:"containerName"`
	ProviderName  string `json:"providerName,omitempty"`
	SSHPublicKey  string `json:"sshPublicKey,omitempty"`
	Algorithm     string `json:"algorithm,omitempty"`
	Length        int    `json:"length,omitempty"`
}

type KeyManagerConfig struct {
	Keys             []*KeyConfig `json:"keys,omitempty"`
	PinTimeout       int          `json:"pinTimeout,omitempty"`
	PageantEnabled   bool         `json:"pageant"`
	VSockEnabled     bool         `json:"vsock"`
	NamedPipeEnabled bool         `json:"namedpipe"`
	CygwinEnabled    bool         `json:"cygwin"`
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
}

func (k *Key) AlgorithmReadable() string {
	if k.algorithm == ALG_ECDSA_RSA {
		return fmt.Sprintf("%s-%d", k.algorithm, k.length)
	} else {
		return k.algorithm
	}
}

func (k *Key) SSHPublicKeyString() string {
	if k.SSHPublicKey != nil {
		pkBytes := ssh.MarshalAuthorizedKey(*k.SSHPublicKey)
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
		fmt.Println(k.SSHPublicKeyLocation)

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
		NCryptFreeObject(k.handle)
	}
	k.handle = 0
}

func (k *Key) SignSSH(b []byte) (*ssh.Signature, error) {
	if k.signer != nil {
		sshSigner, err := ssh.NewSignerFromSigner(*k.signer)

		if err != nil {
			return nil, err
		}

		signature, err := sshSigner.Sign(rand.Reader, b)
		if err == nil {
			k.Missing = false
		}

		return signature, err
	}

	return nil, fmt.Errorf("invalid signer")
}

func (k *Key) SignWithAlgorithmSSH(b []byte, algorithm string) (*ssh.Signature, error) {
	if k.signer != nil {
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
	}

	return nil, fmt.Errorf("invalid signer")
}

func (k *Key) SetHWND(hwnd uintptr) {
	if k.signer != nil {
		if ncryptSigner, ok := (*k.signer).(*Signer); ok {
			ncryptSigner.SetHwnd(hwnd)
		}
	}
}

func (k *Key) SetTimeout(timeout int) {
	if k.signer != nil {
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
		fmt.Printf("Using default config\n")
		// create a default config
		kmc = KeyManagerConfig{
			Keys:             nil,
			PinTimeout:       5,
			CygwinEnabled:    true,
			PageantEnabled:   true,
			VSockEnabled:     true,
			NamedPipeEnabled: true,
		}
	} else {
		fmt.Printf("Loading %s\n", configPath)
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
		Keys:            make(map[string]*Key),
		providerHandles: make(map[string]uintptr),
		configPath:      configPath,
		config:          &kmc,
		hwnd:            0,
		publicKeysDir:   publicKeysDir,
	}
	km.providerHandles = make(map[string]uintptr)
	km.configPath = configPath

	km.sshAgent = KeyManagerAgent{
		km:     &km,
		locked: false,
		mu:     sync.Mutex{},
	}

	km.namedPipeListener = new(listeners.NamedPipe)
	km.pageantListener = new(listeners.Pageant)
	km.cygwinListener = new(listeners.Cygwin)
	km.cygwinListener.Sockfile = filepath.Join(filepath.Dir(km.configPath), "cygwin-agent.sock")
	km.vSockListener = new(listeners.VSock)

	km.lctx, km.cancel = context.WithCancel(context.Background())

	km.lwg = new(sync.WaitGroup)

	if km.config.CygwinEnabled {
		km.StartListener(km.cygwinListener)
	}

	if km.config.VSockEnabled {
		km.StartListener(km.vSockListener)
	}

	if km.config.NamedPipeEnabled {
		km.StartListener(km.namedPipeListener)
	}

	if km.config.PageantEnabled {
		km.StartListener(km.pageantListener)
	}

	for _, k := range kmc.Keys {
		fmt.Printf("Loading key %s\n", k.Name)
		if k.ProviderName == "" {
			k.ProviderName = ProviderMSSC
		}

		_, err := km.getProviderHandle(k.ProviderName)
		if err != nil {
			return nil, fmt.Errorf("unable to open provider %s for %s: %w", k.ProviderName, k.Name, err)
		}

		_, err = km.LoadKey(k)
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
					fmt.Printf("Unable to load stored public key: %v", err)
				}
			}
		}
	}

	return &km, nil
}

func (km *KeyManager) StartListener(listener listeners.Listener) {
	km.lwg.Add(1)
	go func(l listeners.Listener) {
		fmt.Printf("Starting listener %T\n", l)
		err := l.Run(km.lctx, &km.sshAgent)
		if err != nil {
			fmt.Printf("Error result from listener Run(): %s\n", err)
			return
		}
		km.lwg.Done()
	}(listener)
}

func (km *KeyManager) EnsureListenerIs(listener listeners.Listener, enabled bool) {
	if listener.Running() == enabled {
		return
	}

	if listener.Running() == false && enabled == true {
		km.StartListener(listener)
	}

	if listener.Running() == true && enabled == false {
		listener.Stop()
	}
}

func (km *KeyManager) LoadKey(kc *KeyConfig) (*Key, error) {
	providerHandle, err := km.getProviderHandle(kc.ProviderName)
	if err != nil {
		return nil, err
	}

	// silently determine if the key is available
	keyHandle, err := NCryptOpenKey(providerHandle, kc.ContainerName, 0, NCRYPT_SILENT_FLAG)
	if err != nil {
		return nil, err
	}

	// close and reopen the handle allowing user interaction now
	NCryptFreeObject(keyHandle)
	keyHandle, err = NCryptOpenKey(providerHandle, kc.ContainerName, 0, 0)
	if err != nil {
		return nil, err
	}

	algorithmName, err := NCryptGetPropertyStr(keyHandle, NCRYPT_ALGORITHM_PROPERTY)
	if err != nil {
		NCryptFreeObject(keyHandle)
		return nil, err
	}

	//var keyLength = 0
	//if algorithmName == ALG_ECDSA_RSA {
	//    keyLength, err = NCryptGetPropertyInt(keyHandle, NCRYPT_LENGTH_PROPERTY)
	//    if err == nil {
	//        fmt.Printf("Got length %d\n", keyLength)
	//    } else {
	//        fmt.Printf("%v", err)
	//        keyLength = 0
	//    }
	//}

	signer, err := newNCryptSigner(keyHandle, km.config.PinTimeout)
	if err != nil {
		NCryptFreeObject(keyHandle)
		return nil, err
	}

	sshPub, err := ssh.NewPublicKey(signer.Public())
	if err != nil {
		NCryptFreeObject(keyHandle)
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
		pHandle, err = NCryptOpenStorageProvider(providerName)
		if err != nil {
			return 0, fmt.Errorf("unable to open provider %s: %w", providerName, err)
		}

		km.providerHandles[providerName] = pHandle
	}

	return pHandle, nil
}

func (km *KeyManager) CreateNewNCryptKey(keyName string, containerName string, providerName string, algorithm string, bits int) (*Key, error) {

	if _, keyNameExists := km.Keys[keyName]; keyNameExists {
		return nil, fmt.Errorf("key named %s already exists", keyName)
	}

	if containerName == "" {
		containerUUID, _ := uuid.NewRandom()
		containerName = containerUUID.String()
	}

	algorithmOK := false
	for _, i := range AVAILABLE_ALGORITHMS {
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

	kh, err := NCryptCreatePersistedKey(providerHandle, containerName, algorithm, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to create persisted key: %w", err)
	}

	if algorithm == ALG_ECDSA_RSA {
		err = NCryptSetProperty(kh, NCRYPT_LENGTH_PROPERTY, uint32(bits), 0)

		if err != nil {
			NCryptFreeObject(kh)
			return nil, fmt.Errorf("unable to set key NCRYPT_LENGTH_PROPERTY: %w", err)
		}
	}

	err = NCryptFinalizeKey(kh, 0)
	if err != nil {
		NCryptFreeObject(kh)
		return nil, fmt.Errorf("unable to finalize key: %w", err)
	}

	uc, err := NCryptGetPropertyStr(kh, NCRYPT_UNIQUE_NAME_PROPERTY)
	if err != nil {
		NCryptFreeObject(kh)
		return nil, fmt.Errorf("unable to retrieve NCRYPT_UNIQUE_NAME_PROPERTY: %w", err)
	}

	signer, err := newNCryptSigner(kh, km.config.PinTimeout)
	if err != nil {
		NCryptFreeObject(kh)
		return nil, err
	}

	sshPub, err := ssh.NewPublicKey(signer.Public())
	if err != nil {
		NCryptFreeObject(kh)
		return nil, err
	}

	kc := KeyConfig{
		Name:          keyName,
		Type:          "NCRYPT",
		ContainerName: uc,
		ProviderName:  providerName,
		Length:        bits,
		Algorithm:     algorithm,
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
			NCryptFreeObject(p)
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
	fmt.Printf("Deleting %s - fomrKeystore %v\n", keyToDelete.Name, deleteFromKeystore)

	if deleteFromKeystore {
		err := NCryptDeleteKey(keyToDelete.handle, 0)

		if err != nil {
			fmt.Printf("err: %s", err)
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

func (km *KeyManager) EnableListener(listenerType string, enabled bool) {
	switch listenerType {
	case listeners.TYPE_PAGEANT:
		km.EnsureListenerIs(km.pageantListener, enabled)
		km.config.PageantEnabled = enabled
	case listeners.TYPE_CYGWIN:
		km.EnsureListenerIs(km.cygwinListener, enabled)
		km.config.CygwinEnabled = enabled
	case listeners.TYPE_VSOCK:
		km.EnsureListenerIs(km.vSockListener, enabled)
		km.config.VSockEnabled = enabled
	case listeners.TYPE_NAMED_PIPE:
		km.EnsureListenerIs(km.namedPipeListener, enabled)
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
	if km.notifyChan != nil {
		km.notifyChan <- n
	}
}

func (km *KeyManager) CygwinSocketLocation() string {
	return km.cygwinListener.Sockfile
}
